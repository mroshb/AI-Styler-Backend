package crosscutting

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization  ErrorType = "authorization"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeQuotaExceeded  ErrorType = "quota_exceeded"
	ErrorTypeSystem         ErrorType = "system"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeSecurity       ErrorType = "security"
	ErrorTypePayment        ErrorType = "payment"
	ErrorTypeStorage        ErrorType = "storage"
	ErrorTypeExternal       ErrorType = "external"
)

// ErrorSeverity represents error severity levels
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// ErrorContext represents additional context for errors
type ErrorContext struct {
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Endpoint  string                 `json:"endpoint,omitempty"`
	Method    string                 `json:"method,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// APIError represents a structured API error
type APIError struct {
	Type        ErrorType              `json:"type"`
	Severity    ErrorSeverity          `json:"severity"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Details     string                 `json:"details,omitempty"`
	Context     *ErrorContext          `json:"context,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Retryable   bool                   `json:"retryable"`
	RetryAfter  *time.Duration         `json:"retry_after,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorHandlerConfig represents configuration for error handling
type ErrorHandlerConfig struct {
	// Error mapping
	DefaultSeverity ErrorSeverity `json:"default_severity"`

	// User-friendly messages
	ShowDetailedErrors bool `json:"show_detailed_errors"`
	IncludeStackTraces bool `json:"include_stack_traces"`

	// Error codes
	CustomErrorCodes map[string]string `json:"custom_error_codes"`

	// Retry configuration
	DefaultRetryable  bool          `json:"default_retryable"`
	DefaultRetryAfter time.Duration `json:"default_retry_after"`

	// Logging
	LogErrors      bool `json:"log_errors"`
	LogUserContext bool `json:"log_user_context"`

	// Alerting
	AlertOnErrors   bool                   `json:"alert_on_errors"`
	AlertThresholds map[ErrorSeverity]bool `json:"alert_thresholds"`
}

// DefaultErrorHandlerConfig returns default error handler configuration
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		DefaultSeverity:    ErrorSeverityMedium,
		ShowDetailedErrors: false,
		IncludeStackTraces: false,
		DefaultRetryable:   false,
		DefaultRetryAfter:  5 * time.Second,
		LogErrors:          true,
		LogUserContext:     true,
		AlertOnErrors:      true,
		AlertThresholds: map[ErrorSeverity]bool{
			ErrorSeverityLow:      false,
			ErrorSeverityMedium:   false,
			ErrorSeverityHigh:     true,
			ErrorSeverityCritical: true,
		},
		CustomErrorCodes: map[string]string{
			"INVALID_INPUT":          "The provided input is invalid",
			"AUTHENTICATION_FAILED":  "Authentication failed",
			"AUTHORIZATION_DENIED":   "You don't have permission to perform this action",
			"RESOURCE_NOT_FOUND":     "The requested resource was not found",
			"RATE_LIMIT_EXCEEDED":    "Too many requests. Please try again later",
			"QUOTA_EXCEEDED":         "You have exceeded your quota limit",
			"SYSTEM_ERROR":           "An internal system error occurred",
			"NETWORK_ERROR":          "A network error occurred",
			"TIMEOUT_ERROR":          "The request timed out",
			"SECURITY_THREAT":        "A security threat was detected",
			"PAYMENT_FAILED":         "Payment processing failed",
			"STORAGE_ERROR":          "Storage operation failed",
			"EXTERNAL_SERVICE_ERROR": "External service is unavailable",
		},
	}
}

// ErrorHandler provides comprehensive error handling functionality
type ErrorHandler struct {
	config  *ErrorHandlerConfig
	logger  *StructuredLogger
	alerter *AlertingService
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *ErrorHandlerConfig, logger *StructuredLogger, alerter *AlertingService) *ErrorHandler {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}

	return &ErrorHandler{
		config:  config,
		logger:  logger,
		alerter: alerter,
	}
}

