package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"ai-styler/internal/auth"
	"ai-styler/internal/common"
	"ai-styler/internal/config"
	"ai-styler/internal/monitoring"
	"ai-styler/internal/security"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuite provides comprehensive testing utilities
type TestSuite struct {
	DB           *sql.DB
	RedisClient  *redis.Client
	Config       *config.Config
	AuthStore    auth.Store
	TokenService auth.TokenService
	RateLimiter  auth.RateLimiter
}

// SetupTestSuite sets up a complete test suite
func SetupTestSuite(t *testing.T) *TestSuite {
	// Load test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Override with test settings
	cfg.Database.Name = "ai-styler_test"
	cfg.Redis.DB = 1 // Use different Redis DB for tests
	cfg.Monitoring.Environment = "testing"

	// Connect to test database
	db, err := sql.Open("postgres", buildTestDSN(cfg))
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	// Connect to test Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	require.NoError(t, redisClient.Ping(context.Background()).Err())

	// Initialize stores and services
	authStore := auth.NewPostgresStore(db)
	rateLimiter := auth.NewInMemoryLimiter()
	tokenService := auth.NewSimpleTokenService()

	return &TestSuite{
		DB:           db,
		RedisClient:  redisClient,
		Config:       cfg,
		AuthStore:    authStore,
		TokenService: tokenService,
		RateLimiter:  rateLimiter,
	}
}

// Cleanup cleans up test resources
func (ts *TestSuite) Cleanup(t *testing.T) {
	// Clean up test data
	ts.cleanupTestData(t)

	// Close connections
	ts.DB.Close()
	ts.RedisClient.Close()
}

// cleanupTestData removes all test data
func (ts *TestSuite) cleanupTestData(t *testing.T) {
	// Clean up in reverse order of dependencies
	tables := []string{
		"sessions",
		"otps",
		"audit_logs",
		"user_conversions",
		"images",
		"albums",
		"payments",
		"rate_limits",
		"users",
		"vendors",
		"payment_plans",
	}

	for _, table := range tables {
		_, err := ts.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Failed to clean up table %s: %v", table, err)
		}
	}

	// Clean up Redis
	ts.RedisClient.FlushDB(context.Background())
}

// CreateTestUser creates a test user
func (ts *TestSuite) CreateTestUser(t *testing.T, phone, password, role string) *auth.User {
	hasher := security.NewBCryptHasher(4) // Lower cost for faster tests
	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	_, err = ts.AuthStore.CreateUser(context.Background(), phone, hash, role, "Test User", "")
	require.NoError(t, err)

	user, err := ts.AuthStore.GetUserByPhone(context.Background(), phone)
	require.NoError(t, err)

	return &user
}

// CreateTestSession creates a test session
func (ts *TestSuite) CreateTestSession(t *testing.T, userID string) (string, string) {
	accessToken, refreshToken, _, err := ts.TokenService.IssueTokens(
		context.Background(),
		userID,
		"+1234567890",
		"user",
		"test-agent",
	)
	require.NoError(t, err)

	return accessToken, refreshToken
}

// buildTestDSN builds test database connection string
func buildTestDSN(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}

