package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DBJobQueue implements JobQueue interface using database
type DBJobQueue struct {
	db *sql.DB
}

// NewDBJobQueue creates a new database job queue
func NewDBJobQueue(db *sql.DB) JobQueue {
	return &DBJobQueue{db: db}
}

// EnqueueJob adds a job to the queue
func (q *DBJobQueue) EnqueueJob(ctx context.Context, job *WorkerJob) error {
	query := `
		INSERT INTO worker_jobs (
			id, type, conversion_id, user_id, priority, status, retry_count, 
			max_retries, payload, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	payloadJSON := fmt.Sprintf(`{"userImageId":"%s","clothImageId":"%s"}`,
		job.Payload.UserImageID, job.Payload.ClothImageID)

	_, err := q.db.ExecContext(ctx, query,
		job.ID,
		job.Type,
		job.ConversionID,
		job.UserID,
		int(job.Priority),
		string(job.Status),
		job.RetryCount,
		job.MaxRetries,
		payloadJSON,
		job.CreatedAt,
		job.UpdatedAt,
	)

	return err
}

// DequeueJob removes and returns a job from the queue
// Uses FOR UPDATE SKIP LOCKED to prevent race conditions when multiple workers try to get the same job
func (q *DBJobQueue) DequeueJob(ctx context.Context, workerID string) (*WorkerJob, error) {
	query := `
		UPDATE worker_jobs 
		SET status = 'processing', worker_id = $1, started_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM worker_jobs 
			WHERE status = 'pending' 
			ORDER BY priority DESC, created_at ASC 
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, type, conversion_id, user_id, priority, status, worker_id, 
		          retry_count, max_retries, payload, created_at, updated_at, started_at`

	var job WorkerJob
	var priority int
	var status string
	var payloadJSON string
	var startedAt sql.NullTime

	err := q.db.QueryRowContext(ctx, query, workerID).Scan(
		&job.ID,
		&job.Type,
		&job.ConversionID,
		&job.UserID,
		&priority,
		&status,
		&job.WorkerID,
		&job.RetryCount,
		&job.MaxRetries,
		&payloadJSON,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No jobs available
		}
		// Check if error is "relation does not exist" - table hasn't been created yet
		errStr := err.Error()
		hasWorkerJobs := strings.Contains(strings.ToLower(errStr), "worker_jobs")
		hasDoesNotExist := strings.Contains(strings.ToLower(errStr), "does not exist")
		if errStr != "" && (errStr == `pq: relation "worker_jobs" does not exist` || 
			errStr == `relation "worker_jobs" does not exist` ||
			(hasWorkerJobs && hasDoesNotExist)) {
			return nil, fmt.Errorf("worker_jobs table does not exist - please run migrations: %w", err)
		}
		return nil, err
	}

	job.Priority = JobPriority(priority)
	job.Status = JobStatus(status)
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}

	// Parse payload JSON
	if err := parsePayloadJSON(payloadJSON, &job.Payload); err != nil {
		// Fallback to placeholder if parsing fails
	job.Payload = JobPayload{
		UserImageID:  "placeholder",
		ClothImageID: "placeholder",
		}
	}

	return &job, nil
}

// parsePayloadJSON parses the payload JSON string into JobPayload
func parsePayloadJSON(payloadJSON string, payload *JobPayload) error {
	if payloadJSON == "" {
		return fmt.Errorf("empty payload")
	}

	var payloadData map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payloadData); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	payload.UserImageID = getStringFromMap(payloadData, "userImageId")
	payload.ClothImageID = getStringFromMap(payloadData, "clothImageId")
	
	// Initialize Options map if it doesn't exist
	if payload.Options == nil {
		payload.Options = make(map[string]interface{})
	}
	
	if options, ok := payloadData["options"].(map[string]interface{}); ok {
		payload.Options = options
	}

	return nil
}

// getStringFromMap extracts a string value from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// UpdateJobStatus updates the status of a job
func (q *DBJobQueue) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, workerID string) error {
	query := `
		UPDATE worker_jobs 
		SET status = $1, worker_id = $2, updated_at = NOW()
		WHERE id = $3`

	_, err := q.db.ExecContext(ctx, query, string(status), workerID, jobID)
	return err
}

// CompleteJob marks a job as completed
func (q *DBJobQueue) CompleteJob(ctx context.Context, jobID string, result interface{}) error {
	query := `
		UPDATE worker_jobs 
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1`

	_, err := q.db.ExecContext(ctx, query, jobID)
	return err
}

// FailJob marks a job as failed
func (q *DBJobQueue) FailJob(ctx context.Context, jobID string, errorMessage string) error {
	query := `
		UPDATE worker_jobs 
		SET status = 'failed', error_message = $1, updated_at = NOW()
		WHERE id = $2`

	_, err := q.db.ExecContext(ctx, query, errorMessage, jobID)
	return err
}

// UpdateJobRetryCount updates the retry count and error message for a job
func (q *DBJobQueue) UpdateJobRetryCount(ctx context.Context, jobID string, retryCount int, errorMessage string) error {
	query := `
		UPDATE worker_jobs 
		SET retry_count = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3`

	_, err := q.db.ExecContext(ctx, query, retryCount, errorMessage, jobID)
	return err
}

// GetJob retrieves a job by ID
func (q *DBJobQueue) GetJob(ctx context.Context, jobID string) (*WorkerJob, error) {
	query := `
		SELECT id, type, conversion_id, user_id, priority, status, worker_id, 
		       retry_count, max_retries, payload, created_at, updated_at, started_at, completed_at
		FROM worker_jobs 
		WHERE id = $1`

	var job WorkerJob
	var priority int
	var status string
	var payloadJSON string
	var startedAt, completedAt sql.NullTime

	err := q.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.Type,
		&job.ConversionID,
		&job.UserID,
		&priority,
		&status,
		&job.WorkerID,
		&job.RetryCount,
		&job.MaxRetries,
		&payloadJSON,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
	)

	if err != nil {
		return nil, err
	}

	job.Priority = JobPriority(priority)
	job.Status = JobStatus(status)
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	// Parse payload JSON
	if err := parsePayloadJSON(payloadJSON, &job.Payload); err != nil {
		// Fallback to placeholder if parsing fails
	job.Payload = JobPayload{
		UserImageID:  "placeholder",
		ClothImageID: "placeholder",
		}
	}

	return &job, nil
}

// GetQueueStats returns queue statistics
func (q *DBJobQueue) GetQueueStats(ctx context.Context) (*WorkerStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_jobs,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_jobs,
			COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_jobs,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_jobs,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_jobs
		FROM worker_jobs`

	var stats WorkerStats
	err := q.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalJobs,
		&stats.PendingJobs,
		&stats.ProcessingJobs,
		&stats.CompletedJobs,
		&stats.FailedJobs,
	)

	if err != nil {
		return nil, err
	}

	// Calculate success rate
	if stats.TotalJobs > 0 {
		stats.SuccessRate = float64(stats.CompletedJobs) / float64(stats.TotalJobs) * 100
	}

	return &stats, nil
}

