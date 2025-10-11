# Logging & Monitoring System

This document describes the comprehensive logging and monitoring system implemented for the AI Styler application.

## Overview

The monitoring system provides:
- **Centralized structured logging** with JSON format
- **Error tracking** with Sentry integration
- **Real-time alerts** via Telegram
- **Health monitoring** with comprehensive endpoints
- **Performance metrics** collection
- **Security monitoring** and alerting

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │───▶│  Monitoring     │───▶│   External      │
│                 │    │  Service        │    │   Services      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   Middleware    │
                    │   Layer         │
                    └─────────────────┘
```

## Components

### 1. Structured Logging (`internal/logging/`)

**Features:**
- JSON-formatted logs with consistent structure
- Context-aware logging with user/vendor/conversion IDs
- Multiple log levels (Debug, Info, Warn, Error, Fatal)
- Caller information and stack traces
- Configurable output destinations

**Usage:**
```go
// Initialize logger
logger := logging.NewStructuredLogger(logging.LoggerConfig{
    Level:       logging.LogLevelInfo,
    Format:      "json",
    Output:      "stdout",
    Service:     "ai-stayler",
    Version:     "1.0.0",
    Environment: "production",
})

// Log with context
logger.Info(ctx, "User action completed", map[string]interface{}{
    "user_id": "user-123",
    "action":  "conversion_started",
    "conversion_id": "conv-456",
})
```

**Log Structure:**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "service": "ai-stayler",
  "version": "1.0.0",
  "environment": "production",
  "message": "User action completed",
  "user_id": "user-123",
  "conversion_id": "conv-456",
  "caller": "service.go:123",
  "trace_id": "abc123def456"
}
```

### 2. Sentry Integration (`internal/monitoring/sentry.go`)

**Features:**
- Error tracking and performance monitoring
- Custom error classification
- User context and breadcrumbs
- Release tracking
- Custom tags and metadata

**Configuration:**
```go
sentryConfig := monitoring.SentryConfig{
    DSN:              "https://your-dsn@sentry.io/project",
    Environment:      "production",
    Release:          "1.0.0",
    Debug:            false,
    SampleRate:       1.0,
    TracesSampleRate: 0.1,
    AttachStacktrace: true,
    MaxBreadcrumbs:   50,
}
```

**Usage:**
```go
// Capture error with context
monitor.CaptureError(ctx, err, map[string]interface{}{
    "user_id": "user-123",
    "conversion_id": "conv-456",
    "operation": "image_processing",
})

// Capture business error
monitor.CaptureBusinessError(ctx, &common.BusinessError{
    Type:    common.ErrorTypeValidation,
    Code:    "INVALID_FORMAT",
    Message: "Unsupported image format",
    Context: map[string]interface{}{
        "file_type": "bmp",
        "user_id":   "user-123",
    },
})
```

### 3. Telegram Alerts (`internal/monitoring/telegram.go`)

**Features:**
- Real-time critical error alerts
- Quota and performance monitoring
- Security incident notifications
- System health alerts
- Customizable alert formatting

**Configuration:**
```go
telegramConfig := monitoring.TelegramConfig{
    BotToken: "your-bot-token",
    ChatID:   "your-chat-id",
    Enabled:  true,
    Timeout:  10 * time.Second,
}
```

**Alert Types:**
- **Critical Errors**: System failures, panics
- **Business Errors**: Validation failures, quota exceeded
- **Security Alerts**: Unauthorized access, suspicious activity
- **Performance Alerts**: High response times, resource usage
- **Health Alerts**: Service degradation, component failures

**Alert Format:**
```
🔥 *System Error: gemini_api*

📝 *Message:* API request failed with status 500

🏢 *Service:* ai-stayler
🌍 *Environment:* production
⏰ *Time:* 2024-01-15 10:30:00 UTC
⚠️ *Severity:* critical
🏷️ *Type:* gemini_api
👤 *User ID:* user-123
🔄 *Conversion ID:* conv-456
🔍 *Trace ID:* abc123def456

📋 *Context:*
• operation: image_generation
• retry_count: 3
• error_code: API_TIMEOUT
```

### 4. Health Monitoring (`internal/monitoring/health.go`)

**Endpoints:**
- `GET /api/health/` - Overall health status
- `GET /api/health/ready` - Readiness probe
- `GET /api/health/live` - Liveness probe
- `GET /api/health/system` - System information
- `GET /api/health/metrics` - Performance metrics

