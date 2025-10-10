package payment

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PaymentErrorHandler handles payment-related errors
type PaymentErrorHandler struct {
	alertService AlertService
	logger       Logger
	config       PaymentErrorConfig
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
	Type      string
	Severity  string
	Title     string
	Message   string
	Context   map[string]interface{}
	Timestamp time.Time
	Service   string
	UserID    *string
	PaymentID *string
}

// PaymentErrorConfig represents payment error handling configuration
type PaymentErrorConfig struct {
	MaxRetries      int
	BaseRetryDelay  time.Duration
	MaxRetryDelay   time.Duration
	AlertThresholds map[string]bool
	LogLevels       map[string]string
}

// PaymentErrorType represents different types of payment errors
type PaymentErrorType string

const (
	PaymentErrorTypeGatewayFailure    PaymentErrorType = "gateway_failure"
	PaymentErrorTypeNetworkTimeout    PaymentErrorType = "network_timeout"
	PaymentErrorTypeInvalidAmount     PaymentErrorType = "invalid_amount"
	PaymentErrorTypeDuplicatePayment  PaymentErrorType = "duplicate_payment"
	PaymentErrorTypeUserCancelled     PaymentErrorType = "user_cancelled"
	PaymentErrorTypeInsufficientFunds PaymentErrorType = "insufficient_funds"
	PaymentErrorTypeCardDeclined      PaymentErrorType = "card_declined"
	PaymentErrorTypeExpiredCard       PaymentErrorType = "expired_card"
	PaymentErrorTypeInvalidCard       PaymentErrorType = "invalid_card"
	PaymentErrorTypeFraudDetected     PaymentErrorType = "fraud_detected"
	PaymentErrorTypeSystemError       PaymentErrorType = "system_error"
)

// PaymentError represents a payment error with context
type PaymentError struct {
	Type        PaymentErrorType
	Code        string
	Message     string
	UserID      string
	PaymentID   string
	Amount      int64
	Currency    string
	Gateway     string
	Retryable   bool
	ShouldAlert bool
	Context     map[string]interface{}
	Timestamp   time.Time
}

func (e *PaymentError) Error() string {
	return e.Message
}

// NewPaymentErrorHandler creates a new payment error handler
func NewPaymentErrorHandler(alertService AlertService, logger Logger, config PaymentErrorConfig) *PaymentErrorHandler {
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
		config.AlertThresholds = map[string]bool{
			"gateway_failure":    true,
			"network_timeout":    true,
			"system_error":       true,
			"fraud_detected":     true,
			"invalid_amount":     false,
			"duplicate_payment":  false,
			"user_cancelled":     false,
			"insufficient_funds": false,
			"card_declined":      false,
			"expired_card":       false,
			"invalid_card":       false,
		}
	}
	if config.LogLevels == nil {
		config.LogLevels = map[string]string{
			"gateway_failure":    "error",
			"network_timeout":    "warn",
			"system_error":       "error",
			"fraud_detected":     "error",
			"invalid_amount":     "warn",
			"duplicate_payment":  "info",
			"user_cancelled":     "info",
			"insufficient_funds": "info",
			"card_declined":      "info",
			"expired_card":       "info",
			"invalid_card":       "warn",
		}
	}

	return &PaymentErrorHandler{
		alertService: alertService,
		logger:       logger,
		config:       config,
	}
}

// HandlePaymentError handles payment errors
func (h *PaymentErrorHandler) HandlePaymentError(ctx context.Context, err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	// Classify the error
	paymentErr := h.classifyPaymentError(err, context)

	// Log the error
	h.logPaymentError(ctx, paymentErr)

	// Send alert if needed
	if paymentErr.ShouldAlert {
		h.sendPaymentAlert(ctx, paymentErr)
	}

	return paymentErr
}

