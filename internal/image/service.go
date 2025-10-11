package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Service provides image management functionality
type Service struct {
	store          Store
	fileStorage    FileStorage
	imageProcessor ImageProcessor
	usageTracker   UsageTracker
	cache          Cache
	notifier       NotificationService
	auditLogger    AuditLogger
	rateLimiter    RateLimiter
	config         StorageConfig
}

// NewService creates a new image service
func NewService(
	store Store,
	fileStorage FileStorage,
	imageProcessor ImageProcessor,
	usageTracker UsageTracker,
	cache Cache,
	notifier NotificationService,
	auditLogger AuditLogger,
	rateLimiter RateLimiter,
	config StorageConfig,
) *Service {
	return &Service{
		store:          store,
		fileStorage:    fileStorage,
		imageProcessor: imageProcessor,
		usageTracker:   usageTracker,
		cache:          cache,
		notifier:       notifier,
		auditLogger:    auditLogger,
		rateLimiter:    rateLimiter,
		config:         config,
	}
}

// UploadImage uploads a new image
func (s *Service) UploadImage(ctx context.Context, userID *string, vendorID *string, req UploadImageRequest) (Image, error) {
	// Validate input
	if err := s.validateUploadRequest(req); err != nil {
		return Image{}, err
	}

	// Determine image type and ownership
	imageType, ownerUserID, ownerVendorID, err := s.determineImageOwnership(userID, vendorID, req.Type)
	if err != nil {
		return Image{}, err
	}

	// Check rate limiting
	rateLimitKey := s.getRateLimitKey(ownerUserID, ownerVendorID, imageType)
	if !s.rateLimiter.Allow(ctx, rateLimitKey, 50, int64(time.Hour.Seconds())) {
		return Image{}, errors.New("rate limit exceeded for image upload")
	}

	// Check quota
	canUpload, err := s.store.CanUploadImage(ctx, ownerUserID, ownerVendorID, imageType, req.FileSize)
	if err != nil {
		return Image{}, fmt.Errorf("failed to check upload permission: %w", err)
	}
	if !canUpload {
		return Image{}, errors.New("image quota exceeded")
	}

	// Read file data from request
	fileData, err := io.ReadAll(req.File)
	if err != nil {
		return Image{}, fmt.Errorf("failed to read file data: %w", err)
	}

	// Validate image
	if err := s.imageProcessor.ValidateImage(ctx, fileData, req.FileName, req.MimeType); err != nil {
		return Image{}, fmt.Errorf("image validation failed: %w", err)
	}

	// Process image
	processedData, width, height, err := s.imageProcessor.ProcessImage(ctx, fileData, req.FileName)
	if err != nil {
		return Image{}, fmt.Errorf("failed to process image: %w", err)
	}

	// Generate storage path
	storagePath := s.generateStoragePath(imageType, ownerUserID, ownerVendorID)

	// Upload original image
	originalURL, err := s.fileStorage.UploadFile(ctx, processedData, req.FileName, storagePath)
	if err != nil {
		return Image{}, fmt.Errorf("failed to upload image: %w", err)
	}

	// Generate and upload thumbnail
	var thumbnailURL *string
	thumbnailData, err := s.imageProcessor.GenerateThumbnail(ctx, processedData, req.FileName, 300, 300)
	if err != nil {
		// Log error but continue without thumbnail
		_ = s.auditLogger.LogImageAction(ctx, "", ownerUserID, ownerVendorID, "thumbnail_generation_failed", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		thumbURL, err := s.fileStorage.UploadFile(ctx, thumbnailData, "thumb_"+req.FileName, storagePath+"/thumbnails")
		if err != nil {
			// Log error but continue without thumbnail
			_ = s.auditLogger.LogImageAction(ctx, "", ownerUserID, ownerVendorID, "thumbnail_upload_failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			thumbnailURL = &thumbURL
		}
	}

	// Create image record
	createReq := CreateImageRequest{
		UserID:       ownerUserID,
		VendorID:     ownerVendorID,
		Type:         imageType,
		FileName:     req.FileName,
		OriginalURL:  originalURL,
		ThumbnailURL: thumbnailURL,
		FileSize:     int64(len(processedData)),
		MimeType:     req.MimeType,
		Width:        &width,
		Height:       &height,
		IsPublic:     req.IsPublic,
		Tags:         req.Tags,
		Metadata:     req.Metadata,
	}

	image, err := s.store.CreateImage(ctx, createReq)
	if err != nil {
		// Clean up uploaded files
		_ = s.fileStorage.DeleteFile(ctx, originalURL)
		if thumbnailURL != nil {
			_ = s.fileStorage.DeleteFile(ctx, *thumbnailURL)
		}
		return Image{}, fmt.Errorf("failed to create image record: %w", err)
	}

	// Record usage
	_ = s.usageTracker.RecordUsage(ctx, image.ID, ownerUserID, ActionUpload, map[string]interface{}{
		"file_size": image.FileSize,
		"mime_type": image.MimeType,
		"is_public": image.IsPublic,
	})

	// Log the action
	_ = s.auditLogger.LogImageAction(ctx, image.ID, ownerUserID, ownerVendorID, "image_uploaded", map[string]interface{}{
		"file_name": image.FileName,
		"file_size": image.FileSize,
		"mime_type": image.MimeType,
		"is_public": image.IsPublic,
	})

	// Send notification
	_ = s.notifier.SendImageUploaded(ctx, ownerUserID, ownerVendorID, image.ID, imageType)

	// Cache the image
	_ = s.cache.CacheImage(ctx, image.ID, image)

	return image, nil
}

// GetImage retrieves a specific image
func (s *Service) GetImage(ctx context.Context, imageID string) (Image, error) {
	// Try cache first
	if cachedImage, err := s.cache.GetCachedImage(ctx, imageID); err == nil {
		return cachedImage, nil
	}

	image, err := s.store.GetImage(ctx, imageID)
	if err != nil {
		return Image{}, fmt.Errorf("failed to get image: %w", err)
	}

	// Cache the image
	_ = s.cache.CacheImage(ctx, image.ID, image)

	return image, nil
}

// UpdateImage updates an image
func (s *Service) UpdateImage(ctx context.Context, imageID string, req UpdateImageRequest) (Image, error) {
	// Validate input
	if err := s.validateUpdateRequest(req); err != nil {
		return Image{}, err
	}

	image, err := s.store.UpdateImage(ctx, imageID, req)
	if err != nil {
		return Image{}, fmt.Errorf("failed to update image: %w", err)
	}

	// Record usage
	_ = s.usageTracker.RecordUsage(ctx, imageID, nil, ActionUpdate, map[string]interface{}{
		"updated_fields": s.getUpdatedFields(req),
	})

	// Log the action
	_ = s.auditLogger.LogImageAction(ctx, imageID, image.UserID, image.VendorID, "image_updated", map[string]interface{}{
		"updated_fields": s.getUpdatedFields(req),
	})

	// Update cache
	_ = s.cache.CacheImage(ctx, image.ID, image)

	return image, nil
}

// DeleteImage deletes an image
func (s *Service) DeleteImage(ctx context.Context, imageID string) error {
	// Get image info before deletion for logging and cleanup
	image, err := s.store.GetImage(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	err = s.store.DeleteImage(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	// Clean up files
	_ = s.fileStorage.DeleteFile(ctx, image.OriginalURL)
	if image.ThumbnailURL != nil {
		_ = s.fileStorage.DeleteFile(ctx, *image.ThumbnailURL)
	}

	// Record usage
	_ = s.usageTracker.RecordUsage(ctx, imageID, image.UserID, ActionDelete, map[string]interface{}{
		"file_name": image.FileName,
		"file_size": image.FileSize,
	})

	// Log the action
	_ = s.auditLogger.LogImageAction(ctx, imageID, image.UserID, image.VendorID, "image_deleted", map[string]interface{}{
		"file_name": image.FileName,
		"file_size": image.FileSize,
	})

	// Send notification
	_ = s.notifier.SendImageDeleted(ctx, image.UserID, image.VendorID, imageID, image.Type)

	// Clear cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("image:%s", imageID))

	return nil
}

// ListImages retrieves images with filtering and pagination
func (s *Service) ListImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	response, err := s.store.ListImages(ctx, req)
	if err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to list images: %w", err)
	}

	return response, nil
}

// GenerateSignedURL generates a signed URL for image access
func (s *Service) GenerateSignedURL(ctx context.Context, imageID string, accessType string) (SignedURLResponse, error) {
	// Get image
	image, err := s.GetImage(ctx, imageID)
	if err != nil {
		return SignedURLResponse{}, fmt.Errorf("failed to get image: %w", err)
	}

	// Try cache first
	if cachedURL, err := s.cache.GetCachedSignedURL(ctx, imageID); err == nil {
		return SignedURLResponse{
			URL:        cachedURL,
			ExpiresAt:  time.Now().Add(time.Duration(s.config.SignedURLTTL) * time.Second),
			ImageID:    imageID,
			AccessType: accessType,
		}, nil
	}

	// Generate signed URL
	url, err := s.fileStorage.GenerateSignedURL(ctx, image.OriginalURL, accessType, s.config.SignedURLTTL)
	if err != nil {
		return SignedURLResponse{}, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	// Cache the signed URL
	_ = s.cache.CacheSignedURL(ctx, imageID, url, s.config.SignedURLTTL)

	// Record usage
	_ = s.usageTracker.RecordUsage(ctx, imageID, image.UserID, ActionView, map[string]interface{}{
		"access_type": accessType,
		"signed_url":  true,
	})

	return SignedURLResponse{
		URL:        url,
		ExpiresAt:  time.Now().Add(time.Duration(s.config.SignedURLTTL) * time.Second),
		ImageID:    imageID,
		AccessType: accessType,
	}, nil
}

// GetImageUsageHistory retrieves usage history for an image
func (s *Service) GetImageUsageHistory(ctx context.Context, imageID string, req ImageUsageHistoryRequest) (ImageUsageHistoryResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	response, err := s.usageTracker.GetUsageHistory(ctx, imageID, req)
	if err != nil {
		return ImageUsageHistoryResponse{}, fmt.Errorf("failed to get usage history: %w", err)
	}

	return response, nil
}

// GetQuotaStatus retrieves current quota status
func (s *Service) GetQuotaStatus(ctx context.Context, userID *string, vendorID *string) (QuotaStatus, error) {
	status, err := s.store.GetQuotaStatus(ctx, userID, vendorID)
	if err != nil {
		return QuotaStatus{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	// Send warning if quota is low
	if status.UserImagesRemaining <= 1 && status.UserImagesRemaining > 0 {
		_ = s.notifier.SendQuotaWarning(ctx, userID, vendorID, "user_images", status.UserImagesRemaining)
	}
	if status.VendorImagesRemaining <= 1 && status.VendorImagesRemaining > 0 {
		_ = s.notifier.SendQuotaWarning(ctx, userID, vendorID, "vendor_images", status.VendorImagesRemaining)
	}

	return status, nil
}

// GetImageStats retrieves image statistics
func (s *Service) GetImageStats(ctx context.Context, userID *string, vendorID *string) (ImageStats, error) {
	stats, err := s.store.GetImageStats(ctx, userID, vendorID)
	if err != nil {
		return ImageStats{}, fmt.Errorf("failed to get image stats: %w", err)
	}

	return stats, nil
}

// Validation functions

func (s *Service) validateUploadRequest(req UploadImageRequest) error {
	if strings.TrimSpace(req.FileName) == "" {
		return errors.New("file name is required")
	}
	if len(req.FileName) > 255 {
		return errors.New("file name too long")
	}
	if req.FileSize <= 0 {
		return errors.New("file size must be positive")
	}
	if req.FileSize > s.config.MaxFileSize {
		return errors.New("file size too large")
	}
	if !s.isValidMimeType(req.MimeType) {
		return errors.New("unsupported file type")
	}
	if req.Type == "" {
		return errors.New("image type is required")
	}
	if !s.isValidImageType(req.Type) {
		return errors.New("invalid image type")
	}
	if len(req.Tags) > 20 {
		return errors.New("too many tags")
	}
	for _, tag := range req.Tags {
		if len(tag) > 50 {
			return errors.New("tag too long")
		}
	}
	return nil
}

func (s *Service) validateUpdateRequest(req UpdateImageRequest) error {
	if req.Tags != nil {
		if len(req.Tags) > 20 {
			return errors.New("too many tags")
		}
		for _, tag := range req.Tags {
			if len(tag) > 50 {
				return errors.New("tag too long")
			}
		}
	}
	return nil
}

// Helper functions

func (s *Service) determineImageOwnership(userID *string, vendorID *string, imageType ImageType) (ImageType, *string, *string, error) {
	switch imageType {
	case ImageTypeUser:
		if userID == nil {
			return "", nil, nil, errors.New("user ID required for user images")
		}
		return ImageTypeUser, userID, nil, nil
	case ImageTypeVendor:
		if vendorID == nil {
			return "", nil, nil, errors.New("vendor ID required for vendor images")
		}
		return ImageTypeVendor, nil, vendorID, nil
	case ImageTypeResult:
		if userID != nil {
			return ImageTypeResult, userID, nil, nil
		} else if vendorID != nil {
			return ImageTypeResult, nil, vendorID, nil
		}
		return "", nil, nil, errors.New("user ID or vendor ID required for result images")
	default:
		return "", nil, nil, errors.New("invalid image type")
	}
}

func (s *Service) getRateLimitKey(userID *string, vendorID *string, imageType ImageType) string {
	if userID != nil {
		return fmt.Sprintf("image:user:%s:%s", *userID, imageType)
	}
	return fmt.Sprintf("image:vendor:%s:%s", *vendorID, imageType)
}

func (s *Service) generateStoragePath(imageType ImageType, userID *string, vendorID *string) string {
	switch imageType {
	case ImageTypeUser:
		return fmt.Sprintf("%s/%s", StoragePathUsers, *userID)
	case ImageTypeVendor:
		return fmt.Sprintf("%s/%s", StoragePathVendors, *vendorID)
	case ImageTypeResult:
		if userID != nil {
			return fmt.Sprintf("%s/%s", StoragePathResults, *userID)
		}
		return fmt.Sprintf("%s/vendor/%s", StoragePathResults, *vendorID)
	default:
		return "unknown"
	}
}

func (s *Service) isValidMimeType(mimeType string) bool {
	for _, validType := range s.config.AllowedTypes {
		if mimeType == validType {
			return true
		}
	}
	return false
}

func (s *Service) isValidImageType(imageType ImageType) bool {
	return imageType == ImageTypeUser || imageType == ImageTypeVendor || imageType == ImageTypeResult
}

func (s *Service) getUpdatedFields(req UpdateImageRequest) []string {
	var fields []string
	if req.IsPublic != nil {
		fields = append(fields, "is_public")
	}
	if req.Tags != nil {
		fields = append(fields, "tags")
	}
	if req.Metadata != nil {
		fields = append(fields, "metadata")
	}
	return fields
}
