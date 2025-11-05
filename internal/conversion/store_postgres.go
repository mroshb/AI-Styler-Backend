package conversion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// postgresStore implements the Store interface using PostgreSQL
type postgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *sql.DB) Store {
	return &postgresStore{db: db}
}

// CheckUserQuota checks if user can perform conversion
func (s *postgresStore) CheckUserQuota(ctx context.Context, userID string) (QuotaCheck, error) {
	query := `SELECT * FROM get_user_quota_status($1)`

	var quota QuotaCheck
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&quota.RemainingFree, &quota.RemainingPaid, &quota.TotalRemaining,
		&quota.PlanName, &quota.MonthlyLimit,
	)
	if err != nil {
		return QuotaCheck{}, fmt.Errorf("failed to check quota: %w", err)
	}

	quota.CanConvert = quota.TotalRemaining > 0
	return quota, nil
}

// CreateConversion creates a new conversion
func (s *postgresStore) CreateConversion(ctx context.Context, userID, userImageID, clothImageID, styleName string) (string, error) {
	query := `
		INSERT INTO conversions (user_id, user_image_id, cloth_image_id, status, style_name)
		VALUES ($1, $2, $3, 'pending', $4)
		RETURNING id`

	var conversionID string
	err := s.db.QueryRowContext(ctx, query, userID, userImageID, clothImageID, styleName).Scan(&conversionID)
	if err != nil {
		return "", fmt.Errorf("failed to create conversion: %w", err)
	}

	return conversionID, nil
}

// GetConversion retrieves a conversion by ID
func (s *postgresStore) GetConversion(ctx context.Context, conversionID string) (Conversion, error) {
	query := `
		SELECT id, user_id, user_image_id, cloth_image_id, result_image_id, status,
		       error_message, processing_time_ms, created_at, updated_at
		FROM conversions 
		WHERE id = $1`

	var conv Conversion
	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conv.ID, &conv.UserID, &conv.UserImageID, &conv.ClothImageID, &conv.ResultImageID,
		&conv.Status, &conv.ErrorMessage, &conv.ProcessingTimeMs, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Conversion{}, errors.New("conversion not found")
		}
		return Conversion{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	return conv, nil
}

// GetConversionWithDetails retrieves a conversion with detailed information
func (s *postgresStore) GetConversionWithDetails(ctx context.Context, conversionID string) (ConversionResponse, error) {
	query := `
		SELECT c.id, c.user_id, c.user_image_id, c.cloth_image_id, c.result_image_id,
		       c.status, c.error_message, c.processing_time_ms, c.created_at, c.updated_at,
		       ui.original_url as user_image_url, ci.original_url as cloth_image_url,
		       ri.original_url as result_image_url
		FROM conversions c
		LEFT JOIN images ui ON c.user_image_id = ui.id
		LEFT JOIN images ci ON c.cloth_image_id = ci.id
		LEFT JOIN images ri ON c.result_image_id = ri.id
		WHERE c.id = $1`

	var resp ConversionResponse
	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&resp.ID, &resp.UserID, &resp.UserImageID, &resp.ClothImageID, &resp.ResultImageID,
		&resp.Status, &resp.ErrorMessage, &resp.ProcessingTimeMs, &resp.CreatedAt, &resp.UpdatedAt,
		&resp.UserImageURL, &resp.ClothImageURL, &resp.ResultImageURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversionResponse{}, errors.New("conversion not found")
		}
		return ConversionResponse{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	return resp, nil
}

