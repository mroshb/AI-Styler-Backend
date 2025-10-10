package conversion

import (
	"time"
)

// Conversion represents a conversion request
type Conversion struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	UserImageID      string     `json:"userImageId"`
	ClothImageID     string     `json:"clothImageId"`
	Status           string     `json:"status"` // "pending", "processing", "completed", "failed"
	ResultImageID    *string    `json:"resultImageId,omitempty"`
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int       `json:"processingTimeMs,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// ConversionRequest represents the request to create a new conversion
type ConversionRequest struct {
	UserImageID  string `json:"userImageId" binding:"required"`
	ClothImageID string `json:"clothImageId" binding:"required"`
}

// ConversionResponse represents the response for conversion operations
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
	UserImageURL     string     `json:"userImageUrl,omitempty"`
	ClothImageURL    string     `json:"clothImageUrl,omitempty"`
	ResultImageURL   string     `json:"resultImageUrl,omitempty"`
}

// ConversionListRequest represents the request to list conversions
type ConversionListRequest struct {
	Page      int       `json:"page" form:"page"`
	PageSize  int       `json:"pageSize" form:"pageSize"`
	Status    string    `json:"status" form:"status"`
	UserID    string    `json:"userId" form:"userId"`
	StartDate time.Time `json:"startDate" form:"startDate"`
	EndDate   time.Time `json:"endDate" form:"endDate"`
}

// ConversionListResponse represents the response for conversion listing
type ConversionListResponse struct {
	Conversions []ConversionResponse `json:"conversions"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"pageSize"`
	TotalPages  int                  `json:"totalPages"`
}

// UpdateConversionRequest represents the request to update a conversion
type UpdateConversionRequest struct {
	Status           *string `json:"status,omitempty"`
	ResultImageID    *string `json:"resultImageId,omitempty"`
	ErrorMessage     *string `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int    `json:"processingTimeMs,omitempty"`
}

// QuotaCheck represents the result of a quota check
type QuotaCheck struct {
	CanConvert     bool   `json:"canConvert"`
	RemainingFree  int    `json:"remainingFree"`
	RemainingPaid  int    `json:"remainingPaid"`
	TotalRemaining int    `json:"totalRemaining"`
	PlanName       string `json:"planName"`
	MonthlyLimit   int    `json:"monthlyLimit"`
}

// Conversion status constants
const (
	ConversionStatusPending    = "pending"
	ConversionStatusProcessing = "processing"
	ConversionStatusCompleted  = "completed"
	ConversionStatusFailed     = "failed"
)

// Conversion type constants
const (
	ConversionTypeFree = "free"
	ConversionTypePaid = "paid"
)

// Default values
const (
	DefaultFreeConversionsLimit = 2
	DefaultPageSize             = 20
	MaxPageSize                 = 100
)

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}
