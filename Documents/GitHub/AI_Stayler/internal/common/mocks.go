package common

import (
	"context"
	"os"
	"time"
)

// Mock implementations for missing dependencies

// MockConversionProcessor for conversion service
type MockConversionProcessor struct{}

func (m *MockConversionProcessor) ProcessConversion(ctx context.Context, conversionID, inputFileURL, styleName string) error {
	return nil
}

func (m *MockConversionProcessor) CancelProcessing(ctx context.Context, conversionID string) error {
	return nil
}

// MockNotificationService for user service
type MockNotificationService struct{}

func (m *MockNotificationService) SendConversionComplete(ctx context.Context, userID, conversionID, outputFileURL string) error {
	return nil
}

func (m *MockNotificationService) SendConversionFailed(ctx context.Context, userID, conversionID, reason string) error {
	return nil
}

func (m *MockNotificationService) SendQuotaWarning(ctx context.Context, userID string, remainingConversions int) error {
	return nil
}

func (m *MockNotificationService) SendPlanExpired(ctx context.Context, userID string) error {
	return nil
}

// Additional methods for other services
func (m *MockNotificationService) SendConversionCompleted(ctx context.Context, userID, conversionID, outputFileURL string) error {
	return nil
}

func (m *MockNotificationService) SendAlbumCreated(ctx context.Context, vendorID, albumID, albumName string) error {
	return nil
}

func (m *MockNotificationService) SendImageDeleted(ctx context.Context, userID, imageID string) error {
	return nil
}

func (m *MockNotificationService) SendShareAccessed(ctx context.Context, shareID, userID string) error {
	return nil
}

func (m *MockNotificationService) SendEmail(ctx context.Context, to, subject, body string) error {
	return nil
}

func (m *MockNotificationService) SendPaymentSuccess(ctx context.Context, userID, paymentID, planName string) error {
	return nil
}

func (m *MockNotificationService) SendPaymentFailed(ctx context.Context, userID, paymentID, reason string) error {
	return nil
}

func (m *MockNotificationService) SendPlanActivated(ctx context.Context, userID, planName string) error {
	return nil
}

// MockRateLimiter for user service
type MockRateLimiter struct{}

func (m *MockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	return true
}

func (m *MockRateLimiter) AllowWithTTL(ctx context.Context, key string, limit int, ttl int64) bool {
	return true
}

func (m *MockRateLimiter) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	return 1, nil
}

func (m *MockRateLimiter) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return true, nil
}

// MockAuditLogger for user service
type MockAuditLogger struct{}

