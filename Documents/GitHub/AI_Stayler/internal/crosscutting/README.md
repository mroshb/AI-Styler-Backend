# 🔧 Cross-Cutting Enhancement Layer

A comprehensive cross-cutting enhancement layer for the Go backend of an AI clothing try-on platform, providing global rate limiting, retry policies, quota enforcement, security checks, signed URLs, alerting, structured logging, extensibility, and graceful error handling.

## 🚀 Features

### ✅ Implemented Features

- **Global Rate Limiting**: Per IP and per user with configurable limits and windows
- **Retry Policies**: Exponential backoff for all services (Worker, Gemini API, Storage)
- **Quota & Plan Enforcement**: Across all endpoints with monthly/daily/hourly limits
- **Security Checks**: File upload validation, virus scanning, payload inspection
- **Signed URLs**: Secure file access for user, vendor, and result images
- **Alerting System**: Telegram notifications for critical failures and security issues
- **Structured Logging**: Comprehensive logging for all API calls, conversions, payments, and storage operations
- **Extensibility**: Framework for new services (analytics, recommendations) to hook into existing pipelines
- **Edge Case Handling**: Graceful error handling with meaningful messages for frontend

## 📁 Package Structure

```
internal/crosscutting/
├── rate_limiter.go           # Global rate limiting per IP and user
├── retry_service.go          # Retry policies with exponential backoff
├── quota_enforcer.go         # Quota & plan enforcement
├── security_checker.go       # Security checks for file uploads
├── signed_url_generator.go    # Signed URL generation and validation
├── alerting_service.go       # Telegram alerting system
├── structured_logger.go      # Structured logging for all operations
├── extensibility_framework.go # Extensibility framework for new services
├── error_handler.go          # Graceful error handling
├── crosscutting_layer.go     # Main integration layer
└── examples.go              # Usage examples and integration guide
```

## 🔧 Quick Start

### Basic Usage

```go
package main

import (
    "AI_Styler/internal/crosscutting"
    "github.com/gin-gonic/gin"
)

func main() {
    // Create configuration
    config := crosscutting.DefaultCrossCuttingConfig()
    
    // Customize for your needs
    config.RateLimiting.GlobalPerIP = 1000
    config.SecurityChecks.MaxFileSize = 100 * 1024 * 1024 // 100MB
    config.Alerting.TelegramBotToken = "your-bot-token"
    config.Alerting.TelegramChatID = "your-chat-id"
    
    // Create cross-cutting layer
    ccl := crosscutting.NewCrossCuttingLayer(config)
    defer ccl.Close()
    
    // Create Gin router
    r := gin.New()
    
    // Apply middleware
    r.Use(ccl.Middleware())
    r.Use(ccl.FileUploadMiddleware())
    r.Use(ccl.SignedURLMiddleware())
    
    // Your routes here...
    r.POST("/api/conversion/create", handleConversion)
    r.GET("/api/storage/files/:id", handleFileAccess)
    
    r.Run(":8080")
}
```

### Production Configuration

```go
config := &crosscutting.CrossCuttingConfig{
    RateLimiting: &crosscutting.RateLimiterConfig{
        GlobalPerIP:    500,
        GlobalPerUser:  2000,
        GlobalWindow:   1 * time.Hour,
        EndpointLimits: map[string]crosscutting.EndpointLimit{
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
        PlanLimits: map[string]crosscutting.PlanLimit{
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
    SecurityChecks: &crosscutting.SecurityConfig{
        MaxFileSize:    100 * 1024 * 1024, // 100MB
        AllowedTypes:   []string{"image/jpeg", "image/png", "image/webp"},
        VirusScanEnabled: true,
        PayloadInspectionEnabled: true,
        ImageValidationEnabled: true,
    },
    Alerting: &crosscutting.AlertConfig{
        TelegramEnabled:  true,
        TelegramBotToken: "your-production-bot-token",
        TelegramChatID:   "your-production-chat-id",
        EmailEnabled:     true,
        SMTPHost:        "smtp.your-domain.com",
        EmailFrom:       "alerts@your-domain.com",
    },
    Logging: &crosscutting.LogConfig{
        OutputFormat: "json",
        OutputFile:   "/var/log/ai-styler/app.log",
        MinLevel:     crosscutting.LogLevelInfo,
        MaxSize:      100, // 100MB
        MaxAge:       30,  // 30 days
    },
    Enabled: true,
    Debug:   false,
}
```

