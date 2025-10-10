package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// NewImageStorageService creates a new image storage service
func NewImageStorageService(storage StorageServiceInterface, config ImageStorageConfig) *ImageStorageService {
	return &ImageStorageService{
		storage: storage,
		config:  config,
	}
}

// UploadImage uploads an image with specialized handling
func (s *ImageStorageService) UploadImage(ctx context.Context, req ImageUploadRequest) (*ImageUploadResponse, error) {
	// Validate request
	if err := s.validateUploadRequest(req); err != nil {
		return nil, err
	}

	// Read file data
	data, err := io.ReadAll(req.File)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Validate file size
	if int64(len(data)) > s.config.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size")
	}

	// Validate MIME type
	if !s.isValidMimeType(req.ContentType) {
		return nil, fmt.Errorf("unsupported file type: %s", req.ContentType)
	}

	// Generate unique filename
	ext := filepath.Ext(req.FileName)
	baseName := strings.TrimSuffix(req.FileName, ext)
	uniqueFileName := fmt.Sprintf("%s_%s%s", baseName, generateUniqueID()[:8], ext)

	// Determine storage path based on image type
	storagePath := s.getStoragePath(req.ImageType, req.OwnerID)

	// Upload original image
	filePath, err := s.storage.UploadFile(ctx, data, uniqueFileName, storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Generate thumbnail
	thumbnailPath := ""
	if len(s.config.ThumbnailSizes) > 0 {
		thumbnailData, err := s.generateThumbnail(data, req.ContentType, s.config.ThumbnailSizes[0])
		if err == nil {
			thumbFileName := fmt.Sprintf("thumb_%s", uniqueFileName)
			thumbPath, err := s.storage.UploadFile(ctx, thumbnailData, thumbFileName, storagePath+"/thumbnails")
			if err == nil {
				thumbnailPath = thumbPath
			}
		}
	}

	// Get file info
	fileInfo, err := s.storage.GetFileInfo(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Generate signed URL for immediate access
	signedURL, err := s.storage.GenerateSignedURL(ctx, filePath, AccessTypeView, DefaultSignedURLTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return &ImageUploadResponse{
		ImageID:      generateUniqueID(),
		FilePath:     filePath,
		ThumbnailURL: thumbnailPath,
		OriginalURL:  signedURL,
		FileSize:     fileInfo.Size,
		Checksum:     fileInfo.Checksum,
		UploadedAt:   time.Now(),
	}, nil
}

// GetImageAccess generates signed URL for image access
func (s *ImageStorageService) GetImageAccess(ctx context.Context, req ImageAccessRequest) (*ImageAccessResponse, error) {
	// Get image metadata (this would typically come from database)
	imageInfo, err := s.getImageMetadata(ctx, req.ImageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image metadata: %w", err)
	}

	// Check access permissions
	if err := s.checkAccessPermissions(ctx, imageInfo, req.RequesterID); err != nil {
		return nil, fmt.Errorf("access denied: %w", err)
	}

	// Generate signed URL
	signedURL, err := s.storage.GenerateSignedURL(ctx, imageInfo.FilePath, req.AccessType, req.TTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	// Get file info
	fileInfo, err := s.storage.GetFileInfo(ctx, imageInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &ImageAccessResponse{
		SignedURL:  signedURL,
		ExpiresAt:  time.Now().Add(time.Duration(req.TTL) * time.Second),
		AccessType: req.AccessType,
		ImageInfo:  fileInfo,
	}, nil
}

// DeleteImage deletes an image and its backups
func (s *ImageStorageService) DeleteImage(ctx context.Context, imageID string) error {
	// Get image metadata
	imageInfo, err := s.getImageMetadata(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image metadata: %w", err)
	}

	// Delete original file
	if err := s.storage.DeleteFile(ctx, imageInfo.FilePath); err != nil {
		return fmt.Errorf("failed to delete original file: %w", err)
	}

	// Delete thumbnail if exists
	if imageInfo.ThumbnailPath != "" {
		_ = s.storage.DeleteFile(ctx, imageInfo.ThumbnailPath)
	}

	// Delete backups (this would be implemented based on backup strategy)
	_ = s.deleteImageBackups(ctx, imageID)

	return nil
}

// SearchImages searches for images based on criteria
func (s *ImageStorageService) SearchImages(ctx context.Context, req ImageSearchRequest) (*ImageSearchResponse, error) {
	// This would typically query the database for image metadata
	// For now, we'll implement a basic file system search

	var images []ImageMetadata

	// Determine search directory
	searchDir := s.getSearchDirectory(req.ImageType, req.OwnerID)

	// List files in directory
	fileInfos, err := s.storage.ListFiles(ctx, searchDir, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Convert file info to image metadata
	for _, fileInfo := range fileInfos {
		imageMeta := s.fileInfoToImageMetadata(fileInfo, req.ImageType, req.OwnerID)
		if s.matchesSearchCriteria(imageMeta, req) {
			images = append(images, imageMeta)
		}
	}

	return &ImageSearchResponse{
		Images:     images,
		Total:      int64(len(images)),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: (len(images) + req.PageSize - 1) / req.PageSize,
		Query:      req.Query,
	}, nil
}

// GetStorageQuota returns storage quota information
func (s *ImageStorageService) GetStorageQuota(ctx context.Context, userID string) (*StorageQuota, error) {
	// Get storage stats
	stats, err := s.storage.GetStorageStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	// Calculate quota based on user
	quota := &StorageQuota{
		UserID:       userID,
		TotalUsed:    stats.UserSize + stats.ResultSize,
		TotalLimit:   s.getTotalLimit(),
		UserImages:   stats.UserFiles,
		ClothImages:  stats.ClothFiles,
		ResultImages: stats.ResultFiles,
		UserLimit:    DefaultUserLimit,
		ClothLimit:   DefaultClothLimit,
		ResultLimit:  DefaultResultLimit,
		LastUpdated:  time.Now(),
	}

	// Calculate usage percentage
	if quota.TotalLimit > 0 {
		quota.UsagePercent = float64(quota.TotalUsed) / float64(quota.TotalLimit) * 100
	}

	return quota, nil
}

// GetStorageHealth returns storage system health
func (s *ImageStorageService) GetStorageHealth(ctx context.Context) (*StorageHealth, error) {
	// Get disk usage
	diskUsage, err := s.storage.GetDiskUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	// Calculate health status
	status := HealthStatusHealthy
	if diskUsage.TotalSize > int64(float64(diskUsage.TotalSpace)*0.9) {
		status = HealthStatusCritical
	} else if diskUsage.TotalSize > int64(float64(diskUsage.TotalSpace)*0.8) {
		status = HealthStatusWarning
	}

	return &StorageHealth{
		Status:     status,
		DiskUsage:  float64(diskUsage.TotalSize) / float64(diskUsage.TotalSpace) * 100,
		FreeSpace:  diskUsage.TotalSpace - diskUsage.TotalSize,
		TotalSpace: diskUsage.TotalSpace,
		CheckedAt:  time.Now(),
	}, nil
}

// PerformBatchOperation performs batch operations on images
func (s *ImageStorageService) PerformBatchOperation(ctx context.Context, operation ImageBatchOperation) (*ImageBatchResponse, error) {
	var results []BatchResult
	successCount := 0
	failureCount := 0
	var errors []string

	for _, imageID := range operation.ImageIDs {
		result := BatchResult{ImageID: imageID}

		switch operation.Operation {
		case "delete":
			err := s.DeleteImage(ctx, imageID)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				failureCount++
				errors = append(errors, fmt.Sprintf("Failed to delete %s: %v", imageID, err))
			} else {
				result.Success = true
				result.Message = "Image deleted successfully"
				successCount++
			}
		case "backup":
			err := s.createImageBackup(ctx, imageID)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				failureCount++
				errors = append(errors, fmt.Sprintf("Failed to backup %s: %v", imageID, err))
			} else {
				result.Success = true
				result.Message = "Image backed up successfully"
				successCount++
			}
		default:
			result.Success = false
			result.Error = "Unsupported operation"
			failureCount++
			errors = append(errors, fmt.Sprintf("Unsupported operation: %s", operation.Operation))
		}

		results = append(results, result)
	}

	return &ImageBatchResponse{
		SuccessCount: successCount,
		FailureCount: failureCount,
		Errors:       errors,
		Results:      results,
	}, nil
}

// Helper methods

func (s *ImageStorageService) validateUploadRequest(req ImageUploadRequest) error {
	if req.FileName == "" {
		return fmt.Errorf("file name is required")
	}
	if req.ContentType == "" {
		return fmt.Errorf("content type is required")
	}
	if req.Size <= 0 {
		return fmt.Errorf("file size must be positive")
	}
	if req.ImageType == "" {
		return fmt.Errorf("image type is required")
	}
	if req.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}
	return nil
}

func (s *ImageStorageService) isValidMimeType(mimeType string) bool {
	for _, validType := range s.config.AllowedTypes {
		if mimeType == validType {
			return true
		}
	}
	return false
}

func (s *ImageStorageService) getStoragePath(imageType, ownerID string) string {
	switch imageType {
	case ImageTypeUser:
		return fmt.Sprintf("%s/%s", StoragePathUsers, ownerID)
	case ImageTypeCloth:
		return fmt.Sprintf("%s/%s", StoragePathCloth, ownerID)
	case ImageTypeResult:
		return fmt.Sprintf("%s/%s", StoragePathResults, ownerID)
	default:
		return fmt.Sprintf("images/unknown/%s", ownerID)
	}
}

func (s *ImageStorageService) getSearchDirectory(imageType, ownerID string) string {
	if imageType != "" && ownerID != "" {
		return s.getStoragePath(imageType, ownerID)
	}
	return filepath.Join(s.config.BasePath, "images")
}

func (s *ImageStorageService) generateThumbnail(data []byte, contentType string, size ThumbnailSize) ([]byte, error) {
	// This would typically use an image processing library like imaging or graphicsmagick
	// For now, return the original data as a placeholder
	return data, nil
}

func (s *ImageStorageService) getImageMetadata(ctx context.Context, imageID string) (*ImageMetadata, error) {
	// This would typically query the database
	// For now, return a placeholder
	return &ImageMetadata{
		ImageID:   imageID,
		FilePath:  "/placeholder/path",
		FileName:  "placeholder.jpg",
		ImageType: ImageTypeUser,
		OwnerID:   "placeholder-owner",
		IsPublic:  false,
		CreatedAt: time.Now(),
	}, nil
}

func (s *ImageStorageService) checkAccessPermissions(ctx context.Context, imageInfo *ImageMetadata, requesterID string) error {
	// Check if image is public
	if imageInfo.IsPublic {
		return nil
	}

	// Check if requester is the owner
	if imageInfo.OwnerID == requesterID {
		return nil
	}

	// Additional permission checks would go here
	return fmt.Errorf("access denied")
}

func (s *ImageStorageService) deleteImageBackups(ctx context.Context, imageID string) error {
	// Implementation would depend on backup strategy
	return nil
}

func (s *ImageStorageService) createImageBackup(ctx context.Context, imageID string) error {
	// Get image metadata
	imageInfo, err := s.getImageMetadata(ctx, imageID)
	if err != nil {
		return err
	}

	// Create backup
	return s.storage.CreateBackup(ctx, imageInfo.FilePath)
}

func (s *ImageStorageService) fileInfoToImageMetadata(fileInfo FileInfo, imageType, ownerID string) ImageMetadata {
	return ImageMetadata{
		FilePath:     fileInfo.Path,
		FileName:     filepath.Base(fileInfo.Path),
		FileSize:     fileInfo.Size,
		MimeType:     fileInfo.MimeType,
		Checksum:     fileInfo.Checksum,
		ImageType:    imageType,
		OwnerID:      ownerID,
		IsPublic:     false,
		CreatedAt:    fileInfo.CreatedAt,
		UpdatedAt:    fileInfo.ModifiedAt,
		LastAccessed: time.Now(),
		IsBackedUp:   fileInfo.IsBackedUp,
		BackupPath:   fileInfo.BackupPath,
	}
}

func (s *ImageStorageService) matchesSearchCriteria(image ImageMetadata, req ImageSearchRequest) bool {
	// Check image type
	if req.ImageType != "" && image.ImageType != req.ImageType {
		return false
	}

	// Check owner
	if req.OwnerID != "" && image.OwnerID != req.OwnerID {
		return false
	}

	// Check public status
	if req.IsPublic != nil && image.IsPublic != *req.IsPublic {
		return false
	}

	// Check MIME type
	if req.MimeType != "" && image.MimeType != req.MimeType {
		return false
	}

	// Check file size range
	if req.MinSize > 0 && image.FileSize < req.MinSize {
		return false
	}
	if req.MaxSize > 0 && image.FileSize > req.MaxSize {
		return false
	}

	// Check date range
	if !req.DateFrom.IsZero() && image.CreatedAt.Before(req.DateFrom) {
		return false
	}
	if !req.DateTo.IsZero() && image.CreatedAt.After(req.DateTo) {
		return false
	}

	// Check tags
	if len(req.Tags) > 0 {
		found := false
		for _, searchTag := range req.Tags {
			for _, imageTag := range image.Tags {
				if strings.EqualFold(imageTag, searchTag) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check query string
	if req.Query != "" {
		query := strings.ToLower(req.Query)
		fileName := strings.ToLower(image.FileName)
		if !strings.Contains(fileName, query) {
			return false
		}
	}

	return true
}

func (s *ImageStorageService) getTotalLimit() int64 {
	// This would typically be configurable per user/vendor
	return 5 * 1024 * 1024 * 1024 // 5GB default
}
