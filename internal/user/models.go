package user

import (
	"time"
)

// UserProfile represents a user's profile information
type UserProfile struct {
	ID                   string     `json:"id"`
	Phone                string     `json:"phone"`
	Name                 *string    `json:"name,omitempty"`
	AvatarURL            *string    `json:"avatarUrl,omitempty"`
	Bio                  *string    `json:"bio,omitempty"`
	Role                 string     `json:"role"`
	IsPhoneVerified      bool       `json:"isPhoneVerified"`
	IsActive             bool       `json:"isActive"`
	LastLoginAt          *time.Time `json:"lastLoginAt,omitempty"`
	FreeConversionsUsed  int        `json:"freeConversionsUsed"`
	FreeConversionsLimit int        `json:"freeConversionsLimit"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

// UserConversion represents a conversion activity
type UserConversion struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	ConversionType   string     `json:"conversionType"` // "free" or "paid"
	InputFileURL     string     `json:"inputFileUrl"`
	OutputFileURL    *string    `json:"outputFileUrl,omitempty"`
	StyleName        string     `json:"styleName"`
	Status           string     `json:"status"` // "pending", "processing", "completed", "failed"
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int       `json:"processingTimeMs,omitempty"`
	FileSizeBytes    *int64     `json:"fileSizeBytes,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// UserPlan represents a user's subscription plan
type UserPlan struct {
	ID                       string     `json:"id"`
	PlanID                   string     `json:"planId"`
	UserID                   string     `json:"userId"`
	PlanName                 string     `json:"planName"` // "free", "basic", "premium", "enterprise"
	DisplayName              string     `json:"displayName"`
	Description              string     `json:"description"`
	Features                 []string   `json:"features"`
	Status                   string     `json:"status"` // "active", "cancelled", "expired", "suspended"
	MonthlyConversionsLimit  int        `json:"monthlyConversionsLimit"`
	ConversionsUsedThisMonth int        `json:"conversionsUsedThisMonth"`
	PricePerMonthCents       int        `json:"pricePerMonthCents"`
	BillingCycleStartDate    *time.Time `json:"billingCycleStartDate,omitempty"`
	BillingCycleEndDate      *time.Time `json:"billingCycleEndDate,omitempty"`
	AutoRenew                bool       `json:"autoRenew"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
	ExpiresAt                *time.Time `json:"expiresAt,omitempty"`
}

// ConversionQuota represents monthly conversion usage
type ConversionQuota struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"userId"`
	YearMonth            string    `json:"yearMonth"` // Format: YYYY-MM
	FreeConversionsUsed  int       `json:"freeConversionsUsed"`
	PaidConversionsUsed  int       `json:"paidConversionsUsed"`
	TotalConversionsUsed int       `json:"totalConversionsUsed"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// QuotaStatus represents current user quota information
type QuotaStatus struct {
	FreeConversionsRemaining  int    `json:"freeConversionsRemaining"`
	PaidConversionsRemaining  int    `json:"paidConversionsRemaining"`
	TotalConversionsRemaining int    `json:"totalConversionsRemaining"`
	PlanName                  string `json:"planName"`
	MonthlyLimit              int    `json:"monthlyLimit"`
}

// UpdateProfileRequest represents the request to update user profile
type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
	Bio       *string `json:"bio,omitempty"`
}

// ConversionHistoryRequest represents the request to get conversion history
type ConversionHistoryRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Status   string `json:"status" form:"status"`
	Type     string `json:"type" form:"type"`
}

// ConversionHistoryResponse represents the response for conversion history
type ConversionHistoryResponse struct {
	Conversions []UserConversion `json:"conversions"`
	Total       int              `json:"total"`
	Page        int              `json:"page"`
	PageSize    int              `json:"pageSize"`
	TotalPages  int              `json:"totalPages"`
}

// CreateConversionRequest represents the request to create a new conversion
type CreateConversionRequest struct {
	InputFileURL string `json:"inputFileUrl" binding:"required"`
	StyleName    string `json:"styleName" binding:"required"`
	Type         string `json:"type" binding:"required,oneof=free paid"`
}

// UpdateConversionRequest represents the request to update a conversion
type UpdateConversionRequest struct {
	OutputFileURL    *string `json:"outputFileUrl,omitempty"`
	Status           *string `json:"status,omitempty"`
	ErrorMessage     *string `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int    `json:"processingTimeMs,omitempty"`
	FileSizeBytes    *int64  `json:"fileSizeBytes,omitempty"`
}

// Plan constants
const (
	PlanFree       = "free"
	PlanBasic      = "basic"
	PlanPremium    = "premium"
	PlanEnterprise = "enterprise"
)

// Conversion type constants
const (
	ConversionTypeFree = "free"
	ConversionTypePaid = "paid"
)

// Conversion status constants
const (
	ConversionStatusPending    = "pending"
	ConversionStatusProcessing = "processing"
	ConversionStatusCompleted  = "completed"
	ConversionStatusFailed     = "failed"
)

// Plan status constants
const (
	PlanStatusActive    = "active"
	PlanStatusCancelled = "cancelled"
	PlanStatusExpired   = "expired"
	PlanStatusSuspended = "suspended"
)
