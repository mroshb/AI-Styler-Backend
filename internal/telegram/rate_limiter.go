package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter provides rate limiting functionality using Redis sliding window
type RateLimiter struct {
	redis *redis.Client
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redisClient,
	}
}

// Allow checks if a request is allowed based on rate limit
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if rl.redis == nil {
		// If Redis is not available, allow all requests (graceful degradation)
		return true, nil
	}

	now := time.Now()
	windowStart := now.Add(-window)

	// Use sliding window log algorithm
	// Key format: rate_limit:{key}
	redisKey := fmt.Sprintf("rate_limit:%s", key)

	// Remove old entries outside the window
	pipe := rl.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.Unix()))
	pipe.ZCard(ctx, redisKey)
	pipe.ZAdd(ctx, redisKey, &redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	pipe.Expire(ctx, redisKey, window)

	results, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to execute rate limit check: %w", err)
	}

	// Get count after removing old entries
	count := results[1].(*redis.IntCmd).Val()

	// Check if limit exceeded (before adding new entry)
	if count >= int64(limit) {
		return false, nil
	}

	// New entry was added, check again
	newCount := results[2].(*redis.IntCmd).Val()
	return newCount <= int64(limit), nil
}

// AllowUserMessage checks if user can send a message
func (rl *RateLimiter) AllowUserMessage(ctx context.Context, telegramUserID int64, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("user_message:%d", telegramUserID)
	return rl.Allow(ctx, key, limit, window)
}

// AllowUserConversion checks if user can create a conversion
func (rl *RateLimiter) AllowUserConversion(ctx context.Context, telegramUserID int64, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("user_conversion:%d", telegramUserID)
	return rl.Allow(ctx, key, limit, window)
}

// AllowIP checks if IP can make a request
func (rl *RateLimiter) AllowIP(ctx context.Context, ip string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ip:%s", ip)
	return rl.Allow(ctx, key, limit, window)
}

// GetRemaining returns remaining requests for a key
func (rl *RateLimiter) GetRemaining(ctx context.Context, key string, limit int, window time.Duration) (int, error) {
	if rl.redis == nil {
		return limit, nil
	}

	now := time.Now()
	windowStart := now.Add(-window)

	redisKey := fmt.Sprintf("rate_limit:%s", key)

	// Remove old entries
	rl.redis.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.Unix()))

	// Get count
	count, err := rl.redis.ZCard(ctx, redisKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get remaining count: %w", err)
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

