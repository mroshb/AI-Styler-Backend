package user

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockStore implements Store for testing
type MockStore struct {
	profiles             map[string]UserProfile
	conversions          map[string]UserConversion
	plans                map[string]UserPlan
	quotaStatus          map[string]QuotaStatus
	canConvertResults    map[string]bool
	conversionHistory    map[string]ConversionHistoryResponse
	updateProfileFunc    func(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error)
	createConversionFunc func(ctx context.Context, userID string, req CreateConversionRequest) (UserConversion, error)
}

func NewMockStore() *MockStore {
	return &MockStore{
		profiles:          make(map[string]UserProfile),
		conversions:       make(map[string]UserConversion),
		plans:             make(map[string]UserPlan),
		quotaStatus:       make(map[string]QuotaStatus),
		canConvertResults: make(map[string]bool),
		conversionHistory: make(map[string]ConversionHistoryResponse),
	}
}

func (m *MockStore) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	profile, exists := m.profiles[userID]
	if !exists {
		return UserProfile{}, errors.New("user not found")
	}
	return profile, nil
}

func (m *MockStore) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, userID, req)
	}

	profile, exists := m.profiles[userID]
	if !exists {
		return UserProfile{}, errors.New("user not found")
	}

	if req.Name != nil {
		profile.Name = req.Name
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = req.AvatarURL
	}
	if req.Bio != nil {
		profile.Bio = req.Bio
	}
	profile.UpdatedAt = time.Now()

	m.profiles[userID] = profile
	return profile, nil
}

func (m *MockStore) CreateConversion(ctx context.Context, userID string, req CreateConversionRequest) (UserConversion, error) {
	if m.createConversionFunc != nil {
		return m.createConversionFunc(ctx, userID, req)
	}

	conversion := UserConversion{
		ID:             "conv-123",
		UserID:         userID,
		ConversionType: req.Type,
		InputFileURL:   req.InputFileURL,
		StyleName:      req.StyleName,
		Status:         ConversionStatusPending,
		CreatedAt:      time.Now(),
	}

	m.conversions[conversion.ID] = conversion
	return conversion, nil
}

func (m *MockStore) GetConversion(ctx context.Context, conversionID string) (UserConversion, error) {
	conversion, exists := m.conversions[conversionID]
	if !exists {
		return UserConversion{}, errors.New("conversion not found")
	}
	return conversion, nil
}

func (m *MockStore) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) (UserConversion, error) {
	conversion, exists := m.conversions[conversionID]
	if !exists {
		return UserConversion{}, errors.New("conversion not found")
	}

	if req.OutputFileURL != nil {
		conversion.OutputFileURL = req.OutputFileURL
	}
	if req.Status != nil {
		conversion.Status = *req.Status
	}
	if req.ErrorMessage != nil {
		conversion.ErrorMessage = req.ErrorMessage
	}
	if req.ProcessingTimeMs != nil {
		conversion.ProcessingTimeMs = req.ProcessingTimeMs
	}
	if req.FileSizeBytes != nil {
		conversion.FileSizeBytes = req.FileSizeBytes
	}

	m.conversions[conversionID] = conversion
	return conversion, nil
}

func (m *MockStore) GetConversionHistory(ctx context.Context, userID string, req ConversionHistoryRequest) (ConversionHistoryResponse, error) {
	history, exists := m.conversionHistory[userID]
	if !exists {
		return ConversionHistoryResponse{
			Conversions: []UserConversion{},
			Total:       0,
			Page:        req.Page,
			PageSize:    req.PageSize,
			TotalPages:  0,
		}, nil
	}
	return history, nil
}

func (m *MockStore) GetUserPlan(ctx context.Context, userID string) (UserPlan, error) {
	plan, exists := m.plans[userID]
	if !exists {
		return UserPlan{
			UserID:                   userID,
			PlanName:                 PlanFree,
			Status:                   PlanStatusActive,
			MonthlyConversionsLimit:  0,
			ConversionsUsedThisMonth: 0,
			PricePerMonthCents:       0,
			AutoRenew:                true,
			CreatedAt:                time.Now(),
			UpdatedAt:                time.Now(),
		}, nil
	}
	return plan, nil
}

