package share

import (
	"context"
	"time"
)

// Store defines the interface for share data operations
type Store interface {
	// Shared link operations
	CreateSharedLink(ctx context.Context, conversionID, userID, shareToken, signedURL string, expiresAt time.Time, maxAccessCount *int) (string, error)
	GetSharedLink(ctx context.Context, shareID string) (SharedLink, error)
	GetSharedLinkByToken(ctx context.Context, shareToken string) (ActiveSharedLink, error)
	UpdateSharedLink(ctx context.Context, shareID string, updates map[string]interface{}) error
	DeactivateSharedLink(ctx context.Context, shareID, userID string) error
	ListUserSharedLinks(ctx context.Context, userID string, limit, offset int) ([]ActiveSharedLink, error)

	// Access log operations
	LogSharedLinkAccess(ctx context.Context, sharedLinkID string, req AccessShareRequest, success bool, errorMessage string) error

	// Statistics operations
	GetSharedLinkStats(ctx context.Context, userID, conversionID string) (SharedLinkStats, error)

	// Cleanup operations
	CleanupExpiredLinks(ctx context.Context) (int, error)

	// Additional operations
	ValidateSharedLinkAccess(ctx context.Context, shareToken string) (bool, error)
	GetSharedLinkDetails(ctx context.Context, shareToken string) (SharedLinkDetails, error)
	GetSharedLinkAccessLogs(ctx context.Context, shareID string, limit, offset int) ([]AccessLog, error)
	GetPopularSharedLinks(ctx context.Context, limit int) ([]PopularSharedLink, error)
}

// ConversionService defines the interface for conversion operations
type ConversionService interface {
	GetConversion(ctx context.Context, conversionID, userID string) (ConversionResponse, error)
	ValidateConversionOwnership(ctx context.Context, conversionID, userID string) error
}

// ConversionResponse represents a conversion response from the conversion service
type ConversionResponse struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	UserImageID      string     `json:"userImageId"`
	ClothImageID     string     `json:"clothImageId"`
	Status           string     `json:"status"`
	ResultImageID    *string    `json:"resultImageId,omitempty"`
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int       `json:"processingTimeMs,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// ImageService defines the interface for image operations
type ImageService interface {
	GetImage(ctx context.Context, imageID string) (ImageResponse, error)
	GenerateSignedURL(ctx context.Context, imageID string, accessType string, ttl int64) (string, error)
}

// ImageResponse represents an image response from the image service
type ImageResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	VendorID     string    `json:"vendorId"`
	Type         string    `json:"type"`
	FileName     string    `json:"fileName"`
	OriginalURL  string    `json:"originalUrl"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	FileSize     int64     `json:"fileSize"`
	MimeType     string    `json:"mimeType"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	IsPublic     bool      `json:"isPublic"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// NotificationService defines the interface for notification operations
type NotificationService interface {
	SendShareCreated(ctx context.Context, userID, shareID, shareToken string) error
	SendShareAccessed(ctx context.Context, userID, shareID string, accessCount int) error
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogShareCreated(ctx context.Context, userID, conversionID, shareID string) error
	LogShareAccessed(ctx context.Context, shareID, ipAddress, userAgent string) error
	LogShareDeactivated(ctx context.Context, userID, shareID string) error
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	RecordShareCreated(ctx context.Context, userID, conversionID string) error
	RecordShareAccessed(ctx context.Context, shareID string, success bool) error
	RecordShareExpired(ctx context.Context, shareID string) error
}
