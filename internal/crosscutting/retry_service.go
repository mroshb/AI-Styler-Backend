package crosscutting

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig represents configuration for retry policies
type RetryConfig struct {
	// Basic retry settings
	MaxRetries int           `json:"max_retries"`
	BaseDelay  time.Duration `json:"base_delay"`
	MaxDelay   time.Duration `json:"max_delay"`
	Multiplier float64       `json:"multiplier"`
	Jitter     bool          `json:"jitter"`

	// Backoff strategy
	BackoffType BackoffType `json:"backoff_type"`

	// Service-specific configurations
	ServiceConfigs map[string]ServiceRetryConfig `json:"service_configs"`

	// Retryable error patterns
	RetryableErrors    []string `json:"retryable_errors"`
	NonRetryableErrors []string `json:"non_retryable_errors"`
}

// BackoffType represents different backoff strategies
type BackoffType string

const (
	BackoffTypeExponential BackoffType = "exponential"
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeFixed       BackoffType = "fixed"
	BackoffTypeCustom      BackoffType = "custom"
)

// ServiceRetryConfig represents retry configuration for specific services
type ServiceRetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	Multiplier      float64       `json:"multiplier"`
	BackoffType     BackoffType   `json:"backoff_type"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  1,
		BaseDelay:   time.Second,
		MaxDelay:    5 * time.Minute,
		Multiplier:  2.0,
		Jitter:      true,
		BackoffType: BackoffTypeExponential,
		RetryableErrors: []string{
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
			"connection timeout",
			"read timeout",
			"write timeout",
		},
		NonRetryableErrors: []string{
			"invalid input",
			"validation failed",
			"unauthorized",
			"forbidden",
			"not found",
			"bad request",
			"invalid format",
			"unsupported format",
			"file too large",
			"quota exceeded",
			"permission denied",
			"authentication failed",
			"invalid credentials",
			"malformed request",
			"invalid parameter",
			"duplicate",
			"already exists",
		},
		ServiceConfigs: map[string]ServiceRetryConfig{
			"gemini_api": {
				MaxRetries:  1,
				BaseDelay:   2 * time.Second,
				MaxDelay:    30 * time.Second,
				Multiplier:  2.0,
				BackoffType: BackoffTypeExponential,
				RetryableErrors: []string{
					"rate limit",
					"quota exceeded",
					"service unavailable",
					"timeout",
					"internal server error",
				},
			},
			"worker": {
				MaxRetries:  1,
				BaseDelay:   time.Second,
				MaxDelay:    10 * time.Second,
				Multiplier:  1.5,
				BackoffType: BackoffTypeLinear,
				RetryableErrors: []string{
					"timeout",
					"service unavailable",
					"internal server error",
				},
			},
			"storage": {
				MaxRetries:  1,
				BaseDelay:   500 * time.Millisecond,
				MaxDelay:    15 * time.Second,
				Multiplier:  2.0,
				BackoffType: BackoffTypeExponential,
				RetryableErrors: []string{
					"timeout",
					"connection refused",
					"service unavailable",
					"internal server error",
				},
			},
			"payment": {
				MaxRetries:  1,
				BaseDelay:   3 * time.Second,
				MaxDelay:    10 * time.Second,
				Multiplier:  2.0,
				BackoffType: BackoffTypeExponential,
				RetryableErrors: []string{
					"timeout",
					"service unavailable",
					"internal server error",
				},
			},
			"notification": {
				MaxRetries:  1,
				BaseDelay:   time.Second,
				MaxDelay:    5 * time.Second,
				Multiplier:  1.5,
				BackoffType: BackoffTypeLinear,
				RetryableErrors: []string{
					"timeout",
					"service unavailable",
					"internal server error",
				},
			},
		},
	}
}

// RetryService provides comprehensive retry functionality
type RetryService struct {
	config *RetryConfig
}

// NewRetryService creates a new retry service
func NewRetryService(config *RetryConfig) *RetryService {
	if config == nil {
		config = DefaultRetryConfig()
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
func (rs *RetryService) Retry(ctx context.Context, service string, fn RetryFunc) error {
	err := fn(ctx)
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	return nil
}

// RetryWithResult executes a function and returns the result (single attempt only)
func (rs *RetryService) RetryWithResult(ctx context.Context, service string, fn RetryWithResultFunc) (interface{}, error) {
	res, err := fn(ctx)
	if err != nil {
		return res, fmt.Errorf("operation failed: %w", err)
	}
	return res, nil
}

// RetryWithCustomDelay executes a function (single attempt only)
func (rs *RetryService) RetryWithCustomDelay(ctx context.Context, service string, fn RetryFunc, delayFunc func(attempt int) time.Duration) error {
	err := fn(ctx)
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	return nil
}

// getServiceConfig returns the retry configuration for a specific service
func (rs *RetryService) getServiceConfig(service string) ServiceRetryConfig {
	if config, exists := rs.config.ServiceConfigs[service]; exists {
		return config
	}

	// Return default configuration
	return ServiceRetryConfig{
		MaxRetries:      rs.config.MaxRetries,
		BaseDelay:       rs.config.BaseDelay,
		MaxDelay:        rs.config.MaxDelay,
		Multiplier:      rs.config.Multiplier,
		BackoffType:     rs.config.BackoffType,
		RetryableErrors: rs.config.RetryableErrors,
	}
}

// calculateDelay calculates the delay for the next retry attempt
func (rs *RetryService) calculateDelay(attempt int, config ServiceRetryConfig) time.Duration {
	var delay time.Duration

	switch config.BackoffType {
	case BackoffTypeExponential:
		delay = time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt)))
	case BackoffTypeLinear:
		delay = config.BaseDelay + time.Duration(attempt)*time.Second
	case BackoffTypeFixed:
		delay = config.BaseDelay
	case BackoffTypeCustom:
		// For custom backoff, we'll use exponential as default
		delay = time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt)))
	default:
		delay = time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt)))
	}

	// Apply jitter if enabled
	if rs.config.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		delay += jitter
	}

	// Cap at max delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	return delay
}

// isRetryableError determines if an error is retryable
func (rs *RetryService) isRetryableError(err error, retryableErrors []string) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// Check non-retryable errors first
	for _, nonRetryableError := range rs.config.NonRetryableErrors {
		if rs.containsError(errorStr, nonRetryableError) {
			return false
		}
	}

	// Check retryable errors
	for _, retryableError := range retryableErrors {
		if rs.containsError(errorStr, retryableError) {
			return true
		}
	}

	// Default to retryable for unknown errors
	return true
}

// containsError checks if a string contains a substring (case-insensitive)
func (rs *RetryService) containsError(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					rs.containsSubstring(s, substr)))
}

// containsSubstring performs case-insensitive substring search
func (rs *RetryService) containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// GetRetryStats returns retry statistics
func (rs *RetryService) GetRetryStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":          rs.config,
		"service_configs": rs.config.ServiceConfigs,
	}
}

// UpdateServiceConfig updates the retry configuration for a specific service
func (rs *RetryService) UpdateServiceConfig(service string, config ServiceRetryConfig) {
	rs.config.ServiceConfigs[service] = config
}

// AddRetryableError adds a retryable error pattern
func (rs *RetryService) AddRetryableError(pattern string) {
	rs.config.RetryableErrors = append(rs.config.RetryableErrors, pattern)
}

// AddNonRetryableError adds a non-retryable error pattern
func (rs *RetryService) AddNonRetryableError(pattern string) {
	rs.config.NonRetryableErrors = append(rs.config.NonRetryableErrors, pattern)
}
