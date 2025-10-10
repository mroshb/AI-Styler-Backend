package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

// CreateSharedLink creates a new shared link
func (s *postgresStore) CreateSharedLink(ctx context.Context, conversionID, userID, shareToken, signedURL string, expiresAt time.Time, maxAccessCount *int) (string, error) {
	query := `
		INSERT INTO shared_links (conversion_id, user_id, share_token, signed_url, expires_at, max_access_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var shareID string
	err := s.db.QueryRowContext(ctx, query, conversionID, userID, shareToken, signedURL, expiresAt, maxAccessCount).Scan(&shareID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", errors.New("share token already exists")
		}
		return "", fmt.Errorf("failed to create shared link: %w", err)
	}

	return shareID, nil
}

// GetSharedLink retrieves a shared link by ID
func (s *postgresStore) GetSharedLink(ctx context.Context, shareID string) (SharedLink, error) {
	query := `
		SELECT id, conversion_id, user_id, share_token, signed_url, expires_at,
		       max_access_count, access_count, is_active, created_at, updated_at
		FROM shared_links 
		WHERE id = $1`

	var link SharedLink
	err := s.db.QueryRowContext(ctx, query, shareID).Scan(
		&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken, &link.SignedURL,
		&link.ExpiresAt, &link.MaxAccessCount, &link.AccessCount, &link.IsActive,
		&link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SharedLink{}, errors.New("shared link not found")
		}
		return SharedLink{}, fmt.Errorf("failed to get shared link: %w", err)
	}

	return link, nil
}

// GetSharedLinkByToken retrieves a shared link by token
func (s *postgresStore) GetSharedLinkByToken(ctx context.Context, shareToken string) (ActiveSharedLink, error) {
	query := `
		SELECT id, conversion_id, user_id, share_token, signed_url, expires_at,
		       max_access_count, access_count, is_active, created_at, updated_at
		FROM shared_links 
		WHERE share_token = $1`

	var link ActiveSharedLink
	err := s.db.QueryRowContext(ctx, query, shareToken).Scan(
		&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken, &link.SignedURL,
		&link.ExpiresAt, &link.MaxAccessCount, &link.AccessCount, &link.IsActive,
		&link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ActiveSharedLink{}, errors.New("shared link not found")
		}
		return ActiveSharedLink{}, fmt.Errorf("failed to get shared link: %w", err)
	}

	return link, nil
}

// UpdateSharedLink updates a shared link
func (s *postgresStore) UpdateSharedLink(ctx context.Context, shareID string, updates map[string]interface{}) error {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(setParts) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE shared_links 
		SET %s, updated_at = NOW()
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	args = append(args, shareID)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update shared link: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shared link not found")
	}

	return nil
}

// DeactivateSharedLink deactivates a shared link
func (s *postgresStore) DeactivateSharedLink(ctx context.Context, shareID, userID string) error {
	query := `
		UPDATE shared_links 
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND user_id = $2`

	result, err := s.db.ExecContext(ctx, query, shareID, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate shared link: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shared link not found or access denied")
	}

	return nil
}

// ListUserSharedLinks lists user's shared links
func (s *postgresStore) ListUserSharedLinks(ctx context.Context, userID string, limit, offset int) ([]ActiveSharedLink, error) {
	query := `
		SELECT id, conversion_id, user_id, share_token, signed_url, expires_at,
		       max_access_count, access_count, is_active, created_at, updated_at
		FROM shared_links 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list shared links: %w", err)
	}
	defer rows.Close()

	var links []ActiveSharedLink
	for rows.Next() {
		var link ActiveSharedLink
		err := rows.Scan(
			&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken, &link.SignedURL,
			&link.ExpiresAt, &link.MaxAccessCount, &link.AccessCount, &link.IsActive,
			&link.CreatedAt, &link.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shared link: %w", err)
		}
		links = append(links, link)
	}

	return links, nil
}

// GetSharedLinkStats gets statistics for shared links
func (s *postgresStore) GetSharedLinkStats(ctx context.Context, userID, conversionID string) (SharedLinkStats, error) {
	query := `SELECT * FROM get_shared_link_stats($1, $2)`

	var stats SharedLinkStats
	err := s.db.QueryRowContext(ctx, query, userID, conversionID).Scan(
		&stats.TotalSharedLinks, &stats.ActiveSharedLinks,
		&stats.TotalAccessCount, &stats.UniqueAccessCount,
	)
	if err != nil {
		return SharedLinkStats{}, fmt.Errorf("failed to get shared link stats: %w", err)
	}

	return stats, nil
}

