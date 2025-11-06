package payment

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/wire"
)

// ProviderSet is the wire provider set for payment package
var ProviderSet = wire.NewSet(
	NewPaymentStore,
	NewZarinpalGateway,
	NewService,
	NewHandler,
	wire.Bind(new(PaymentStore), new(*PaymentStoreImpl)),
	wire.Bind(new(PaymentGateway), new(*ZarinpalGateway)),
)

// NewZarinpalGatewayFromConfig creates a Zarinpal gateway from config
func NewZarinpalGatewayFromConfig(configService PaymentConfigService) *ZarinpalGateway {
	return NewZarinpalGateway(
		configService.GetZarinpalMerchantID(),
		configService.GetZarinpalBaseURL(),
	)
}

// NewPaymentService creates a payment service with all dependencies
func NewPaymentService(
	db *sql.DB,
	userService UserService,
	notifier NotificationService,
	quotaService QuotaService,
	auditLogger AuditLogger,
	rateLimiter RateLimiter,
	configService PaymentConfigService,
) *Service {
	store := NewPaymentStore(db)
	gateway := NewZarinpalGatewayFromConfig(configService)

	return NewService(
		store,
		gateway,
		userService,
		notifier,
		quotaService,
		auditLogger,
		rateLimiter,
		configService,
	)
}

// WirePaymentService creates a payment service with all dependencies
func WirePaymentService(db *sql.DB) (*Service, *Handler) {
	// Create store
	store := NewPaymentStore(db)

	// Create mock dependencies (replace with real implementations in production)
	userService := NewMockUserService()
	notifier := NewMockNotificationService()
	quotaService := NewMockQuotaService()
	auditLogger := NewMockAuditLogger()
	rateLimiter := NewMockRateLimiter()
	configService := NewMockConfigService()

	// Create gateway
	gateway := NewZarinpalGateway("test-merchant-id", "https://api.zarinpal.com")

	// Create service
	service := NewService(
		store,
		gateway,
		userService,
		notifier,
		quotaService,
		auditLogger,
		rateLimiter,
		configService,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// Mock implementations for testing and development
type MockUserService struct{}

func NewMockUserService() *MockUserService {
	return &MockUserService{}
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{
		"id":    userID,
		"email": "user@example.com",
		"phone": "+1234567890",
	}, nil
}

func (m *MockUserService) GetUserPlan(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{
		"userID": userID,
		"planID": "basic",
		"status": "active",
	}, nil
}

func (m *MockUserService) UpdateUserPlan(ctx context.Context, planID string, status string) (interface{}, error) {
	return map[string]interface{}{
		"planID": planID,
		"status": status,
	}, nil
}

func (m *MockUserService) CreateUserPlan(ctx context.Context, userID string, planName string) (interface{}, error) {
	return map[string]interface{}{
		"userID":   userID,
		"planName": planName,
		"status":   "active",
	}, nil
}

func (m *MockUserService) UpdateUserQuota(ctx context.Context, userID string, quotaType string, amount int) error {
	return nil
}

type MockNotificationService struct{}

func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{}
}

func (m *MockNotificationService) SendPaymentNotification(ctx context.Context, userID string, paymentID string, status string) error {
	return nil
}

func (m *MockNotificationService) SendPaymentSuccess(ctx context.Context, userID string, paymentID string, planName string) error {
	return nil
}

func (m *MockNotificationService) SendPaymentFailed(ctx context.Context, userID string, paymentID string, reason string) error {
	return nil
}

func (m *MockNotificationService) SendPlanActivated(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *MockNotificationService) SendPlanExpired(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *MockNotificationService) SendEmail(ctx context.Context, email string, subject string, body string) error {
	return nil
}

func (m *MockNotificationService) SendSMS(ctx context.Context, phone string, message string) error {
	return nil
}

type MockQuotaService struct{}

func NewMockQuotaService() *MockQuotaService {
	return &MockQuotaService{}
}

func (m *MockQuotaService) CheckUserQuota(ctx context.Context, userID string) (bool, error) {
	return true, nil
}

func (m *MockQuotaService) UpdateUserQuota(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *MockQuotaService) ResetMonthlyQuota(ctx context.Context, userID string) error {
	return nil
}

func (m *MockQuotaService) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{
		"remaining": 100,
		"total":     1000,
		"used":      900,
	}, nil
}

type MockAuditLogger struct{}

func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{}
}

func (m *MockAuditLogger) LogPaymentAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error {
	return nil
}

type MockRateLimiter struct{}

func NewMockRateLimiter() *MockRateLimiter {
	return &MockRateLimiter{}
}

func (m *MockRateLimiter) CheckRateLimit(ctx context.Context, userID string) (bool, error) {
	return true, nil
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	return true
}

func (m *MockRateLimiter) RecordRequest(ctx context.Context, userID string) error {
	return nil
}

type MockConfigService struct{}

func NewMockConfigService() *MockConfigService {
	return &MockConfigService{}
}

func (m *MockConfigService) GetZarinpalMerchantID() string {
	return "test-merchant-id"
}

func (m *MockConfigService) GetZarinpalBaseURL() string {
	return "https://api.zarinpal.com"
}

func (m *MockConfigService) GetZibalMerchantID() string {
	return "test-merchant-id"
}

func (m *MockConfigService) GetZibalBaseURL() string {
	return "https://gateway.zibal.ir"
}

func (m *MockConfigService) GetPaymentCallbackURL() string {
	return "https://example.com/callback"
}

func (m *MockConfigService) GetPaymentReturnURL() string {
	return "https://example.com/return"
}

func (m *MockConfigService) GetPaymentExpiryMinutes() int {
	return 30
}

func (m *MockConfigService) GetPaymentTimeout() time.Duration {
	return 30 * time.Minute
}
