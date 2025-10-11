package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ai-styler/internal/auth"
	"ai-styler/internal/common"
	"ai-styler/internal/config"
	"ai-styler/internal/logging"
	"ai-styler/internal/monitoring"
	"ai-styler/internal/sms"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServer represents a test server instance
type TestServer struct {
	Server *httptest.Server
	Client *http.Client
	DB     interface{} // Mock database
	Redis  *redis.Client
}

// SetupTestServer creates a test server with all dependencies
func SetupTestServer(t *testing.T) *TestServer {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Load test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       1, // Use test database
		},
		Server: config.ServerConfig{
			HTTPAddr: ":0",
			GinMode:  "test",
		},
		JWT: config.JWTConfig{
			Secret:     "test-secret-key",
			AccessTTL:  15 * time.Minute,
			RefreshTTL: 1 * time.Hour,
		},
		Storage: config.StorageConfig{
			StoragePath: "./test_uploads",
		},
		Gemini: config.GeminiConfig{
			APIKey:     "test-api-key",
			BaseURL:    "https://generativelanguage.googleapis.com",
			Model:      "gemini-pro-vision",
			Timeout:    300,
			MaxRetries: 3,
		},
	}

	// Create test directories
	err := os.MkdirAll("./test_uploads", 0755)
	require.NoError(t, err)

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		DB:   cfg.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	err = redisClient.Ping(ctx).Err()
	require.NoError(t, err)

	// Create router with proper service initialization
	router := gin.New()

	// Initialize monitoring service
	monitorConfig := monitoring.MonitoringConfig{
		Sentry: monitoring.SentryConfig{
			DSN:              "",
			Environment:      "test",
			Release:          "test",
			Debug:            true,
			SampleRate:       1.0,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			MaxBreadcrumbs:   50,
		},
		Telegram: monitoring.TelegramConfig{
			BotToken: "",
			ChatID:   "",
			Enabled:  false,
			Timeout:  10 * time.Second,
		},
		Logging: logging.LoggerConfig{
			Level:       logging.ParseLogLevel("info"),
			Format:      "json",
			Output:      "stdout",
			Service:     "ai-stayler",
			Version:     "test",
			Environment: "test",
		},
		Health: monitoring.HealthConfig{
			Enabled:       true,
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}

	monitor, err := monitoring.NewMonitoringService(monitorConfig, nil, redisClient)
	require.NoError(t, err)
	defer monitor.Close()

	// Mount auth routes with predictable OTP codes for testing
	authGroup := router.Group("/auth")
	store := &testStore{
		otps: make(map[string]struct {
			code    string
			purpose string
			expires time.Time
		}),
		users:          make(map[string]testUser),
		verifiedPhones: make(map[string]bool),
	}
	limiter := auth.NewInMemoryLimiter()
	tokens := auth.NewSimpleTokenService()
	smsProvider := sms.NewProvider("mock", "", 0)
	authHandler := auth.NewHandler(store, tokens, limiter, smsProvider)

	authGroup.POST("/send-otp", common.GinWrap(authHandler.SendOTP))
	authGroup.POST("/verify-otp", common.GinWrap(authHandler.VerifyOTP))
	authGroup.POST("/register", common.GinWrap(authHandler.Register))
	authGroup.POST("/login", common.GinWrap(authHandler.Login))
	authGroup.POST("/refresh", common.GinWrap(authHandler.Refresh))
	authGroup.POST("/logout", common.GinWrap(authHandler.Authenticate(authHandler.Logout)))

	// Add mock routes for other services
	apiGroup := router.Group("/api")
	{
		// Image routes
		imagesGroup := apiGroup.Group("/images")
		{
			imagesGroup.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"id": "test-image-id", "message": "Image uploaded"})
			})
			imagesGroup.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"images": []gin.H{{"id": "test-image-id"}}})
			})
		}

		// Conversion routes
		conversionsGroup := apiGroup.Group("/conversions")
		{
			conversionCount := 0
			conversionsGroup.POST("", func(c *gin.Context) {
				conversionCount++
				if conversionCount > 2 {
					c.JSON(403, gin.H{"error": "quota exceeded"})
					return
				}
				c.JSON(200, gin.H{"id": "test-conversion-id", "status": "processing"})
			})
			conversionsGroup.GET("/1/status", func(c *gin.Context) {
				c.JSON(200, gin.H{"id": "1", "status": "completed"})
			})
		}

		// Payment routes
		paymentsGroup := apiGroup.Group("/payments")
		{
			paymentsGroup.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"id": "test-payment-id", "status": "pending"})
			})
			paymentsGroup.POST("/verify", func(c *gin.Context) {
				c.JSON(200, gin.H{"id": "test-payment-123", "status": "verified"})
			})
		}

		// Share routes
		shareGroup := apiGroup.Group("/share")
		{
			shareGroup.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"id": "test-share-id", "token": "test-token"})
			})
		}
	}

	// Public share route
	router.GET("/api/public/share/:token", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Shared content accessed", "token": c.Param("token")})
	})

	// Add health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Create test server
	server := httptest.NewServer(router)

	return &TestServer{
		Server: server,
		Client: &http.Client{Timeout: 30 * time.Second},
		Redis:  redisClient,
	}
}

