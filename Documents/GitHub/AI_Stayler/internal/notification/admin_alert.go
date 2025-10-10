package notification

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AdminAlertService handles admin alerts via Telegram
type AdminAlertService struct {
	telegramProvider TelegramProvider
	config           AdminAlertConfig
	alertQueue       chan AdminAlert
	stopChan         chan struct{}
	alertCounters    map[string]AlertCounter
}

// AdminAlertConfig represents admin alert configuration
type AdminAlertConfig struct {
	TelegramChatID  string
	Enabled         bool
	MaxQueueSize    int
	BatchSize       int
	BatchTimeout    time.Duration
	RetryCount      int
	RetryDelay      time.Duration
	AlertThresholds map[string]int // error_type -> max_alerts_per_hour
	CooldownPeriod  time.Duration
}

// AdminAlert represents an admin alert
type AdminAlert struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Severity     string                 `json:"severity"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Context      map[string]interface{} `json:"context"`
	Timestamp    time.Time              `json:"timestamp"`
	Service      string                 `json:"service"`
	UserID       *string                `json:"user_id,omitempty"`
	ConversionID *string                `json:"conversion_id,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	Priority     int                    `json:"priority"` // Higher number = higher priority
}

// AlertCounter tracks alert frequency
type AlertCounter struct {
	Count     int
	LastAlert time.Time
}

// NewAdminAlertService creates a new admin alert service
func NewAdminAlertService(telegramProvider TelegramProvider, config AdminAlertConfig) *AdminAlertService {
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = 1000
	}
	if config.BatchSize == 0 {
		config.BatchSize = 5
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 30 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.CooldownPeriod == 0 {
		config.CooldownPeriod = 1 * time.Hour
	}
	if config.AlertThresholds == nil {
		config.AlertThresholds = map[string]int{
			"system":          10,
			"network":         5,
			"timeout":         5,
			"rate_limit":      3,
			"quota_exceeded":  2,
			"validation":      1,
			"not_found":       1,
			"unauthorized":    3,
			"forbidden":       3,
			"gemini_api":      5,
			"payment_gateway": 3,
			"storage":         5,
			"notification":    2,
			"file_corrupted":  3,
			"file_missing":    2,
			"file_too_large":  2,
			"invalid_format":  2,
		}
	}

	service := &AdminAlertService{
		telegramProvider: telegramProvider,
		config:           config,
		alertQueue:       make(chan AdminAlert, config.MaxQueueSize),
		stopChan:         make(chan struct{}),
		alertCounters:    make(map[string]AlertCounter),
	}

	return service
}

// Start starts the admin alert service
func (s *AdminAlertService) Start(ctx context.Context) {
	if !s.config.Enabled {
		log.Println("Admin alert service is disabled")
		return
	}

	log.Println("Starting admin alert service")

	go s.processAlerts(ctx)
}

// Stop stops the admin alert service
func (s *AdminAlertService) Stop() {
	close(s.stopChan)
	log.Println("Admin alert service stopped")
}

// SendAlert sends an alert to the admin
func (s *AdminAlertService) SendAlert(ctx context.Context, alert AdminAlert) error {
	if !s.config.Enabled {
		return nil
	}

	// Set default values
	if alert.ID == "" {
		alert.ID = generateAlertID()
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}
	if alert.Service == "" {
		alert.Service = "ai-stayler"
	}

	// Check if alert should be throttled
	if s.shouldThrottleAlert(alert.Type) {
		log.Printf("Alert throttled for type: %s", alert.Type)
		return nil
	}

	// Add to queue
	select {
	case s.alertQueue <- alert:
		return nil
	default:
		return fmt.Errorf("alert queue is full")
	}
}

// processAlerts processes alerts from the queue
func (s *AdminAlertService) processAlerts(ctx context.Context) {
	batch := make([]AdminAlert, 0, s.config.BatchSize)
	ticker := time.NewTicker(s.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Process remaining alerts
			if len(batch) > 0 {
				s.sendBatch(ctx, batch)
			}
			return
		case <-s.stopChan:
			// Process remaining alerts
			if len(batch) > 0 {
				s.sendBatch(ctx, batch)
			}
			return
		case alert := <-s.alertQueue:
			batch = append(batch, alert)

			// Send batch if it's full
			if len(batch) >= s.config.BatchSize {
				s.sendBatch(ctx, batch)
				batch = make([]AdminAlert, 0, s.config.BatchSize)
			}
		case <-ticker.C:
			// Send batch on timeout
			if len(batch) > 0 {
				s.sendBatch(ctx, batch)
				batch = make([]AdminAlert, 0, s.config.BatchSize)
			}
		}
	}
}

