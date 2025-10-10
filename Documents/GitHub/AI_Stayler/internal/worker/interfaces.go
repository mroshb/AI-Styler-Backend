package worker

import (
	"context"
	"time"

	"AI_Styler/internal/conversion"
	"AI_Styler/internal/image"
)

// JobQueue defines the interface for job queue operations
type JobQueue interface {
	// Job operations
	EnqueueJob(ctx context.Context, job *WorkerJob) error
	DequeueJob(ctx context.Context, workerID string) (*WorkerJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, workerID string) error
	CompleteJob(ctx context.Context, jobID string, result interface{}) error
	FailJob(ctx context.Context, jobID string, errorMessage string) error
	GetJob(ctx context.Context, jobID string) (*WorkerJob, error)

	// Queue management
	GetQueueStats(ctx context.Context) (*WorkerStats, error)
	CleanupOldJobs(ctx context.Context, olderThan time.Time) error
	GetPendingJobs(ctx context.Context, limit int) ([]*WorkerJob, error)
}

// ImageProcessor defines the interface for image processing operations
type ImageProcessor interface {
	// Image processing
	ProcessImage(ctx context.Context, data []byte, fileName string) ([]byte, int, int, error)
	GenerateThumbnail(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error)
	ResizeImage(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error)

	// Image validation
	ValidateImage(ctx context.Context, data []byte, fileName string, mimeType string) error
	GetImageDimensions(ctx context.Context, data []byte) (int, int, error)
}

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	// File operations
	UploadFile(ctx context.Context, data []byte, fileName string, path string) (string, error)
	DeleteFile(ctx context.Context, filePath string) error
	GetFile(ctx context.Context, filePath string) ([]byte, error)

	// Signed URL operations
	GenerateSignedURL(ctx context.Context, filePath string, accessType string, ttl int64) (string, error)
	ValidateSignedURL(ctx context.Context, signedURL string) (bool, string, error)
}

// ConversionStore defines the interface for conversion data operations
type ConversionStore interface {
	// Conversion operations
	GetConversion(ctx context.Context, conversionID string) (conversion.Conversion, error)
	UpdateConversion(ctx context.Context, conversionID string, req conversion.UpdateConversionRequest) error

	// Job operations
	CreateConversionJob(ctx context.Context, conversionID string) error
	GetNextJob(ctx context.Context) (*conversion.ConversionJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status string, workerID string) error
	CompleteJob(ctx context.Context, jobID string, resultImageID string, processingTimeMs int) error
	FailJob(ctx context.Context, jobID string, errorMessage string) error
}

// ImageStore defines the interface for image data operations
type ImageStore interface {
	// Image operations
	GetImage(ctx context.Context, imageID string) (image.Image, error)
	CreateImage(ctx context.Context, req image.CreateImageRequest) (image.Image, error)
	UpdateImage(ctx context.Context, imageID string, req image.UpdateImageRequest) (image.Image, error)

	// Quota operations
	CanUploadImage(ctx context.Context, userID *string, vendorID *string, imageType image.ImageType, fileSize int64) (bool, error)
}

// GeminiAPI defines the interface for Gemini API operations
type GeminiAPI interface {
	// Image conversion
	ConvertImage(ctx context.Context, userImageData, clothImageData []byte, options map[string]interface{}) ([]byte, error)
	GetConversionStatus(ctx context.Context, jobID string) (string, error)
	CancelConversion(ctx context.Context, jobID string) error

	// Health check
	HealthCheck(ctx context.Context) error
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendConversionStarted(ctx context.Context, userID, conversionID string) error
	SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error
	SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error
}

// MetricsCollector defines the interface for collecting worker metrics
type MetricsCollector interface {
	RecordJobStart(ctx context.Context, jobID, jobType string) error
	RecordJobComplete(ctx context.Context, jobID string, processingTimeMs int, success bool) error
	RecordJobError(ctx context.Context, jobID string, errorType string) error
	RecordWorkerHealth(ctx context.Context, workerID string, status string) error
	GetWorkerMetrics(ctx context.Context, timeRange string) (map[string]interface{}, error)
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*WorkerHealth, error)
	RegisterWorker(ctx context.Context, workerID string) error
	UnregisterWorker(ctx context.Context, workerID string) error
	GetWorkerList(ctx context.Context) ([]*WorkerHealth, error)
}

// RetryHandler defines the interface for retry operations
type RetryHandler interface {
	ShouldRetry(ctx context.Context, job *WorkerJob, err error) bool
	GetRetryDelay(ctx context.Context, job *WorkerJob) time.Duration
	IncrementRetryCount(ctx context.Context, job *WorkerJob) error
}

// WorkerService defines the main worker service interface
type WorkerService interface {
	// Worker management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetStatus(ctx context.Context) (*WorkerStats, error)
	GetHealth(ctx context.Context) (*WorkerHealth, error)

	// Job management
	EnqueueJob(ctx context.Context, jobType string, conversionID, userID string, payload JobPayload) error
	ProcessJob(ctx context.Context, job *WorkerJob) error
	CancelJob(ctx context.Context, jobID string) error

	// Configuration
	UpdateConfig(ctx context.Context, config *WorkerConfig) error
	GetConfig(ctx context.Context) (*WorkerConfig, error)
}
