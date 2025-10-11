package admin

import (
	"context"
	"errors"
	"fmt"
)

// Service provides admin functionality
type Service struct {
	store       Store
	notifier    NotificationService
	auditLogger AuditLogger
}

// NewService creates a new admin service
func NewService(
	store Store,
	notifier NotificationService,
	auditLogger AuditLogger,
) *Service {
	return &Service{
		store:       store,
		notifier:    notifier,
		auditLogger: auditLogger,
	}
}

// User management

// GetUsers retrieves a list of users with pagination and filtering
func (s *Service) GetUsers(ctx context.Context, req UserListRequest) (UserListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetUsers(ctx, req)
}

// GetUser retrieves a specific user by ID
func (s *Service) GetUser(ctx context.Context, userID string) (AdminUser, error) {
	if userID == "" {
		return AdminUser{}, errors.New("user ID is required")
	}

	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return AdminUser{}, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user's information
func (s *Service) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (AdminUser, error) {
	if userID == "" {
		return AdminUser{}, errors.New("user ID is required")
	}

	// Validate role if provided
	if req.Role != nil {
		validRoles := []string{"user", "vendor", "admin"}
		valid := false
		for _, role := range validRoles {
			if *req.Role == role {
				valid = true
				break
			}
		}
		if !valid {
			return AdminUser{}, errors.New("invalid role")
		}
	}

	// Validate free conversions limit if provided
	if req.FreeConversionsLimit != nil && *req.FreeConversionsLimit < 0 {
		return AdminUser{}, errors.New("free conversions limit cannot be negative")
	}

	user, err := s.store.UpdateUser(ctx, userID, req)
	if err != nil {
		return AdminUser{}, fmt.Errorf("failed to update user: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"user_id": userID,
		"changes": req,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionUpdate, ResourceUser, &userID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return user, nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID is required")
	}

	err := s.store.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"user_id": userID,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionDelete, ResourceUser, &userID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// SuspendUser suspends a user account
func (s *Service) SuspendUser(ctx context.Context, userID string, reason string) error {
	if userID == "" {
		return errors.New("user ID is required")
	}
	if reason == "" {
		return errors.New("reason is required")
	}

	req := UpdateUserRequest{
		IsActive: boolPtr(false),
	}

	_, err := s.store.UpdateUser(ctx, userID, req)
	if err != nil {
		return fmt.Errorf("failed to suspend user: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"user_id": userID,
		"reason":  reason,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionSuspend, ResourceUser, &userID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	// Send notification to user
	notificationData := map[string]interface{}{
		"reason": reason,
		"action": "account_suspended",
	}
	if err := s.notifier.SendNotification(ctx, userID, "account_suspended", notificationData); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to send suspension notification: %v\n", err)
	}

	return nil
}

// ActivateUser activates a user account
func (s *Service) ActivateUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID is required")
	}

	req := UpdateUserRequest{
		IsActive: boolPtr(true),
	}

	_, err := s.store.UpdateUser(ctx, userID, req)
	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"user_id": userID,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionActivate, ResourceUser, &userID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	// Send notification to user
	notificationData := map[string]interface{}{
		"action": "account_activated",
	}
	if err := s.notifier.SendNotification(ctx, userID, "account_activated", notificationData); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to send activation notification: %v\n", err)
	}

	return nil
}

// Vendor management

// GetVendors retrieves a list of vendors with pagination and filtering
func (s *Service) GetVendors(ctx context.Context, req VendorListRequest) (VendorListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetVendors(ctx, req)
}

// GetVendor retrieves a specific vendor by ID
func (s *Service) GetVendor(ctx context.Context, vendorID string) (AdminVendor, error) {
	if vendorID == "" {
		return AdminVendor{}, errors.New("vendor ID is required")
	}

	vendor, err := s.store.GetVendor(ctx, vendorID)
	if err != nil {
		return AdminVendor{}, fmt.Errorf("failed to get vendor: %w", err)
	}

	return vendor, nil
}

// UpdateVendor updates a vendor's information
func (s *Service) UpdateVendor(ctx context.Context, vendorID string, req UpdateVendorRequest) (AdminVendor, error) {
	if vendorID == "" {
		return AdminVendor{}, errors.New("vendor ID is required")
	}

	// Validate free images limit if provided
	if req.FreeImagesLimit != nil && *req.FreeImagesLimit < 0 {
		return AdminVendor{}, errors.New("free images limit cannot be negative")
	}

	vendor, err := s.store.UpdateVendor(ctx, vendorID, req)
	if err != nil {
		return AdminVendor{}, fmt.Errorf("failed to update vendor: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"vendor_id": vendorID,
		"changes":   req,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionUpdate, ResourceVendor, &vendorID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return vendor, nil
}

// DeleteVendor deletes a vendor
func (s *Service) DeleteVendor(ctx context.Context, vendorID string) error {
	if vendorID == "" {
		return errors.New("vendor ID is required")
	}

	err := s.store.DeleteVendor(ctx, vendorID)
	if err != nil {
		return fmt.Errorf("failed to delete vendor: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"vendor_id": vendorID,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionDelete, ResourceVendor, &vendorID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// SuspendVendor suspends a vendor account
func (s *Service) SuspendVendor(ctx context.Context, vendorID string, reason string) error {
	if vendorID == "" {
		return errors.New("vendor ID is required")
	}
	if reason == "" {
		return errors.New("reason is required")
	}

	req := UpdateVendorRequest{
		IsActive: boolPtr(false),
	}

	_, err := s.store.UpdateVendor(ctx, vendorID, req)
	if err != nil {
		return fmt.Errorf("failed to suspend vendor: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"vendor_id": vendorID,
		"reason":    reason,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionSuspend, ResourceVendor, &vendorID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// ActivateVendor activates a vendor account
func (s *Service) ActivateVendor(ctx context.Context, vendorID string) error {
	if vendorID == "" {
		return errors.New("vendor ID is required")
	}

	req := UpdateVendorRequest{
		IsActive: boolPtr(true),
	}

	_, err := s.store.UpdateVendor(ctx, vendorID, req)
	if err != nil {
		return fmt.Errorf("failed to activate vendor: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"vendor_id": vendorID,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionActivate, ResourceVendor, &vendorID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// VerifyVendor verifies a vendor account
func (s *Service) VerifyVendor(ctx context.Context, vendorID string) error {
	if vendorID == "" {
		return errors.New("vendor ID is required")
	}

	req := UpdateVendorRequest{
		IsVerified: boolPtr(true),
	}

	_, err := s.store.UpdateVendor(ctx, vendorID, req)
	if err != nil {
		return fmt.Errorf("failed to verify vendor: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"vendor_id": vendorID,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionVerify, ResourceVendor, &vendorID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// Plan management

// GetPlans retrieves a list of plans with pagination and filtering
func (s *Service) GetPlans(ctx context.Context, req PlanListRequest) (PlanListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetPlans(ctx, req)
}

// GetPlan retrieves a specific plan by ID
func (s *Service) GetPlan(ctx context.Context, planID string) (AdminPlan, error) {
	if planID == "" {
		return AdminPlan{}, errors.New("plan ID is required")
	}

	plan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return AdminPlan{}, fmt.Errorf("failed to get plan: %w", err)
	}

	return plan, nil
}

// CreatePlan creates a new subscription plan
func (s *Service) CreatePlan(ctx context.Context, req CreatePlanRequest) (AdminPlan, error) {
	if req.Name == "" {
		return AdminPlan{}, errors.New("plan name is required")
	}
	if req.DisplayName == "" {
		return AdminPlan{}, errors.New("display name is required")
	}
	if req.PricePerMonthCents < 0 {
		return AdminPlan{}, errors.New("price cannot be negative")
	}
	if req.MonthlyConversionsLimit < 0 {
		return AdminPlan{}, errors.New("monthly conversions limit cannot be negative")
	}

	plan, err := s.store.CreatePlan(ctx, req)
	if err != nil {
		return AdminPlan{}, fmt.Errorf("failed to create plan: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"plan_name": req.Name,
		"plan_data": req,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionCreate, ResourcePlan, &plan.ID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return plan, nil
}

// UpdatePlan updates a subscription plan
func (s *Service) UpdatePlan(ctx context.Context, planID string, req UpdatePlanRequest) (AdminPlan, error) {
	if planID == "" {
		return AdminPlan{}, errors.New("plan ID is required")
	}

	// Validate price if provided
	if req.PricePerMonthCents != nil && *req.PricePerMonthCents < 0 {
		return AdminPlan{}, errors.New("price cannot be negative")
	}

	// Validate monthly conversions limit if provided
	if req.MonthlyConversionsLimit != nil && *req.MonthlyConversionsLimit < 0 {
		return AdminPlan{}, errors.New("monthly conversions limit cannot be negative")
	}

	plan, err := s.store.UpdatePlan(ctx, planID, req)
	if err != nil {
		return AdminPlan{}, fmt.Errorf("failed to update plan: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"plan_id": planID,
		"changes": req,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionUpdate, ResourcePlan, &planID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return plan, nil
}

// DeletePlan deletes a subscription plan
func (s *Service) DeletePlan(ctx context.Context, planID string) error {
	if planID == "" {
		return errors.New("plan ID is required")
	}

	err := s.store.DeletePlan(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"plan_id": planID,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionDelete, ResourcePlan, &planID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// Payment management

// GetPayments retrieves a list of payments with pagination and filtering
func (s *Service) GetPayments(ctx context.Context, req PaymentListRequest) (PaymentListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetPayments(ctx, req)
}

// GetPayment retrieves a specific payment by ID
func (s *Service) GetPayment(ctx context.Context, paymentID string) (AdminPayment, error) {
	if paymentID == "" {
		return AdminPayment{}, errors.New("payment ID is required")
	}

	payment, err := s.store.GetPayment(ctx, paymentID)
	if err != nil {
		return AdminPayment{}, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// Conversion management

// GetConversions retrieves a list of conversions with pagination and filtering
func (s *Service) GetConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetConversions(ctx, req)
}

// GetConversion retrieves a specific conversion by ID
func (s *Service) GetConversion(ctx context.Context, conversionID string) (AdminConversion, error) {
	if conversionID == "" {
		return AdminConversion{}, errors.New("conversion ID is required")
	}

	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return AdminConversion{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	return conversion, nil
}

// Image management

// GetImages retrieves a list of images with pagination and filtering
func (s *Service) GetImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetImages(ctx, req)
}

// GetImage retrieves a specific image by ID
func (s *Service) GetImage(ctx context.Context, imageID string) (AdminImage, error) {
	if imageID == "" {
		return AdminImage{}, errors.New("image ID is required")
	}

	image, err := s.store.GetImage(ctx, imageID)
	if err != nil {
		return AdminImage{}, fmt.Errorf("failed to get image: %w", err)
	}

	return image, nil
}

// Audit trail

// GetAuditLogs retrieves a list of audit logs with pagination and filtering
func (s *Service) GetAuditLogs(ctx context.Context, req AuditLogListRequest) (AuditLogListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return s.store.GetAuditLogs(ctx, req)
}

// Quota management

// RevokeUserQuota revokes quota from a user
func (s *Service) RevokeUserQuota(ctx context.Context, req RevokeQuotaRequest) error {
	if req.UserID == "" {
		return errors.New("user ID is required")
	}
	if req.QuotaType == "" {
		return errors.New("quota type is required")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if req.Reason == "" {
		return errors.New("reason is required")
	}

	validQuotaTypes := []string{"free", "paid"}
	valid := false
	for _, qt := range validQuotaTypes {
		if req.QuotaType == qt {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("invalid quota type")
	}

	err := s.store.RevokeUserQuota(ctx, req.UserID, req.QuotaType, req.Amount, req.Reason)
	if err != nil {
		return fmt.Errorf("failed to revoke user quota: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"user_id":    req.UserID,
		"quota_type": req.QuotaType,
		"amount":     req.Amount,
		"reason":     req.Reason,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionRevoke, ResourceQuota, &req.UserID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	// Send notification to user
	notificationData := map[string]interface{}{
		"quota_type": req.QuotaType,
		"amount":     req.Amount,
		"reason":     req.Reason,
		"action":     "quota_revoked",
	}
	if err := s.notifier.SendNotification(ctx, req.UserID, "quota_revoked", notificationData); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to send quota revocation notification: %v\n", err)
	}

	return nil
}

// RevokeVendorQuota revokes quota from a vendor
func (s *Service) RevokeVendorQuota(ctx context.Context, vendorID string, quotaType string, amount int, reason string) error {
	if vendorID == "" {
		return errors.New("vendor ID is required")
	}
	if quotaType == "" {
		return errors.New("quota type is required")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if reason == "" {
		return errors.New("reason is required")
	}

	validQuotaTypes := []string{"free", "paid"}
	valid := false
	for _, qt := range validQuotaTypes {
		if quotaType == qt {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("invalid quota type")
	}

	err := s.store.RevokeVendorQuota(ctx, vendorID, quotaType, amount, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke vendor quota: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"vendor_id":  vendorID,
		"quota_type": quotaType,
		"amount":     amount,
		"reason":     reason,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionRevoke, ResourceQuota, &vendorID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	return nil
}

// RevokeUserPlan revokes a user's subscription plan
func (s *Service) RevokeUserPlan(ctx context.Context, req RevokePlanRequest) error {
	if req.UserID == "" {
		return errors.New("user ID is required")
	}
	if req.Reason == "" {
		return errors.New("reason is required")
	}

	err := s.store.RevokeUserPlan(ctx, req.UserID, req.Reason)
	if err != nil {
		return fmt.Errorf("failed to revoke user plan: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"user_id": req.UserID,
		"reason":  req.Reason,
	}
	if err := s.auditLogger.LogAction(ctx, nil, ActorTypeAdmin, ActionRevoke, ResourcePlan, &req.UserID, metadata); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit action: %v\n", err)
	}

	// Send notification to user
	notificationData := map[string]interface{}{
		"reason": req.Reason,
		"action": "plan_revoked",
	}
	if err := s.notifier.SendNotification(ctx, req.UserID, "plan_revoked", notificationData); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to send plan revocation notification: %v\n", err)
	}

	return nil
}

// Statistics

// GetSystemStats retrieves system-wide statistics
func (s *Service) GetSystemStats(ctx context.Context) (AdminStats, error) {
	stats, err := s.store.GetSystemStats(ctx)
	if err != nil {
		return AdminStats{}, fmt.Errorf("failed to get system stats: %w", err)
	}

	return stats, nil
}

// GetUserStats retrieves user statistics
func (s *Service) GetUserStats(ctx context.Context) (int, int, error) {
	total, active, err := s.store.GetUserStats(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get user stats: %w", err)
	}

	return total, active, nil
}

// GetVendorStats retrieves vendor statistics
func (s *Service) GetVendorStats(ctx context.Context) (int, int, error) {
	total, active, err := s.store.GetVendorStats(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get vendor stats: %w", err)
	}

	return total, active, nil
}

// GetPaymentStats retrieves payment statistics
func (s *Service) GetPaymentStats(ctx context.Context) (int, int64, error) {
	total, revenue, err := s.store.GetPaymentStats(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get payment stats: %w", err)
	}

	return total, revenue, nil
}

// GetConversionStats retrieves conversion statistics
func (s *Service) GetConversionStats(ctx context.Context) (int, int, int, error) {
	total, pending, failed, err := s.store.GetConversionStats(ctx)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get conversion stats: %w", err)
	}

	return total, pending, failed, nil
}

// GetImageStats retrieves image statistics
func (s *Service) GetImageStats(ctx context.Context) (int, error) {
	total, err := s.store.GetImageStats(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get image stats: %w", err)
	}

	return total, nil
}