// LogSharedLinkAccess logs access to a shared link
func (s *postgresStore) LogSharedLinkAccess(ctx context.Context, shareID string, req AccessShareRequest, success bool, errorMessage string) error {
	query := `
		INSERT INTO shared_link_access_logs (shared_link_id, ip_address, user_agent, success, error_message)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.db.ExecContext(ctx, query, shareID, req.IPAddress, req.UserAgent, success, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to log shared link access: %w", err)
	}

	// Update access count if successful
	if success {
		updateQuery := `
			UPDATE shared_links 
			SET access_count = access_count + 1, updated_at = NOW()
			WHERE id = $1`
		_, err = s.db.ExecContext(ctx, updateQuery, shareID)
		if err != nil {
			return fmt.Errorf("failed to update access count: %w", err)
		}
	}

	return nil
}

// CleanupExpiredLinks removes expired shared links
func (s *postgresStore) CleanupExpiredLinks(ctx context.Context) (int, error) {
	query := `SELECT cleanup_expired_shared_links()`

	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired links: %w", err)
	}

	return count, nil
}

// ValidateSharedLinkAccess validates access to a shared link
func (s *postgresStore) ValidateSharedLinkAccess(ctx context.Context, shareToken string) (bool, error) {
	query := `SELECT is_valid FROM validate_shared_link_access($1)`

	var isValid bool
	err := s.db.QueryRowContext(ctx, query, shareToken).Scan(&isValid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("shared link not found")
		}
		return false, fmt.Errorf("failed to validate shared link access: %w", err)
	}

	return isValid, nil
}

// GetSharedLinkDetails gets detailed information about a shared link
func (s *postgresStore) GetSharedLinkDetails(ctx context.Context, shareToken string) (SharedLinkDetails, error) {
	query := `
		SELECT sl.id, sl.conversion_id, sl.user_id, sl.share_token, sl.signed_url,
		       sl.expires_at, sl.max_access_count, sl.access_count, sl.is_active,
		       sl.created_at, sl.updated_at,
		       c.status as conversion_status,
		       u.name as user_name
		FROM shared_links sl
		JOIN conversions c ON sl.conversion_id = c.id
		JOIN users u ON sl.user_id = u.id
		WHERE sl.share_token = $1`

	var details SharedLinkDetails
	err := s.db.QueryRowContext(ctx, query, shareToken).Scan(
		&details.ID, &details.ConversionID, &details.UserID, &details.ShareToken,
		&details.SignedURL, &details.ExpiresAt, &details.MaxAccessCount,
		&details.AccessCount, &details.IsActive, &details.CreatedAt, &details.UpdatedAt,
		&details.ConversionStatus, &details.UserName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SharedLinkDetails{}, errors.New("shared link not found")
		}
		return SharedLinkDetails{}, fmt.Errorf("failed to get shared link details: %w", err)
	}

	return details, nil
}

// GetSharedLinkAccessLogs gets access logs for a shared link
func (s *postgresStore) GetSharedLinkAccessLogs(ctx context.Context, shareID string, limit, offset int) ([]AccessLog, error) {
	query := `
		SELECT id, shared_link_id, ip_address, user_agent, accessed_at, success, error_message
		FROM shared_link_access_logs 
		WHERE shared_link_id = $1
		ORDER BY accessed_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, shareID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get access logs: %w", err)
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var log AccessLog
		err := rows.Scan(
			&log.ID, &log.SharedLinkID, &log.IPAddress, &log.UserAgent,
			&log.AccessedAt, &log.Success, &log.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan access log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetPopularSharedLinks gets most accessed shared links
func (s *postgresStore) GetPopularSharedLinks(ctx context.Context, limit int) ([]PopularSharedLink, error) {
	query := `
		SELECT sl.id, sl.conversion_id, sl.user_id, sl.share_token, sl.access_count,
		       sl.created_at, u.name as user_name, c.status as conversion_status
		FROM shared_links sl
		JOIN users u ON sl.user_id = u.id
		JOIN conversions c ON sl.conversion_id = c.id
		WHERE sl.is_active = true AND sl.access_count > 0
		ORDER BY sl.access_count DESC, sl.created_at DESC
		LIMIT $1`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular shared links: %w", err)
	}
	defer rows.Close()

	var links []PopularSharedLink
	for rows.Next() {
		var link PopularSharedLink
		err := rows.Scan(
			&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken,
			&link.AccessCount, &link.CreatedAt, &link.UserName, &link.ConversionStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan popular shared link: %w", err)
		}
		links = append(links, link)
	}

	return links, nil
}
