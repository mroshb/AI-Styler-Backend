package notification

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// Store implements NotificationStore interface
type Store struct {
	db *sql.DB
}

// NewStore creates a new notification store
func NewStore(db *sql.DB) NotificationStore {
	return &Store{db: db}
}

// CreateNotification creates a new notification
func (s Store) CreateNotification(ctx context.Context, notification Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, type, title, message, data, channels, priority, 
			status, created_at, scheduled_for, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	channels := make([]string, len(notification.Channels))
	for i, channel := range notification.Channels {
		channels[i] = string(channel)
	}

	_, err := s.db.ExecContext(ctx, query,
		notification.ID,
		notification.UserID,
		string(notification.Type),
		notification.Title,
		notification.Message,
		pq.Array(notification.Data),
		pq.Array(channels),
		string(notification.Priority),
		string(notification.Status),
		notification.CreatedAt,
		notification.ScheduledFor,
		notification.ExpiresAt,
	)

	return err
}

// GetNotification retrieves a notification by ID
func (s Store) GetNotification(ctx context.Context, notificationID string) (Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, channels, priority, 
		       status, created_at, scheduled_for, sent_at, read_at, expires_at
		FROM notifications 
		WHERE id = $1`

	var notification Notification
	var userID sql.NullString
	var scheduledFor, sentAt, readAt, expiresAt sql.NullTime
	var data, channels []string

	err := s.db.QueryRowContext(ctx, query, notificationID).Scan(
		&notification.ID,
		&userID,
		&notification.Type,
		&notification.Title,
		&notification.Message,
		pq.Array(&data),
		pq.Array(&channels),
		&notification.Priority,
		&notification.Status,
		&notification.CreatedAt,
		&scheduledFor,
		&sentAt,
		&readAt,
		&expiresAt,
	)

	if err != nil {
		return Notification{}, err
	}

	if userID.Valid {
		notification.UserID = &userID.String
	}
	if scheduledFor.Valid {
		notification.ScheduledFor = &scheduledFor.Time
	}
	if sentAt.Valid {
		notification.SentAt = &sentAt.Time
	}
	if readAt.Valid {
		notification.ReadAt = &readAt.Time
	}
	if expiresAt.Valid {
		notification.ExpiresAt = &expiresAt.Time
	}

	// Convert channels
	notification.Channels = make([]NotificationChannel, len(channels))
	for i, channel := range channels {
		notification.Channels[i] = NotificationChannel(channel)
	}

	// Convert data to map
	notification.Data = make(map[string]interface{})
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			notification.Data[data[i]] = data[i+1]
		}
	}

	return notification, nil
}

// ListNotifications lists notifications based on criteria
func (s Store) ListNotifications(ctx context.Context, req NotificationListRequest) (NotificationListResponse, error) {
	query := `
		SELECT id, user_id, type, title, message, data, channels, priority, 
		       status, created_at, scheduled_for, sent_at, read_at, expires_at
		FROM notifications 
		WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if req.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *req.UserID)
		argIndex++
	}

	if req.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, string(*req.Type))
		argIndex++
	}

	if req.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*req.Status))
		argIndex++
	}

	if req.Channel != nil {
		query += fmt.Sprintf(" AND $%d = ANY(channels)", argIndex)
		args = append(args, string(*req.Channel))
		argIndex++
	}

	if req.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, string(*req.Priority))
		argIndex++
	}

	if req.From != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *req.From)
		argIndex++
	}

	if req.To != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *req.To)
		argIndex++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS count_query"
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return NotificationListResponse{}, err
	}

	// Add pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return NotificationListResponse{}, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		var userID sql.NullString
		var scheduledFor, sentAt, readAt, expiresAt sql.NullTime
		var data, channels []string

		err := rows.Scan(
			&notification.ID,
			&userID,
			&notification.Type,
			&notification.Title,
			&notification.Message,
			pq.Array(&data),
			pq.Array(&channels),
			&notification.Priority,
			&notification.Status,
			&notification.CreatedAt,
			&scheduledFor,
			&sentAt,
			&readAt,
			&expiresAt,
		)

		if err != nil {
			return NotificationListResponse{}, err
		}

		if userID.Valid {
			notification.UserID = &userID.String
		}
		if scheduledFor.Valid {
			notification.ScheduledFor = &scheduledFor.Time
		}
		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}
		if readAt.Valid {
			notification.ReadAt = &readAt.Time
		}
		if expiresAt.Valid {
			notification.ExpiresAt = &expiresAt.Time
		}

		// Convert channels
		notification.Channels = make([]NotificationChannel, len(channels))
		for i, channel := range channels {
			notification.Channels[i] = NotificationChannel(channel)
		}

		// Convert data to map
		notification.Data = make(map[string]interface{})
		for i := 0; i < len(data); i += 2 {
			if i+1 < len(data) {
				notification.Data[data[i]] = data[i+1]
			}
		}

		notifications = append(notifications, notification)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return NotificationListResponse{
		Notifications: notifications,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// UpdateNotification updates a notification
func (s Store) UpdateNotification(ctx context.Context, notificationID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE notifications SET %s WHERE id = $%d",
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf("UPDATE notifications SET %s, %s WHERE id = $%d",
			query[21:len(query)-12], setParts[i], argIndex+1)
		argIndex++
	}

	args = append(args, notificationID)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteNotification deletes a notification
func (s Store) DeleteNotification(ctx context.Context, notificationID string) error {
	query := "DELETE FROM notifications WHERE id = $1"
	_, err := s.db.ExecContext(ctx, query, notificationID)
	return err
}

// MarkAsRead marks a notification as read
func (s Store) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	query := `
		UPDATE notifications 
		SET status = 'read', read_at = NOW() 
		WHERE id = $1 AND user_id = $2`

	_, err := s.db.ExecContext(ctx, query, notificationID, userID)
	return err
}

