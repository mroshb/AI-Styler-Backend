package crosscutting

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// CrossCuttingConfig represents configuration for the cross-cutting layer
type CrossCuttingConfig struct {
	// Rate limiting
	RateLimiting *RateLimiterConfig `json:"rate_limiting"`

	// Retry policies
	RetryPolicies *RetryConfig `json:"retry_policies"`

	// Quota enforcement
	QuotaEnforcement *QuotaConfig `json:"quota_enforcement"`

	// Security checks
	SecurityChecks *SecurityConfig `json:"security_checks"`

	// Signed URLs
	SignedURLs *SignedURLConfig `json:"signed_urls"`

	// Alerting
	Alerting *AlertConfig `json:"alerting"`

	// Structured logging
	Logging *LogConfig `json:"logging"`

	// Error handling
	ErrorHandling *ErrorHandlerConfig `json:"error_handling"`

	// Extensibility
	Extensibility *ExtensibilityConfig `json:"extensibility"`

	// Global settings
	Enabled bool `json:"enabled"`
	Debug   bool `json:"debug"`
}

// DefaultCrossCuttingConfig returns default cross-cutting configuration
func DefaultCrossCuttingConfig() *CrossCuttingConfig {
	return &CrossCuttingConfig{
		RateLimiting:     DefaultRateLimiterConfig(),
		RetryPolicies:    DefaultRetryConfig(),
		QuotaEnforcement: DefaultQuotaConfig(),
		SecurityChecks:   DefaultSecurityConfig(),
		SignedURLs:       DefaultSignedURLConfig(),
		Alerting:         DefaultAlertConfig(),
		Logging:          DefaultLogConfig(),
		ErrorHandling:    DefaultErrorHandlerConfig(),
		Extensibility:    DefaultExtensibilityConfig(),
		Enabled:          true,
		Debug:            false,
	}
}

// CrossCuttingLayer provides comprehensive cross-cutting functionality
type CrossCuttingLayer struct {
	config          *CrossCuttingConfig
	rateLimiter     *RateLimiter
	retryService    *RetryService
	quotaEnforcer   *QuotaEnforcer
	securityChecker *SecurityChecker
	signedURLGen    *SignedURLGenerator
	alertingService *AlertingService
	logger          *StructuredLogger
	errorHandler    *ErrorHandler
	extensibility   *ExtensibilityFramework
}

// NewCrossCuttingLayer creates a new cross-cutting layer
func NewCrossCuttingLayer(config *CrossCuttingConfig) *CrossCuttingLayer {
	if config == nil {
		config = DefaultCrossCuttingConfig()
	}

	// Initialize components
	logger := NewStructuredLogger(config.Logging)
	alertingService := NewAlertingService(config.Alerting)
	errorHandler := NewErrorHandler(config.ErrorHandling, logger, alertingService)

	return &CrossCuttingLayer{
		config:          config,
		rateLimiter:     NewRateLimiter(config.RateLimiting),
		retryService:    NewRetryService(config.RetryPolicies),
		quotaEnforcer:   NewQuotaEnforcer(config.QuotaEnforcement, nil),           // Store would be injected
		securityChecker: NewSecurityChecker(config.SecurityChecks, nil, nil, nil), // Implementations would be injected
		signedURLGen:    NewSignedURLGenerator("https://api.example.com", config.SignedURLs),
		alertingService: alertingService,
		logger:          logger,
		errorHandler:    errorHandler,
		extensibility:   NewExtensibilityFramework(config.Extensibility, logger),
	}
}