// sendBatch sends a batch of alerts
func (s *AdminAlertService) sendBatch(ctx context.Context, alerts []AdminAlert) {
	if len(alerts) == 0 {
		return
	}

	// Sort alerts by priority (higher priority first)
	sortAlertsByPriority(alerts)

	// Create batch message
	message := s.createBatchMessage(alerts)

	// Send to Telegram
	if err := s.sendTelegramAlert(ctx, message); err != nil {
		log.Printf("Failed to send admin alert batch: %v", err)

		// Retry individual alerts
		for _, alert := range alerts {
			s.retryAlert(ctx, alert)
		}
	} else {
		// Update alert counters
		for _, alert := range alerts {
			s.updateAlertCounter(alert.Type)
		}
	}
}

// createBatchMessage creates a formatted message for a batch of alerts
func (s *AdminAlertService) createBatchMessage(alerts []AdminAlert) string {
	if len(alerts) == 1 {
		return s.formatSingleAlert(alerts[0])
	}

	message := fmt.Sprintf("🚨 *Admin Alert Batch* (%d alerts)\n\n", len(alerts))

	// Group alerts by severity
	criticalAlerts := make([]AdminAlert, 0)
	highAlerts := make([]AdminAlert, 0)
	mediumAlerts := make([]AdminAlert, 0)
	lowAlerts := make([]AdminAlert, 0)

	for _, alert := range alerts {
		switch alert.Severity {
		case "critical":
			criticalAlerts = append(criticalAlerts, alert)
		case "high":
			highAlerts = append(highAlerts, alert)
		case "medium":
			mediumAlerts = append(mediumAlerts, alert)
		case "low":
			lowAlerts = append(lowAlerts, alert)
		}
	}

	// Add alerts by severity
	if len(criticalAlerts) > 0 {
		message += "🔴 *CRITICAL*\n"
		for _, alert := range criticalAlerts {
			message += s.formatAlertSummary(alert) + "\n"
		}
		message += "\n"
	}

	if len(highAlerts) > 0 {
		message += "🟠 *HIGH*\n"
		for _, alert := range highAlerts {
			message += s.formatAlertSummary(alert) + "\n"
		}
		message += "\n"
	}

	if len(mediumAlerts) > 0 {
		message += "🟡 *MEDIUM*\n"
		for _, alert := range mediumAlerts {
			message += s.formatAlertSummary(alert) + "\n"
		}
		message += "\n"
	}

	if len(lowAlerts) > 0 {
		message += "🟢 *LOW*\n"
		for _, alert := range lowAlerts {
			message += s.formatAlertSummary(alert) + "\n"
		}
	}

	// Add timestamp
	message += fmt.Sprintf("\n_Generated at: %s_", time.Now().Format("2006-01-02 15:04:05 UTC"))

	return message
}

// formatSingleAlert formats a single alert
func (s *AdminAlertService) formatSingleAlert(alert AdminAlert) string {
	severityEmoji := s.getSeverityEmoji(alert.Severity)

	message := fmt.Sprintf("%s *%s Alert*\n\n", severityEmoji, alert.Severity)
	message += fmt.Sprintf("*Type:* %s\n", alert.Type)
	message += fmt.Sprintf("*Title:* %s\n", alert.Title)
	message += fmt.Sprintf("*Message:* %s\n", alert.Message)
	message += fmt.Sprintf("*Service:* %s\n", alert.Service)
	message += fmt.Sprintf("*Time:* %s\n", alert.Timestamp.Format("2006-01-02 15:04:05 UTC"))

	if alert.UserID != nil {
		message += fmt.Sprintf("*User ID:* %s\n", *alert.UserID)
	}
	if alert.ConversionID != nil {
		message += fmt.Sprintf("*Conversion ID:* %s\n", *alert.ConversionID)
	}

	// Add context if available
	if len(alert.Context) > 0 {
		message += "\n*Context:*\n"
		for k, v := range alert.Context {
			message += fmt.Sprintf("• %s: %v\n", k, v)
		}
	}

	return message
}

