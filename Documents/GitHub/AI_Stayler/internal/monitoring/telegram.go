package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"AI_Styler/internal/common"
)

// TelegramConfig represents Telegram configuration
type TelegramConfig struct {
	BotToken string
	ChatID   string
	Enabled  bool
	Timeout  time.Duration
}

// TelegramAlert represents a Telegram alert
type TelegramAlert struct {
	Type         common.ErrorType
	Severity     common.SeverityLevel
	Title        string
	Message      string
	Context      map[string]interface{}
	Timestamp    time.Time
	Service      string
	UserID       *string
	VendorID     *string
	ConversionID *string
	TraceID      *string
	Environment  string
}

// TelegramMonitor provides Telegram integration for alerts
type TelegramMonitor struct {
	config TelegramConfig
	client *http.Client
}

// NewTelegramMonitor creates a new Telegram monitor
func NewTelegramMonitor(config TelegramConfig) *TelegramMonitor {
	if !config.Enabled || config.BotToken == "" || config.ChatID == "" {
		return &TelegramMonitor{} // Return empty monitor if not configured
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &TelegramMonitor{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SendAlert sends an alert to Telegram (implements common.AlertService interface)
func (t *TelegramMonitor) SendAlert(ctx context.Context, alert common.Alert) error {
	if t.client == nil {
		return nil // Skip if not configured
	}

	// Convert common.Alert to TelegramAlert
	telegramAlert := TelegramAlert{
		Type:         alert.Type,
		Severity:     alert.Severity,
		Title:        alert.Title,
		Message:      alert.Message,
		Context:      alert.Context,
		Timestamp:    alert.Timestamp,
		Service:      alert.Service,
		UserID:       alert.UserID,
		ConversionID: alert.ConversionID,
		Environment:  t.getEnvironment(),
	}

	message := t.formatAlertMessage(telegramAlert)
	return t.sendMessage(ctx, message)
}

// SendTelegramAlert sends a Telegram-specific alert
func (t *TelegramMonitor) SendTelegramAlert(ctx context.Context, alert TelegramAlert) error {
	if t.client == nil {
		return nil // Skip if not configured
	}

	message := t.formatAlertMessage(alert)
	return t.sendMessage(ctx, message)
}

// SendCriticalAlert sends a critical alert with high priority
func (t *TelegramMonitor) SendCriticalAlert(ctx context.Context, title string, message string, context map[string]interface{}) error {
	alert := TelegramAlert{
		Type:        common.ErrorTypeSystem,
		Severity:    common.SeverityCritical,
		Title:       title,
		Message:     message,
		Context:     context,
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// SendErrorAlert sends an error alert
func (t *TelegramMonitor) SendErrorAlert(ctx context.Context, err error, context map[string]interface{}) error {
	alert := TelegramAlert{
		Type:        common.ErrorTypeSystem,
		Severity:    common.SeverityHigh,
		Title:       "System Error",
		Message:     err.Error(),
		Context:     context,
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// SendBusinessAlert sends a business logic alert
func (t *TelegramMonitor) SendBusinessAlert(ctx context.Context, alertType common.ErrorType, message string, context map[string]interface{}) error {
	alert := TelegramAlert{
		Type:        alertType,
		Severity:    common.SeverityMedium,
		Title:       fmt.Sprintf("Business Logic Alert: %s", string(alertType)),
		Message:     message,
		Context:     context,
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// SendPerformanceAlert sends a performance alert
func (t *TelegramMonitor) SendPerformanceAlert(ctx context.Context, metric string, value interface{}, threshold interface{}, context map[string]interface{}) error {
	alert := TelegramAlert{
		Type:        common.ErrorTypeSystem,
		Severity:    common.SeverityMedium,
		Title:       "Performance Alert",
		Message:     fmt.Sprintf("Metric '%s' exceeded threshold: %v > %v", metric, value, threshold),
		Context:     context,
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// SendQuotaAlert sends a quota-related alert
func (t *TelegramMonitor) SendQuotaAlert(ctx context.Context, resource string, usage int, limit int, userID string) error {
	alert := TelegramAlert{
		Type:     common.ErrorTypeQuotaExceeded,
		Severity: common.SeverityMedium,
		Title:    "Quota Alert",
		Message:  fmt.Sprintf("Quota exceeded for %s: %d/%d", resource, usage, limit),
		Context: map[string]interface{}{
			"resource": resource,
			"usage":    usage,
			"limit":    limit,
		},
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		UserID:      &userID,
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// SendSecurityAlert sends a security-related alert
func (t *TelegramMonitor) SendSecurityAlert(ctx context.Context, event string, details string, context map[string]interface{}) error {
	alert := TelegramAlert{
		Type:        common.ErrorTypeUnauthorized,
		Severity:    common.SeverityHigh,
		Title:       "Security Alert",
		Message:     fmt.Sprintf("%s: %s", event, details),
		Context:     context,
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// SendSystemHealthAlert sends a system health alert
func (t *TelegramMonitor) SendSystemHealthAlert(ctx context.Context, component string, status string, details string) error {
	alert := TelegramAlert{
		Type:     common.ErrorTypeSystem,
		Severity: common.SeverityHigh,
		Title:    "System Health Alert",
		Message:  fmt.Sprintf("Component '%s' is %s: %s", component, status, details),
		Context: map[string]interface{}{
			"component": component,
			"status":    status,
			"details":   details,
		},
		Timestamp:   time.Now(),
		Service:     "ai-stayler",
		Environment: t.getEnvironment(),
	}

	return t.SendTelegramAlert(ctx, alert)
}

// formatAlertMessage formats an alert for Telegram
func (t *TelegramMonitor) formatAlertMessage(alert TelegramAlert) string {
	var message bytes.Buffer

	// Add emoji based on severity
	emoji := t.getSeverityEmoji(alert.Severity)
	message.WriteString(fmt.Sprintf("%s *%s*\n\n", emoji, alert.Title))

	// Add message
	message.WriteString(fmt.Sprintf("📝 *Message:* %s\n", alert.Message))

	// Add service and environment
	message.WriteString(fmt.Sprintf("🏢 *Service:* %s\n", alert.Service))
	message.WriteString(fmt.Sprintf("🌍 *Environment:* %s\n", alert.Environment))

	// Add timestamp
	message.WriteString(fmt.Sprintf("⏰ *Time:* %s\n", alert.Timestamp.Format("2006-01-02 15:04:05 UTC")))

	// Add severity and type
	message.WriteString(fmt.Sprintf("⚠️ *Severity:* %s\n", alert.Severity))
	message.WriteString(fmt.Sprintf("🏷️ *Type:* %s\n", alert.Type))

	// Add user information if available
	if alert.UserID != nil {
		message.WriteString(fmt.Sprintf("👤 *User ID:* %s\n", *alert.UserID))
	}

	// Add vendor information if available
	if alert.VendorID != nil {
		message.WriteString(fmt.Sprintf("🏪 *Vendor ID:* %s\n", *alert.VendorID))
	}

	// Add conversion information if available
	if alert.ConversionID != nil {
		message.WriteString(fmt.Sprintf("🔄 *Conversion ID:* %s\n", *alert.ConversionID))
	}

	// Add trace information if available
	if alert.TraceID != nil {
		message.WriteString(fmt.Sprintf("🔍 *Trace ID:* %s\n", *alert.TraceID))
	}

	// Add context information
	if len(alert.Context) > 0 {
		message.WriteString("\n📋 *Context:*\n")
		for key, value := range alert.Context {
			message.WriteString(fmt.Sprintf("• %s: %v\n", key, value))
		}
	}

	return message.String()
}

// getSeverityEmoji returns an emoji based on severity level
func (t *TelegramMonitor) getSeverityEmoji(severity common.SeverityLevel) string {
	switch severity {
	case common.SeverityLow:
		return "ℹ️"
	case common.SeverityMedium:
		return "⚠️"
	case common.SeverityHigh:
		return "🚨"
	case common.SeverityCritical:
		return "🔥"
	default:
		return "⚠️"
	}
}

// sendMessage sends a message to Telegram
func (t *TelegramMonitor) sendMessage(ctx context.Context, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.config.BotToken)

	payload := map[string]interface{}{
		"chat_id":                  t.config.ChatID,
		"text":                     message,
		"parse_mode":               "Markdown",
		"disable_web_page_preview": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// getEnvironment returns the current environment
func (t *TelegramMonitor) getEnvironment() string {
	// This could be read from config or environment variable
	return "production" // Default to production
}

// IsEnabled returns whether Telegram monitoring is enabled
func (t *TelegramMonitor) IsEnabled() bool {
	return t.client != nil
}

// GetDefaultTelegramConfig returns default Telegram configuration
func GetDefaultTelegramConfig() TelegramConfig {
	return TelegramConfig{
		Enabled: false,
		Timeout: 10 * time.Second,
	}
}

// SendTestMessage sends a test message to verify configuration
func (t *TelegramMonitor) SendTestMessage(ctx context.Context) error {
	if t.client == nil {
		return fmt.Errorf("telegram monitor not configured")
	}

	message := "🧪 *Test Message*\n\nThis is a test message from AI Styler monitoring system.\n\n✅ Configuration is working correctly!"
	return t.sendMessage(ctx, message)
}

// SendDailyReport sends a daily system report
func (t *TelegramMonitor) SendDailyReport(ctx context.Context, stats map[string]interface{}) error {
	if t.client == nil {
		return nil
	}

	var message bytes.Buffer
	message.WriteString("📊 *Daily System Report*\n\n")

	// Add date
	message.WriteString(fmt.Sprintf("📅 *Date:* %s\n\n", time.Now().Format("2006-01-02")))

	// Add statistics
	message.WriteString("📈 *Statistics:*\n")
	for key, value := range stats {
		message.WriteString(fmt.Sprintf("• %s: %v\n", key, value))
	}

	return t.sendMessage(ctx, message.String())
}
