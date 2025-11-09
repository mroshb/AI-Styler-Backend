package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-styler/internal/security"
	"ai-styler/internal/sms"
)

// Mock implementations for testing
type mockStore struct {
	users          map[string]User
	verifiedPhones map[string]bool
	otps           map[string]struct {
		code    string
		purpose string
		expires time.Time
	}
}

func newMockStore() *mockStore {
	return &mockStore{
		users:          make(map[string]User),
		verifiedPhones: make(map[string]bool),
		otps: make(map[string]struct {
			code    string
			purpose string
			expires time.Time
		}),
	}
}

func (m *mockStore) CreateOTP(ctx context.Context, phone, purpose string, digits int, ttl time.Duration) (string, time.Time, error) {
	code := "123456"
	exp := time.Now().Add(ttl)
	m.otps[phone] = struct {
		code    string
		purpose string
		expires time.Time
	}{
		code: code, purpose: purpose, expires: exp,
	}
	return code, exp, nil
}

func (m *mockStore) VerifyOTP(ctx context.Context, phone, code, purpose string) (bool, error) {
	otp, ok := m.otps[phone]
	if !ok || otp.purpose != purpose {
		return false, ErrOTPInvalid
	}
	if time.Now().After(otp.expires) {
		return false, ErrOTPExpired
	}
	if otp.code != code {
		return false, ErrOTPInvalid
	}
	delete(m.otps, phone)
	return true, nil
}

func (m *mockStore) MarkPhoneVerified(ctx context.Context, phone string) error {
	m.verifiedPhones[phone] = true
	return nil
}

func (m *mockStore) UserExists(ctx context.Context, phone string) (bool, error) {
	_, ok := m.users[phone]
	return ok, nil
}

func (m *mockStore) IsPhoneVerified(ctx context.Context, phone string) (bool, error) {
	return m.verifiedPhones[phone], nil
}

func (m *mockStore) CreateUser(ctx context.Context, phone, passwordHash, role, displayName, companyName string) (string, error) {
	id := "user-" + phone
	m.users[phone] = User{
		ID: id, Phone: phone, PasswordHash: passwordHash,
		Role: role, Name: displayName, IsPhoneVerified: true,
		IsActive: true, CreatedAt: time.Now(),
	}
	return id, nil
}

func (m *mockStore) GetUserByPhone(ctx context.Context, phone string) (User, error) {
	user, ok := m.users[phone]
	if !ok {
		return User{}, ErrOTPInvalid // reuse error for not found
	}
	return user, nil
}

type mockRateLimiter struct{}

func (m *mockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	return true // Always allow for testing
}

type mockTokenService struct{}

func (m *mockTokenService) IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (string, string, time.Time, error) {
	return "access-token", "refresh-token", time.Now().Add(30 * 24 * time.Hour), nil
}

