package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ai-styler/internal/logging"
	"ai-styler/internal/monitoring"
	"ai-styler/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return uuid.New().String()[:32]
}

// RequestLoggerMiddleware provides request logging
type RequestLoggerMiddleware struct {
	logger *logging.StructuredLogger
}

// NewRequestLoggerMiddleware creates a new request logger middleware
func NewRequestLoggerMiddleware(logger *logging.StructuredLogger) *RequestLoggerMiddleware {
	return &RequestLoggerMiddleware{
		logger: logger,
	}
}

// RequestLogging returns a Gin middleware for request logging
func (m *RequestLoggerMiddleware) RequestLogging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Log structured request information
		m.logger.Info(context.Background(), "HTTP Request", map[string]interface{}{
			"method":     param.Method,
			"path":       param.Path,
			"status":     param.StatusCode,
			"latency":    param.Latency,
			"client_ip":  param.ClientIP,
			"user_agent": param.Request.UserAgent(),
			"timestamp":  param.TimeStamp,
		})
		return ""
	})
}

// MonitoringMiddleware provides comprehensive monitoring
type MonitoringMiddleware struct {
	monitor *monitoring.MonitoringService
	perfMon *monitoring.PerformanceMonitor
}

// NewMonitoringMiddleware creates a new monitoring middleware
func NewMonitoringMiddleware(monitor *monitoring.MonitoringService) *MonitoringMiddleware {
	return &MonitoringMiddleware{
		monitor: monitor,
		perfMon: monitoring.NewPerformanceMonitor("ai-styler"),
	}
}

// ErrorHandling returns a Gin middleware for error handling
func (m *MonitoringMiddleware) ErrorHandling() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			m.monitor.CaptureError(c.Request.Context(), err, map[string]interface{}{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
				"status": c.Writer.Status(),
			})
		}
	}
}

// PerformanceMonitoring returns a Gin middleware for performance monitoring
func (m *MonitoringMiddleware) PerformanceMonitoring() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Add request ID to context
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Monitor the request
		m.perfMon.MonitorRequest(c.Request.Context(), c.Request.Method, c.Request.URL.Path, func(ctx context.Context) error {
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return nil
		})

		// Record response time
		duration := time.Since(start)
		m.monitor.CapturePerformanceMetric(c.Request.Context(), "http_request_duration", duration.Seconds(), "seconds", map[string]string{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"status": fmt.Sprintf("%d", c.Writer.Status()),
		})
	}
}

// SecurityMonitoring returns a Gin middleware for security monitoring
func (m *MonitoringMiddleware) SecurityMonitoring() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Monitor for suspicious activity
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Check for suspicious patterns
		if m.isSuspiciousRequest(c.Request) {
			m.monitor.SendSecurityAlert(c.Request.Context(), "suspicious_request", fmt.Sprintf("Suspicious request from %s", clientIP), map[string]interface{}{
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
			})
		}

		c.Next()
	}
}

// isSuspiciousRequest checks if a request is suspicious
func (m *MonitoringMiddleware) isSuspiciousRequest(r *http.Request) bool {
	// Check for common attack patterns
	path := r.URL.Path
	userAgent := r.UserAgent()

	// SQL injection patterns
	sqlPatterns := []string{"'", "\"", "union", "select", "drop", "insert", "update", "delete"}
	for _, pattern := range sqlPatterns {
		if contains(path, pattern) || contains(userAgent, pattern) {
			return true
		}
	}

	// XSS patterns
	xssPatterns := []string{"<script", "javascript:", "onload=", "onerror="}
	for _, pattern := range xssPatterns {
		if contains(path, pattern) || contains(userAgent, pattern) {
			return true
		}
	}

	// Path traversal patterns
	if contains(path, "../") || contains(path, "..\\") {
		return true
	}

	return false
}

// ContextMiddleware provides context management
type ContextMiddleware struct{}

// NewContextMiddleware creates a new context middleware
func NewContextMiddleware() *ContextMiddleware {
	return &ContextMiddleware{}
}

// InjectContext returns a Gin middleware for context injection
func (m *ContextMiddleware) InjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add request ID to context
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Add trace ID to context
		traceID := uuid.New().String()
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)

		// Add start time to context
		c.Set("start_time", time.Now())

		// Add values to request context
		ctx := context.WithValue(c.Request.Context(), "trace_id", traceID)
		ctx = context.WithValue(ctx, "request_id", requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// UserContext returns a Gin middleware for user context
func (m *ContextMiddleware) UserContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user information from JWT token if present
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// This would be implemented with proper JWT validation
			// For now, we'll just set a placeholder
			c.Set("user_id", "placeholder")
		}

		c.Next()
	}
}

// VendorContext returns a Gin middleware for vendor context
func (m *ContextMiddleware) VendorContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract vendor information from JWT token if present
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// This would be implemented with proper JWT validation
			// For now, we'll just set a placeholder
			c.Set("vendor_id", "placeholder")
		}

		c.Next()
	}
}

// ConversionContext returns a Gin middleware for conversion context
func (m *ContextMiddleware) ConversionContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract conversion information from request if present
		conversionID := c.Param("conversion_id")
		if conversionID != "" {
			c.Set("conversion_id", conversionID)
		}

		c.Next()
	}
}

// RecoveryMiddleware provides panic recovery
type RecoveryMiddleware struct {
	monitor *monitoring.MonitoringService
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(monitor *monitoring.MonitoringService) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		monitor: monitor,
	}
}

// Recovery returns a Gin middleware for panic recovery
func (m *RecoveryMiddleware) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				m.monitor.CaptureError(c.Request.Context(), fmt.Errorf("panic: %v", err), map[string]interface{}{
					"method": c.Request.Method,
					"path":   c.Request.URL.Path,
					"panic":  err,
				})

				// Return error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "An unexpected error occurred",
				})
			}
		}()

		c.Next()
	}
}

// SecurityMiddleware provides security features
type SecurityMiddleware struct {
	config *security.SecurityConfig
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(config *security.SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		config: config,
	}
}

// CORSMiddleware returns a Gin middleware for CORS
func (m *SecurityMiddleware) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.config.CORSEnabled {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware returns a Gin middleware for security headers
func (m *SecurityMiddleware) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.config.SecurityHeadersEnabled {
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("X-Frame-Options", "DENY")
			c.Header("X-XSS-Protection", "1; mode=block")
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Header("Content-Security-Policy", "default-src 'self'")
		}

		c.Next()
	}
}

// JWTAuthMiddleware returns a Gin middleware for JWT authentication
func (m *SecurityMiddleware) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// This would be implemented with proper JWT validation
		// For now, we'll just check if the header exists
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminAuthMiddleware returns a Gin middleware for admin authentication
func (m *SecurityMiddleware) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This would check if the user has admin role
		// For now, we'll just pass through
		c.Next()
	}
}

// OptionalAuthMiddleware returns a Gin middleware for optional authentication
func (m *SecurityMiddleware) OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// This would validate the JWT token
			// For now, we'll just set a placeholder
			c.Set("authenticated", true)
		} else {
			c.Set("authenticated", false)
		}

		c.Next()
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