func (m *MockAuditLogger) LogUserAction(ctx context.Context, userID, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogConversionAction(ctx context.Context, userID, conversionID, action string, metadata map[string]interface{}) error {
	return nil
}

// Additional methods for other services
func (m *MockAuditLogger) LogVendorAction(ctx context.Context, vendorID, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogConversionError(ctx context.Context, conversionID, errorMessage string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogImageAction(ctx context.Context, userID, imageID, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogShareAccessed(ctx context.Context, shareID, userID string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogAction(ctx context.Context, userID, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogPaymentAction(ctx context.Context, userID, action string, metadata map[string]interface{}) error {
	return nil
}

// MockImageProcessor for vendor service
type MockImageProcessor struct{}

func (m *MockImageProcessor) ProcessImage(ctx context.Context, fileData []byte, fileName string) (processedData []byte, width, height int, err error) {
	return fileData, 1920, 1080, nil
}

func (m *MockImageProcessor) GenerateThumbnail(ctx context.Context, fileData []byte, fileName string, maxWidth, maxHeight int) (thumbnailData []byte, err error) {
	// Return a smaller version of the data
	thumbnailSize := len(fileData) / 4
	if thumbnailSize < 100 {
		thumbnailSize = 100
	}
	return make([]byte, thumbnailSize), nil
}

func (m *MockImageProcessor) ValidateImage(ctx context.Context, fileData []byte, fileName string) error {
	return nil
}

// MockCache for vendor service
type MockCache struct{}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCache) CacheImage(ctx context.Context, imageID string, imageData []byte, expiration time.Duration) error {
	return nil
}

// MockImageService for conversion service
type MockImageService struct{}

func (m *MockImageService) GetImage(ctx context.Context, imageID string) (interface{}, error) {
	return map[string]interface{}{"id": imageID, "url": "http://example.com/image.jpg"}, nil
}

func (m *MockImageService) GenerateSignedURL(ctx context.Context, imageID string, accessType string, ttl int64) (string, error) {
	return "http://example.com/signed-url", nil
}

func (m *MockImageService) CreateResultImage(ctx context.Context, conversionID string, imageData []byte) (string, error) {
	return "result-image-123", nil
}

// MockWorkerService for conversion service
type MockWorkerService struct{}

func (m *MockWorkerService) EnqueueConversion(ctx context.Context, conversionID string) error {
	return nil
}

func (m *MockWorkerService) GetJobStatus(ctx context.Context, jobID string) (string, error) {
	return "pending", nil
}

func (m *MockWorkerService) CancelJob(ctx context.Context, jobID string) error {
	return nil
}

// MockMetricsCollector for conversion service
type MockMetricsCollector struct{}

func (m *MockMetricsCollector) RecordConversionStarted(ctx context.Context, userID string) error {
	return nil
}

func (m *MockMetricsCollector) RecordConversionCompleted(ctx context.Context, userID string, duration time.Duration) error {
	return nil
}

func (m *MockMetricsCollector) RecordConversionFailed(ctx context.Context, userID string, reason string) error {
	return nil
}

func (m *MockMetricsCollector) GetConversionMetrics(ctx context.Context, userID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_conversions": 10,
		"successful":        8,
		"failed":            2,
	}, nil
}

// MockUsageTracker for image service
type MockUsageTracker struct{}

func (m *MockUsageTracker) TrackImageUsage(ctx context.Context, userID, imageID string) error {
	return nil
}

func (m *MockUsageTracker) GetImageUsageCount(ctx context.Context, imageID string) (int, error) {
	return 0, nil
}

func (m *MockUsageTracker) GetUserImageUsage(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (m *MockUsageTracker) GetUsageHistory(ctx context.Context, userID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// MockStorageConfig for image service
type MockStorageConfig struct{}

func (m *MockStorageConfig) GetMaxFileSize() int64 {
	return 10 * 1024 * 1024 // 10MB
}

func (m *MockStorageConfig) GetAllowedMimeTypes() []string {
	return []string{"image/jpeg", "image/png", "image/gif"}
}

func (m *MockStorageConfig) GetStoragePath() string {
	return "./uploads"
}

// MockPaymentGateway for payment service
type MockPaymentGateway struct{}

func (m *MockPaymentGateway) CreatePayment(ctx context.Context, amount int64, description string) (string, string, error) {
	return "payment-123", "http://example.com/payment-url", nil
}

func (m *MockPaymentGateway) VerifyPayment(ctx context.Context, paymentID string) (bool, error) {
	return true, nil
}

func (m *MockPaymentGateway) GetPaymentStatus(ctx context.Context, paymentID string) (string, error) {
	return "completed", nil
}

// MockQuotaService for payment service
type MockQuotaService struct{}

func (m *MockQuotaService) UpdateUserQuota(ctx context.Context, userID string, planName string) error {
	return nil
}

func (m *MockQuotaService) ResetMonthlyQuota(ctx context.Context, userID string) error {
	return nil
}

func (m *MockQuotaService) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{"remaining": 10, "total": 10}, nil
}

// MockPaymentConfigService for payment service
type MockPaymentConfigService struct{}

func (m *MockPaymentConfigService) GetPlans() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"name":        "basic",
			"price":       1000,
			"conversions": 10,
		},
	}
}

func (m *MockPaymentConfigService) GetPlanByName(name string) (interface{}, error) {
	return map[string]interface{}{
		"name":        name,
		"price":       1000,
		"conversions": 10,
	}, nil
}

// MockConversionService for share service
type MockConversionService struct{}

func (m *MockConversionService) GetConversion(ctx context.Context, conversionID, userID string) (interface{}, error) {
	return map[string]interface{}{
		"id":              conversionID,
		"user_id":         userID,
		"status":          "completed",
		"result_image_id": "result-123",
	}, nil
}

func (m *MockConversionService) ValidateConversionOwnership(ctx context.Context, conversionID, userID string) error {
	return nil
}

// MockShareImageService for share service
type MockShareImageService struct{}

func (m *MockShareImageService) GetImage(ctx context.Context, imageID string) (interface{}, error) {
	return map[string]interface{}{
		"id":           imageID,
		"original_url": "http://example.com/image.jpg",
	}, nil
}

func (m *MockShareImageService) GenerateSignedURL(ctx context.Context, imageID string, accessType string, ttl int64) (string, error) {
	return "http://example.com/signed-url", nil
}

// MockMetricsCollector for share service
type MockShareMetricsCollector struct{}

func (m *MockShareMetricsCollector) RecordShareCreated(ctx context.Context, userID, conversionID string) error {
	return nil
}

func (m *MockShareMetricsCollector) RecordShareAccessed(ctx context.Context, shareID string, success bool) error {
	return nil
}

func (m *MockShareMetricsCollector) RecordShareExpired(ctx context.Context, shareID string) error {
	return nil
}

// Mock stores for testing

// MockUserStore implements user.Store interface
type MockUserStore struct{}

func (m *MockUserStore) GetProfile(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) UpdateProfile(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) CreateConversion(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) GetConversion(ctx context.Context, conversionID string) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) UpdateConversion(ctx context.Context, conversionID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) GetConversionHistory(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) GetPlan(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) UpdatePlan(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) GetUsage(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

func (m *MockUserStore) IncrementUsage(ctx context.Context, userID string, usageType string) error {
	return nil
}

func (m *MockUserStore) ResetUsage(ctx context.Context, userID string) error {
	return nil
}

// MockVendorStore implements vendor.Store interface
type MockVendorStore struct{}

func (m *MockVendorStore) GetProfile(ctx context.Context, vendorID string) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) UpdateProfile(ctx context.Context, vendorID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) CreateAlbum(ctx context.Context, vendorID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) GetAlbum(ctx context.Context, albumID string) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) UpdateAlbum(ctx context.Context, albumID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) DeleteAlbum(ctx context.Context, albumID string) error {
	return nil
}

