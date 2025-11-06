package payment

import (
	"context"
	"time"
)

// PaymentStore defines the interface for payment data operations
type PaymentStore interface {
	// Payment operations
	CreatePayment(ctx context.Context, payment Payment) (Payment, error)
	GetPayment(ctx context.Context, paymentID string) (Payment, error)
	GetPaymentByTrackID(ctx context.Context, trackID string) (Payment, error)
	UpdatePayment(ctx context.Context, paymentID string, updates map[string]interface{}) (Payment, error)
	GetPaymentHistory(ctx context.Context, userID string, req PaymentHistoryRequest) (PaymentHistoryResponse, error)

	// Plan operations
	GetPlan(ctx context.Context, planID string) (PaymentPlan, error)
	GetAllPlans(ctx context.Context) ([]PaymentPlan, error)
	CreatePlan(ctx context.Context, plan PaymentPlan) (PaymentPlan, error)
	UpdatePlan(ctx context.Context, planID string, updates map[string]interface{}) (PaymentPlan, error)

	// User plan operations
	GetUserActivePlan(ctx context.Context, userID string) (PaymentPlan, error)
	ActivateUserPlan(ctx context.Context, userID string, planID string, paymentID string) error
	DeactivateUserPlan(ctx context.Context, userID string) error
}

// PaymentGateway defines the interface for payment gateway operations
type PaymentGateway interface {
	// Create payment request
	CreatePayment(ctx context.Context, req ZarinpalRequest) (ZarinpalResponse, error)

	// Verify payment
	VerifyPayment(ctx context.Context, req ZarinpalVerifyRequest) (ZarinpalVerifyResponse, error)

	// Get payment URL
	GetPaymentURL(trackID string) string

	// Get gateway name
	GetGatewayName() string
}

// UserService defines the interface for user operations
type UserService interface {
	GetUserPlan(ctx context.Context, userID string) (interface{}, error)
	UpdateUserPlan(ctx context.Context, planID string, status string) (interface{}, error)
	CreateUserPlan(ctx context.Context, userID string, planName string) (interface{}, error)
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendPaymentSuccess(ctx context.Context, userID string, paymentID string, planName string) error
	SendPaymentFailed(ctx context.Context, userID string, paymentID string, reason string) error
	SendPlanActivated(ctx context.Context, userID string, planName string) error
	SendPlanExpired(ctx context.Context, userID string, planName string) error
}

// QuotaService defines the interface for quota management
type QuotaService interface {
	UpdateUserQuota(ctx context.Context, userID string, planName string) error
	ResetMonthlyQuota(ctx context.Context, userID string) error
	GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogPaymentAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) bool
}

// PaymentConfigService defines the interface for payment configuration
type PaymentConfigService interface {
	GetZarinpalMerchantID() string
	GetZarinpalBaseURL() string
	GetZibalMerchantID() string
	GetZibalBaseURL() string
	GetPaymentCallbackURL() string
	GetPaymentReturnURL() string
	GetPaymentExpiryMinutes() int
}
