package admin

import (
	"time"
)

// AdminUser represents a user from admin perspective
type AdminUser struct {
	ID                   string     `json:"id"`
	Phone                string     `json:"phone"`
	Name                 *string    `json:"name,omitempty"`
	AvatarURL            *string    `json:"avatarUrl,omitempty"`
	Bio                  *string    `json:"bio,omitempty"`
	Role                 string     `json:"role"`
	IsPhoneVerified      bool       `json:"isPhoneVerified"`
	FreeConversionsUsed  int        `json:"freeConversionsUsed"`
	FreeConversionsLimit int        `json:"freeConversionsLimit"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	LastLoginAt          *time.Time `json:"lastLoginAt,omitempty"`
	IsActive             bool       `json:"isActive"`
}

// AdminVendor represents a vendor from admin perspective
type AdminVendor struct {
	ID              string      `json:"id"`
	UserID          string      `json:"userId"`
	BusinessName    string      `json:"businessName"`
	AvatarURL       *string     `json:"avatarUrl,omitempty"`
	Bio             *string     `json:"bio,omitempty"`
	ContactInfo     ContactInfo `json:"contactInfo"`
	SocialLinks     SocialLinks `json:"socialLinks"`
	IsVerified      bool        `json:"isVerified"`
	IsActive        bool        `json:"isActive"`
	FreeImagesUsed  int         `json:"freeImagesUsed"`
	FreeImagesLimit int         `json:"freeImagesLimit"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	LastLoginAt     *time.Time  `json:"lastLoginAt,omitempty"`
}

// ContactInfo represents vendor contact information
type ContactInfo struct {
	Email      *string `json:"email,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Website    *string `json:"website,omitempty"`
	Address    *string `json:"address,omitempty"`
	City       *string `json:"city,omitempty"`
	Country    *string `json:"country,omitempty"`
	PostalCode *string `json:"postalCode,omitempty"`
}

// SocialLinks represents vendor social media links
type SocialLinks struct {
	Instagram *string `json:"instagram,omitempty"`
	Facebook  *string `json:"facebook,omitempty"`
	Twitter   *string `json:"twitter,omitempty"`
	LinkedIn  *string `json:"linkedin,omitempty"`
	YouTube   *string `json:"youtube,omitempty"`
	TikTok    *string `json:"tiktok,omitempty"`
}

// AdminPlan represents a subscription plan from admin perspective
type AdminPlan struct {
	ID                      string    `json:"id"`
	Name                    string    `json:"name"`
	DisplayName             string    `json:"displayName"`
	Description             string    `json:"description"`
	PricePerMonthCents      int64     `json:"pricePerMonthCents"`
	MonthlyConversionsLimit int       `json:"monthlyConversionsLimit"`
	Features                []string  `json:"features"`
	IsActive                bool      `json:"isActive"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
	SubscriberCount         int       `json:"subscriberCount"`
}

