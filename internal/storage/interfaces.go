package storage

import (
	"context"
	"io"
	"time"
)

// StorageService defines the interface for file storage operations
type StorageServiceInterface interface {
	// File operations
	UploadFile(ctx context.Context, data []byte, fileName string, path string) (string, error)
	DeleteFile(ctx context.Context, filePath string) error
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	GetFileInfo(ctx context.Context, filePath string) (*FileInfo, error)
	CopyFile(ctx context.Context, srcPath, dstPath string) error
	MoveFile(ctx context.Context, srcPath, dstPath string) error

	// Signed URL operations
	GenerateSignedURL(ctx context.Context, filePath string, accessType string, ttl int64) (string, error)
	ValidateSignedURL(ctx context.Context, signedURL string) (bool, string, error)

	// Backup and retention operations
	CreateBackup(ctx context.Context, filePath string) error
	RestoreFromBackup(ctx context.Context, filePath string, backupDate string) error
	CleanupOldBackups(ctx context.Context, daysToKeep int) error

	// Statistics and monitoring
	GetStorageStats(ctx context.Context) (*StorageStats, error)
	GetDiskUsage(ctx context.Context) (*DiskUsage, error)
	ListFiles(ctx context.Context, directory string, page, pageSize int) ([]FileInfo, error)
}

// ImageStorageService provides specialized image storage functionality
type ImageStorageService struct {
	storage StorageServiceInterface
	config  ImageStorageConfig
}

// ImageStorageConfig holds configuration for image storage
type ImageStorageConfig struct {
	BasePath        string          `json:"basePath"`
	MaxFileSize     int64           `json:"maxFileSize"`
	AllowedTypes    []string        `json:"allowedTypes"`
	ThumbnailSizes  []ThumbnailSize `json:"thumbnailSizes"`
	RetentionPolicy RetentionPolicy `json:"retentionPolicy"`
	BackupPolicy    BackupPolicy    `json:"backupPolicy"`
}

// ThumbnailSize defines thumbnail dimensions
type ThumbnailSize struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// RetentionPolicy defines file retention rules
type RetentionPolicy struct {
	KeepImagesForever bool   `json:"keepImagesForever"`
	MaxAge            int    `json:"maxAge"`          // in days
	CleanupSchedule   string `json:"cleanupSchedule"` // cron expression
}

// BackupPolicy defines backup rules
type BackupPolicy struct {
	Enabled          bool   `json:"enabled"`
	BackupFrequency  string `json:"backupFrequency"` // daily, weekly, monthly
	RetentionDays    int    `json:"retentionDays"`
	CompressionLevel int    `json:"compressionLevel"`
}

// ImageUploadRequest represents an image upload request
type ImageUploadRequest struct {
	File        io.Reader              `json:"-"`
	FileName    string                 `json:"fileName"`
	ContentType string                 `json:"contentType"`
	Size        int64                  `json:"size"`
	ImageType   string                 `json:"imageType"` // user, cloth, result
	OwnerID     string                 `json:"ownerId"`
	IsPublic    bool                   `json:"isPublic"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ImageUploadResponse represents the response after image upload
type ImageUploadResponse struct {
	ImageID      string    `json:"imageId"`
	FilePath     string    `json:"filePath"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	OriginalURL  string    `json:"originalUrl"`
	FileSize     int64     `json:"fileSize"`
	Checksum     string    `json:"checksum"`
	UploadedAt   time.Time `json:"uploadedAt"`
}

// ImageAccessRequest represents a request to access an image
type ImageAccessRequest struct {
	ImageID     string `json:"imageId"`
	AccessType  string `json:"accessType"` // view, download
	TTL         int64  `json:"ttl"`        // time to live in seconds
	RequesterID string `json:"requesterId"`
}

// ImageAccessResponse represents the response for image access
type ImageAccessResponse struct {
	SignedURL  string    `json:"signedUrl"`
	ExpiresAt  time.Time `json:"expiresAt"`
	AccessType string    `json:"accessType"`
	ImageInfo  *FileInfo `json:"imageInfo"`
}

// ImageMetadata represents comprehensive image metadata
type ImageMetadata struct {
	ImageID       string                 `json:"imageId"`
	FilePath      string                 `json:"filePath"`
	FileName      string                 `json:"fileName"`
	FileSize      int64                  `json:"fileSize"`
	MimeType      string                 `json:"mimeType"`
	Width         int                    `json:"width"`
	Height        int                    `json:"height"`
	Checksum      string                 `json:"checksum"`
	ImageType     string                 `json:"imageType"`
	OwnerID       string                 `json:"ownerId"`
	IsPublic      bool                   `json:"isPublic"`
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	LastAccessed  time.Time              `json:"lastAccessed"`
	AccessCount   int64                  `json:"accessCount"`
	IsBackedUp    bool                   `json:"isBackedUp"`
	BackupPath    string                 `json:"backupPath,omitempty"`
	ThumbnailPath string                 `json:"thumbnailPath,omitempty"`
}

// StorageQuota represents storage quota information
type StorageQuota struct {
	UserID       string    `json:"userId"`
	TotalUsed    int64     `json:"totalUsed"`
	TotalLimit   int64     `json:"totalLimit"`
	UserImages   int64     `json:"userImages"`
	ClothImages  int64     `json:"clothImages"`
	ResultImages int64     `json:"resultImages"`
	UserLimit    int64     `json:"userLimit"`
	ClothLimit   int64     `json:"clothLimit"`
	ResultLimit  int64     `json:"resultLimit"`
	UsagePercent float64   `json:"usagePercent"`
	LastUpdated  time.Time `json:"lastUpdated"`
}

