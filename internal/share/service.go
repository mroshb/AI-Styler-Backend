package share

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// Service provides share management functionality
type Service struct {
	store             Store
	conversionService ConversionService
	imageService      ImageService
	notifier          NotificationService
	auditLogger       AuditLogger
	metrics           MetricsCollector
}

// NewService creates a new share service
func NewService(
	store Store,
	conversionService ConversionService,
	imageService ImageService,
	notifier NotificationService,
	auditLogger AuditLogger,
	metrics MetricsCollector,
) *Service {
	return &Service{
		store:             store,
		conversionService: conversionService,
		imageService:      imageService,
		notifier:          notifier,
		auditLogger:       auditLogger,
		metrics:           metrics,
	}
}

// CreateSharedLink creates a new shared link for a conversion result
func (s *Service) CreateSharedLink(ctx context.Context, userID string, req CreateShareRequest) (CreateShareResponse, error) {
	// Validate conversion exists and is owned by user
	conversion, err := s.conversionService.GetConversion(ctx, req.ConversionID, userID)
	if err != nil {
		return CreateShareResponse{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	// Validate conversion is completed
	if conversion.Status != "completed" {
		return CreateShareResponse{}, fmt.Errorf("conversion must be completed to share")
	}

	if conversion.ResultImageID == nil {
		return CreateShareResponse{}, fmt.Errorf("conversion has no result image")
	}

	// Validate expiry time
	if req.ExpiryMinutes < MinExpiryMinutes || req.ExpiryMinutes > MaxExpiryMinutes {
		return CreateShareResponse{}, fmt.Errorf("expiry time must be between %d and %d minutes", MinExpiryMinutes, MaxExpiryMinutes)
	}

	// Generate unique share token
	shareToken, err := s.generateShareToken()
	if err != nil {
		return CreateShareResponse{}, fmt.Errorf("failed to generate share token: %w", err)
	}

	// Calculate expiry time
	expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)

	// Generate signed URL for the result image
	signedURL, err := s.imageService.GenerateSignedURL(ctx, *conversion.ResultImageID, AccessTypeView, int64(req.ExpiryMinutes*60))
	if err != nil {
		return CreateShareResponse{}, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	// Create shared link in database
	shareID, err := s.store.CreateSharedLink(ctx, req.ConversionID, userID, shareToken, signedURL, expiresAt, req.MaxAccessCount)
	if err != nil {
		return CreateShareResponse{}, fmt.Errorf("failed to create shared link: %w", err)
	}

	// Log audit
	if err := s.auditLogger.LogShareCreated(ctx, userID, req.ConversionID, shareID); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to log share creation audit: %v\n", err)
	}

	// Record metrics
	if err := s.metrics.RecordShareCreated(ctx, userID, req.ConversionID); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to record share creation metrics: %v\n", err)
	}

	// Send notification
	if err := s.notifier.SendShareCreated(ctx, userID, shareID, shareToken); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to send share creation notification: %v\n", err)
	}

	// Generate public URL
	publicURL := fmt.Sprintf("/api/share/%s", shareToken)

	return CreateShareResponse{
		ShareID:    shareID,
		ShareToken: shareToken,
		SignedURL:  signedURL,
		ExpiresAt:  expiresAt,
		PublicURL:  publicURL,
	}, nil
}

