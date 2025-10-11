package notification

import (
	"context"
)

// NotificationService defines the interface for notification operations
type NotificationService interface {
	// Core notification operations
	CreateNotification(ctx context.Context, req CreateNotificationRequest) (Notification, error)
	GetNotification(ctx context.Context, notificationID string) (Notification, error)
	ListNotifications(ctx context.Context, req NotificationListRequest) (NotificationListResponse, error)
	MarkAsRead(ctx context.Context, notificationID, userID string) error
	DeleteNotification(ctx context.Context, notificationID, userID string) error

	// Specific notification types
	SendConversionStarted(ctx context.Context, userID, conversionID string) error
	SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error
	SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error
	SendQuotaExhausted(ctx context.Context, userID string, quotaType string) error
	SendQuotaWarning(ctx context.Context, userID string, quotaType string, remaining int) error
	SendQuotaReset(ctx context.Context, userID string) error
	SendPaymentSuccess(ctx context.Context, userID, paymentID, planName string) error
	SendPaymentFailed(ctx context.Context, userID, paymentID, reason string) error
	SendPlanActivated(ctx context.Context, userID, planName string) error
	SendPlanExpired(ctx context.Context, userID, planName string) error
	SendCriticalError(ctx context.Context, errorType, message string, metadata map[string]interface{}) error
	SendSystemMaintenance(ctx context.Context, message string, scheduledFor *string) error

	// User preferences
	GetNotificationPreferences(ctx context.Context, userID string) (NotificationPreference, error)
	UpdateNotificationPreferences(ctx context.Context, userID string, req UpdateNotificationPreferenceRequest) error

	// Statistics
	GetNotificationStats(ctx context.Context, timeRange string) (NotificationStats, error)

	// WebSocket operations
	BroadcastToUser(ctx context.Context, userID string, message WebSocketMessage) error
	BroadcastToAll(ctx context.Context, message WebSocketMessage) error
}

// NotificationStore defines the interface for notification data operations
type NotificationStore interface {
	// Notification operations
	CreateNotification(ctx context.Context, notification Notification) error
	GetNotification(ctx context.Context, notificationID string) (Notification, error)
	ListNotifications(ctx context.Context, req NotificationListRequest) (NotificationListResponse, error)
	UpdateNotification(ctx context.Context, notificationID string, updates map[string]interface{}) error
	DeleteNotification(ctx context.Context, notificationID string) error
	MarkAsRead(ctx context.Context, notificationID, userID string) error

	// Delivery operations
	CreateDelivery(ctx context.Context, delivery NotificationDelivery) error
	UpdateDelivery(ctx context.Context, deliveryID string, updates map[string]interface{}) error
	GetFailedDeliveries(ctx context.Context, limit int) ([]NotificationDelivery, error)
	GetDeliveriesByNotification(ctx context.Context, notificationID string) ([]NotificationDelivery, error)

	// Preference operations
	GetNotificationPreferences(ctx context.Context, userID string) (NotificationPreference, error)
	UpdateNotificationPreferences(ctx context.Context, userID string, prefs NotificationPreference) error
	CreateNotificationPreferences(ctx context.Context, prefs NotificationPreference) error

	// Template operations
	GetTemplate(ctx context.Context, notificationType NotificationType, channel NotificationChannel) (NotificationTemplate, error)
	CreateTemplate(ctx context.Context, template NotificationTemplate) error
	UpdateTemplate(ctx context.Context, templateID string, updates map[string]interface{}) error
	ListTemplates(ctx context.Context) ([]NotificationTemplate, error)

	// Statistics
	GetNotificationStats(ctx context.Context, timeRange string) (NotificationStats, error)
}

// EmailProvider defines the interface for email sending
type EmailProvider interface {
	SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error
	SendTemplateEmail(ctx context.Context, to, templateID string, data map[string]interface{}) error
	ValidateEmail(email string) bool
}

// SMSProvider defines the interface for SMS sending
type SMSProvider interface {
	SendSMS(ctx context.Context, phone, message string) error
	SendTemplateSMS(ctx context.Context, phone, templateID string, data map[string]interface{}) error
	ValidatePhone(phone string) bool
}