func (m *mockTokenService) ValidateAccess(ctx context.Context, token string) (TokenClaims, error) {
	return TokenClaims{
		UserID: "test-user", Phone: "+9123456789", Role: "user",
		SessionID: "test-session", ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

func (m *mockTokenService) Rotate(ctx context.Context, refresh string) (string, string, time.Time, error) {
	return "new-access-token", "new-refresh-token", time.Now().Add(30 * 24 * time.Hour), nil
}

func (m *mockTokenService) RevokeSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockTokenService) RevokeAll(ctx context.Context, userID string) error {
	return nil
}

func TestHandler_SendOTP(t *testing.T) {
	handler := NewHandler(newMockStore(), &mockTokenService{}, &mockRateLimiter{}, &sms.MockSMSProvider{})

	tests := []struct {
		name           string
		request        sendOtpReq
		expectedStatus int
	}{
		{
			name: "valid request",
			request: sendOtpReq{
				Phone:   "+9123456789",
				Purpose: "phone_verify",
				Channel: "sms",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid phone",
			request: sendOtpReq{
				Phone:   "invalid",
				Purpose: "phone_verify",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/send-otp", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SendOTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response sendOtpResp
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				if !response.Sent {
					t.Error("Expected sent to be true")
				}
			}
		})
	}
}

func TestHandler_VerifyOTP(t *testing.T) {
	store := newMockStore()
	// Pre-create an OTP
	store.CreateOTP(context.Background(), "+9123456789", "phone_verify", 6, 5*time.Minute)

	handler := NewHandler(store, &mockTokenService{}, &mockRateLimiter{}, &sms.MockSMSProvider{})

	tests := []struct {
		name           string
		request        verifyReq
		expectedStatus int
	}{
		{
			name: "valid verification",
			request: verifyReq{
				Phone:   "+9123456789",
				Code:    "123456",
				Purpose: "phone_verify",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid code",
			request: verifyReq{
				Phone:   "+9123456789",
				Code:    "000000",
				Purpose: "phone_verify",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/verify-otp", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.VerifyOTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_CheckUser(t *testing.T) {
	store := newMockStore()
	store.users["+9123000000"] = User{ID: "existing-user", Phone: "+9123000000"}

	handler := NewHandler(store, &mockTokenService{}, &mockRateLimiter{}, &sms.MockSMSProvider{})

	tests := []struct {
		name           string
		request        checkUserReq
		expectedStatus int
		expectedBody   *checkUserResp
	}{
		{
			name: "user exists",
			request: checkUserReq{
				Phone: "+9123000000",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   &checkUserResp{Registered: true},
		},
		{
			name: "user does not exist",
			request: checkUserReq{
				Phone: "+9123999999",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   &checkUserResp{Registered: false},
		},
		{
			name: "invalid phone",
			request: checkUserReq{
				Phone: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/check-user", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CheckUser(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != nil {
				var resp checkUserResp
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp != *tt.expectedBody {
					t.Fatalf("Expected response %+v, got %+v", *tt.expectedBody, resp)
				}
			}
		})
	}
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		request        registerReq
		expectedStatus int
		setupStore     func() *mockStore
	}{
		{
			name: "valid registration",
			request: registerReq{
				Phone:    "+9123456789",
				Password: "password123456",
				Role:     "user",
			},
			expectedStatus: http.StatusCreated,
			setupStore: func() *mockStore {
				store := newMockStore()
				store.MarkPhoneVerified(context.Background(), "+9123456789")
				return store
			},
		},
		{
			name: "short password",
			request: registerReq{
				Phone:    "+9123456790",
				Password: "short",
				Role:     "user",
			},
			expectedStatus: http.StatusBadRequest,
			setupStore: func() *mockStore {
				store := newMockStore()
				store.MarkPhoneVerified(context.Background(), "+9123456790")
				return store
			},
		},
		{
			name: "invalid role",
			request: registerReq{
				Phone:    "+9123456791",
				Password: "password123456",
				Role:     "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			setupStore: func() *mockStore {
				store := newMockStore()
				store.MarkPhoneVerified(context.Background(), "+9123456791")
				return store
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			handler := NewHandler(store, &mockTokenService{}, &mockRateLimiter{}, &sms.MockSMSProvider{})

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	store := newMockStore()

	// Create a test hasher with the same configuration as the handler
	testHasher := security.NewBCryptHasher(12) // Same cost as handler default
	password := "password123456"
	hashedPassword, err := testHasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Pre-create a user and verify phone
	store.CreateUser(context.Background(), "+9123456789", hashedPassword, "user", "", "")
	store.MarkPhoneVerified(context.Background(), "+9123456789")

	// Create handler with forced bcrypt hasher
	handler := &Handler{
		store:       store,
		tokens:      &mockTokenService{},
		rateLimiter: &mockRateLimiter{},
		sms:         &sms.MockSMSProvider{},
		hasher:      testHasher, // Use the same hasher
	}

	tests := []struct {
		name           string
		request        loginReq
		expectedStatus int
	}{
		{
			name: "valid login",
			request: loginReq{
				Phone:    "+9123456789",
				Password: "password123456",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid password",
			request: loginReq{
				Phone:    "+9123456789",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			request: loginReq{
				Phone:    "+9999999999",
				Password: "password123456",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
