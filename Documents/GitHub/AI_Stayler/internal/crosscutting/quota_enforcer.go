package crosscutting

import (
	"context"
	"fmt"
	"time"
)

// QuotaConfig represents configuration for quota enforcement
type QuotaConfig struct {
	// Plan definitions
	Plans map[string]PlanQuota `json:"plans"`

	// Default quotas
	DefaultQuota PlanQuota `json:"default_quota"`

	// Enforcement settings
	EnforcementEnabled bool          `json:"enforcement_enabled"`
	GracePeriod        time.Duration `json:"grace_period"`

	// Reset settings
	ResetSchedule string `json:"reset_schedule"` // cron expression
	ResetTimeZone string `json:"reset_timezone"`
}

// PlanQuota represents quota limits for a specific plan
type PlanQuota struct {
	PlanName        string                 `json:"plan_name"`
	MonthlyLimits   map[string]int         `json:"monthly_limits"`
	DailyLimits     map[string]int         `json:"daily_limits"`
	HourlyLimits    map[string]int         `json:"hourly_limits"`
	ConcurrentLimit int                    `json:"concurrent_limit"`
	Features        map[string]bool        `json:"features"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// QuotaType represents different types of quotas
type QuotaType string

const (
	QuotaTypeConversions QuotaType = "conversions"
	QuotaTypeImages      QuotaType = "images"
	QuotaTypeStorage     QuotaType = "storage"
	QuotaTypeBandwidth   QuotaType = "bandwidth"
	QuotaTypeAPI         QuotaType = "api_calls"
)

// QuotaUsage represents current usage for a user
type QuotaUsage struct {
	UserID       string                 `json:"user_id"`
	PlanName     string                 `json:"plan_name"`
	MonthlyUsage map[string]int         `json:"monthly_usage"`
	DailyUsage   map[string]int         `json:"daily_usage"`
	HourlyUsage  map[string]int         `json:"hourly_usage"`
	Concurrent   int                    `json:"concurrent"`
	LastReset    time.Time              `json:"last_reset"`
	NextReset    time.Time              `json:"next_reset"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// QuotaCheckResult represents the result of a quota check
type QuotaCheckResult struct {
	Allowed           bool                   `json:"allowed"`
	Reason            string                 `json:"reason"`
	Remaining         map[string]int         `json:"remaining"`
	ResetTime         time.Time              `json:"reset_time"`
	UpgradeSuggestion string                 `json:"upgrade_suggestion"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// DefaultQuotaConfig returns default quota configuration
func DefaultQuotaConfig() *QuotaConfig {
	return &QuotaConfig{
		EnforcementEnabled: true,
		GracePeriod:        5 * time.Minute,
		ResetSchedule:      "0 0 1 * *", // First day of every month
		ResetTimeZone:      "UTC",
		DefaultQuota: PlanQuota{
			PlanName: "free",
			MonthlyLimits: map[string]int{
				string(QuotaTypeConversions): 10,
				string(QuotaTypeImages):      50,
				string(QuotaTypeStorage):     100, // MB
				string(QuotaTypeBandwidth):   500, // MB
				string(QuotaTypeAPI):         1000,
			},
			DailyLimits: map[string]int{
				string(QuotaTypeConversions): 2,
				string(QuotaTypeImages):      10,
				string(QuotaTypeStorage):     10, // MB
				string(QuotaTypeBandwidth):   50, // MB
				string(QuotaTypeAPI):         100,
			},
			HourlyLimits: map[string]int{
				string(QuotaTypeConversions): 1,
				string(QuotaTypeImages):      5,
				string(QuotaTypeStorage):     5,  // MB
				string(QuotaTypeBandwidth):   25, // MB
				string(QuotaTypeAPI):         20,
			},
			ConcurrentLimit: 2,
			Features: map[string]bool{
				"high_resolution":  false,
				"batch_processing": false,
				"priority_support": false,
				"api_access":       true,
			},
		},
		Plans: map[string]PlanQuota{
			"free": {
				PlanName: "free",
				MonthlyLimits: map[string]int{
					string(QuotaTypeConversions): 10,
					string(QuotaTypeImages):      50,
					string(QuotaTypeStorage):     100, // MB
					string(QuotaTypeBandwidth):   500, // MB
					string(QuotaTypeAPI):         1000,
				},
				DailyLimits: map[string]int{
					string(QuotaTypeConversions): 2,
					string(QuotaTypeImages):      10,
					string(QuotaTypeStorage):     10, // MB
					string(QuotaTypeBandwidth):   50, // MB
					string(QuotaTypeAPI):         100,
				},
				HourlyLimits: map[string]int{
					string(QuotaTypeConversions): 1,
					string(QuotaTypeImages):      5,
					string(QuotaTypeStorage):     5,  // MB
					string(QuotaTypeBandwidth):   25, // MB
					string(QuotaTypeAPI):         20,
				},
				ConcurrentLimit: 2,
				Features: map[string]bool{
					"high_resolution":  false,
					"batch_processing": false,
					"priority_support": false,
					"api_access":       true,
				},
			},
			"premium": {
				PlanName: "premium",
				MonthlyLimits: map[string]int{
					string(QuotaTypeConversions): 100,
					string(QuotaTypeImages):      500,
					string(QuotaTypeStorage):     1000, // MB
					string(QuotaTypeBandwidth):   5000, // MB
					string(QuotaTypeAPI):         10000,
				},
				DailyLimits: map[string]int{
					string(QuotaTypeConversions): 20,
					string(QuotaTypeImages):      100,
					string(QuotaTypeStorage):     100, // MB
					string(QuotaTypeBandwidth):   500, // MB
					string(QuotaTypeAPI):         1000,
				},
				HourlyLimits: map[string]int{
					string(QuotaTypeConversions): 5,
					string(QuotaTypeImages):      25,
					string(QuotaTypeStorage):     25,  // MB
					string(QuotaTypeBandwidth):   125, // MB
					string(QuotaTypeAPI):         100,
				},
				ConcurrentLimit: 5,
				Features: map[string]bool{
					"high_resolution":  true,
					"batch_processing": true,
					"priority_support": false,
					"api_access":       true,
				},
			},
			"enterprise": {
				PlanName: "enterprise",
				MonthlyLimits: map[string]int{
					string(QuotaTypeConversions): 1000,
					string(QuotaTypeImages):      5000,
					string(QuotaTypeStorage):     10000, // MB
					string(QuotaTypeBandwidth):   50000, // MB
					string(QuotaTypeAPI):         100000,
				},
				DailyLimits: map[string]int{
					string(QuotaTypeConversions): 200,
					string(QuotaTypeImages):      1000,
					string(QuotaTypeStorage):     1000, // MB
					string(QuotaTypeBandwidth):   5000, // MB
					string(QuotaTypeAPI):         10000,
				},
				HourlyLimits: map[string]int{
					string(QuotaTypeConversions): 50,
					string(QuotaTypeImages):      250,
					string(QuotaTypeStorage):     250,  // MB
					string(QuotaTypeBandwidth):   1250, // MB
					string(QuotaTypeAPI):         1000,
				},
				ConcurrentLimit: 20,
				Features: map[string]bool{
					"high_resolution":  true,
					"batch_processing": true,
					"priority_support": true,
					"api_access":       true,
				},
			},
		},
	}
}

// QuotaEnforcer provides quota enforcement functionality
type QuotaEnforcer struct {
	config *QuotaConfig
	store  QuotaStore
}

// QuotaStore interface for quota data storage
type QuotaStore interface {
	GetUserQuota(ctx context.Context, userID string) (*QuotaUsage, error)
	UpdateUserQuota(ctx context.Context, userID string, usage *QuotaUsage) error
	ResetUserQuota(ctx context.Context, userID string) error
	GetAllUserQuotas(ctx context.Context) ([]*QuotaUsage, error)
	IncrementUsage(ctx context.Context, userID string, quotaType QuotaType, amount int) error
	DecrementUsage(ctx context.Context, userID string, quotaType QuotaType, amount int) error
}

// NewQuotaEnforcer creates a new quota enforcer
func NewQuotaEnforcer(config *QuotaConfig, store QuotaStore) *QuotaEnforcer {
	if config == nil {
		config = DefaultQuotaConfig()
	}

	return &QuotaEnforcer{
		config: config,
		store:  store,
	}
}

// CheckQuota checks if a user can perform an action within their quota limits
func (qe *QuotaEnforcer) CheckQuota(ctx context.Context, userID string, quotaType QuotaType, amount int) (*QuotaCheckResult, error) {
	if !qe.config.EnforcementEnabled {
		return &QuotaCheckResult{
			Allowed: true,
			Reason:  "quota enforcement disabled",
		}, nil
	}

	// Get user quota usage
	usage, err := qe.store.GetUserQuota(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quota: %w", err)
	}

	// Get plan quota limits
	planQuota, exists := qe.config.Plans[usage.PlanName]
	if !exists {
		planQuota = qe.config.DefaultQuota
	}

	// Check monthly limits
	if monthlyLimit, exists := planQuota.MonthlyLimits[string(quotaType)]; exists {
		if usage.MonthlyUsage[string(quotaType)]+amount > monthlyLimit {
			return &QuotaCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("monthly quota exceeded for %s", quotaType),
				Remaining: map[string]int{
					string(quotaType): monthlyLimit - usage.MonthlyUsage[string(quotaType)],
				},
				ResetTime:         usage.NextReset,
				UpgradeSuggestion: qe.getUpgradeSuggestion(usage.PlanName, quotaType),
			}, nil
		}
	}

	// Check daily limits
	if dailyLimit, exists := planQuota.DailyLimits[string(quotaType)]; exists {
		if usage.DailyUsage[string(quotaType)]+amount > dailyLimit {
			return &QuotaCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("daily quota exceeded for %s", quotaType),
				Remaining: map[string]int{
					string(quotaType): dailyLimit - usage.DailyUsage[string(quotaType)],
				},
				ResetTime: qe.getNextDailyReset(),
			}, nil
		}
	}

	// Check hourly limits
	if hourlyLimit, exists := planQuota.HourlyLimits[string(quotaType)]; exists {
		if usage.HourlyUsage[string(quotaType)]+amount > hourlyLimit {
			return &QuotaCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("hourly quota exceeded for %s", quotaType),
				Remaining: map[string]int{
					string(quotaType): hourlyLimit - usage.HourlyUsage[string(quotaType)],
				},
				ResetTime: qe.getNextHourlyReset(),
			}, nil
		}
	}

	// Check concurrent limits
	if usage.Concurrent >= planQuota.ConcurrentLimit {
		return &QuotaCheckResult{
			Allowed: false,
			Reason:  "concurrent limit exceeded",
			Remaining: map[string]int{
				"concurrent": planQuota.ConcurrentLimit - usage.Concurrent,
			},
		}, nil
	}

	// Calculate remaining quotas
	remaining := make(map[string]int)
	if monthlyLimit, exists := planQuota.MonthlyLimits[string(quotaType)]; exists {
		remaining[string(quotaType)] = monthlyLimit - usage.MonthlyUsage[string(quotaType)]
	}

	return &QuotaCheckResult{
		Allowed:   true,
		Reason:    "quota check passed",
		Remaining: remaining,
		ResetTime: usage.NextReset,
	}, nil
}

// ConsumeQuota consumes quota for a user
func (qe *QuotaEnforcer) ConsumeQuota(ctx context.Context, userID string, quotaType QuotaType, amount int) error {
	if !qe.config.EnforcementEnabled {
		return nil
	}

	// Check quota first
	result, err := qe.CheckQuota(ctx, userID, quotaType, amount)
	if err != nil {
		return fmt.Errorf("failed to check quota: %w", err)
	}

	if !result.Allowed {
		return fmt.Errorf("quota check failed: %s", result.Reason)
	}

	// Increment usage
	err = qe.store.IncrementUsage(ctx, userID, quotaType, amount)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	return nil
}

// ReleaseQuota releases quota for a user (e.g., when a conversion fails)
func (qe *QuotaEnforcer) ReleaseQuota(ctx context.Context, userID string, quotaType QuotaType, amount int) error {
	if !qe.config.EnforcementEnabled {
		return nil
	}

	err := qe.store.DecrementUsage(ctx, userID, quotaType, amount)
	if err != nil {
		return fmt.Errorf("failed to decrement usage: %w", err)
	}

	return nil
}

// GetUserQuotaStatus returns the current quota status for a user
func (qe *QuotaEnforcer) GetUserQuotaStatus(ctx context.Context, userID string) (*QuotaUsage, error) {
	return qe.store.GetUserQuota(ctx, userID)
}

// ResetUserQuota resets quota for a user
func (qe *QuotaEnforcer) ResetUserQuota(ctx context.Context, userID string) error {
	return qe.store.ResetUserQuota(ctx, userID)
}

// UpdateUserPlan updates a user's plan
func (qe *QuotaEnforcer) UpdateUserPlan(ctx context.Context, userID string, planName string) error {
	usage, err := qe.store.GetUserQuota(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user quota: %w", err)
	}

	usage.PlanName = planName
	usage.LastReset = time.Now()
	usage.NextReset = qe.getNextMonthlyReset()

	// Reset usage counters
	usage.MonthlyUsage = make(map[string]int)
	usage.DailyUsage = make(map[string]int)
	usage.HourlyUsage = make(map[string]int)
	usage.Concurrent = 0

	return qe.store.UpdateUserQuota(ctx, userID, usage)
}

// CheckFeatureAccess checks if a user has access to a specific feature
func (qe *QuotaEnforcer) CheckFeatureAccess(ctx context.Context, userID string, feature string) (bool, error) {
	usage, err := qe.store.GetUserQuota(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user quota: %w", err)
	}

	planQuota, exists := qe.config.Plans[usage.PlanName]
	if !exists {
		planQuota = qe.config.DefaultQuota
	}

	hasAccess, exists := planQuota.Features[feature]
	return hasAccess && exists, nil
}

// getUpgradeSuggestion suggests a plan upgrade based on quota type
func (qe *QuotaEnforcer) getUpgradeSuggestion(currentPlan string, _ QuotaType) string {
	switch currentPlan {
	case "free":
		return "premium"
	case "premium":
		return "enterprise"
	default:
		return ""
	}
}

// getNextDailyReset returns the next daily reset time
func (qe *QuotaEnforcer) getNextDailyReset() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
}

// getNextHourlyReset returns the next hourly reset time
func (qe *QuotaEnforcer) getNextHourlyReset() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
}

// getNextMonthlyReset returns the next monthly reset time
func (qe *QuotaEnforcer) getNextMonthlyReset() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
}

// GetQuotaStats returns quota enforcement statistics
func (qe *QuotaEnforcer) GetQuotaStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":              qe.config,
		"enforcement_enabled": qe.config.EnforcementEnabled,
		"plans":               qe.config.Plans,
	}
}