// HandleError handles an error and returns a structured API error
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, errorType ErrorType, context *ErrorContext) *APIError {
	if err == nil {
		return nil
	}

	// Create API error
	apiError := &APIError{
		Type:      errorType,
		Severity:  eh.determineSeverity(errorType),
		Code:      eh.getErrorCode(errorType),
		Message:   eh.getUserFriendlyMessage(errorType),
		Details:   err.Error(),
		Context:   context,
		Timestamp: time.Now(),
		Retryable: eh.isRetryable(errorType),
		Metadata:  make(map[string]interface{}),
	}

	// Add retry after if retryable
	if apiError.Retryable {
		apiError.RetryAfter = &eh.config.DefaultRetryAfter
	}

	// Add suggestions
	apiError.Suggestions = eh.getSuggestions(errorType)

	// Log error if enabled
	if eh.config.LogErrors && eh.logger != nil {
		eh.logError(ctx, apiError)
	}

	// Send alert if enabled
	if eh.config.AlertOnErrors && eh.alerter != nil && eh.shouldAlert(apiError.Severity) {
		eh.sendAlert(ctx, apiError)
	}

	return apiError
}

// HandleValidationError handles validation errors
func (eh *ErrorHandler) HandleValidationError(ctx context.Context, field, message string, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("validation failed for field %s: %s", field, message), ErrorTypeValidation, context)
}

// HandleAuthenticationError handles authentication errors
func (eh *ErrorHandler) HandleAuthenticationError(ctx context.Context, message string, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("authentication failed: %s", message), ErrorTypeAuthentication, context)
}

// HandleAuthorizationError handles authorization errors
func (eh *ErrorHandler) HandleAuthorizationError(ctx context.Context, resource, action string, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("authorization denied for %s on %s", action, resource), ErrorTypeAuthorization, context)
}

// HandleNotFoundError handles not found errors
func (eh *ErrorHandler) HandleNotFoundError(ctx context.Context, resource string, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("resource not found: %s", resource), ErrorTypeNotFound, context)
}

// HandleRateLimitError handles rate limit errors
func (eh *ErrorHandler) HandleRateLimitError(ctx context.Context, limit int, window time.Duration, context *ErrorContext) *APIError {
	apiError := eh.HandleError(ctx, fmt.Errorf("rate limit exceeded: %d requests per %v", limit, window), ErrorTypeRateLimit, context)
	apiError.RetryAfter = &window
	return apiError
}

// HandleQuotaExceededError handles quota exceeded errors
func (eh *ErrorHandler) HandleQuotaExceededError(ctx context.Context, quotaType string, usage, limit int, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("quota exceeded for %s: %d/%d", quotaType, usage, limit), ErrorTypeQuotaExceeded, context)
}

// HandleSystemError handles system errors
func (eh *ErrorHandler) HandleSystemError(ctx context.Context, component string, err error, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("system error in %s: %w", component, err), ErrorTypeSystem, context)
}

// HandleNetworkError handles network errors
func (eh *ErrorHandler) HandleNetworkError(ctx context.Context, service string, err error, context *ErrorContext) *APIError {
	apiError := eh.HandleError(ctx, fmt.Errorf("network error connecting to %s: %w", service, err), ErrorTypeNetwork, context)
	apiError.Retryable = true
	return apiError
}

// HandleTimeoutError handles timeout errors
func (eh *ErrorHandler) HandleTimeoutError(ctx context.Context, operation string, timeout time.Duration, context *ErrorContext) *APIError {
	apiError := eh.HandleError(ctx, fmt.Errorf("timeout after %v for operation %s", timeout, operation), ErrorTypeTimeout, context)
	apiError.Retryable = true
	return apiError
}

// HandleSecurityError handles security errors
func (eh *ErrorHandler) HandleSecurityError(ctx context.Context, threat string, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("security threat detected: %s", threat), ErrorTypeSecurity, context)
}

// HandlePaymentError handles payment errors
func (eh *ErrorHandler) HandlePaymentError(ctx context.Context, reason string, context *ErrorContext) *APIError {
	return eh.HandleError(ctx, fmt.Errorf("payment failed: %s", reason), ErrorTypePayment, context)
}

