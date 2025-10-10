package conversion

import (
	"context"
	"database/sql"
	"fmt"
)

// store implements the Store interface
type store struct {
	db *sql.DB
}

// NewStore creates a new conversion store
func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

// CreateConversion creates a new conversion request
func (s *store) CreateConversion(ctx context.Context, userID, userImageID, clothImageID string) (string, error) {
	query := `
		SELECT create_conversion($1, $2, $3)
	`

	var conversionID string
	err := s.db.QueryRowContext(ctx, query, userID, userImageID, clothImageID).Scan(&conversionID)
	if err != nil {
		return "", fmt.Errorf("failed to create conversion: %w", err)
	}

	return conversionID, nil
}

// GetConversion retrieves a conversion by ID
func (s *store) GetConversion(ctx context.Context, conversionID string) (Conversion, error) {
	query := `
		SELECT id, user_id, user_image_id, cloth_image_id, status, result_image_id, 
		       error_message, processing_time_ms, created_at, updated_at, completed_at
		FROM conversions 
		WHERE id = $1
	`

	var conv Conversion
	var resultImageID sql.NullString
	var errorMessage sql.NullString
	var processingTimeMs sql.NullInt32
	var completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conv.ID, &conv.UserID, &conv.UserImageID, &conv.ClothImageID, &conv.Status,
		&resultImageID, &errorMessage, &processingTimeMs, &conv.CreatedAt, &conv.UpdatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Conversion{}, fmt.Errorf("conversion not found")
		}
		return Conversion{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	if resultImageID.Valid {
		conv.ResultImageID = &resultImageID.String
	}
	if errorMessage.Valid {
		conv.ErrorMessage = &errorMessage.String
	}
	if processingTimeMs.Valid {
		timeMs := int(processingTimeMs.Int32)
		conv.ProcessingTimeMs = &timeMs
	}
	if completedAt.Valid {
		conv.CompletedAt = &completedAt.Time
	}

	return conv, nil
}

// GetConversionWithDetails retrieves a conversion with image details
func (s *store) GetConversionWithDetails(ctx context.Context, conversionID string) (ConversionResponse, error) {
	query := `
		SELECT * FROM get_conversion_with_details($1)
	`

	var conv ConversionResponse
	var resultImageID sql.NullString
	var errorMessage sql.NullString
	var processingTimeMs sql.NullInt32
	var completedAt sql.NullTime
	var userImageURL sql.NullString
	var clothImageURL sql.NullString
	var resultImageURL sql.NullString

	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conv.ID, &conv.UserID, &conv.UserImageID, &conv.ClothImageID, &conv.Status,
		&resultImageID, &errorMessage, &processingTimeMs, &conv.CreatedAt, &conv.UpdatedAt, &completedAt,
		&userImageURL, &clothImageURL, &resultImageURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ConversionResponse{}, fmt.Errorf("conversion not found")
		}
		return ConversionResponse{}, fmt.Errorf("failed to get conversion details: %w", err)
	}

	if resultImageID.Valid {
		conv.ResultImageID = &resultImageID.String
	}
	if errorMessage.Valid {
		conv.ErrorMessage = &errorMessage.String
	}
	if processingTimeMs.Valid {
		timeMs := int(processingTimeMs.Int32)
		conv.ProcessingTimeMs = &timeMs
	}
	if completedAt.Valid {
		conv.CompletedAt = &completedAt.Time
	}

	return conv, nil
}

// UpdateConversion updates a conversion
func (s *store) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) error {
	query := `
		SELECT update_conversion_status($1, $2, $3, $4, $5)
	`

	var status string
	if req.Status != nil {
		status = *req.Status
	} else {
		status = "" // Don't update status
	}

	var resultImageID sql.NullString
	if req.ResultImageID != nil {
		resultImageID = sql.NullString{String: *req.ResultImageID, Valid: true}
	}

	var errorMessage sql.NullString
	if req.ErrorMessage != nil {
		errorMessage = sql.NullString{String: *req.ErrorMessage, Valid: true}
	}

	var processingTimeMs sql.NullInt32
	if req.ProcessingTimeMs != nil {
		processingTimeMs = sql.NullInt32{Int32: int32(*req.ProcessingTimeMs), Valid: true}
	}

	var success bool
	err := s.db.QueryRowContext(ctx, query, conversionID, status, resultImageID, errorMessage, processingTimeMs).Scan(&success)
	if err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}

	if !success {
		return fmt.Errorf("conversion not found")
	}

	return nil
}

