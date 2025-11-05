package image

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DBStore implements the Store interface using PostgreSQL
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new database store
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// CreateImage creates a new image record
func (s *DBStore) CreateImage(ctx context.Context, req CreateImageRequest) (Image, error) {
	query := `
		INSERT INTO images (
			id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
			file_size, mime_type, width, height, is_public, tags, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, created_at, updated_at`

	var image Image
	imageID := uuid.New().String()

	// Handle nil/empty tags - use pq.StringArray
	var tagsArg interface{}
	if req.Tags != nil && len(req.Tags) > 0 {
		tagsArg = pq.StringArray(req.Tags)
	} else {
		tagsArg = pq.StringArray{}
	}

	// Handle nil/empty metadata - convert to JSONB
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

	err := s.db.QueryRowContext(ctx, query,
		imageID,
		req.UserID,
		req.VendorID,
		req.Type,
		req.FileName,
		req.OriginalURL,
		req.ThumbnailURL,
		req.FileSize,
		req.MimeType,
		req.Width,
		req.Height,
		req.IsPublic,
		tagsArg,
		metadataJSONStr,
	).Scan(&image.ID, &image.CreatedAt, &image.UpdatedAt)

	if err != nil {
		return Image{}, fmt.Errorf("failed to create image: %w", err)
	}

	// Fill in the rest of the image data
	image.UserID = req.UserID
	image.VendorID = req.VendorID
	image.Type = req.Type
	image.FileName = req.FileName
	image.OriginalURL = req.OriginalURL
	image.ThumbnailURL = req.ThumbnailURL
	image.FileSize = req.FileSize
	image.MimeType = req.MimeType
	image.Width = req.Width
	image.Height = req.Height
	image.IsPublic = req.IsPublic
	image.Tags = req.Tags
	image.Metadata = req.Metadata

	return image, nil
}