// Middleware returns Gin middleware for the cross-cutting layer
func (ccl *CrossCuttingLayer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ccl.config.Enabled {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		start := time.Now()

		// Extract context information
		userID := c.GetString("user_id")
		ipAddress := c.ClientIP()
		endpoint := c.Request.URL.Path
		method := c.Request.Method

		// Create error context
		errorContext := &ErrorContext{
			UserID:    userID,
			RequestID: c.GetString("request_id"),
			TraceID:   c.GetString("trace_id"),
			IPAddress: ipAddress,
			UserAgent: c.Request.UserAgent(),
			Endpoint:  endpoint,
			Method:    method,
		}

		// Rate limiting
		if ccl.config.RateLimiting != nil {
			allowed, err := ccl.rateLimiter.Allow(ctx, ipAddress, userID, endpoint, "")
			if err != nil {
				ccl.logger.Error(ctx, "Rate limiting error", map[string]interface{}{
					"error": err.Error(),
					"ip":    ipAddress,
					"user":  userID,
				})
			}

			if !allowed {
				apiError := ccl.errorHandler.HandleRateLimitError(ctx, 100, time.Hour, errorContext)
				c.JSON(ccl.errorHandler.GetHTTPStatus(ErrorTypeRateLimit), apiError)
				c.Abort()
				return
			}
		}

		// Security checks
		if ccl.config.SecurityChecks != nil {
			securityResult, err := ccl.securityChecker.CheckRequest(ctx, c.Request)
			if err != nil {
				ccl.logger.Error(ctx, "Security check error", map[string]interface{}{
					"error": err.Error(),
				})
			}

			if !securityResult.Allowed {
				apiError := ccl.errorHandler.HandleSecurityError(ctx, securityResult.Reason, errorContext)
				c.JSON(ccl.errorHandler.GetHTTPStatus(ErrorTypeSecurity), apiError)
				c.Abort()
				return
			}
		}

		// Quota enforcement (for protected endpoints)
		if ccl.config.QuotaEnforcement != nil && userID != "" {
			// This would need to be implemented based on the specific endpoint
			// For now, we'll skip quota checks for non-conversion endpoints
			if ccl.isQuotaProtectedEndpoint(endpoint) {
				quotaResult, err := ccl.quotaEnforcer.CheckQuota(ctx, userID, QuotaTypeConversions, 1)
				if err != nil {
					ccl.logger.Error(ctx, "Quota check error", map[string]interface{}{
						"error": err.Error(),
						"user":  userID,
					})
				}

				if !quotaResult.Allowed {
					apiError := ccl.errorHandler.HandleQuotaExceededError(ctx, "conversions", 0, 0, errorContext)
					c.JSON(ccl.errorHandler.GetHTTPStatus(ErrorTypeQuotaExceeded), apiError)
					c.Abort()
					return
				}
			}
		}

		// Process request
		c.Next()

		// Log request completion
		duration := time.Since(start)
		ccl.logger.LogAPIRequest(ctx, method, endpoint, c.Writer.Status(), duration, map[string]interface{}{
			"user_id": userID,
			"ip":      ipAddress,
		})

		// Send alerts for errors
		if c.Writer.Status() >= 500 {
			ccl.alertingService.SendSystemAlert(ctx, "Server Error",
				fmt.Sprintf("Server error %d on %s %s", c.Writer.Status(), method, endpoint),
				"crosscutting_middleware", AlertSeverityHigh, map[string]interface{}{
					"status_code": c.Writer.Status(),
					"method":      method,
					"endpoint":    endpoint,
					"user_id":     userID,
					"ip":          ipAddress,
				})
		}
	}
}

// FileUploadMiddleware returns middleware for file upload security
func (ccl *CrossCuttingLayer) FileUploadMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ccl.config.Enabled || ccl.config.SecurityChecks == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// Check for file uploads
		if c.Request.Method == "POST" && c.Request.Header.Get("Content-Type") == "multipart/form-data" {
			form, err := c.MultipartForm()
			if err != nil {
				apiError := ccl.errorHandler.HandleValidationError(ctx, "form", "Invalid multipart form", &ErrorContext{
					Endpoint: c.Request.URL.Path,
					Method:   c.Request.Method,
				})
				c.JSON(ccl.errorHandler.GetHTTPStatus(ErrorTypeValidation), apiError)
				c.Abort()
				return
			}

			// Check each uploaded file
			for _, fileHeaders := range form.File {
				for _, fileHeader := range fileHeaders {
					securityResult, err := ccl.securityChecker.CheckFileUpload(ctx, fileHeader)
					if err != nil {
						ccl.logger.Error(ctx, "File security check error", map[string]interface{}{
							"error":    err.Error(),
							"filename": fileHeader.Filename,
						})
						continue
					}

					if !securityResult.Allowed {
						apiError := ccl.errorHandler.HandleSecurityError(ctx, securityResult.Reason, &ErrorContext{
							Endpoint: c.Request.URL.Path,
							Method:   c.Request.Method,
							Metadata: map[string]interface{}{
								"filename": fileHeader.Filename,
								"threats":  securityResult.Threats,
							},
						})
						c.JSON(ccl.errorHandler.GetHTTPStatus(ErrorTypeSecurity), apiError)
						c.Abort()
						return
					}
				}
			}
		}

		c.Next()
	}
}