// ListConversions lists conversions with pagination
func (s *store) ListConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error) {
	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = DefaultPageSize
	}
	if req.PageSize > MaxPageSize {
		req.PageSize = MaxPageSize
	}

	offset := (req.Page - 1) * req.PageSize

	// Build query
	whereClause := "WHERE user_id = $1"
	args := []interface{}{req.UserID}
	argIndex := 2

	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM conversions %s", whereClause)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to count conversions: %w", err)
	}

	// Get conversions
	query := fmt.Sprintf(`
		SELECT id, user_id, user_image_id, cloth_image_id, status, result_image_id, 
		       error_message, processing_time_ms, created_at, updated_at, completed_at
		FROM conversions 
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, req.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to list conversions: %w", err)
	}
	defer rows.Close()

	var conversions []ConversionResponse
	for rows.Next() {
		var conv ConversionResponse
		var resultImageID sql.NullString
		var errorMessage sql.NullString
		var processingTimeMs sql.NullInt32
		var completedAt sql.NullTime

		err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.UserImageID, &conv.ClothImageID, &conv.Status,
			&resultImageID, &errorMessage, &processingTimeMs, &conv.CreatedAt, &conv.UpdatedAt, &completedAt,
		)
		if err != nil {
			return ConversionListResponse{}, fmt.Errorf("failed to scan conversion: %w", err)
		}

		if resultImageID.Valid {
			conv.ResultImageID = &resultImageID.String
		}
		if errorMessage.Valid {
			conv.ErrorMessage = &errorMessage.String
		}
		if processingTimeMs.Valid {
			timeMs := int(processingTimeMs.Int32)
			conv.ProcessingTimeMs = &timeMs
		}
		if completedAt.Valid {
			conv.CompletedAt = &completedAt.Time
		}

		conversions = append(conversions, conv)
	}

	if err = rows.Err(); err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to iterate conversions: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return ConversionListResponse{
		Conversions: conversions,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
	}, nil
}

// DeleteConversion deletes a conversion
func (s *store) DeleteConversion(ctx context.Context, conversionID string) error {
	query := `DELETE FROM conversions WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, conversionID)
	if err != nil {
		return fmt.Errorf("failed to delete conversion: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversion not found")
	}

	return nil
}

// CheckUserQuota checks user's conversion quota
func (s *store) CheckUserQuota(ctx context.Context, userID string) (QuotaCheck, error) {
	query := `SELECT * FROM get_user_quota_status($1)`

	var quota QuotaCheck
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&quota.RemainingFree,
		&quota.RemainingPaid,
		&quota.TotalRemaining,
		&quota.PlanName,
		&quota.MonthlyLimit,
	)
	if err != nil {
		return QuotaCheck{}, fmt.Errorf("failed to check user quota: %w", err)
	}

	quota.CanConvert = quota.TotalRemaining > 0

	return quota, nil
}

// ReserveQuota reserves quota for a conversion
func (s *store) ReserveQuota(ctx context.Context, userID string) error {
	// This is handled by the create_conversion function
	return nil
}

// ReleaseQuota releases reserved quota
func (s *store) ReleaseQuota(ctx context.Context, userID string) error {
	// This would be implemented if we need to release quota on failure
	return nil
}

// CreateConversionJob creates a background job for conversion
func (s *store) CreateConversionJob(ctx context.Context, conversionID string) error {
	query := `
		INSERT INTO conversion_jobs (conversion_id, priority)
		VALUES ($1, 0)
	`

	_, err := s.db.ExecContext(ctx, query, conversionID)
	if err != nil {
		return fmt.Errorf("failed to create conversion job: %w", err)
	}

	return nil
}

// GetNextJob gets the next job to process
func (s *store) GetNextJob(ctx context.Context) (*ConversionJob, error) {
	query := `
		SELECT id, conversion_id, status, worker_id, priority, retry_count, max_retries, 
		       error_message, created_at, updated_at
		FROM conversion_jobs 
		WHERE status = 'queued'
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job ConversionJob
	var workerID sql.NullString
	var errorMessage sql.NullString

	err := s.db.QueryRowContext(ctx, query).Scan(
		&job.ID, &job.ConversionID, &job.Status, &workerID, &job.Priority,
		&job.RetryCount, &job.MaxRetries, &errorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to get next job: %w", err)
	}

	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}

	return &job, nil
}

// UpdateJobStatus updates job status
func (s *store) UpdateJobStatus(ctx context.Context, jobID, status, workerID string) error {
	query := `
		UPDATE conversion_jobs 
		SET status = $1, worker_id = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err := s.db.ExecContext(ctx, query, status, workerID, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// CompleteJob marks a job as completed
func (s *store) CompleteJob(ctx context.Context, jobID, resultImageID string, processingTimeMs int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update job status
	_, err = tx.ExecContext(ctx, `
		UPDATE conversion_jobs 
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update conversion status
	_, err = tx.ExecContext(ctx, `
		UPDATE conversions 
		SET status = 'completed', result_image_id = $1, processing_time_ms = $2, 
		    completed_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT conversion_id FROM conversion_jobs WHERE id = $3)
	`, resultImageID, processingTimeMs, jobID)
	if err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}

	return tx.Commit()
}

// FailJob marks a job as failed
func (s *store) FailJob(ctx context.Context, jobID, errorMessage string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update job status
	_, err = tx.ExecContext(ctx, `
		UPDATE conversion_jobs 
		SET status = 'failed', error_message = $1, completed_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, errorMessage, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update conversion status
	_, err = tx.ExecContext(ctx, `
		UPDATE conversions 
		SET status = 'failed', error_message = $1, completed_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT conversion_id FROM conversion_jobs WHERE id = $2)
	`, errorMessage, jobID)
	if err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}

	return tx.Commit()
}
