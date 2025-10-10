package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// HTTPErrorHandler handles HTTP errors consistently
type HTTPErrorHandler struct {
	logger Logger
}

// NewHTTPErrorHandler creates a new HTTP error handler
func NewHTTPErrorHandler(logger Logger) *HTTPErrorHandler {
	return &HTTPErrorHandler{
		logger: logger,
	}
}

// WriteError writes a standardized error response
func (h *HTTPErrorHandler) WriteError(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
	response := ErrorResponse{
		Error:     http.StatusText(statusCode),
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}

	// Note: Request ID extraction would need to be handled differently
	// as we can't access the gin context directly from http.ResponseWriter

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)

	// Log the error
	if h.logger != nil {
		h.logger.Error(context.Background(), "HTTP Error", map[string]interface{}{
			"status_code": statusCode,
			"code":        code,
			"message":     message,
			"details":     details,
		})
	}
}

// WriteValidationError writes a validation error response
func (h *HTTPErrorHandler) WriteValidationError(w http.ResponseWriter, field string, message string) {
	details := map[string]interface{}{
		"field": field,
	}
	h.WriteError(w, http.StatusBadRequest, "validation_error", message, details)
}

// WriteUnauthorizedError writes an unauthorized error response
func (h *HTTPErrorHandler) WriteUnauthorizedError(w http.ResponseWriter, message string) {
	h.WriteError(w, http.StatusUnauthorized, "unauthorized", message, nil)
}

// WriteForbiddenError writes a forbidden error response
func (h *HTTPErrorHandler) WriteForbiddenError(w http.ResponseWriter, message string) {
	h.WriteError(w, http.StatusForbidden, "forbidden", message, nil)
}

// WriteNotFoundError writes a not found error response
func (h *HTTPErrorHandler) WriteNotFoundError(w http.ResponseWriter, resource string) {
	details := map[string]interface{}{
		"resource": resource,
	}
	h.WriteError(w, http.StatusNotFound, "not_found", fmt.Sprintf("%s not found", resource), details)
}

// WriteInternalServerError writes an internal server error response
func (h *HTTPErrorHandler) WriteInternalServerError(w http.ResponseWriter, message string) {
	h.WriteError(w, http.StatusInternalServerError, "internal_error", message, nil)
}

// WriteRateLimitError writes a rate limit error response
func (h *HTTPErrorHandler) WriteRateLimitError(w http.ResponseWriter, retryAfter time.Duration) {
	details := map[string]interface{}{
		"retry_after": retryAfter.Seconds(),
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
	h.WriteError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests", details)
}

// WriteQuotaExceededError writes a quota exceeded error response
func (h *HTTPErrorHandler) WriteQuotaExceededError(w http.ResponseWriter, resource string, limit int) {
	details := map[string]interface{}{
		"resource": resource,
		"limit":    limit,
	}
	h.WriteError(w, http.StatusTooManyRequests, "quota_exceeded", fmt.Sprintf("%s quota exceeded", resource), details)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	return fmt.Sprintf("validation failed: %d errors", len(ve.Errors))
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field, message, value string) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// WriteValidationErrors writes validation errors response
func (h *HTTPErrorHandler) WriteValidationErrors(w http.ResponseWriter, errors ValidationErrors) {
	details := map[string]interface{}{
		"validation_errors": errors.Errors,
	}
	h.WriteError(w, http.StatusBadRequest, "validation_failed", "Validation failed", details)
}

// HTTPBusinessError represents a business logic error for HTTP responses
type HTTPBusinessError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (be HTTPBusinessError) Error() string {
	return be.Message
}

// NewHTTPBusinessError creates a new HTTP business error
func NewHTTPBusinessError(code, message string, details map[string]interface{}) HTTPBusinessError {
	return HTTPBusinessError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WriteBusinessError writes a business error response
func (h *HTTPErrorHandler) WriteBusinessError(w http.ResponseWriter, err HTTPBusinessError) {
	h.WriteError(w, http.StatusUnprocessableEntity, err.Code, err.Message, err.Details)
}

// HTTPSystemError represents a system error for HTTP responses
type HTTPSystemError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (se HTTPSystemError) Error() string {
	return se.Message
}

// NewHTTPSystemError creates a new HTTP system error
func NewHTTPSystemError(code, message string, details map[string]interface{}) HTTPSystemError {
	return HTTPSystemError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WriteSystemError writes a system error response
func (h *HTTPErrorHandler) WriteSystemError(w http.ResponseWriter, err HTTPSystemError) {
	h.WriteError(w, http.StatusInternalServerError, err.Code, err.Message, err.Details)
}

// GinErrorHandler wraps the HTTP error handler for Gin
func GinErrorHandler(errorHandler *HTTPErrorHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Check if it's a custom error type
			switch e := err.Err.(type) {
			case ValidationErrors:
				errorHandler.WriteValidationErrors(c.Writer, e)
			case HTTPBusinessError:
				errorHandler.WriteBusinessError(c.Writer, e)
			case HTTPSystemError:
				errorHandler.WriteSystemError(c.Writer, e)
			default:
				errorHandler.WriteInternalServerError(c.Writer, "An unexpected error occurred")
			}
		}
	}
}
