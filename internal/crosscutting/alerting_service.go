package crosscutting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AlertConfig represents configuration for alerting system
type AlertConfig struct {
	// Telegram configuration
	TelegramEnabled  bool   `json:"telegram_enabled"`
	TelegramBotToken string `json:"telegram_bot_token"`
	TelegramChatID   string `json:"telegram_chat_id"`
	TelegramAPIURL   string `json:"telegram_api_url"`

	// Email configuration
	EmailEnabled bool     `json:"email_enabled"`
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	SMTPUsername string   `json:"smtp_username"`
	SMTPPassword string   `json:"smtp_password"`
	EmailFrom    string   `json:"email_from"`
	EmailTo      []string `json:"email_to"`

	// Webhook configuration
	WebhookEnabled bool   `json:"webhook_enabled"`
	WebhookURL     string `json:"webhook_url"`
	WebhookSecret  string `json:"webhook_secret"`

	// Alert thresholds
	CriticalThreshold time.Duration `json:"critical_threshold"`
	HighThreshold     time.Duration `json:"high_threshold"`
	MediumThreshold   time.Duration `json:"medium_threshold"`

	// Rate limiting for alerts
	AlertRateLimit  int           `json:"alert_rate_limit"`
	AlertRateWindow time.Duration `json:"alert_rate_window"`

	// Retry configuration
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`
}

// AlertType represents different types of alerts
type AlertType string

const (
	AlertTypeSecurity    AlertType = "security"
	AlertTypeSystem      AlertType = "system"
	AlertTypePerformance AlertType = "performance"
	AlertTypeQuota       AlertType = "quota"
	AlertTypePayment     AlertType = "payment"
	AlertTypeStorage     AlertType = "storage"
	AlertTypeAPI         AlertType = "api"
	AlertTypeDatabase    AlertType = "database"
	AlertTypeNetwork     AlertType = "network"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents an alert to be sent
type Alert struct {
	ID        string                 `json:"id"`
	Type      AlertType              `json:"type"`
	Severity  AlertSeverity          `json:"severity"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	Tags      []string               `json:"tags"`
}

// AlertResult represents the result of sending an alert
type AlertResult struct {
	Success    bool      `json:"success"`
	Channel    string    `json:"channel"`
	MessageID  string    `json:"message_id,omitempty"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"retry_count"`
}

// DefaultAlertConfig returns default alert configuration
func DefaultAlertConfig() *AlertConfig {
	return &AlertConfig{
		TelegramEnabled:   true,
		TelegramAPIURL:    "https://api.telegram.org/bot",
		EmailEnabled:      false,
		WebhookEnabled:    false,
		CriticalThreshold: 1 * time.Minute,
		HighThreshold:     5 * time.Minute,
		MediumThreshold:   15 * time.Minute,
		AlertRateLimit:    10,
		AlertRateWindow:   1 * time.Hour,
		MaxRetries:        3,
		RetryDelay:        5 * time.Second,
	}
}

// AlertingService provides comprehensive alerting functionality
type AlertingService struct {
	config      *AlertConfig
	client      *http.Client
	rateLimiter *AlertRateLimiter
}

// AlertRateLimiter provides rate limiting for alerts
type AlertRateLimiter struct {
	alerts map[string]time.Time
}

// NewAlertingService creates a new alerting service
func NewAlertingService(config *AlertConfig) *AlertingService {
	if config == nil {
		config = DefaultAlertConfig()
	}

	return &AlertingService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: &AlertRateLimiter{
			alerts: make(map[string]time.Time),
		},
	}
}

