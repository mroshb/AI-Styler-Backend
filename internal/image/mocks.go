package image

import (
	"context"
	"fmt"
)

// Mock implementations for testing

// MockFileStorage implements FileStorage interface for testing
type MockFileStorage struct{}

func NewMockFileStorage() *MockFileStorage {
	return &MockFileStorage{}
}

func (m *MockFileStorage) UploadFile(ctx context.Context, data []byte, fileName string, path string) (string, error) {
	return fmt.Sprintf("https://mock-storage.com/%s/%s", path, fileName), nil
}

func (m *MockFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return nil
}

func (m *MockFileStorage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	return []byte("mock file content"), nil
}

func (m *MockFileStorage) GenerateSignedURL(ctx context.Context, filePath string, accessType string, ttl int64) (string, error) {
	return fmt.Sprintf("https://mock-signed-url.com/%s?expires=%d", filePath, ttl), nil
}

func (m *MockFileStorage) ValidateSignedURL(ctx context.Context, signedURL string) (bool, string, error) {
	return true, "mock-file-path", nil
}

// MockImageProcessor implements ImageProcessor interface for testing
type MockImageProcessor struct{}

func NewMockImageProcessor() *MockImageProcessor {
	return &MockImageProcessor{}
}

func (m *MockImageProcessor) ProcessImage(ctx context.Context, data []byte, fileName string) ([]byte, int, int, error) {
	return data, 800, 600, nil
}

func (m *MockImageProcessor) GenerateThumbnail(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error) {
	return []byte("mock thumbnail"), nil
}

func (m *MockImageProcessor) ResizeImage(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error) {
	return data, nil
}

func (m *MockImageProcessor) ValidateImage(ctx context.Context, data []byte, fileName string, mimeType string) error {
	return nil
}

func (m *MockImageProcessor) GetImageDimensions(ctx context.Context, data []byte) (int, int, error) {
	return 800, 600, nil
}

// MockUsageTracker implements UsageTracker interface for testing
type MockUsageTracker struct{}

func NewMockUsageTracker() *MockUsageTracker {
	return &MockUsageTracker{}
}

func (m *MockUsageTracker) RecordUsage(ctx context.Context, imageID string, userID *string, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockUsageTracker) GetUsageHistory(ctx context.Context, imageID string, req ImageUsageHistoryRequest) (ImageUsageHistoryResponse, error) {
	return ImageUsageHistoryResponse{
		History:    []ImageUsageHistory{},
		Total:      0,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 0,
	}, nil
}

func (m *MockUsageTracker) GetUsageStats(ctx context.Context, imageID string) (UsageStats, error) {
	return UsageStats{
		TotalViews:      10,
		TotalDownloads:  5,
		UniqueUsers:     3,
		RecentViews:     2,
		RecentDownloads: 1,
	}, nil
}

// MockCache implements Cache interface for testing
type MockCache struct{}

func NewMockCache() *MockCache {
	return &MockCache{}
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("not found")
}

func (m *MockCache) Set(ctx context.Context, key string, value string, ttl int64) error {
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCache) CacheImage(ctx context.Context, imageID string, image Image) error {
	return nil
}

func (m *MockCache) GetCachedImage(ctx context.Context, imageID string) (Image, error) {
	return Image{}, fmt.Errorf("not found")
}

func (m *MockCache) CacheSignedURL(ctx context.Context, imageID string, url string, ttl int64) error {
	return nil
}

func (m *MockCache) GetCachedSignedURL(ctx context.Context, imageID string) (string, error) {
	return "", fmt.Errorf("not found")
}

// MockNotificationService implements NotificationService interface for testing
type MockNotificationService struct{}

func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{}
}

func (m *MockNotificationService) SendImageUploaded(ctx context.Context, userID *string, vendorID *string, imageID string, imageType ImageType) error {
	return nil
}

func (m *MockNotificationService) SendImageDeleted(ctx context.Context, userID *string, vendorID *string, imageID string, imageType ImageType) error {
	return nil
}

func (m *MockNotificationService) SendQuotaWarning(ctx context.Context, userID *string, vendorID *string, quotaType string, remaining int) error {
	return nil
}

// MockAuditLogger implements AuditLogger interface for testing
type MockAuditLogger struct{}

func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{}
}

func (m *MockAuditLogger) LogImageAction(ctx context.Context, imageID string, userID *string, vendorID *string, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockAuditLogger) LogQuotaAction(ctx context.Context, userID *string, vendorID *string, action string, metadata map[string]interface{}) error {
	return nil
}

// MockRateLimiter implements RateLimiter interface for testing
type MockRateLimiter struct{}

func NewMockRateLimiter() *MockRateLimiter {
	return &MockRateLimiter{}
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string, limit int, window int64) bool {
	return true
}

func (m *MockRateLimiter) GetRemaining(ctx context.Context, key string, limit int, window int64) int {
	return limit
}

func (m *MockRateLimiter) Reset(ctx context.Context, key string) error {
	return nil
}