// CreateDelivery creates a delivery record
func (s Store) CreateDelivery(ctx context.Context, delivery NotificationDelivery) error {
	query := `
		INSERT INTO notification_deliveries (
			id, notification_id, channel, recipient, status, error_message,
			sent_at, delivered_at, read_at, retry_count, next_retry_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := s.db.ExecContext(ctx, query,
		delivery.ID,
		delivery.NotificationID,
		string(delivery.Channel),
		delivery.Recipient,
		string(delivery.Status),
		delivery.ErrorMessage,
		delivery.SentAt,
		delivery.DeliveredAt,
		delivery.ReadAt,
		delivery.RetryCount,
		delivery.NextRetryAt,
		delivery.CreatedAt,
		delivery.UpdatedAt,
	)

	return err
}

// UpdateDelivery updates a delivery record
func (s Store) UpdateDelivery(ctx context.Context, deliveryID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE notification_deliveries SET %s WHERE id = $%d",
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf("UPDATE notification_deliveries SET %s, %s WHERE id = $%d",
			query[30:len(query)-12], setParts[i], argIndex+1)
		argIndex++
	}

	args = append(args, deliveryID)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetFailedDeliveries gets failed delivery records
func (s Store) GetFailedDeliveries(ctx context.Context, limit int) ([]NotificationDelivery, error) {
	query := `
		SELECT id, notification_id, channel, recipient, status, error_message,
		       sent_at, delivered_at, read_at, retry_count, next_retry_at, created_at, updated_at
		FROM notification_deliveries 
		WHERE status = 'failed' 
		ORDER BY created_at ASC 
		LIMIT $1`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []NotificationDelivery
	for rows.Next() {
		var delivery NotificationDelivery
		var sentAt, deliveredAt, readAt, nextRetryAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&delivery.ID,
			&delivery.NotificationID,
			&delivery.Channel,
			&delivery.Recipient,
			&delivery.Status,
			&errorMessage,
			&sentAt,
			&deliveredAt,
			&readAt,
			&delivery.RetryCount,
			&nextRetryAt,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if errorMessage.Valid {
			delivery.ErrorMessage = &errorMessage.String
		}
		if sentAt.Valid {
			delivery.SentAt = &sentAt.Time
		}
		if deliveredAt.Valid {
			delivery.DeliveredAt = &deliveredAt.Time
		}
		if readAt.Valid {
			delivery.ReadAt = &readAt.Time
		}
		if nextRetryAt.Valid {
			delivery.NextRetryAt = &nextRetryAt.Time
		}

		deliveries = append(deliveries, delivery)
	}

	return deliveries, nil
}

// GetDeliveriesByNotification gets delivery records for a notification
func (s Store) GetDeliveriesByNotification(ctx context.Context, notificationID string) ([]NotificationDelivery, error) {
	query := `
		SELECT id, notification_id, channel, recipient, status, error_message,
		       sent_at, delivered_at, read_at, retry_count, next_retry_at, created_at, updated_at
		FROM notification_deliveries 
		WHERE notification_id = $1 
		ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, notificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []NotificationDelivery
	for rows.Next() {
		var delivery NotificationDelivery
		var sentAt, deliveredAt, readAt, nextRetryAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&delivery.ID,
			&delivery.NotificationID,
			&delivery.Channel,
			&delivery.Recipient,
			&delivery.Status,
			&errorMessage,
			&sentAt,
			&deliveredAt,
			&readAt,
			&delivery.RetryCount,
			&nextRetryAt,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if errorMessage.Valid {
			delivery.ErrorMessage = &errorMessage.String
		}
		if sentAt.Valid {
			delivery.SentAt = &sentAt.Time
		}
		if deliveredAt.Valid {
			delivery.DeliveredAt = &deliveredAt.Time
		}
		if readAt.Valid {
			delivery.ReadAt = &readAt.Time
		}
		if nextRetryAt.Valid {
			delivery.NextRetryAt = &nextRetryAt.Time
		}

		deliveries = append(deliveries, delivery)
	}

	return deliveries, nil
}

