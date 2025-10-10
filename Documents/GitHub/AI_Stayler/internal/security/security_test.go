package security

import (
	"context"
	"testing"
	"time"
)

func TestBCryptHasher(t *testing.T) {
	hasher := NewBCryptHasher(12)

	password := "test-password-123"

	// Test hashing
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	if hash == password {
		t.Fatal("Hash should not equal original password")
	}

	// Test verification
	if !hasher.Verify(password, hash) {
		t.Fatal("Password verification should succeed")
	}

	// Test wrong password
	if hasher.Verify("wrong-password", hash) {
		t.Fatal("Wrong password should not verify")
	}

	// Test algorithm name
	if hasher.GetAlgorithm() != "bcrypt" {
		t.Fatalf("Expected algorithm 'bcrypt', got '%s'", hasher.GetAlgorithm())
	}
}

func TestArgon2Hasher(t *testing.T) {
	hasher := NewArgon2Hasher(65536, 3, 2, 16, 32)

	password := "test-password-123"

	// Test hashing
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	if hash == password {
		t.Fatal("Hash should not equal original password")
	}

	// Test verification
	if !hasher.Verify(password, hash) {
		t.Fatal("Password verification should succeed")
	}

	// Test wrong password
	if hasher.Verify("wrong-password", hash) {
		t.Fatal("Wrong password should not verify")
	}

	// Test algorithm name
	if hasher.GetAlgorithm() != "argon2id" {
		t.Fatalf("Expected algorithm 'argon2id', got '%s'", hasher.GetAlgorithm())
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewInMemoryRateLimiter()

	key := "test-key"
	limit := 3
	window := time.Minute

	// Test initial requests
	for i := 0; i < limit; i++ {
		if !limiter.Allow(key, limit, window) {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Test rate limit exceeded
	if limiter.Allow(key, limit, window) {
		t.Fatal("Request should be rate limited")
	}

	// Test remaining count
	remaining := limiter.GetRemaining(key, limit, window)
	if remaining != 0 {
		t.Fatalf("Expected 0 remaining requests, got %d", remaining)
	}

	// Test reset
	err := limiter.Reset(key)
	if err != nil {
		t.Fatalf("Failed to reset rate limiter: %v", err)
	}

	// Test after reset
	if !limiter.Allow(key, limit, window) {
		t.Fatal("Request should be allowed after reset")
	}
}

func TestImageScanner(t *testing.T) {
	scanner := NewMockImageScanner()

	// Test clean image
	cleanImageData := []byte("clean image data")
	result, err := scanner.ScanImage(cleanImageData, "test.jpg")
	if err != nil {
		t.Fatalf("Failed to scan image: %v", err)
	}

	if !result.IsClean {
		t.Fatal("Clean image should be marked as clean")
	}

	if len(result.Threats) > 0 {
		t.Fatal("Clean image should have no threats")
	}

	if scanner.IsMalicious(result) {
		t.Fatal("Clean image should not be malicious")
	}

	// Test large image (should be flagged)
	largeImageData := make([]byte, 11*1024*1024) // 11MB
	result, err = scanner.ScanImage(largeImageData, "large.jpg")
	if err != nil {
		t.Fatalf("Failed to scan large image: %v", err)
	}

	if result.IsClean {
		t.Fatal("Large image should be flagged")
	}

	if len(result.Threats) == 0 {
		t.Fatal("Large image should have threats")
	}

	if !scanner.IsMalicious(result) {
		t.Fatal("Large image should be malicious")
	}
}

func TestSignedURLGenerator(t *testing.T) {
	generator := NewMockSignedURLGenerator("https://storage.example.com", "secret")

	bucket := "test-bucket"
	key := "test-file.jpg"
	expiration := time.Hour

	// Test URL generation
	url, err := generator.GenerateSignedURL(bucket, key, expiration)
	if err != nil {
		t.Fatalf("Failed to generate signed URL: %v", err)
	}

	if url == "" {
		t.Fatal("Signed URL should not be empty")
	}

	// Test URL verification
	valid, err := generator.VerifySignedURL(url)
	if err != nil {
		t.Fatalf("Failed to verify signed URL: %v", err)
	}

	if !valid {
		t.Fatal("Generated URL should be valid")
	}
}

func TestSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	if !config.RateLimitEnabled {
		t.Fatal("Rate limiting should be enabled by default")
	}

	if config.RateLimitPerIP <= 0 {
		t.Fatal("Rate limit per IP should be positive")
	}

	if config.RateLimitPerUser <= 0 {
		t.Fatal("Rate limit per user should be positive")
	}

	if config.RateLimitWindow <= 0 {
		t.Fatal("Rate limit window should be positive")
	}

	if config.JWTSecret == "" {
		t.Fatal("JWT secret should not be empty")
	}

	if config.JWTExpiration <= 0 {
		t.Fatal("JWT expiration should be positive")
	}

	if !config.CORSEnabled {
		t.Fatal("CORS should be enabled by default")
	}

	if len(config.AllowedOrigins) == 0 {
		t.Fatal("Allowed origins should not be empty")
	}

	if len(config.AllowedMethods) == 0 {
		t.Fatal("Allowed methods should not be empty")
	}

	if len(config.AllowedHeaders) == 0 {
		t.Fatal("Allowed headers should not be empty")
	}

	if !config.SecurityHeadersEnabled {
		t.Fatal("Security headers should be enabled by default")
	}

	if !config.ImageScanEnabled {
		t.Fatal("Image scanning should be enabled by default")
	}

	if !config.SignedURLEnabled {
		t.Fatal("Signed URLs should be enabled by default")
	}

	if config.SignedURLExpiration <= 0 {
		t.Fatal("Signed URL expiration should be positive")
	}
}

func TestSecurityMiddleware(t *testing.T) {
	config := DefaultSecurityConfig()
	middleware := NewSecurityMiddleware(config)

	if middleware.config == nil {
		t.Fatal("Security middleware config should not be nil")
	}

	if middleware.rateLimiter == nil {
		t.Fatal("Rate limiter should not be nil")
	}

	if middleware.jwtSigner == nil {
		t.Fatal("JWT signer should not be nil")
	}

	if middleware.imageScanner == nil {
		t.Fatal("Image scanner should not be nil")
	}

	if middleware.urlGenerator == nil {
		t.Fatal("URL generator should not be nil")
	}
}

func TestTLSConfig(t *testing.T) {
	config := DefaultTLSConfig()

	if config.MinVersion == 0 {
		t.Fatal("Min TLS version should be set")
	}

	if config.MaxVersion == 0 {
		t.Fatal("Max TLS version should be set")
	}

	if len(config.CipherSuites) == 0 {
		t.Fatal("Cipher suites should not be empty")
	}

	if config.HSTSMaxAge <= 0 {
		t.Fatal("HSTS max age should be positive")
	}

	// Test TLS config creation
	tlsConfig, err := config.CreateTLSConfig()
	if err != nil {
		t.Fatalf("Failed to create TLS config: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("TLS config should not be nil")
	}

	if tlsConfig.MinVersion != config.MinVersion {
		t.Fatal("TLS config min version should match")
	}

	if tlsConfig.MaxVersion != config.MaxVersion {
		t.Fatal("TLS config max version should match")
	}
}

func TestSecureTLSConfig(t *testing.T) {
	config := SecureTLSConfig()

	if config.MinVersion != config.MaxVersion {
		t.Fatal("Secure config should use same min and max version")
	}

	if config.MinVersion < 0x0304 { // TLS 1.3
		t.Fatal("Secure config should use TLS 1.3")
	}

	if !config.SessionTicketsDisabled {
		t.Fatal("Secure config should disable session tickets")
	}

	if !config.HSTSPreload {
		t.Fatal("Secure config should enable HSTS preload")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test with no values
	if _, ok := GetUserIDFromContext(ctx); ok {
		t.Fatal("Should not find user ID in empty context")
	}

	if _, ok := GetUserRoleFromContext(ctx); ok {
		t.Fatal("Should not find user role in empty context")
	}

	if _, ok := GetScanResultFromContext(ctx); ok {
		t.Fatal("Should not find scan result in empty context")
	}

	// Test with values
	ctx = context.WithValue(ctx, UserIDKey, "user123")
	ctx = context.WithValue(ctx, UserRoleKey, "admin")
	ctx = context.WithValue(ctx, ScanResultKey, &ScanResult{IsClean: true})

	if userID, ok := GetUserIDFromContext(ctx); !ok || userID != "user123" {
		t.Fatal("Should find correct user ID")
	}

	if userRole, ok := GetUserRoleFromContext(ctx); !ok || userRole != "admin" {
		t.Fatal("Should find correct user role")
	}

	if result, ok := GetScanResultFromContext(ctx); !ok || !result.IsClean {
		t.Fatal("Should find correct scan result")
	}
}