// TestAuthFlow tests the complete authentication flow
func TestComprehensiveAuthFlow(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.Cleanup(t)

	ctx := context.Background()
	phone := "+1234567890"
	password := "TestPassword123!"

	// Test OTP sending
	code, _, err := ts.AuthStore.CreateOTP(ctx, phone, "phone_verify", 6, 5*time.Minute)
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// Test OTP verification
	verified, err := ts.AuthStore.VerifyOTP(ctx, phone, code, "phone_verify")
	require.NoError(t, err)
	assert.True(t, verified)

	// Test phone verification marking
	err = ts.AuthStore.MarkPhoneVerified(ctx, phone)
	require.NoError(t, err)

	// Test user creation
	user := ts.CreateTestUser(t, phone, password, "user")
	assert.Equal(t, phone, user.Phone)
	assert.Equal(t, "user", user.Role)
	assert.True(t, user.IsPhoneVerified)

	// Test token issuance
	accessToken, refreshToken, _, err := ts.TokenService.IssueTokens(
		ctx,
		user.ID,
		phone,
		user.Role,
		"test-agent",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// Test token validation
	claims, err := ts.TokenService.ValidateAccess(ctx, accessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, phone, claims.Phone)
	assert.Equal(t, user.Role, claims.Role)

	// Test token rotation
	newAccessToken, newRefreshToken, _, err := ts.TokenService.Rotate(ctx, refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEmpty(t, newRefreshToken)

	// Test new token validation
	newClaims, err := ts.TokenService.ValidateAccess(ctx, newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, newClaims.UserID)
}

// TestRateLimiting tests rate limiting functionality
func TestRateLimiting(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.Cleanup(t)

	ctx := context.Background()
	key := "test_rate_limit"
	limit := 3
	window := time.Minute

	// Test initial allowance
	for i := 0; i < limit; i++ {
		allowed := ts.RateLimiter.Allow(ctx, key, limit, window)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// Test rate limit exceeded
	allowed := ts.RateLimiter.Allow(ctx, key, limit, window)
	assert.False(t, allowed, "Request should be rate limited")
}

// TestSecurityFeatures tests security features
func TestSecurityFeatures(t *testing.T) {
	// Test password hashing
	hasher := security.NewBCryptHasher(4) // Lower cost for faster tests
	password := "TestPassword123!"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Test password verification
	valid := hasher.Verify(password, hash)
	assert.True(t, valid)

	// Test invalid password
	invalid := hasher.Verify("WrongPassword", hash)
	assert.False(t, invalid)

	// Test Argon2 hashing
	argon2Hasher := security.NewArgon2Hasher(65536, 3, 2, 16, 32)
	argon2Hash, err := argon2Hasher.Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, argon2Hash)
	assert.NotEqual(t, password, argon2Hash)

	// Test Argon2 verification
	argon2Valid := argon2Hasher.Verify(password, argon2Hash)
	assert.True(t, argon2Valid)

	// Test JWT signing
	jwtSigner := security.NewProductionJWTSigner("test-secret-key", "test-issuer")
	userID := "test-user-id"
	sessionID := "test-session-id"
	role := "user"
	phone := "+1234567890"
	expiresAt := time.Now().Add(time.Hour)

	accessToken, err := jwtSigner.Sign(userID, sessionID, role, phone, expiresAt)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	// Test JWT verification
	claims, err := jwtSigner.Verify(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, phone, claims.Phone)
}

// TestErrorHandling tests error handling
func TestComprehensiveErrorHandling(t *testing.T) {
	// Test validation errors
	validationErrors := common.ValidationErrors{}
	validationErrors.Add("phone", "Invalid phone format", "+1234567890")
	validationErrors.Add("password", "Password too short", "123")

	assert.True(t, validationErrors.HasErrors())
	assert.Len(t, validationErrors.Errors, 2)
	assert.Equal(t, "phone", validationErrors.Errors[0].Field)
	assert.Equal(t, "Invalid phone format", validationErrors.Errors[0].Message)

	// Test business error
	businessError := common.NewHTTPBusinessError("INVALID_INPUT", "Invalid input provided", map[string]interface{}{
		"field": "phone",
	})
	assert.Equal(t, "INVALID_INPUT", businessError.Code)
	assert.Equal(t, "Invalid input provided", businessError.Message)

	// Test system error
	systemError := common.NewHTTPSystemError("DATABASE_ERROR", "Database connection failed", map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	})
	assert.Equal(t, "DATABASE_ERROR", systemError.Code)
	assert.Equal(t, "Database connection failed", systemError.Message)
}

// TestPerformanceMonitoring tests performance monitoring
func TestPerformanceMonitoring(t *testing.T) {
	perfMonitor := monitoring.NewPerformanceMonitor("test-service")
	ctx := context.Background()

	// Test request monitoring
	err := perfMonitor.MonitorRequest(ctx, "GET", "/test", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	})
	require.NoError(t, err)

	// Test database query monitoring
	err = perfMonitor.MonitorDatabaseQuery(ctx, "SELECT * FROM users", func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond) // Simulate query
		return nil
	})
	require.NoError(t, err)

	// Test external API monitoring
	err = perfMonitor.MonitorExternalAPI(ctx, "test-service", "/api/test", func(ctx context.Context) error {
		time.Sleep(15 * time.Millisecond) // Simulate API call
		return nil
	})
	require.NoError(t, err)

	// Test metrics collection
	metrics := perfMonitor.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "counters")
	assert.Contains(t, metrics, "histograms")
	assert.Contains(t, metrics, "gauges")
}

// BenchmarkPasswordHashing benchmarks password hashing performance
func BenchmarkPasswordHashing(b *testing.B) {
	hasher := security.NewBCryptHasher(4) // Lower cost for faster benchmarks
	password := "TestPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash, err := hasher.Hash(password)
		if err != nil {
			b.Fatal(err)
		}
		_ = hash
	}
}

// BenchmarkPasswordVerification benchmarks password verification performance
func BenchmarkPasswordVerification(b *testing.B) {
	hasher := security.NewBCryptHasher(4)
	password := "TestPassword123!"
	hash, err := hasher.Hash(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid := hasher.Verify(password, hash)
		if !valid {
			b.Fatal("Password verification failed")
		}
	}
}

// BenchmarkJWTSigning benchmarks JWT signing performance
func BenchmarkJWTSigning(b *testing.B) {
	jwtSigner := security.NewProductionJWTSigner("test-secret-key", "test-issuer")
	userID := "test-user-id"
	sessionID := "test-session-id"
	role := "user"
	phone := "+1234567890"
	expiresAt := time.Now().Add(time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, err := jwtSigner.Sign(userID, sessionID, role, phone, expiresAt)
		if err != nil {
			b.Fatal(err)
		}
		_ = token
	}
}

// BenchmarkJWTVerification benchmarks JWT verification performance
func BenchmarkJWTVerification(b *testing.B) {
	jwtSigner := security.NewProductionJWTSigner("test-secret-key", "test-issuer")
	userID := "test-user-id"
	sessionID := "test-session-id"
	role := "user"
	phone := "+1234567890"
	expiresAt := time.Now().Add(time.Hour)

	token, err := jwtSigner.Sign(userID, sessionID, role, phone, expiresAt)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		claims, err := jwtSigner.Verify(token)
		if err != nil {
			b.Fatal(err)
		}
		_ = claims
	}
}