**Health Checks:**
- **Database**: Connection status, pool metrics
- **Redis**: Connection status, memory usage
- **System**: Memory usage, goroutine count, CPU

**Response Format:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "checks": [
    {
      "name": "database",
      "status": "healthy",
      "message": "Database connection healthy",
      "duration": "2ms",
      "last_checked": "2024-01-15T10:30:00Z",
      "details": {
        "open_connections": 5,
        "in_use": 2,
        "idle": 3
      }
    }
  ],
  "summary": {
    "total": 3,
    "healthy": 3,
    "degraded": 0,
    "unhealthy": 0
  }
}
```

### 5. Middleware Integration (`internal/middleware/monitoring.go`)

**Middleware Stack:**
1. **Recovery**: Panic recovery with monitoring
2. **Context Injection**: Request ID, user context
3. **Request Logging**: Structured request/response logging
4. **Error Handling**: Automatic error capture
5. **Performance Monitoring**: Response time tracking
6. **Security Monitoring**: Suspicious activity detection

**Request Context:**
```go
// Automatically injected context fields
ctx.Value("request_id")    // Unique request identifier
ctx.Value("trace_id")      // Distributed tracing ID
ctx.Value("user_id")       // Authenticated user ID
ctx.Value("vendor_id")     // Vendor ID (if applicable)
ctx.Value("conversion_id") // Conversion ID (if applicable)
```

## Configuration

### Environment Variables

```bash
# Logging
LOG_LEVEL=info                    # debug, info, warn, error, fatal
ENVIRONMENT=production           # development, staging, production
VERSION=1.0.0                   # Application version

# Sentry
SENTRY_DSN=https://your-dsn@sentry.io/project

# Telegram
TELEGRAM_BOT_TOKEN=your-bot-token
TELEGRAM_CHAT_ID=your-chat-id

# Health Monitoring
HEALTH_ENABLED=true
```

### Configuration Structure

```go
type MonitoringConfig struct {
    Sentry: SentryConfig{
        DSN:              "https://your-dsn@sentry.io/project",
        Environment:      "production",
        Release:          "1.0.0",
        Debug:            false,
        SampleRate:       1.0,
        TracesSampleRate: 0.1,
        AttachStacktrace: true,
        MaxBreadcrumbs:   50,
    },
    Telegram: TelegramConfig{
        BotToken: "your-bot-token",
        ChatID:   "your-chat-id",
        Enabled:  true,
        Timeout:  10 * time.Second,
    },
    Logging: LoggerConfig{
        Level:       LogLevelInfo,
        Format:      "json",
        Output:      "stdout",
        Service:     "ai-styler",
        Version:     "1.0.0",
        Environment: "production",
    },
    Health: HealthConfig{
        Enabled:       true,
        CheckInterval: 30 * time.Second,
        Timeout:       10 * time.Second,
    },
}
```

## Usage Examples

### 1. Service Integration

```go
// Initialize monitoring service
monitor, err := monitoring.NewMonitoringService(config, db, redisClient)
if err != nil {
    log.Fatal("Failed to initialize monitoring:", err)
}
defer monitor.Close()