// GetNotificationPreferences gets user notification preferences
func (s Store) GetNotificationPreferences(ctx context.Context, userID string) (NotificationPreference, error) {
	query := `
		SELECT user_id, email_enabled, sms_enabled, telegram_enabled, websocket_enabled, 
		       push_enabled, preferences, quiet_hours_start, quiet_hours_end, timezone, created_at, updated_at
		FROM notification_preferences 
		WHERE user_id = $1`

	var prefs NotificationPreference
	var preferences map[string]bool

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.UserID,
		&prefs.EmailEnabled,
		&prefs.SMSEnabled,
		&prefs.TelegramEnabled,
		&prefs.WebSocketEnabled,
		&prefs.PushEnabled,
		&preferences,
		&prefs.QuietHoursStart,
		&prefs.QuietHoursEnd,
		&prefs.Timezone,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err != nil {
		return NotificationPreference{}, err
	}

	// Convert preferences map
	prefs.Preferences = make(map[NotificationType]bool)
	for key, value := range preferences {
		prefs.Preferences[NotificationType(key)] = value
	}

	return prefs, nil
}

// UpdateNotificationPreferences updates user notification preferences
func (s Store) UpdateNotificationPreferences(ctx context.Context, userID string, prefs NotificationPreference) error {
	query := `
		UPDATE notification_preferences 
		SET email_enabled = $2, sms_enabled = $3, telegram_enabled = $4, websocket_enabled = $5,
		    push_enabled = $6, preferences = $7, quiet_hours_start = $8, quiet_hours_end = $9,
		    timezone = $10, updated_at = $11
		WHERE user_id = $1`

	// Convert preferences map
	preferences := make(map[string]bool)
	for key, value := range prefs.Preferences {
		preferences[string(key)] = value
	}

	_, err := s.db.ExecContext(ctx, query,
		userID,
		prefs.EmailEnabled,
		prefs.SMSEnabled,
		prefs.TelegramEnabled,
		prefs.WebSocketEnabled,
		prefs.PushEnabled,
		preferences,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
		prefs.Timezone,
		prefs.UpdatedAt,
	)

	return err
}