func (m *MockVendorStore) ListAlbums(ctx context.Context, vendorID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) UploadImage(ctx context.Context, vendorID, albumID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) GetImage(ctx context.Context, imageID string) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) UpdateImage(ctx context.Context, imageID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockVendorStore) DeleteImage(ctx context.Context, imageID string) error {
	return nil
}

func (m *MockVendorStore) ListImages(ctx context.Context, albumID string, req interface{}) (interface{}, error) {
	return nil, nil
}

// MockConversionStore implements conversion.Store interface
type MockConversionStore struct{}

func (m *MockConversionStore) CreateConversion(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockConversionStore) GetConversion(ctx context.Context, conversionID string) (interface{}, error) {
	return nil, nil
}

func (m *MockConversionStore) UpdateConversion(ctx context.Context, conversionID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockConversionStore) DeleteConversion(ctx context.Context, conversionID string) error {
	return nil
}

func (m *MockConversionStore) ListConversions(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockConversionStore) GetConversionStats(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

// MockImageStore implements image.Store interface
type MockImageStore struct{}

func (m *MockImageStore) CreateImage(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockImageStore) GetImage(ctx context.Context, imageID string) (interface{}, error) {
	return nil, nil
}

func (m *MockImageStore) UpdateImage(ctx context.Context, imageID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockImageStore) DeleteImage(ctx context.Context, imageID string) error {
	return nil
}

func (m *MockImageStore) ListImages(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockImageStore) GetImageStats(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

// MockPaymentStore implements payment.Store interface
type MockPaymentStore struct{}

func (m *MockPaymentStore) CreatePayment(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockPaymentStore) GetPayment(ctx context.Context, paymentID string) (interface{}, error) {
	return nil, nil
}

func (m *MockPaymentStore) UpdatePayment(ctx context.Context, paymentID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockPaymentStore) ListPayments(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockPaymentStore) GetPaymentHistory(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockPaymentStore) GetPaymentStats(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}

// MockShareStore implements share.Store interface
type MockShareStore struct{}

func (m *MockShareStore) CreateSharedLink(ctx context.Context, conversionID, userID, shareToken, signedURL string, expiresAt time.Time, maxAccessCount *int) (string, error) {
	return "share-123", nil
}

func (m *MockShareStore) GetSharedLink(ctx context.Context, shareID string) (interface{}, error) {
	return nil, nil
}

func (m *MockShareStore) GetSharedLinkByToken(ctx context.Context, shareToken string) (interface{}, error) {
	return nil, nil
}

func (m *MockShareStore) UpdateSharedLink(ctx context.Context, shareID string, updates map[string]interface{}) error {
	return nil
}

func (m *MockShareStore) DeactivateSharedLink(ctx context.Context, shareID, userID string) error {
	return nil
}

func (m *MockShareStore) ListUserSharedLinks(ctx context.Context, userID string, limit, offset int) (interface{}, error) {
	return nil, nil
}

func (m *MockShareStore) LogSharedLinkAccess(ctx context.Context, sharedLinkID string, req interface{}, success bool, errorMessage string) error {
	return nil
}

func (m *MockShareStore) GetSharedLinkStats(ctx context.Context, userID, conversionID string) (interface{}, error) {
	return nil, nil
}

func (m *MockShareStore) CleanupExpiredLinks(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *MockShareStore) ValidateSharedLinkAccess(ctx context.Context, shareToken string) (bool, error) {
	return true, nil
}

func (m *MockShareStore) GetSharedLinkDetails(ctx context.Context, shareToken string) (interface{}, error) {
	return nil, nil
}

func (m *MockShareStore) GetSharedLinkAccessLogs(ctx context.Context, shareID string, limit, offset int) (interface{}, error) {
	return nil, nil
}

func (m *MockShareStore) GetPopularSharedLinks(ctx context.Context, limit int) (interface{}, error) {
	return nil, nil
}

// MockMonitoringService implements monitoring interface
type MockMonitoringService struct{}

func (m *MockMonitoringService) RecordMetric(name string, value float64, tags map[string]string) error {
	return nil
}

func (m *MockMonitoringService) IncrementCounter(name string, tags map[string]string) error {
	return nil
}

func (m *MockMonitoringService) RecordHistogram(name string, value float64, tags map[string]string) error {
	return nil
}

func (m *MockMonitoringService) RecordTiming(name string, duration time.Duration, tags map[string]string) error {
	return nil
}

func (m *MockMonitoringService) SetGauge(name string, value float64, tags map[string]string) error {
	return nil
}

func (m *MockMonitoringService) HealthCheck() error {
	return nil
}

func (m *MockMonitoringService) GetMetrics() (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

// MockUserService for payment service
type MockUserService struct{}

func (m *MockUserService) GetUserPlan(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{
		"plan_id": "basic",
		"status":  "active",
	}, nil
}

func (m *MockUserService) UpdateUserPlan(ctx context.Context, planID string, status string) (interface{}, error) {
	return map[string]interface{}{
		"plan_id": planID,
		"status":  status,
	}, nil
}

func (m *MockUserService) CreateUserPlan(ctx context.Context, userID string, planName string) (interface{}, error) {
	return map[string]interface{}{
		"user_id":   userID,
		"plan_name": planName,
		"status":    "active",
	}, nil
}

// MockFileStorage for user service
type MockFileStorage struct{}

func (m *MockFileStorage) UploadFile(ctx context.Context, fileData []byte, fileName string) (string, error) {
	return "https://example.com/files/" + fileName, nil
}

func (m *MockFileStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	return "https://example.com/files/" + filePath, nil
}

func (m *MockFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return nil
}

// MockVendorFileStorage for vendor service
type MockVendorFileStorage struct{}

func (m *MockVendorFileStorage) UploadFile(ctx context.Context, fileData []byte, fileName string, folder string) (string, error) {
	return "https://example.com/vendor/" + folder + "/" + fileName, nil
}

func (m *MockVendorFileStorage) UploadImage(ctx context.Context, fileData []byte, fileName string, folder string) (originalURL, thumbnailURL string, err error) {
	originalURL = "https://example.com/vendor/" + folder + "/" + fileName
	thumbnailURL = originalURL + "_thumb"
	return originalURL, thumbnailURL, nil
}

func (m *MockVendorFileStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	return "https://example.com/vendor/" + filePath, nil
}

func (m *MockVendorFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return nil
}

func (m *MockVendorFileStorage) GenerateThumbnail(ctx context.Context, originalURL string, width, height int) (string, error) {
	return originalURL + "_thumb", nil
}

// MockImageFileStorage for image service
type MockImageFileStorage struct{}

func (m *MockImageFileStorage) UploadFile(ctx context.Context, fileData []byte, fileName string) (string, error) {
	return "https://example.com/images/" + fileName, nil
}

func (m *MockImageFileStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	return "https://example.com/images/" + filePath, nil
}

func (m *MockImageFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return nil
}

func (m *MockImageFileStorage) GenerateSignedURL(ctx context.Context, filePath, accessType string, ttl int64) (string, error) {
	return "https://example.com/signed/" + filePath, nil
}

func (m *MockImageFileStorage) GetFile(ctx context.Context, filePath string) (*os.File, error) {
	return os.Open(filePath)
}
