package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// Service provides user management functionality
type Service struct {
	store       Store
	auditLogger AuditLogger
}

// NewService creates a new user service
func NewService(
	store Store,
	auditLogger AuditLogger,
) *Service {
	return &Service{
		store:       store,
		auditLogger: auditLogger,
	}
}

// GetProfile retrieves a user's profile
func (s *Service) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	profile, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to get profile: %w", err)
	}
	return profile, nil
}

// UpdateProfile updates a user's profile
func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error) {
	// Validate input
	if req.Name != nil && len(*req.Name) > 100 {
		return UserProfile{}, errors.New("name too long")
	}
	if req.Bio != nil && len(*req.Bio) > 500 {
		return UserProfile{}, errors.New("bio too long")
	}
	if req.AvatarURL != nil && len(*req.AvatarURL) > 500 {
		return UserProfile{}, errors.New("avatar URL too long")
	}

	profile, err := s.store.UpdateProfile(ctx, userID, req)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to update profile: %w", err)
	}

	// Log the action
	metadata := map[string]interface{}{
		"updated_fields": getUpdatedFields(req),
	}
	_ = s.auditLogger.LogUserAction(ctx, userID, "profile_updated", metadata)

	return profile, nil
}

// Helper functions

func getUpdatedFields(req UpdateProfileRequest) []string {
	var fields []string
	if req.Name != nil {
		fields = append(fields, "name")
	}
	if req.AvatarURL != nil {
		fields = append(fields, "avatar_url")
	}
	if req.Bio != nil {
		fields = append(fields, "bio")
	}
	return fields
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Error handling helpers

func isUniqueConstraintError(err error, constraint string) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505" && pqErr.Constraint == constraint
	}
	return false
}

func isForeignKeyError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23503"
	}
	return false
}

func isNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