## 🛡️ Security Features

### File Upload Security

```go
// Automatic security checks for file uploads
r.Use(ccl.FileUploadMiddleware())

// Manual security check
securityResult, err := ccl.securityChecker.CheckFileUpload(ctx, fileHeader)
if !securityResult.Allowed {
    // Handle security threat
    for _, threat := range securityResult.Threats {
        fmt.Printf("Threat: %s - %s\n", threat.Type, threat.Description)
    }
}
```

### Signed URLs

```go
// Generate signed URL
signedURLReq := &crosscutting.SignedURLRequest{
    Path:       "/api/storage/files/image123",
    Method:     "GET",
    Expiration: 24 * time.Hour,
    IPAddress:  clientIP,
    UserID:     userID,
}

signedURL, err := ccl.signedURLGen.GenerateSignedURL(ctx, signedURLReq)

// Validate signed URL
validationResult, err := ccl.signedURLGen.ValidateSignedURL(ctx, url, clientIP, userAgent, referer)
```

## 📊 Rate Limiting

### Global Rate Limiting

```go
// Check rate limits
allowed, err := ccl.rateLimiter.Allow(ctx, ipAddress, userID, endpoint, plan)
if !allowed {
    // Handle rate limit exceeded
}
```

### Endpoint-Specific Limits

```go
config.RateLimiting.EndpointLimits = map[string]crosscutting.EndpointLimit{
    "/api/auth/send-otp": {
        PerIP:   5,
        PerUser: 10,
        Window:  time.Hour,
    },
    "/api/conversion/create": {
        PerIP:   10,
        PerUser: 100,
        Window:  time.Hour,
    },
}
```

## 🔄 Retry Policies

### Service-Specific Retry Configuration

```go
config.RetryPolicies.ServiceConfigs = map[string]crosscutting.ServiceRetryConfig{
    "gemini_api": {
        MaxRetries:   5,
        BaseDelay:    2 * time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
        BackoffType:  crosscutting.BackoffTypeExponential,
    },
    "worker": {
        MaxRetries:   3,
        BaseDelay:    time.Second,
        MaxDelay:     10 * time.Second,
        Multiplier:   1.5,
        BackoffType:  crosscutting.BackoffTypeLinear,
    },
}
```

### Using Retry Service

```go
err := ccl.retryService.Retry(ctx, "gemini_api", func(ctx context.Context) error {
    // Your API call here
    return geminiAPI.Call(ctx, request)
})
```

## 📈 Quota Enforcement

### Plan-Based Quotas

```go
config.QuotaEnforcement.Plans = map[string]crosscutting.PlanQuota{
    "free": {
        MonthlyLimits: map[string]int{
            string(crosscutting.QuotaTypeConversions): 10,
            string(crosscutting.QuotaTypeImages):      50,
            string(crosscutting.QuotaTypeStorage):     100, // MB
        },
        DailyLimits: map[string]int{
            string(crosscutting.QuotaTypeConversions): 2,
            string(crosscutting.QuotaTypeImages):      10,
        },
        ConcurrentLimit: 2,
    },
    "premium": {
        MonthlyLimits: map[string]int{
            string(crosscutting.QuotaTypeConversions): 100,
            string(crosscutting.QuotaTypeImages):      500,
            string(crosscutting.QuotaTypeStorage):     1000, // MB
        },
        ConcurrentLimit: 5,
    },
}
```

### Quota Checking

```go
quotaResult, err := ccl.quotaEnforcer.CheckQuota(ctx, userID, crosscutting.QuotaTypeConversions, 1)
if !quotaResult.Allowed {
    fmt.Printf("Quota exceeded: %s\n", quotaResult.Reason)
    fmt.Printf("Remaining: %v\n", quotaResult.Remaining)
}
```

## 🚨 Alerting System

### Telegram Alerts

```go
// Send security alert
err := ccl.alertingService.SendSecurityAlert(ctx, "Suspicious Activity", 
    "Multiple failed login attempts detected", "auth_service", "192.168.1.1", 
    map[string]interface{}{
        "attempts": 5,
        "timeframe": "5 minutes",
    })

// Send system alert
err := ccl.alertingService.SendSystemAlert(ctx, "High CPU Usage", 
    "CPU usage is above 90%", "monitoring", crosscutting.AlertSeverityHigh, 
    map[string]interface{}{
        "cpu_usage": 95.5,
        "threshold": 90.0,
    })
```