// HandleStorageError handles storage errors
func (eh *ErrorHandler) HandleStorageError(ctx context.Context, operation string, err error, context *ErrorContext) *APIError {
	apiError := eh.HandleError(ctx, fmt.Errorf("storage error during %s: %w", operation, err), ErrorTypeStorage, context)
	apiError.Retryable = true
	return apiError
}

// HandleExternalServiceError handles external service errors
func (eh *ErrorHandler) HandleExternalServiceError(ctx context.Context, service string, err error, context *ErrorContext) *APIError {
	apiError := eh.HandleError(ctx, fmt.Errorf("external service error from %s: %w", service, err), ErrorTypeExternal, context)
	apiError.Retryable = true
	return apiError
}

// determineSeverity determines error severity based on type
func (eh *ErrorHandler) determineSeverity(errorType ErrorType) ErrorSeverity {
	severityMap := map[ErrorType]ErrorSeverity{
		ErrorTypeValidation:     ErrorSeverityLow,
		ErrorTypeAuthentication: ErrorSeverityMedium,
		ErrorTypeAuthorization:  ErrorSeverityMedium,
		ErrorTypeNotFound:       ErrorSeverityLow,
		ErrorTypeConflict:       ErrorSeverityMedium,
		ErrorTypeRateLimit:      ErrorSeverityMedium,
		ErrorTypeQuotaExceeded:  ErrorSeverityMedium,
		ErrorTypeSystem:         ErrorSeverityHigh,
		ErrorTypeNetwork:        ErrorSeverityMedium,
		ErrorTypeTimeout:        ErrorSeverityMedium,
		ErrorTypeSecurity:       ErrorSeverityCritical,
		ErrorTypePayment:        ErrorSeverityHigh,
		ErrorTypeStorage:        ErrorSeverityHigh,
		ErrorTypeExternal:       ErrorSeverityMedium,
	}

	if severity, exists := severityMap[errorType]; exists {
		return severity
	}

	return eh.config.DefaultSeverity
}

// getErrorCode returns the error code for the given type
func (eh *ErrorHandler) getErrorCode(errorType ErrorType) string {
	codeMap := map[ErrorType]string{
		ErrorTypeValidation:     "INVALID_INPUT",
		ErrorTypeAuthentication: "AUTHENTICATION_FAILED",
		ErrorTypeAuthorization:  "AUTHORIZATION_DENIED",
		ErrorTypeNotFound:       "RESOURCE_NOT_FOUND",
		ErrorTypeConflict:       "RESOURCE_CONFLICT",
		ErrorTypeRateLimit:      "RATE_LIMIT_EXCEEDED",
		ErrorTypeQuotaExceeded:  "QUOTA_EXCEEDED",
		ErrorTypeSystem:         "SYSTEM_ERROR",
		ErrorTypeNetwork:        "NETWORK_ERROR",
		ErrorTypeTimeout:        "TIMEOUT_ERROR",
		ErrorTypeSecurity:       "SECURITY_THREAT",
		ErrorTypePayment:        "PAYMENT_FAILED",
		ErrorTypeStorage:        "STORAGE_ERROR",
		ErrorTypeExternal:       "EXTERNAL_SERVICE_ERROR",
	}

	if code, exists := codeMap[errorType]; exists {
		return code
	}

	return "UNKNOWN_ERROR"
}

// getUserFriendlyMessage returns a user-friendly error message
func (eh *ErrorHandler) getUserFriendlyMessage(errorType ErrorType) string {
	code := eh.getErrorCode(errorType)
	if message, exists := eh.config.CustomErrorCodes[code]; exists {
		return message
	}

	// Default messages
	messageMap := map[ErrorType]string{
		ErrorTypeValidation:     "The provided input is invalid",
		ErrorTypeAuthentication: "Authentication failed",
		ErrorTypeAuthorization:  "You don't have permission to perform this action",
		ErrorTypeNotFound:       "The requested resource was not found",
		ErrorTypeConflict:       "The resource already exists",
		ErrorTypeRateLimit:      "Too many requests. Please try again later",
		ErrorTypeQuotaExceeded:  "You have exceeded your quota limit",
		ErrorTypeSystem:         "An internal system error occurred",
		ErrorTypeNetwork:        "A network error occurred",
		ErrorTypeTimeout:        "The request timed out",
		ErrorTypeSecurity:       "A security threat was detected",
		ErrorTypePayment:        "Payment processing failed",
		ErrorTypeStorage:        "Storage operation failed",
		ErrorTypeExternal:       "External service is unavailable",
	}

	if message, exists := messageMap[errorType]; exists {
		return message
	}

	return "An unexpected error occurred"
}