// UpdateConversion updates a conversion
func (s *postgresStore) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) error {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}
	if req.ResultImageID != nil {
		setParts = append(setParts, fmt.Sprintf("result_image_id = $%d", argIndex))
		args = append(args, *req.ResultImageID)
		argIndex++
	}
	if req.ErrorMessage != nil {
		setParts = append(setParts, fmt.Sprintf("error_message = $%d", argIndex))
		args = append(args, *req.ErrorMessage)
		argIndex++
	}
	if req.ProcessingTimeMs != nil {
		setParts = append(setParts, fmt.Sprintf("processing_time_ms = $%d", argIndex))
		args = append(args, *req.ProcessingTimeMs)
		argIndex++
	}

	if len(setParts) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE conversions 
		SET %s, updated_at = NOW()
		WHERE id = $%d`,
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf(`
			UPDATE conversions 
			SET %s, updated_at = NOW()
			WHERE id = $%d`,
			fmt.Sprintf("%s", setParts[i]), argIndex)
	}

	args = append(args, conversionID)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("conversion not found")
	}

	return nil
}

// DeleteConversion deletes a conversion
func (s *postgresStore) DeleteConversion(ctx context.Context, conversionID string) error {
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
		return errors.New("conversion not found")
	}

	return nil
}

// ListConversions lists user's conversions
func (s *postgresStore) ListConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error) {
	query := `
		SELECT c.id, c.user_id, c.user_image_id, c.cloth_image_id, c.result_image_id,
		       c.status, c.error_message, c.processing_time_ms, c.created_at, c.updated_at,
		       ui.original_url as user_image_url, ci.original_url as cloth_image_url,
		       ri.original_url as result_image_url
		FROM conversions c
		LEFT JOIN images ui ON c.user_image_id = ui.id
		LEFT JOIN images ci ON c.cloth_image_id = ci.id
		LEFT JOIN images ri ON c.result_image_id = ri.id
		WHERE c.user_id = $1`

	args := []interface{}{req.UserID}
	argIndex := 2

	// Add status filter if provided
	if req.Status != "" {
		query += fmt.Sprintf(" AND c.status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	// Add date range filter if provided
	if !req.StartDate.IsZero() {
		query += fmt.Sprintf(" AND c.created_at >= $%d", argIndex)
		args = append(args, req.StartDate)
		argIndex++
	}
	if !req.EndDate.IsZero() {
		query += fmt.Sprintf(" AND c.created_at <= $%d", argIndex)
		args = append(args, req.EndDate)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	offset := (req.Page - 1) * req.PageSize
	args = append(args, req.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to list conversions: %w", err)
	}
	defer rows.Close()

	var conversions []ConversionResponse
	for rows.Next() {
		var conv ConversionResponse
		err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.UserImageID, &conv.ClothImageID, &conv.ResultImageID,
			&conv.Status, &conv.ErrorMessage, &conv.ProcessingTimeMs, &conv.CreatedAt, &conv.UpdatedAt,
			&conv.UserImageURL, &conv.ClothImageURL, &conv.ResultImageURL,
		)
		if err != nil {
			return ConversionListResponse{}, fmt.Errorf("failed to scan conversion: %w", err)
		}
		conversions = append(conversions, conv)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM conversions WHERE user_id = $1`
	countArgs := []interface{}{req.UserID}
	countArgIndex := 2

	if req.Status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", countArgIndex)
		countArgs = append(countArgs, req.Status)
		countArgIndex++
	}
	if !req.StartDate.IsZero() {
		countQuery += fmt.Sprintf(" AND created_at >= $%d", countArgIndex)
		countArgs = append(countArgs, req.StartDate)
		countArgIndex++
	}
	if !req.EndDate.IsZero() {
		countQuery += fmt.Sprintf(" AND created_at <= $%d", countArgIndex)
		countArgs = append(countArgs, req.EndDate)
		countArgIndex++
	}

	var total int
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to get total count: %w", err)
	}

	return ConversionListResponse{
		Conversions: conversions,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}, nil
}

// GetConversionStats gets conversion statistics
func (s *postgresStore) GetConversionStats(ctx context.Context, userID string, timeRange string) (map[string]interface{}, error) {
	var timeFilter string
	switch timeRange {
	case "today":
		timeFilter = "created_at >= CURRENT_DATE"
	case "week":
		timeFilter = "created_at >= CURRENT_DATE - INTERVAL '7 days'"
	case "month":
		timeFilter = "created_at >= CURRENT_DATE - INTERVAL '30 days'"
	case "year":
		timeFilter = "created_at >= CURRENT_DATE - INTERVAL '365 days'"
	default:
		timeFilter = "1=1"
	}

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_conversions,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_conversions,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_conversions,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_conversions,
			COUNT(*) FILTER (WHERE status = 'processing') as processing_conversions,
			AVG(processing_time_ms) FILTER (WHERE status = 'completed') as avg_processing_time_ms
		FROM conversions 
		WHERE user_id = $1 AND %s`, timeFilter)

	var stats map[string]interface{}
	var total, completed, failed, pending, processing int
	var avgProcessingTime sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&total, &completed, &failed, &pending, &processing, &avgProcessingTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion stats: %w", err)
	}

	stats = map[string]interface{}{
		"total_conversions":      total,
		"completed_conversions":  completed,
		"failed_conversions":     failed,
		"pending_conversions":    pending,
		"processing_conversions": processing,
		"success_rate":           float64(completed) / float64(total) * 100,
	}

	if avgProcessingTime.Valid {
		stats["avg_processing_time_ms"] = avgProcessingTime.Float64
	} else {
		stats["avg_processing_time_ms"] = 0
	}

	return stats, nil
}

// CompleteJob marks a job as completed
func (s *postgresStore) CompleteJob(ctx context.Context, jobID, resultImageID string, processingTimeMs int) error {
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

// CreateConversionJob creates a background job for conversion
func (s *postgresStore) CreateConversionJob(ctx context.Context, conversionID string) error {
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
func (s *postgresStore) GetNextJob(ctx context.Context) (*ConversionJob, error) {
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
		if errors.Is(err, sql.ErrNoRows) {
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
func (s *postgresStore) UpdateJobStatus(ctx context.Context, jobID, status, workerID string) error {
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

// FailJob marks a job as failed
func (s *postgresStore) FailJob(ctx context.Context, jobID, errorMessage string) error {
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

// ReserveQuota reserves quota for a conversion
func (s *postgresStore) ReserveQuota(ctx context.Context, userID string) error {
	// This is handled by the create_conversion function
	return nil
}

// ReleaseQuota releases reserved quota
func (s *postgresStore) ReleaseQuota(ctx context.Context, userID string) error {
	// This would be implemented if we need to release quota on failure
	return nil
}
