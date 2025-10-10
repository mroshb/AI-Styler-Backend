package share

import (
	"context"
	"database/sql"
	"time"
)

// WireShareService creates a share service with all dependencies
func WireShareService(db *sql.DB) (*Service, *Handler) {
	// Create store
	store := NewStore(db)

	// Create mock dependencies (replace with real implementations in production)
	conversionService := NewMockConversionService()
	imageService := NewMockImageService()
	notifier := NewMockNotificationService()
	auditLogger := NewMockAuditLogger()
	metrics := NewMockMetricsCollector()

	// Create service
	service := NewService(
		store,
		conversionService,
		imageService,
		notifier,
		auditLogger,
		metrics,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// Mock implementations for testing and development
type MockConversionService struct{}

func NewMockConversionService() *MockConversionService {
	return &MockConversionService{}
}

func (m *MockConversionService) GetConversion(ctx context.Context, conversionID, userID string) (ConversionResponse, error) {
	return ConversionResponse{
		ID:            conversionID,
		UserID:        userID,
		UserImageID:   "user-image-id",
		ClothImageID:  "cloth-image-id",
		Status:        "completed",
		ResultImageID: stringPtr("result-image-id"),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (m *MockConversionService) ValidateConversionOwnership(ctx context.Context, conversionID, userID string) error {
	return nil
}

type MockImageService struct{}

func NewMockImageService() *MockImageService {
	return &MockImageService{}
}

func (m *MockImageService) GetImage(ctx context.Context, imageID string) (ImageResponse, error) {
	return ImageResponse{
		ID:           imageID,
		UserID:       "user-id",
		VendorID:     "vendor-id",
		Type:         "result",
		FileName:     "image.jpg",
		OriginalURL:  "https://example.com/image.jpg",
		ThumbnailURL: "https://example.com/thumb.jpg",
		FileSize:     1024,
		MimeType:     "image/jpeg",
		Width:        800,
		Height:       600,
		IsPublic:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *MockImageService) GenerateSignedURL(ctx context.Context, imageID string, accessType string, ttl int64) (string, error) {
	return "https://example.com/signed-url", nil
}

type MockNotificationService struct{}

func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{}
}

func (m *MockNotificationService) SendShareCreated(ctx context.Context, userID, shareID, shareToken string) error {
	return nil
}

func (m *MockNotificationService) SendShareAccessed(ctx context.Context, userID, shareID string, accessCount int) error {
	return nil
}

type MockAuditLogger struct{}

func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{}
}

func (m *MockAuditLogger) LogShareCreated(ctx context.Context, userID, conversionID, shareID string) error {
	return nil
}

func (m *MockAuditLogger) LogShareAccessed(ctx context.Context, shareID, ipAddress, userAgent string) error {
	return nil
}

func (m *MockAuditLogger) LogShareDeactivated(ctx context.Context, userID, shareID string) error {
	return nil
}

type MockMetricsCollector struct{}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{}
}

func (m *MockMetricsCollector) RecordShareCreated(ctx context.Context, userID, conversionID string) error {
	return nil
}

func (m *MockMetricsCollector) RecordShareAccessed(ctx context.Context, shareID string, success bool) error {
	return nil
}

func (m *MockMetricsCollector) RecordShareExpired(ctx context.Context, shareID string) error {
	return nil
}