// SendAlert sends an alert through configured channels
func (as *AlertingService) SendAlert(ctx context.Context, alert *Alert) (*AlertResult, error) {
	// Check rate limiting
	if !as.rateLimiter.Allow(alert.ID) {
		return &AlertResult{
			Success: false,
			Error:   "Alert rate limited",
		}, nil
	}

	// Set timestamp if not set
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Set ID if not set
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
	}

	var results []*AlertResult

	// Send via Telegram
	if as.config.TelegramEnabled {
		result, err := as.sendTelegramAlert(ctx, alert)
		if err != nil {
			result = &AlertResult{
				Success: false,
				Channel: "telegram",
				Error:   err.Error(),
			}
		}
		results = append(results, result)
	}

	// Send via Email
	if as.config.EmailEnabled {
		result, err := as.sendEmailAlert(ctx, alert)
		if err != nil {
			result = &AlertResult{
				Success: false,
				Channel: "email",
				Error:   err.Error(),
			}
		}
		results = append(results, result)
	}

	// Send via Webhook
	if as.config.WebhookEnabled {
		result, err := as.sendWebhookAlert(ctx, alert)
		if err != nil {
			result = &AlertResult{
				Success: false,
				Channel: "webhook",
				Error:   err.Error(),
			}
		}
		results = append(results, result)
	}

	// Return first successful result or first error
	for _, result := range results {
		if result.Success {
			return result, nil
		}
	}

	if len(results) > 0 {
		return results[0], nil
	}

	return &AlertResult{
		Success: false,
		Error:   "No alert channels configured",
	}, nil
}

// SendSecurityAlert sends a security-specific alert
func (as *AlertingService) SendSecurityAlert(ctx context.Context, title, message, source, ipAddress string, metadata map[string]interface{}) error {
	alert := &Alert{
		Type:      AlertTypeSecurity,
		Severity:  AlertSeverityHigh,
		Title:     title,
		Message:   message,
		Source:    source,
		IPAddress: ipAddress,
		Metadata:  metadata,
		Tags:      []string{"security", "threat"},
	}

	_, err := as.SendAlert(ctx, alert)
	return err
}

// SendSystemAlert sends a system-specific alert
func (as *AlertingService) SendSystemAlert(ctx context.Context, title, message, source string, severity AlertSeverity, metadata map[string]interface{}) error {
	alert := &Alert{
		Type:     AlertTypeSystem,
		Severity: severity,
		Title:    title,
		Message:  message,
		Source:   source,
		Metadata: metadata,
		Tags:     []string{"system", "infrastructure"},
	}

	_, err := as.SendAlert(ctx, alert)
	return err
}

// SendQuotaAlert sends a quota-specific alert
func (as *AlertingService) SendQuotaAlert(ctx context.Context, userID, planName string, quotaType string, usage, limit int) error {
	alert := &Alert{
		Type:     AlertTypeQuota,
		Severity: AlertSeverityMedium,
		Title:    "Quota Limit Approaching",
		Message:  fmt.Sprintf("User %s (%s plan) has used %d/%d %s", userID, planName, usage, limit, quotaType),
		Source:   "quota_enforcer",
		UserID:   userID,
		Metadata: map[string]interface{}{
			"plan_name":  planName,
			"quota_type": quotaType,
			"usage":      usage,
			"limit":      limit,
			"percentage": float64(usage) / float64(limit) * 100,
		},
		Tags: []string{"quota", "usage"},
	}

	_, err := as.SendAlert(ctx, alert)
	return err
}

// SendPerformanceAlert sends a performance-specific alert
func (as *AlertingService) SendPerformanceAlert(ctx context.Context, metric string, value float64, threshold float64, unit string) error {
	severity := AlertSeverityMedium
	if value > threshold*2 {
		severity = AlertSeverityHigh
	}
	if value > threshold*5 {
		severity = AlertSeverityCritical
	}

	alert := &Alert{
		Type:     AlertTypePerformance,
		Severity: severity,
		Title:    "Performance Alert",
		Message:  fmt.Sprintf("Metric %s is %f %s (threshold: %f %s)", metric, value, unit, threshold, unit),
		Source:   "monitoring",
		Metadata: map[string]interface{}{
			"metric":    metric,
			"value":     value,
			"threshold": threshold,
			"unit":      unit,
			"ratio":     value / threshold,
		},
		Tags: []string{"performance", "monitoring"},
	}

	_, err := as.SendAlert(ctx, alert)
	return err
}

