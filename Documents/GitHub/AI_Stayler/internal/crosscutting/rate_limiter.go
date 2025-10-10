package crosscutting

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiterConfig represents configuration for rate limiting
type RateLimiterConfig struct {
	// Global limits
	GlobalPerIP   int           `json:"global_per_ip"`
	GlobalPerUser int           `json:"global_per_user"`
	GlobalWindow  time.Duration `json:"global_window"`

	// Endpoint-specific limits
	EndpointLimits map[string]EndpointLimit `json:"endpoint_limits"`

	// Plan-based limits
	PlanLimits map[string]PlanLimit `json:"plan_limits"`

	// Cleanup configuration
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxEntries      int           `json:"max_entries"`
}

// EndpointLimit represents rate limits for specific endpoints
type EndpointLimit struct {
	PerIP   int           `json:"per_ip"`
	PerUser int           `json:"per_user"`
	Window  time.Duration `json:"window"`
}

// PlanLimit represents rate limits for user plans
type PlanLimit struct {
	PerIP   int           `json:"per_ip"`
	PerUser int           `json:"per_user"`
	Window  time.Duration `json:"window"`
}

// DefaultRateLimiterConfig returns default configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		GlobalPerIP:     1000,
		GlobalPerUser:   5000,
		GlobalWindow:    time.Hour,
		CleanupInterval: 5 * time.Minute,
		MaxEntries:      100000,
		EndpointLimits: map[string]EndpointLimit{
			"/api/auth/send-otp": {
				PerIP:   5,
				PerUser: 10,
				Window:  time.Hour,
			},
			"/api/auth/verify-otp": {
				PerIP:   20,
				PerUser: 50,
				Window:  time.Hour,
			},
			"/api/conversion/create": {
				PerIP:   10,
				PerUser: 100,
				Window:  time.Hour,
			},
			"/api/image/upload": {
				PerIP:   20,
				PerUser: 200,
				Window:  time.Hour,
			},
			"/api/payment/create": {
				PerIP:   5,
				PerUser: 20,
				Window:  time.Hour,
			},
		},
		PlanLimits: map[string]PlanLimit{
			"free": {
				PerIP:   100,
				PerUser: 500,
				Window:  time.Hour,
			},
			"premium": {
				PerIP:   500,
				PerUser: 2000,
				Window:  time.Hour,
			},
			"enterprise": {
				PerIP:   2000,
				PerUser: 10000,
				Window:  time.Hour,
			},
		},
	}
}

