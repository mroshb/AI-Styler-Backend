package common

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
	BackoffType     BackoffType
	RetryableErrors []string
}

// BackoffType represents the type of backoff strategy
type BackoffType string

const (
	BackoffTypeExponential BackoffType = "exponential"
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeFixed       BackoffType = "fixed"
	BackoffTypeCustom      BackoffType = "custom"
)

// RetryService provides retry functionality with various backoff strategies
type RetryService struct {
	config RetryConfig
}

// NewRetryService creates a new retry service
func NewRetryService(config RetryConfig) *RetryService {
	if config.MaxRetries == 0 {
		config.MaxRetries = 1
	}
	if config.BaseDelay == 0 {
		config.BaseDelay = time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 5 * time.Minute
	}
	if config.Multiplier == 0 {
		config.Multiplier = 2.0
	}
	if config.BackoffType == "" {
		config.BackoffType = BackoffTypeExponential
	}
	if config.RetryableErrors == nil {
		config.RetryableErrors = []string{
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
	}

	return &RetryService{
		config: config,
	}
}

// RetryFunc represents a function that can be retried
type RetryFunc func(ctx context.Context) error

// RetryWithResultFunc represents a function that returns a result and can be retried
type RetryWithResultFunc func(ctx context.Context) (interface{}, error)

// Retry executes a function (single attempt only)
func (r *RetryService) Retry(ctx context.Context, fn RetryFunc) error {
	err := fn(ctx)
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	return nil
}

// RetryWithResult executes a function and returns the result (single attempt only)
func (r *RetryService) RetryWithResult(ctx context.Context, fn RetryWithResultFunc) (interface{}, error) {
	res, err := fn(ctx)
	if err != nil {
		return res, fmt.Errorf("operation failed: %w", err)
	}
	return res, nil
}

// RetryWithCustomDelay executes a function (single attempt only)
func (r *RetryService) RetryWithCustomDelay(ctx context.Context, fn RetryFunc, delayFunc func(attempt int) time.Duration) error {
	err := fn(ctx)
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	return nil
}

// RetryWithExponentialBackoff executes a function with exponential backoff
func (r *RetryService) RetryWithExponentialBackoff(ctx context.Context, fn RetryFunc) error {
	config := r.config
	config.BackoffType = BackoffTypeExponential
	config.Multiplier = 2.0

	retryService := &RetryService{config: config}
	return retryService.Retry(ctx, fn)
}

// RetryWithLinearBackoff executes a function with linear backoff
func (r *RetryService) RetryWithLinearBackoff(ctx context.Context, fn RetryFunc) error {
	config := r.config
	config.BackoffType = BackoffTypeLinear
	config.Multiplier = 1.0

	retryService := &RetryService{config: config}
	return retryService.Retry(ctx, fn)
}

// RetryWithFixedDelay executes a function with fixed delay
func (r *RetryService) RetryWithFixedDelay(ctx context.Context, fn RetryFunc) error {
	config := r.config
	config.BackoffType = BackoffTypeFixed
	config.Multiplier = 1.0

	retryService := &RetryService{config: config}
	return retryService.Retry(ctx, fn)
}

// RetryWithJitter executes a function with jitter to avoid thundering herd
func (r *RetryService) RetryWithJitter(ctx context.Context, fn RetryFunc) error {
	config := r.config
	config.Jitter = true

	retryService := &RetryService{config: config}
	return retryService.Retry(ctx, fn)
}

// calculateDelay calculates the delay for the given attempt
func (r *RetryService) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.BackoffType {
	case BackoffTypeExponential:
		delay = time.Duration(float64(r.config.BaseDelay) * math.Pow(r.config.Multiplier, float64(attempt)))
	case BackoffTypeLinear:
		delay = r.config.BaseDelay * time.Duration(attempt+1)
	case BackoffTypeFixed:
		delay = r.config.BaseDelay
	case BackoffTypeCustom:
		// For custom backoff, use exponential as default
		delay = time.Duration(float64(r.config.BaseDelay) * math.Pow(r.config.Multiplier, float64(attempt)))
	default:
		delay = time.Duration(float64(r.config.BaseDelay) * math.Pow(r.config.Multiplier, float64(attempt)))
	}

	// Cap at max delay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	// Add jitter if enabled
	if r.config.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		delay += jitter
	}

	return delay
}

// isRetryableError checks if an error is retryable
func (r *RetryService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check against retryable error patterns
	for _, retryableError := range r.config.RetryableErrors {
		if contains(errStr, retryableError) {
			return true
		}
	}

	return false
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

// RetryStats represents retry statistics
type RetryStats struct {
	TotalAttempts      int
	SuccessfulAttempts int
	FailedAttempts     int
	AverageDelay       time.Duration
	TotalDelay         time.Duration
}

// RetryWithStats executes a function and returns statistics (single attempt only)
func (r *RetryService) RetryWithStats(ctx context.Context, fn RetryFunc) (error, RetryStats) {
	stats := RetryStats{}
	stats.TotalAttempts = 1

	err := fn(ctx)
	if err == nil {
		stats.SuccessfulAttempts = 1
		stats.AverageDelay = 0
		stats.TotalDelay = 0
		return nil, stats
	}

	stats.FailedAttempts = 1
	stats.AverageDelay = 0
	stats.TotalDelay = 0
	return fmt.Errorf("operation failed: %w", err), stats
}

// CircuitBreakerRetry executes a function (single attempt only)
func (r *RetryService) CircuitBreakerRetry(ctx context.Context, fn RetryFunc, failureThreshold int, timeout time.Duration) error {
	err := fn(ctx)
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	return nil
}

// Default retry configurations
var (
	// DefaultRetryConfig provides sensible defaults
	DefaultRetryConfig = RetryConfig{
		MaxRetries:  1,
		BaseDelay:   time.Second,
		MaxDelay:    5 * time.Minute,
		Multiplier:  2.0,
		Jitter:      true,
		BackoffType: BackoffTypeExponential,
	}

	// FastRetryConfig for quick retries
	FastRetryConfig = RetryConfig{
		MaxRetries:  1,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      false,
		BackoffType: BackoffTypeExponential,
	}

	// SlowRetryConfig for slow retries
	SlowRetryConfig = RetryConfig{
		MaxRetries:  1,
		BaseDelay:   5 * time.Second,
		MaxDelay:    10 * time.Minute,
		Multiplier:  2.0,
		Jitter:      true,
		BackoffType: BackoffTypeExponential,
	}

	// LinearRetryConfig for linear backoff
	LinearRetryConfig = RetryConfig{
		MaxRetries:  1,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  1.0,
		Jitter:      true,
		BackoffType: BackoffTypeLinear,
	}
)
