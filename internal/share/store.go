package share

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// StoreImpl provides database operations for share service
type StoreImpl struct {
	db *sql.DB
}

// NewStore creates a new share store
func NewStore(db *sql.DB) Store {
	return &StoreImpl{db: db}
}

// CreateSharedLink creates a new shared link
func (s *StoreImpl) CreateSharedLink(ctx context.Context, conversionID, userID, shareToken, signedURL string, expiresAt time.Time, maxAccessCount *int) (string, error) {
	query := `
		INSERT INTO shared_links (conversion_id, user_id, share_token, signed_url, expires_at, max_access_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var shareID string
	err := s.db.QueryRowContext(ctx, query, conversionID, userID, shareToken, signedURL, expiresAt, maxAccessCount).Scan(&shareID)
	if err != nil {
		return "", fmt.Errorf("failed to create shared link: %w", err)
	}

	return shareID, nil
}

// GetSharedLink retrieves a shared link by ID
func (s *StoreImpl) GetSharedLink(ctx context.Context, shareID string) (SharedLink, error) {
	query := `
		SELECT id, conversion_id, user_id, share_token, signed_url, expires_at, 
		       access_count, max_access_count, is_active, created_at, updated_at
		FROM shared_links
		WHERE id = $1
	`

	var link SharedLink
	var maxAccessCount sql.NullInt32

	err := s.db.QueryRowContext(ctx, query, shareID).Scan(
		&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken, &link.SignedURL,
		&link.ExpiresAt, &link.AccessCount, &maxAccessCount, &link.IsActive,
		&link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return SharedLink{}, fmt.Errorf("shared link not found")
		}
		return SharedLink{}, fmt.Errorf("failed to get shared link: %w", err)
	}

	if maxAccessCount.Valid {
		count := int(maxAccessCount.Int32)
		link.MaxAccessCount = &count
	}

	return link, nil
}

// GetSharedLinkByToken retrieves a shared link by token
func (s *StoreImpl) GetSharedLinkByToken(ctx context.Context, shareToken string) (ActiveSharedLink, error) {
	query := `
		SELECT 
			sl.id, sl.conversion_id, sl.user_id, sl.share_token, sl.signed_url,
			sl.expires_at, sl.access_count, sl.max_access_count, sl.is_active, sl.created_at, sl.updated_at,
			c.status, c.result_image_id, i.original_url, i.file_name, i.file_size, i.mime_type,
			EXTRACT(EPOCH FROM (sl.expires_at - NOW()))::INTEGER as seconds_until_expiry
		FROM shared_links sl
		LEFT JOIN conversions c ON sl.conversion_id = c.id
		LEFT JOIN images i ON c.result_image_id = i.id
		WHERE sl.share_token = $1
	`

	var link ActiveSharedLink
	var maxAccessCount sql.NullInt32
	var resultImageID sql.NullString
	var resultImageURL sql.NullString
	var resultImageName sql.NullString
	var resultImageSize sql.NullInt64
	var resultImageMimeType sql.NullString

	err := s.db.QueryRowContext(ctx, query, shareToken).Scan(
		&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken, &link.SignedURL,
		&link.ExpiresAt, &link.AccessCount, &maxAccessCount, &link.IsActive,
		&link.CreatedAt, &link.UpdatedAt, &link.ConversionStatus, &resultImageID,
		&resultImageURL, &resultImageName, &resultImageSize, &resultImageMimeType,
		&link.SecondsUntilExpiry,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActiveSharedLink{}, fmt.Errorf("shared link not found")
		}
		return ActiveSharedLink{}, fmt.Errorf("failed to get shared link by token: %w", err)
	}

	if maxAccessCount.Valid {
		count := int(maxAccessCount.Int32)
		link.MaxAccessCount = &count
	}

	if resultImageID.Valid {
		link.ResultImageID = resultImageID.String
	}
	if resultImageURL.Valid {
		link.ResultImageURL = resultImageURL.String
	}
	if resultImageName.Valid {
		link.ResultImageName = resultImageName.String
	}
	if resultImageSize.Valid {
		link.ResultImageSize = resultImageSize.Int64
	}
	if resultImageMimeType.Valid {
		link.ResultImageMimeType = resultImageMimeType.String
	}

	return link, nil
}

// UpdateSharedLink updates a shared link
func (s *StoreImpl) UpdateSharedLink(ctx context.Context, shareID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE shared_links 
		SET %s
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	args = append(args, shareID)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update shared link: %w", err)
	}

	return nil
}

// DeactivateSharedLink deactivates a shared link
func (s *StoreImpl) DeactivateSharedLink(ctx context.Context, shareID, userID string) error {
	query := `
		UPDATE shared_links 
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, shareID, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate shared link: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("shared link not found or not owned by user")
	}

	return nil
}

