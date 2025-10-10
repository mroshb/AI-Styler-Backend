package notification

import (
	"context"
	"fmt"
	"log"
	"time"
)

// QuotaMonitor monitors user quotas and sends notifications
type QuotaMonitor struct {
	notificationService NotificationService
	quotaService        QuotaService
	userService         UserService
	checkInterval       time.Duration
	warningThresholds   map[string]int // quotaType -> warning percentage
}

// NewQuotaMonitor creates a new quota monitor
func NewQuotaMonitor(
	notificationService NotificationService,
	quotaService QuotaService,
	userService UserService,
	checkInterval time.Duration,
) *QuotaMonitor {
	return &QuotaMonitor{
		notificationService: notificationService,
		quotaService:        quotaService,
		userService:         userService,
		checkInterval:       checkInterval,
		warningThresholds: map[string]int{
			"free": 80, // Warn when 80% of free quota is used
			"paid": 90, // Warn when 90% of paid quota is used
		},
	}
}

// Start starts the quota monitoring process
func (qm *QuotaMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(qm.checkInterval)
	defer ticker.Stop()

	log.Printf("Quota monitor started with check interval: %v", qm.checkInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Quota monitor stopped")
			return
		case <-ticker.C:
			if err := qm.checkQuotas(ctx); err != nil {
				log.Printf("Error checking quotas: %v", err)
			}
		}
	}
}

// checkQuotas checks all user quotas and sends notifications
func (qm *QuotaMonitor) checkQuotas(_ context.Context) error {
	// This is a simplified implementation
	// In production, you'd query the database for all users and check their quotas

	// Get all users (this would be implemented based on your user service)
	// For now, we'll just log that we're checking
	log.Println("Checking user quotas...")

	// Example quota check logic:
	// 1. Get all active users
	// 2. For each user, check their quota status
	// 3. Send warnings if approaching limits
	// 4. Send exhausted notifications if limits reached

	return nil
}

// CheckUserQuota checks a specific user's quota and sends notifications if needed
func (qm *QuotaMonitor) CheckUserQuota(ctx context.Context, userID string) error {
	// Get user quota status
	_, err := qm.quotaService.GetUserQuotaStatus(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user quota status: %w", err)
	}

	// This is a simplified implementation
	// In production, you'd parse the quota status and check against thresholds

	// Example logic:
	// if quotaStatus.RemainingFree <= 0 {
	//     return qm.notificationService.SendQuotaExhausted(ctx, userID, "free")
	// }
	//
	// if quotaStatus.RemainingFree <= quotaStatus.FreeLimit * qm.warningThresholds["free"] / 100 {
	//     return qm.notificationService.SendQuotaWarning(ctx, userID, "free", quotaStatus.RemainingFree)
	// }

	return nil
}

// SendQuotaExhaustedNotification sends a quota exhausted notification
func (qm *QuotaMonitor) SendQuotaExhaustedNotification(ctx context.Context, userID, quotaType string) error {
	return qm.notificationService.SendQuotaExhausted(ctx, userID, quotaType)
}

// SendQuotaWarningNotification sends a quota warning notification
func (qm *QuotaMonitor) SendQuotaWarningNotification(ctx context.Context, userID, quotaType string, remaining int) error {
	return qm.notificationService.SendQuotaWarning(ctx, userID, quotaType, remaining)
}

// SendQuotaResetNotification sends a quota reset notification
func (qm *QuotaMonitor) SendQuotaResetNotification(ctx context.Context, userID string) error {
	return qm.notificationService.SendQuotaReset(ctx, userID)
}

// SetWarningThreshold sets the warning threshold for a quota type
func (qm *QuotaMonitor) SetWarningThreshold(quotaType string, percentage int) {
	if percentage < 0 || percentage > 100 {
		log.Printf("Invalid warning threshold percentage: %d", percentage)
		return
	}
	qm.warningThresholds[quotaType] = percentage
}

// GetWarningThreshold gets the warning threshold for a quota type
func (qm *QuotaMonitor) GetWarningThreshold(quotaType string) int {
	if threshold, exists := qm.warningThresholds[quotaType]; exists {
		return threshold
	}
	return 80 // Default threshold
}
