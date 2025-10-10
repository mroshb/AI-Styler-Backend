package image

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// Mock implementations for testing

type mockStore struct {
	images map[string]Image
	quotas map[string]QuotaStatus
	stats  map[string]ImageStats
}

func newMockStore() *mockStore {
	return &mockStore{
		images: make(map[string]Image),
		quotas: make(map[string]QuotaStatus),
		stats:  make(map[string]ImageStats),
	}
}

func (m *mockStore) CreateImage(ctx context.Context, req CreateImageRequest) (Image, error) {
	image := Image{
		ID:           "test-image-id",
		UserID:       req.UserID,
		VendorID:     req.VendorID,
		Type:         req.Type,
		FileName:     req.FileName,
		OriginalURL:  req.OriginalURL,
		ThumbnailURL: req.ThumbnailURL,
		FileSize:     req.FileSize,
		MimeType:     req.MimeType,
		Width:        req.Width,
		Height:       req.Height,
		IsPublic:     req.IsPublic,
		Tags:         req.Tags,
		Metadata:     req.Metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.images[image.ID] = image
	return image, nil
}

func (m *mockStore) GetImage(ctx context.Context, imageID string) (Image, error) {
	image, exists := m.images[imageID]
	if !exists {
		return Image{}, errors.New("image not found")
	}
	return image, nil
}

func (m *mockStore) UpdateImage(ctx context.Context, imageID string, req UpdateImageRequest) (Image, error) {
	image, exists := m.images[imageID]
	if !exists {
		return Image{}, errors.New("image not found")
	}

	if req.IsPublic != nil {
		image.IsPublic = *req.IsPublic
	}
	if req.Tags != nil {
		image.Tags = req.Tags
	}
	if req.Metadata != nil {
		image.Metadata = req.Metadata
	}
	image.UpdatedAt = time.Now()

	m.images[imageID] = image
	return image, nil
}

func (m *mockStore) DeleteImage(ctx context.Context, imageID string) error {
	if _, exists := m.images[imageID]; !exists {
		return errors.New("image not found")
	}
	delete(m.images, imageID)
	return nil
}

func (m *mockStore) ListImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	// Simple mock implementation
	var images []Image
	for _, image := range m.images {
		images = append(images, image)
	}

	return ImageListResponse{
		Images:     images,
		Total:      len(images),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *mockStore) CanUploadImage(ctx context.Context, userID *string, vendorID *string, imageType ImageType, fileSize int64) (bool, error) {
	key := "default"
	if userID != nil {
		key = *userID
	} else if vendorID != nil {
		key = *vendorID
	}

	quota, exists := m.quotas[key]
	if !exists {
		// Default quota
		quota = QuotaStatus{
			UserImagesRemaining:   100,
			VendorImagesRemaining: 1000,
			TotalImagesRemaining:  100,
			TotalFileSize:         0,
			FileSizeLimit:         1073741824, // 1GB
		}
	}

	// Check image count limit
	var canUpload bool
	switch imageType {
	case ImageTypeUser:
		canUpload = quota.UserImagesRemaining > 0
	case ImageTypeVendor:
		canUpload = quota.VendorImagesRemaining > 0
	default:
		canUpload = quota.TotalImagesRemaining > 0
	}

	if !canUpload {
		return false, nil
	}

	// Check file size limit
	if quota.TotalFileSize+fileSize > quota.FileSizeLimit {
		return false, nil
	}

	return true, nil
}

func (m *mockStore) GetQuotaStatus(ctx context.Context, userID *string, vendorID *string) (QuotaStatus, error) {
	key := "default"
	if userID != nil {
		key = *userID
	} else if vendorID != nil {
		key = *vendorID
	}

	quota, exists := m.quotas[key]
	if !exists {
		quota = QuotaStatus{
			UserImagesRemaining:   100,
			VendorImagesRemaining: 1000,
			TotalImagesRemaining:  100,
			UserImagesLimit:       100,
			VendorImagesLimit:     1000,
			TotalFileSize:         0,
			FileSizeLimit:         1073741824,
		}
	}

	return quota, nil
}

func (m *mockStore) GetImageStats(ctx context.Context, userID *string, vendorID *string) (ImageStats, error) {
	key := "default"
	if userID != nil {
		key = *userID
	} else if vendorID != nil {
		key = *vendorID
	}

	stats, exists := m.stats[key]
	if !exists {
		stats = ImageStats{
			TotalImages:     0,
			UserImages:      0,
			VendorImages:    0,
			ResultImages:    0,
			PublicImages:    0,
			PrivateImages:   0,
			TotalFileSize:   0,
			AverageFileSize: 0,
		}
	}

	return stats, nil
}

type mockFileStorage struct{}

func (m *mockFileStorage) UploadFile(ctx context.Context, data []byte, fileName string, path string) (string, error) {
	return "https://example.com/storage/" + path + "/" + fileName, nil
}

func (m *mockFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return nil
}

func (m *mockFileStorage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	return []byte("test file content"), nil
}

func (m *mockFileStorage) GenerateSignedURL(ctx context.Context, filePath string, accessType string, ttl int64) (string, error) {
	return "https://example.com/signed/" + filePath + "?expires=" + string(rune(ttl)), nil
}

func (m *mockFileStorage) ValidateSignedURL(ctx context.Context, signedURL string) (bool, string, error) {
	return true, "test-file-path", nil
}

type mockImageProcessor struct{}

func (m *mockImageProcessor) ProcessImage(ctx context.Context, data []byte, fileName string) ([]byte, int, int, error) {
	return data, 800, 600, nil
}

func (m *mockImageProcessor) GenerateThumbnail(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error) {
	return data, nil
}

func (m *mockImageProcessor) ResizeImage(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error) {
	return data, nil
}

func (m *mockImageProcessor) ValidateImage(ctx context.Context, data []byte, fileName string, mimeType string) error {
	return nil
}

func (m *mockImageProcessor) GetImageDimensions(ctx context.Context, data []byte) (int, int, error) {
	return 800, 600, nil
}

type mockUsageTracker struct{}

func (m *mockUsageTracker) RecordUsage(ctx context.Context, imageID string, userID *string, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *mockUsageTracker) GetUsageHistory(ctx context.Context, imageID string, req ImageUsageHistoryRequest) (ImageUsageHistoryResponse, error) {
	return ImageUsageHistoryResponse{
		History:    []ImageUsageHistory{},
		Total:      0,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 0,
	}, nil
}

func (m *mockUsageTracker) GetUsageStats(ctx context.Context, imageID string) (UsageStats, error) {
	return UsageStats{
		TotalViews:      0,
		TotalDownloads:  0,
		UniqueUsers:     0,
		RecentViews:     0,
		RecentDownloads: 0,
	}, nil
}

type mockCache struct{}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	return "", errors.New("not found")
}

