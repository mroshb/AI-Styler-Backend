package route

import (
	"ai-styler/internal/config"
	"ai-styler/internal/crosscutting"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// NewWithCrossCutting creates a new router with comprehensive cross-cutting enhancements
func NewWithCrossCutting() *gin.Engine {
	r := gin.New()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Create cross-cutting configuration
	crossCuttingConfig := &crosscutting.CrossCuttingConfig{
		RateLimiting: &crosscutting.RateLimiterConfig{
			GlobalPerIP:     1000,
			GlobalPerUser:   5000,
			GlobalWindow:    cfg.RateLimit.Window,
			CleanupInterval: 5 * time.Minute,
			MaxEntries:      100000,
			EndpointLimits: map[string]crosscutting.EndpointLimit{
				"/api/auth/send-otp": {
					PerIP:   cfg.RateLimit.OTPPerIP,
					PerUser: cfg.RateLimit.OTPPerPhone,
					Window:  cfg.RateLimit.Window,
				},
				"/api/auth/verify-otp": {
					PerIP:   20,
					PerUser: 50,
					Window:  cfg.RateLimit.Window,
				},
				"/api/conversion/create": {
					PerIP:   10,
					PerUser: 100,
					Window:  cfg.RateLimit.Window,
				},
				"/api/image/upload": {
					PerIP:   20,
					PerUser: 200,
					Window:  cfg.RateLimit.Window,
				},
				"/api/payment/create": {
					PerIP:   5,
					PerUser: 20,
					Window:  cfg.RateLimit.Window,
				},
			},
			PlanLimits: map[string]crosscutting.PlanLimit{
				"free": {
					PerIP:   100,
					PerUser: 500,
					Window:  cfg.RateLimit.Window,
				},
				"premium": {
					PerIP:   500,
					PerUser: 2000,
					Window:  cfg.RateLimit.Window,
				},
				"enterprise": {
					PerIP:   2000,
					PerUser: 10000,
					Window:  cfg.RateLimit.Window,
				},
			},
		},
		RetryPolicies: &crosscutting.RetryConfig{
			MaxRetries:  3,
			BaseDelay:   time.Second,
			MaxDelay:    5 * time.Minute,
			Multiplier:  2.0,
			Jitter:      true,
			BackoffType: crosscutting.BackoffTypeExponential,
			ServiceConfigs: map[string]crosscutting.ServiceRetryConfig{
				"gemini_api": {
					MaxRetries:  5,
					BaseDelay:   2 * time.Second,
					MaxDelay:    30 * time.Second,
					Multiplier:  2.0,
					BackoffType: crosscutting.BackoffTypeExponential,
					RetryableErrors: []string{
						"rate limit",
						"quota exceeded",
						"service unavailable",
						"timeout",
						"internal server error",
					},
				},
				"worker": {
					MaxRetries:  3,
					BaseDelay:   time.Second,
					MaxDelay:    10 * time.Second,
					Multiplier:  1.5,
					BackoffType: crosscutting.BackoffTypeLinear,
					RetryableErrors: []string{
						"timeout",
						"service unavailable",
						"internal server error",
					},
				},
				"storage": {
					MaxRetries:  4,
					BaseDelay:   500 * time.Millisecond,
					MaxDelay:    15 * time.Second,
					Multiplier:  2.0,
					BackoffType: crosscutting.BackoffTypeExponential,
					RetryableErrors: []string{
						"timeout",
						"connection refused",
						"service unavailable",
						"internal server error",
					},
				},
			},
		},
		QuotaEnforcement: &crosscutting.QuotaConfig{
			EnforcementEnabled: true,
			GracePeriod:        5 * time.Minute,
			ResetSchedule:      "0 0 1 * *", // First day of every month
			ResetTimeZone:      "UTC",
			Plans: map[string]crosscutting.PlanQuota{
				"free": {
					PlanName: "free",
					MonthlyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 10,
						string(crosscutting.QuotaTypeImages):      50,
						string(crosscutting.QuotaTypeStorage):     100, // MB
						string(crosscutting.QuotaTypeBandwidth):   500, // MB
						string(crosscutting.QuotaTypeAPI):         1000,
					},
					DailyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 2,
						string(crosscutting.QuotaTypeImages):      10,
						string(crosscutting.QuotaTypeStorage):     10, // MB
						string(crosscutting.QuotaTypeBandwidth):   50, // MB
						string(crosscutting.QuotaTypeAPI):         100,
					},
					HourlyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 1,
						string(crosscutting.QuotaTypeImages):      5,
						string(crosscutting.QuotaTypeStorage):     5,  // MB
						string(crosscutting.QuotaTypeBandwidth):   25, // MB
						string(crosscutting.QuotaTypeAPI):         20,
					},
					ConcurrentLimit: 2,
					Features: map[string]bool{
						"high_resolution":  false,
						"batch_processing": false,
						"priority_support": false,
						"api_access":       true,
					},
				},
				"premium": {
					PlanName: "premium",
					MonthlyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 100,
						string(crosscutting.QuotaTypeImages):      500,
						string(crosscutting.QuotaTypeStorage):     1000, // MB
						string(crosscutting.QuotaTypeBandwidth):   5000, // MB
						string(crosscutting.QuotaTypeAPI):         10000,
					},
					DailyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 20,
						string(crosscutting.QuotaTypeImages):      100,
						string(crosscutting.QuotaTypeStorage):     100, // MB
						string(crosscutting.QuotaTypeBandwidth):   500, // MB
						string(crosscutting.QuotaTypeAPI):         1000,
					},
					HourlyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 5,
						string(crosscutting.QuotaTypeImages):      25,
						string(crosscutting.QuotaTypeStorage):     25,  // MB
						string(crosscutting.QuotaTypeBandwidth):   125, // MB
						string(crosscutting.QuotaTypeAPI):         100,
					},
					ConcurrentLimit: 5,
					Features: map[string]bool{
						"high_resolution":  true,
						"batch_processing": true,
						"priority_support": false,
						"api_access":       true,
					},
				},
				"enterprise": {
					PlanName: "enterprise",
					MonthlyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 1000,
						string(crosscutting.QuotaTypeImages):      5000,
						string(crosscutting.QuotaTypeStorage):     10000, // MB
						string(crosscutting.QuotaTypeBandwidth):   50000, // MB
						string(crosscutting.QuotaTypeAPI):         100000,
					},
					DailyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 200,
						string(crosscutting.QuotaTypeImages):      1000,
						string(crosscutting.QuotaTypeStorage):     1000, // MB
						string(crosscutting.QuotaTypeBandwidth):   5000, // MB
						string(crosscutting.QuotaTypeAPI):         10000,
					},
					HourlyLimits: map[string]int{
						string(crosscutting.QuotaTypeConversions): 50,
						string(crosscutting.QuotaTypeImages):      250,
						string(crosscutting.QuotaTypeStorage):     250,  // MB
						string(crosscutting.QuotaTypeBandwidth):   1250, // MB
						string(crosscutting.QuotaTypeAPI):         1000,
					},
					ConcurrentLimit: 20,
					Features: map[string]bool{
						"high_resolution":  true,
						"batch_processing": true,
						"priority_support": true,
						"api_access":       true,
					},
				},
			},
		},
		SecurityChecks: &crosscutting.SecurityConfig{
			MaxFileSize:              50 * 1024 * 1024, // 50MB
			AllowedTypes:             []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"},
			BlockedTypes:             []string{"application/x-executable", "application/x-msdownload", "application/x-javascript"},
			VirusScanEnabled:         true,
			PayloadInspectionEnabled: true,
			MaxPayloadSize:           10 * 1024 * 1024, // 10MB
			ImageValidationEnabled:   true,
			MaxImageWidth:            4096,
			MaxImageHeight:           4096,
			MinImageWidth:            100,
			MinImageHeight:           100,
		},
		SignedURLs: &crosscutting.SignedURLConfig{
			SigningKey:        cfg.JWT.Secret,
			DefaultExpiration: 24 * time.Hour,
			MaxExpiration:     7 * 24 * time.Hour, // 7 days
			RequireSigning: []string{
				"/api/storage/",
				"/api/images/",
				"/api/conversions/",
				"/api/results/",
			},
			AllowedDomains: []string{
				"localhost",
				"127.0.0.1",
				"your-domain.com",
			},
			ValidateIP:        true,
			ValidateReferer:   false,
			ValidateUserAgent: false,
		},
		Alerting: &crosscutting.AlertConfig{
			TelegramEnabled:   true,
			TelegramBotToken:  cfg.Monitoring.TelegramBotToken,
			TelegramChatID:    cfg.Monitoring.TelegramChatID,
			TelegramAPIURL:    "https://api.telegram.org/bot",
			EmailEnabled:      false,
			WebhookEnabled:    false,
			CriticalThreshold: 1 * time.Minute,
			HighThreshold:     5 * time.Minute,
			MediumThreshold:   15 * time.Minute,
			AlertRateLimit:    10,
			AlertRateWindow:   1 * time.Hour,
			MaxRetries:        3,
			RetryDelay:        5 * time.Second,
		},
		Logging: &crosscutting.LogConfig{
			OutputFormat:     "json",
			OutputStdout:     true,
			MinLevel:         crosscutting.LogLevelInfo,
			IncludeTimestamp: true,
			IncludeLevel:     true,
			IncludeCaller:    true,
			IncludeStack:     false,
			BufferSize:       1000,
			MaxSize:          100, // 100MB
			MaxAge:           30,  // 30 days
			MaxBackups:       5,
			DefaultFields: map[string]interface{}{
				"service": "ai-styler",
				"version": "1.0.0",
			},
		},
		ErrorHandling: &crosscutting.ErrorHandlerConfig{
			DefaultSeverity:    crosscutting.ErrorSeverityMedium,
			ShowDetailedErrors: false,
			IncludeStackTraces: false,
			DefaultRetryable:   false,
			DefaultRetryAfter:  5 * time.Second,
			LogErrors:          true,
			LogUserContext:     true,
			AlertOnErrors:      true,
			AlertThresholds: map[crosscutting.ErrorSeverity]bool{
				crosscutting.ErrorSeverityLow:      false,
				crosscutting.ErrorSeverityMedium:   false,
				crosscutting.ErrorSeverityHigh:     true,
				crosscutting.ErrorSeverityCritical: true,
			},
		},
		Extensibility: &crosscutting.ExtensibilityConfig{
			AutoDiscovery:     true,
			MaxPipelines:      100,
			MaxHooks:          1000,
			ParallelExecution: true,
			TimeoutSeconds:    30,
			ContinueOnError:   true,
			RetryOnError:      true,
			MaxRetries:        3,
			EnableMetrics:     true,
			EnableLogging:     true,
		},
		Enabled: true,
		Debug:   false,
	}

	// Create cross-cutting layer
	ccl := crosscutting.NewCrossCuttingLayer(crossCuttingConfig)
	defer ccl.Close()

	// Apply cross-cutting middleware
	r.Use(ccl.Middleware())
	r.Use(ccl.FileUploadMiddleware())
	r.Use(ccl.SignedURLMiddleware())

	// Health endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// Admin stats endpoint
	r.GET("/api/admin/crosscutting/stats", func(c *gin.Context) {
		stats := ccl.GetStats(c.Request.Context())
		c.JSON(200, stats)
	})

	// Auth routes (no auth required)
	mountAuth(r)

	// Protected routes
	protected := r.Group("/")
	protected.Use(crosscuttingMiddleware(ccl))
	{
		// User routes
		mountUser(protected)

		// Vendor routes
		mountVendor(protected)

		// Conversion routes
		mountConversion(protected)

		// Payment routes
		mountPayment(protected)

		// Image routes
		mountImage(protected)

		// Notification routes
		mountNotification(protected)

		// Worker routes
		mountWorker(protected)

		// Storage routes
		mountStorage(protected)

		// Share routes
		mountShare(protected)
	}

	// Admin routes (require admin auth)
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(crosscuttingMiddleware(ccl))
	{
		mountAdmin(adminGroup)
	}

	// Log initialization
	ccl.GetLogger().Info(context.Background(), "Router initialized with cross-cutting enhancements", map[string]interface{}{
		"rate_limiting":      true,
		"retry_policies":     true,
		"quota_enforcement":  true,
		"security_checks":    true,
		"signed_urls":        true,
		"alerting":           true,
		"structured_logging": true,
		"extensibility":      true,
		"error_handling":     true,
	})

	return r
}

// crosscuttingMiddleware provides middleware that integrates cross-cutting functionality
func crosscuttingMiddleware(ccl *crosscutting.CrossCuttingLayer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add cross-cutting layer to context for use in handlers
		c.Set("crosscutting", ccl)
		c.Next()
	}
}
