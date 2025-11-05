package user

import (
	"context"
)

// Store defines the interface for user data operations
type Store interface {
	// Profile operations
	GetProfile(ctx context.Context, userID string) (UserProfile, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error)

	// Utility operations
	GetUserByID(ctx context.Context, userID string) (UserProfile, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogUserAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error
}
