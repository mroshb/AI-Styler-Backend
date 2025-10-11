package monitoring

import (
	"context"
	"testing"
	"time"

	"ai-styler/internal/common"
	"ai-styler/internal/logging"
)

// Define custom types for context keys to avoid SA1029 warnings
type userIDKey struct{}
type conversionIDKey struct{}
type traceIDKey struct{}

func TestStructuredLogger(t *testing.T) {
	config := logging.LoggerConfig{
		Level:       logging.LogLevelInfo,
		Format:      "json",
		Output:      "stdout",
		Service:     "test-service",
		Version:     "1.0.0",
		Environment: "test",
	}

	logger := logging.NewStructuredLogger(config)

	ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-123")
	ctx = context.WithValue(ctx, conversionIDKey{}, "test-conv-456")

	// Test info logging
	logger.Info(ctx, "Test info message", map[string]interface{}{
		"test_field": "test_value",
		"number":     42,
	})

	// Test warning logging
	logger.Warn(ctx, "Test warning message", map[string]interface{}{
		"warning_type": "test_warning",
	})

	// Test error logging
	logger.Error(ctx, "Test error message", map[string]interface{}{
		"error_code": "TEST_ERROR",
		"details":    "This is a test error",
	})
}

func TestSentryMonitor(t *testing.T) {
	config := SentryConfig{
		DSN:              "", // Empty DSN for testing
		Environment:      "test",
		Release:          "1.0.0",
		Debug:            false,
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
		AttachStacktrace: true,
		MaxBreadcrumbs:   50,
	}

	monitor, err := NewSentryMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create Sentry monitor: %v", err)
	}

	ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-123")
	ctx = context.WithValue(ctx, conversionIDKey{}, "test-conv-456")

	// Test error capture
	testErr := &common.SystemError{
		Err:      common.NewGeminiAPIError("test_operation", nil),
		Type:     common.ErrorTypeGeminiAPI,
		Severity: common.SeverityHigh,
		Context: map[string]interface{}{
			"operation": "test_operation",
			"user_id":   "test-user-123",
		},
		ShouldAlert: true,
	}

	monitor.CaptureSystemError(ctx, testErr)

	// Test business error capture
	businessErr := &common.BusinessError{
		Err:     common.NewQuotaExceededError("conversions", 10),
		Type:    common.ErrorTypeQuotaExceeded,
		Code:    "QUOTA_EXCEEDED",
		Message: "Monthly conversion quota exceeded",
		Context: map[string]interface{}{
			"quota_type": "conversions",
			"limit":      10,
			"usage":      15,
		},
	}

	monitor.CaptureBusinessError(ctx, businessErr)

	// Test retryable error capture
	retryableErr := &common.RetryableError{
		Err:          common.NewTimeoutError("test_operation", 30*time.Second),
		MaxRetries:   3,
		CurrentRetry: 1,
		ErrorType:    common.ErrorTypeTimeout,
		Severity:     common.SeverityMedium,
		Context: map[string]interface{}{
			"operation": "test_operation",
			"timeout":   "30s",
		},
		ShouldAlert: false,
	}

	monitor.CaptureRetryableError(ctx, retryableErr)

	// Test performance metric capture
	monitor.CapturePerformanceMetric(ctx, "test_metric", 150.5, "ms", map[string]string{
		"operation": "test_operation",
		"success":   "true",
	})

	// Test custom event capture
	monitor.CaptureCustomEvent(ctx, "test_event", map[string]interface{}{
		"event_data": "test_value",
		"timestamp":  time.Now(),
	})

	// Test breadcrumb
	monitor.AddBreadcrumb(ctx, "Test breadcrumb", "test", "info", map[string]interface{}{
		"breadcrumb_data": "test_value",
	})

	// Test transaction
	transaction := monitor.StartTransaction(ctx, "test_transaction", "test_operation")
	if transaction != nil {
		transaction.Finish()
	}

	// Test user setting
	monitor.SetUser(ctx, "test-user-123", "test@example.com", "testuser")

	// Test tag setting
	monitor.SetTag(ctx, "test_tag", "test_value")

	// Test context setting
	monitor.SetContext(ctx, "test_context", map[string]interface{}{
		"context_data": "test_value",
	})

	// Test flush
	flushed := monitor.Flush(5 * time.Second)
	if !flushed {
		t.Log("Sentry flush returned false (expected with empty DSN)")
	}

	// Test close
	monitor.Close()
}

