package notification

import (
	"time"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// Conversion notifications
	NotificationTypeConversionStarted   NotificationType = "conversion_started"
	NotificationTypeConversionCompleted NotificationType = "conversion_completed"
	NotificationTypeConversionFailed    NotificationType = "conversion_failed"

	// Quota notifications
	NotificationTypeQuotaExhausted NotificationType = "quota_exhausted"
	NotificationTypeQuotaWarning   NotificationType = "quota_warning"
	NotificationTypeQuotaReset     NotificationType = "quota_reset"

	// Payment notifications
	NotificationTypePaymentSuccess NotificationType = "payment_success"
	NotificationTypePaymentFailed  NotificationType = "payment_failed"
	NotificationTypePlanActivated  NotificationType = "plan_activated"
	NotificationTypePlanExpired    NotificationType = "plan_expired"

	// System notifications
	NotificationTypeSystemMaintenance NotificationType = "system_maintenance"
	NotificationTypeSystemError       NotificationType = "system_error"
	NotificationTypeCriticalError     NotificationType = "critical_error"

	// User notifications
	NotificationTypeWelcome         NotificationType = "welcome"
	NotificationTypeProfileUpdated  NotificationType = "profile_updated"
	NotificationTypePasswordChanged NotificationType = "password_changed"
)

// NotificationChannel represents the delivery channel
type NotificationChannel string

const (
	ChannelEmail     NotificationChannel = "email"
	ChannelSMS       NotificationChannel = "sms"
	ChannelTelegram  NotificationChannel = "telegram"
	ChannelWebSocket NotificationChannel = "websocket"
	ChannelPush      NotificationChannel = "push"
)

// NotificationPriority represents the priority level
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// Notification represents a notification in the system
type Notification struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"userId,omitempty"` // nil for system-wide notifications
	Type         NotificationType       `json:"type"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Channels     []NotificationChannel  `json:"channels"`
	Priority     NotificationPriority   `json:"priority"`
	Status       NotificationStatus     `json:"status"`
	CreatedAt    time.Time              `json:"createdAt"`
	ScheduledFor *time.Time             `json:"scheduledFor,omitempty"`
	SentAt       *time.Time             `json:"sentAt,omitempty"`
	ReadAt       *time.Time             `json:"readAt,omitempty"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
}

// NotificationStatus represents the delivery status
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSending   NotificationStatus = "sending"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusRead      NotificationStatus = "read"
	StatusExpired   NotificationStatus = "expired"
)

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID        string              `json:"id"`
	Type      NotificationType    `json:"type"`
	Channel   NotificationChannel `json:"channel"`
	Subject   string              `json:"subject"`
	Body      string              `json:"body"`
	Variables []string            `json:"variables"` // List of template variables
	IsActive  bool                `json:"isActive"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	UserID           string                    `json:"userId"`
	EmailEnabled     bool                      `json:"emailEnabled"`
	SMSEnabled       bool                      `json:"smsEnabled"`
	TelegramEnabled  bool                      `json:"telegramEnabled"`
	WebSocketEnabled bool                      `json:"websocketEnabled"`
	PushEnabled      bool                      `json:"pushEnabled"`
	Preferences      map[NotificationType]bool `json:"preferences"`               // Type -> enabled
	QuietHoursStart  *string                   `json:"quietHoursStart,omitempty"` // Format: "HH:MM"
	QuietHoursEnd    *string                   `json:"quietHoursEnd,omitempty"`   // Format: "HH:MM"
	Timezone         string                    `json:"timezone"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
}

// NotificationDelivery represents a specific delivery attempt
type NotificationDelivery struct {
	ID             string              `json:"id"`
	NotificationID string              `json:"notificationId"`
	Channel        NotificationChannel `json:"channel"`
	Recipient      string              `json:"recipient"` // email, phone, telegram_id, etc.
	Status         NotificationStatus  `json:"status"`
	ErrorMessage   *string             `json:"errorMessage,omitempty"`
	SentAt         *time.Time          `json:"sentAt,omitempty"`
	DeliveredAt    *time.Time          `json:"deliveredAt,omitempty"`
	ReadAt         *time.Time          `json:"readAt,omitempty"`
	RetryCount     int                 `json:"retryCount"`
	NextRetryAt    *time.Time          `json:"nextRetryAt,omitempty"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
}