// RateLimiter provides comprehensive rate limiting functionality
type RateLimiter struct {
	config *RateLimiterConfig

	// In-memory storage for rate limiting
	ipLimits   map[string]*LimitEntry
	userLimits map[string]*LimitEntry
	planLimits map[string]*LimitEntry

	// Mutex for thread safety
	mu sync.RWMutex

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// LimitEntry represents a rate limit entry
type LimitEntry struct {
	Count     int
	Window    time.Duration
	LastReset time.Time
	ExpiresAt time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &RateLimiter{
		config:      config,
		ipLimits:    make(map[string]*LimitEntry),
		userLimits:  make(map[string]*LimitEntry),
		planLimits:  make(map[string]*LimitEntry),
		stopCleanup: make(chan bool),
	}

	// Start cleanup routine
	rl.startCleanup()

	return rl
}

// Allow checks if a request is allowed based on IP and user limits
func (rl *RateLimiter) Allow(ctx context.Context, ip, userID, endpoint, plan string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Check global IP limit
	if !rl.checkLimit(rl.ipLimits, fmt.Sprintf("global:ip:%s", ip), rl.config.GlobalPerIP, rl.config.GlobalWindow, now) {
		return false, fmt.Errorf("global IP rate limit exceeded")
	}

	// Check global user limit (if user is authenticated)
	if userID != "" {
		if !rl.checkLimit(rl.userLimits, fmt.Sprintf("global:user:%s", userID), rl.config.GlobalPerUser, rl.config.GlobalWindow, now) {
			return false, fmt.Errorf("global user rate limit exceeded")
		}
	}

	// Check endpoint-specific limits
	if endpointLimit, exists := rl.config.EndpointLimits[endpoint]; exists {
		// Check endpoint IP limit
		if !rl.checkLimit(rl.ipLimits, fmt.Sprintf("endpoint:ip:%s:%s", endpoint, ip), endpointLimit.PerIP, endpointLimit.Window, now) {
			return false, fmt.Errorf("endpoint IP rate limit exceeded for %s", endpoint)
		}

		// Check endpoint user limit (if user is authenticated)
		if userID != "" {
			if !rl.checkLimit(rl.userLimits, fmt.Sprintf("endpoint:user:%s:%s", endpoint, userID), endpointLimit.PerUser, endpointLimit.Window, now) {
				return false, fmt.Errorf("endpoint user rate limit exceeded for %s", endpoint)
			}
		}
	}

	// Check plan-based limits (if user is authenticated and has a plan)
	if userID != "" && plan != "" {
		if planLimit, exists := rl.config.PlanLimits[plan]; exists {
			// Check plan IP limit
			if !rl.checkLimit(rl.ipLimits, fmt.Sprintf("plan:ip:%s:%s", plan, ip), planLimit.PerIP, planLimit.Window, now) {
				return false, fmt.Errorf("plan IP rate limit exceeded for %s", plan)
			}

			// Check plan user limit
			if !rl.checkLimit(rl.userLimits, fmt.Sprintf("plan:user:%s:%s", plan, userID), planLimit.PerUser, planLimit.Window, now) {
				return false, fmt.Errorf("plan user rate limit exceeded for %s", plan)
			}
		}
	}

	// Increment counters for all applicable limits
	rl.incrementLimit(rl.ipLimits, fmt.Sprintf("global:ip:%s", ip), rl.config.GlobalWindow, now)

	if userID != "" {
		rl.incrementLimit(rl.userLimits, fmt.Sprintf("global:user:%s", userID), rl.config.GlobalWindow, now)
	}

	if endpointLimit, exists := rl.config.EndpointLimits[endpoint]; exists {
		rl.incrementLimit(rl.ipLimits, fmt.Sprintf("endpoint:ip:%s:%s", endpoint, ip), endpointLimit.Window, now)

		if userID != "" {
			rl.incrementLimit(rl.userLimits, fmt.Sprintf("endpoint:user:%s:%s", endpoint, userID), endpointLimit.Window, now)
		}
	}

	if userID != "" && plan != "" {
		if planLimit, exists := rl.config.PlanLimits[plan]; exists {
			rl.incrementLimit(rl.ipLimits, fmt.Sprintf("plan:ip:%s:%s", plan, ip), planLimit.Window, now)
			rl.incrementLimit(rl.userLimits, fmt.Sprintf("plan:user:%s:%s", plan, userID), planLimit.Window, now)
		}
	}

	return true, nil
}

// checkLimit checks if a limit entry is within bounds
func (rl *RateLimiter) checkLimit(limits map[string]*LimitEntry, key string, maxCount int, window time.Duration, now time.Time) bool {
	entry, exists := limits[key]
	if !exists {
		return true // No previous entries, allow
	}

	// Check if window has expired
	if now.Sub(entry.LastReset) >= window {
		return true // Window expired, allow
	}

	// Check if count exceeds limit
	return entry.Count < maxCount
}

// incrementLimit increments the count for a limit entry
func (rl *RateLimiter) incrementLimit(limits map[string]*LimitEntry, key string, window time.Duration, now time.Time) {
	entry, exists := limits[key]
	if !exists {
		entry = &LimitEntry{
			Count:     0,
			Window:    window,
			LastReset: now,
			ExpiresAt: now.Add(window * 2), // Keep entry for 2x window for cleanup
		}
		limits[key] = entry
	}

	// Reset if window has expired
	if now.Sub(entry.LastReset) >= window {
		entry.Count = 0
		entry.LastReset = now
		entry.ExpiresAt = now.Add(window * 2)
	}

	entry.Count++
}

// GetStats returns current rate limiting statistics
func (rl *RateLimiter) GetStats(ctx context.Context) map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()

	stats := map[string]interface{}{
		"ip_entries":   len(rl.ipLimits),
		"user_entries": len(rl.userLimits),
		"plan_entries": len(rl.planLimits),
		"config":       rl.config,
	}

	// Count active entries (within their windows)
	activeIPEntries := 0
	activeUserEntries := 0

	for _, entry := range rl.ipLimits {
		if now.Sub(entry.LastReset) < entry.Window {
			activeIPEntries++
		}
	}

	for _, entry := range rl.userLimits {
		if now.Sub(entry.LastReset) < entry.Window {
			activeUserEntries++
		}
	}

	stats["active_ip_entries"] = activeIPEntries
	stats["active_user_entries"] = activeUserEntries

	return stats
}

// ResetLimit resets a specific limit entry
func (rl *RateLimiter) ResetLimit(ctx context.Context, key string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.ipLimits, key)
	delete(rl.userLimits, key)
	delete(rl.planLimits, key)

	return nil
}

// ResetAllLimits resets all limit entries
func (rl *RateLimiter) ResetAllLimits(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.ipLimits = make(map[string]*LimitEntry)
	rl.userLimits = make(map[string]*LimitEntry)
	rl.planLimits = make(map[string]*LimitEntry)

	return nil
}

// startCleanup starts the cleanup routine
func (rl *RateLimiter) startCleanup() {
	rl.cleanupTicker = time.NewTicker(rl.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-rl.cleanupTicker.C:
				rl.cleanup()
			case <-rl.stopCleanup:
				rl.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanup removes expired entries
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Cleanup IP limits
	for key, entry := range rl.ipLimits {
		if now.After(entry.ExpiresAt) {
			delete(rl.ipLimits, key)
		}
	}

	// Cleanup user limits
	for key, entry := range rl.userLimits {
		if now.After(entry.ExpiresAt) {
			delete(rl.userLimits, key)
		}
	}

	// Cleanup plan limits
	for key, entry := range rl.planLimits {
		if now.After(entry.ExpiresAt) {
			delete(rl.planLimits, key)
		}
	}

	// If we have too many entries, remove oldest ones
	if len(rl.ipLimits) > rl.config.MaxEntries {
		rl.trimEntries(rl.ipLimits)
	}
	if len(rl.userLimits) > rl.config.MaxEntries {
		rl.trimEntries(rl.userLimits)
	}
	if len(rl.planLimits) > rl.config.MaxEntries {
		rl.trimEntries(rl.planLimits)
	}
}

// trimEntries removes oldest entries to stay within max entries limit
func (rl *RateLimiter) trimEntries(limits map[string]*LimitEntry) {
	// Simple implementation: remove 10% of entries
	toRemove := len(limits) / 10
	if toRemove == 0 {
		toRemove = 1
	}

	count := 0
	for key := range limits {
		if count >= toRemove {
			break
		}
		delete(limits, key)
		count++
	}
}

// Stop stops the rate limiter and cleanup routine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.stopCleanup <- true
	}
}