// SignedURLMiddleware returns middleware for signed URL validation
func (ccl *CrossCuttingLayer) SignedURLMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ccl.config.Enabled || ccl.config.SignedURLs == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		endpoint := c.Request.URL.Path

		// Check if endpoint requires signed URLs
		if ccl.signedURLGen.RequiresSigning(endpoint) {
			validationResult, err := ccl.signedURLGen.ValidateSignedURL(ctx, c.Request.URL.String(),
				c.ClientIP(), c.Request.UserAgent(), c.Request.Referer())
			if err != nil {
				ccl.logger.Error(ctx, "Signed URL validation error", map[string]interface{}{
					"error":    err.Error(),
					"endpoint": endpoint,
				})
			}

			if !validationResult.Valid {
				apiError := ccl.errorHandler.HandleAuthenticationError(ctx, validationResult.Reason, &ErrorContext{
					Endpoint: endpoint,
					Method:   c.Request.Method,
				})
				c.JSON(ccl.errorHandler.GetHTTPStatus(ErrorTypeAuthentication), apiError)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RetryMiddleware returns middleware for retry functionality
func (ccl *CrossCuttingLayer) RetryMiddleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ccl.config.Enabled || ccl.config.RetryPolicies == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// Wrap the next handler with retry logic
		err := ccl.retryService.Retry(ctx, service, func(ctx context.Context) error {
			// Execute the next handler
			c.Next()

			// Check if there was an error
			if len(c.Errors) > 0 {
				return c.Errors.Last()
			}

			// Check status code
			if c.Writer.Status() >= 500 {
				return fmt.Errorf("server error: %d", c.Writer.Status())
			}

			return nil
		})

		if err != nil {
			ccl.logger.Error(ctx, "Retry middleware error", map[string]interface{}{
				"error":   err.Error(),
				"service": service,
			})
		}
	}
}

// isQuotaProtectedEndpoint checks if an endpoint requires quota protection
func (ccl *CrossCuttingLayer) isQuotaProtectedEndpoint(endpoint string) bool {
	protectedEndpoints := []string{
		"/api/conversion/create",
		"/api/conversion/process",
		"/api/image/upload",
		"/api/image/process",
	}

	for _, protected := range protectedEndpoints {
		if endpoint == protected {
			return true
		}
	}

	return false
}

// GetStats returns comprehensive statistics for the cross-cutting layer
func (ccl *CrossCuttingLayer) GetStats(ctx context.Context) map[string]interface{} {
	stats := map[string]interface{}{
		"config":  ccl.config,
		"enabled": ccl.config.Enabled,
		"debug":   ccl.config.Debug,
	}

	if ccl.rateLimiter != nil {
		stats["rate_limiter"] = ccl.rateLimiter.GetStats(ctx)
	}

	if ccl.retryService != nil {
		stats["retry_service"] = ccl.retryService.GetRetryStats(ctx)
	}

	if ccl.quotaEnforcer != nil {
		stats["quota_enforcer"] = ccl.quotaEnforcer.GetQuotaStats(ctx)
	}

	if ccl.securityChecker != nil {
		stats["security_checker"] = ccl.securityChecker.GetSecurityStats(ctx)
	}

	if ccl.signedURLGen != nil {
		stats["signed_url_generator"] = ccl.signedURLGen.GetSignedURLStats(ctx)
	}

	if ccl.alertingService != nil {
		stats["alerting_service"] = ccl.alertingService.GetAlertStats(ctx)
	}

	if ccl.logger != nil {
		stats["logger"] = ccl.logger.GetLogStats(ctx)
	}

	if ccl.errorHandler != nil {
		stats["error_handler"] = ccl.errorHandler.GetErrorStats(ctx)
	}

	if ccl.extensibility != nil {
		stats["extensibility"] = ccl.extensibility.GetStats(ctx)
	}

	return stats
}

// RegisterServiceHook registers a service hook for extensibility
func (ccl *CrossCuttingLayer) RegisterServiceHook(hook *ServiceHook) error {
	if ccl.extensibility == nil {
		return fmt.Errorf("extensibility framework not initialized")
	}

	return ccl.extensibility.RegisterHook(hook)
}

// ExecuteServicePipeline executes a service pipeline
func (ccl *CrossCuttingLayer) ExecuteServicePipeline(ctx context.Context, pipelineID string, event *ServiceEvent) error {
	if ccl.extensibility == nil {
		return fmt.Errorf("extensibility framework not initialized")
	}

	return ccl.extensibility.ExecutePipeline(ctx, pipelineID, event)
}

// GetLogger returns the structured logger instance
func (ccl *CrossCuttingLayer) GetLogger() *StructuredLogger {
	return ccl.logger
}

// Close closes the cross-cutting layer and cleans up resources
func (ccl *CrossCuttingLayer) Close() error {
	var errors []error

	if ccl.rateLimiter != nil {
		ccl.rateLimiter.Stop()
	}

	if ccl.logger != nil {
		if err := ccl.logger.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errors)
	}

	return nil
}
