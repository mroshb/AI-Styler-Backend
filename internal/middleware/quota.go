package middleware

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// QuotaMiddleware enforces quota limits for users and vendors
type QuotaMiddleware struct {
	redisClient *redis.Client
}

// NewQuotaMiddleware creates a new quota middleware
func NewQuotaMiddleware(redisClient *redis.Client) *QuotaMiddleware {
	return &QuotaMiddleware{
		redisClient: redisClient,
	}
}

// EnforceConversionQuota checks if user has remaining conversions
func (q *QuotaMiddleware) EnforceConversionQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.Next()
			return
		}

		// Check monthly conversion quota
		monthlyKey := "quota:conversions:" + userID + ":" + time.Now().Format("2006-01")
		count, err := q.redisClient.Get(context.Background(), monthlyKey).Int()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
			c.Abort()
			return
		}

		// Get user's plan limit (mock for now)
		limit := 2 // Free plan limit
		if count >= limit {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Monthly conversion quota exceeded",
				"quota": map[string]interface{}{
					"used":      count,
					"limit":     limit,
					"remaining": 0,
				},
			})
			c.Abort()
			return
		}

		c.Set("quota_remaining", limit-count)
		c.Next()
	}
}

// EnforceImageQuota checks if vendor has remaining image upload quota
func (q *QuotaMiddleware) EnforceImageQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		vendorID := c.GetString("vendor_id")
		if vendorID == "" {
			c.Next()
			return
		}

		// Check monthly image upload quota
		monthlyKey := "quota:images:" + vendorID + ":" + time.Now().Format("2006-01")
		count, err := q.redisClient.Get(context.Background(), monthlyKey).Int()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
			c.Abort()
			return
		}

		// Get vendor's plan limit (mock for now)
		limit := 50 // Basic vendor plan limit
		if count >= limit {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Monthly image upload quota exceeded",
				"quota": map[string]interface{}{
					"used":      count,
					"limit":     limit,
					"remaining": 0,
				},
			})
			c.Abort()
			return
		}

		c.Set("quota_remaining", limit-count)
		c.Next()
	}
}

// IncrementConversionQuota increments conversion count after successful conversion
func (q *QuotaMiddleware) IncrementConversionQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only increment if request was successful
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			userID := c.GetString("user_id")
			if userID != "" {
				monthlyKey := "quota:conversions:" + userID + ":" + time.Now().Format("2006-01")
				q.redisClient.Incr(context.Background(), monthlyKey)
				q.redisClient.Expire(context.Background(), monthlyKey, 31*24*time.Hour)
			}
		}
	}
}

// IncrementImageQuota increments image upload count after successful upload
func (q *QuotaMiddleware) IncrementImageQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only increment if request was successful
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			vendorID := c.GetString("vendor_id")
			if vendorID != "" {
				monthlyKey := "quota:images:" + vendorID + ":" + time.Now().Format("2006-01")
				q.redisClient.Incr(context.Background(), monthlyKey)
				q.redisClient.Expire(context.Background(), monthlyKey, 31*24*time.Hour)
			}
		}
	}
}

// FileValidationMiddleware validates uploaded files
type FileValidationMiddleware struct {
	maxFileSize  int64
	allowedTypes []string
	allowedExts  []string
}

// NewFileValidationMiddleware creates a new file validation middleware
func NewFileValidationMiddleware(maxFileSize int64, allowedTypes, allowedExts []string) *FileValidationMiddleware {
	return &FileValidationMiddleware{
		maxFileSize:  maxFileSize,
		allowedTypes: allowedTypes,
		allowedExts:  allowedExts,
	}
}

// ValidateFileUpload validates file uploads
func (f *FileValidationMiddleware) ValidateFileUpload() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
			c.Abort()
			return
		}

		// Check file size
		if file.Size > f.maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "File too large",
				"max_size":  f.maxFileSize,
				"file_size": file.Size,
			})
			c.Abort()
			return
		}

		// Check file extension
		ext := filepath.Ext(file.Filename)
		allowed := false
		for _, allowedExt := range f.allowedExts {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "File type not allowed",
				"allowed_types": f.allowedExts,
				"file_type":     ext,
			})
			c.Abort()
			return
		}

		// Check MIME type
		fileHeader, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
			c.Abort()
			return
		}
		defer fileHeader.Close()

		buffer := make([]byte, 512)
		_, err = fileHeader.Read(buffer)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file header"})
			c.Abort()
			return
		}

		mimeType := http.DetectContentType(buffer)
		allowedMime := false
		for _, allowedType := range f.allowedTypes {
			if mimeType == allowedType {
				allowedMime = true
				break
			}
		}
		if !allowedMime {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "File MIME type not allowed",
				"allowed_types": f.allowedTypes,
				"file_type":     mimeType,
			})
			c.Abort()
			return
		}

		c.Set("validated_file", file)
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
type SecurityHeadersMiddleware struct{}

// NewSecurityHeadersMiddleware creates a new security headers middleware
func NewSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{}
}

// AddSecurityHeaders adds security headers to responses
func (s *SecurityHeadersMiddleware) AddSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HTTPS only)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// RateLimitMiddleware implements rate limiting
type RateLimitMiddleware struct {
	redisClient *redis.Client
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(redisClient *redis.Client) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redisClient: redisClient,
	}
}

// RateLimitByIP limits requests per IP address
func (r *RateLimitMiddleware) RateLimitByIP(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rate_limit:ip:" + ip

		count, err := r.redisClient.Get(context.Background(), key).Int()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
			c.Abort()
			return
		}

		if count >= limit {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":  "Rate limit exceeded",
				"limit":  limit,
				"window": window.String(),
			})
			c.Abort()
			return
		}

		r.redisClient.Incr(context.Background(), key)
		r.redisClient.Expire(context.Background(), key, window)

		c.Next()
	}
}

// RateLimitByUser limits requests per user
func (r *RateLimitMiddleware) RateLimitByUser(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.Next()
			return
		}

		key := "rate_limit:user:" + userID

		count, err := r.redisClient.Get(context.Background(), key).Int()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
			c.Abort()
			return
		}

		if count >= limit {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":  "Rate limit exceeded",
				"limit":  limit,
				"window": window.String(),
			})
			c.Abort()
			return
		}

		r.redisClient.Incr(context.Background(), key)
		r.redisClient.Expire(context.Background(), key, window)

		c.Next()
	}
}
