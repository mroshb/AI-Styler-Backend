package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Service provides user management functionality
type Service struct {
	store       Store
	processor   ConversionProcessor
	notifier    NotificationService
	storage     FileStorage
	rateLimiter RateLimiter
	auditLogger AuditLogger
}

// NewService creates a new user service
func NewService(
	store Store,
	processor ConversionProcessor,
	notifier NotificationService,
	storage FileStorage,
	rateLimiter RateLimiter,
	auditLogger AuditLogger,
) *Service {
	return &Service{
		store:       store,
		processor:   processor,
		notifier:    notifier,
		storage:     storage,
		rateLimiter: rateLimiter,
		auditLogger: auditLogger,
	}
}

// GetProfile retrieves a user's profile
func (s *Service) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	profile, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to get profile: %w", err)
	}
	return profile, nil
}

// UpdateProfile updates a user's profile
func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error) {
	// Validate input
	if req.Name != nil && len(*req.Name) > 100 {
		return UserProfile{}, errors.New("name too long")
	}
	if req.Bio != nil && len(*req.Bio) > 500 {
		return UserProfile{}, errors.New("bio too long")
	}
	if req.AvatarURL != nil && len(*req.AvatarURL) > 500 {
		return UserProfile{}, errors.New("avatar URL too long")
	}

	profile, err := s.store.UpdateProfile(ctx, userID, req)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to update profile: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"updated_fields": getUpdatedFields(req),
	}
	_ = s.auditLogger.LogUserAction(ctx, userID, "profile_updated", metadata)

	return profile, nil
}

// GetConversionHistory retrieves a user's conversion history
func (s *Service) GetConversionHistory(ctx context.Context, userID string, req ConversionHistoryRequest) (ConversionHistoryResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	history, err := s.store.GetConversionHistory(ctx, userID, req)
	if err != nil {
		return ConversionHistoryResponse{}, fmt.Errorf("failed to get conversion history: %w", err)
	}

	return history, nil
}

