package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"ai-styler/internal/common"

	"github.com/getsentry/sentry-go"
)

// SentryConfig represents Sentry configuration
type SentryConfig struct {
	DSN              string
	Environment      string
	Release          string
	Debug            bool
	SampleRate       float64
	TracesSampleRate float64
	AttachStacktrace bool
	MaxBreadcrumbs   int
}

// SentryMonitor provides Sentry integration for error tracking
type SentryMonitor struct {
	config SentryConfig
	hub    *sentry.Hub
}

// NewSentryMonitor creates a new Sentry monitor
func NewSentryMonitor(config SentryConfig) (*SentryMonitor, error) {
	if config.DSN == "" {
		return &SentryMonitor{}, nil // Return empty monitor if no DSN
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		Debug:            config.Debug,
		SampleRate:       config.SampleRate,
		TracesSampleRate: config.TracesSampleRate,
		AttachStacktrace: config.AttachStacktrace,
		MaxBreadcrumbs:   config.MaxBreadcrumbs,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Add custom tags and context
			event.Tags["service"] = "ai-stayler"
			event.Tags["component"] = "backend"
			return event
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	return &SentryMonitor{
		config: config,
		hub:    sentry.CurrentHub(),
	}, nil
}

// CaptureError captures an error with context
func (s *SentryMonitor) CaptureError(ctx context.Context, err error, context map[string]interface{}) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetContext("error_context", context)

	// Extract user information
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			scope.SetUser(sentry.User{ID: id})
		}
	}

	// Extract vendor information
	if vendorID := ctx.Value("vendor_id"); vendorID != nil {
		if id, ok := vendorID.(string); ok {
			scope.SetTag("vendor_id", id)
		}
	}

	// Extract conversion information
	if conversionID := ctx.Value("conversion_id"); conversionID != nil {
		if id, ok := conversionID.(string); ok {
			scope.SetTag("conversion_id", id)
		}
	}

	// Extract trace information
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			scope.SetTag("trace_id", id)
		}
	}

	// Add custom tags based on error type
	if systemErr, ok := err.(*common.SystemError); ok {
		scope.SetTag("error_type", string(systemErr.Type))
		scope.SetTag("severity", string(systemErr.Severity))
		scope.SetLevel(sentry.Level(systemErr.Severity))
	}

	// Capture the error
	s.hub.CaptureException(err)
}

// CaptureMessage captures a message with context
func (s *SentryMonitor) CaptureMessage(ctx context.Context, message string, level sentry.Level, context map[string]interface{}) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetContext("message_context", context)
	scope.SetLevel(level)

	// Extract user information
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			scope.SetUser(sentry.User{ID: id})
		}
	}

	s.hub.CaptureMessage(message)
}

// CapturePanic captures a panic with context
func (s *SentryMonitor) CapturePanic(ctx context.Context, panic interface{}, context map[string]interface{}) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetContext("panic_context", context)
	scope.SetLevel(sentry.LevelFatal)

	// Extract user information
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			scope.SetUser(sentry.User{ID: id})
		}
	}

	s.hub.RecoverWithContext(ctx, panic)
}

// AddBreadcrumb adds a breadcrumb for debugging
func (s *SentryMonitor) AddBreadcrumb(ctx context.Context, message string, category string, level sentry.Level, data map[string]interface{}) {
	if s.hub == nil {
		return
	}

	breadcrumb := &sentry.Breadcrumb{
		Message:   message,
		Category:  category,
		Level:     level,
		Data:      data,
		Timestamp: time.Now(),
	}

	s.hub.AddBreadcrumb(breadcrumb, nil)
}

// StartTransaction starts a new transaction for performance monitoring
func (s *SentryMonitor) StartTransaction(ctx context.Context, name string, operation string) *sentry.Span {
	if s.hub == nil {
		return nil
	}

	transaction := sentry.StartTransaction(ctx, name, sentry.WithOpName(operation))
	return transaction
}

// SetUser sets user information for error tracking
func (s *SentryMonitor) SetUser(ctx context.Context, userID string, email string, username string) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetUser(sentry.User{
		ID:       userID,
		Email:    email,
		Username: username,
	})
}

// SetTag sets a tag for error tracking
func (s *SentryMonitor) SetTag(ctx context.Context, key string, value string) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetTag(key, value)
}

