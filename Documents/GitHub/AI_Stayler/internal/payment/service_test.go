package payment

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Mock implementations for testing

type mockStore struct {
	payments map[string]Payment
	plans    map[string]PaymentPlan
}

func newMockStore() *mockStore {
	return &mockStore{
		payments: make(map[string]Payment),
		plans: map[string]PaymentPlan{
			"plan-1": {
				ID:                      "plan-1",
				Name:                    "basic",
				DisplayName:             "Basic Plan",
				PricePerMonthCents:      50000,
				MonthlyConversionsLimit: 20,
				IsActive:                true,
			},
		},
	}
}

func (m *mockStore) CreatePayment(ctx context.Context, payment Payment) (Payment, error) {
	m.payments[payment.ID] = payment
	return payment, nil
}

func (m *mockStore) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	payment, exists := m.payments[paymentID]
	if !exists {
		return Payment{}, errors.New("payment not found")
	}
	return payment, nil
}

func (m *mockStore) GetPaymentByTrackID(ctx context.Context, trackID string) (Payment, error) {
	for _, payment := range m.payments {
		if payment.GatewayTrackID != nil && *payment.GatewayTrackID == trackID {
			return payment, nil
		}
	}
	return Payment{}, errors.New("payment not found")
}

func (m *mockStore) UpdatePayment(ctx context.Context, paymentID string, updates map[string]interface{}) (Payment, error) {
	payment, exists := m.payments[paymentID]
	if !exists {
		return Payment{}, errors.New("payment not found")
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "status":
			payment.Status = value.(string)
		case "gateway_track_id":
			if v, ok := value.(string); ok {
				payment.GatewayTrackID = &v
			}
		case "gateway_ref_number":
			if v, ok := value.(string); ok {
				payment.GatewayRefNumber = &v
			}
		case "gateway_card_number":
			if v, ok := value.(string); ok {
				payment.GatewayCardNumber = &v
			}
		case "paid_at":
			if v, ok := value.(time.Time); ok {
				payment.PaidAt = &v
			}
		}
	}

	payment.UpdatedAt = time.Now()
	m.payments[paymentID] = payment
	return payment, nil
}