func (m *mockCache) Set(ctx context.Context, key string, value string, ttl int64) error {
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *mockCache) CacheImage(ctx context.Context, imageID string, image Image) error {
	return nil
}

func (m *mockCache) GetCachedImage(ctx context.Context, imageID string) (Image, error) {
	return Image{}, errors.New("not found")
}

func (m *mockCache) CacheSignedURL(ctx context.Context, imageID string, url string, ttl int64) error {
	return nil
}

func (m *mockCache) GetCachedSignedURL(ctx context.Context, imageID string) (string, error) {
	return "", errors.New("not found")
}

type mockNotificationService struct{}

func (m *mockNotificationService) SendImageUploaded(ctx context.Context, userID *string, vendorID *string, imageID string, imageType ImageType) error {
	return nil
}

func (m *mockNotificationService) SendImageDeleted(ctx context.Context, userID *string, vendorID *string, imageID string, imageType ImageType) error {
	return nil
}

func (m *mockNotificationService) SendQuotaWarning(ctx context.Context, userID *string, vendorID *string, quotaType string, remaining int) error {
	return nil
}

type mockAuditLogger struct{}

func (m *mockAuditLogger) LogImageAction(ctx context.Context, imageID string, userID *string, vendorID *string, action string, metadata map[string]interface{}) error {
	return nil
}

func (m *mockAuditLogger) LogQuotaAction(ctx context.Context, userID *string, vendorID *string, action string, metadata map[string]interface{}) error {
	return nil
}

type mockRateLimiter struct{}

func (m *mockRateLimiter) Allow(ctx context.Context, key string, limit int, window int64) bool {
	return true
}

func (m *mockRateLimiter) GetRemaining(ctx context.Context, key string, limit int, window int64) int {
	return limit
}

func (m *mockRateLimiter) Reset(ctx context.Context, key string) error {
	return nil
}

// Test cases