### Quota Alerts

```go
err := ccl.alertingService.SendQuotaAlert(ctx, userID, "premium", "conversions", 95, 100)
```

## 📝 Structured Logging

### API Request Logging

```go
ccl.logger.LogAPIRequest(ctx, "POST", "/api/conversion/create", 200, duration, map[string]interface{}{
    "user_id": userID,
    "conversion_id": conversionID,
})
```

### Conversion Logging

```go
ccl.logger.LogConversion(ctx, conversionID, userID, "completed", map[string]interface{}{
    "processing_time": duration,
    "image_count": 2,
})
```

### Payment Logging

```go
ccl.logger.LogPayment(ctx, paymentID, userID, 29.99, "USD", "success", map[string]interface{}{
    "plan": "premium",
    "gateway": "stripe",
})
```

## 🔌 Extensibility Framework

### Creating Custom Service Hooks

```go
type AnalyticsHook struct {
    name     string
    enabled  bool
    priority crosscutting.ServicePriority
}

func (h *AnalyticsHook) Execute(ctx context.Context, event *crosscutting.ServiceEvent) error {
    // Your analytics processing here
    fmt.Printf("Processing event: %s\n", event.Type)
    return nil
}

func (h *AnalyticsHook) GetName() string { return h.name }
func (h *AnalyticsHook) GetType() crosscutting.ServiceType { return crosscutting.ServiceTypeAnalytics }
func (h *AnalyticsHook) GetPriority() crosscutting.ServicePriority { return h.priority }
func (h *AnalyticsHook) IsEnabled() bool { return h.enabled }
```

### Registering Hooks

```go
hook := &crosscutting.ServiceHook{
    ID:       "analytics_hook",
    Name:     "Analytics Hook",
    Type:     crosscutting.ServiceTypeAnalytics,
    Priority: crosscutting.PriorityMedium,
    Handler:  NewAnalyticsHook(),
    Enabled:  true,
}

err := ccl.RegisterServiceHook(hook)
```

### Creating Service Pipelines

```go
pipeline := &crosscutting.ServicePipeline{
    ID:   "conversion_pipeline",
    Name: "Conversion Processing Pipeline",
    Services: []*crosscutting.ServiceHook{analyticsHook, recommendationHook},
    Config: map[string]interface{}{
        "timeout": 30,
        "retries": 3,
    },
    Enabled: true,
}

err := ccl.extensibility.CreatePipeline(pipeline)
```

### Executing Pipelines

```go
event := ccl.extensibility.CreateEvent("conversion_created", "api", map[string]interface{}{
    "user_id": userID,
    "conversion_id": conversionID,
})

err := ccl.ExecuteServicePipeline(ctx, "conversion_pipeline", event)
```

## ⚠️ Error Handling

### Structured Error Responses

```go
// Validation error
apiError := ccl.errorHandler.HandleValidationError(ctx, "email", "Invalid email format", &crosscutting.ErrorContext{
    UserID: userID,
    Endpoint: "/api/user/update",
})

// Rate limit error
apiError := ccl.errorHandler.HandleRateLimitError(ctx, 100, time.Hour, errorContext)

// Quota exceeded error
apiError := ccl.errorHandler.HandleQuotaExceededError(ctx, "conversions", 10, 10, errorContext)

// Security error
apiError := ccl.errorHandler.HandleSecurityError(ctx, "Malicious file detected", errorContext)
```

### Error Response Format

```json
{
    "type": "validation",
    "severity": "low",
    "code": "INVALID_INPUT",
    "message": "The provided input is invalid",
    "details": "validation failed for field email: Invalid email format",
    "context": {
        "user_id": "user123",
        "endpoint": "/api/user/update",
        "method": "POST"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "retryable": false,
    "suggestions": [
        "Check your input format",
        "Ensure all required fields are provided",
        "Verify field values are within allowed ranges"
    ]
}
```

## 📊 Monitoring and Statistics

### Get Comprehensive Stats