func TestTelegramMonitor(t *testing.T) {
	config := TelegramConfig{
		BotToken: "", // Empty token for testing
		ChatID:   "",
		Enabled:  false,
		Timeout:  10 * time.Second,
	}

	monitor := NewTelegramMonitor(config)

	ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-123")
	ctx = context.WithValue(ctx, conversionIDKey{}, "test-conv-456")

	// Test critical alert
	err := monitor.SendCriticalAlert(ctx, "Test Critical Alert", "This is a test critical alert", map[string]interface{}{
		"alert_type": "critical",
		"test_data":  "test_value",
	})
	if err != nil {
		t.Logf("Critical alert error (expected with empty config): %v", err)
	}

	// Test error alert
	testErr := &common.SystemError{
		Err:      common.NewGeminiAPIError("test_operation", nil),
		Type:     common.ErrorTypeGeminiAPI,
		Severity: common.SeverityHigh,
		Context: map[string]interface{}{
			"operation": "test_operation",
		},
		ShouldAlert: true,
	}

	err = monitor.SendErrorAlert(ctx, testErr, map[string]interface{}{
		"error_context": "test_context",
	})
	if err != nil {
		t.Logf("Error alert error (expected with empty config): %v", err)
	}

	// Test business alert
	err = monitor.SendBusinessAlert(ctx, common.ErrorTypeValidation, "Test business alert", map[string]interface{}{
		"validation_error": "test_validation",
	})
	if err != nil {
		t.Logf("Business alert error (expected with empty config): %v", err)
	}

	// Test performance alert
	err = monitor.SendPerformanceAlert(ctx, "response_time", 5000, 1000, map[string]interface{}{
		"threshold_exceeded": true,
	})
	if err != nil {
		t.Logf("Performance alert error (expected with empty config): %v", err)
	}

	// Test quota alert
	err = monitor.SendQuotaAlert(ctx, "conversions", 15, 10, "test-user-123")
	if err != nil {
		t.Logf("Quota alert error (expected with empty config): %v", err)
	}

	// Test security alert
	err = monitor.SendSecurityAlert(ctx, "Suspicious Activity", "Multiple failed login attempts", map[string]interface{}{
		"ip_address": "192.168.1.1",
		"attempts":   5,
	})
	if err != nil {
		t.Logf("Security alert error (expected with empty config): %v", err)
	}

	// Test system health alert
	err = monitor.SendSystemHealthAlert(ctx, "database", "unhealthy", "Connection timeout")
	if err != nil {
		t.Logf("System health alert error (expected with empty config): %v", err)
	}

	// Test daily report
	stats := map[string]interface{}{
		"total_requests":        1000,
		"successful_requests":   950,
		"failed_requests":       50,
		"average_response_time": "150ms",
	}

	err = monitor.SendDailyReport(ctx, stats)
	if err != nil {
		t.Logf("Daily report error (expected with empty config): %v", err)
	}

	// Test enabled status
	if monitor.IsEnabled() {
		t.Error("Expected monitor to be disabled with empty config")
	}
}

func TestHealthMonitor(t *testing.T) {
	monitor := NewHealthMonitor("1.0.0", "test")

	// Test system health checker
	systemChecker := &SystemHealthChecker{}
	monitor.AddChecker("system", systemChecker)

	ctx := context.Background()
	health := monitor.GetHealth(ctx)

	if health.Status != HealthStatusHealthy && health.Status != HealthStatusDegraded {
		t.Errorf("Expected healthy or degraded status, got %s", health.Status)
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", health.Version)
	}

	if len(health.Checks) != 1 {
		t.Errorf("Expected 1 health check, got %d", len(health.Checks))
	}

	if health.Checks[0].Name != "system" {
		t.Errorf("Expected system check, got %s", health.Checks[0].Name)
	}

	// Test system info
	systemInfo := monitor.GetSystemInfo()
	if systemInfo.GoVersion == "" {
		t.Error("Expected Go version to be set")
	}

	if systemInfo.NumCPU <= 0 {
		t.Error("Expected NumCPU to be positive")
	}

	if systemInfo.NumGoroutine < 0 {
		t.Error("Expected NumGoroutine to be non-negative")
	}
}