func (m *MockStore) CreateUserPlan(ctx context.Context, userID string, planName string) (UserPlan, error) {
	plan := UserPlan{
		ID:                       "plan-123",
		UserID:                   userID,
		PlanName:                 planName,
		Status:                   PlanStatusActive,
		MonthlyConversionsLimit:  10,
		ConversionsUsedThisMonth: 0,
		PricePerMonthCents:       999,
		AutoRenew:                true,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	m.plans[userID] = plan
	return plan, nil
}

func (m *MockStore) UpdateUserPlan(ctx context.Context, planID string, status string) (UserPlan, error) {
	for _, plan := range m.plans {
		if plan.ID == planID {
			plan.Status = status
			plan.UpdatedAt = time.Now()
			m.plans[plan.UserID] = plan
			return plan, nil
		}
	}
	return UserPlan{}, errors.New("plan not found")
}

func (m *MockStore) GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error) {
	status, exists := m.quotaStatus[userID]
	if !exists {
		return QuotaStatus{
			FreeConversionsRemaining:  2,
			PaidConversionsRemaining:  0,
			TotalConversionsRemaining: 2,
			PlanName:                  PlanFree,
			MonthlyLimit:              0,
		}, nil
	}
	return status, nil
}

func (m *MockStore) CanUserConvert(ctx context.Context, userID string, conversionType string) (bool, error) {
	key := userID + ":" + conversionType
	canConvert, exists := m.canConvertResults[key]
	if !exists {
		return true, nil
	}
	return canConvert, nil
}

func (m *MockStore) RecordConversion(ctx context.Context, userID string, conversionType string, inputFileURL string, styleName string) (string, error) {
	return "conv-123", nil
}

func (m *MockStore) GetUserByID(ctx context.Context, userID string) (UserProfile, error) {
	return m.GetProfile(ctx, userID)
}

// Test cases