// SetContext sets context information for error tracking
func (s *SentryMonitor) SetContext(ctx context.Context, key string, value map[string]interface{}) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetContext(key, sentry.Context(value))
}

// Flush flushes pending events
func (s *SentryMonitor) Flush(timeout time.Duration) bool {
	if s.hub == nil {
		return true
	}

	return s.hub.Flush(timeout)
}

// Close closes the Sentry client
func (s *SentryMonitor) Close() {
	if s.hub != nil {
		s.hub.Flush(2 * time.Second)
	}
}

// GetDefaultSentryConfig returns default Sentry configuration
func GetDefaultSentryConfig() SentryConfig {
	return SentryConfig{
		Environment:      "development",
		Release:          "1.0.0",
		Debug:            false,
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
		AttachStacktrace: true,
		MaxBreadcrumbs:   50,
	}
}

// CaptureErrorWithStack captures an error with stack trace
func (s *SentryMonitor) CaptureErrorWithStack(ctx context.Context, err error, context map[string]interface{}) {
	if s.hub == nil {
		return
	}

	// Get stack trace
	stack := make([]byte, 4096)
	n := runtime.Stack(stack, false)
	stackTrace := string(stack[:n])

	// Add stack trace to context
	if context == nil {
		context = make(map[string]interface{})
	}
	context["stack_trace"] = stackTrace

	s.CaptureError(ctx, err, context)
}

// CaptureBusinessError captures a business error with specific context
func (s *SentryMonitor) CaptureBusinessError(ctx context.Context, err *common.BusinessError) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetTag("error_type", string(err.Type))
	scope.SetTag("error_code", err.Code)
	scope.SetLevel(sentry.LevelWarning)

	// Add business error context
	scope.SetContext("business_error", map[string]interface{}{
		"type":    err.Type,
		"code":    err.Code,
		"message": err.Message,
		"context": err.Context,
	})

	s.hub.CaptureException(err)
}

// CaptureSystemError captures a system error with specific context
func (s *SentryMonitor) CaptureSystemError(ctx context.Context, err *common.SystemError) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetTag("error_type", string(err.Type))
	scope.SetTag("severity", string(err.Severity))
	scope.SetLevel(sentry.Level(err.Severity))

	// Add system error context
	scope.SetContext("system_error", map[string]interface{}{
		"type":     err.Type,
		"severity": err.Severity,
		"context":  err.Context,
		"alert":    err.ShouldAlert,
	})

	s.hub.CaptureException(err)
}

// CaptureRetryableError captures a retryable error with retry context
func (s *SentryMonitor) CaptureRetryableError(ctx context.Context, err *common.RetryableError) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetTag("error_type", string(err.ErrorType))
	scope.SetTag("severity", string(err.Severity))
	scope.SetTag("retryable", "true")
	scope.SetTag("current_retry", fmt.Sprintf("%d", err.CurrentRetry))
	scope.SetTag("max_retries", fmt.Sprintf("%d", err.MaxRetries))
	scope.SetLevel(sentry.Level(err.Severity))

	// Add retry context
	scope.SetContext("retry_error", map[string]interface{}{
		"type":          err.ErrorType,
		"severity":      err.Severity,
		"current_retry": err.CurrentRetry,
		"max_retries":   err.MaxRetries,
		"retry_after":   err.RetryAfter.String(),
		"context":       err.Context,
		"alert":         err.ShouldAlert,
	})

	s.hub.CaptureException(err)
}

// CapturePerformanceMetric captures a performance metric
func (s *SentryMonitor) CapturePerformanceMetric(ctx context.Context, name string, value float64, unit string, tags map[string]string) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()

	// Add metric context
	metricContext := map[string]interface{}{
		"name":  name,
		"value": value,
		"unit":  unit,
		"tags":  tags,
	}

	scope.SetContext("performance_metric", metricContext)
	s.hub.CaptureMessage(fmt.Sprintf("Performance metric: %s = %.2f %s", name, value, unit))
}

// CaptureCustomEvent captures a custom event
func (s *SentryMonitor) CaptureCustomEvent(ctx context.Context, eventType string, data map[string]interface{}) {
	if s.hub == nil {
		return
	}

	scope := s.hub.Scope()
	scope.SetTag("event_type", eventType)
	scope.SetContext("custom_event", data)

	s.hub.CaptureMessage(fmt.Sprintf("Custom event: %s", eventType))
}