// GetImage retrieves an image by ID
func (s *DBStore) GetImage(ctx context.Context, imageID string) (Image, error) {
	query := `
		SELECT id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
			   file_size, mime_type, width, height, is_public, tags, metadata,
			   created_at, updated_at
		FROM images
		WHERE id = $1`

	var image Image
	var metadataJSON string
	err := s.db.QueryRowContext(ctx, query, imageID).Scan(
		&image.ID,
		&image.UserID,
		&image.VendorID,
		&image.Type,
		&image.FileName,
		&image.OriginalURL,
		&image.ThumbnailURL,
		&image.FileSize,
		&image.MimeType,
		&image.Width,
		&image.Height,
		&image.IsPublic,
		pq.Array(&image.Tags),
		&metadataJSON,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
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
func (s *DBStore) UpdateImage(ctx context.Context, imageID string, req UpdateImageRequest) (Image, error) {
	// Build dynamic query based on provided fields
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
		args = append(args, pq.StringArray(req.Tags))
		argIndex++
	}

	if req.Metadata != nil {
		setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
		args = append(args, req.Metadata)
		argIndex++
	}

	if len(setParts) == 0 {
		return Image{}, fmt.Errorf("no fields to update")
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE clause
	args = append(args, imageID)

	query := fmt.Sprintf(`
		UPDATE images 
		SET %s
		WHERE id = $%d
		RETURNING id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
				  file_size, mime_type, width, height, is_public, tags, metadata,
				  created_at, updated_at`,
		strings.Join(setParts, ", "),
		argIndex,
	)

	var image Image
	var metadataJSON string
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&image.ID,
		&image.UserID,
		&image.VendorID,
		&image.Type,
		&image.FileName,
		&image.OriginalURL,
		&image.ThumbnailURL,
		&image.FileSize,
		&image.MimeType,
		&image.Width,
		&image.Height,
		&image.IsPublic,
		pq.Array(&image.Tags),
		&metadataJSON,
		&image.CreatedAt,
		&image.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Image{}, fmt.Errorf("image not found")
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
func (s *DBStore) DeleteImage(ctx context.Context, imageID string) error {
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
		return fmt.Errorf("image not found")
	}

	return nil
}

// ListImages retrieves images with filtering and pagination
func (s *DBStore) ListImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	// Build WHERE clause
	whereParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Type != nil {
		whereParts = append(whereParts, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *req.Type)
		argIndex++
	}

	if req.IsPublic != nil {
		whereParts = append(whereParts, fmt.Sprintf("is_public = $%d", argIndex))
		args = append(args, *req.IsPublic)
		argIndex++
	}

	if req.UserID != nil {
		whereParts = append(whereParts, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *req.UserID)
		argIndex++
	}

	if req.VendorID != nil {
		whereParts = append(whereParts, fmt.Sprintf("vendor_id = $%d", argIndex))
		args = append(args, *req.VendorID)
		argIndex++
	}

	if len(req.Tags) > 0 {
		whereParts = append(whereParts, fmt.Sprintf("tags && $%d", argIndex))
		args = append(args, pq.StringArray(req.Tags))
		argIndex++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM images %s", whereClause)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to count images: %w", err)
	}

	// Calculate pagination
	offset := (req.Page - 1) * req.PageSize
	totalPages := (total + req.PageSize - 1) / req.PageSize

	// Add pagination to args
	args = append(args, req.PageSize, offset)

	// Get images
	query := fmt.Sprintf(`
		SELECT id, user_id, vendor_id, type, file_name, original_url, thumbnail_url,
			   file_size, mime_type, width, height, is_public, tags, metadata,
			   created_at, updated_at
		FROM images
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause,
		argIndex,
		argIndex+1,
	)

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
			&image.ID,
			&image.UserID,
			&image.VendorID,
			&image.Type,
			&image.FileName,
			&image.OriginalURL,
			&image.ThumbnailURL,
			&image.FileSize,
			&image.MimeType,
			&image.Width,
			&image.Height,
			&image.IsPublic,
			pq.Array(&image.Tags),
			&metadataJSON,
			&image.CreatedAt,
			&image.UpdatedAt,
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

	if err = rows.Err(); err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to iterate images: %w", err)
	}

	return ImageListResponse{
		Images:     images,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CanUploadImage checks if user/vendor can upload an image
func (s *DBStore) CanUploadImage(ctx context.Context, userID *string, vendorID *string, imageType ImageType, fileSize int64) (bool, error) {
	if userID != nil {
		// Check user quota using database function
		query := `SELECT can_user_convert($1, 'free')`
		var canUpload bool
		err := s.db.QueryRowContext(ctx, query, *userID).Scan(&canUpload)
		if err != nil {
			return false, fmt.Errorf("failed to check user quota: %w", err)
		}
		return canUpload, nil
	}

	if vendorID != nil {
		// Check vendor quota using database function
		query := `SELECT can_vendor_upload_image($1, true)`
		var canUpload bool
		err := s.db.QueryRowContext(ctx, query, *vendorID).Scan(&canUpload)
		if err != nil {
			return false, fmt.Errorf("failed to check vendor quota: %w", err)
		}
		return canUpload, nil
	}

	return false, fmt.Errorf("either userID or vendorID must be provided")
}

// GetQuotaStatus retrieves current quota status
func (s *DBStore) GetQuotaStatus(ctx context.Context, userID *string, vendorID *string) (QuotaStatus, error) {
	if userID != nil {
		query := `SELECT * FROM get_user_quota_status($1)`
		var status QuotaStatus
		err := s.db.QueryRowContext(ctx, query, *userID).Scan(
			&status.UserImagesRemaining,
			&status.PaidImagesRemaining,
			&status.TotalImagesRemaining,
			&status.PlanName,
			&status.MonthlyLimit,
		)
		if err != nil {
			return QuotaStatus{}, fmt.Errorf("failed to get user quota status: %w", err)
		}
		return status, nil
	}

	if vendorID != nil {
		query := `SELECT * FROM get_vendor_quota_status($1)`
		var status QuotaStatus
		err := s.db.QueryRowContext(ctx, query, *vendorID).Scan(
			&status.VendorImagesRemaining,
			&status.PaidImagesRemaining,
			&status.TotalImagesRemaining,
			&status.PlanName,
			&status.MonthlyLimit,
		)
		if err != nil {
			return QuotaStatus{}, fmt.Errorf("failed to get vendor quota status: %w", err)
		}
		return status, nil
	}

	return QuotaStatus{}, fmt.Errorf("either userID or vendorID must be provided")
}

// GetImageStats retrieves image statistics
func (s *DBStore) GetImageStats(ctx context.Context, userID *string, vendorID *string) (ImageStats, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `
			SELECT 
				COUNT(*) as total_images,
				COUNT(*) FILTER (WHERE type = 'user') as user_images,
				COUNT(*) FILTER (WHERE type = 'vendor') as vendor_images,
				COUNT(*) FILTER (WHERE type = 'result') as result_images,
				COUNT(*) FILTER (WHERE is_public = true) as public_images,
				COUNT(*) FILTER (WHERE is_public = false) as private_images,
				SUM(file_size) as total_file_size,
				AVG(file_size) as average_file_size,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days') as images_last_30_days
			FROM images
			WHERE user_id = $1`
		args = []interface{}{*userID}
	} else if vendorID != nil {
		query = `
			SELECT 
				COUNT(*) as total_images,
				COUNT(*) FILTER (WHERE type = 'user') as user_images,
				COUNT(*) FILTER (WHERE type = 'vendor') as vendor_images,
				COUNT(*) FILTER (WHERE type = 'result') as result_images,
				COUNT(*) FILTER (WHERE is_public = true) as public_images,
				COUNT(*) FILTER (WHERE is_public = false) as private_images,
				SUM(file_size) as total_file_size,
				AVG(file_size) as average_file_size,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days') as images_last_30_days
			FROM images
			WHERE vendor_id = $1`
		args = []interface{}{*vendorID}
	} else {
		query = `
			SELECT 
				COUNT(*) as total_images,
				COUNT(*) FILTER (WHERE type = 'user') as user_images,
				COUNT(*) FILTER (WHERE type = 'vendor') as vendor_images,
				COUNT(*) FILTER (WHERE type = 'result') as result_images,
				COUNT(*) FILTER (WHERE is_public = true) as public_images,
				COUNT(*) FILTER (WHERE is_public = false) as private_images,
				SUM(file_size) as total_file_size,
				AVG(file_size) as average_file_size,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days') as images_last_30_days
			FROM images`
		args = []interface{}{}
	}

	var stats ImageStats
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalImages,
		&stats.UserImages,
		&stats.VendorImages,
		&stats.ResultImages,
		&stats.PublicImages,
		&stats.PrivateImages,
		&stats.TotalFileSize,
		&stats.AverageFileSize,
		&stats.ImagesLast30Days,
	)

	if err != nil {
		return ImageStats{}, fmt.Errorf("failed to get image stats: %w", err)
	}

	stats.TotalSizeBytes = stats.TotalFileSize
	return stats, nil
}
