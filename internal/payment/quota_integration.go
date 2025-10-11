package payment

import (
	"context"
	"fmt"
)

// QuotaIntegration handles quota management integration
type QuotaIntegration struct {
	quotaService QuotaService
}

// NewQuotaIntegration creates a new quota integration
func NewQuotaIntegration(quotaService QuotaService) *QuotaIntegration {
	return &QuotaIntegration{
		quotaService: quotaService,
	}
}

// UpdateUserQuotaAfterPayment updates user quota after successful payment
func (q *QuotaIntegration) UpdateUserQuotaAfterPayment(ctx context.Context, userID string, planID string) error {
	// Get plan details to determine quota limits
	// This would typically involve getting the plan from the store
	// For now, we'll use a simple mapping
	_ = map[string]int{
		"free":     2,
		"basic":    20,
		"advanced": 100,
	}

	// Get plan name from plan ID (this is a simplified approach)
	// In a real implementation, you'd query the plan from the database
	planName := "basic" // Default fallback

	// Update user quota based on plan
	err := q.quotaService.UpdateUserQuota(ctx, userID, planName)
	if err != nil {
		return fmt.Errorf("failed to update user quota: %w", err)
	}

	return nil
}

// ResetMonthlyQuota resets user's monthly quota
func (q *QuotaIntegration) ResetMonthlyQuota(ctx context.Context, userID string) error {
	err := q.quotaService.ResetMonthlyQuota(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to reset monthly quota: %w", err)
	}

	return nil
}

// GetUserQuotaStatus gets current user quota status
func (q *QuotaIntegration) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	status, err := q.quotaService.GetUserQuotaStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quota status: %w", err)
	}

	return status, nil
}

// PlanQuotaMapping defines quota limits for each plan
var PlanQuotaMapping = map[string]int{
	PlanFree:     2,
	PlanBasic:    20,
	PlanAdvanced: 100,
}

// GetQuotaLimitForPlan returns the quota limit for a given plan
func GetQuotaLimitForPlan(planName string) int {
	if limit, exists := PlanQuotaMapping[planName]; exists {
		return limit
	}
	return 0 // Default to no quota
}

// IsPlanUpgrade checks if the new plan is an upgrade from the current plan
func IsPlanUpgrade(currentPlan, newPlan string) bool {
	planOrder := map[string]int{
		PlanFree:     0,
		PlanBasic:    1,
		PlanAdvanced: 2,
	}

	currentOrder, currentExists := planOrder[currentPlan]
	newOrder, newExists := planOrder[newPlan]

	if !currentExists || !newExists {
		return false
	}

	return newOrder > currentOrder
}

// GetPlanUpgradePrice calculates the upgrade price between plans
func GetPlanUpgradePrice(currentPlan, newPlan string) int64 {
	planPrices := map[string]int64{
		PlanFree:     0,
		PlanBasic:    50000,
		PlanAdvanced: 150000,
	}

	currentPrice, currentExists := planPrices[currentPlan]
	newPrice, newExists := planPrices[newPlan]

	if !currentExists || !newExists {
		return 0
	}

	// Return the difference in price
	upgradePrice := newPrice - currentPrice
	if upgradePrice < 0 {
		return 0 // No refund for downgrades
	}

	return upgradePrice
}
