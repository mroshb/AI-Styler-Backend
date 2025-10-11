package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-styler/internal/common"
)

func TestHandler_GetProfile(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

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

	// Create request
	req := httptest.NewRequest("GET", "/profile", nil)
	req = req.WithContext(common.SetUserIDInContext(req.Context(), userID))
	w := httptest.NewRecorder()

	// Execute
	handler.GetProfile(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response UserProfile
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != expectedProfile.ID {
		t.Errorf("Expected ID %s, got %s", expectedProfile.ID, response.ID)
	}
}

func TestHandler_UpdateProfile(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

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

	// Create request
	updateReq := UpdateProfileRequest{
		Name: stringPtr("Jane Doe"),
		Bio:  stringPtr("Software developer"),
	}
	reqBody, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(common.SetUserIDInContext(req.Context(), userID))
	w := httptest.NewRecorder()

	// Execute
	handler.UpdateProfile(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response UserProfile
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Name == nil || *response.Name != "Jane Doe" {
		t.Errorf("Expected name Jane Doe, got %v", response.Name)
	}
}

func TestHandler_GetProfile_Unauthorized(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

	// Create request without user ID in context
	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetProfile(w, req)

	// Verify response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_CreateConversion(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

	// Setup test data
	userID := "user-123"
	store.canConvertResults[userID+":free"] = true

	// Create request
	createReq := CreateConversionRequest{
		InputFileURL: "https://example.com/input.jpg",
		StyleName:    "vintage",
		Type:         ConversionTypeFree,
	}
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/conversions", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(common.SetUserIDInContext(req.Context(), userID))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateConversion(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response UserConversion
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, response.UserID)
	}
	if response.ConversionType != ConversionTypeFree {
		t.Errorf("Expected conversion type %s, got %s", ConversionTypeFree, response.ConversionType)
	}
}

func TestHandler_CreateConversion_QuotaExceeded(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

	// Setup test data - user cannot convert
	userID := "user-123"
	store.canConvertResults[userID+":free"] = false

	// Create request
	createReq := CreateConversionRequest{
		InputFileURL: "https://example.com/input.jpg",
		StyleName:    "vintage",
		Type:         ConversionTypeFree,
	}
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/conversions", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(common.SetUserIDInContext(req.Context(), userID))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateConversion(w, req)

	// Verify response
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestHandler_GetConversionHistory(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

	// Setup test data
	userID := "user-123"
	expectedHistory := ConversionHistoryResponse{
		Conversions: []UserConversion{
			{
				ID:             "conv-1",
				UserID:         userID,
				ConversionType: ConversionTypeFree,
				InputFileURL:   "https://example.com/input1.jpg",
				StyleName:      "vintage",
				Status:         ConversionStatusCompleted,
				CreatedAt:      time.Now(),
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
	store.conversionHistory[userID] = expectedHistory

	// Create request
	req := httptest.NewRequest("GET", "/conversions", nil)
	req = req.WithContext(common.SetUserIDInContext(req.Context(), userID))
	w := httptest.NewRecorder()

	// Execute
	handler.GetConversionHistory(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ConversionHistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Conversions) != 1 {
		t.Errorf("Expected 1 conversion, got %d", len(response.Conversions))
	}
	if response.Total != 1 {
		t.Errorf("Expected total 1, got %d", response.Total)
	}
}

func TestHandler_GetQuotaStatus(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

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

	// Create request
	req := httptest.NewRequest("GET", "/quota", nil)
	req = req.WithContext(common.SetUserIDInContext(req.Context(), userID))
	w := httptest.NewRecorder()

	// Execute
	handler.GetQuotaStatus(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response QuotaStatus
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.FreeConversionsRemaining != expectedStatus.FreeConversionsRemaining {
		t.Errorf("Expected free conversions remaining %d, got %d",
			expectedStatus.FreeConversionsRemaining, response.FreeConversionsRemaining)
	}
}

func TestHandler_CreateUserPlan(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

	// Create request
	createReq := map[string]string{
		"planName": PlanBasic,
	}
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/plan", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(common.SetUserIDInContext(req.Context(), "user-123"))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateUserPlan(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response UserPlan
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.PlanName != PlanBasic {
		t.Errorf("Expected plan name %s, got %s", PlanBasic, response.PlanName)
	}
}

func TestHandler_CreateUserPlan_InvalidPlan(t *testing.T) {
	store := NewMockStore()
	_, handler := WireUserServiceWithMocks(store)

	// Create request with invalid plan
	createReq := map[string]string{
		"planName": "invalid-plan",
	}
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/plan", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(common.SetUserIDInContext(req.Context(), "user-123"))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateUserPlan(w, req)

	// Verify response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}