// isRetryable determines if an error type is retryable
func (eh *ErrorHandler) isRetryable(errorType ErrorType) bool {
	retryableTypes := map[ErrorType]bool{
		ErrorTypeNetwork:        true,
		ErrorTypeTimeout:        true,
		ErrorTypeStorage:        true,
		ErrorTypeExternal:       true,
		ErrorTypeSystem:         false, // Depends on specific error
		ErrorTypeValidation:     false,
		ErrorTypeAuthentication: false,
		ErrorTypeAuthorization:  false,
		ErrorTypeNotFound:       false,
		ErrorTypeConflict:       false,
		ErrorTypeRateLimit:      true,
		ErrorTypeQuotaExceeded:  false,
		ErrorTypeSecurity:       false,
		ErrorTypePayment:        false,
	}

	if retryable, exists := retryableTypes[errorType]; exists {
		return retryable
	}

	return eh.config.DefaultRetryable
}

// getSuggestions returns helpful suggestions for the error
func (eh *ErrorHandler) getSuggestions(errorType ErrorType) []string {
	suggestionMap := map[ErrorType][]string{
		ErrorTypeValidation: {
			"Check your input format",
			"Ensure all required fields are provided",
			"Verify field values are within allowed ranges",
		},
		ErrorTypeAuthentication: {
			"Check your credentials",
			"Ensure your account is active",
			"Try logging in again",
		},
		ErrorTypeAuthorization: {
			"Contact your administrator for access",
			"Check if you have the required permissions",
			"Verify your account status",
		},
		ErrorTypeNotFound: {
			"Check the resource ID",
			"Verify the resource exists",
			"Try refreshing the page",
		},
		ErrorTypeRateLimit: {
			"Wait before making another request",
			"Consider upgrading your plan",
			"Reduce request frequency",
		},
		ErrorTypeQuotaExceeded: {
			"Upgrade your plan for higher limits",
			"Wait for quota reset",
			"Optimize your usage",
		},
		ErrorTypeNetwork: {
			"Check your internet connection",
			"Try again in a few moments",
			"Contact support if the issue persists",
		},
		ErrorTypeTimeout: {
			"Try again with a simpler request",
			"Check your network connection",
			"Contact support if the issue persists",
		},
		ErrorTypeSecurity: {
			"Contact security team immediately",
			"Do not retry the same action",
			"Report suspicious activity",
		},
		ErrorTypePayment: {
			"Check your payment method",
			"Verify billing information",
			"Contact payment support",
		},
		ErrorTypeStorage: {
			"Try again in a few moments",
			"Check file size and format",
			"Contact support if the issue persists",
		},
		ErrorTypeExternal: {
			"Try again later",
			"Check service status",
			"Contact support if the issue persists",
		},
	}

	if suggestions, exists := suggestionMap[errorType]; exists {
		return suggestions
	}

	return []string{"Please try again or contact support"}
}

