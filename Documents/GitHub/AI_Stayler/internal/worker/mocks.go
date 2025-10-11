package worker

import (
	"context"
	"time"

	"ai-styler/internal/conversion"
	"ai-styler/internal/image"
)

// Mock implementations for testing

// MockJobQueue implements JobQueue interface
type MockJobQueue struct{}

func NewMockJobQueue() JobQueue {
	return &MockJobQueue{}
}

func (m *MockJobQueue) EnqueueJob(ctx context.Context, job *WorkerJob) error {
	return nil
}

func (m *MockJobQueue) DequeueJob(ctx context.Context, workerID string) (*WorkerJob, error) {
	return &WorkerJob{
		ID:           "mock-job-1",
		Type:         "image_conversion",
		ConversionID: "conv-1",
		UserID:       "user-1",
		Priority:     JobPriorityNormal,
		Status:       JobStatusPending,
		RetryCount:   0,
		MaxRetries:   3,
		Payload: JobPayload{
			UserImageID:  "img-1",
			ClothImageID: "img-2",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockJobQueue) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, workerID string) error {
	return nil
}

func (m *MockJobQueue) CompleteJob(ctx context.Context, jobID string, result interface{}) error {
	return nil
}

func (m *MockJobQueue) FailJob(ctx context.Context, jobID string, errorMessage string) error {
	return nil
}

func (m *MockJobQueue) GetJob(ctx context.Context, jobID string) (*WorkerJob, error) {
	return &WorkerJob{
		ID:           jobID,
		Type:         "image_conversion",
		ConversionID: "conv-1",
		UserID:       "user-1",
		Priority:     JobPriorityNormal,
		Status:       JobStatusPending,
		RetryCount:   0,
		MaxRetries:   3,
		Payload: JobPayload{
			UserImageID:  "img-1",
			ClothImageID: "img-2",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockJobQueue) GetQueueStats(ctx context.Context) (*WorkerStats, error) {
	return &WorkerStats{
		TotalJobs:      100,
		PendingJobs:    10,
		ProcessingJobs: 5,
		CompletedJobs:  80,
		FailedJobs:     5,
		ActiveWorkers:  3,
		SuccessRate:    80.0,
	}, nil
}

func (m *MockJobQueue) CleanupOldJobs(ctx context.Context, olderThan time.Time) error {
	return nil
}

func (m *MockJobQueue) GetPendingJobs(ctx context.Context, limit int) ([]*WorkerJob, error) {
	return []*WorkerJob{}, nil
}

// MockGeminiAPI implements GeminiAPI interface
type MockGeminiAPI struct{}

func NewMockGeminiAPI() GeminiAPI {
	return &MockGeminiAPI{}
}

func (m *MockGeminiAPI) ConvertImage(ctx context.Context, userImageData, clothImageData []byte, options map[string]interface{}) ([]byte, error) {
	// Return mock converted image data
	return []byte("mock-converted-image-data"), nil
}

func (m *MockGeminiAPI) GetConversionStatus(ctx context.Context, jobID string) (string, error) {
	return "completed", nil
}

func (m *MockGeminiAPI) CancelConversion(ctx context.Context, jobID string) error {
	return nil
}

func (m *MockGeminiAPI) HealthCheck(ctx context.Context) error {
	return nil
}

// MockMetricsCollector implements MetricsCollector interface
type MockMetricsCollector struct{}

func NewMockMetricsCollector() MetricsCollector {
	return &MockMetricsCollector{}
}

func (m *MockMetricsCollector) RecordJobStart(ctx context.Context, jobID, jobType string) error {
	return nil
}

func (m *MockMetricsCollector) RecordJobComplete(ctx context.Context, jobID string, processingTimeMs int, success bool) error {
	return nil
}

func (m *MockMetricsCollector) RecordJobError(ctx context.Context, jobID string, errorType string) error {
	return nil
}

func (m *MockMetricsCollector) RecordWorkerHealth(ctx context.Context, workerID string, status string) error {
	return nil
}

func (m *MockMetricsCollector) GetWorkerMetrics(ctx context.Context, timeRange string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"totalJobs":     100,
		"completedJobs": 95,
		"failedJobs":    5,
		"successRate":   95.0,
	}, nil
}

// MockHealthChecker implements HealthChecker interface
type MockHealthChecker struct{}

func NewMockHealthChecker() HealthChecker {
	return &MockHealthChecker{}
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context) (*WorkerHealth, error) {
	return &WorkerHealth{
		WorkerID:      "mock-worker-1",
		Status:        "healthy",
		LastSeen:      time.Now(),
		JobsProcessed: 100,
		Uptime:        3600,
	}, nil
}

func (m *MockHealthChecker) RegisterWorker(ctx context.Context, workerID string) error {
	return nil
}

func (m *MockHealthChecker) UnregisterWorker(ctx context.Context, workerID string) error {
	return nil
}

func (m *MockHealthChecker) GetWorkerList(ctx context.Context) ([]*WorkerHealth, error) {
	return []*WorkerHealth{
		{
			WorkerID:      "mock-worker-1",
			Status:        "healthy",
			LastSeen:      time.Now(),
			JobsProcessed: 100,
			Uptime:        3600,
		},
	}, nil
}

// MockRetryHandler implements RetryHandler interface
type MockRetryHandler struct{}

func NewMockRetryHandler() RetryHandler {
	return &MockRetryHandler{}
}

func (m *MockRetryHandler) ShouldRetry(ctx context.Context, job *WorkerJob, err error) bool {
	return job.RetryCount < job.MaxRetries
}

func (m *MockRetryHandler) GetRetryDelay(ctx context.Context, job *WorkerJob) time.Duration {
	return time.Second * time.Duration(job.RetryCount+1)
}

func (m *MockRetryHandler) IncrementRetryCount(ctx context.Context, job *WorkerJob) error {
	job.RetryCount++
	return nil
}

// MockFileStorage implements FileStorage interface
type MockFileStorage struct{}

func NewMockFileStorage() FileStorage {
	return &MockFileStorage{}
}

func (m *MockFileStorage) UploadFile(ctx context.Context, data []byte, fileName string, path string) (string, error) {
	return "/mock/path/" + fileName, nil
}

func (m *MockFileStorage) DeleteFile(ctx context.Context, filePath string) error {
	return nil
}

func (m *MockFileStorage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	return []byte("mock file data"), nil
}

func (m *MockFileStorage) GenerateSignedURL(ctx context.Context, filePath string, accessType string, ttl int64) (string, error) {
	return "https://mock-signed-url.com/" + filePath, nil
}

func (m *MockFileStorage) ValidateSignedURL(ctx context.Context, signedURL string) (bool, string, error) {
	return true, "/mock/path/file.jpg", nil
}

// MockConversionStore implements ConversionStore interface
type MockConversionStore struct{}

func NewMockConversionStore() ConversionStore {
	return &MockConversionStore{}
}

func (m *MockConversionStore) GetConversion(ctx context.Context, conversionID string) (conversion.Conversion, error) {
	return conversion.Conversion{
		ID:               conversionID,
		UserID:           "user-1",
		UserImageID:      "img-1",
		ClothImageID:     "img-2",
		Status:           "completed",
		ResultImageID:    stringPtr("result-img-1"),
		ProcessingTimeMs: intPtr(5000),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CompletedAt:      timePtr(time.Now()),
	}, nil
}

func (m *MockConversionStore) UpdateConversion(ctx context.Context, conversionID string, req conversion.UpdateConversionRequest) error {
	return nil
}

func (m *MockConversionStore) CreateConversionJob(ctx context.Context, conversionID string) error {
	return nil
}

func (m *MockConversionStore) GetNextJob(ctx context.Context) (*conversion.ConversionJob, error) {
	return &conversion.ConversionJob{
		ID:           "job-1",
		ConversionID: "conv-1",
		Status:       "pending",
		Priority:     5,
		RetryCount:   0,
		MaxRetries:   3,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}, nil
}

func (m *MockConversionStore) UpdateJobStatus(ctx context.Context, jobID string, status string, workerID string) error {
	return nil
}

func (m *MockConversionStore) CompleteJob(ctx context.Context, jobID string, resultImageID string, processingTimeMs int) error {
	return nil
}

func (m *MockConversionStore) FailJob(ctx context.Context, jobID string, errorMessage string) error {
	return nil
}

// MockImageStore implements ImageStore interface
type MockImageStore struct{}

func NewMockImageStore() ImageStore {
	return &MockImageStore{}
}

func (m *MockImageStore) GetImage(ctx context.Context, imageID string) (image.Image, error) {
	return image.Image{
		ID:          imageID,
		UserID:      stringPtr("user-1"),
		Type:        image.ImageTypeUser,
		FileName:    "test.jpg",
		OriginalURL: "https://example.com/test.jpg",
		FileSize:    1024000,
		MimeType:    "image/jpeg",
		Width:       intPtr(800),
		Height:      intPtr(600),
		IsPublic:    true,
		Tags:        []string{"test"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (m *MockImageStore) CreateImage(ctx context.Context, req image.CreateImageRequest) (image.Image, error) {
	return image.Image{
		ID:           "new-image-id",
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
	}, nil
}

func (m *MockImageStore) UpdateImage(ctx context.Context, imageID string, req image.UpdateImageRequest) (image.Image, error) {
	return image.Image{
		ID:          imageID,
		UserID:      stringPtr("user-1"),
		Type:        image.ImageTypeUser,
		FileName:    "updated.jpg",
		OriginalURL: "https://example.com/updated.jpg",
		FileSize:    1024000,
		MimeType:    "image/jpeg",
		Width:       intPtr(800),
		Height:      intPtr(600),
		IsPublic:    true,
		Tags:        []string{"updated"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (m *MockImageStore) CanUploadImage(ctx context.Context, userID *string, vendorID *string, imageType image.ImageType, fileSize int64) (bool, error) {
	return true, nil
}

// MockNotificationService implements NotificationService interface
type MockNotificationService struct{}

func NewMockNotificationService() NotificationService {
	return &MockNotificationService{}
}

func (m *MockNotificationService) SendConversionStarted(ctx context.Context, userID, conversionID string) error {
	return nil
}

func (m *MockNotificationService) SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error {
	return nil
}

func (m *MockNotificationService) SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error {
	return nil
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}