// formatAlertSummary formats an alert summary for batch messages
func (s *AdminAlertService) formatAlertSummary(alert AdminAlert) string {
	severityEmoji := s.getSeverityEmoji(alert.Severity)
	return fmt.Sprintf("%s %s: %s", severityEmoji, alert.Type, alert.Title)
}

// getSeverityEmoji returns emoji for severity level
func (s *AdminAlertService) getSeverityEmoji(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

// sendTelegramAlert sends an alert to Telegram
func (s *AdminAlertService) sendTelegramAlert(ctx context.Context, message string) error {
	return s.telegramProvider.SendMessage(ctx, s.config.TelegramChatID, message)
}

// retryAlert retries sending an alert
func (s *AdminAlertService) retryAlert(ctx context.Context, alert AdminAlert) {
	if alert.RetryCount >= s.config.RetryCount {
		log.Printf("Alert %s exceeded max retries", alert.ID)
		return
	}

	alert.RetryCount++

	// Wait before retry
	time.Sleep(s.config.RetryDelay)

	// Retry sending
	if err := s.sendTelegramAlert(ctx, s.formatSingleAlert(alert)); err != nil {
		log.Printf("Failed to retry alert %s: %v", alert.ID, err)
	} else {
		s.updateAlertCounter(alert.Type)
	}
}

// shouldThrottleAlert checks if an alert should be throttled
func (s *AdminAlertService) shouldThrottleAlert(alertType string) bool {
	threshold, exists := s.config.AlertThresholds[alertType]
	if !exists {
		return false
	}

	counter, exists := s.alertCounters[alertType]
	if !exists {
		return false
	}

	// Check if we're within the cooldown period
	if time.Since(counter.LastAlert) < s.config.CooldownPeriod {
		return counter.Count >= threshold
	}

	// Reset counter if cooldown period has passed
	s.alertCounters[alertType] = AlertCounter{
		Count:     0,
		LastAlert: time.Now(),
	}

	return false
}

// updateAlertCounter updates the alert counter
func (s *AdminAlertService) updateAlertCounter(alertType string) {
	counter, exists := s.alertCounters[alertType]
	if !exists {
		counter = AlertCounter{
			Count:     0,
			LastAlert: time.Now(),
		}
	}

	counter.Count++
	counter.LastAlert = time.Now()
	s.alertCounters[alertType] = counter
}

// sortAlertsByPriority sorts alerts by priority (higher priority first)
func sortAlertsByPriority(alerts []AdminAlert) {
	// Simple bubble sort for small batches
	for i := 0; i < len(alerts)-1; i++ {
		for j := 0; j < len(alerts)-i-1; j++ {
			if alerts[j].Priority < alerts[j+1].Priority {
				alerts[j], alerts[j+1] = alerts[j+1], alerts[j]
			}
		}
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// Alert severity levels
const (
	AlertSeverityLow      = "low"
	AlertSeverityMedium   = "medium"
	AlertSeverityHigh     = "high"
	AlertSeverityCritical = "critical"
)

// Alert types
const (
	AlertTypeSystem         = "system"
	AlertTypeNetwork        = "network"
	AlertTypeTimeout        = "timeout"
	AlertTypeRateLimit      = "rate_limit"
	AlertTypeQuotaExceeded  = "quota_exceeded"
	AlertTypeValidation     = "validation"
	AlertTypeNotFound       = "not_found"
	AlertTypeUnauthorized   = "unauthorized"
	AlertTypeForbidden      = "forbidden"
	AlertTypeGeminiAPI      = "gemini_api"
	AlertTypePaymentGateway = "payment_gateway"
	AlertTypeStorage        = "storage"
	AlertTypeNotification   = "notification"
	AlertTypeFileCorrupted  = "file_corrupted"
	AlertTypeFileMissing    = "file_missing"
	AlertTypeFileTooLarge   = "file_too_large"
	AlertTypeInvalidFormat  = "invalid_format"
)
