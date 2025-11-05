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


// Helper functions

func stringPtr(s string) *string {
	return &s
}
