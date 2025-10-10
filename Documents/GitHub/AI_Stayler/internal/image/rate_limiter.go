package image

import (
	"context"
	"sync"
	"time"
)

// RateLimiterImpl implements the RateLimiter interface
type RateLimiterImpl struct {
	limits map[string]*limitEntry
	mutex  sync.RWMutex
}

type limitEntry struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiterImpl {
	return &RateLimiterImpl{
		limits: make(map[string]*limitEntry),
	}
}

// Allow checks if a request is allowed within the rate limit
func (r *RateLimiterImpl) Allow(ctx context.Context, key string, limit int, window int64) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	entry, exists := r.limits[key]

	if !exists || now.After(entry.resetTime) {
		// Create new entry or reset expired entry
		r.limits[key] = &limitEntry{
			count:     1,
			resetTime: now.Add(time.Duration(window) * time.Second),
		}
		return true
	}

	if entry.count >= limit {
		return false
	}

	entry.count++
	return true
}

// GetRemaining returns the number of remaining requests
func (r *RateLimiterImpl) GetRemaining(ctx context.Context, key string, limit int, window int64) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	now := time.Now()
	entry, exists := r.limits[key]

	if !exists || now.After(entry.resetTime) {
		return limit
	}

	remaining := limit - entry.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset resets the rate limit for a key
func (r *RateLimiterImpl) Reset(ctx context.Context, key string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.limits, key)
	return nil
}
