package conversion

import (
	"context"
)

// Store defines the interface for conversion data operations
type Store interface {
	// Conversion operations
	CreateConversion(ctx context.Context, userID, userImageID, clothImageID string) (string, error)
	GetConversion(ctx context.Context, conversionID string) (Conversion, error)
	GetConversionWithDetails(ctx context.Context, conversionID string) (ConversionResponse, error)
	UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) error
	ListConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error)
	DeleteConversion(ctx context.Context, conversionID string) error

	// Quota operations
	CheckUserQuota(ctx context.Context, userID string) (QuotaCheck, error)
	ReserveQuota(ctx context.Context, userID string) error
	ReleaseQuota(ctx context.Context, userID string) error

	// Job operations
	CreateConversionJob(ctx context.Context, conversionID string) error
	GetNextJob(ctx context.Context) (*ConversionJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status string, workerID string) error
	CompleteJob(ctx context.Context, jobID string, resultImageID string, processingTimeMs int) error
	FailJob(ctx context.Context, jobID string, errorMessage string) error
}

// ConversionJob represents a background conversion job
type ConversionJob struct {
	ID           string `json:"id"`
	ConversionID string `json:"conversionId"`
	Status       string `json:"status"`
	WorkerID     string `json:"workerId,omitempty"`
	Priority     int    `json:"priority"`
	RetryCount   int    `json:"retryCount"`
	MaxRetries   int    `json:"maxRetries"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ImageService defines the interface for image operations
type ImageService interface {
	GetImage(ctx context.Context, imageID string) (ImageInfo, error)
	ValidateImageAccess(ctx context.Context, imageID, userID string) error
	CreateResultImage(ctx context.Context, userID string, imageData []byte, metadata map[string]interface{}) (string, error)
}

// ImageInfo represents basic image information
type ImageInfo struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	VendorID    string `json:"vendorId"`
	OriginalURL string `json:"originalUrl"`
	MimeType    string `json:"mimeType"`
	FileSize    int64  `json:"fileSize"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	IsPublic    bool   `json:"isPublic"`
}

// ConversionProcessor defines the interface for processing conversions
type ConversionProcessor interface {
	ProcessConversion(ctx context.Context, userImageID, clothImageID string) (string, error)
	GetProcessingStatus(ctx context.Context, jobID string) (string, error)
	CancelProcessing(ctx context.Context, jobID string) error
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendConversionStarted(ctx context.Context, userID, conversionID string) error
	SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error
	SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	CheckRateLimit(ctx context.Context, userID string) (bool, error)
	RecordRequest(ctx context.Context, userID string) error
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogConversionRequest(ctx context.Context, userID, conversionID string, request ConversionRequest) error
	LogConversionUpdate(ctx context.Context, userID, conversionID string, update UpdateConversionRequest) error
	LogConversionError(ctx context.Context, userID, conversionID string, error error) error
}

// WorkerService defines the interface for background job processing
type WorkerService interface {
	EnqueueConversion(ctx context.Context, conversionID string) error
	ProcessConversion(ctx context.Context, jobID string) error
	GetJobStatus(ctx context.Context, jobID string) (string, error)
	CancelJob(ctx context.Context, jobID string) error
}

// MetricsCollector defines the interface for collecting conversion metrics
type MetricsCollector interface {
	RecordConversionStart(ctx context.Context, conversionID, userID string) error
	RecordConversionComplete(ctx context.Context, conversionID string, processingTimeMs int, success bool) error
	RecordConversionError(ctx context.Context, conversionID string, errorType string) error
	GetConversionMetrics(ctx context.Context, userID string, timeRange string) (map[string]interface{}, error)
}