func (m *mockStore) GetPaymentHistory(ctx context.Context, userID string, req PaymentHistoryRequest) (PaymentHistoryResponse, error) {
	var userPayments []Payment
	for _, payment := range m.payments {
		if payment.UserID == userID {
			userPayments = append(userPayments, payment)
		}
	}

	// Convert Payment to PaymentHistoryItem
	var historyItems []PaymentHistoryItem
	for _, payment := range userPayments {
		// Get plan details for the payment
		plan, exists := m.plans[payment.PlanID]
		if !exists {
			// Use default values if plan not found
			plan = PaymentPlan{
				Name:        "unknown",
				DisplayName: "Unknown Plan",
			}
		}

		historyItem := PaymentHistoryItem{
			PaymentID:        payment.ID,
			UserID:           payment.UserID,
			VendorID:         payment.VendorID,
			PlanID:           payment.PlanID,
			Amount:           payment.Amount,
			Currency:         payment.Currency,
			Status:           payment.Status,
			PaymentMethod:    payment.PaymentMethod,
			Gateway:          payment.Gateway,
			GatewayTrackID:   payment.GatewayTrackID,
			GatewayRefNumber: payment.GatewayRefNumber,
			Description:      payment.Description,
			CreatedAt:        payment.CreatedAt,
			PaidAt:           payment.PaidAt,
			PlanName:         plan.Name,
			PlanDisplayName:  plan.DisplayName,
		}
		historyItems = append(historyItems, historyItem)
	}

	return PaymentHistoryResponse{
		Payments:   historyItems,
		Total:      len(historyItems),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *mockStore) GetPlan(ctx context.Context, planID string) (PaymentPlan, error) {
	plan, exists := m.plans[planID]
	if !exists {
		return PaymentPlan{}, errors.New("plan not found")
	}
	return plan, nil
}

func (m *mockStore) GetAllPlans(ctx context.Context) ([]PaymentPlan, error) {
	var plans []PaymentPlan
	for _, plan := range m.plans {
		plans = append(plans, plan)
	}
	return plans, nil
}

func (m *mockStore) CreatePlan(ctx context.Context, plan PaymentPlan) (PaymentPlan, error) {
	m.plans[plan.ID] = plan
	return plan, nil
}

func (m *mockStore) UpdatePlan(ctx context.Context, planID string, updates map[string]interface{}) (PaymentPlan, error) {
	plan, exists := m.plans[planID]
	if !exists {
		return PaymentPlan{}, errors.New("plan not found")
	}
	plan.UpdatedAt = time.Now()
	m.plans[planID] = plan
	return plan, nil
}

func (m *mockStore) GetUserActivePlan(ctx context.Context, userID string) (PaymentPlan, error) {
	return PaymentPlan{}, errors.New("no active plan found")
}

func (m *mockStore) ActivateUserPlan(ctx context.Context, userID string, planID string, paymentID string) error {
	return nil
}

func (m *mockStore) DeactivateUserPlan(ctx context.Context, userID string) error {
	return nil
}

type mockGateway struct {
	createPaymentResponse ZarinpalResponse
	createPaymentError    error
	verifyPaymentResponse ZarinpalVerifyResponse
	verifyPaymentError    error
}

func newMockGateway() *mockGateway {
	return &mockGateway{
		createPaymentResponse: ZarinpalResponse{
			TrackID: "test-track-id",
			Result:  ZarinpalSuccess,
			Message: "Success",
		},
		verifyPaymentResponse: ZarinpalVerifyResponse{
			Result:     ZarinpalSuccess,
			Message:    "Success",
			Amount:     50000,
			RefNumber:  "test-ref-number",
			CardNumber: "1234****5678",
		},
	}
}

func (m *mockGateway) CreatePayment(ctx context.Context, req ZarinpalRequest) (ZarinpalResponse, error) {
	if m.createPaymentError != nil {
		return ZarinpalResponse{}, m.createPaymentError
	}
	return m.createPaymentResponse, nil
}

func (m *mockGateway) VerifyPayment(ctx context.Context, req ZarinpalVerifyRequest) (ZarinpalVerifyResponse, error) {
	if m.verifyPaymentError != nil {
		return ZarinpalVerifyResponse{}, m.verifyPaymentError
	}
	return m.verifyPaymentResponse, nil
}

func (m *mockGateway) GetPaymentURL(trackID string) string {
	return "https://gateway.zibal.ir/start/" + trackID
}

func (m *mockGateway) GetGatewayName() string {
	return "zarinpal"
}

type mockUserService struct{}
type mockNotificationService struct{}
type mockQuotaService struct{}
type mockAuditLogger struct{}
type mockRateLimiter struct{}
type mockPaymentConfigService struct{}

func (m *mockUserService) GetUserPlan(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

func (m *mockUserService) UpdateUserPlan(ctx context.Context, planID string, status string) (interface{}, error) {
	return nil, nil
}

func (m *mockUserService) CreateUserPlan(ctx context.Context, userID string, planName string) (interface{}, error) {
	return nil, nil
}

func (m *mockNotificationService) SendPaymentSuccess(ctx context.Context, userID string, paymentID string, planName string) error {
	return nil
}

func (m *mockNotificationService) SendPaymentFailed(ctx context.Context, userID string, paymentID string, reason string) error {
	return nil
}

func (m *mockNotificationService) SendPlanActivated(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *mockNotificationService) SendPlanExpired(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *mockQuotaService) UpdateUserQuota(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *mockQuotaService) ResetMonthlyQuota(ctx context.Context, userID string) error {
	return nil
}

func (m *mockQuotaService) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

func (m *mockAuditLogger) LogPaymentAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	return true
}

func (m *mockPaymentConfigService) GetZarinpalMerchantID() string {
	return "test-merchant"
}

func (m *mockPaymentConfigService) GetZarinpalBaseURL() string {
	return "https://gateway.zibal.ir"
}

func (m *mockPaymentConfigService) GetPaymentCallbackURL() string {
	return "https://test.com/callback"
}

func (m *mockPaymentConfigService) GetPaymentReturnURL() string {
	return "https://test.com/return"
}

func (m *mockPaymentConfigService) GetPaymentExpiryMinutes() int {
	return 30
}

// Tests

func TestCreatePayment(t *testing.T) {
	store := newMockStore()
	gateway := newMockGateway()
	userService := &mockUserService{}
	notifier := &mockNotificationService{}
	quotaService := &mockQuotaService{}
	auditLogger := &mockAuditLogger{}
	rateLimiter := &mockRateLimiter{}
	configService := &mockPaymentConfigService{}

	service := NewService(store, gateway, userService, notifier, quotaService, auditLogger, rateLimiter, configService)

	req := CreatePaymentRequest{
		PlanID:      "plan-1",
		ReturnURL:   "https://test.com/return",
		Description: "Test payment",
	}

	resp, err := service.CreatePayment(context.Background(), "user-1", req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.PaymentID == "" {
		t.Error("Expected payment ID to be set")
	}

	if resp.GatewayURL == "" {
		t.Error("Expected gateway URL to be set")
	}

	if resp.TrackID != "test-track-id" {
		t.Errorf("Expected track ID to be 'test-track-id', got %s", resp.TrackID)
	}
}

func TestCreatePaymentInvalidPlan(t *testing.T) {
	store := newMockStore()
	gateway := newMockGateway()
	userService := &mockUserService{}
	notifier := &mockNotificationService{}
	quotaService := &mockQuotaService{}
	auditLogger := &mockAuditLogger{}
	rateLimiter := &mockRateLimiter{}
	configService := &mockPaymentConfigService{}

	service := NewService(store, gateway, userService, notifier, quotaService, auditLogger, rateLimiter, configService)

	req := CreatePaymentRequest{
		PlanID:      "invalid-plan",
		ReturnURL:   "https://test.com/return",
		Description: "Test payment",
	}

	_, err := service.CreatePayment(context.Background(), "user-1", req)
	if err == nil {
		t.Error("Expected error for invalid plan")
	}
}

func TestVerifyPayment(t *testing.T) {
	store := newMockStore()
	gateway := newMockGateway()
	userService := &mockUserService{}
	notifier := &mockNotificationService{}
	quotaService := &mockQuotaService{}
	auditLogger := &mockAuditLogger{}
	rateLimiter := &mockRateLimiter{}
	configService := &mockPaymentConfigService{}

	service := NewService(store, gateway, userService, notifier, quotaService, auditLogger, rateLimiter, configService)

	// Create a payment first
	payment := Payment{
		ID:             "payment-1",
		UserID:         "user-1",
		PlanID:         "plan-1",
		Amount:         50000,
		Status:         PaymentStatusPending,
		GatewayTrackID: stringPtr("test-track-id"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	store.payments["payment-1"] = payment

	webhook := PaymentWebhook{
		TrackID: "test-track-id",
		Success: true,
		Status:  ZarinpalStatusPaid,
	}

	err := service.VerifyPayment(context.Background(), webhook)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that payment was updated
	updatedPayment, exists := store.payments["payment-1"]
	if !exists {
		t.Fatal("Payment not found")
	}

	if updatedPayment.Status != PaymentStatusCompleted {
		t.Errorf("Expected payment status to be completed, got %s", updatedPayment.Status)
	}
}

func TestGetPaymentStatus(t *testing.T) {
	store := newMockStore()
	gateway := newMockGateway()
	userService := &mockUserService{}
	notifier := &mockNotificationService{}
	quotaService := &mockQuotaService{}
	auditLogger := &mockAuditLogger{}
	rateLimiter := &mockRateLimiter{}
	configService := &mockPaymentConfigService{}

	service := NewService(store, gateway, userService, notifier, quotaService, auditLogger, rateLimiter, configService)

	// Create a payment
	payment := Payment{
		ID:             "payment-1",
		UserID:         "user-1",
		PlanID:         "plan-1",
		Amount:         50000,
		Status:         PaymentStatusCompleted,
		GatewayTrackID: stringPtr("test-track-id"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	store.payments["payment-1"] = payment

	resp, err := service.GetPaymentStatus(context.Background(), "user-1", "payment-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.PaymentID != "payment-1" {
		t.Errorf("Expected payment ID to be 'payment-1', got %s", resp.PaymentID)
	}

	if resp.Status != PaymentStatusCompleted {
		t.Errorf("Expected status to be completed, got %s", resp.Status)
	}
}

func TestGetPlans(t *testing.T) {
	store := newMockStore()
	gateway := newMockGateway()
	userService := &mockUserService{}
	notifier := &mockNotificationService{}
	quotaService := &mockQuotaService{}
	auditLogger := &mockAuditLogger{}
	rateLimiter := &mockRateLimiter{}
	configService := &mockPaymentConfigService{}

	service := NewService(store, gateway, userService, notifier, quotaService, auditLogger, rateLimiter, configService)

	plans, err := service.GetPlans(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plans) == 0 {
		t.Error("Expected at least one plan")
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}