func TestService_GetProfile(t *testing.T) {
	store := NewMockStore()
	auditLogger := NewMockAuditLogger()
	service := NewService(store, nil, nil, nil, nil, auditLogger)

	// Setup test data
	userID := "user-123"
	expectedProfile := UserProfile{
		ID:                   userID,
		Phone:                "+1234567890",
		Name:                 stringPtr("John Doe"),
		Role:                 "user",
		IsPhoneVerified:      true,
		FreeConversionsUsed:  0,
		FreeConversionsLimit: 2,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	store.profiles[userID] = expectedProfile

	// Test
	profile, err := service.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if profile.ID != expectedProfile.ID {
		t.Errorf("Expected ID %s, got %s", expectedProfile.ID, profile.ID)
	}
	if profile.Name == nil || *profile.Name != "John Doe" {
		t.Errorf("Expected name John Doe, got %v", profile.Name)
	}
}

func TestService_UpdateProfile(t *testing.T) {
	store := NewMockStore()
	auditLogger := NewMockAuditLogger()
	service := NewService(store, nil, nil, nil, nil, auditLogger)

	// Setup test data
	userID := "user-123"
	initialProfile := UserProfile{
		ID:                   userID,
		Phone:                "+1234567890",
		Role:                 "user",
		IsPhoneVerified:      true,
		FreeConversionsUsed:  0,
		FreeConversionsLimit: 2,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	store.profiles[userID] = initialProfile

	// Test update
	updateReq := UpdateProfileRequest{
		Name: stringPtr("Jane Doe"),
		Bio:  stringPtr("Software developer"),
	}

	profile, err := service.UpdateProfile(context.Background(), userID, updateReq)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if profile.Name == nil || *profile.Name != "Jane Doe" {
		t.Errorf("Expected name Jane Doe, got %v", profile.Name)
	}
	if profile.Bio == nil || *profile.Bio != "Software developer" {
		t.Errorf("Expected bio Software developer, got %v", profile.Bio)
	}
}

func TestService_CreateConversion(t *testing.T) {
	store := NewMockStore()
	rateLimiter := NewMockRateLimiter()
	auditLogger := NewMockAuditLogger()
	processor := NewMockConversionProcessor()
	notifier := NewMockNotificationService()
	service := NewService(store, processor, notifier, nil, rateLimiter, auditLogger)

	// Setup test data
	userID := "user-123"
	store.canConvertResults[userID+":free"] = true

	// Test create conversion
	req := CreateConversionRequest{
		InputFileURL: "https://example.com/input.jpg",
		StyleName:    "vintage",
		Type:         ConversionTypeFree,
	}

	conversion, err := service.CreateConversion(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if conversion.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, conversion.UserID)
	}
	if conversion.ConversionType != ConversionTypeFree {
		t.Errorf("Expected conversion type %s, got %s", ConversionTypeFree, conversion.ConversionType)
	}
	if conversion.StyleName != "vintage" {
		t.Errorf("Expected style name vintage, got %s", conversion.StyleName)
	}
}

func TestService_CreateConversion_QuotaExceeded(t *testing.T) {
	store := NewMockStore()
	rateLimiter := NewMockRateLimiter()
	auditLogger := NewMockAuditLogger()
	processor := NewMockConversionProcessor()
	notifier := NewMockNotificationService()
	service := NewService(store, processor, notifier, nil, rateLimiter, auditLogger)

	// Setup test data - user cannot convert
	userID := "user-123"
	store.canConvertResults[userID+":free"] = false

	// Test create conversion
	req := CreateConversionRequest{
		InputFileURL: "https://example.com/input.jpg",
		StyleName:    "vintage",
		Type:         ConversionTypeFree,
	}

	_, err := service.CreateConversion(context.Background(), userID, req)
	if err == nil {
		t.Fatal("Expected error for quota exceeded, got nil")
	}

	if !containsStringInString(err.Error(), "quota exceeded") {
		t.Errorf("Expected quota exceeded error, got %v", err)
	}
}

func TestService_GetQuotaStatus(t *testing.T) {
	store := NewMockStore()
	auditLogger := NewMockAuditLogger()
	service := NewService(store, nil, nil, nil, nil, auditLogger)

	// Setup test data
	userID := "user-123"
	expectedStatus := QuotaStatus{
		FreeConversionsRemaining:  1,
		PaidConversionsRemaining:  5,
		TotalConversionsRemaining: 6,
		PlanName:                  PlanBasic,
		MonthlyLimit:              10,
	}
	store.quotaStatus[userID] = expectedStatus

	// Test
	status, err := service.GetQuotaStatus(context.Background(), userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.FreeConversionsRemaining != expectedStatus.FreeConversionsRemaining {
		t.Errorf("Expected free conversions remaining %d, got %d",
			expectedStatus.FreeConversionsRemaining, status.FreeConversionsRemaining)
	}
	if status.PlanName != expectedStatus.PlanName {
		t.Errorf("Expected plan name %s, got %s", expectedStatus.PlanName, status.PlanName)
	}
}

func TestService_CreateUserPlan(t *testing.T) {
	store := NewMockStore()
	auditLogger := NewMockAuditLogger()
	service := NewService(store, nil, nil, nil, nil, auditLogger)

	// Test create plan
	userID := "user-123"
	planName := PlanBasic

	plan, err := service.CreateUserPlan(context.Background(), userID, planName)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if plan.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, plan.UserID)
	}
	if plan.PlanName != planName {
		t.Errorf("Expected plan name %s, got %s", planName, plan.PlanName)
	}
	if plan.Status != PlanStatusActive {
		t.Errorf("Expected status %s, got %s", PlanStatusActive, plan.Status)
	}
}

func TestService_CreateUserPlan_InvalidPlan(t *testing.T) {
	store := NewMockStore()
	auditLogger := NewMockAuditLogger()
	service := NewService(store, nil, nil, nil, nil, auditLogger)

	// Test invalid plan
	userID := "user-123"
	invalidPlan := "invalid-plan"

	_, err := service.CreateUserPlan(context.Background(), userID, invalidPlan)
	if err == nil {
		t.Fatal("Expected error for invalid plan, got nil")
	}

	if !containsStringInString(err.Error(), "invalid plan name") {
		t.Errorf("Expected invalid plan name error, got %v", err)
	}
}

// Helper functions

func containsStringInString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && containsStringInString(s[1:], substr)
}
