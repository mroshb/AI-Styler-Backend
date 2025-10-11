package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ai-styler/internal/common"
	"ai-styler/internal/logging"

	"github.com/getsentry/sentry-go"
	"github.com/go-redis/redis/v8"
)

// MonitoringConfig represents the monitoring configuration
type MonitoringConfig struct {
	Sentry   SentryConfig
	Telegram TelegramConfig
	Logging  logging.LoggerConfig
	Health   HealthConfig
}

// HealthConfig represents health monitoring configuration
type HealthConfig struct {
	Enabled       bool
	CheckInterval time.Duration
	Timeout       time.Duration
}

// MonitoringService provides comprehensive monitoring capabilities
type MonitoringService struct {
	logger       *logging.StructuredLogger
	sentry       *SentryMonitor
	telegram     *TelegramMonitor
	health       *HealthMonitor
	config       MonitoringConfig
	errorHandler *common.ErrorHandler
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(config MonitoringConfig, db *sql.DB, redisClient *redis.Client) (*MonitoringService, error) {
	// Initialize logger
	logger := logging.NewStructuredLogger(config.Logging)

	// Initialize Sentry
	sentry, err := NewSentryMonitor(config.Sentry)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize Sentry", map[string]interface{}{
			"error": err.Error(),
		})
		// Continue without Sentry if initialization fails
	}

	// Initialize Telegram
	telegram := NewTelegramMonitor(config.Telegram)

	// Initialize health monitor
	health := NewHealthMonitor(config.Logging.Version, config.Logging.Environment)

	// Add health checkers
	if db != nil {
		health.AddChecker("database", &DatabaseHealthChecker{db: db})
	}
	if redisClient != nil {
		health.AddChecker("redis", &RedisHealthChecker{client: redisClient})
	}
	health.AddChecker("system", &SystemHealthChecker{})

	// Create error handler
	errorHandler := common.NewErrorHandler(telegram, logger, common.ErrorConfig{
		MaxRetries:     3,
		BaseRetryDelay: time.Second,
		MaxRetryDelay:  5 * time.Minute,
		AlertThresholds: map[common.SeverityLevel]bool{
			common.SeverityHigh:     true,
			common.SeverityCritical: true,
		},
	})

	service := &MonitoringService{
		logger:       logger,
		sentry:       sentry,
		telegram:     telegram,
		health:       health,
		config:       config,
		errorHandler: errorHandler,
	}

	// Start background monitoring if enabled
	if config.Health.Enabled {
		go service.startHealthMonitoring()
	}

	return service, nil
}

// Logger returns the structured logger
func (m *MonitoringService) Logger() *logging.StructuredLogger {
	return m.logger
}

// Sentry returns the Sentry monitor
func (m *MonitoringService) Sentry() *SentryMonitor {
	return m.sentry
}

// Telegram returns the Telegram monitor
func (m *MonitoringService) Telegram() *TelegramMonitor {
	return m.telegram
}

// Health returns the health monitor
func (m *MonitoringService) Health() *HealthMonitor {
	return m.health
}

// ErrorHandler returns the error handler
func (m *MonitoringService) ErrorHandler() *common.ErrorHandler {
	return m.errorHandler
}

// LogInfo logs an info message with context
func (m *MonitoringService) LogInfo(ctx context.Context, msg string, fields map[string]interface{}) {
	m.logger.Info(ctx, msg, fields)
}

// LogWarn logs a warning message with context
func (m *MonitoringService) LogWarn(ctx context.Context, msg string, fields map[string]interface{}) {
	m.logger.Warn(ctx, msg, fields)
}

// LogError logs an error message with context
func (m *MonitoringService) LogError(ctx context.Context, msg string, fields map[string]interface{}) {
	m.logger.Error(ctx, msg, fields)
}

// LogFatal logs a fatal message and exits
func (m *MonitoringService) LogFatal(ctx context.Context, msg string, fields map[string]interface{}) {
	m.logger.Fatal(ctx, msg, fields)
}

// CaptureError captures an error with full context
func (m *MonitoringService) CaptureError(ctx context.Context, err error, context map[string]interface{}) {
	// Log the error
	m.LogError(ctx, "Error captured", map[string]interface{}{
		"error":   err.Error(),
		"context": context,
	})

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CaptureError(ctx, err, context)
	}

	// Handle through error handler for alerts
	m.errorHandler.HandleError(ctx, err, context)
}

// CaptureBusinessError captures a business error
func (m *MonitoringService) CaptureBusinessError(ctx context.Context, err *common.BusinessError) {
	// Log the error
	m.LogWarn(ctx, "Business error captured", map[string]interface{}{
		"error_type": err.Type,
		"error_code": err.Code,
		"message":    err.Message,
		"context":    err.Context,
	})

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CaptureBusinessError(ctx, err)
	}

	// Send Telegram alert for business errors
	if m.telegram != nil && m.telegram.IsEnabled() {
		m.telegram.SendBusinessAlert(ctx, err.Type, err.Message, err.Context)
	}
}