// WebSocketMessage represents a real-time message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// TelegramConfig represents Telegram bot configuration
type TelegramConfig struct {
	BotToken     string `json:"botToken"`
	ChatID       string `json:"chatId"` // For admin alerts
	Enabled      bool   `json:"enabled"`
	RetryCount   int    `json:"retryCount"`
	RetryDelayMs int    `json:"retryDelayMs"`
}

// EmailConfig represents email configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUsername string `json:"smtpUsername"`
	SMTPPassword string `json:"smtpPassword"`
	FromEmail    string `json:"fromEmail"`
	FromName     string `json:"fromName"`
	Enabled      bool   `json:"enabled"`
	Username     string `json:"username"` // Alias for SMTPUsername for backward compatibility
	Password     string `json:"password"` // Alias for SMTPPassword for backward compatibility
}

// NotificationConfig represents the overall notification configuration
type NotificationConfig struct {
	Email     EmailConfig     `json:"email"`
	Telegram  TelegramConfig  `json:"telegram"`
	SMS       SMSConfig       `json:"sms"`
	WebSocket WebSocketConfig `json:"websocket"`
	Retry     RetryConfig     `json:"retry"`
}

// SMSConfig represents SMS configuration
type SMSConfig struct {
	Provider   string `json:"provider"`
	APIKey     string `json:"apiKey"`
	TemplateID int    `json:"templateId"`
	Enabled    bool   `json:"enabled"`
	RetryCount int    `json:"retryCount"`
}

// WebSocketConfig represents WebSocket configuration
type WebSocketConfig struct {
	Enabled        bool `json:"enabled"`
	Port           int  `json:"port"`
	MaxConnections int  `json:"maxConnections"`
	PingInterval   int  `json:"pingInterval"` // seconds
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries int `json:"maxRetries"`
	BaseDelay  int `json:"baseDelay"` // milliseconds
	MaxDelay   int `json:"maxDelay"`  // milliseconds
}

// CreateNotificationRequest represents a request to create a notification
type CreateNotificationRequest struct {
	UserID       *string                `json:"userId,omitempty"`
	Type         NotificationType       `json:"type"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Channels     []NotificationChannel  `json:"channels,omitempty"`
	Priority     NotificationPriority   `json:"priority,omitempty"`
	ScheduledFor *time.Time             `json:"scheduledFor,omitempty"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
}

// NotificationListRequest represents a request to list notifications
type NotificationListRequest struct {
	UserID   *string               `json:"userId,omitempty"`
	Type     *NotificationType     `json:"type,omitempty"`
	Status   *NotificationStatus   `json:"status,omitempty"`
	Channel  *NotificationChannel  `json:"channel,omitempty"`
	Priority *NotificationPriority `json:"priority,omitempty"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	From     *time.Time            `json:"from,omitempty"`
	To       *time.Time            `json:"to,omitempty"`
}

// NotificationListResponse represents the response for listing notifications
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	PageSize      int            `json:"pageSize"`
	TotalPages    int            `json:"totalPages"`
}

// UpdateNotificationPreferenceRequest represents a request to update notification preferences
type UpdateNotificationPreferenceRequest struct {
	EmailEnabled     *bool                     `json:"emailEnabled,omitempty"`
	SMSEnabled       *bool                     `json:"smsEnabled,omitempty"`
	TelegramEnabled  *bool                     `json:"telegramEnabled,omitempty"`
	WebSocketEnabled *bool                     `json:"websocketEnabled,omitempty"`
	PushEnabled      *bool                     `json:"pushEnabled,omitempty"`
	Preferences      map[NotificationType]bool `json:"preferences,omitempty"`
	QuietHoursStart  *string                   `json:"quietHoursStart,omitempty"`
	QuietHoursEnd    *string                   `json:"quietHoursEnd,omitempty"`
	Timezone         *string                   `json:"timezone,omitempty"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	TotalSent             int64                          `json:"totalSent"`
	TotalDelivered        int64                          `json:"totalDelivered"`
	TotalFailed           int64                          `json:"totalFailed"`
	TotalRead             int64                          `json:"totalRead"`
	ByChannel             map[NotificationChannel]int64  `json:"byChannel"`
	ByType                map[NotificationType]int64     `json:"byType"`
	ByPriority            map[NotificationPriority]int64 `json:"byPriority"`
	AverageDeliveryTimeMs int64                          `json:"averageDeliveryTimeMs"`
}
