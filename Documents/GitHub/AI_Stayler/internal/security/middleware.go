package security

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	// Rate limiting
	RateLimitEnabled bool
	RateLimitPerIP   int
	RateLimitPerUser int
	RateLimitWindow  time.Duration

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// CORS
	CORSEnabled    bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string

	// Security headers
	SecurityHeadersEnabled bool

	// Image scanning
	ImageScanEnabled bool

	// Signed URLs
	SignedURLEnabled    bool
	SignedURLExpiration time.Duration
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		RateLimitEnabled:       true,
		RateLimitPerIP:         100,
		RateLimitPerUser:       1000,
		RateLimitWindow:        time.Hour,
		JWTSecret:              "your-secret-key-change-in-production",
		JWTExpiration:          15 * time.Minute,
		CORSEnabled:            true,
		AllowedOrigins:         []string{"*"},
		AllowedMethods:         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:         []string{"*"},
		SecurityHeadersEnabled: true,
		ImageScanEnabled:       true,
		SignedURLEnabled:       true,
		SignedURLExpiration:    24 * time.Hour,
	}
}

// SecurityMiddleware provides comprehensive security middleware
type SecurityMiddleware struct {
	config       *SecurityConfig
	rateLimiter  RateLimiter
	jwtSigner    JWTSigner
	imageScanner ImageScanner
	urlGenerator SignedURLGenerator
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(config *SecurityConfig) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityMiddleware{
		config:       config,
		rateLimiter:  NewInMemoryRateLimiter(),
		jwtSigner:    NewSimpleJWTSigner(config.JWTSecret),
		imageScanner: NewMockImageScanner(),
		urlGenerator: NewMockSignedURLGenerator("https://storage.example.com", config.JWTSecret),
	}
}

// RateLimitMiddleware implements rate limiting per IP and user
func (sm *SecurityMiddleware) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.RateLimitEnabled {
			c.Next()
			return
		}

		// Get client IP
		clientIP := sm.getClientIP(c)

		// Rate limit by IP
		ipKey := fmt.Sprintf("ip:%s", clientIP)
		if !sm.rateLimiter.Allow(ipKey, sm.config.RateLimitPerIP, sm.config.RateLimitWindow) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests from this IP",
				"retry_after": sm.config.RateLimitWindow.Seconds(),
			})
			c.Abort()
			return
		}

		// Rate limit by user (if authenticated)
		if userID, exists := c.Get("user_id"); exists {
			userKey := fmt.Sprintf("user:%s", userID)
			if !sm.rateLimiter.Allow(userKey, sm.config.RateLimitPerUser, sm.config.RateLimitWindow) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":       "rate_limit_exceeded",
					"message":     "Too many requests from this user",
					"retry_after": sm.config.RateLimitWindow.Seconds(),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// JWTAuthMiddleware implements JWT authentication
func (sm *SecurityMiddleware) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Token required",
			})
			c.Abort()
			return
		}

		// Verify token
		claims, err := sm.jwtSigner.Verify(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid token",
			})
			c.Abort()
			return
		}

		// Set user context
		if userID, ok := claims["sub"].(string); ok {
			c.Set("user_id", userID)
		}
		if role, ok := claims["role"].(string); ok {
			c.Set("user_role", role)
		}

		c.Next()
	}
}

// CORSMiddleware implements CORS headers
func (sm *SecurityMiddleware) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.CORSEnabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin != "" {
			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range sm.config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(sm.config.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(sm.config.AllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func (sm *SecurityMiddleware) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.SecurityHeadersEnabled {
			c.Next()
			return
		}

		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}

// ImageScanMiddleware scans uploaded images for threats
func (sm *SecurityMiddleware) ImageScanMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.ImageScanEnabled {
			c.Next()
			return
		}

		// Check if this is a file upload request
		if c.Request.Method == "POST" && strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
			// Get uploaded file
			file, err := c.FormFile("file")
			if err != nil {
				c.Next()
				return
			}

			// Open file
			src, err := file.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_file",
					"message": "Could not open uploaded file",
				})
				c.Abort()
				return
			}
			defer src.Close()

			// Read file data
			fileData := make([]byte, file.Size)
			_, err = src.Read(fileData)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_file",
					"message": "Could not read uploaded file",
				})
				c.Abort()
				return
			}

			// Scan image
			result, err := sm.imageScanner.ScanImage(fileData, file.Filename)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "scan_failed",
					"message": "Could not scan uploaded file",
				})
				c.Abort()
				return
			}

			// Check if malicious
			if sm.imageScanner.IsMalicious(result) {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "malicious_file",
					"message": "Uploaded file contains malicious content",
					"threats": result.Threats,
				})
				c.Abort()
				return
			}

			// Store scan result in context for later use
			c.Set("scan_result", result)
		}

		c.Next()
	}
}

// AdminAuthMiddleware ensures only admin users can access admin routes
func (sm *SecurityMiddleware) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		// Check if user is admin
		userRole, exists := c.Get("user_role")
		if !exists || userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Admin access required",
			})
			c.Abort()
			return
		}

		// Set admin context
		c.Set("admin_user_id", userID)
		c.Next()
	}
}

// OptionalAuthMiddleware provides optional authentication
func (sm *SecurityMiddleware) OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Try to authenticate
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, err := sm.jwtSigner.Verify(token); err == nil {
				if userID, ok := claims["sub"].(string); ok {
					c.Set("user_id", userID)
				}
				if role, ok := claims["role"].(string); ok {
					c.Set("user_role", role)
				}
			}
		}

		c.Next()
	}
}

// getClientIP extracts the real client IP address
func (sm *SecurityMiddleware) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := c.ClientIP()
	if ip == "" {
		return "unknown"
	}

	return ip
}

// GenerateSignedURL generates a signed URL for accessing a resource
func (sm *SecurityMiddleware) GenerateSignedURL(bucket, key string) (string, error) {
	if !sm.config.SignedURLEnabled {
		return fmt.Sprintf("https://storage.example.com/%s/%s", bucket, key), nil
	}

	return sm.urlGenerator.GenerateSignedURL(bucket, key, sm.config.SignedURLExpiration)
}

// VerifySignedURL verifies a signed URL
func (sm *SecurityMiddleware) VerifySignedURL(url string) (bool, error) {
	if !sm.config.SignedURLEnabled {
		return true, nil
	}

	return sm.urlGenerator.VerifySignedURL(url)
}

// GetRateLimitInfo returns rate limit information for a key
func (sm *SecurityMiddleware) GetRateLimitInfo(key string) (remaining int, resetTime time.Time) {
	remaining = sm.rateLimiter.GetRemaining(key, sm.config.RateLimitPerIP, sm.config.RateLimitWindow)
	resetTime = time.Now().Add(sm.config.RateLimitWindow)
	return remaining, resetTime
}

// Context keys for storing security information
type contextKey string

const (
	UserIDKey     contextKey = "user_id"
	UserRoleKey   contextKey = "user_role"
	AdminUserKey  contextKey = "admin_user_id"
	ScanResultKey contextKey = "scan_result"
)

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID, true
	}
	return "", false
}

// GetUserRoleFromContext extracts user role from context
func GetUserRoleFromContext(ctx context.Context) (string, bool) {
	if userRole, ok := ctx.Value(UserRoleKey).(string); ok {
		return userRole, true
	}
	return "", false
}

// GetScanResultFromContext extracts scan result from context
func GetScanResultFromContext(ctx context.Context) (*ScanResult, bool) {
	if result, ok := ctx.Value(ScanResultKey).(*ScanResult); ok {
		return result, true
	}
	return nil, false
}