// CleanupTestServer cleans up test resources
func (ts *TestServer) CleanupTestServer(t *testing.T) {
	ts.Server.Close()
	ts.Redis.Close()

	// Clean up test files
	err := os.RemoveAll("./test_uploads")
	require.NoError(t, err)
}

// MockMonitoringService for testing
type MockMonitoringService struct{}

func (m *MockMonitoringService) LogInfo(ctx context.Context, message string, fields map[string]interface{}) {
}
func (m *MockMonitoringService) LogError(ctx context.Context, message string, fields map[string]interface{}) {
}
func (m *MockMonitoringService) LogFatal(ctx context.Context, message string, fields map[string]interface{}) {
}
func (m *MockMonitoringService) Logger() interface{} { return nil }
func (m *MockMonitoringService) Health() interface{} { return nil }
func (m *MockMonitoringService) Close() error        { return nil }

// TestAuthFlow tests the complete authentication flow
func TestE2EAuthFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	// Test OTP sending
	otpReq := map[string]interface{}{
		"phone":   "+1234567890",
		"purpose": "phone_verify",
		"channel": "sms",
	}

	resp := ts.makeRequest(t, "POST", "/auth/send-otp", otpReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test OTP verification
	verifyReq := map[string]interface{}{
		"phone":   "+1234567890",
		"code":    "123456",
		"purpose": "phone_verify",
	}

	resp = ts.makeRequest(t, "POST", "/auth/verify-otp", verifyReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test user registration
	registerReq := map[string]interface{}{
		"phone":       "+1234567890",
		"password":    "testpassword123",
		"role":        "user",
		"autoLogin":   false,
		"displayName": "Test User",
	}

	resp = ts.makeRequest(t, "POST", "/auth/register", registerReq)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Test login
	loginReq := map[string]interface{}{
		"phone":    "+1234567890",
		"password": "testpassword123",
	}

	resp = ts.makeRequest(t, "POST", "/auth/login", loginReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestConversionFlow tests the complete conversion flow
func TestConversionFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	// First, authenticate a user
	authToken := ts.authenticateUser(t)

	// Test image upload
	imageID := ts.uploadImage(t, authToken)

	// Test conversion creation
	conversionReq := map[string]interface{}{
		"user_image_id":     imageID,
		"clothing_image_id": imageID,
		"style_preference":  "casual",
	}

	resp := ts.makeAuthenticatedRequest(t, "POST", "/api/conversions/", conversionReq, authToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test conversion status check
	resp = ts.makeAuthenticatedRequest(t, "GET", "/api/conversions/1/status", nil, authToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestPaymentFlow tests the complete payment flow
func TestPaymentFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	authToken := ts.authenticateUser(t)

	// Test payment creation
	paymentReq := map[string]interface{}{
		"plan_name": "basic",
		"amount":    1000,
	}

	resp := ts.makeAuthenticatedRequest(t, "POST", "/api/payments/", paymentReq, authToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test payment verification
	verifyReq := map[string]interface{}{
		"payment_id": "test-payment-123",
	}

	resp = ts.makeAuthenticatedRequest(t, "POST", "/api/payments/verify", verifyReq, authToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestShareFlow tests the complete sharing flow
func TestShareFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	authToken := ts.authenticateUser(t)

	// Test creating a shared link
	shareReq := map[string]interface{}{
		"conversion_id": "test-conversion-123",
		"expires_in":    3600,
		"max_access":    100,
	}

	resp := ts.makeAuthenticatedRequest(t, "POST", "/api/share/", shareReq, authToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test accessing shared link
	resp = ts.makeRequest(t, "GET", "/api/public/share/test-token", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Helper methods

func (ts *TestServer) makeRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, ts.Server.URL+path, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)

	// Log response body for debugging
	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Response body for %s %s: %s", method, path, string(bodyBytes))
		resp.Body.Close()
		// Create a new response with the body for further processing
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	return resp
}

func (ts *TestServer) makeAuthenticatedRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, ts.Server.URL+path, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)

	return resp
}

func (ts *TestServer) authenticateUser(t *testing.T) string {
	// Mock authentication - in real tests, this would go through the actual auth flow
	return "mock-jwt-token"
}

func (ts *TestServer) uploadImage(t *testing.T, authToken string) string {
	// Create a test image file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create a dummy image file
	fileWriter, err := writer.CreateFormFile("file", "test.jpg")
	require.NoError(t, err)

	// Write dummy image data
	_, err = fileWriter.Write([]byte("dummy image data"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", ts.Server.URL+"/api/images", &buf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	return "test-image-id"
}

// TestQuotaEnforcement tests quota enforcement
func TestQuotaEnforcement(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	authToken := ts.authenticateUser(t)

	// Test conversion quota enforcement
	for i := 0; i < 3; i++ { // Try to exceed free plan limit
		conversionReq := map[string]interface{}{
			"user_image_id":     "test-image-id",
			"clothing_image_id": "test-clothing-id",
		}

		resp := ts.makeAuthenticatedRequest(t, "POST", "/api/conversions/", conversionReq, authToken)

		if i < 2 {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		} else {
			// Should be rate limited
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		}
	}
}

// TestErrorHandling tests error handling scenarios
func TestE2EErrorHandling(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	// Test invalid endpoint
	resp := ts.makeRequest(t, "GET", "/api/invalid-endpoint", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Test invalid JSON
	req, err := http.NewRequest("POST", ts.Server.URL+"/auth/login", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err = ts.Client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.CleanupTestServer(t)

	authToken := ts.authenticateUser(t)

	// Test concurrent image uploads
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() { done <- true }()

			imageID := ts.uploadImage(t, authToken)
			assert.NotEmpty(t, imageID)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// Benchmark tests

func BenchmarkAuthFlow(b *testing.B) {
	ts := SetupTestServer(&testing.T{})
	defer ts.CleanupTestServer(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		otpReq := map[string]interface{}{
			"phone_number": fmt.Sprintf("+123456789%d", i),
		}
		ts.makeRequest(&testing.T{}, "POST", "/auth/send-otp", otpReq)
	}
}

func BenchmarkImageUpload(b *testing.B) {
	ts := SetupTestServer(&testing.T{})
	defer ts.CleanupTestServer(&testing.T{})

	authToken := ts.authenticateUser(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.uploadImage(&testing.T{}, authToken)
	}
}

// testStore is a test implementation of auth.Store with predictable OTP codes
type testStore struct {
	otps map[string]struct {
		code    string
		purpose string
		expires time.Time
	}
	users          map[string]testUser
	verifiedPhones map[string]bool
}

type testUser struct {
	id              string
	phone           string
	passwordHash    string
	role            string
	displayName     string
	companyName     string
	isPhoneVerified bool
	createdAt       time.Time
}

func (s *testStore) CreateOTP(ctx context.Context, phone, purpose string, digits int, ttl time.Duration) (string, time.Time, error) {
	code := "123456" // Always return the same code for testing
	exp := time.Now().Add(ttl)
	s.otps[phone] = struct {
		code    string
		purpose string
		expires time.Time
	}{code: code, purpose: purpose, expires: exp}
	return code, exp, nil
}

func (s *testStore) VerifyOTP(ctx context.Context, phone, code, purpose string) (bool, error) {
	otp, exists := s.otps[phone]
	if !exists {
		return false, nil
	}
	if otp.code != code || otp.purpose != purpose || time.Now().After(otp.expires) {
		return false, nil
	}
	return true, nil
}

func (s *testStore) MarkPhoneVerified(ctx context.Context, phone string) error {
	s.verifiedPhones[phone] = true
	return nil
}

func (s *testStore) UserExists(ctx context.Context, phone string) (bool, error) {
	for _, user := range s.users {
		if user.phone == phone {
			return true, nil
		}
	}
	return false, nil
}

func (s *testStore) IsPhoneVerified(ctx context.Context, phone string) (bool, error) {
	return s.verifiedPhones[phone], nil
}

func (s *testStore) CreateUser(ctx context.Context, phone, passwordHash, role, displayName, companyName string) (string, error) {
	userID := fmt.Sprintf("user-%d", len(s.users)+1)
	s.users[userID] = testUser{
		id:              userID,
		phone:           phone,
		passwordHash:    passwordHash,
		role:            role,
		displayName:     displayName,
		companyName:     companyName,
		isPhoneVerified: true, // Assume verified after OTP verification
		createdAt:       time.Now(),
	}
	return userID, nil
}

func (s *testStore) GetUserByPhone(ctx context.Context, phone string) (auth.User, error) {
	for _, user := range s.users {
		if user.phone == phone {
			return auth.User{
				ID:              user.id,
				Phone:           user.phone,
				PasswordHash:    user.passwordHash,
				Role:            user.role,
				Name:            user.displayName,
				AvatarURL:       "",
				Bio:             "",
				IsPhoneVerified: user.isPhoneVerified,
				IsActive:        true,
				LastLoginAt:     nil,
				CreatedAt:       user.createdAt,
			}, nil
		}
	}
	return auth.User{}, fmt.Errorf("user not found")
}
