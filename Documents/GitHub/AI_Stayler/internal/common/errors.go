package common

import (
	"context"
	"fmt"
	"time"
)

// ErrorType represents different types of errors in the system
type ErrorType string

const (
	// System Errors
	ErrorTypeSystem        ErrorType = "system"
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeTimeout       ErrorType = "timeout"
	ErrorTypeRateLimit     ErrorType = "rate_limit"
	ErrorTypeQuotaExceeded ErrorType = "quota_exceeded"

	// Business Logic Errors
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeForbidden    ErrorType = "forbidden"

	// External Service Errors
	ErrorTypeGeminiAPI      ErrorType = "gemini_api"
	ErrorTypePaymentGateway ErrorType = "payment_gateway"
	ErrorTypeStorage        ErrorType = "storage"
	ErrorTypeNotification   ErrorType = "notification"

	// File/Data Errors
	ErrorTypeFileCorrupted ErrorType = "file_corrupted"
	ErrorTypeFileMissing   ErrorType = "file_missing"
	ErrorTypeFileTooLarge  ErrorType = "file_too_large"
	ErrorTypeInvalidFormat ErrorType = "invalid_format"
)

// SeverityLevel represents the severity of an error
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err          error
	RetryAfter   time.Duration
	MaxRetries   int
	CurrentRetry int
	ErrorType    ErrorType
	Severity     SeverityLevel
	Context      map[string]interface{}
	ShouldAlert  bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is retryable
func (e *RetryableError) IsRetryable() bool {
	return e.CurrentRetry < e.MaxRetries
}