// sendTelegramAlert sends alert via Telegram
func (as *AlertingService) sendTelegramAlert(ctx context.Context, alert *Alert) (*AlertResult, error) {
	if as.config.TelegramBotToken == "" || as.config.TelegramChatID == "" {
		return nil, fmt.Errorf("telegram configuration missing")
	}

	// Format message
	message := as.formatTelegramMessage(alert)

	// Create request
	url := fmt.Sprintf("%s%s/sendMessage", as.config.TelegramAPIURL, as.config.TelegramBotToken)

	payload := map[string]interface{}{
		"chat_id":    as.config.TelegramChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := as.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return &AlertResult{
		Success:   true,
		Channel:   "telegram",
		Timestamp: time.Now(),
	}, nil
}

// sendEmailAlert sends alert via email
func (as *AlertingService) sendEmailAlert(_ context.Context, _ *Alert) (*AlertResult, error) {
	// Email implementation would go here
	// For now, return success
	return &AlertResult{
		Success:   true,
		Channel:   "email",
		Timestamp: time.Now(),
	}, nil
}

// sendWebhookAlert sends alert via webhook
func (as *AlertingService) sendWebhookAlert(ctx context.Context, alert *Alert) (*AlertResult, error) {
	if as.config.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL not configured")
	}

	jsonData, err := json.Marshal(alert)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", as.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if as.config.WebhookSecret != "" {
		req.Header.Set("X-Webhook-Secret", as.config.WebhookSecret)
	}

	resp, err := as.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return &AlertResult{
		Success:   true,
		Channel:   "webhook",
		Timestamp: time.Now(),
	}, nil
}

// formatTelegramMessage formats alert for Telegram
func (as *AlertingService) formatTelegramMessage(alert *Alert) string {
	severityEmoji := map[AlertSeverity]string{
		AlertSeverityLow:      "🟢",
		AlertSeverityMedium:   "🟡",
		AlertSeverityHigh:     "🟠",
		AlertSeverityCritical: "🔴",
	}

	emoji := severityEmoji[alert.Severity]

	message := fmt.Sprintf("%s <b>%s</b>\n", emoji, alert.Title)
	message += fmt.Sprintf("Type: %s\n", alert.Type)
	message += fmt.Sprintf("Severity: %s\n", alert.Severity)
	message += fmt.Sprintf("Source: %s\n", alert.Source)
	message += fmt.Sprintf("Time: %s\n", alert.Timestamp.Format("2006-01-02 15:04:05 UTC"))

	if alert.Message != "" {
		message += fmt.Sprintf("\n%s", alert.Message)
	}

	if alert.UserID != "" {
		message += fmt.Sprintf("\nUser: %s", alert.UserID)
	}

	if alert.IPAddress != "" {
		message += fmt.Sprintf("\nIP: %s", alert.IPAddress)
	}

	if len(alert.Tags) > 0 {
		message += fmt.Sprintf("\nTags: %s", fmt.Sprintf("%v", alert.Tags))
	}

	return message
}

// Allow checks if an alert is allowed based on rate limiting
func (arl *AlertRateLimiter) Allow(alertID string) bool {
	now := time.Now()

	// Simple rate limiting: allow one alert per ID per hour
	if lastSent, exists := arl.alerts[alertID]; exists {
		if now.Sub(lastSent) < time.Hour {
			return false
		}
	}

	arl.alerts[alertID] = now
	return true
}

// GetAlertStats returns alerting statistics
func (as *AlertingService) GetAlertStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":              as.config,
		"telegram_enabled":    as.config.TelegramEnabled,
		"email_enabled":       as.config.EmailEnabled,
		"webhook_enabled":     as.config.WebhookEnabled,
		"rate_limited_alerts": len(as.rateLimiter.alerts),
	}
}

// UpdateConfig updates the alerting configuration
func (as *AlertingService) UpdateConfig(config *AlertConfig) {
	as.config = config
}
