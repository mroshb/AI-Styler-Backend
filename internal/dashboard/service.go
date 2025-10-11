package dashboard

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Service implements the dashboard business logic
type Service struct {
	store             Store
	userService       UserService
	conversionService ConversionService
	vendorService     VendorService
	paymentService    PaymentService
	cache             Cache
	metricsCollector  MetricsCollector
	auditLogger       AuditLogger
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(
	store Store,
	userService UserService,
	conversionService ConversionService,
	vendorService VendorService,
	paymentService PaymentService,
	cache Cache,
	metricsCollector MetricsCollector,
	auditLogger AuditLogger,
) *Service {
	return &Service{
		store:             store,
		userService:       userService,
		conversionService: conversionService,
		vendorService:     vendorService,
		paymentService:    paymentService,
		cache:             cache,
		metricsCollector:  metricsCollector,
		auditLogger:       auditLogger,
	}
}

// GetDashboardData retrieves comprehensive dashboard data for a user
func (s *Service) GetDashboardData(ctx context.Context, userID string, req DashboardRequest) (DashboardData, error) {
	// Log dashboard access
	if err := s.auditLogger.LogDashboardAccess(ctx, userID, "dashboard_view"); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to log dashboard access: %v\n", err)
	}

	// Record metrics
	if err := s.metricsCollector.RecordDashboardView(ctx, userID); err != nil {
		fmt.Printf("Failed to record dashboard metrics: %v\n", err)
	}

	// Check cache first
	if cachedData, err := s.store.GetCachedDashboardData(ctx, userID); err == nil && cachedData != nil {
		return *cachedData, nil
	}

	// Set default limits
	if req.HistoryLimit <= 0 {
		req.HistoryLimit = DefaultHistoryLimit
	}
	if req.GalleryLimit <= 0 {
		req.GalleryLimit = DefaultGalleryLimit
	}
	if req.HistoryLimit > MaxHistoryLimit {
		req.HistoryLimit = MaxHistoryLimit
	}
	if req.GalleryLimit > MaxGalleryLimit {
		req.GalleryLimit = MaxGalleryLimit
	}

	// Set default includes
	if !req.IncludeHistory {
		req.IncludeHistory = true
	}
	if !req.IncludeGallery {
		req.IncludeGallery = true
	}
	if !req.IncludeStatistics {
		req.IncludeStatistics = true
	}

	// Get user information
	userInfo, err := s.getUserInfo(ctx, userID)
	if err != nil {
		return DashboardData{}, fmt.Errorf("failed to get user info: %w", err)
	}

	// Get quota status
	quotaStatus, err := s.store.GetQuotaStatus(ctx, userID)
	if err != nil {
		return DashboardData{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	// Get conversion history
	var conversionHistory ConversionHistory
	if req.IncludeHistory {
		conversionHistory, err = s.store.GetConversionHistory(ctx, userID, req.HistoryLimit)
		if err != nil {
			return DashboardData{}, fmt.Errorf("failed to get conversion history: %w", err)
		}
	}

	// Get vendor gallery
	var vendorGallery VendorGallery
	if req.IncludeGallery {
		vendorGallery, err = s.store.GetVendorGallery(ctx, req.GalleryLimit)
		if err != nil {
			return DashboardData{}, fmt.Errorf("failed to get vendor gallery: %w", err)
		}
	}

	// Get plan status
	planStatus, err := s.store.GetPlanStatus(ctx, userID)
	if err != nil {
		return DashboardData{}, fmt.Errorf("failed to get plan status: %w", err)
	}

	// Get upgrade prompt
	upgradePrompt, err := s.store.ShouldShowUpgradePrompt(ctx, userID)
	if err != nil {
		fmt.Printf("Failed to get upgrade prompt: %v\n", err)
	}

	// Get statistics
	var statistics DashboardStatistics
	if req.IncludeStatistics {
		statistics, err = s.store.GetDashboardStatistics(ctx, userID)
		if err != nil {
			return DashboardData{}, fmt.Errorf("failed to get statistics: %w", err)
		}
	}

	// Get recent activity
	recentActivity, err := s.store.GetRecentActivity(ctx, userID, 10)
	if err != nil {
		fmt.Printf("Failed to get recent activity: %v\n", err)
		recentActivity = []RecentActivity{}
	}

	// Build dashboard data
	dashboardData := DashboardData{
		User:              userInfo,
		QuotaStatus:       quotaStatus,
		ConversionHistory: conversionHistory,
		VendorGallery:     vendorGallery,
		PlanStatus:        planStatus,
		UpgradePrompt:     upgradePrompt,
		Statistics:        statistics,
		RecentActivity:    recentActivity,
	}

	// Cache the result for 5 minutes
	if err := s.store.SetCachedDashboardData(ctx, userID, dashboardData, 300); err != nil {
		fmt.Printf("Failed to cache dashboard data: %v\n", err)
	}

	return dashboardData, nil
}

// CheckQuota checks if user can perform conversions and provides upgrade recommendations
func (s *Service) CheckQuota(ctx context.Context, userID string) (QuotaCheckResponse, error) {
	// Get quota status
	quotaStatus, err := s.store.GetQuotaStatus(ctx, userID)
	if err != nil {
		return QuotaCheckResponse{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	// Determine if user can convert
	canConvert := quotaStatus.TotalConversionsRemaining > 0

	// Get upgrade prompt if needed
	var upgradePrompt *UpgradePrompt
	if !canConvert || quotaStatus.TotalConversionsRemaining <= 2 {
		upgradePrompt, err = s.store.ShouldShowUpgradePrompt(ctx, userID)
		if err != nil {
			fmt.Printf("Failed to get upgrade prompt: %v\n", err)
		}
	}

	// Determine recommended action
	recommendedAction := s.getRecommendedAction(quotaStatus, upgradePrompt)

	// Record metrics
	if err := s.metricsCollector.RecordQuotaCheck(ctx, userID, canConvert); err != nil {
		fmt.Printf("Failed to record quota check metrics: %v\n", err)
	}

	// Log quota check
	if err := s.auditLogger.LogQuotaCheck(ctx, userID, QuotaCheckResponse{
		CanConvert:        canConvert,
		QuotaStatus:       quotaStatus,
		UpgradePrompt:     upgradePrompt,
		RecommendedAction: recommendedAction,
	}); err != nil {
		fmt.Printf("Failed to log quota check: %v\n", err)
	}

	return QuotaCheckResponse{
		CanConvert:        canConvert,
		QuotaStatus:       quotaStatus,
		UpgradePrompt:     upgradePrompt,
		RecommendedAction: recommendedAction,
	}, nil
}

// getUserInfo retrieves user information
func (s *Service) getUserInfo(ctx context.Context, userID string) (UserInfo, error) {
	_, err := s.userService.GetProfile(ctx, userID)
	if err != nil {
		return UserInfo{}, err
	}

	// Convert to dashboard UserInfo format
	// This would need to be adapted based on the actual user service response format
	userInfo := UserInfo{
		ID:              userID,
		Name:            stringPtr("User Name"), // This would come from userProfile
		AvatarURL:       stringPtr(""),          // This would come from userProfile
		Phone:           "1234567890",           // This would come from userProfile
		IsPhoneVerified: true,                   // This would come from userProfile
		Role:            "user",                 // This would come from userProfile
		CreatedAt:       time.Now(),             // This would come from userProfile
		LastLoginAt:     timePtr(time.Now()),    // This would come from userProfile
	}

	return userInfo, nil
}

// getRecommendedAction determines the recommended action based on quota status
func (s *Service) getRecommendedAction(quotaStatus QuotaStatus, upgradePrompt *UpgradePrompt) string {
	if quotaStatus.TotalConversionsRemaining <= 0 {
		return "upgrade_plan"
	}

	if quotaStatus.TotalConversionsRemaining <= 2 {
		return "consider_upgrade"
	}

	if quotaStatus.UsagePercentage >= 80 {
		return "monitor_usage"
	}

	return "continue_normal"
}

// InvalidateCache invalidates dashboard cache for a user
func (s *Service) InvalidateCache(ctx context.Context, userID string) error {
	return s.store.InvalidateDashboardCache(ctx, userID)
}

// GetQuotaStatus retrieves current quota status
func (s *Service) GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error) {
	return s.store.GetQuotaStatus(ctx, userID)
}

// GetConversionHistory retrieves conversion history
func (s *Service) GetConversionHistory(ctx context.Context, userID string, limit int) (ConversionHistory, error) {
	return s.store.GetConversionHistory(ctx, userID, limit)
}

// GetVendorGallery retrieves vendor gallery information
func (s *Service) GetVendorGallery(ctx context.Context, limit int) (VendorGallery, error) {
	return s.store.GetVendorGallery(ctx, limit)
}

// GetPlanStatus retrieves current plan status
func (s *Service) GetPlanStatus(ctx context.Context, userID string) (PlanStatus, error) {
	return s.store.GetPlanStatus(ctx, userID)
}

// GetDashboardStatistics retrieves dashboard statistics
func (s *Service) GetDashboardStatistics(ctx context.Context, userID string) (DashboardStatistics, error) {
	return s.store.GetDashboardStatistics(ctx, userID)
}

// GetRecentActivity retrieves recent user activity
func (s *Service) GetRecentActivity(ctx context.Context, userID string, limit int) ([]RecentActivity, error) {
	return s.store.GetRecentActivity(ctx, userID, limit)
}

// Helper function to calculate usage percentage
func calculateUsagePercentage(used, limit int) float64 {
	if limit <= 0 {
		return 0
	}
	return math.Min(float64(used)/float64(limit)*100, 100)
}

// Helper function to determine urgency level
func getUrgencyLevel(remaining int) string {
	if remaining <= 0 {
		return UrgencyHigh
	}
	if remaining <= 2 {
		return UrgencyMedium
	}
	return UrgencyLow
}