// CleanupOldJobs removes old completed/failed jobs
func (q *DBJobQueue) CleanupOldJobs(ctx context.Context, olderThan time.Time) error {
	query := `
		DELETE FROM worker_jobs 
		WHERE (status = 'completed' OR status = 'failed') 
		AND updated_at < $1`

	_, err := q.db.ExecContext(ctx, query, olderThan)
	return err
}

// GetPendingJobs returns pending jobs
func (q *DBJobQueue) GetPendingJobs(ctx context.Context, limit int) ([]*WorkerJob, error) {
	query := `
		SELECT id, type, conversion_id, user_id, priority, status, worker_id, 
		       retry_count, max_retries, payload, created_at, updated_at
		FROM worker_jobs 
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC 
		LIMIT $1`

	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*WorkerJob
	for rows.Next() {
		var job WorkerJob
		var priority int
		var status string
		var payloadJSON string

		err := rows.Scan(
			&job.ID,
			&job.Type,
			&job.ConversionID,
			&job.UserID,
			&priority,
			&status,
			&job.WorkerID,
			&job.RetryCount,
			&job.MaxRetries,
			&payloadJSON,
			&job.CreatedAt,
			&job.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		job.Priority = JobPriority(priority)
		job.Status = JobStatus(status)

		// Parse payload JSON
		if err := parsePayloadJSON(payloadJSON, &job.Payload); err != nil {
			// Fallback to placeholder if parsing fails
		job.Payload = JobPayload{
			UserImageID:  "placeholder",
			ClothImageID: "placeholder",
			}
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}
