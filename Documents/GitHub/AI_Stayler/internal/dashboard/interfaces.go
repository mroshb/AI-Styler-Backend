package dashboard

import (
	"context"
)

// Store defines the interface for dashboard data operations
type Store interface {
	// Dashboard data operations
	GetDashboardData(ctx context.Context, userID string, req DashboardRequest) (DashboardData, error)
	GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error)
	GetConversionHistory(ctx context.Context, userID string, limit int) (ConversionHistory, error)
	GetVendorGallery(ctx context.Context, limit int) (VendorGallery, error)
	GetPlanStatus(ctx context.Context, userID string) (PlanStatus, error)
	GetDashboardStatistics(ctx context.Context, userID string) (DashboardStatistics, error)
	GetRecentActivity(ctx context.Context, userID string, limit int) ([]RecentActivity, error)

	// Quota operations
	CheckQuota(ctx context.Context, userID string) (QuotaCheckResponse, error)
	ShouldShowUpgradePrompt(ctx context.Context, userID string) (*UpgradePrompt, error)

	// Cache operations
	GetCachedDashboardData(ctx context.Context, userID string) (*DashboardData, error)
	SetCachedDashboardData(ctx context.Context, userID string, data DashboardData, ttl int) error
	InvalidateDashboardCache(ctx context.Context, userID string) error
}

// UserService defines the interface for user operations
type UserService interface {
	GetProfile(ctx context.Context, userID string) (interface{}, error)
	GetQuotaStatus(ctx context.Context, userID string) (interface{}, error)
	GetConversionHistory(ctx context.Context, userID string, req interface{}) (interface{}, error)
	GetUserPlan(ctx context.Context, userID string) (interface{}, error)
}

// ConversionService defines the interface for conversion operations
type ConversionService interface {
	GetConversions(ctx context.Context, req interface{}) (interface{}, error)
	GetConversion(ctx context.Context, conversionID string) (interface{}, error)
	GetConversionStatistics(ctx context.Context, userID string) (interface{}, error)
}

// VendorService defines the interface for vendor operations
type VendorService interface {
	GetPublicAlbums(ctx context.Context, req interface{}) (interface{}, error)
	GetPublicImages(ctx context.Context, req interface{}) (interface{}, error)
	GetVendorStats(ctx context.Context, vendorID string) (interface{}, error)
}

// PaymentService defines the interface for payment operations
type PaymentService interface {
	GetUserPayments(ctx context.Context, userID string) (interface{}, error)
	GetAvailablePlans(ctx context.Context) (interface{}, error)
	GetPaymentHistory(ctx context.Context, userID string, req interface{}) (interface{}, error)
}

// NotificationService defines the interface for notification operations
type NotificationService interface {
	GetUserNotifications(ctx context.Context, userID string, limit int) (interface{}, error)
	MarkNotificationAsRead(ctx context.Context, userID, notificationID string) error
}

// Cache defines the interface for caching operations
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl int) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordDashboardView(ctx context.Context, userID string) error
	RecordQuotaCheck(ctx context.Context, userID string, canConvert bool) error
	RecordUpgradePromptShown(ctx context.Context, userID string, planName string) error
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogDashboardAccess(ctx context.Context, userID string, requestType string) error
	LogQuotaCheck(ctx context.Context, userID string, result interface{}) error
}
