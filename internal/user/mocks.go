package user

import (
	"context"
	"time"
)

// MockConversionProcessor implements ConversionProcessor for testing
type MockConversionProcessor struct {
	processFunc func(ctx context.Context, conversionID string, inputFileURL string, styleName string) (string, error)
}

// NewMockConversionProcessor creates a new mock conversion processor
func NewMockConversionProcessor() *MockConversionProcessor {
	return &MockConversionProcessor{
		processFunc: func(ctx context.Context, conversionID string, inputFileURL string, styleName string) (string, error) {
			// Simulate processing delay
			time.Sleep(100 * time.Millisecond)
			// Return a mock output URL
			return "https://example.com/output/" + conversionID + ".jpg", nil
		},
	}
}

// ProcessConversion simulates conversion processing
func (m *MockConversionProcessor) ProcessConversion(ctx context.Context, conversionID string, inputFileURL string, styleName string) (string, error) {
	return m.processFunc(ctx, conversionID, inputFileURL, styleName)
}

// SetProcessFunc allows setting a custom process function
func (m *MockConversionProcessor) SetProcessFunc(fn func(ctx context.Context, conversionID string, inputFileURL string, styleName string) (string, error)) {
	m.processFunc = fn
}

// MockNotificationService implements NotificationService for testing
type MockNotificationService struct {
	conversionCompleteFunc func(ctx context.Context, userID string, conversionID string, outputFileURL string) error
	conversionFailedFunc   func(ctx context.Context, userID string, conversionID string, errorMessage string) error
	quotaWarningFunc       func(ctx context.Context, userID string, remainingConversions int) error
}

// NewMockNotificationService creates a new mock notification service
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		conversionCompleteFunc: func(ctx context.Context, userID string, conversionID string, outputFileURL string) error {
			return nil
		},
		conversionFailedFunc: func(ctx context.Context, userID string, conversionID string, errorMessage string) error {
			return nil
		},
		quotaWarningFunc: func(ctx context.Context, userID string, remainingConversions int) error {
			return nil
		},
	}
}

// SendConversionComplete simulates sending conversion complete notification
func (m *MockNotificationService) SendConversionComplete(ctx context.Context, userID string, conversionID string, outputFileURL string) error {
	return m.conversionCompleteFunc(ctx, userID, conversionID, outputFileURL)
}

// SendConversionFailed simulates sending conversion failed notification
func (m *MockNotificationService) SendConversionFailed(ctx context.Context, userID string, conversionID string, errorMessage string) error {
	return m.conversionFailedFunc(ctx, userID, conversionID, errorMessage)
}

// SendQuotaWarning simulates sending quota warning notification
func (m *MockNotificationService) SendQuotaWarning(ctx context.Context, userID string, remainingConversions int) error {
	return m.quotaWarningFunc(ctx, userID, remainingConversions)
}

// SetConversionCompleteFunc allows setting a custom conversion complete function
func (m *MockNotificationService) SetConversionCompleteFunc(fn func(ctx context.Context, userID string, conversionID string, outputFileURL string) error) {
	m.conversionCompleteFunc = fn
}

// SetConversionFailedFunc allows setting a custom conversion failed function
func (m *MockNotificationService) SetConversionFailedFunc(fn func(ctx context.Context, userID string, conversionID string, errorMessage string) error) {
	m.conversionFailedFunc = fn
}

// SetQuotaWarningFunc allows setting a custom quota warning function
func (m *MockNotificationService) SetQuotaWarningFunc(fn func(ctx context.Context, userID string, remainingConversions int) error) {
	m.quotaWarningFunc = fn
}

// MockFileStorage implements FileStorage for testing
type MockFileStorage struct {
	uploadFunc func(ctx context.Context, fileData []byte, fileName string) (string, error)
	getURLFunc func(ctx context.Context, filePath string) (string, error)
	deleteFunc func(ctx context.Context, filePath string) error
}

// NewMockFileStorage creates a new mock file storage
func NewMockFileStorage() *MockFileStorage {
	return &MockFileStorage{
		uploadFunc: func(ctx context.Context, fileData []byte, fileName string) (string, error) {
			return "https://example.com/files/" + fileName, nil
		},
		getURLFunc: func(ctx context.Context, filePath string) (string, error) {
			return "https://example.com/files/" + filePath, nil
		},
		deleteFunc: func(ctx context.Context, filePath string) error {
			return nil
		},
	}
}

// UploadFile simulates file upload
func (m *MockFileStorage) UploadFile(ctx context.Context, fileData []byte, fileName string) (string, error) {
	return m.uploadFunc(ctx, fileData, fileName)
}

// GetFileURL simulates getting file URL
func (m *MockFileStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	return m.getURLFunc(ctx, filePath)
}

// DeleteFile simulates file deletion
func (m *MockFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return m.deleteFunc(ctx, filePath)
}

// SetUploadFunc allows setting a custom upload function
func (m *MockFileStorage) SetUploadFunc(fn func(ctx context.Context, fileData []byte, fileName string) (string, error)) {
	m.uploadFunc = fn
}

// SetGetURLFunc allows setting a custom get URL function
func (m *MockFileStorage) SetGetURLFunc(fn func(ctx context.Context, filePath string) (string, error)) {
	m.getURLFunc = fn
}

// SetDeleteFunc allows setting a custom delete function
func (m *MockFileStorage) SetDeleteFunc(fn func(ctx context.Context, filePath string) error) {
	m.deleteFunc = fn
}

// MockRateLimiter implements RateLimiter for testing
type MockRateLimiter struct {
	allowFunc func(ctx context.Context, key string, limit int, window time.Duration) bool
}

// NewMockRateLimiter creates a new mock rate limiter
func NewMockRateLimiter() *MockRateLimiter {
	return &MockRateLimiter{
		allowFunc: func(ctx context.Context, key string, limit int, window time.Duration) bool {
			return true
		},
	}
}

// Allow simulates rate limiting check
func (m *MockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	return m.allowFunc(ctx, key, limit, window)
}

// SetAllowFunc allows setting a custom allow function
func (m *MockRateLimiter) SetAllowFunc(fn func(ctx context.Context, key string, limit int, window time.Duration) bool) {
	m.allowFunc = fn
}

// MockAuditLogger implements AuditLogger for testing
type MockAuditLogger struct {
	logFunc func(ctx context.Context, userID string, action string, metadata map[string]interface{}) error
}

// NewMockAuditLogger creates a new mock audit logger
func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{
		logFunc: func(ctx context.Context, userID string, action string, metadata map[string]interface{}) error {
			return nil
		},
	}
}

// LogUserAction simulates audit logging
func (m *MockAuditLogger) LogUserAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error {
	return m.logFunc(ctx, userID, action, metadata)
}

// SetLogFunc allows setting a custom log function
func (m *MockAuditLogger) SetLogFunc(fn func(ctx context.Context, userID string, action string, metadata map[string]interface{}) error) {
	m.logFunc = fn
}