// ListUserSharedLinks lists user's shared links
func (s *StoreImpl) ListUserSharedLinks(ctx context.Context, userID string, limit, offset int) ([]ActiveSharedLink, error) {
	query := `
		SELECT 
			sl.id, sl.conversion_id, sl.user_id, sl.share_token, sl.signed_url,
			sl.expires_at, sl.access_count, sl.max_access_count, sl.created_at,
			c.status, c.result_image_id, i.original_url, i.file_name, i.file_size, i.mime_type,
			EXTRACT(EPOCH FROM (sl.expires_at - NOW()))::INTEGER as seconds_until_expiry
		FROM shared_links sl
		LEFT JOIN conversions c ON sl.conversion_id = c.id
		LEFT JOIN images i ON c.result_image_id = i.id
		WHERE sl.user_id = $1
		ORDER BY sl.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list shared links: %w", err)
	}
	defer rows.Close()

	var links []ActiveSharedLink
	for rows.Next() {
		var link ActiveSharedLink
		var maxAccessCount sql.NullInt32
		var resultImageID sql.NullString
		var resultImageURL sql.NullString
		var resultImageName sql.NullString
		var resultImageSize sql.NullInt64
		var resultImageMimeType sql.NullString

		err := rows.Scan(
			&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken, &link.SignedURL,
			&link.ExpiresAt, &link.AccessCount, &maxAccessCount, &link.CreatedAt,
			&link.ConversionStatus, &resultImageID, &resultImageURL, &resultImageName,
			&resultImageSize, &resultImageMimeType, &link.SecondsUntilExpiry,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shared link: %w", err)
		}

		if maxAccessCount.Valid {
			count := int(maxAccessCount.Int32)
			link.MaxAccessCount = &count
		}

		if resultImageID.Valid {
			link.ResultImageID = resultImageID.String
		}
		if resultImageURL.Valid {
			link.ResultImageURL = resultImageURL.String
		}
		if resultImageName.Valid {
			link.ResultImageName = resultImageName.String
		}
		if resultImageSize.Valid {
			link.ResultImageSize = resultImageSize.Int64
		}
		if resultImageMimeType.Valid {
			link.ResultImageMimeType = resultImageMimeType.String
		}

		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared links: %w", err)
	}

	return links, nil
}

// LogSharedLinkAccess logs an access attempt to a shared link
func (s *StoreImpl) LogSharedLinkAccess(ctx context.Context, sharedLinkID string, req AccessShareRequest, success bool, errorMessage string) error {
	query := `
		INSERT INTO shared_link_access_logs (
			shared_link_id, ip_address, user_agent, referer, 
			access_type, success, error_message, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	metadata := map[string]interface{}{
		"access_type": req.AccessType,
		"timestamp":   time.Now(),
	}

	_, err := s.db.ExecContext(ctx, query,
		sharedLinkID, req.IPAddress, req.UserAgent, req.Referer,
		req.AccessType, success, errorMessage, metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to log shared link access: %w", err)
	}

	return nil
}

// GetSharedLinkStats gets statistics for shared links
func (s *StoreImpl) GetSharedLinkStats(ctx context.Context, userID, conversionID string) (SharedLinkStats, error) {
	query := `
		SELECT 
			COUNT(*)::INTEGER as total_links,
			COUNT(CASE WHEN sl.is_active = true AND sl.expires_at > NOW() THEN 1 END)::INTEGER as active_links,
			COUNT(CASE WHEN sl.expires_at <= NOW() THEN 1 END)::INTEGER as expired_links,
			COALESCE(SUM(sl.access_count), 0) as total_access_count,
			COALESCE(COUNT(DISTINCT sla.ip_address), 0) as unique_ip_addresses
		FROM shared_links sl
		LEFT JOIN shared_link_access_logs sla ON sl.id = sla.shared_link_id AND sla.success = true
		WHERE ($1 IS NULL OR sl.user_id = $1)
		AND ($2 IS NULL OR sl.conversion_id = $2)
	`

	var stats SharedLinkStats
	err := s.db.QueryRowContext(ctx, query, userID, conversionID).Scan(
		&stats.TotalSharedLinks, &stats.ActiveSharedLinks, &stats.ExpiredLinks,
		&stats.TotalAccessCount, &stats.UniqueIPAddresses,
	)
	if err != nil {
		return SharedLinkStats{}, fmt.Errorf("failed to get shared link stats: %w", err)
	}

	return stats, nil
}

// CleanupExpiredLinks removes expired shared links
func (s *StoreImpl) CleanupExpiredLinks(ctx context.Context) (int, error) {
	query := `
		WITH deleted_links AS (
			DELETE FROM shared_links 
			WHERE expires_at < NOW() - INTERVAL '1 hour'
			RETURNING id
		)
		DELETE FROM shared_link_access_logs 
		WHERE shared_link_id IN (SELECT id FROM deleted_links)
	`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired links: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get cleanup count: %w", err)
	}

	return int(count), nil
}