// AccessSharedLink validates and provides access to a shared link
func (s *Service) AccessSharedLink(ctx context.Context, req AccessShareRequest) (AccessShareResponse, error) {
	// Get shared link by token
	sharedLink, err := s.store.GetSharedLinkByToken(ctx, req.ShareToken)
	if err != nil {
		// Log failed access attempt
		s.store.LogSharedLinkAccess(ctx, "", req, false, "Share token not found")
		return AccessShareResponse{
			Success:      false,
			ErrorMessage: "Share token not found",
		}, nil
	}

	// Check if link is active
	if !sharedLink.IsActive {
		s.store.LogSharedLinkAccess(ctx, sharedLink.ID, req, false, "Share link is inactive")
		return AccessShareResponse{
			Success:      false,
			ErrorMessage: "Share link is inactive",
		}, nil
	}

	// Check if link has expired
	if time.Now().After(sharedLink.ExpiresAt) {
		s.store.LogSharedLinkAccess(ctx, sharedLink.ID, req, false, "Share link has expired")
		return AccessShareResponse{
			Success:      false,
			ErrorMessage: "Share link has expired",
		}, nil
	}

	// Check access count limit
	if sharedLink.MaxAccessCount != nil && sharedLink.AccessCount >= *sharedLink.MaxAccessCount {
		s.store.LogSharedLinkAccess(ctx, sharedLink.ID, req, false, "Share link access limit exceeded")
		return AccessShareResponse{
			Success:      false,
			ErrorMessage: "Share link access limit exceeded",
		}, nil
	}

	// Get conversion details
	conversion, err := s.conversionService.GetConversion(ctx, sharedLink.ConversionID, sharedLink.UserID)
	if err != nil {
		s.store.LogSharedLinkAccess(ctx, sharedLink.ID, req, false, "Failed to get conversion details")
		return AccessShareResponse{
			Success:      false,
			ErrorMessage: "Failed to get conversion details",
		}, nil
	}

	// Get result image details
	var resultImageURL string
	if conversion.ResultImageID != nil {
		image, err := s.imageService.GetImage(ctx, *conversion.ResultImageID)
		if err == nil {
			resultImageURL = image.OriginalURL
		}
	}

	// Update access count
	newAccessCount := sharedLink.AccessCount + 1
	updates := map[string]interface{}{
		"access_count": newAccessCount,
		"updated_at":   time.Now(),
	}
	if err := s.store.UpdateSharedLink(ctx, sharedLink.ID, updates); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to update access count: %v\n", err)
	}

	// Log successful access
	s.store.LogSharedLinkAccess(ctx, sharedLink.ID, req, true, "")

	// Log audit
	if err := s.auditLogger.LogShareAccessed(ctx, sharedLink.ID, req.IPAddress, req.UserAgent); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to log share access audit: %v\n", err)
	}

	// Record metrics
	if err := s.metrics.RecordShareAccessed(ctx, sharedLink.ID, true); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to record share access metrics: %v\n", err)
	}

	// Send notification to owner
	if err := s.notifier.SendShareAccessed(ctx, sharedLink.UserID, sharedLink.ID, newAccessCount); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to send share access notification: %v\n", err)
	}

	// Calculate seconds until expiry
	secondsUntilExpiry := int(time.Until(sharedLink.ExpiresAt).Seconds())

	return AccessShareResponse{
		Success:            true,
		ConversionID:       sharedLink.ConversionID,
		ResultImageURL:     resultImageURL,
		AccessCount:        newAccessCount,
		SecondsUntilExpiry: secondsUntilExpiry,
	}, nil
}

// DeactivateSharedLink deactivates a shared link
func (s *Service) DeactivateSharedLink(ctx context.Context, shareID, userID string) error {
	// Deactivate the shared link
	if err := s.store.DeactivateSharedLink(ctx, shareID, userID); err != nil {
		return fmt.Errorf("failed to deactivate shared link: %w", err)
	}

	// Log audit
	if err := s.auditLogger.LogShareDeactivated(ctx, userID, shareID); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to log share deactivation audit: %v\n", err)
	}

	return nil
}

// ListUserSharedLinks lists user's shared links
func (s *Service) ListUserSharedLinks(ctx context.Context, userID string, limit, offset int) ([]ActiveSharedLink, error) {
	links, err := s.store.ListUserSharedLinks(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list shared links: %w", err)
	}

	return links, nil
}

// GetSharedLinkStats gets statistics for shared links
func (s *Service) GetSharedLinkStats(ctx context.Context, userID, conversionID string) (SharedLinkStats, error) {
	stats, err := s.store.GetSharedLinkStats(ctx, userID, conversionID)
	if err != nil {
		return SharedLinkStats{}, fmt.Errorf("failed to get shared link stats: %w", err)
	}

	return stats, nil
}

// CleanupExpiredLinks removes expired shared links
func (s *Service) CleanupExpiredLinks(ctx context.Context) (int, error) {
	count, err := s.store.CleanupExpiredLinks(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired links: %w", err)
	}

	return count, nil
}

// generateShareToken generates a cryptographically secure random token
func (s *Service) generateShareToken() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, ShareTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode as base64url (URL-safe base64)
	token := base64.URLEncoding.EncodeToString(bytes)
	return token, nil
}