// TelegramProvider defines the interface for Telegram messaging
type TelegramProvider interface {
	SendMessage(ctx context.Context, chatID, message string) error
	SendTemplateMessage(ctx context.Context, chatID, templateID string, data map[string]interface{}) error
	SendPhoto(ctx context.Context, chatID, photoURL, caption string) error
	SendDocument(ctx context.Context, chatID, documentURL, caption string) error
	SetWebhook(ctx context.Context, webhookURL string) error
	GetUpdates(ctx context.Context) ([]TelegramUpdate, error)
}

// WebSocketProvider defines the interface for WebSocket operations
type WebSocketProvider interface {
	BroadcastToUser(ctx context.Context, userID string, message WebSocketMessage) error
	BroadcastToAll(ctx context.Context, message WebSocketMessage) error
	GetConnectedUsers(ctx context.Context) ([]string, error)
	IsUserConnected(ctx context.Context, userID string) bool
	CloseUserConnection(ctx context.Context, userID string) error
}

// TemplateEngine defines the interface for template processing
type TemplateEngine interface {
	ProcessTemplate(template string, data map[string]interface{}) (string, error)
	ProcessEmailTemplate(templateID string, data map[string]interface{}) (subject, body string, err error)
	ProcessSMSTemplate(templateID string, data map[string]interface{}) (string, error)
	ProcessTelegramTemplate(templateID string, data map[string]interface{}) (string, error)
}

// QuotaService defines the interface for quota operations
type QuotaService interface {
	GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error)
	CheckUserQuota(ctx context.Context, userID string) (interface{}, error)
}

// UserService defines the interface for user operations
type UserService interface {
	GetUser(ctx context.Context, userID string) (interface{}, error)
	GetUserByPhone(ctx context.Context, phone string) (interface{}, error)
	GetUserByEmail(ctx context.Context, email string) (interface{}, error)
}

// ConversionService defines the interface for conversion operations
type ConversionService interface {
	GetConversion(ctx context.Context, conversionID string) (interface{}, error)
	GetConversionWithDetails(ctx context.Context, conversionID string) (interface{}, error)
}

// PaymentService defines the interface for payment operations
type PaymentService interface {
	GetPayment(ctx context.Context, paymentID string) (interface{}, error)
	GetUserPayments(ctx context.Context, userID string) (interface{}, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogNotificationSent(ctx context.Context, userID *string, notificationType NotificationType, channel NotificationChannel, success bool, errorMessage *string) error
	LogNotificationRead(ctx context.Context, userID, notificationID string) error
	LogNotificationDeleted(ctx context.Context, userID, notificationID string) error
}

// MetricsCollector defines the interface for collecting notification metrics
type MetricsCollector interface {
	RecordNotificationSent(ctx context.Context, notificationType NotificationType, channel NotificationChannel) error
	RecordNotificationDelivered(ctx context.Context, notificationType NotificationType, channel NotificationChannel, deliveryTimeMs int64) error
	RecordNotificationFailed(ctx context.Context, notificationType NotificationType, channel NotificationChannel, errorType string) error
	RecordNotificationRead(ctx context.Context, notificationType NotificationType, channel NotificationChannel) error
	GetNotificationMetrics(ctx context.Context, timeRange string) (map[string]interface{}, error)
}

// RetryHandler defines the interface for retry operations
type RetryHandler interface {
	ShouldRetry(ctx context.Context, delivery NotificationDelivery, err error) bool
	GetRetryDelay(ctx context.Context, delivery NotificationDelivery) int64 // milliseconds
	IncrementRetryCount(ctx context.Context, deliveryID string) error
}

// TelegramUpdate represents a Telegram webhook update
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      *TelegramChat `json:"chat,omitempty"`
	Text      string        `json:"text,omitempty"`
	Date      int64         `json:"date"`
}

// TelegramUser represents a Telegram user
type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// TelegramChat represents a Telegram chat
type TelegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}