// HandlePaymentFailure handles payment failures
func (h *PaymentErrorHandler) HandlePaymentFailure(ctx context.Context, paymentID, userID string, err error, context map[string]interface{}) error {
	paymentErr := &PaymentError{
		Type:      PaymentErrorTypeGatewayFailure,
		Code:      "PAYMENT_FAILED",
		Message:   err.Error(),
		UserID:    userID,
		PaymentID: paymentID,
		Retryable: h.isRetryableError(err),
		Context:   context,
		Timestamp: time.Now(),
	}

	// Determine if we should alert
	paymentErr.ShouldAlert = h.config.AlertThresholds[string(paymentErr.Type)]

	// Log the error
	h.logPaymentError(ctx, paymentErr)

	// Send alert if needed
	if paymentErr.ShouldAlert {
		h.sendPaymentAlert(ctx, paymentErr)
	}

	// Update payment status in database
	if err := h.updatePaymentStatus(ctx, paymentID, "failed", err.Error()); err != nil {
		h.logger.Error(ctx, "Failed to update payment status", map[string]interface{}{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
	}

	return paymentErr
}

// HandlePaymentCancellation handles payment cancellations
func (h *PaymentErrorHandler) HandlePaymentCancellation(ctx context.Context, paymentID, userID string, reason string, context map[string]interface{}) error {
	paymentErr := &PaymentError{
		Type:      PaymentErrorTypeUserCancelled,
		Code:      "PAYMENT_CANCELLED",
		Message:   fmt.Sprintf("Payment cancelled: %s", reason),
		UserID:    userID,
		PaymentID: paymentID,
		Retryable: false,
		Context:   context,
		Timestamp: time.Now(),
	}

	// Determine if we should alert
	paymentErr.ShouldAlert = h.config.AlertThresholds[string(paymentErr.Type)]

	// Log the error
	h.logPaymentError(ctx, paymentErr)

	// Send alert if needed
	if paymentErr.ShouldAlert {
		h.sendPaymentAlert(ctx, paymentErr)
	}

	// Update payment status in database
	if err := h.updatePaymentStatus(ctx, paymentID, "cancelled", reason); err != nil {
		h.logger.Error(ctx, "Failed to update payment status", map[string]interface{}{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
	}

	return paymentErr
}

// HandlePaymentTimeout handles payment timeouts
func (h *PaymentErrorHandler) HandlePaymentTimeout(ctx context.Context, paymentID, userID string, timeout time.Duration, context map[string]interface{}) error {
	paymentErr := &PaymentError{
		Type:      PaymentErrorTypeNetworkTimeout,
		Code:      "PAYMENT_TIMEOUT",
		Message:   fmt.Sprintf("Payment timed out after %v", timeout),
		UserID:    userID,
		PaymentID: paymentID,
		Retryable: true,
		Context:   context,
		Timestamp: time.Now(),
	}

	// Determine if we should alert
	paymentErr.ShouldAlert = h.config.AlertThresholds[string(paymentErr.Type)]

	// Log the error
	h.logPaymentError(ctx, paymentErr)

	// Send alert if needed
	if paymentErr.ShouldAlert {
		h.sendPaymentAlert(ctx, paymentErr)
	}

	// Update payment status in database
	if err := h.updatePaymentStatus(ctx, paymentID, "timeout", paymentErr.Message); err != nil {
		h.logger.Error(ctx, "Failed to update payment status", map[string]interface{}{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
	}

	return paymentErr
}

// HandleFraudDetection handles fraud detection
func (h *PaymentErrorHandler) HandleFraudDetection(ctx context.Context, paymentID, userID string, reason string, context map[string]interface{}) error {
	paymentErr := &PaymentError{
		Type:      PaymentErrorTypeFraudDetected,
		Code:      "FRAUD_DETECTED",
		Message:   fmt.Sprintf("Fraud detected: %s", reason),
		UserID:    userID,
		PaymentID: paymentID,
		Retryable: false,
		Context:   context,
		Timestamp: time.Now(),
	}

	// Always alert for fraud
	paymentErr.ShouldAlert = true

	// Log the error
	h.logPaymentError(ctx, paymentErr)

	// Send alert
	h.sendPaymentAlert(ctx, paymentErr)

	// Update payment status in database
	if err := h.updatePaymentStatus(ctx, paymentID, "fraud_detected", paymentErr.Message); err != nil {
		h.logger.Error(ctx, "Failed to update payment status", map[string]interface{}{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
	}

	// Block user if necessary
	if err := h.blockUserForFraud(ctx, userID, reason); err != nil {
		h.logger.Error(ctx, "Failed to block user for fraud", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	return paymentErr
}

// classifyPaymentError classifies a payment error
func (h *PaymentErrorHandler) classifyPaymentError(err error, context map[string]interface{}) *PaymentError {
	errStr := err.Error()

	// Extract context information
	userID := ""
	paymentID := ""
	amount := int64(0)
	currency := ""
	gateway := ""

	if userIDVal, ok := context["user_id"].(string); ok {
		userID = userIDVal
	}
	if paymentIDVal, ok := context["payment_id"].(string); ok {
		paymentID = paymentIDVal
	}
	if amountVal, ok := context["amount"].(int64); ok {
		amount = amountVal
	}
	if currencyVal, ok := context["currency"].(string); ok {
		currency = currencyVal
	}
	if gatewayVal, ok := context["gateway"].(string); ok {
		gateway = gatewayVal
	}

	// Classify error type
	var errorType PaymentErrorType
	var code string
	var retryable bool

	switch {
	case contains(errStr, "timeout") || contains(errStr, "deadline exceeded"):
		errorType = PaymentErrorTypeNetworkTimeout
		code = "TIMEOUT"
		retryable = true
	case contains(errStr, "insufficient funds"):
		errorType = PaymentErrorTypeInsufficientFunds
		code = "INSUFFICIENT_FUNDS"
		retryable = false
	case contains(errStr, "card declined"):
		errorType = PaymentErrorTypeCardDeclined
		code = "CARD_DECLINED"
		retryable = false
	case contains(errStr, "expired card"):
		errorType = PaymentErrorTypeExpiredCard
		code = "EXPIRED_CARD"
		retryable = false
	case contains(errStr, "invalid card"):
		errorType = PaymentErrorTypeInvalidCard
		code = "INVALID_CARD"
		retryable = false
	case contains(errStr, "fraud"):
		errorType = PaymentErrorTypeFraudDetected
		code = "FRAUD_DETECTED"
		retryable = false
	case contains(errStr, "duplicate"):
		errorType = PaymentErrorTypeDuplicatePayment
		code = "DUPLICATE_PAYMENT"
		retryable = false
	case contains(errStr, "invalid amount"):
		errorType = PaymentErrorTypeInvalidAmount
		code = "INVALID_AMOUNT"
		retryable = false
	case contains(errStr, "cancelled") || contains(errStr, "canceled"):
		errorType = PaymentErrorTypeUserCancelled
		code = "USER_CANCELLED"
		retryable = false
	default:
		errorType = PaymentErrorTypeSystemError
		code = "SYSTEM_ERROR"
		retryable = h.isRetryableError(err)
	}

	return &PaymentError{
		Type:      errorType,
		Code:      code,
		Message:   err.Error(),
		UserID:    userID,
		PaymentID: paymentID,
		Amount:    amount,
		Currency:  currency,
		Gateway:   gateway,
		Retryable: retryable,
		Context:   context,
		Timestamp: time.Now(),
	}
}

// logPaymentError logs a payment error
func (h *PaymentErrorHandler) logPaymentError(ctx context.Context, paymentErr *PaymentError) {
	fields := map[string]interface{}{
		"error_type":    paymentErr.Type,
		"error_code":    paymentErr.Code,
		"error_message": paymentErr.Message,
		"user_id":       paymentErr.UserID,
		"payment_id":    paymentErr.PaymentID,
		"amount":        paymentErr.Amount,
		"currency":      paymentErr.Currency,
		"gateway":       paymentErr.Gateway,
		"retryable":     paymentErr.Retryable,
		"timestamp":     paymentErr.Timestamp,
	}

	// Add context fields
	for k, v := range paymentErr.Context {
		fields[k] = v
	}

	logLevel := h.config.LogLevels[string(paymentErr.Type)]
	switch logLevel {
	case "error":
		h.logger.Error(ctx, "Payment error occurred", fields)
	case "warn":
		h.logger.Warn(ctx, "Payment warning occurred", fields)
	case "info":
		h.logger.Info(ctx, "Payment info", fields)
	default:
		h.logger.Error(ctx, "Payment error occurred", fields)
	}
}

// sendPaymentAlert sends a payment alert
func (h *PaymentErrorHandler) sendPaymentAlert(ctx context.Context, paymentErr *PaymentError) {
	if h.alertService == nil {
		return
	}

	alert := Alert{
		Type:      string(paymentErr.Type),
		Severity:  h.getSeverityLevel(paymentErr.Type),
		Title:     fmt.Sprintf("Payment %s", paymentErr.Code),
		Message:   paymentErr.Message,
		Context:   paymentErr.Context,
		Timestamp: paymentErr.Timestamp,
		Service:   "payment-service",
		UserID:    &paymentErr.UserID,
		PaymentID: &paymentErr.PaymentID,
	}

	if err := h.alertService.SendAlert(ctx, alert); err != nil {
		h.logger.Error(ctx, "Failed to send payment alert", map[string]interface{}{
			"alert_error":   err.Error(),
			"payment_error": paymentErr.Error(),
		})
	}
}

// getSeverityLevel returns the severity level for a payment error type
func (h *PaymentErrorHandler) getSeverityLevel(errorType PaymentErrorType) string {
	switch errorType {
	case PaymentErrorTypeFraudDetected:
		return "critical"
	case PaymentErrorTypeGatewayFailure, PaymentErrorTypeSystemError:
		return "high"
	case PaymentErrorTypeNetworkTimeout, PaymentErrorTypeInvalidAmount, PaymentErrorTypeInvalidCard:
		return "medium"
	default:
		return "low"
	}
}

// isRetryableError checks if an error is retryable
func (h *PaymentErrorHandler) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retryable errors
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network unreachable",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"server error",
		"internal server error",
		"bad gateway",
		"gateway timeout",
		"request timeout",
		"context deadline exceeded",
	}

	for _, retryableError := range retryableErrors {
		if contains(errStr, retryableError) {
			return true
		}
	}

	return false
}

// updatePaymentStatus updates payment status in database
func (h *PaymentErrorHandler) updatePaymentStatus(ctx context.Context, paymentID, status, errorMessage string) error {
	// This would typically update the payment status in the database
	// For now, we'll just log it
	h.logger.Info(ctx, "Payment status updated", map[string]interface{}{
		"payment_id":    paymentID,
		"status":        status,
		"error_message": errorMessage,
	})
	return nil
}

// blockUserForFraud blocks a user for fraud
func (h *PaymentErrorHandler) blockUserForFraud(ctx context.Context, userID, reason string) error {
	// This would typically block the user in the database
	// For now, we'll just log it
	h.logger.Warn(ctx, "User blocked for fraud", map[string]interface{}{
		"user_id": userID,
		"reason":  reason,
	})
	return nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring performs case-insensitive substring search
func containsSubstring(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