// CaptureSystemError captures a system error
func (m *MonitoringService) CaptureSystemError(ctx context.Context, err *common.SystemError) {
	// Log the error
	logLevel := "error"
	if err.Severity == common.SeverityLow {
		logLevel = "warn"
	}

	if logLevel == "error" {
		m.LogError(ctx, "System error captured", map[string]interface{}{
			"error_type": err.Type,
			"severity":   err.Severity,
			"context":    err.Context,
		})
	} else {
		m.LogWarn(ctx, "System error captured", map[string]interface{}{
			"error_type": err.Type,
			"severity":   err.Severity,
			"context":    err.Context,
		})
	}

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CaptureSystemError(ctx, err)
	}

	// Send Telegram alert for critical system errors
	if err.ShouldAlert && m.telegram != nil && m.telegram.IsEnabled() {
		m.telegram.SendCriticalAlert(ctx, fmt.Sprintf("System Error: %s", err.Type), err.Error(), err.Context)
	}
}

// CaptureRetryableError captures a retryable error
func (m *MonitoringService) CaptureRetryableError(ctx context.Context, err *common.RetryableError) {
	// Log the error
	m.LogWarn(ctx, "Retryable error captured", map[string]interface{}{
		"error_type":    err.ErrorType,
		"severity":      err.Severity,
		"current_retry": err.CurrentRetry,
		"max_retries":   err.MaxRetries,
		"context":       err.Context,
	})

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CaptureRetryableError(ctx, err)
	}

	// Send Telegram alert for retryable errors that should alert
	if err.ShouldAlert && m.telegram != nil && m.telegram.IsEnabled() {
		m.telegram.SendErrorAlert(ctx, err, err.Context)
	}
}

// CapturePerformanceMetric captures a performance metric
func (m *MonitoringService) CapturePerformanceMetric(ctx context.Context, name string, value float64, unit string, tags map[string]string) {
	// Log the metric
	m.LogInfo(ctx, "Performance metric captured", map[string]interface{}{
		"metric_name": name,
		"value":       value,
		"unit":        unit,
		"tags":        tags,
	})

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CapturePerformanceMetric(ctx, name, value, unit, tags)
	}
}

// CaptureCustomEvent captures a custom event
func (m *MonitoringService) CaptureCustomEvent(ctx context.Context, eventType string, data map[string]interface{}) {
	// Log the event
	m.LogInfo(ctx, "Custom event captured", map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CaptureCustomEvent(ctx, eventType, data)
	}
}

// SendQuotaAlert sends a quota alert
func (m *MonitoringService) SendQuotaAlert(ctx context.Context, resource string, usage int, limit int, userID string) {
	// Log the quota alert
	m.LogWarn(ctx, "Quota alert triggered", map[string]interface{}{
		"resource": resource,
		"usage":    usage,
		"limit":    limit,
		"user_id":  userID,
	})

	// Send Telegram alert
	if m.telegram != nil && m.telegram.IsEnabled() {
		m.telegram.SendQuotaAlert(ctx, resource, usage, limit, userID)
	}
}

// SendSecurityAlert sends a security alert
func (m *MonitoringService) SendSecurityAlert(ctx context.Context, event string, details string, context map[string]interface{}) {
	// Log the security alert
	m.LogError(ctx, "Security alert triggered", map[string]interface{}{
		"event":   event,
		"details": details,
		"context": context,
	})

	// Send to Sentry
	if m.sentry != nil {
		m.sentry.CaptureMessage(ctx, fmt.Sprintf("Security Alert: %s - %s", event, details), sentry.LevelError, context)
	}

	// Send Telegram alert
	if m.telegram != nil && m.telegram.IsEnabled() {
		m.telegram.SendSecurityAlert(ctx, event, details, context)
	}
}

// SendSystemHealthAlert sends a system health alert
func (m *MonitoringService) SendSystemHealthAlert(ctx context.Context, component string, status string, details string) {
	// Log the health alert
	m.LogWarn(ctx, "System health alert triggered", map[string]interface{}{
		"component": component,
		"status":    status,
		"details":   details,
	})

	// Send Telegram alert
	if m.telegram != nil && m.telegram.IsEnabled() {
		m.telegram.SendSystemHealthAlert(ctx, component, status, details)
	}
}

// startHealthMonitoring starts background health monitoring
func (m *MonitoringService) startHealthMonitoring() {
	ticker := time.NewTicker(m.config.Health.CheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), m.config.Health.Timeout)

		health := m.health.GetHealth(ctx)

		// Check for unhealthy components
		for _, check := range health.Checks {
			if check.Status == HealthStatusUnhealthy {
				m.SendSystemHealthAlert(ctx, check.Name, "unhealthy", check.Message)
			}
		}

		cancel()
	}
}

// Close closes the monitoring service
func (m *MonitoringService) Close() {
	if m.sentry != nil {
		m.sentry.Close()
	}
}

// GetDefaultMonitoringConfig returns default monitoring configuration
func GetDefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		Sentry:   GetDefaultSentryConfig(),
		Telegram: GetDefaultTelegramConfig(),
		Logging:  logging.GetDefaultLoggerConfig(),
		Health: HealthConfig{
			Enabled:       true,
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}
}
