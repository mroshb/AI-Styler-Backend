package image

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// postgresStore implements the Store interface using PostgreSQL
type postgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *sql.DB) Store {
	return &postgresStore{db: db}
}

// CreateImage creates a new image record
func (s *postgresStore) CreateImage(ctx context.Context, req CreateImageRequest) (Image, error) {
	query := `
		INSERT INTO images (user_id, vendor_id, type, file_name, original_url, thumbnail_url,
		                   file_size, mime_type, width, height, is_public, tags, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
		          file_size, mime_type, width, height, is_public, tags, metadata,
		          created_at, updated_at`

	var image Image
	var metadataJSON string
	
	// Convert metadata to JSONB
	var metadataJSONStr string
	if req.Metadata != nil && len(req.Metadata) > 0 {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return Image{}, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSONStr = string(metadataBytes)
	} else {
		metadataJSONStr = "{}"
	}
	
	// Handle nil/empty tags - use pq.StringArray which implements driver.Valuer
	var tagsArg interface{}
	if req.Tags != nil && len(req.Tags) > 0 {
		// Non-empty slice - convert to pq.StringArray
		tagsArg = pq.StringArray(req.Tags)
	} else {
		// Empty or nil - use empty pq.StringArray
		tagsArg = pq.StringArray{}
	}
	
	err := s.db.QueryRowContext(ctx, query,
		req.UserID, req.VendorID, req.Type, req.FileName, req.OriginalURL, req.ThumbnailURL,
		req.FileSize, req.MimeType, req.Width, req.Height, req.IsPublic,
		tagsArg, metadataJSONStr,
	).Scan(
		&image.ID, &image.UserID, &image.VendorID, &image.Type, &image.FileName,
		&image.OriginalURL, &image.ThumbnailURL, &image.FileSize, &image.MimeType,
		&image.Width, &image.Height, &image.IsPublic, pq.Array(&image.Tags),
		&metadataJSON, &image.CreatedAt, &image.UpdatedAt,
	)
	if err != nil {
		return Image{}, fmt.Errorf("failed to create image: %w", err)
	}

	// Parse metadata JSON
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &image.Metadata); err != nil {
			return Image{}, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	return image, nil
}

// GetImage retrieves a specific image
func (s *postgresStore) GetImage(ctx context.Context, imageID string) (Image, error) {
	query := `
		SELECT id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
		       file_size, mime_type, width, height, is_public, tags, metadata,
		       created_at, updated_at
		FROM images 
		WHERE id = $1`

	var image Image
	var metadataJSON string
	err := s.db.QueryRowContext(ctx, query, imageID).Scan(
		&image.ID, &image.UserID, &image.VendorID, &image.Type, &image.FileName,
		&image.OriginalURL, &image.ThumbnailURL, &image.FileSize, &image.MimeType,
		&image.Width, &image.Height, &image.IsPublic, pq.Array(&image.Tags),
		&metadataJSON, &image.CreatedAt, &image.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Image{}, errors.New("image not found")
		}
		return Image{}, fmt.Errorf("failed to get image: %w", err)
	}

	// Parse metadata JSON
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &image.Metadata); err != nil {
			return Image{}, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	return image, nil
}

// UpdateImage updates an image
func (s *postgresStore) UpdateImage(ctx context.Context, imageID string, req UpdateImageRequest) (Image, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.IsPublic != nil {
		setParts = append(setParts, fmt.Sprintf("is_public = $%d", argIndex))
		args = append(args, *req.IsPublic)
		argIndex++
	}
	if req.Tags != nil {
		setParts = append(setParts, fmt.Sprintf("tags = $%d", argIndex))
		// Use pq.StringArray for PostgreSQL array type
		args = append(args, pq.StringArray(req.Tags))
		argIndex++
	}
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return Image{}, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
		args = append(args, string(metadataBytes))
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetImage(ctx, imageID)
	}

	// Join all set parts
	setClause := strings.Join(setParts, ", ")
	
	// Add imageID to args
	args = append(args, imageID)
	imageIDArgIndex := argIndex

	query := fmt.Sprintf(`
		UPDATE images 
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
		          file_size, mime_type, width, height, is_public, tags, metadata,
		          created_at, updated_at`,
		setClause, imageIDArgIndex)

	var image Image
	var metadataJSON string
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&image.ID, &image.UserID, &image.VendorID, &image.Type, &image.FileName,
		&image.OriginalURL, &image.ThumbnailURL, &image.FileSize, &image.MimeType,
		&image.Width, &image.Height, &image.IsPublic, pq.Array(&image.Tags),
		&metadataJSON, &image.CreatedAt, &image.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Image{}, errors.New("image not found")
		}
		return Image{}, fmt.Errorf("failed to update image: %w", err)
	}

	// Parse metadata JSON
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &image.Metadata); err != nil {
			return Image{}, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	return image, nil
}