// GetRetryDelay calculates the retry delay with exponential backoff
func (e *RetryableError) GetRetryDelay() time.Duration {
	if e.RetryAfter > 0 {
		return e.RetryAfter
	}

	// Exponential backoff: 2^retry * base delay
	baseDelay := time.Second
	delay := time.Duration(1<<uint(e.CurrentRetry)) * baseDelay

	// Cap at 5 minutes
	maxDelay := 5 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// SystemError represents system-level errors
type SystemError struct {
	Err         error
	Type        ErrorType
	Severity    SeverityLevel
	Context     map[string]interface{}
	ShouldAlert bool
}

func (e *SystemError) Error() string {
	return e.Err.Error()
}

func (e *SystemError) Unwrap() error {
	return e.Err
}

// BusinessError represents business logic errors
type BusinessError struct {
	Err     error
	Type    ErrorType
	Code    string
	Message string
	Context map[string]interface{}
}

func (e *BusinessError) Error() string {
	return e.Err.Error()
}

func (e *BusinessError) Unwrap() error {
	return e.Err
}

// ErrorHandler handles different types of errors
type ErrorHandler struct {
	alertService AlertService
	logger       Logger
	config       ErrorConfig
}

// AlertService interface for sending alerts
type AlertService interface {
	SendAlert(ctx context.Context, alert Alert) error
}

// Logger interface for logging
type Logger interface {
	Error(ctx context.Context, msg string, fields map[string]interface{})
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	Info(ctx context.Context, msg string, fields map[string]interface{})
}

// Alert represents an alert to be sent
type Alert struct {
	Type         ErrorType
	Severity     SeverityLevel
	Title        string
	Message      string
	Context      map[string]interface{}
	Timestamp    time.Time
	Service      string
	UserID       *string
	ConversionID *string
}

// ErrorConfig represents error handling configuration
type ErrorConfig struct {
	MaxRetries      int
	BaseRetryDelay  time.Duration
	MaxRetryDelay   time.Duration
	AlertThresholds map[SeverityLevel]bool
	LogLevels       map[ErrorType]string
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(alertService AlertService, logger Logger, config ErrorConfig) *ErrorHandler {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.BaseRetryDelay == 0 {
		config.BaseRetryDelay = time.Second
	}
	if config.MaxRetryDelay == 0 {
		config.MaxRetryDelay = 5 * time.Minute
	}
	if config.AlertThresholds == nil {
		config.AlertThresholds = map[SeverityLevel]bool{
			SeverityHigh:     true,
			SeverityCritical: true,
		}
	}
	if config.LogLevels == nil {
		config.LogLevels = map[ErrorType]string{
			ErrorTypeSystem:         "error",
			ErrorTypeNetwork:        "warn",
			ErrorTypeTimeout:        "warn",
			ErrorTypeRateLimit:      "info",
			ErrorTypeQuotaExceeded:  "info",
			ErrorTypeValidation:     "warn",
			ErrorTypeNotFound:       "info",
			ErrorTypeUnauthorized:   "warn",
			ErrorTypeForbidden:      "warn",
			ErrorTypeGeminiAPI:      "error",
			ErrorTypePaymentGateway: "error",
			ErrorTypeStorage:        "error",
			ErrorTypeNotification:   "warn",
			ErrorTypeFileCorrupted:  "error",
			ErrorTypeFileMissing:    "warn",
			ErrorTypeFileTooLarge:   "warn",
			ErrorTypeInvalidFormat:  "warn",
		}
	}

	return &ErrorHandler{
		alertService: alertService,
		logger:       logger,
		config:       config,
	}
}

// HandleError handles different types of errors
func (h *ErrorHandler) HandleError(ctx context.Context, err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	// Determine error type and severity
	errorType, severity := h.classifyError(err)

	// Log the error
	h.logError(ctx, err, errorType, severity, context)

	// Send alert if needed
	if h.shouldAlert(severity) {
		h.sendAlert(ctx, err, errorType, severity, context)
	}

	// Wrap error with additional context
	return h.wrapError(err, errorType, severity, context)
}

// HandleRetryableError handles retryable errors
func (h *ErrorHandler) HandleRetryableError(ctx context.Context, err error, context map[string]interface{}, currentRetry int) *RetryableError {
	errorType, severity := h.classifyError(err)

	retryableErr := &RetryableError{
		Err:          err,
		MaxRetries:   h.config.MaxRetries,
		CurrentRetry: currentRetry,
		ErrorType:    errorType,
		Severity:     severity,
		Context:      context,
		ShouldAlert:  h.shouldAlert(severity),
	}

	// Log the error
	h.logError(ctx, err, errorType, severity, context)

	// Send alert if needed
	if retryableErr.ShouldAlert {
		h.sendAlert(ctx, err, errorType, severity, context)
	}

	return retryableErr
}

// classifyError classifies an error by type and severity
func (h *ErrorHandler) classifyError(err error) (ErrorType, SeverityLevel) {
	errStr := err.Error()

	// Network and timeout errors
	if contains(errStr, "timeout") || contains(errStr, "deadline exceeded") {
		return ErrorTypeTimeout, SeverityMedium
	}
	if contains(errStr, "connection") || contains(errStr, "network") {
		return ErrorTypeNetwork, SeverityMedium
	}

	// Rate limiting
	if contains(errStr, "rate limit") || contains(errStr, "too many requests") {
		return ErrorTypeRateLimit, SeverityLow
	}

	// Quota errors
	if contains(errStr, "quota exceeded") || contains(errStr, "quota limit") {
		return ErrorTypeQuotaExceeded, SeverityLow
	}

	// Gemini API errors
	if contains(errStr, "gemini") || contains(errStr, "generative") {
		return ErrorTypeGeminiAPI, SeverityHigh
	}

	// Payment gateway errors
	if contains(errStr, "payment") || contains(errStr, "gateway") || contains(errStr, "zarinpal") {
		return ErrorTypePaymentGateway, SeverityHigh
	}

	// Storage errors
	if contains(errStr, "storage") || contains(errStr, "upload") || contains(errStr, "download") {
		return ErrorTypeStorage, SeverityHigh
	}

	// File errors
	if contains(errStr, "file") && contains(errStr, "corrupt") {
		return ErrorTypeFileCorrupted, SeverityHigh
	}
	if contains(errStr, "file") && contains(errStr, "missing") {
		return ErrorTypeFileMissing, SeverityMedium
	}
	if contains(errStr, "file") && contains(errStr, "too large") {
		return ErrorTypeFileTooLarge, SeverityMedium
	}
	if contains(errStr, "invalid format") || contains(errStr, "unsupported format") {
		return ErrorTypeInvalidFormat, SeverityMedium
	}

	// Authentication/Authorization
	if contains(errStr, "unauthorized") {
		return ErrorTypeUnauthorized, SeverityMedium
	}
	if contains(errStr, "forbidden") {
		return ErrorTypeForbidden, SeverityMedium
	}

	// Validation errors
	if contains(errStr, "validation") || contains(errStr, "invalid") {
		return ErrorTypeValidation, SeverityLow
	}

	// Not found errors
	if contains(errStr, "not found") {
		return ErrorTypeNotFound, SeverityLow
	}

	// Default to system error
	return ErrorTypeSystem, SeverityMedium
}

// logError logs an error with appropriate level
func (h *ErrorHandler) logError(ctx context.Context, err error, errorType ErrorType, severity SeverityLevel, context map[string]interface{}) {
	fields := map[string]interface{}{
		"error_type": errorType,
		"severity":   severity,
		"error":      err.Error(),
	}

	// Add context fields
	for k, v := range context {
		fields[k] = v
	}

	logLevel := h.config.LogLevels[errorType]
	switch logLevel {
	case "error":
		h.logger.Error(ctx, "Error occurred", fields)
	case "warn":
		h.logger.Warn(ctx, "Warning occurred", fields)
	case "info":
		h.logger.Info(ctx, "Info message", fields)
	default:
		h.logger.Error(ctx, "Error occurred", fields)
	}
}

// sendAlert sends an alert if needed
func (h *ErrorHandler) sendAlert(ctx context.Context, err error, errorType ErrorType, severity SeverityLevel, context map[string]interface{}) {
	if h.alertService == nil {
		return
	}

	alert := Alert{
		Type:      errorType,
		Severity:  severity,
		Title:     fmt.Sprintf("%s Error", string(errorType)),
		Message:   err.Error(),
		Context:   context,
		Timestamp: time.Now(),
		Service:   "ai-stayler",
	}

	// Extract user and conversion IDs from context
	if userID, ok := context["user_id"].(string); ok {
		alert.UserID = &userID
	}
	if conversionID, ok := context["conversion_id"].(string); ok {
		alert.ConversionID = &conversionID
	}

	if err := h.alertService.SendAlert(ctx, alert); err != nil {
		h.logger.Error(ctx, "Failed to send alert", map[string]interface{}{
			"alert_error":    err.Error(),
			"original_error": err.Error(),
		})
	}
}

// shouldAlert determines if an alert should be sent
func (h *ErrorHandler) shouldAlert(severity SeverityLevel) bool {
	return h.config.AlertThresholds[severity]
}

// wrapError wraps an error with additional context
func (h *ErrorHandler) wrapError(err error, errorType ErrorType, severity SeverityLevel, context map[string]interface{}) error {
	return &SystemError{
		Err:         err,
		Type:        errorType,
		Severity:    severity,
		Context:     context,
		ShouldAlert: h.shouldAlert(severity),
	}
}

// Common error constructors
func NewTimeoutError(operation string, timeout time.Duration) error {
	return fmt.Errorf("operation '%s' timed out after %v", operation, timeout)
}

func NewQuotaExceededError(resource string, limit int) error {
	return fmt.Errorf("quota exceeded for %s: limit %d", resource, limit)
}

func NewFileCorruptedError(filename string) error {
	return fmt.Errorf("file '%s' is corrupted or invalid", filename)
}

func NewFileMissingError(filename string) error {
	return fmt.Errorf("file '%s' is missing", filename)
}

func NewGeminiAPIError(operation string, err error) error {
	return fmt.Errorf("gemini API error during %s: %w", operation, err)
}

func NewPaymentGatewayError(operation string, err error) error {
	return fmt.Errorf("payment gateway error during %s: %w", operation, err)
}

func NewStorageError(operation string, err error) error {
	return fmt.Errorf("storage error during %s: %w", operation, err)
}