// AdminPayment represents a payment from admin perspective
type AdminPayment struct {
	ID                string     `json:"id"`
	UserID            string     `json:"userId"`
	UserPhone         string     `json:"userPhone"`
	PlanID            string     `json:"planId"`
	PlanName          string     `json:"planName"`
	Amount            int64      `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	PaymentMethod     string     `json:"paymentMethod"`
	Gateway           string     `json:"gateway"`
	GatewayTrackID    *string    `json:"gatewayTrackId,omitempty"`
	GatewayRefNumber  *string    `json:"gatewayRefNumber,omitempty"`
	GatewayCardNumber *string    `json:"gatewayCardNumber,omitempty"`
	Description       string     `json:"description"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	PaidAt            *time.Time `json:"paidAt,omitempty"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
}

// AdminConversion represents a conversion from admin perspective
type AdminConversion struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	UserPhone        string     `json:"userPhone"`
	ConversionType   string     `json:"conversionType"`
	InputFileURL     string     `json:"inputFileUrl"`
	OutputFileURL    *string    `json:"outputFileUrl,omitempty"`
	StyleName        string     `json:"styleName"`
	Status           string     `json:"status"`
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int       `json:"processingTimeMs,omitempty"`
	FileSizeBytes    *int64     `json:"fileSizeBytes,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// AdminImage represents a vendor image from admin perspective
type AdminImage struct {
	ID           string    `json:"id"`
	VendorID     string    `json:"vendorId"`
	VendorName   string    `json:"vendorName"`
	AlbumID      *string   `json:"albumId,omitempty"`
	AlbumName    *string   `json:"albumName,omitempty"`
	FileName     string    `json:"fileName"`
	OriginalURL  string    `json:"originalUrl"`
	ThumbnailURL *string   `json:"thumbnailUrl,omitempty"`
	FileSize     int64     `json:"fileSize"`
	MimeType     string    `json:"mimeType"`
	Width        *int      `json:"width,omitempty"`
	Height       *int      `json:"height,omitempty"`
	IsFree       bool      `json:"isFree"`
	IsPublic     bool      `json:"isPublic"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID         string                 `json:"id"`
	UserID     *string                `json:"userId,omitempty"`
	ActorType  string                 `json:"actorType"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID *string                `json:"resourceId,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"createdAt"`
}

// AdminStats represents system statistics
type AdminStats struct {
	TotalUsers         int   `json:"totalUsers"`
	ActiveUsers        int   `json:"activeUsers"`
	TotalVendors       int   `json:"totalVendors"`
	ActiveVendors      int   `json:"activeVendors"`
	TotalConversions   int   `json:"totalConversions"`
	TotalPayments      int   `json:"totalPayments"`
	TotalRevenue       int64 `json:"totalRevenue"`
	TotalImages        int   `json:"totalImages"`
	PendingConversions int   `json:"pendingConversions"`
	FailedConversions  int   `json:"failedConversions"`
}

// UserListRequest represents the request to list users
type UserListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Role     string `json:"role" form:"role"`
	Search   string `json:"search" form:"search"`
	IsActive *bool  `json:"isActive" form:"isActive"`
}

// UserListResponse represents the response for user listing
type UserListResponse struct {
	Users      []AdminUser `json:"users"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// VendorListRequest represents the request to list vendors
type VendorListRequest struct {
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"pageSize" form:"pageSize"`
	Search     string `json:"search" form:"search"`
	IsActive   *bool  `json:"isActive" form:"isActive"`
	IsVerified *bool  `json:"isVerified" form:"isVerified"`
}

// VendorListResponse represents the response for vendor listing
type VendorListResponse struct {
	Vendors    []AdminVendor `json:"vendors"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// PlanListRequest represents the request to list plans
type PlanListRequest struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"pageSize" form:"pageSize"`
	IsActive *bool `json:"isActive" form:"isActive"`
}

// PlanListResponse represents the response for plan listing
type PlanListResponse struct {
	Plans      []AdminPlan `json:"plans"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// PaymentListRequest represents the request to list payments
type PaymentListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Status   string `json:"status" form:"status"`
	UserID   string `json:"userId" form:"userId"`
	PlanID   string `json:"planId" form:"planId"`
	DateFrom string `json:"dateFrom" form:"dateFrom"`
	DateTo   string `json:"dateTo" form:"dateTo"`
}

// PaymentListResponse represents the response for payment listing
type PaymentListResponse struct {
	Payments   []AdminPayment `json:"payments"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalPages int            `json:"totalPages"`
}

// ConversionListRequest represents the request to list conversions
type ConversionListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Status   string `json:"status" form:"status"`
	UserID   string `json:"userId" form:"userId"`
	Type     string `json:"type" form:"type"`
	DateFrom string `json:"dateFrom" form:"dateFrom"`
	DateTo   string `json:"dateTo" form:"dateTo"`
}

// ConversionListResponse represents the response for conversion listing
type ConversionListResponse struct {
	Conversions []AdminConversion `json:"conversions"`
	Total       int               `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"pageSize"`
	TotalPages  int               `json:"totalPages"`
}

// ImageListRequest represents the request to list images
type ImageListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	VendorID string `json:"vendorId" form:"vendorId"`
	IsPublic *bool  `json:"isPublic" form:"isPublic"`
	IsFree   *bool  `json:"isFree" form:"isFree"`
	DateFrom string `json:"dateFrom" form:"dateFrom"`
	DateTo   string `json:"dateTo" form:"dateTo"`
}

