package crosscutting

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ExampleServiceHook demonstrates how to create a custom service hook
type ExampleAnalyticsHook struct {
	name     string
	enabled  bool
	priority ServicePriority
}

func NewExampleAnalyticsHook() *ExampleAnalyticsHook {
	return &ExampleAnalyticsHook{
		name:     "analytics_hook",
		enabled:  true,
		priority: PriorityMedium,
	}
}

func (h *ExampleAnalyticsHook) Execute(ctx context.Context, event *ServiceEvent) error {
	// Example analytics processing
	fmt.Printf("Analytics hook processing event: %s\n", event.Type)

	// Simulate some processing
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (h *ExampleAnalyticsHook) GetName() string {
	return h.name
}

func (h *ExampleAnalyticsHook) GetType() ServiceType {
	return ServiceTypeAnalytics
}

func (h *ExampleAnalyticsHook) GetPriority() ServicePriority {
	return h.priority
}

func (h *ExampleAnalyticsHook) IsEnabled() bool {
	return h.enabled
}

// ExampleUsage demonstrates how to use the cross-cutting layer
func ExampleUsage() {
	// Create configuration
	config := DefaultCrossCuttingConfig()

	// Customize configuration
	config.RateLimiting.GlobalPerIP = 1000
	config.RateLimiting.GlobalPerUser = 5000
	config.SecurityChecks.MaxFileSize = 100 * 1024 * 1024 // 100MB
	config.Alerting.TelegramEnabled = true
	config.Alerting.TelegramBotToken = "your-bot-token"
	config.Alerting.TelegramChatID = "your-chat-id"

	// Create cross-cutting layer
	ccl := NewCrossCuttingLayer(config)
	defer ccl.Close()

	// Create Gin router
	r := gin.New()

	// Apply cross-cutting middleware
	r.Use(ccl.Middleware())
	r.Use(ccl.FileUploadMiddleware())
	r.Use(ccl.SignedURLMiddleware())

	// Register service hooks
	analyticsHook := &ServiceHook{
		ID:       "analytics_hook",
		Name:     "Analytics Hook",
		Type:     ServiceTypeAnalytics,
		Priority: PriorityMedium,
		Handler:  NewExampleAnalyticsHook(),
		Enabled:  true,
	}

	err := ccl.RegisterServiceHook(analyticsHook)
	if err != nil {
		fmt.Printf("Failed to register analytics hook: %v\n", err)
	}

	// Create a service pipeline
	pipeline := &ServicePipeline{
		ID:       "conversion_pipeline",
		Name:     "Conversion Processing Pipeline",
		Services: []*ServiceHook{analyticsHook},
		Config: map[string]interface{}{
			"timeout": 30,
			"retries": 3,
		},
		Enabled: true,
	}

	err = ccl.extensibility.CreatePipeline(pipeline)
	if err != nil {
		fmt.Printf("Failed to create pipeline: %v\n", err)
	}

	// Example API endpoints
	r.POST("/api/conversion/create", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Create service event
		event := ccl.extensibility.CreateEvent("conversion_created", "api", map[string]interface{}{
			"user_id":  c.GetString("user_id"),
			"endpoint": "/api/conversion/create",
		})

		// Execute pipeline
		err := ccl.ExecuteServicePipeline(ctx, "conversion_pipeline", event)
		if err != nil {
			ccl.logger.Error(ctx, "Pipeline execution failed", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(200, gin.H{"status": "success"})
	})

	r.GET("/api/storage/files/:id", func(c *gin.Context) {
		// This endpoint requires signed URLs
		fileID := c.Param("id")

		// Generate signed URL
		signedURLReq := &SignedURLRequest{
			Path:       fmt.Sprintf("/api/storage/files/%s", fileID),
			Method:     "GET",
			Expiration: 24 * time.Hour,
			IPAddress:  c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			UserID:     c.GetString("user_id"),
		}

		signedURL, err := ccl.signedURLGen.GenerateSignedURL(context.Background(), signedURLReq)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate signed URL"})
			return
		}

		c.JSON(200, gin.H{
			"signed_url": signedURL.URL,
			"expires_at": signedURL.ExpiresAt,
		})
	})

	r.GET("/api/admin/stats", func(c *gin.Context) {
		// Admin endpoint to view cross-cutting statistics
		stats := ccl.GetStats(c.Request.Context())
		c.JSON(200, stats)
	})

	// Start server
	r.Run(":8080")
}

// IntegrationExample shows how to integrate with existing services
func IntegrationExample() {
	// Create cross-cutting layer
	ccl := NewCrossCuttingLayer(nil)
	defer ccl.Close()

	ctx := context.Background()

	// Example: Rate limiting check
	allowed, err := ccl.rateLimiter.Allow(ctx, "192.168.1.1", "user123", "/api/conversion/create", "premium")
	if err != nil {
		fmt.Printf("Rate limiting error: %v\n", err)
	}
	if !allowed {
		fmt.Println("Rate limit exceeded")
	}

	// Example: Quota enforcement
	quotaResult, err := ccl.quotaEnforcer.CheckQuota(ctx, "user123", QuotaTypeConversions, 1)
	if err != nil {
		fmt.Printf("Quota check error: %v\n", err)
	}
	if !quotaResult.Allowed {
		fmt.Printf("Quota exceeded: %s\n", quotaResult.Reason)
	}

	// Example: Security check
	securityResult, err := ccl.securityChecker.CheckRequest(ctx, &http.Request{})
	if err != nil {
		fmt.Printf("Security check error: %v\n", err)
	}
	if !securityResult.Allowed {
		fmt.Printf("Security threat detected: %s\n", securityResult.Reason)
	}

	// Example: Retry service
	err = ccl.retryService.Retry(ctx, "gemini_api", func(ctx context.Context) error {
		// Simulate API call
		fmt.Println("Making API call...")
		return fmt.Errorf("temporary failure")
	})
	if err != nil {
		fmt.Printf("Retry failed: %v\n", err)
	}

	// Example: Alerting
	err = ccl.alertingService.SendSecurityAlert(ctx, "Suspicious Activity",
		"Multiple failed login attempts detected", "auth_service", "192.168.1.1",
		map[string]interface{}{
			"attempts":  5,
			"timeframe": "5 minutes",
		})
	if err != nil {
		fmt.Printf("Alert sending failed: %v\n", err)
	}

	// Example: Structured logging
	ccl.logger.Info(ctx, "Cross-cutting layer initialized", map[string]interface{}{
		"components": []string{"rate_limiter", "quota_enforcer", "security_checker", "retry_service", "alerting_service"},
	})

	// Example: Error handling
	apiError := ccl.errorHandler.HandleValidationError(ctx, "email", "Invalid email format", &ErrorContext{
		UserID:   "user123",
		Endpoint: "/api/user/update",
	})
	fmt.Printf("API Error: %+v\n", apiError)
}

// ProductionConfigurationExample shows production-ready configuration
func ProductionConfigurationExample() *CrossCuttingConfig {
	config := &CrossCuttingConfig{
		RateLimiting: &RateLimiterConfig{
			GlobalPerIP:     500,
			GlobalPerUser:   2000,
			GlobalWindow:    1 * time.Hour,
			CleanupInterval: 5 * time.Minute,
			MaxEntries:      100000,
			EndpointLimits: map[string]EndpointLimit{
				"/api/auth/send-otp": {
					PerIP:   5,
					PerUser: 10,
					Window:  time.Hour,
				},
				"/api/conversion/create": {
					PerIP:   10,
					PerUser: 50,
					Window:  time.Hour,
				},
			},
			PlanLimits: map[string]PlanLimit{
				"free": {
					PerIP:   100,
					PerUser: 500,
					Window:  time.Hour,
				},
				"premium": {
					PerIP:   500,
					PerUser: 2000,
					Window:  time.Hour,
				},
			},
		},
		SecurityChecks: &SecurityConfig{
			MaxFileSize:              100 * 1024 * 1024, // 100MB
			AllowedTypes:             []string{"image/jpeg", "image/png", "image/webp"},
			VirusScanEnabled:         true,
			PayloadInspectionEnabled: true,
			MaxPayloadSize:           10 * 1024 * 1024, // 10MB
			ImageValidationEnabled:   true,
			MaxImageWidth:            4096,
			MaxImageHeight:           4096,
		},
		Alerting: &AlertConfig{
			TelegramEnabled:  true,
			TelegramBotToken: "your-production-bot-token",
			TelegramChatID:   "your-production-chat-id",
			EmailEnabled:     true,
			SMTPHost:         "smtp.your-domain.com",
			SMTPPort:         587,
			EmailFrom:        "alerts@your-domain.com",
			EmailTo:          []string{"admin@your-domain.com", "security@your-domain.com"},
		},
		Logging: &LogConfig{
			OutputFormat:     "json",
			OutputFile:       "/var/log/ai-styler/app.log",
			MinLevel:         LogLevelInfo,
			IncludeTimestamp: true,
			IncludeLevel:     true,
			IncludeCaller:    true,
			MaxSize:          100, // 100MB
			MaxAge:           30,  // 30 days
			MaxBackups:       5,
		},
		ErrorHandling: &ErrorHandlerConfig{
			ShowDetailedErrors: false, // Hide detailed errors in production
			LogErrors:          true,
			AlertOnErrors:      true,
			AlertThresholds: map[ErrorSeverity]bool{
				ErrorSeverityHigh:     true,
				ErrorSeverityCritical: true,
			},
		},
		Enabled: true,
		Debug:   false,
	}

	return config
}