// ValidateSharedLinkAccess validates access to a shared link
func (s *StoreImpl) ValidateSharedLinkAccess(ctx context.Context, shareToken string) (bool, error) {
	query := `
		SELECT 
			CASE 
				WHEN sl.is_active = true 
					AND sl.expires_at > NOW() 
					AND (sl.max_access_count IS NULL OR sl.access_count < sl.max_access_count)
				THEN true 
				ELSE false 
			END as is_valid
		FROM shared_links sl
		WHERE sl.share_token = $1
	`

	var isValid bool
	err := s.db.QueryRowContext(ctx, query, shareToken).Scan(&isValid)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("shared link not found")
		}
		return false, fmt.Errorf("failed to validate shared link access: %w", err)
	}

	return isValid, nil
}

// GetSharedLinkDetails gets detailed information about a shared link
func (s *StoreImpl) GetSharedLinkDetails(ctx context.Context, shareToken string) (SharedLinkDetails, error) {
	query := `
		SELECT 
			sl.id, sl.conversion_id, sl.user_id, sl.share_token, sl.signed_url,
			sl.expires_at, sl.max_access_count, sl.access_count, sl.is_active,
			sl.created_at, sl.updated_at, c.status, u.name
		FROM shared_links sl
		LEFT JOIN conversions c ON sl.conversion_id = c.id
		LEFT JOIN users u ON sl.user_id = u.id
		WHERE sl.share_token = $1
	`

	var details SharedLinkDetails
	var maxAccessCount sql.NullInt32
	var userName sql.NullString

	err := s.db.QueryRowContext(ctx, query, shareToken).Scan(
		&details.ID, &details.ConversionID, &details.UserID, &details.ShareToken,
		&details.SignedURL, &details.ExpiresAt, &maxAccessCount, &details.AccessCount,
		&details.IsActive, &details.CreatedAt, &details.UpdatedAt,
		&details.ConversionStatus, &userName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return SharedLinkDetails{}, fmt.Errorf("shared link not found")
		}
		return SharedLinkDetails{}, fmt.Errorf("failed to get shared link details: %w", err)
	}

	if maxAccessCount.Valid {
		count := int(maxAccessCount.Int32)
		details.MaxAccessCount = &count
	}

	if userName.Valid {
		details.UserName = userName.String
	}

	return details, nil
}

// GetSharedLinkAccessLogs gets access logs for a shared link
func (s *StoreImpl) GetSharedLinkAccessLogs(ctx context.Context, shareID string, limit, offset int) ([]AccessLog, error) {
	query := `
		SELECT id, shared_link_id, ip_address, user_agent, accessed_at, success, error_message
		FROM shared_link_access_logs 
		WHERE shared_link_id = $1
		ORDER BY accessed_at DESC
		LIMIT $2 OFFSET $3
	`

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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating access logs: %w", err)
	}

	return logs, nil
}

// GetPopularSharedLinks gets most accessed shared links
func (s *StoreImpl) GetPopularSharedLinks(ctx context.Context, limit int) ([]PopularSharedLink, error) {
	query := `
		SELECT 
			sl.id, sl.conversion_id, sl.user_id, sl.share_token, sl.access_count,
			sl.created_at, u.name, c.status
		FROM shared_links sl
		LEFT JOIN users u ON sl.user_id = u.id
		LEFT JOIN conversions c ON sl.conversion_id = c.id
		WHERE sl.is_active = true AND sl.access_count > 0
		ORDER BY sl.access_count DESC, sl.created_at DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular shared links: %w", err)
	}
	defer rows.Close()

	var links []PopularSharedLink
	for rows.Next() {
		var link PopularSharedLink
		var userName sql.NullString
		var conversionStatus sql.NullString

		err := rows.Scan(
			&link.ID, &link.ConversionID, &link.UserID, &link.ShareToken,
			&link.AccessCount, &link.CreatedAt, &userName, &conversionStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan popular shared link: %w", err)
		}

		if userName.Valid {
			link.UserName = userName.String
		}
		if conversionStatus.Valid {
			link.ConversionStatus = conversionStatus.String
		}

		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating popular shared links: %w", err)
	}

	return links, nil
}