```go
stats := ccl.GetStats(ctx)
fmt.Printf("Rate Limiter Stats: %+v\n", stats["rate_limiter"])
fmt.Printf("Quota Enforcer Stats: %+v\n", stats["quota_enforcer"])
fmt.Printf("Security Checker Stats: %+v\n", stats["security_checker"])
```

### Service Metrics

```go
metrics := ccl.extensibility.GetMetrics()
fmt.Printf("Hook Executions: %+v\n", metrics.HookExecutions)
fmt.Printf("Pipeline Executions: %+v\n", metrics.PipelineExecutions)
```

## 🔧 Configuration Reference

### Rate Limiting Configuration

```go
type RateLimiterConfig struct {
    GlobalPerIP    int                    // Global requests per IP per window
    GlobalPerUser  int                    // Global requests per user per window
    GlobalWindow   time.Duration          // Rate limit window
    EndpointLimits map[string]EndpointLimit // Endpoint-specific limits
    PlanLimits     map[string]PlanLimit   // Plan-based limits
    CleanupInterval time.Duration         // Cleanup interval for expired entries
    MaxEntries     int                    // Maximum number of entries to keep
}
```

### Security Configuration

```go
type SecurityConfig struct {
    MaxFileSize              int64         // Maximum file size in bytes
    AllowedTypes             []string      // Allowed MIME types
    BlockedTypes             []string      // Blocked MIME types
    VirusScanEnabled         bool          // Enable virus scanning
    PayloadInspectionEnabled bool          // Enable payload inspection
    MaxPayloadSize           int64         // Maximum payload size
    ImageValidationEnabled   bool          // Enable image validation
    MaxImageWidth            int           // Maximum image width
    MaxImageHeight           int           // Maximum image height
}
```

### Alerting Configuration

```go
type AlertConfig struct {
    TelegramEnabled    bool     // Enable Telegram alerts
    TelegramBotToken   string   // Telegram bot token
    TelegramChatID     string   // Telegram chat ID
    EmailEnabled       bool     // Enable email alerts
    SMTPHost          string   // SMTP host
    SMTPPort          int      // SMTP port
    EmailFrom         string   // From email address
    EmailTo           []string // To email addresses
    CriticalThreshold time.Duration // Critical alert threshold
    HighThreshold     time.Duration // High alert threshold
}
```

## 🚀 Deployment

### Docker Integration

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/internal/crosscutting ./internal/crosscutting
CMD ["./main"]
```

### Environment Variables

```bash
# Telegram Configuration
TELEGRAM_BOT_TOKEN=your-bot-token
TELEGRAM_CHAT_ID=your-chat-id

# Email Configuration
SMTP_HOST=smtp.your-domain.com
SMTP_PORT=587
SMTP_USERNAME=alerts@your-domain.com
SMTP_PASSWORD=your-password

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE=/var/log/ai-styler/app.log

# Security Configuration
MAX_FILE_SIZE=104857600  # 100MB
VIRUS_SCAN_ENABLED=true
PAYLOAD_INSPECTION_ENABLED=true
```

## 🧪 Testing

### Unit Tests

```go
func TestRateLimiter(t *testing.T) {
    config := crosscutting.DefaultRateLimiterConfig()
    rl := crosscutting.NewRateLimiter(config)
    
    ctx := context.Background()
    allowed, err := rl.Allow(ctx, "192.168.1.1", "user123", "/api/test", "free")
    assert.NoError(t, err)
    assert.True(t, allowed)
}
```

### Integration Tests

```go
func TestCrossCuttingIntegration(t *testing.T) {
    config := crosscutting.DefaultCrossCuttingConfig()
    ccl := crosscutting.NewCrossCuttingLayer(config)
    defer ccl.Close()
    
    // Test middleware integration
    r := gin.New()
    r.Use(ccl.Middleware())
    
    r.POST("/test", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "success"})
    })
    
    // Test request
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/test", nil)
    r.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}
```

## 📚 Best Practices

1. **Configuration Management**: Use environment-specific configurations
2. **Error Handling**: Always provide meaningful error messages to users
3. **Logging**: Log all critical operations with structured data
4. **Security**: Enable all security checks in production
5. **Monitoring**: Set up alerts for critical thresholds
6. **Performance**: Monitor rate limiting and quota usage
7. **Extensibility**: Use the framework for new service integrations

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the examples in `examples.go`

---

**Built with ❤️ for the AI Stayler platform**
