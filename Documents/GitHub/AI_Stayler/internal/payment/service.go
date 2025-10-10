package payment

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Service provides payment management functionality
type Service struct {
	store         PaymentStore
	gateway       PaymentGateway
	userService   UserService
	notifier      NotificationService
	quotaService  QuotaService
	auditLogger   AuditLogger
	rateLimiter   RateLimiter
	configService PaymentConfigService
}

// NewService creates a new payment service
func NewService(
	store PaymentStore,
	gateway PaymentGateway,
	userService UserService,
	notifier NotificationService,
	quotaService QuotaService,
	auditLogger AuditLogger,
	rateLimiter RateLimiter,
	configService PaymentConfigService,
) *Service {
	return &Service{
		store:         store,
		gateway:       gateway,
		userService:   userService,
		notifier:      notifier,
		quotaService:  quotaService,
		auditLogger:   auditLogger,
		rateLimiter:   rateLimiter,
		configService: configService,
	}
}

// CreatePayment creates a new payment for a plan
func (s *Service) CreatePayment(ctx context.Context, userID string, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	// Validate input
	if req.PlanID == "" {
		return CreatePaymentResponse{}, errors.New("plan ID is required")
	}
	if req.ReturnURL == "" {
		return CreatePaymentResponse{}, errors.New("return URL is required")
	}

	// Check rate limiting
	rateLimitKey := fmt.Sprintf("payment:user:%s", userID)
	if !s.rateLimiter.Allow(ctx, rateLimitKey, 5, time.Hour) {
		return CreatePaymentResponse{}, errors.New("rate limit exceeded")
	}

	// Get plan details
	plan, err := s.store.GetPlan(ctx, req.PlanID)
	if err != nil {
		return CreatePaymentResponse{}, fmt.Errorf("failed to get plan: %w", err)
	}

	if !plan.IsActive {
		return CreatePaymentResponse{}, errors.New("plan is not active")
	}

	// Check if user already has an active plan
	_, err = s.store.GetUserActivePlan(ctx, userID)
	if err == nil {
		return CreatePaymentResponse{}, errors.New("user already has an active plan")
	}

	// Generate payment ID
	paymentID := generatePaymentID()

	// Create payment record
	payment := Payment{
		ID:            paymentID,
		UserID:        userID,
		PlanID:        req.PlanID,
		Amount:        plan.PricePerMonthCents,
		Currency:      CurrencyIRR,
		Status:        PaymentStatusPending,
		PaymentMethod: PaymentMethodZarinpal,
		Gateway:       s.gateway.GetGatewayName(),
		Description:   req.Description,
		CallbackURL:   s.configService.GetPaymentCallbackURL(),
		ReturnURL:     req.ReturnURL,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ExpiresAt:     timePtr(time.Now().Add(time.Duration(s.configService.GetPaymentExpiryMinutes()) * time.Minute)),
	}

	// Save payment to database
	_, err = s.store.CreatePayment(ctx, payment)
	if err != nil {
		return CreatePaymentResponse{}, fmt.Errorf("failed to create payment record: %w", err)
	}

	// Create gateway payment request
	gatewayReq := ZarinpalRequest{
		Amount:      plan.PricePerMonthCents,
		CallbackURL: s.configService.GetPaymentCallbackURL(),
		Description: req.Description,
		OrderID:     paymentID,
	}

	// Send request to gateway
	gatewayResp, err := s.gateway.CreatePayment(ctx, gatewayReq)
	if err != nil {
		// Update payment status to failed
		s.store.UpdatePayment(ctx, paymentID, map[string]interface{}{
			"status": PaymentStatusFailed,
		})
		return CreatePaymentResponse{}, fmt.Errorf("failed to create gateway payment: %w", err)
	}

	// Update payment with gateway track ID
	updatedPayment, err := s.store.UpdatePayment(ctx, paymentID, map[string]interface{}{
		"gateway_track_id": gatewayResp.TrackID,
	})
	if err != nil {
		return CreatePaymentResponse{}, fmt.Errorf("failed to update payment with track ID: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"payment_id": paymentID,
		"plan_id":    req.PlanID,
		"amount":     plan.PricePerMonthCents,
		"track_id":   gatewayResp.TrackID,
	}
	_ = s.auditLogger.LogPaymentAction(ctx, userID, "payment_created", metadata)

	return CreatePaymentResponse{
		PaymentID:  paymentID,
		GatewayURL: s.gateway.GetPaymentURL(gatewayResp.TrackID),
		TrackID:    gatewayResp.TrackID,
		ExpiresAt:  *updatedPayment.ExpiresAt,
	}, nil
}

// VerifyPayment verifies a payment using webhook data
func (s *Service) VerifyPayment(ctx context.Context, webhook PaymentWebhook) error {
	// Get payment by track ID
	payment, err := s.store.GetPaymentByTrackID(ctx, webhook.TrackID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Check if payment is already processed
	if payment.Status == PaymentStatusCompleted {
		return nil // Already processed
	}

	// Verify with gateway
	verifyReq := ZarinpalVerifyRequest{
		TrackID: webhook.TrackID,
	}

	verifyResp, err := s.gateway.VerifyPayment(ctx, verifyReq)
	if err != nil {
		// Update payment status to failed
		s.store.UpdatePayment(ctx, payment.ID, map[string]interface{}{
			"status": PaymentStatusFailed,
		})
		return fmt.Errorf("failed to verify payment: %w", err)
	}

	// Check if payment was successful
	if verifyResp.Result != ZarinpalSuccess {
		// Update payment status to failed
		s.store.UpdatePayment(ctx, payment.ID, map[string]interface{}{
			"status": PaymentStatusFailed,
		})
		return fmt.Errorf("payment verification failed: %s", verifyResp.Message)
	}

	// Update payment with success details
	now := time.Now()
	updates := map[string]interface{}{
		"status":              PaymentStatusCompleted,
		"gateway_ref_number":  verifyResp.RefNumber,
		"gateway_card_number": verifyResp.CardNumber,
		"paid_at":             now,
	}

	updatedPayment, err := s.store.UpdatePayment(ctx, payment.ID, updates)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Activate user plan
	err = s.store.ActivateUserPlan(ctx, payment.UserID, payment.PlanID, payment.ID)
	if err != nil {
		return fmt.Errorf("failed to activate user plan: %w", err)
	}

	// Update user quota
	err = s.quotaService.UpdateUserQuota(ctx, payment.UserID, updatedPayment.PlanID)
	if err != nil {
		// Log error but don't fail the payment
		_ = s.auditLogger.LogPaymentAction(ctx, payment.UserID, "quota_update_failed", map[string]interface{}{
			"payment_id": payment.ID,
			"error":      err.Error(),
		})
	}

	// Get plan details for notification
	plan, err := s.store.GetPlan(ctx, payment.PlanID)
	if err != nil {
		// Log error but don't fail the payment
		_ = s.auditLogger.LogPaymentAction(ctx, payment.UserID, "plan_lookup_failed", map[string]interface{}{
			"payment_id": payment.ID,
			"error":      err.Error(),
		})
		plan = PaymentPlan{Name: "Unknown Plan"}
	}

	// Send notifications
	_ = s.notifier.SendPaymentSuccess(ctx, payment.UserID, payment.ID, plan.Name)
	_ = s.notifier.SendPlanActivated(ctx, payment.UserID, plan.Name)

	// Log successful payment
	metadata := map[string]interface{}{
		"payment_id":  payment.ID,
		"plan_id":     payment.PlanID,
		"amount":      payment.Amount,
		"ref_number":  verifyResp.RefNumber,
		"card_number": verifyResp.CardNumber,
	}
	_ = s.auditLogger.LogPaymentAction(ctx, payment.UserID, "payment_completed", metadata)

	return nil
}

// GetPaymentStatus retrieves the status of a payment
func (s *Service) GetPaymentStatus(ctx context.Context, userID, paymentID string) (PaymentStatusResponse, error) {
	// Get payment
	payment, err := s.store.GetPayment(ctx, paymentID)
	if err != nil {
		return PaymentStatusResponse{}, fmt.Errorf("failed to get payment: %w", err)
	}

	// Verify ownership
	if payment.UserID != userID {
		return PaymentStatusResponse{}, errors.New("payment not found")
	}

	// Get plan details
	plan, err := s.store.GetPlan(ctx, payment.PlanID)
	if err != nil {
		return PaymentStatusResponse{}, fmt.Errorf("failed to get plan: %w", err)
	}

	// Build gateway info
	var gatewayInfo *GatewayInfo
	if payment.GatewayTrackID != nil {
		gatewayInfo = &GatewayInfo{
			TrackID:    *payment.GatewayTrackID,
			RefNumber:  getStringValue(payment.GatewayRefNumber),
			CardNumber: getStringValue(payment.GatewayCardNumber),
			Gateway:    payment.Gateway,
		}
	}

	return PaymentStatusResponse{
		PaymentID:   payment.ID,
		Status:      payment.Status,
		Amount:      payment.Amount,
		PlanName:    plan.Name,
		PaidAt:      payment.PaidAt,
		GatewayInfo: gatewayInfo,
	}, nil
}

// GetPaymentHistory retrieves payment history for a user
func (s *Service) GetPaymentHistory(ctx context.Context, userID string, req PaymentHistoryRequest) (PaymentHistoryResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	history, err := s.store.GetPaymentHistory(ctx, userID, req)
	if err != nil {
		return PaymentHistoryResponse{}, fmt.Errorf("failed to get payment history: %w", err)
	}

	return history, nil
}

// GetPlans retrieves all available payment plans
func (s *Service) GetPlans(ctx context.Context) ([]PaymentPlan, error) {
	plans, err := s.store.GetAllPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}

	return plans, nil
}

// GetUserActivePlan retrieves the user's active plan
func (s *Service) GetUserActivePlan(ctx context.Context, userID string) (PaymentPlan, error) {
	plan, err := s.store.GetUserActivePlan(ctx, userID)
	if err != nil {
		return PaymentPlan{}, fmt.Errorf("failed to get user active plan: %w", err)
	}

	return plan, nil
}

// CancelPayment cancels a pending payment
func (s *Service) CancelPayment(ctx context.Context, userID, paymentID string) error {
	// Get payment
	payment, err := s.store.GetPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Verify ownership
	if payment.UserID != userID {
		return errors.New("payment not found")
	}

	// Check if payment can be cancelled
	if payment.Status != PaymentStatusPending {
		return errors.New("payment cannot be cancelled")
	}

	// Update payment status
	_, err = s.store.UpdatePayment(ctx, paymentID, map[string]interface{}{
		"status": PaymentStatusCancelled,
	})
	if err != nil {
		return fmt.Errorf("failed to cancel payment: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"payment_id": paymentID,
		"plan_id":    payment.PlanID,
	}
	_ = s.auditLogger.LogPaymentAction(ctx, userID, "payment_cancelled", metadata)

	return nil
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// generatePaymentID generates a unique payment ID
func generatePaymentID() string {
	// Simple ID generation - in production, use a proper UUID library
	return fmt.Sprintf("pay_%d", time.Now().UnixNano())
}
