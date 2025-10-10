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
		config.MaxRetries = 3
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

// Retry executes a function with retry logic
func (r *RetryService) Retry(ctx context.Context, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxRetries-1 {
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", r.config.MaxRetries, lastErr)
}

// RetryWithResult executes a function with retry logic and returns the result
func (r *RetryService) RetryWithResult(ctx context.Context, fn RetryWithResultFunc) (interface{}, error) {
	var result interface{}
	var lastErr error

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		res, err := fn(ctx)
		if err == nil {
			return res, nil
		}

		result = res
		lastErr = err

		// Check if error is retryable
		if !r.isRetryableError(err) {
			return result, fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxRetries-1 {
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result, fmt.Errorf("failed after %d attempts: %w", r.config.MaxRetries, lastErr)
}

// RetryWithCustomDelay executes a function with custom delay calculation
func (r *RetryService) RetryWithCustomDelay(ctx context.Context, fn RetryFunc, delayFunc func(attempt int) time.Duration) error {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxRetries-1 {
			break
		}

		// Use custom delay function
		delay := delayFunc(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", r.config.MaxRetries, lastErr)
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

// RetryWithStats executes a function with retry logic and returns statistics
func (r *RetryService) RetryWithStats(ctx context.Context, fn RetryFunc) (error, RetryStats) {
	stats := RetryStats{}
	var lastErr error
	var totalDelay time.Duration

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		stats.TotalAttempts++

		err := fn(ctx)
		if err == nil {
			stats.SuccessfulAttempts++
			stats.AverageDelay = time.Duration(totalDelay.Nanoseconds() / int64(attempt))
			stats.TotalDelay = totalDelay
			return nil, stats
		}

		lastErr = err
		stats.FailedAttempts++

		// Check if error is retryable
		if !r.isRetryableError(err) {
			stats.AverageDelay = time.Duration(totalDelay.Nanoseconds() / int64(attempt))
			stats.TotalDelay = totalDelay
			return fmt.Errorf("non-retryable error: %w", err), stats
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxRetries-1 {
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)
		totalDelay += delay

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			stats.AverageDelay = time.Duration(totalDelay.Nanoseconds() / int64(attempt))
			stats.TotalDelay = totalDelay
			return fmt.Errorf("context cancelled: %w", ctx.Err()), stats
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	stats.AverageDelay = time.Duration(totalDelay.Nanoseconds() / int64(stats.TotalAttempts))
	stats.TotalDelay = totalDelay
	return fmt.Errorf("failed after %d attempts: %w", r.config.MaxRetries, lastErr), stats
}

// CircuitBreakerRetry executes a function with circuit breaker pattern
func (r *RetryService) CircuitBreakerRetry(ctx context.Context, fn RetryFunc, failureThreshold int, timeout time.Duration) error {
	// Simple circuit breaker implementation
	// In production, you'd want a more sophisticated circuit breaker

	var consecutiveFailures int
	var lastFailureTime time.Time

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		// Check circuit breaker
		if consecutiveFailures >= failureThreshold {
			if time.Since(lastFailureTime) < timeout {
				return fmt.Errorf("circuit breaker is open")
			}
			// Reset circuit breaker
			consecutiveFailures = 0
		}

		err := fn(ctx)
		if err == nil {
			consecutiveFailures = 0
			return nil
		}

		consecutiveFailures++
		lastFailureTime = time.Now()

		// Check if error is retryable
		if !r.isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxRetries-1 {
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("failed after %d attempts", r.config.MaxRetries)
}

// Default retry configurations
var (
	// DefaultRetryConfig provides sensible defaults
	DefaultRetryConfig = RetryConfig{
		MaxRetries:  3,
		BaseDelay:   time.Second,
		MaxDelay:    5 * time.Minute,
		Multiplier:  2.0,
		Jitter:      true,
		BackoffType: BackoffTypeExponential,
	}

	// FastRetryConfig for quick retries
	FastRetryConfig = RetryConfig{
		MaxRetries:  2,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      false,
		BackoffType: BackoffTypeExponential,
	}

	// SlowRetryConfig for slow retries
	SlowRetryConfig = RetryConfig{
		MaxRetries:  5,
		BaseDelay:   5 * time.Second,
		MaxDelay:    10 * time.Minute,
		Multiplier:  2.0,
		Jitter:      true,
		BackoffType: BackoffTypeExponential,
	}

	// LinearRetryConfig for linear backoff
	LinearRetryConfig = RetryConfig{
		MaxRetries:  3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  1.0,
		Jitter:      true,
		BackoffType: BackoffTypeLinear,
	}
)
