package admin

import (
	"context"
)

// Store defines the interface for admin data operations
type Store interface {
	// User operations
	GetUsers(ctx context.Context, req UserListRequest) (UserListResponse, error)
	GetUser(ctx context.Context, userID string) (AdminUser, error)
	UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (AdminUser, error)
	DeleteUser(ctx context.Context, userID string) error
	GetUserStats(ctx context.Context) (int, int, error) // total, active

	// Vendor operations
	GetVendors(ctx context.Context, req VendorListRequest) (VendorListResponse, error)
	GetVendor(ctx context.Context, vendorID string) (AdminVendor, error)
	UpdateVendor(ctx context.Context, vendorID string, req UpdateVendorRequest) (AdminVendor, error)
	DeleteVendor(ctx context.Context, vendorID string) error
	GetVendorStats(ctx context.Context) (int, int, error) // total, active

	// Plan operations
	GetPlans(ctx context.Context, req PlanListRequest) (PlanListResponse, error)
	GetPlan(ctx context.Context, planID string) (AdminPlan, error)
	CreatePlan(ctx context.Context, req CreatePlanRequest) (AdminPlan, error)
	UpdatePlan(ctx context.Context, planID string, req UpdatePlanRequest) (AdminPlan, error)
	DeletePlan(ctx context.Context, planID string) error

	// Payment operations
	GetPayments(ctx context.Context, req PaymentListRequest) (PaymentListResponse, error)
	GetPayment(ctx context.Context, paymentID string) (AdminPayment, error)
	GetPaymentStats(ctx context.Context) (int, int64, error) // total, revenue

	// Conversion operations
	GetConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error)
	GetConversion(ctx context.Context, conversionID string) (AdminConversion, error)
	GetConversionStats(ctx context.Context) (int, int, int, error) // total, pending, failed

	// Image operations
	GetImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error)
	GetImage(ctx context.Context, imageID string) (AdminImage, error)
	GetImageStats(ctx context.Context) (int, error) // total

	// Audit log operations
	GetAuditLogs(ctx context.Context, req AuditLogListRequest) (AuditLogListResponse, error)
	CreateAuditLog(ctx context.Context, log AuditLog) error

	// Quota operations
	RevokeUserQuota(ctx context.Context, userID string, quotaType string, amount int, reason string) error
	RevokeVendorQuota(ctx context.Context, vendorID string, quotaType string, amount int, reason string) error
	RevokeUserPlan(ctx context.Context, userID string, reason string) error

	// Statistics
	GetSystemStats(ctx context.Context) (AdminStats, error)
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendNotification(ctx context.Context, userID string, notificationType string, data map[string]interface{}) error
	SendEmail(ctx context.Context, email string, subject string, body string) error
	SendSMS(ctx context.Context, phone string, message string) error
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogAction(ctx context.Context, userID *string, actorType, action, resource string, resourceID *string, metadata map[string]interface{}) error
}

// AdminService defines the main admin service interface
type AdminService interface {
	// User management
	GetUsers(ctx context.Context, req UserListRequest) (UserListResponse, error)
	GetUser(ctx context.Context, userID string) (AdminUser, error)
	UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (AdminUser, error)
	DeleteUser(ctx context.Context, userID string) error
	SuspendUser(ctx context.Context, userID string, reason string) error
	ActivateUser(ctx context.Context, userID string) error

	// Vendor management
	GetVendors(ctx context.Context, req VendorListRequest) (VendorListResponse, error)
	GetVendor(ctx context.Context, vendorID string) (AdminVendor, error)
	UpdateVendor(ctx context.Context, vendorID string, req UpdateVendorRequest) (AdminVendor, error)
	DeleteVendor(ctx context.Context, vendorID string) error
	SuspendVendor(ctx context.Context, vendorID string, reason string) error
	ActivateVendor(ctx context.Context, vendorID string) error
	VerifyVendor(ctx context.Context, vendorID string) error

	// Plan management
	GetPlans(ctx context.Context, req PlanListRequest) (PlanListResponse, error)
	GetPlan(ctx context.Context, planID string) (AdminPlan, error)
	CreatePlan(ctx context.Context, req CreatePlanRequest) (AdminPlan, error)
	UpdatePlan(ctx context.Context, planID string, req UpdatePlanRequest) (AdminPlan, error)
	DeletePlan(ctx context.Context, planID string) error

	// Payment management
	GetPayments(ctx context.Context, req PaymentListRequest) (PaymentListResponse, error)
	GetPayment(ctx context.Context, paymentID string) (AdminPayment, error)

	// Conversion management
	GetConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error)
	GetConversion(ctx context.Context, conversionID string) (AdminConversion, error)

	// Image management
	GetImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error)
	GetImage(ctx context.Context, imageID string) (AdminImage, error)

	// Audit trail
	GetAuditLogs(ctx context.Context, req AuditLogListRequest) (AuditLogListResponse, error)

	// Quota management
	RevokeUserQuota(ctx context.Context, req RevokeQuotaRequest) error
	RevokeVendorQuota(ctx context.Context, vendorID string, quotaType string, amount int, reason string) error
	RevokeUserPlan(ctx context.Context, req RevokePlanRequest) error

	// Statistics
	GetSystemStats(ctx context.Context) (AdminStats, error)
	GetUserStats(ctx context.Context) (int, int, error)
	GetVendorStats(ctx context.Context) (int, int, error)
	GetPaymentStats(ctx context.Context) (int, int64, error)
	GetConversionStats(ctx context.Context) (int, int, int, error)
	GetImageStats(ctx context.Context) (int, error)
}