// Use in service
func (s *ConversionService) ProcessImage(ctx context.Context, req ProcessRequest) error {
    // Log operation start
    s.monitor.LogInfo(ctx, "Image processing started", map[string]interface{}{
        "user_id":       req.UserID,
        "conversion_id": req.ConversionID,
        "file_size":     len(req.ImageData),
    })

    // Process image
    result, err := s.processor.Process(req.ImageData)
    if err != nil {
        // Capture error with context
        s.monitor.CaptureError(ctx, err, map[string]interface{}{
            "user_id":       req.UserID,
            "conversion_id": req.ConversionID,
            "operation":     "image_processing",
        })
        return err
    }

    // Log success
    s.monitor.LogInfo(ctx, "Image processing completed", map[string]interface{}{
        "user_id":       req.UserID,
        "conversion_id": req.ConversionID,
        "processing_time": time.Since(start).Milliseconds(),
    })

    return nil
}
```

### 2. Error Handling

```go
// Business error handling
func (s *UserService) UpdateProfile(ctx context.Context, userID string, req UpdateRequest) error {
    if err := s.validateProfile(req); err != nil {
        businessErr := &common.BusinessError{
            Type:    common.ErrorTypeValidation,
            Code:    "INVALID_PROFILE",
            Message: err.Error(),
            Context: map[string]interface{}{
                "user_id": userID,
                "fields":  req.Fields,
            },
        }
        
        s.monitor.CaptureBusinessError(ctx, businessErr)
        return businessErr
    }
    
    // Continue with update...
}
```

### 3. Performance Monitoring

```go
// Performance metric capture
func (s *ImageService) GenerateThumbnail(ctx context.Context, imageID string) error {
    start := time.Now()
    
    // Generate thumbnail
    err := s.processor.GenerateThumbnail(imageID)
    
    // Capture performance metric
    s.monitor.CapturePerformanceMetric(ctx, "thumbnail_generation", 
        float64(time.Since(start).Milliseconds()), "ms", 
        map[string]string{
            "image_id": imageID,
            "success":  strconv.FormatBool(err == nil),
        })
    
    return err
}
```

### 4. Quota Monitoring

```go
// Quota alert
func (s *QuotaService) CheckQuota(ctx context.Context, userID string) error {
    usage, limit, err := s.getUserQuota(ctx, userID)
    if err != nil {
        return err
    }
    
    if usage >= limit {
        s.monitor.SendQuotaAlert(ctx, "monthly_conversions", usage, limit, userID)
        return errors.New("quota exceeded")
    }
    
    return nil
}
```

## Monitoring Dashboard

### Health Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/api/health/` | Overall health | Complete health status |
| `/api/health/ready` | Readiness probe | `{"status": "ready"}` |
| `/api/health/live` | Liveness probe | `{"status": "alive"}` |
| `/api/health/system` | System info | System metrics |
| `/api/health/metrics` | Performance | All metrics |

### Log Aggregation

**Recommended Stack:**
- **ELK Stack**: Elasticsearch, Logstash, Kibana
- **Fluentd**: Log collection and forwarding
- **Grafana**: Metrics visualization
- **Prometheus**: Metrics collection

**Log Shipping:**
```bash
# Using Fluentd
<source>
  @type tail
  path /var/log/ai-styler/*.log
  pos_file /var/log/fluentd/ai-styler.log.pos
  tag ai-styler
  format json
</source>

<match ai-styler>
  @type elasticsearch
  host elasticsearch.example.com
  port 9200
  index_name ai-styler-logs
</match>
```

## Best Practices

### 1. Logging Guidelines

- **Use structured logging** with consistent field names
- **Include context** (user_id, conversion_id, trace_id)
- **Log at appropriate levels** (Info for business events, Error for failures)
- **Avoid logging sensitive data** (passwords, tokens, personal data)
- **Use meaningful messages** that describe what happened

### 2. Error Handling

- **Capture errors immediately** when they occur
- **Include relevant context** in error reports
- **Use appropriate error types** (System, Business, Retryable)
- **Set correct severity levels** for alerting

### 3. Performance Monitoring

- **Monitor key metrics** (response time, throughput, error rate)
- **Set up alerts** for performance degradation
- **Track business metrics** (conversions, user activity)
- **Monitor resource usage** (CPU, memory, database connections)

### 4. Security Monitoring

- **Monitor authentication failures**
- **Track suspicious activity** (high error rates, unusual patterns)
- **Alert on security incidents** immediately
- **Log security events** with appropriate detail

## Troubleshooting

### Common Issues

1. **Sentry not receiving events**
   - Check DSN configuration
   - Verify network connectivity
   - Check Sentry project settings

2. **Telegram alerts not working**
   - Verify bot token and chat ID
   - Check bot permissions
   - Test with `/api/health/` endpoint

3. **Health checks failing**
   - Check database/Redis connectivity
   - Verify service dependencies
   - Check resource limits

4. **High log volume**
   - Adjust log levels
   - Implement log sampling
   - Use log rotation

### Debug Mode

Enable debug logging for troubleshooting:
```bash
LOG_LEVEL=debug
SENTRY_DEBUG=true
```

This will provide detailed information about monitoring system operations.

## Future Enhancements

1. **Distributed Tracing**: OpenTelemetry integration
2. **Custom Metrics**: Business-specific metrics
3. **Alert Rules**: Configurable alert thresholds
4. **Dashboard**: Real-time monitoring dashboard
5. **Log Analytics**: Advanced log analysis and insights
6. **A/B Testing**: Feature flag monitoring
7. **Cost Monitoring**: Resource cost tracking
8. **Compliance**: Audit logging for compliance requirements