func TestUploadImage(t *testing.T) {
	service := NewService(
		newMockStore(),
		&mockFileStorage{},
		&mockImageProcessor{},
		&mockUsageTracker{},
		&mockCache{},
		&mockNotificationService{},
		&mockAuditLogger{},
		&mockRateLimiter{},
		StorageConfig{
			MaxFileSize:  10 * 1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png"},
		},
	)

	userID := "test-user-id"
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	req := UploadImageRequest{
		Type:     ImageTypeUser,
		FileName: "test.jpg",
		FileSize: 1024,
		MimeType: "image/jpeg",
		IsPublic: false,
		Tags:     []string{"test"},
		File:     &mockReader{data: testData},
	}

	image, err := service.UploadImage(context.Background(), &userID, nil, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if image.UserID == nil || *image.UserID != userID {
		t.Errorf("Expected user ID %s, got %v", userID, image.UserID)
	}

	if image.Type != ImageTypeUser {
		t.Errorf("Expected type %s, got %s", ImageTypeUser, image.Type)
	}

	if image.FileName != "test.jpg" {
		t.Errorf("Expected file name test.jpg, got %s", image.FileName)
	}
}

func TestUploadImageValidation(t *testing.T) {
	service := NewService(
		newMockStore(),
		&mockFileStorage{},
		&mockImageProcessor{},
		&mockUsageTracker{},
		&mockCache{},
		&mockNotificationService{},
		&mockAuditLogger{},
		&mockRateLimiter{},
		StorageConfig{
			MaxFileSize:  10 * 1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png"},
		},
	)

	userID := "test-user-id"

	// Test empty file name
	req := UploadImageRequest{
		Type:     ImageTypeUser,
		FileName: "",
		FileSize: 1024,
		MimeType: "image/jpeg",
		File:     &mockReader{data: []byte("test")},
	}

	_, err := service.UploadImage(context.Background(), &userID, nil, req)
	if err == nil {
		t.Error("Expected error for empty file name")
	}

	// Test unsupported MIME type
	req = UploadImageRequest{
		Type:     ImageTypeUser,
		FileName: "test.txt",
		FileSize: 1024,
		MimeType: "text/plain",
		File:     &mockReader{data: []byte("test")},
	}

	_, err = service.UploadImage(context.Background(), &userID, nil, req)
	if err == nil {
		t.Error("Expected error for unsupported MIME type")
	}

	// Test file too large
	largeData := make([]byte, 20*1024*1024)
	req = UploadImageRequest{
		Type:     ImageTypeUser,
		FileName: "test.jpg",
		FileSize: 20 * 1024 * 1024, // 20MB
		MimeType: "image/jpeg",
		File:     &mockReader{data: largeData},
	}

	_, err = service.UploadImage(context.Background(), &userID, nil, req)
	if err == nil {
		t.Error("Expected error for file too large")
	}
}

func TestGetImage(t *testing.T) {
	store := newMockStore()
	service := NewService(
		store,
		&mockFileStorage{},
		&mockImageProcessor{},
		&mockUsageTracker{},
		&mockCache{},
		&mockNotificationService{},
		&mockAuditLogger{},
		&mockRateLimiter{},
		StorageConfig{
			MaxFileSize:  10 * 1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png"},
		},
	)

	// Create a test image
	userID := "test-user-id"
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	req := UploadImageRequest{
		Type:     ImageTypeUser,
		FileName: "test.jpg",
		FileSize: 1024,
		MimeType: "image/jpeg",
		File:     &mockReader{data: testData},
	}

	image, err := service.UploadImage(context.Background(), &userID, nil, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Get the image
	retrievedImage, err := service.GetImage(context.Background(), image.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrievedImage.ID != image.ID {
		t.Errorf("Expected image ID %s, got %s", image.ID, retrievedImage.ID)
	}
}

func TestDeleteImage(t *testing.T) {
	store := newMockStore()
	service := NewService(
		store,
		&mockFileStorage{},
		&mockImageProcessor{},
		&mockUsageTracker{},
		&mockCache{},
		&mockNotificationService{},
		&mockAuditLogger{},
		&mockRateLimiter{},
		StorageConfig{
			MaxFileSize:  10 * 1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png"},
		},
	)

	// Create a test image
	userID := "test-user-id"
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	req := UploadImageRequest{
		Type:     ImageTypeUser,
		FileName: "test.jpg",
		FileSize: 1024,
		MimeType: "image/jpeg",
		File:     &mockReader{data: testData},
	}

	image, err := service.UploadImage(context.Background(), &userID, nil, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Delete the image
	err = service.DeleteImage(context.Background(), image.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Try to get the deleted image
	_, err = service.GetImage(context.Background(), image.ID)
	if err == nil {
		t.Error("Expected error for deleted image")
	}
}

// Helper types for testing

type mockReader struct {
	data []byte
	pos  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}