// logError logs the error
func (eh *ErrorHandler) logError(ctx context.Context, apiError *APIError) {
	if eh.logger == nil {
		return
	}

	fields := map[string]interface{}{
		"error_type":    apiError.Type,
		"error_code":    apiError.Code,
		"error_message": apiError.Message,
		"error_details": apiError.Details,
		"severity":      apiError.Severity,
		"retryable":     apiError.Retryable,
	}

	if apiError.Context != nil {
		if eh.config.LogUserContext {
			if apiError.Context.UserID != "" {
				fields["user_id"] = apiError.Context.UserID
			}
			if apiError.Context.RequestID != "" {
				fields["request_id"] = apiError.Context.RequestID
			}
			if apiError.Context.TraceID != "" {
				fields["trace_id"] = apiError.Context.TraceID
			}
		}

		if apiError.Context.Endpoint != "" {
			fields["endpoint"] = apiError.Context.Endpoint
		}
		if apiError.Context.Method != "" {
			fields["method"] = apiError.Context.Method
		}
		if apiError.Context.IPAddress != "" {
			fields["ip_address"] = apiError.Context.IPAddress
		}
	}

	// Log based on severity
	switch apiError.Severity {
	case ErrorSeverityCritical:
		eh.logger.Error(ctx, "Critical error occurred", fields)
	case ErrorSeverityHigh:
		eh.logger.Error(ctx, "High severity error occurred", fields)
	case ErrorSeverityMedium:
		eh.logger.Warn(ctx, "Medium severity error occurred", fields)
	case ErrorSeverityLow:
		eh.logger.Info(ctx, "Low severity error occurred", fields)
	}
}

// sendAlert sends an alert for the error
func (eh *ErrorHandler) sendAlert(ctx context.Context, apiError *APIError) {
	if eh.alerter == nil {
		return
	}

	severity := "medium"
	switch apiError.Severity {
	case ErrorSeverityCritical:
		severity = "critical"
	case ErrorSeverityHigh:
		severity = "high"
	case ErrorSeverityMedium:
		severity = "medium"
	case ErrorSeverityLow:
		severity = "low"
	}

	metadata := map[string]interface{}{
		"error_type": apiError.Type,
		"error_code": apiError.Code,
		"severity":   apiError.Severity,
		"retryable":  apiError.Retryable,
	}

	if apiError.Context != nil {
		metadata["endpoint"] = apiError.Context.Endpoint
		metadata["method"] = apiError.Context.Method
		metadata["user_id"] = apiError.Context.UserID
	}

	eh.alerter.SendSystemAlert(ctx, fmt.Sprintf("Error Alert: %s", apiError.Code), apiError.Message, "error_handler", AlertSeverity(severity), metadata)
}

// shouldAlert determines if an alert should be sent
func (eh *ErrorHandler) shouldAlert(severity ErrorSeverity) bool {
	if alert, exists := eh.config.AlertThresholds[severity]; exists {
		return alert
	}
	return false
}

// GetHTTPStatus returns the appropriate HTTP status code for the error type
func (eh *ErrorHandler) GetHTTPStatus(errorType ErrorType) int {
	statusMap := map[ErrorType]int{
		ErrorTypeValidation:     http.StatusBadRequest,
		ErrorTypeAuthentication: http.StatusUnauthorized,
		ErrorTypeAuthorization:  http.StatusForbidden,
		ErrorTypeNotFound:       http.StatusNotFound,
		ErrorTypeConflict:       http.StatusConflict,
		ErrorTypeRateLimit:      http.StatusTooManyRequests,
		ErrorTypeQuotaExceeded:  http.StatusTooManyRequests,
		ErrorTypeSystem:         http.StatusInternalServerError,
		ErrorTypeNetwork:        http.StatusBadGateway,
		ErrorTypeTimeout:        http.StatusRequestTimeout,
		ErrorTypeSecurity:       http.StatusForbidden,
		ErrorTypePayment:        http.StatusPaymentRequired,
		ErrorTypeStorage:        http.StatusServiceUnavailable,
		ErrorTypeExternal:       http.StatusBadGateway,
	}

	if status, exists := statusMap[errorType]; exists {
		return status
	}

	return http.StatusInternalServerError
}

// GetErrorStats returns error handling statistics
func (eh *ErrorHandler) GetErrorStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":             eh.config,
		"custom_error_codes": len(eh.config.CustomErrorCodes),
		"alert_thresholds":   eh.config.AlertThresholds,
		"log_errors":         eh.config.LogErrors,
		"alert_on_errors":    eh.config.AlertOnErrors,
	}
}