// DeleteImage deletes an image
func (s *postgresStore) DeleteImage(ctx context.Context, imageID string) error {
	query := `DELETE FROM images WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("image not found")
	}

	return nil
}

// ListImages retrieves images with filtering and pagination
func (s *postgresStore) ListImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	query := `
		SELECT id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
		       file_size, mime_type, width, height, is_public, tags, metadata,
		       created_at, updated_at
		FROM images 
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *req.UserID)
		argIndex++
	}
	if req.VendorID != nil {
		query += fmt.Sprintf(" AND vendor_id = $%d", argIndex)
		args = append(args, *req.VendorID)
		argIndex++
	}
	if req.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *req.Type)
		argIndex++
	}
	if req.IsPublic != nil {
		query += fmt.Sprintf(" AND is_public = $%d", argIndex)
		args = append(args, *req.IsPublic)
		argIndex++
	}
	if len(req.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, pq.StringArray(req.Tags))
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	offset := (req.Page - 1) * req.PageSize
	args = append(args, req.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to list images: %w", err)
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		var image Image
		var metadataJSON string
		err := rows.Scan(
			&image.ID, &image.UserID, &image.VendorID, &image.Type, &image.FileName,
			&image.OriginalURL, &image.ThumbnailURL, &image.FileSize, &image.MimeType,
			&image.Width, &image.Height, &image.IsPublic, pq.Array(&image.Tags),
			&metadataJSON, &image.CreatedAt, &image.UpdatedAt,
		)
		if err != nil {
			return ImageListResponse{}, fmt.Errorf("failed to scan image: %w", err)
		}

		// Parse metadata JSON
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &image.Metadata); err != nil {
				return ImageListResponse{}, fmt.Errorf("failed to parse metadata: %w", err)
			}
		}

		images = append(images, image)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM images WHERE 1=1`
	countArgs := []interface{}{}
	countArgIndex := 1

	if req.UserID != nil {
		countQuery += fmt.Sprintf(" AND user_id = $%d", countArgIndex)
		countArgs = append(countArgs, *req.UserID)
		countArgIndex++
	}
	if req.VendorID != nil {
		countQuery += fmt.Sprintf(" AND vendor_id = $%d", countArgIndex)
		countArgs = append(countArgs, *req.VendorID)
		countArgIndex++
	}
	if req.Type != nil {
		countQuery += fmt.Sprintf(" AND type = $%d", countArgIndex)
		countArgs = append(countArgs, *req.Type)
		countArgIndex++
	}
	if req.IsPublic != nil {
		countQuery += fmt.Sprintf(" AND is_public = $%d", countArgIndex)
		countArgs = append(countArgs, *req.IsPublic)
		countArgIndex++
	}
	if len(req.Tags) > 0 {
		countQuery += fmt.Sprintf(" AND tags && $%d", countArgIndex)
		countArgs = append(countArgs, pq.StringArray(req.Tags))
		countArgIndex++
	}

	var total int
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate total pages
	totalPages := (total + req.PageSize - 1) / req.PageSize
	if totalPages == 0 && total > 0 {
		totalPages = 1
	}

	return ImageListResponse{
		Images:     images,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CanUploadImage checks if user/vendor can upload image
func (s *postgresStore) CanUploadImage(ctx context.Context, userID *string, vendorID *string, imageType ImageType, fileSize int64) (bool, error) {
	if imageType == ImageTypeUser && userID != nil && *userID != "" {
		// For now, users can upload unlimited images
		return true, nil
	} else if imageType == ImageTypeVendor && vendorID != nil && *vendorID != "" {
		query := `SELECT can_vendor_upload_image($1, true)`
		var canUpload bool
		err := s.db.QueryRowContext(ctx, query, *vendorID).Scan(&canUpload)
		if err != nil {
			return false, fmt.Errorf("failed to check vendor upload permission: %w", err)
		}
		return canUpload, nil
	} else if imageType == ImageTypeResult {
		// Result images are created by the system, always allowed
		return true, nil
	}

	return false, errors.New("invalid image type or missing user/vendor ID")
}

// GetQuotaStatus retrieves current quota status
func (s *postgresStore) GetQuotaStatus(ctx context.Context, userID *string, vendorID *string) (QuotaStatus, error) {
	var status QuotaStatus

	if userID != nil && *userID != "" {
		query := `SELECT * FROM get_user_quota_status($1)`
		err := s.db.QueryRowContext(ctx, query, *userID).Scan(
			&status.UserImagesRemaining, &status.PaidImagesRemaining,
			&status.TotalImagesRemaining, &status.PlanName, &status.MonthlyLimit,
		)
		if err != nil {
			return QuotaStatus{}, fmt.Errorf("failed to get user quota status: %w", err)
		}
	} else if vendorID != nil && *vendorID != "" {
		query := `SELECT * FROM get_vendor_quota_status($1)`
		err := s.db.QueryRowContext(ctx, query, *vendorID).Scan(
			&status.VendorImagesRemaining, &status.PaidImagesRemaining,
			&status.TotalImagesRemaining, &status.PlanName, &status.MonthlyLimit,
		)
		if err != nil {
			return QuotaStatus{}, fmt.Errorf("failed to get vendor quota status: %w", err)
		}
	}

	return status, nil
}

// GetImageStats retrieves image statistics
func (s *postgresStore) GetImageStats(ctx context.Context, userID *string, vendorID *string) (ImageStats, error) {
	var stats ImageStats
	var query string
	var args []interface{}

	if userID != nil && *userID != "" {
		query = `
			SELECT 
				COUNT(*) as total_images,
				COUNT(*) FILTER (WHERE type = 'user') as user_images,
				0 as vendor_images,
				COUNT(*) FILTER (WHERE type = 'result') as result_images,
				COUNT(*) FILTER (WHERE is_public = true) as public_images,
				COUNT(*) FILTER (WHERE is_public = false) as private_images,
				COALESCE(SUM(file_size), 0) as total_file_size,
				COALESCE(AVG(file_size), 0) as average_file_size,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days') as images_last_30_days
			FROM images 
			WHERE user_id = $1`
		args = []interface{}{*userID}
	} else if vendorID != nil && *vendorID != "" {
		query = `
			SELECT 
				COUNT(*) as total_images,
				0 as user_images,
				COUNT(*) FILTER (WHERE type = 'vendor') as vendor_images,
				COUNT(*) FILTER (WHERE type = 'result') as result_images,
				COUNT(*) FILTER (WHERE is_public = true) as public_images,
				COUNT(*) FILTER (WHERE is_public = false) as private_images,
				COALESCE(SUM(file_size), 0) as total_file_size,
				COALESCE(AVG(file_size), 0) as average_file_size,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days') as images_last_30_days
			FROM images 
			WHERE vendor_id = $1`
		args = []interface{}{*vendorID}
	} else {
		return ImageStats{}, errors.New("user ID or vendor ID required")
	}

	var totalFileSize sql.NullInt64
	var averageFileSize sql.NullFloat64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalImages, &stats.UserImages, &stats.VendorImages, &stats.ResultImages,
		&stats.PublicImages, &stats.PrivateImages, &totalFileSize, &averageFileSize,
		&stats.ImagesLast30Days,
	)
	if err != nil {
		return ImageStats{}, fmt.Errorf("failed to get image stats: %w", err)
	}

	if totalFileSize.Valid {
		stats.TotalFileSize = totalFileSize.Int64
		stats.TotalSizeBytes = totalFileSize.Int64
	} else {
		stats.TotalFileSize = 0
		stats.TotalSizeBytes = 0
	}

	if averageFileSize.Valid {
		stats.AverageFileSize = averageFileSize.Float64
	} else {
		stats.AverageFileSize = 0.0
	}

	return stats, nil
}

// ValidateImageAccess validates if user has access to an image
func (s *postgresStore) ValidateImageAccess(ctx context.Context, imageID, userID string) error {
	query := `
		SELECT user_id, vendor_id, is_public 
		FROM images 
		WHERE id = $1`

	var imageUserID, imageVendorID sql.NullString
	var isPublic bool
	err := s.db.QueryRowContext(ctx, query, imageID).Scan(&imageUserID, &imageVendorID, &isPublic)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("image not found")
		}
		return fmt.Errorf("failed to validate image access: %w", err)
	}

	// Check if image is public
	if isPublic {
		return nil
	}

	// Check if user owns the image
	if imageUserID.Valid && imageUserID.String == userID {
		return nil
	}

	// Check if user is a vendor and owns the image
	if imageVendorID.Valid && imageVendorID.String == userID {
		return nil
	}

	return errors.New("access denied")
}
