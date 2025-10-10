package image

import (
	"io"
	"time"
)

// ImageType represents the type of image (user, vendor, result)
type ImageType string

const (
	ImageTypeUser   ImageType = "user"
	ImageTypeVendor ImageType = "vendor"
	ImageTypeResult ImageType = "result"
)

// Image represents an uploaded image
type Image struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"userId,omitempty"`   // For user images
	VendorID     *string                `json:"vendorId,omitempty"` // For vendor images
	Type         ImageType              `json:"type"`
	FileName     string                 `json:"fileName"`
	OriginalURL  string                 `json:"originalUrl"`
	ThumbnailURL *string                `json:"thumbnailUrl,omitempty"`
	FileSize     int64                  `json:"fileSize"`
	MimeType     string                 `json:"mimeType"`
	Width        *int                   `json:"width,omitempty"`
	Height       *int                   `json:"height,omitempty"`
	IsPublic     bool                   `json:"isPublic"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// ImageUsageHistory represents the usage history of an image
type ImageUsageHistory struct {
	ID        string                 `json:"id"`
	ImageID   string                 `json:"imageId"`
	UserID    *string                `json:"userId,omitempty"`
	Action    string                 `json:"action"` // upload, view, download, delete, etc.
	IPAddress *string                `json:"ipAddress,omitempty"`
	UserAgent *string                `json:"userAgent,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
}

// UploadImageRequest represents the request to upload an image
type UploadImageRequest struct {
	Type     ImageType              `json:"type" binding:"required"`
	FileName string                 `json:"fileName" binding:"required"`
	FileSize int64                  `json:"fileSize" binding:"required"`
	MimeType string                 `json:"mimeType" binding:"required"`
	Width    *int                   `json:"width,omitempty"`
	Height   *int                   `json:"height,omitempty"`
	IsPublic bool                   `json:"isPublic"`
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	File     io.Reader              `json:"-"` // File data reader
}

// UpdateImageRequest represents the request to update an image
type UpdateImageRequest struct {
	IsPublic *bool                  `json:"isPublic,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ImageListRequest represents the request to list images
type ImageListRequest struct {
	Page     int        `json:"page" form:"page"`
	PageSize int        `json:"pageSize" form:"pageSize"`
	Type     *ImageType `json:"type" form:"type"`
	IsPublic *bool      `json:"isPublic" form:"isPublic"`
	Tags     []string   `json:"tags" form:"tags"`
	UserID   *string    `json:"userId" form:"userId"`
	VendorID *string    `json:"vendorId" form:"vendorId"`
}

// ImageListResponse represents the response for image listing
type ImageListResponse struct {
	Images     []Image `json:"images"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

// ImageUsageHistoryRequest represents the request to get image usage history
type ImageUsageHistoryRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Action   string `json:"action" form:"action"`
}

// ImageUsageHistoryResponse represents the response for image usage history
type ImageUsageHistoryResponse struct {
	History    []ImageUsageHistory `json:"history"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	TotalPages int                 `json:"totalPages"`
}

// SignedURLResponse represents the response for signed URL generation
type SignedURLResponse struct {
	URL        string    `json:"url"`
	ExpiresAt  time.Time `json:"expiresAt"`
	ImageID    string    `json:"imageId"`
	AccessType string    `json:"accessType"` // view, download
}

// ImageStats represents image statistics
type ImageStats struct {
	TotalImages      int   `json:"totalImages"`
	UserImages       int   `json:"userImages"`
	VendorImages     int   `json:"vendorImages"`
	ResultImages     int   `json:"resultImages"`
	PublicImages     int   `json:"publicImages"`
	PrivateImages    int   `json:"privateImages"`
	TotalFileSize    int64 `json:"totalFileSize"`
	TotalSizeBytes   int64 `json:"totalSizeBytes"`
	AverageFileSize  int64 `json:"averageFileSize"`
	ImagesLast30Days int   `json:"imagesLast30Days"`
}

// Constants
const (
	// Image types
	ImageTypeUserConst   = "user"
	ImageTypeVendorConst = "vendor"
	ImageTypeResultConst = "result"

	// Usage actions
	ActionUpload   = "upload"
	ActionView     = "view"
	ActionDownload = "download"
	ActionDelete   = "delete"
	ActionUpdate   = "update"

	// Access types
	AccessTypeView     = "view"
	AccessTypeDownload = "download"
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

// Maximum file size (50MB for general images, 10MB for user uploads)
const (
	MaxImageFileSize     = 50 * 1024 * 1024 // 50MB
	MaxUserImageFileSize = 10 * 1024 * 1024 // 10MB
)

// Image storage paths
const (
	StoragePathUsers   = "users"
	StoragePathVendors = "vendors"
	StoragePathResults = "results"
)

// Helper function for creating bool pointers
func boolPtr(b bool) *bool {
	return &b
}