// ImageListResponse represents the response for image listing
type ImageListResponse struct {
	Images     []AdminImage `json:"images"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalPages int          `json:"totalPages"`
}

// AuditLogListRequest represents the request to list audit logs
type AuditLogListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	UserID   string `json:"userId" form:"userId"`
	Action   string `json:"action" form:"action"`
	Resource string `json:"resource" form:"resource"`
	DateFrom string `json:"dateFrom" form:"dateFrom"`
	DateTo   string `json:"dateTo" form:"dateTo"`
}

// AuditLogListResponse represents the response for audit log listing
type AuditLogListResponse struct {
	AuditLogs  []AuditLog `json:"auditLogs"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Name                 *string `json:"name,omitempty"`
	AvatarURL            *string `json:"avatarUrl,omitempty"`
	Bio                  *string `json:"bio,omitempty"`
	Role                 *string `json:"role,omitempty"`
	IsPhoneVerified      *bool   `json:"isPhoneVerified,omitempty"`
	FreeConversionsLimit *int    `json:"freeConversionsLimit,omitempty"`
	IsActive             *bool   `json:"isActive,omitempty"`
}

// UpdateVendorRequest represents the request to update a vendor
type UpdateVendorRequest struct {
	BusinessName    *string      `json:"businessName,omitempty"`
	AvatarURL       *string      `json:"avatarUrl,omitempty"`
	Bio             *string      `json:"bio,omitempty"`
	ContactInfo     *ContactInfo `json:"contactInfo,omitempty"`
	SocialLinks     *SocialLinks `json:"socialLinks,omitempty"`
	IsVerified      *bool        `json:"isVerified,omitempty"`
	IsActive        *bool        `json:"isActive,omitempty"`
	FreeImagesLimit *int         `json:"freeImagesLimit,omitempty"`
}

// CreatePlanRequest represents the request to create a plan
type CreatePlanRequest struct {
	Name                    string   `json:"name" binding:"required"`
	DisplayName             string   `json:"displayName" binding:"required"`
	Description             string   `json:"description"`
	PricePerMonthCents      int64    `json:"pricePerMonthCents" binding:"required"`
	MonthlyConversionsLimit int      `json:"monthlyConversionsLimit" binding:"required"`
	Features                []string `json:"features"`
	IsActive                bool     `json:"isActive"`
}

// UpdatePlanRequest represents the request to update a plan
type UpdatePlanRequest struct {
	DisplayName             *string  `json:"displayName,omitempty"`
	Description             *string  `json:"description,omitempty"`
	PricePerMonthCents      *int64   `json:"pricePerMonthCents,omitempty"`
	MonthlyConversionsLimit *int     `json:"monthlyConversionsLimit,omitempty"`
	Features                []string `json:"features,omitempty"`
	IsActive                *bool    `json:"isActive,omitempty"`
}

// RevokeQuotaRequest represents the request to revoke quota
type RevokeQuotaRequest struct {
	UserID    string `json:"userId" binding:"required"`
	QuotaType string `json:"quotaType" binding:"required,oneof=free paid"`
	Amount    int    `json:"amount" binding:"required,min=1"`
	Reason    string `json:"reason" binding:"required"`
}

// RevokePlanRequest represents the request to revoke a plan
type RevokePlanRequest struct {
	UserID string `json:"userId" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

// Constants
const (
	// Actor types
	ActorTypeSystem = "system"
	ActorTypeUser   = "user"
	ActorTypeAdmin  = "admin"

	// Actions
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionRevoke   = "revoke"
	ActionSuspend  = "suspend"
	ActionActivate = "activate"
	ActionVerify   = "verify"

	// Resources
	ResourceUser       = "user"
	ResourceVendor     = "vendor"
	ResourcePlan       = "plan"
	ResourcePayment    = "payment"
	ResourceQuota      = "quota"
	ResourceImage      = "image"
	ResourceConversion = "conversion"
)

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function for creating bool pointers
func boolPtr(b bool) *bool {
	return &b
}
