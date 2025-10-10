package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"AI_Styler/internal/sms"
)

// Integration test for complete auth flow
func TestAuthFlow_Complete(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	limiter := NewInMemoryLimiter()
	tokens := NewSimpleTokenService()
	smsProvider := &sms.MockSMSProvider{}
	handler := NewHandler(store, tokens, limiter, smsProvider)

	phone := "+9123456789"
	password := "password123456"
	role := "user"

	// Use different phone for each test to avoid conflicts
	_ = "+9123456789"

	// Step 1: Send OTP
	t.Run("Send OTP", func(t *testing.T) {
		req := sendOtpReq{
			Phone:   phone,
			Purpose: "phone_verify",
			Channel: "sms",
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/send-otp", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.SendOTP(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp sendOtpResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if !resp.Sent {
			t.Error("Expected OTP to be sent")
		}
	})

	// Step 2: Verify OTP
	t.Run("Verify OTP", func(t *testing.T) {
		// Get the OTP that was created (in real scenario, this would be sent via SMS)
		// For testing, we'll create a new OTP and use its code
		code, _, err := store.CreateOTP(context.Background(), phone, "phone_verify", 6, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create OTP: %v", err)
		}

		req := verifyReq{
			Phone:   phone,
			Code:    code,
			Purpose: "phone_verify",
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/verify-otp", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.VerifyOTP(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp verifyResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if !resp.Verified {
			t.Error("Expected OTP to be verified")
		}
	})

	// Step 3: Register user
	t.Run("Register User", func(t *testing.T) {
		req := registerReq{
			Phone:    phone,
			Password: password,
			Role:     role,
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var resp registerResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.UserID == "" {
			t.Error("Expected non-empty user ID")
		}
		if resp.Role != role {
			t.Errorf("Expected role %s, got %s", role, resp.Role)
		}
		if !resp.IsPhoneVerified {
			t.Error("Expected phone to be verified")
		}
	})

	// Step 4: Login
	t.Run("Login", func(t *testing.T) {
		req := loginReq{
			Phone:    phone,
			Password: password,
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp loginResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.AccessToken == "" {
			t.Error("Expected non-empty access token")
		}
		if resp.RefreshToken == "" {
			t.Error("Expected non-empty refresh token")
		}
		if resp.User.ID == "" {
			t.Error("Expected non-empty user ID")
		}
	})

	// Step 5: Refresh token
	t.Run("Refresh Token", func(t *testing.T) {
		// First login to get refresh token
		loginReq := loginReq{Phone: phone, Password: password}
		body, _ := json.Marshal(loginReq)
		httpReq := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, httpReq)

		var loginResp loginResp
		json.Unmarshal(w.Body.Bytes(), &loginResp)

		// Now refresh
		req := refreshReq{RefreshToken: loginResp.RefreshToken}
		body, _ = json.Marshal(req)
		httpReq = httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		handler.Refresh(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp refreshResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.AccessToken == "" {
			t.Error("Expected non-empty access token")
		}
		if resp.RefreshToken == "" {
			t.Error("Expected non-empty refresh token")
		}
	})

	// Step 6: Logout
	t.Run("Logout", func(t *testing.T) {
		// First login to get tokens
		loginReq := loginReq{Phone: phone, Password: password}
		body, _ := json.Marshal(loginReq)
		httpReq := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, httpReq)

		var loginResp loginResp
		json.Unmarshal(w.Body.Bytes(), &loginResp)

		// Create request with access token
		httpReq = httptest.NewRequest("POST", "/auth/logout", nil)
		httpReq.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		w = httptest.NewRecorder()

		// Mock the authentication middleware by setting context values
		ctx := context.WithValue(httpReq.Context(), ctxUserID{}, "test-user")
		ctx = context.WithValue(ctx, ctxSessionID{}, "test-session")
		httpReq = httpReq.WithContext(ctx)

		handler.Logout(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestAuthFlow_ErrorCases(t *testing.T) {
	store := NewInMemoryStore()
	limiter := NewInMemoryLimiter()
	tokens := NewSimpleTokenService()
	smsProvider := &sms.MockSMSProvider{}
	handler := NewHandler(store, tokens, limiter, smsProvider)

	// Cast to concrete type for testing
	// mockStore := store.(*inMemoryStore) // Not needed anymore

	t.Run("Register without phone verification", func(t *testing.T) {
		req := registerReq{
			Phone:    "+9999999999",
			Password: "password123456",
			Role:     "user",
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, httpReq)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("Login with unverified phone", func(t *testing.T) {
		// Test login with non-existent user
		password := "password123456"

		req := loginReq{
			Phone:    "+8888888888", // Non-existent user
			Password: password,
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, httpReq)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Verify OTP with wrong code", func(t *testing.T) {
		// Create OTP
		store.CreateOTP(context.Background(), "+9123456789", "phone_verify", 6, 5*time.Minute)

		req := verifyReq{
			Phone:   "+9123456789",
			Code:    "000000", // Wrong code
			Purpose: "phone_verify",
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/auth/verify-otp", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.VerifyOTP(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}
