package user

import (
	"context"
	"time"
)

// Store defines the interface for user data operations
type Store interface {
	// Profile operations
	GetProfile(ctx context.Context, userID string) (UserProfile, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error)

	// Conversion operations
	CreateConversion(ctx context.Context, userID string, req CreateConversionRequest) (UserConversion, error)
	GetConversion(ctx context.Context, conversionID string) (UserConversion, error)
	UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) (UserConversion, error)
	GetConversionHistory(ctx context.Context, userID string, req ConversionHistoryRequest) (ConversionHistoryResponse, error)

	// Plan operations
	GetUserPlan(ctx context.Context, userID string) (UserPlan, error)
	CreateUserPlan(ctx context.Context, userID string, planName string) (UserPlan, error)
	UpdateUserPlan(ctx context.Context, planID string, status string) (UserPlan, error)

	// Quota operations
	GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error)
	CanUserConvert(ctx context.Context, userID string, conversionType string) (bool, error)
	RecordConversion(ctx context.Context, userID string, conversionType string, inputFileURL string, styleName string) (string, error)

	// Utility operations
	GetUserByID(ctx context.Context, userID string) (UserProfile, error)
}

// ConversionProcessor defines the interface for processing conversions
type ConversionProcessor interface {
	ProcessConversion(ctx context.Context, conversionID string, inputFileURL string, styleName string) error
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendConversionComplete(ctx context.Context, userID string, conversionID string, outputFileURL string) error
	SendConversionFailed(ctx context.Context, userID string, conversionID string, errorMessage string) error
	SendQuotaWarning(ctx context.Context, userID string, remainingConversions int) error
}

// FileStorage defines the interface for file operations
type FileStorage interface {
	UploadFile(ctx context.Context, fileData []byte, fileName string) (string, error)
	GetFileURL(ctx context.Context, filePath string) (string, error)
	DeleteFile(ctx context.Context, filePath string) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) bool
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogUserAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error
}
