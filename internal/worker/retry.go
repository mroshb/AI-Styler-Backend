package worker

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryService implements the RetryHandler interface
type RetryService struct {
	config *RetryConfig
}

// NewRetryService creates a new retry service
func NewRetryService(config *RetryConfig) *RetryService {
	if config == nil {
		config = getDefaultRetryConfig()
	}

	return &RetryService{
		config: config,
	}
}

// ShouldRetry determines if a job should be retried based on the error
func (r *RetryService) ShouldRetry(ctx context.Context, job *WorkerJob, err error) bool {
	// No retries - always return false
	return false
}

// GetRetryDelay calculates the delay before the next retry attempt
func (r *RetryService) GetRetryDelay(ctx context.Context, job *WorkerJob) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(job.RetryCount))

	// Cap the delay at max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter if enabled
	if r.config.Jitter {
		// Add random jitter between 0.5 and 1.5 of the calculated delay
		jitterFactor := 0.5 + rand.Float64() // 0.5 to 1.5
		delay = delay * jitterFactor
	}

	return time.Duration(delay)
}

// IncrementRetryCount increments the retry count for a job
func (r *RetryService) IncrementRetryCount(ctx context.Context, job *WorkerJob) error {
	job.RetryCount++
	job.UpdatedAt = time.Now()
	return nil
}

// isRetryableError determines if an error is retryable
func (r *RetryService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// Network-related errors (retryable)
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
		"service unavailable",
		"gateway timeout",
		"request timeout",
		"context deadline exceeded",
	}

	// Check if error contains any retryable error patterns
	for _, retryableError := range retryableErrors {
		if containsRetry(errorStr, retryableError) {
			return true
		}
	}

	// Non-retryable errors
	nonRetryableErrors := []string{
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
	}

	// Check if error contains any non-retryable error patterns
	for _, nonRetryableError := range nonRetryableErrors {
		if containsRetry(errorStr, nonRetryableError) {
			return false
		}
	}

	// Default to retryable for unknown errors
	return true
}

// containsRetry checks if a string contains a substring (case-insensitive)
func containsRetry(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstringRetry(s, substr)))
}

// containsSubstringRetry performs case-insensitive substring search
func containsSubstringRetry(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	// Simple case-insensitive search
	sLower := toLowerCase(s)
	substrLower := toLowerCase(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLowerCase converts a string to lowercase
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// getDefaultRetryConfig returns default retry configuration
func getDefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    1,
		InitialDelay:  5 * time.Second,
		MaxDelay:      5 * time.Minute,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryPolicy defines different retry policies
type RetryPolicy string

const (
	RetryPolicyExponential RetryPolicy = "exponential"
	RetryPolicyLinear      RetryPolicy = "linear"
	RetryPolicyFixed       RetryPolicy = "fixed"
)

// AdvancedRetryService provides more sophisticated retry logic
type AdvancedRetryService struct {
	config *RetryConfig
	policy RetryPolicy
}

// NewAdvancedRetryService creates a new advanced retry service
func NewAdvancedRetryService(config *RetryConfig, policy RetryPolicy) *AdvancedRetryService {
	if config == nil {
		config = getDefaultRetryConfig()
	}

	return &AdvancedRetryService{
		config: config,
		policy: policy,
	}
}

// ShouldRetry determines if a job should be retried
func (r *AdvancedRetryService) ShouldRetry(ctx context.Context, job *WorkerJob, err error) bool {
	// No retries - always return false
	return false
}

// GetRetryDelay calculates the delay based on the retry policy
func (r *AdvancedRetryService) GetRetryDelay(ctx context.Context, job *WorkerJob) time.Duration {
	switch r.policy {
	case RetryPolicyLinear:
		return r.calculateLinearDelay(job)
	case RetryPolicyFixed:
		return r.calculateFixedDelay(job)
	case RetryPolicyExponential:
		fallthrough
	default:
		return r.calculateExponentialDelay(job)
	}
}

// IncrementRetryCount increments the retry count
func (r *AdvancedRetryService) IncrementRetryCount(ctx context.Context, job *WorkerJob) error {
	job.RetryCount++
	job.UpdatedAt = time.Now()
	return nil
}

// calculateExponentialDelay calculates exponential backoff delay
func (r *AdvancedRetryService) calculateExponentialDelay(job *WorkerJob) time.Duration {
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(job.RetryCount))

	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	if r.config.Jitter {
		jitterFactor := 0.5 + rand.Float64()
		delay = delay * jitterFactor
	}

	return time.Duration(delay)
}

// calculateLinearDelay calculates linear delay
func (r *AdvancedRetryService) calculateLinearDelay(job *WorkerJob) time.Duration {
	delay := r.config.InitialDelay + time.Duration(job.RetryCount)*r.config.InitialDelay

	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	if r.config.Jitter {
		jitterFactor := 0.5 + rand.Float64()
		delay = time.Duration(float64(delay) * jitterFactor)
	}

	return delay
}

// calculateFixedDelay calculates fixed delay
func (r *AdvancedRetryService) calculateFixedDelay(job *WorkerJob) time.Duration {
	delay := r.config.InitialDelay

	if r.config.Jitter {
		jitterFactor := 0.5 + rand.Float64()
		delay = time.Duration(float64(delay) * jitterFactor)
	}

	return delay
}

// RetryStats tracks retry statistics
type RetryStats struct {
	TotalRetries      int64 `json:"totalRetries"`
	SuccessfulRetries int64 `json:"successfulRetries"`
	FailedRetries     int64 `json:"failedRetries"`
	AverageDelay      int64 `json:"averageDelay"` // in milliseconds
}

// RetryTracker tracks retry statistics
type RetryTracker struct {
	stats map[string]*RetryStats
	mutex sync.RWMutex
}

// NewRetryTracker creates a new retry tracker
func NewRetryTracker() *RetryTracker {
	return &RetryTracker{
		stats: make(map[string]*RetryStats),
	}
}

// RecordRetry records a retry attempt
func (rt *RetryTracker) RecordRetry(jobType string, success bool, delay time.Duration) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	stats, exists := rt.stats[jobType]
	if !exists {
		stats = &RetryStats{}
		rt.stats[jobType] = stats
	}

	stats.TotalRetries++
	if success {
		stats.SuccessfulRetries++
	} else {
		stats.FailedRetries++
	}

	// Update average delay
	stats.AverageDelay = (stats.AverageDelay*int64(stats.TotalRetries-1) + delay.Milliseconds()) / int64(stats.TotalRetries)
}

// GetStats returns retry statistics for a job type
func (rt *RetryTracker) GetStats(jobType string) *RetryStats {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	stats, exists := rt.stats[jobType]
	if !exists {
		return &RetryStats{}
	}

	// Return a copy to avoid race conditions
	return &RetryStats{
		TotalRetries:      stats.TotalRetries,
		SuccessfulRetries: stats.SuccessfulRetries,
		FailedRetries:     stats.FailedRetries,
		AverageDelay:      stats.AverageDelay,
	}
}

// GetAllStats returns all retry statistics
func (rt *RetryTracker) GetAllStats() map[string]*RetryStats {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	result := make(map[string]*RetryStats)
	for jobType, stats := range rt.stats {
		result[jobType] = &RetryStats{
			TotalRetries:      stats.TotalRetries,
			SuccessfulRetries: stats.SuccessfulRetries,
			FailedRetries:     stats.FailedRetries,
			AverageDelay:      stats.AverageDelay,
		}
	}

	return result
}