// StorageHealth represents storage system health status
type StorageHealth struct {
	Status       string    `json:"status"` // healthy, warning, critical
	DiskUsage    float64   `json:"diskUsage"`
	FreeSpace    int64     `json:"freeSpace"`
	TotalSpace   int64     `json:"totalSpace"`
	BackupStatus string    `json:"backupStatus"`
	LastBackup   time.Time `json:"lastBackup"`
	ErrorCount   int       `json:"errorCount"`
	LastError    string    `json:"lastError,omitempty"`
	CheckedAt    time.Time `json:"checkedAt"`
}

// ImageSearchRequest represents a request to search images
type ImageSearchRequest struct {
	Query     string    `json:"query"`
	ImageType string    `json:"imageType,omitempty"`
	OwnerID   string    `json:"ownerId,omitempty"`
	IsPublic  *bool     `json:"isPublic,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	DateFrom  time.Time `json:"dateFrom,omitempty"`
	DateTo    time.Time `json:"dateTo,omitempty"`
	MinSize   int64     `json:"minSize,omitempty"`
	MaxSize   int64     `json:"maxSize,omitempty"`
	MimeType  string    `json:"mimeType,omitempty"`
	Page      int       `json:"page"`
	PageSize  int       `json:"pageSize"`
	SortBy    string    `json:"sortBy"`    // name, size, date, access_count
	SortOrder string    `json:"sortOrder"` // asc, desc
}

// ImageSearchResponse represents the response for image search
type ImageSearchResponse struct {
	Images     []ImageMetadata `json:"images"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
	Query      string          `json:"query"`
}

// ImageBatchOperation represents a batch operation on images
type ImageBatchOperation struct {
	Operation  string                 `json:"operation"` // delete, move, copy, backup
	ImageIDs   []string               `json:"imageIds"`
	TargetPath string                 `json:"targetPath,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// ImageBatchResponse represents the response for batch operations
type ImageBatchResponse struct {
	SuccessCount int           `json:"successCount"`
	FailureCount int           `json:"failureCount"`
	Errors       []string      `json:"errors,omitempty"`
	Results      []BatchResult `json:"results"`
}

// BatchResult represents the result of a single batch operation
type BatchResult struct {
	ImageID string `json:"imageId"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// StorageEvent represents a storage-related event
type StorageEvent struct {
	EventType string                 `json:"eventType"` // upload, delete, access, backup, cleanup
	ImageID   string                 `json:"imageId"`
	FilePath  string                 `json:"filePath"`
	UserID    string                 `json:"userId,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// StorageMetrics represents storage performance metrics
type StorageMetrics struct {
	UploadCount         int64     `json:"uploadCount"`
	DownloadCount       int64     `json:"downloadCount"`
	DeleteCount         int64     `json:"deleteCount"`
	BackupCount         int64     `json:"backupCount"`
	AverageUploadTime   float64   `json:"averageUploadTime"`
	AverageDownloadTime float64   `json:"averageDownloadTime"`
	ErrorRate           float64   `json:"errorRate"`
	StorageEfficiency   float64   `json:"storageEfficiency"`
	LastCalculated      time.Time `json:"lastCalculated"`
}

// Constants for storage operations
const (
	// Image types
	ImageTypeUser   = "user"
	ImageTypeCloth  = "cloth"
	ImageTypeResult = "result"

	// Access types
	AccessTypeView     = "view"
	AccessTypeDownload = "download"

	// Storage paths
	StoragePathUsers   = "images/user"
	StoragePathCloth   = "images/cloth"
	StoragePathResults = "images/result"
	StoragePathBackups = "backups"

	// Default limits
	DefaultMaxFileSize = 50 * 1024 * 1024 // 50MB
	DefaultUserLimit   = 100
	DefaultClothLimit  = 1000
	DefaultResultLimit = 500

	// Default TTL for signed URLs (1 hour)
	DefaultSignedURLTTL = 3600

	// Health status
	HealthStatusHealthy  = "healthy"
	HealthStatusWarning  = "warning"
	HealthStatusCritical = "critical"

	// Backup status
	BackupStatusSuccess = "success"
	BackupStatusFailed  = "failed"
	BackupStatusPending = "pending"
)

// Supported image MIME types
var SupportedImageTypes = []string{
	"image/jpeg",
	"image/jpg",
	"image/png",
	"image/gif",
	"image/webp",
	"image/svg+xml",
	"image/bmp",
	"image/tiff",
}

// Default thumbnail sizes
var DefaultThumbnailSizes = []ThumbnailSize{
	{Name: "small", Width: 150, Height: 150},
	{Name: "medium", Width: 300, Height: 300},
	{Name: "large", Width: 600, Height: 600},
}

// Default retention policy (keep images forever)
var DefaultRetentionPolicy = RetentionPolicy{
	KeepImagesForever: true,
	MaxAge:            0,
	CleanupSchedule:   "0 2 * * *", // Daily at 2 AM
}

// Default backup policy
var DefaultBackupPolicy = BackupPolicy{
	Enabled:          true,
	BackupFrequency:  "daily",
	RetentionDays:    365, // Keep backups for 1 year
	CompressionLevel: 6,
}
