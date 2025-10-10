package security

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisRateLimiter implements rate limiting using Redis with sliding window
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
	}
}

// Allow checks if a request is allowed based on rate limiting rules
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	now := time.Now()
	windowStart := now.Add(-window)

	// Use Redis pipeline for atomic operations
	pipe := rl.client.Pipeline()

	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests
	pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiration
	pipe.Expire(ctx, key, window)

	// Execute pipeline
	results, err := pipe.Exec(ctx)
	if err != nil {
		// If Redis is down, allow the request (fail open)
		return true
	}

	// Get the count before adding current request
	count := results[1].(*redis.IntCmd).Val()

	return int(count) < limit
}

// GetRemaining returns the number of remaining requests allowed
func (rl *RedisRateLimiter) GetRemaining(ctx context.Context, key string, limit int, window time.Duration) int {
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove expired entries
	rl.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests
	count, err := rl.client.ZCard(ctx, key).Result()
	if err != nil {
		return limit // If Redis is down, return full limit
	}

	remaining := limit - int(count)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset clears all requests for a key
func (rl *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	return rl.client.Del(ctx, key).Err()
}

// GetWindowStart returns the start of the current window
func (rl *RedisRateLimiter) GetWindowStart(ctx context.Context, key string, window time.Duration) time.Time {
	now := time.Now()
	windowStart := now.Add(-window)

	// Get the oldest entry in the window
	result, err := rl.client.ZRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil || len(result) == 0 {
		return windowStart
	}

	oldestTime := time.Unix(0, int64(result[0].Score))
	if oldestTime.After(windowStart) {
		return oldestTime
	}

	return windowStart
}

// RateLimitInfo contains information about current rate limit status
type RateLimitInfo struct {
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetAt   time.Time     `json:"reset_at"`
	Window    time.Duration `json:"window"`
}

// GetRateLimitInfo returns detailed rate limit information
func (rl *RedisRateLimiter) GetRateLimitInfo(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitInfo, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	// Clean up expired entries
	rl.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests
	count, err := rl.client.ZCard(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (when the oldest request expires)
	resetAt := now.Add(window)
	if count > 0 {
		// Get the oldest entry
		result, err := rl.client.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err == nil && len(result) > 0 {
			oldestTime := time.Unix(0, int64(result[0].Score))
			resetAt = oldestTime.Add(window)
		}
	}

	return &RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   resetAt,
		Window:    window,
	}, nil
}

// CircuitBreaker implements circuit breaker pattern for external services
type CircuitBreaker struct {
	name             string
	failureCount     int
	lastFailure      time.Time
	state            CircuitState
	failureThreshold int
	resetTimeout     time.Duration
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return fmt.Errorf("circuit breaker %s is open", cb.name)
		}
	}

	err := fn()

	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		}

		return err
	}

	// Success - reset failure count
	cb.failureCount = 0
	cb.state = StateClosed

	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	return cb.failureCount
}