// CreateConversion creates a new conversion
func (s *Service) CreateConversion(ctx context.Context, userID string, req CreateConversionRequest) (UserConversion, error) {
	// Validate input
	if req.InputFileURL == "" {
		return UserConversion{}, errors.New("input file URL is required")
	}
	if req.StyleName == "" {
		return UserConversion{}, errors.New("style name is required")
	}
	if req.Type != ConversionTypeFree && req.Type != ConversionTypePaid {
		return UserConversion{}, errors.New("invalid conversion type")
	}

	// Check rate limiting
	rateLimitKey := fmt.Sprintf("conversion:user:%s", userID)
	if !s.rateLimiter.Allow(ctx, rateLimitKey, 10, time.Hour) {
		return UserConversion{}, errors.New("rate limit exceeded")
	}

	// Check if user can convert
	canConvert, err := s.store.CanUserConvert(ctx, userID, req.Type)
	if err != nil {
		return UserConversion{}, fmt.Errorf("failed to check conversion quota: %w", err)
	}
	if !canConvert {
		return UserConversion{}, errors.New("conversion quota exceeded")
	}

	// Create conversion
	conversion, err := s.store.CreateConversion(ctx, userID, req)
	if err != nil {
		return UserConversion{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"conversion_id":   conversion.ID,
		"conversion_type": req.Type,
		"style_name":      req.StyleName,
	}
	_ = s.auditLogger.LogUserAction(ctx, userID, "conversion_created", metadata)

	// Start processing asynchronously
	go func() {
		ctx := context.Background()
		if err := s.processor.ProcessConversion(ctx, conversion.ID, req.InputFileURL, req.StyleName); err != nil {
			// Update conversion status to failed
			updateReq := UpdateConversionRequest{
				Status:       &[]string{ConversionStatusFailed}[0],
				ErrorMessage: &[]string{err.Error()}[0],
			}
			_, _ = s.store.UpdateConversion(ctx, conversion.ID, updateReq)
			_ = s.notifier.SendConversionFailed(ctx, userID, conversion.ID, err.Error())
		}
	}()

	return conversion, nil
}

// GetConversion retrieves a specific conversion
func (s *Service) GetConversion(ctx context.Context, userID, conversionID string) (UserConversion, error) {
	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return UserConversion{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	// Verify ownership
	if conversion.UserID != userID {
		return UserConversion{}, errors.New("conversion not found")
	}

	return conversion, nil
}

// UpdateConversion updates a conversion (typically used by the processor)
func (s *Service) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) (UserConversion, error) {
	conversion, err := s.store.UpdateConversion(ctx, conversionID, req)
	if err != nil {
		return UserConversion{}, fmt.Errorf("failed to update conversion: %w", err)
	}

	// Send notifications if conversion completed or failed
	if req.Status != nil {
		if *req.Status == ConversionStatusCompleted && req.OutputFileURL != nil {
			_ = s.notifier.SendConversionComplete(ctx, conversion.UserID, conversionID, *req.OutputFileURL)
		} else if *req.Status == ConversionStatusFailed && req.ErrorMessage != nil {
			_ = s.notifier.SendConversionFailed(ctx, conversion.UserID, conversionID, *req.ErrorMessage)
		}
	}

	return conversion, nil
}

// GetQuotaStatus retrieves current quota status for a user
func (s *Service) GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error) {
	status, err := s.store.GetQuotaStatus(ctx, userID)
	if err != nil {
		return QuotaStatus{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	// Send warning if quota is low
	if status.TotalConversionsRemaining <= 1 && status.TotalConversionsRemaining > 0 {
		_ = s.notifier.SendQuotaWarning(ctx, userID, status.TotalConversionsRemaining)
	}

	return status, nil
}

// GetUserPlan retrieves a user's current plan
func (s *Service) GetUserPlan(ctx context.Context, userID string) (UserPlan, error) {
	plan, err := s.store.GetUserPlan(ctx, userID)
	if err != nil {
		return UserPlan{}, fmt.Errorf("failed to get user plan: %w", err)
	}
	return plan, nil
}

// CreateUserPlan creates a new plan for a user
func (s *Service) CreateUserPlan(ctx context.Context, userID string, planName string) (UserPlan, error) {
	// Validate plan name
	validPlans := []string{PlanFree, PlanBasic, PlanPremium, PlanEnterprise}
	if !containsString(validPlans, planName) {
		return UserPlan{}, errors.New("invalid plan name")
	}

	plan, err := s.store.CreateUserPlan(ctx, userID, planName)
	if err != nil {
		return UserPlan{}, fmt.Errorf("failed to create user plan: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"plan_name": planName,
		"plan_id":   plan.ID,
	}
	_ = s.auditLogger.LogUserAction(ctx, userID, "plan_created", metadata)

	return plan, nil
}

// UpdateUserPlan updates a user's plan status
func (s *Service) UpdateUserPlan(ctx context.Context, planID string, status string) (UserPlan, error) {
	// Validate status
	validStatuses := []string{PlanStatusActive, PlanStatusCancelled, PlanStatusExpired, PlanStatusSuspended}
	if !containsString(validStatuses, status) {
		return UserPlan{}, errors.New("invalid plan status")
	}

	plan, err := s.store.UpdateUserPlan(ctx, planID, status)
	if err != nil {
		return UserPlan{}, fmt.Errorf("failed to update user plan: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"plan_id":    planID,
		"new_status": status,
	}
	_ = s.auditLogger.LogUserAction(ctx, plan.UserID, "plan_updated", metadata)

	return plan, nil
}

// Helper functions

func getUpdatedFields(req UpdateProfileRequest) []string {
	var fields []string
	if req.Name != nil {
		fields = append(fields, "name")
	}
	if req.AvatarURL != nil {
		fields = append(fields, "avatar_url")
	}
	if req.Bio != nil {
		fields = append(fields, "bio")
	}
	return fields
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Error handling helpers

func isUniqueConstraintError(err error, constraint string) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505" && pqErr.Constraint == constraint
	}
	return false
}

func isForeignKeyError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23503"
	}
	return false
}

func isNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
