package auth

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryStore_CreateOTP(t *testing.T) {
	store := NewInMemoryStore()
	phone := "+9123456789"
	purpose := "phone_verify"
	digits := 6
	ttl := 5 * time.Minute

	code, expiresAt, err := store.CreateOTP(context.Background(), phone, purpose, digits, ttl)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(code) != digits {
		t.Errorf("Expected code length %d, got %d", digits, len(code))
	}

	if time.Until(expiresAt) < ttl-time.Second {
		t.Error("Expiration time too short")
	}
}

func TestInMemoryStore_VerifyOTP(t *testing.T) {
	store := NewInMemoryStore()
	phone := "+9123456789"
	purpose := "phone_verify"
	ttl := 5 * time.Minute

	// Create OTP
	code, _, err := store.CreateOTP(context.Background(), phone, purpose, 6, ttl)
	if err != nil {
		t.Fatalf("Failed to create OTP: %v", err)
	}

	// Test valid verification
	valid, err := store.VerifyOTP(context.Background(), phone, code, purpose)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !valid {
		t.Error("Expected verification to be valid")
	}

	// Test invalid code
	valid, err = store.VerifyOTP(context.Background(), phone, "000000", purpose)
	if err == nil {
		t.Error("Expected error for invalid code")
	}
	if valid {
		t.Error("Expected verification to be invalid")
	}
}

func TestInMemoryStore_UserExists(t *testing.T) {
	store := NewInMemoryStore()
	phone := "+9123456789"

	// Test non-existent user
	exists, err := store.UserExists(context.Background(), phone)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if exists {
		t.Error("Expected user to not exist")
	}

	// Create user
	_, err = store.CreateUser(context.Background(), phone, "hash", "user", "", "")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test existing user
	exists, err = store.UserExists(context.Background(), phone)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !exists {
		t.Error("Expected user to exist")
	}
}

func TestInMemoryStore_CreateUser(t *testing.T) {
	store := NewInMemoryStore()
	phone := "+9123456789"
	passwordHash := "hashedpassword"
	role := "user"
	displayName := "Test User"
	companyName := "Test Company"

	userID, err := store.CreateUser(context.Background(), phone, passwordHash, role, displayName, companyName)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if userID == "" {
		t.Error("Expected non-empty user ID")
	}

	// Verify user was created
	user, err := store.GetUserByPhone(context.Background(), phone)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if user.Phone != phone {
		t.Errorf("Expected phone %s, got %s", phone, user.Phone)
	}
	if user.Role != role {
		t.Errorf("Expected role %s, got %s", role, user.Role)
	}
	if !user.IsPhoneVerified {
		t.Error("Expected user to be phone verified")
	}
}

func TestInMemoryRateLimiter_Allow(t *testing.T) {
	limiter := NewInMemoryLimiter()
	key := "test-key"
	limit := 3
	window := time.Minute

	// Should allow first 3 requests
	for i := 0; i < limit; i++ {
		if !limiter.Allow(context.Background(), key, limit, window) {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// Should deny 4th request
	if limiter.Allow(context.Background(), key, limit, window) {
		t.Error("Expected 4th request to be denied")
	}
}

func TestSimpleTokenService_IssueTokens(t *testing.T) {
	service := NewSimpleTokenService()
	userID := "test-user"
	phone := "+9123456789"
	role := "user"
	userAgent := "test-agent"

	access, refresh, expires, err := service.IssueTokens(context.Background(), userID, phone, role, userAgent)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if access == "" {
		t.Error("Expected non-empty access token")
	}
	if refresh == "" {
		t.Error("Expected non-empty refresh token")
	}
	if time.Until(expires) < 24*time.Hour {
		t.Error("Expected refresh token to expire in at least 24 hours")
	}
}

func TestSimpleTokenService_ValidateAccess(t *testing.T) {
	service := NewSimpleTokenService()
	userID := "test-user"
	phone := "+9123456789"
	role := "user"

	// Issue tokens
	access, _, _, err := service.IssueTokens(context.Background(), userID, phone, role, "")
	if err != nil {
		t.Fatalf("Failed to issue tokens: %v", err)
	}

	// Validate access token
	claims, err := service.ValidateAccess(context.Background(), access)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}
}

func TestSimpleTokenService_Rotate(t *testing.T) {
	service := NewSimpleTokenService()
	userID := "test-user"
	phone := "+9123456789"
	role := "user"

	// Issue initial tokens
	_, refresh, _, err := service.IssueTokens(context.Background(), userID, phone, role, "")
	if err != nil {
		t.Fatalf("Failed to issue tokens: %v", err)
	}

	// Rotate tokens
	newAccess, newRefresh, expires, err := service.Rotate(context.Background(), refresh)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if newAccess == "" {
		t.Error("Expected non-empty new access token")
	}
	if newRefresh == "" {
		t.Error("Expected non-empty new refresh token")
	}
	if time.Until(expires) < 24*time.Hour {
		t.Error("Expected new refresh token to expire in at least 24 hours")
	}

	// Old refresh token should be invalid
	_, _, _, err = service.Rotate(context.Background(), refresh)
	if err == nil {
		t.Error("Expected error when rotating with old refresh token")
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"
	hash1 := TestHashPassword(password)
	hash2 := TestHashPassword(password)

	// Different passwords should produce different hashes (bcrypt uses random salt)
	if hash1 == hash2 {
		t.Error("bcrypt should produce different hashes due to random salt")
	}

	// Different password should produce different hash
	hash3 := TestHashPassword("differentpassword")
	if hash1 == hash3 {
		t.Error("Different passwords should produce different hashes")
	}

	// Hash should not be empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestPasswordVerification(t *testing.T) {
	password := "testpassword123"
	hash := TestHashPassword(password)

	// Correct password should verify
	if !TestVerifyPassword(password, hash) {
		t.Error("Correct password should verify")
	}

	// Wrong password should not verify
	if TestVerifyPassword("wrongpassword", hash) {
		t.Error("Wrong password should not verify")
	}
}
