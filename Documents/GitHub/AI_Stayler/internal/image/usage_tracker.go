package image

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DBUsageTracker implements the UsageTracker interface using PostgreSQL
type DBUsageTracker struct {
	db *sql.DB
}

// NewDBUsageTracker creates a new database usage tracker
func NewDBUsageTracker(db *sql.DB) *DBUsageTracker {
	return &DBUsageTracker{db: db}
}

// RecordUsage records image usage
func (t *DBUsageTracker) RecordUsage(ctx context.Context, imageID string, userID *string, action string, metadata map[string]interface{}) error {
	query := `
		INSERT INTO image_usage_history (
			id, image_id, user_id, action, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)`

	usageID := uuid.New().String()
	_, err := t.db.ExecContext(ctx, query,
		usageID,
		imageID,
		userID,
		action,
		metadata,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	return nil
}

// GetUsageHistory retrieves usage history for an image
func (t *DBUsageTracker) GetUsageHistory(ctx context.Context, imageID string, req ImageUsageHistoryRequest) (ImageUsageHistoryResponse, error) {
	// Build WHERE clause
	whereParts := []string{"image_id = $1"}
	args := []interface{}{imageID}
	argIndex := 2

	if req.Action != "" {
		whereParts = append(whereParts, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, req.Action)
		argIndex++
	}

	whereClause := "WHERE " + strings.Join(whereParts, " AND ")

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM image_usage_history %s", whereClause)
	var total int
	err := t.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return ImageUsageHistoryResponse{}, fmt.Errorf("failed to count usage history: %w", err)
	}

	// Calculate pagination
	offset := (req.Page - 1) * req.PageSize
	totalPages := (total + req.PageSize - 1) / req.PageSize

	// Add pagination to args
	args = append(args, req.PageSize, offset)

	// Get usage history
	query := fmt.Sprintf(`
		SELECT id, image_id, user_id, action, ip_address, user_agent, metadata, created_at
		FROM image_usage_history
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause,
		argIndex,
		argIndex+1,
	)

	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ImageUsageHistoryResponse{}, fmt.Errorf("failed to get usage history: %w", err)
	}
	defer rows.Close()

	var history []ImageUsageHistory
	for rows.Next() {
		var usage ImageUsageHistory
		err := rows.Scan(
			&usage.ID,
			&usage.ImageID,
			&usage.UserID,
			&usage.Action,
			&usage.IPAddress,
			&usage.UserAgent,
			&usage.Metadata,
			&usage.CreatedAt,
		)
		if err != nil {
			return ImageUsageHistoryResponse{}, fmt.Errorf("failed to scan usage history: %w", err)
		}
		history = append(history, usage)
	}

	if err = rows.Err(); err != nil {
		return ImageUsageHistoryResponse{}, fmt.Errorf("failed to iterate usage history: %w", err)
	}

	return ImageUsageHistoryResponse{
		History:    history,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUsageStats retrieves usage statistics for an image
func (t *DBUsageTracker) GetUsageStats(ctx context.Context, imageID string) (UsageStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_views,
			COUNT(*) FILTER (WHERE action = 'download') as total_downloads,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') as recent_views,
			COUNT(*) FILTER (WHERE action = 'download' AND created_at >= NOW() - INTERVAL '24 hours') as recent_downloads
		FROM image_usage_history
		WHERE image_id = $1`

	var stats UsageStats
	err := t.db.QueryRowContext(ctx, query, imageID).Scan(
		&stats.TotalViews,
		&stats.TotalDownloads,
		&stats.UniqueUsers,
		&stats.RecentViews,
		&stats.RecentDownloads,
	)

	if err != nil {
		return UsageStats{}, fmt.Errorf("failed to get usage stats: %w", err)
	}

	return stats, nil
}