func TestMonitoringService(t *testing.T) {
	config := MonitoringConfig{
		Sentry: SentryConfig{
			DSN:              "",
			Environment:      "test",
			Release:          "1.0.0",
			Debug:            false,
			SampleRate:       1.0,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			MaxBreadcrumbs:   50,
		},
		Telegram: TelegramConfig{
			BotToken: "",
			ChatID:   "",
			Enabled:  false,
			Timeout:  10 * time.Second,
		},
		Logging: logging.LoggerConfig{
			Level:       logging.LogLevelInfo,
			Format:      "json",
			Output:      "stdout",
			Service:     "test-service",
			Version:     "1.0.0",
			Environment: "test",
		},
		Health: HealthConfig{
			Enabled:       false, // Disable for testing
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}

	monitor, err := NewMonitoringService(config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create monitoring service: %v", err)
	}
	defer monitor.Close()

	ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-123")
	ctx = context.WithValue(ctx, conversionIDKey{}, "test-conv-456")

	// Test logging methods
	monitor.LogInfo(ctx, "Test info message", map[string]interface{}{
		"test_field": "test_value",
	})

	monitor.LogWarn(ctx, "Test warning message", map[string]interface{}{
		"warning_type": "test_warning",
	})

	monitor.LogError(ctx, "Test error message", map[string]interface{}{
		"error_code": "TEST_ERROR",
	})

	// Test error capture
	testErr := &common.SystemError{
		Err:      common.NewGeminiAPIError("test_operation", nil),
		Type:     common.ErrorTypeGeminiAPI,
		Severity: common.SeverityHigh,
		Context: map[string]interface{}{
			"operation": "test_operation",
		},
		ShouldAlert: true,
	}

	monitor.CaptureSystemError(ctx, testErr)

	// Test business error capture
	businessErr := &common.BusinessError{
		Err:     common.NewQuotaExceededError("conversions", 10),
		Type:    common.ErrorTypeQuotaExceeded,
		Code:    "QUOTA_EXCEEDED",
		Message: "Monthly conversion quota exceeded",
		Context: map[string]interface{}{
			"quota_type": "conversions",
			"limit":      10,
			"usage":      15,
		},
	}

	monitor.CaptureBusinessError(ctx, businessErr)

	// Test retryable error capture
	retryableErr := &common.RetryableError{
		Err:          common.NewTimeoutError("test_operation", 30*time.Second),
		MaxRetries:   3,
		CurrentRetry: 1,
		ErrorType:    common.ErrorTypeTimeout,
		Severity:     common.SeverityMedium,
		Context: map[string]interface{}{
			"operation": "test_operation",
			"timeout":   "30s",
		},
		ShouldAlert: false,
	}

	monitor.CaptureRetryableError(ctx, retryableErr)

	// Test performance metric capture
	monitor.CapturePerformanceMetric(ctx, "test_metric", 150.5, "ms", map[string]string{
		"operation": "test_operation",
		"success":   "true",
	})

	// Test custom event capture
	monitor.CaptureCustomEvent(ctx, "test_event", map[string]interface{}{
		"event_data": "test_value",
		"timestamp":  time.Now(),
	})

	// Test quota alert
	monitor.SendQuotaAlert(ctx, "conversions", 15, 10, "test-user-123")

	// Test security alert
	monitor.SendSecurityAlert(ctx, "Suspicious Activity", "Multiple failed login attempts", map[string]interface{}{
		"ip_address": "192.168.1.1",
		"attempts":   5,
	})

	// Test system health alert
	monitor.SendSystemHealthAlert(ctx, "database", "unhealthy", "Connection timeout")

	// Test component access
	if monitor.Logger() == nil {
		t.Error("Expected logger to be available")
	}

	if monitor.Sentry() == nil {
		t.Error("Expected Sentry monitor to be available")
	}

	if monitor.Telegram() == nil {
		t.Error("Expected Telegram monitor to be available")
	}

	if monitor.Health() == nil {
		t.Error("Expected health monitor to be available")
	}

	if monitor.ErrorHandler() == nil {
		t.Error("Expected error handler to be available")
	}
}

func TestHealthCheckers(t *testing.T) {
	// Test system health checker
	systemChecker := &SystemHealthChecker{}
	ctx := context.Background()

	check := systemChecker.Check(ctx)
	if check.Name != "" {
		t.Error("Expected empty name for system checker")
	}

	if check.Status != HealthStatusHealthy && check.Status != HealthStatusDegraded {
		t.Errorf("Expected healthy or degraded status, got %s", check.Status)
	}

	if check.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	if check.LastChecked.IsZero() {
		t.Error("Expected LastChecked to be set")
	}

	if check.Details == nil {
		t.Error("Expected details to be set")
	}

	// Check for expected details
	expectedDetails := []string{"memory_usage_percent", "goroutine_count", "num_cpu", "go_version"}
	for _, detail := range expectedDetails {
		if _, exists := check.Details[detail]; !exists {
			t.Errorf("Expected detail %s to be present", detail)
		}
	}
}

func TestMonitoringIntegration(t *testing.T) {
	// Test the complete monitoring flow
	config := MonitoringConfig{
		Sentry: SentryConfig{
			DSN:              "",
			Environment:      "test",
			Release:          "1.0.0",
			Debug:            false,
			SampleRate:       1.0,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			MaxBreadcrumbs:   50,
		},
		Telegram: TelegramConfig{
			BotToken: "",
			ChatID:   "",
			Enabled:  false,
			Timeout:  10 * time.Second,
		},
		Logging: logging.LoggerConfig{
			Level:       logging.LogLevelInfo,
			Format:      "json",
			Output:      "stdout",
			Service:     "test-service",
			Version:     "1.0.0",
			Environment: "test",
		},
		Health: HealthConfig{
			Enabled:       false,
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}

	monitor, err := NewMonitoringService(config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create monitoring service: %v", err)
	}
	defer monitor.Close()

	ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-123")
	ctx = context.WithValue(ctx, conversionIDKey{}, "test-conv-456")
	ctx = context.WithValue(ctx, traceIDKey{}, "test-trace-789")

	// Simulate a complete request flow
	monitor.LogInfo(ctx, "Request started", map[string]interface{}{
		"method": "POST",
		"path":   "/api/conversions",
	})

	// Simulate processing
	monitor.CapturePerformanceMetric(ctx, "request_processing", 150.5, "ms", map[string]string{
		"operation": "image_conversion",
	})

	// Simulate success
	monitor.LogInfo(ctx, "Request completed successfully", map[string]interface{}{
		"status_code": 200,
		"duration_ms": 150.5,
	})

	// Test error scenario
	testErr := &common.SystemError{
		Err:      common.NewGeminiAPIError("image_processing", nil),
		Type:     common.ErrorTypeGeminiAPI,
		Severity: common.SeverityHigh,
		Context: map[string]interface{}{
			"operation":       "image_processing",
			"image_size":      "2MB",
			"processing_time": "5s",
		},
		ShouldAlert: true,
	}

	monitor.CaptureSystemError(ctx, testErr)

	// Test quota scenario
	monitor.SendQuotaAlert(ctx, "monthly_conversions", 95, 100, "test-user-123")

	// Test health check
	health := monitor.Health().GetHealth(ctx)
	if health.Status == "" {
		t.Error("Expected health status to be set")
	}

	// Test system info
	systemInfo := monitor.Health().GetSystemInfo()
	if systemInfo.GoVersion == "" {
		t.Error("Expected system info to be available")
	}
}
