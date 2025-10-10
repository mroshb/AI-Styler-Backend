package share

import (
	"time"
)

// SharedLink represents a shared conversion result
type SharedLink struct {
	ID             string    `json:"id"`
	ConversionID   string    `json:"conversionId"`
	UserID         string    `json:"userId"`
	ShareToken     string    `json:"shareToken"`
	SignedURL      string    `json:"signedUrl"`
	ExpiresAt      time.Time `json:"expiresAt"`
	AccessCount    int       `json:"accessCount"`
	MaxAccessCount *int      `json:"maxAccessCount,omitempty"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// CreateShareRequest represents the request to create a shared link
type CreateShareRequest struct {
	ConversionID   string `json:"conversionId" binding:"required"`
	ExpiryMinutes  int    `json:"expiryMinutes" binding:"min=1,max=5"`
	MaxAccessCount *int   `json:"maxAccessCount,omitempty"`
}

// CreateShareResponse represents the response for creating a shared link
type CreateShareResponse struct {
	ShareID    string    `json:"shareId"`
	ShareToken string    `json:"shareToken"`
	SignedURL  string    `json:"signedUrl"`
	ExpiresAt  time.Time `json:"expiresAt"`
	PublicURL  string    `json:"publicUrl"` // The public URL that users can access
}

// AccessShareRequest represents the request to access a shared link
type AccessShareRequest struct {
	ShareToken string `json:"shareToken" binding:"required"`
	AccessType string `json:"accessType" binding:"oneof=view download"`
	IPAddress  string `json:"ipAddress,omitempty"`
	UserAgent  string `json:"userAgent,omitempty"`
	Referer    string `json:"referer,omitempty"`
}

// AccessShareResponse represents the response for accessing a shared link
type AccessShareResponse struct {
	Success            bool   `json:"success"`
	ConversionID       string `json:"conversionId,omitempty"`
	ResultImageURL     string `json:"resultImageUrl,omitempty"`
	ErrorMessage       string `json:"errorMessage,omitempty"`
	AccessCount        int    `json:"accessCount,omitempty"`
	SecondsUntilExpiry int    `json:"secondsUntilExpiry,omitempty"`
}

// SharedLinkAccessLog represents an access log entry
type SharedLinkAccessLog struct {
	ID           string                 `json:"id"`
	SharedLinkID string                 `json:"sharedLinkId"`
	IPAddress    string                 `json:"ipAddress,omitempty"`
	UserAgent    string                 `json:"userAgent,omitempty"`
	Referer      string                 `json:"referer,omitempty"`
	AccessType   string                 `json:"accessType"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
}

// SharedLinkStats represents statistics for shared links
type SharedLinkStats struct {
	TotalSharedLinks  int   `json:"totalSharedLinks"`
	ActiveSharedLinks int   `json:"activeSharedLinks"`
	ExpiredLinks      int   `json:"expiredLinks"`
	TotalAccessCount  int64 `json:"totalAccessCount"`
	UniqueAccessCount int64 `json:"uniqueAccessCount"`
	UniqueIPAddresses int64 `json:"uniqueIpAddresses"`
}

// ActiveSharedLink represents an active shared link with conversion details
type ActiveSharedLink struct {
	ID                  string    `json:"id"`
	ConversionID        string    `json:"conversionId"`
	UserID              string    `json:"userId"`
	ShareToken          string    `json:"shareToken"`
	SignedURL           string    `json:"signedUrl"`
	ExpiresAt           time.Time `json:"expiresAt"`
	AccessCount         int       `json:"accessCount"`
	MaxAccessCount      *int      `json:"maxAccessCount,omitempty"`
	IsActive            bool      `json:"isActive"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
	ConversionStatus    string    `json:"conversionStatus"`
	ResultImageID       string    `json:"resultImageId"`
	ResultImageURL      string    `json:"resultImageUrl"`
	ResultImageName     string    `json:"resultImageName"`
	ResultImageSize     int64     `json:"resultImageSize"`
	ResultImageMimeType string    `json:"resultImageMimeType"`
	SecondsUntilExpiry  int       `json:"secondsUntilExpiry"`
}

// SharedLinkDetails represents detailed information about a shared link
type SharedLinkDetails struct {
	ID               string    `json:"id"`
	ConversionID     string    `json:"conversionId"`
	UserID           string    `json:"userId"`
	ShareToken       string    `json:"shareToken"`
	SignedURL        string    `json:"signedUrl"`
	ExpiresAt        time.Time `json:"expiresAt"`
	MaxAccessCount   *int      `json:"maxAccessCount,omitempty"`
	AccessCount      int       `json:"accessCount"`
	IsActive         bool      `json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ConversionStatus string    `json:"conversionStatus"`
	UserName         string    `json:"userName"`
}

// AccessLog represents an access log entry
type AccessLog struct {
	ID           string    `json:"id"`
	SharedLinkID string    `json:"sharedLinkId"`
	IPAddress    string    `json:"ipAddress,omitempty"`
	UserAgent    string    `json:"userAgent,omitempty"`
	AccessedAt   time.Time `json:"accessedAt"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}

// PopularSharedLink represents a popular shared link
type PopularSharedLink struct {
	ID               string    `json:"id"`
	ConversionID     string    `json:"conversionId"`
	UserID           string    `json:"userId"`
	ShareToken       string    `json:"shareToken"`
	AccessCount      int       `json:"accessCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UserName         string    `json:"userName"`
	ConversionStatus string    `json:"conversionStatus"`
}

// Share service constants
const (
	MinExpiryMinutes     = 1
	MaxExpiryMinutes     = 5
	DefaultExpiryMinutes = 5

	AccessTypeView     = "view"
	AccessTypeDownload = "download"

	ShareTokenLength = 32 // Base64 encoded, so actual token is longer
)

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
