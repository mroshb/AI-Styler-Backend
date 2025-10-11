package image

import (
	"context"
	"io"
)

// Store defines the interface for image data persistence
type Store interface {
	// Image operations
	CreateImage(ctx context.Context, req CreateImageRequest) (Image, error)
	GetImage(ctx context.Context, imageID string) (Image, error)
	UpdateImage(ctx context.Context, imageID string, req UpdateImageRequest) (Image, error)
	DeleteImage(ctx context.Context, imageID string) error
	ListImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error)

	// Quota operations
	CanUploadImage(ctx context.Context, userID *string, vendorID *string, imageType ImageType, fileSize int64) (bool, error)
	GetQuotaStatus(ctx context.Context, userID *string, vendorID *string) (QuotaStatus, error)

	// Statistics
	GetImageStats(ctx context.Context, userID *string, vendorID *string) (ImageStats, error)
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

// UsageTracker defines the interface for tracking image usage
type UsageTracker interface {
	// Usage tracking
	RecordUsage(ctx context.Context, imageID string, userID *string, action string, metadata map[string]interface{}) error
	GetUsageHistory(ctx context.Context, imageID string, req ImageUsageHistoryRequest) (ImageUsageHistoryResponse, error)
	GetUsageStats(ctx context.Context, imageID string) (UsageStats, error)
}

// Cache defines the interface for caching operations
type Cache interface {
	// Basic operations
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl int64) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error

	// Image-specific operations
	CacheImage(ctx context.Context, imageID string, image Image) error
	GetCachedImage(ctx context.Context, imageID string) (Image, error)
	CacheSignedURL(ctx context.Context, imageID string, url string, ttl int64) error
	GetCachedSignedURL(ctx context.Context, imageID string) (string, error)
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	// Image notifications
	SendImageUploaded(ctx context.Context, userID *string, vendorID *string, imageID string, imageType ImageType) error
	SendImageDeleted(ctx context.Context, userID *string, vendorID *string, imageID string, imageType ImageType) error
	SendQuotaWarning(ctx context.Context, userID *string, vendorID *string, quotaType string, remaining int) error
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// Image audit logging
	LogImageAction(ctx context.Context, imageID string, userID *string, vendorID *string, action string, metadata map[string]interface{}) error
	LogQuotaAction(ctx context.Context, userID *string, vendorID *string, action string, metadata map[string]interface{}) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Rate limiting
	Allow(ctx context.Context, key string, limit int, window int64) bool
	GetRemaining(ctx context.Context, key string, limit int, window int64) int
	Reset(ctx context.Context, key string) error
}

// Request types for store operations

type CreateImageRequest struct {
	UserID       *string                `json:"userId,omitempty"`
	VendorID     *string                `json:"vendorId,omitempty"`
	Type         ImageType              `json:"type"`
	FileName     string                 `json:"fileName"`
	OriginalURL  string                 `json:"originalUrl"`
	ThumbnailURL *string                `json:"thumbnailUrl,omitempty"`
	FileSize     int64                  `json:"fileSize"`
	MimeType     string                 `json:"mimeType"`
	Width        *int                   `json:"width,omitempty"`
	Height       *int                   `json:"height,omitempty"`
	IsPublic     bool                   `json:"isPublic"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Response types

type QuotaStatus struct {
	UserImagesRemaining   int    `json:"userImagesRemaining"`
	VendorImagesRemaining int    `json:"vendorImagesRemaining"`
	PaidImagesRemaining   int    `json:"paidImagesRemaining"`
	TotalImagesRemaining  int    `json:"totalImagesRemaining"`
	UserImagesUsed        int    `json:"userImagesUsed"`
	VendorImagesUsed      int    `json:"vendorImagesUsed"`
	TotalImagesUsed       int    `json:"totalImagesUsed"`
	UserImagesLimit       int    `json:"userImagesLimit"`
	VendorImagesLimit     int    `json:"vendorImagesLimit"`
	TotalFileSize         int64  `json:"totalFileSize"`
	FileSizeLimit         int64  `json:"fileSizeLimit"`
	PlanName              string `json:"planName"`
	MonthlyLimit          int    `json:"monthlyLimit"`
}

type UsageStats struct {
	TotalViews      int `json:"totalViews"`
	TotalDownloads  int `json:"totalDownloads"`
	UniqueUsers     int `json:"uniqueUsers"`
	RecentViews     int `json:"recentViews"`     // Last 24 hours
	RecentDownloads int `json:"recentDownloads"` // Last 24 hours
}

// File upload request
type FileUploadRequest struct {
	File        io.Reader
	FileName    string
	ContentType string
	Size        int64
}

// Image processing options
type ImageProcessingOptions struct {
	GenerateThumbnail bool
	ThumbnailWidth    int
	ThumbnailHeight   int
	ResizeWidth       *int
	ResizeHeight      *int
	Quality           int
	Format            string
}

// Storage configuration
type StorageConfig struct {
	BasePath      string
	MaxFileSize   int64
	AllowedTypes  []string
	ThumbnailPath string
	SignedURLTTL  int64
}

// Validation rules
type ValidationRules struct {
	MaxFileSize     int64
	AllowedTypes    []string
	MaxWidth        int
	MaxHeight       int
	MinWidth        int
	MinHeight       int
	MaxTags         int
	MaxTagLength    int
	MaxMetadataSize int
}