// CreateNotificationPreferences creates user notification preferences
func (s Store) CreateNotificationPreferences(ctx context.Context, prefs NotificationPreference) error {
	query := `
		INSERT INTO notification_preferences (
			user_id, email_enabled, sms_enabled, telegram_enabled, websocket_enabled,
			push_enabled, preferences, quiet_hours_start, quiet_hours_end, timezone, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	// Convert preferences map
	preferences := make(map[string]bool)
	for key, value := range prefs.Preferences {
		preferences[string(key)] = value
	}

	_, err := s.db.ExecContext(ctx, query,
		prefs.UserID,
		prefs.EmailEnabled,
		prefs.SMSEnabled,
		prefs.TelegramEnabled,
		prefs.WebSocketEnabled,
		prefs.PushEnabled,
		preferences,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
		prefs.Timezone,
		prefs.CreatedAt,
		prefs.UpdatedAt,
	)

	return err
}

// GetTemplate gets a notification template
func (s Store) GetTemplate(ctx context.Context, notificationType NotificationType, channel NotificationChannel) (NotificationTemplate, error) {
	query := `
		SELECT id, type, channel, subject, body, variables, is_active, created_at, updated_at
		FROM notification_templates 
		WHERE type = $1 AND channel = $2 AND is_active = true`

	var template NotificationTemplate
	var variables []string

	err := s.db.QueryRowContext(ctx, query, string(notificationType), string(channel)).Scan(
		&template.ID,
		&template.Type,
		&template.Channel,
		&template.Subject,
		&template.Body,
		pq.Array(&variables),
		&template.IsActive,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		return NotificationTemplate{}, err
	}

	template.Variables = variables
	return template, nil
}

// CreateTemplate creates a notification template
func (s Store) CreateTemplate(ctx context.Context, template NotificationTemplate) error {
	query := `
		INSERT INTO notification_templates (
			id, type, channel, subject, body, variables, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.ExecContext(ctx, query,
		template.ID,
		string(template.Type),
		string(template.Channel),
		template.Subject,
		template.Body,
		pq.Array(template.Variables),
		template.IsActive,
		template.CreatedAt,
		template.UpdatedAt,
	)

	return err
}

// UpdateTemplate updates a notification template
func (s Store) UpdateTemplate(ctx context.Context, templateID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE notification_templates SET %s WHERE id = $%d",
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf("UPDATE notification_templates SET %s, %s WHERE id = $%d",
			query[32:len(query)-12], setParts[i], argIndex+1)
		argIndex++
	}

	args = append(args, templateID)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ListTemplates lists all notification templates
func (s Store) ListTemplates(ctx context.Context) ([]NotificationTemplate, error) {
	query := `
		SELECT id, type, channel, subject, body, variables, is_active, created_at, updated_at
		FROM notification_templates 
		ORDER BY type, channel`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []NotificationTemplate
	for rows.Next() {
		var template NotificationTemplate
		var variables []string

		err := rows.Scan(
			&template.ID,
			&template.Type,
			&template.Channel,
			&template.Subject,
			&template.Body,
			pq.Array(&variables),
			&template.IsActive,
			&template.CreatedAt,
			&template.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		template.Variables = variables
		templates = append(templates, template)
	}

	return templates, nil
}

// GetNotificationStats gets notification statistics
func (s Store) GetNotificationStats(ctx context.Context, timeRange string) (NotificationStats, error) {
	// This is a simplified implementation
	// In production, you'd have more sophisticated queries based on timeRange

	query := `
		SELECT 
			COUNT(*) as total_sent,
			COUNT(CASE WHEN status = 'delivered' THEN 1 END) as total_delivered,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as total_failed,
			COUNT(CASE WHEN status = 'read' THEN 1 END) as total_read
		FROM notification_deliveries 
		WHERE created_at >= NOW() - INTERVAL '1 day'`

	var stats NotificationStats
	err := s.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalSent,
		&stats.TotalDelivered,
		&stats.TotalFailed,
		&stats.TotalRead,
	)

	if err != nil {
		return NotificationStats{}, err
	}

	// Initialize maps
	stats.ByChannel = make(map[NotificationChannel]int64)
	stats.ByType = make(map[NotificationType]int64)
	stats.ByPriority = make(map[NotificationPriority]int64)

	return stats, nil
}
