package notification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// Service provides notification functionality
type Service struct {
	store             NotificationStore
	emailProvider     EmailProvider
	smsProvider       SMSProvider
	telegramProvider  TelegramProvider
	websocketProvider WebSocketProvider
	templateEngine    TemplateEngine
	quotaService      QuotaService
	userService       UserService
	conversionService ConversionService
	paymentService    PaymentService
	auditLogger       AuditLogger
	metrics           MetricsCollector
	retryHandler      RetryHandler
	config            NotificationConfig
}

// NewService creates a new notification service
func NewService(
	store NotificationStore,
	emailProvider EmailProvider,
	smsProvider SMSProvider,
	telegramProvider TelegramProvider,
	websocketProvider WebSocketProvider,
	templateEngine TemplateEngine,
	quotaService QuotaService,
	userService UserService,
	conversionService ConversionService,
	paymentService PaymentService,
	auditLogger AuditLogger,
	metrics MetricsCollector,
	retryHandler RetryHandler,
	config NotificationConfig,
) *Service {
	return &Service{
		store:             store,
		emailProvider:     emailProvider,
		smsProvider:       smsProvider,
		telegramProvider:  telegramProvider,
		websocketProvider: websocketProvider,
		templateEngine:    templateEngine,
		quotaService:      quotaService,
		userService:       userService,
		conversionService: conversionService,
		paymentService:    paymentService,
		auditLogger:       auditLogger,
		metrics:           metrics,
		retryHandler:      retryHandler,
		config:            config,
	}
}

// CreateNotification creates a new notification
func (s *Service) CreateNotification(ctx context.Context, req CreateNotificationRequest) (Notification, error) {
	// Generate notification ID
	notificationID := generateID()

	// Set default values
	if req.Priority == "" {
		req.Priority = PriorityNormal
	}

	// If no channels specified, use user preferences
	if len(req.Channels) == 0 && req.UserID != nil {
		prefs, err := s.GetNotificationPreferences(ctx, *req.UserID)
		if err == nil {
			req.Channels = s.getEnabledChannels(prefs)
		} else {
			// Default channels if preferences not found
			req.Channels = []NotificationChannel{ChannelEmail, ChannelWebSocket}
		}
	}

	// Create notification
	notification := Notification{
		ID:           notificationID,
		UserID:       req.UserID,
		Type:         req.Type,
		Title:        req.Title,
		Message:      req.Message,
		Data:         req.Data,
		Channels:     req.Channels,
		Priority:     req.Priority,
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		ScheduledFor: req.ScheduledFor,
		ExpiresAt:    req.ExpiresAt,
	}

	// Save notification
	if err := s.store.CreateNotification(ctx, notification); err != nil {
		return Notification{}, fmt.Errorf("failed to create notification: %w", err)
	}

	// Process notification immediately if not scheduled
	if req.ScheduledFor == nil || req.ScheduledFor.Before(time.Now()) {
		go s.processNotification(context.Background(), notification)
	}

	return notification, nil
}

// GetNotification retrieves a notification by ID
func (s *Service) GetNotification(ctx context.Context, notificationID string) (Notification, error) {
	return s.store.GetNotification(ctx, notificationID)
}

// ListNotifications lists notifications based on criteria
func (s *Service) ListNotifications(ctx context.Context, req NotificationListRequest) (NotificationListResponse, error) {
	return s.store.ListNotifications(ctx, req)
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	if err := s.store.MarkAsRead(ctx, notificationID, userID); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	// Log audit
	if err := s.auditLogger.LogNotificationRead(ctx, userID, notificationID); err != nil {
		log.Printf("Failed to log notification read: %v", err)
	}

	// Record metrics
	if err := s.metrics.RecordNotificationRead(ctx, NotificationType(""), ChannelWebSocket); err != nil {
		log.Printf("Failed to record notification read metrics: %v", err)
	}

	return nil
}

// DeleteNotification deletes a notification
func (s *Service) DeleteNotification(ctx context.Context, notificationID, userID string) error {
	if err := s.store.DeleteNotification(ctx, notificationID); err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	// Log audit
	if err := s.auditLogger.LogNotificationDeleted(ctx, userID, notificationID); err != nil {
		log.Printf("Failed to log notification deletion: %v", err)
	}

	return nil
}

// SendConversionStarted sends a conversion started notification
func (s *Service) SendConversionStarted(ctx context.Context, userID, conversionID string) error {
	// Get conversion details
	_, err := s.conversionService.GetConversionWithDetails(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion details: %w", err)
	}

	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypeConversionStarted,
		Title:   "Conversion Started",
		Message: fmt.Sprintf("Your image conversion has started. Conversion ID: %s", conversionID),
		Data: map[string]interface{}{
			"conversionId": conversionID,
			"status":       "started",
		},
		Priority: PriorityNormal,
	}

	_, err = s.CreateNotification(ctx, req)
	return err
}

// SendConversionCompleted sends a conversion completed notification
func (s *Service) SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error {
	// Get conversion details
	_, err := s.conversionService.GetConversionWithDetails(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion details: %w", err)
	}

	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypeConversionCompleted,
		Title:   "Conversion Completed",
		Message: fmt.Sprintf("Your image conversion has completed successfully! Result image ID: %s", resultImageID),
		Data: map[string]interface{}{
			"conversionId":  conversionID,
			"resultImageId": resultImageID,
			"status":        "completed",
		},
		Priority: PriorityNormal,
	}

	_, err = s.CreateNotification(ctx, req)
	return err
}

// SendConversionFailed sends a conversion failed notification
func (s *Service) SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error {
	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypeConversionFailed,
		Title:   "Conversion Failed",
		Message: fmt.Sprintf("Your image conversion failed: %s", errorMessage),
		Data: map[string]interface{}{
			"conversionId": conversionID,
			"errorMessage": errorMessage,
			"status":       "failed",
		},
		Priority: PriorityHigh,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendQuotaExhausted sends a quota exhausted notification
func (s *Service) SendQuotaExhausted(ctx context.Context, userID string, quotaType string) error {
	// Get user details
	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user details: %w", err)
	}

	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypeQuotaExhausted,
		Title:   "Quota Exhausted",
		Message: fmt.Sprintf("Your %s quota has been exhausted. Please upgrade your plan to continue.", quotaType),
		Data: map[string]interface{}{
			"quotaType": quotaType,
			"user":      user,
		},
		Priority: PriorityHigh,
	}

	_, err = s.CreateNotification(ctx, req)
	return err
}

// SendQuotaWarning sends a quota warning notification
func (s *Service) SendQuotaWarning(ctx context.Context, userID string, quotaType string, remaining int) error {
	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypeQuotaWarning,
		Title:   "Quota Warning",
		Message: fmt.Sprintf("You have %d %s conversions remaining this month.", remaining, quotaType),
		Data: map[string]interface{}{
			"quotaType": quotaType,
			"remaining": remaining,
		},
		Priority: PriorityNormal,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendQuotaReset sends a quota reset notification
func (s *Service) SendQuotaReset(ctx context.Context, userID string) error {
	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypeQuotaReset,
		Title:   "Quota Reset",
		Message: "Your monthly quota has been reset. You can now use your conversions again.",
		Data: map[string]interface{}{
			"resetDate": time.Now().Format("2006-01-02"),
		},
		Priority: PriorityNormal,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendPaymentSuccess sends a payment success notification
func (s *Service) SendPaymentSuccess(ctx context.Context, userID, paymentID, planName string) error {
	// Get payment details
	payment, err := s.paymentService.GetPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment details: %w", err)
	}

	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypePaymentSuccess,
		Title:   "Payment Successful",
		Message: fmt.Sprintf("Your payment for %s plan has been processed successfully.", planName),
		Data: map[string]interface{}{
			"paymentId": paymentID,
			"planName":  planName,
			"payment":   payment,
		},
		Priority: PriorityNormal,
	}

	_, err = s.CreateNotification(ctx, req)
	return err
}

// SendPaymentFailed sends a payment failed notification
func (s *Service) SendPaymentFailed(ctx context.Context, userID, paymentID, reason string) error {
	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypePaymentFailed,
		Title:   "Payment Failed",
		Message: fmt.Sprintf("Your payment failed: %s", reason),
		Data: map[string]interface{}{
			"paymentId": paymentID,
			"reason":    reason,
		},
		Priority: PriorityHigh,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendPlanActivated sends a plan activated notification
func (s *Service) SendPlanActivated(ctx context.Context, userID, planName string) error {
	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypePlanActivated,
		Title:   "Plan Activated",
		Message: fmt.Sprintf("Your %s plan has been activated successfully!", planName),
		Data: map[string]interface{}{
			"planName": planName,
		},
		Priority: PriorityNormal,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendPlanExpired sends a plan expired notification
func (s *Service) SendPlanExpired(ctx context.Context, userID, planName string) error {
	// Create notification
	req := CreateNotificationRequest{
		UserID:  &userID,
		Type:    NotificationTypePlanExpired,
		Title:   "Plan Expired",
		Message: fmt.Sprintf("Your %s plan has expired. Please renew to continue using premium features.", planName),
		Data: map[string]interface{}{
			"planName": planName,
		},
		Priority: PriorityHigh,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendCriticalError sends a critical error alert to Telegram
func (s *Service) SendCriticalError(ctx context.Context, errorType, message string, metadata map[string]interface{}) error {
	// Create notification for admin
	req := CreateNotificationRequest{
		Type:    NotificationTypeCriticalError,
		Title:   fmt.Sprintf("Critical Error: %s", errorType),
		Message: message,
		Data: map[string]interface{}{
			"errorType": errorType,
			"metadata":  metadata,
			"timestamp": time.Now().Format(time.RFC3339),
		},
		Channels: []NotificationChannel{ChannelTelegram},
		Priority: PriorityCritical,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// SendSystemMaintenance sends a system maintenance notification
func (s *Service) SendSystemMaintenance(ctx context.Context, message string, scheduledFor *string) error {
	// Create system-wide notification
	req := CreateNotificationRequest{
		Type:    NotificationTypeSystemMaintenance,
		Title:   "System Maintenance",
		Message: message,
		Data: map[string]interface{}{
			"scheduledFor": scheduledFor,
		},
		Channels: []NotificationChannel{ChannelEmail, ChannelWebSocket},
		Priority: PriorityHigh,
	}

	_, err := s.CreateNotification(ctx, req)
	return err
}

// GetNotificationPreferences gets user notification preferences
func (s *Service) GetNotificationPreferences(ctx context.Context, userID string) (NotificationPreference, error) {
	return s.store.GetNotificationPreferences(ctx, userID)
}

// UpdateNotificationPreferences updates user notification preferences
func (s *Service) UpdateNotificationPreferences(ctx context.Context, userID string, req UpdateNotificationPreferenceRequest) error {
	// Get current preferences
	prefs, err := s.GetNotificationPreferences(ctx, userID)
	if err != nil {
		// Create new preferences if not found
		prefs = NotificationPreference{
			UserID:           userID,
			EmailEnabled:     true,
			SMSEnabled:       false,
			TelegramEnabled:  false,
			WebSocketEnabled: true,
			PushEnabled:      false,
			Preferences:      make(map[NotificationType]bool),
			Timezone:         "UTC",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
	}

	// Update preferences
	if req.EmailEnabled != nil {
		prefs.EmailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		prefs.SMSEnabled = *req.SMSEnabled
	}
	if req.TelegramEnabled != nil {
		prefs.TelegramEnabled = *req.TelegramEnabled
	}
	if req.WebSocketEnabled != nil {
		prefs.WebSocketEnabled = *req.WebSocketEnabled
	}
	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.Preferences != nil {
		for notificationType, enabled := range req.Preferences {
			prefs.Preferences[notificationType] = enabled
		}
	}
	if req.QuietHoursStart != nil {
		prefs.QuietHoursStart = req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		prefs.QuietHoursEnd = req.QuietHoursEnd
	}
	if req.Timezone != nil {
		prefs.Timezone = *req.Timezone
	}

	prefs.UpdatedAt = time.Now()

	// Save preferences
	if err := s.store.UpdateNotificationPreferences(ctx, userID, prefs); err != nil {
		return fmt.Errorf("failed to update notification preferences: %w", err)
	}

	return nil
}

// GetNotificationStats gets notification statistics
func (s *Service) GetNotificationStats(ctx context.Context, timeRange string) (NotificationStats, error) {
	return s.store.GetNotificationStats(ctx, timeRange)
}

// BroadcastToUser broadcasts a message to a specific user via WebSocket
func (s *Service) BroadcastToUser(ctx context.Context, userID string, message WebSocketMessage) error {
	return s.websocketProvider.BroadcastToUser(ctx, userID, message)
}

// BroadcastToAll broadcasts a message to all connected users via WebSocket
func (s *Service) BroadcastToAll(ctx context.Context, message WebSocketMessage) error {
	return s.websocketProvider.BroadcastToAll(ctx, message)
}

// processNotification processes a notification for delivery
func (s *Service) processNotification(ctx context.Context, notification Notification) {
	// Update status to sending
	if err := s.store.UpdateNotification(ctx, notification.ID, map[string]interface{}{
		"status": StatusSending,
	}); err != nil {
		log.Printf("Failed to update notification status: %v", err)
		return
	}

	// Get user preferences if user-specific notification
	var prefs NotificationPreference
	if notification.UserID != nil {
		var err error
		prefs, err = s.GetNotificationPreferences(ctx, *notification.UserID)
		if err != nil {
			log.Printf("Failed to get user preferences: %v", err)
			// Use default preferences
			prefs = s.getDefaultPreferences()
		}
	}

	// Process each channel
	for _, channel := range notification.Channels {
		go s.processChannel(ctx, notification, channel, prefs)
	}
}

// processChannel processes a notification for a specific channel
func (s *Service) processChannel(ctx context.Context, notification Notification, channel NotificationChannel, prefs NotificationPreference) {
	// Check if channel is enabled for user
	if notification.UserID != nil && !s.isChannelEnabled(channel, prefs) {
		return
	}

	// Check if notification type is enabled for user
	if notification.UserID != nil && !s.isNotificationTypeEnabled(notification.Type, prefs) {
		return
	}

	// Check quiet hours
	if notification.UserID != nil && s.isInQuietHours(prefs) {
		// Schedule for later
		s.scheduleForLater(ctx, notification, channel)
		return
	}

	// Create delivery record
	delivery := NotificationDelivery{
		ID:             generateID(),
		NotificationID: notification.ID,
		Channel:        channel,
		Recipient:      s.getRecipient(notification, channel),
		Status:         StatusPending,
		RetryCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.store.CreateDelivery(ctx, delivery); err != nil {
		log.Printf("Failed to create delivery record: %v", err)
		return
	}

	// Send notification
	if err := s.sendNotification(ctx, notification, channel, delivery); err != nil {
		log.Printf("Failed to send notification via %s: %v", channel, err)
		s.handleDeliveryFailure(ctx, delivery, err)
		return
	}

	// Update delivery status
	s.updateDeliveryStatus(ctx, delivery.ID, StatusSent, nil)
}

// sendNotification sends a notification via the specified channel
func (s *Service) sendNotification(ctx context.Context, notification Notification, channel NotificationChannel, delivery NotificationDelivery) error {
	switch channel {
	case ChannelEmail:
		return s.sendEmail(ctx, notification, delivery)
	case ChannelSMS:
		return s.sendSMS(ctx, notification, delivery)
	case ChannelTelegram:
		return s.sendTelegram(ctx, notification, delivery)
	case ChannelWebSocket:
		return s.sendWebSocket(ctx, notification, delivery)
	default:
		return fmt.Errorf("unsupported channel: %s", channel)
	}
}

// sendEmail sends an email notification
func (s *Service) sendEmail(ctx context.Context, notification Notification, delivery NotificationDelivery) error {
	if !s.config.Email.Enabled {
		return fmt.Errorf("email notifications are disabled")
	}

	// Get user email
	user, err := s.userService.GetUser(ctx, *notification.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}

	// Process template
	subject, body, err := s.templateEngine.ProcessEmailTemplate(string(notification.Type), map[string]interface{}{
		"notification": notification,
		"user":         user,
	})
	if err != nil {
		// Fallback to simple template
		subject = notification.Title
		body = notification.Message
	}

	// Send email
	if err := s.emailProvider.SendEmail(ctx, delivery.Recipient, subject, body, true); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Record metrics
	if err := s.metrics.RecordNotificationSent(ctx, notification.Type, ChannelEmail); err != nil {
		log.Printf("Failed to record email metrics: %v", err)
	}

	return nil
}

// sendSMS sends an SMS notification
func (s *Service) sendSMS(ctx context.Context, notification Notification, delivery NotificationDelivery) error {
	if !s.config.SMS.Enabled {
		return fmt.Errorf("SMS notifications are disabled")
	}

	// Process template
	message, err := s.templateEngine.ProcessSMSTemplate(string(notification.Type), map[string]interface{}{
		"notification": notification,
	})
	if err != nil {
		// Fallback to simple message
		message = notification.Message
	}

	// Send SMS
	if err := s.smsProvider.SendSMS(ctx, delivery.Recipient, message); err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	// Record metrics
	if err := s.metrics.RecordNotificationSent(ctx, notification.Type, ChannelSMS); err != nil {
		log.Printf("Failed to record SMS metrics: %v", err)
	}

	return nil
}

// sendTelegram sends a Telegram notification
func (s *Service) sendTelegram(ctx context.Context, notification Notification, delivery NotificationDelivery) error {
	if !s.config.Telegram.Enabled {
		return fmt.Errorf("Telegram notifications are disabled")
	}

	// Process template
	message, err := s.templateEngine.ProcessTelegramTemplate(string(notification.Type), map[string]interface{}{
		"notification": notification,
	})
	if err != nil {
		// Fallback to simple message
		message = fmt.Sprintf("*%s*\n\n%s", notification.Title, notification.Message)
	}

	// Send Telegram message
	if err := s.telegramProvider.SendMessage(ctx, delivery.Recipient, message); err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

	// Record metrics
	if err := s.metrics.RecordNotificationSent(ctx, notification.Type, ChannelTelegram); err != nil {
		log.Printf("Failed to record Telegram metrics: %v", err)
	}

	return nil
}

// sendWebSocket sends a WebSocket notification
func (s *Service) sendWebSocket(ctx context.Context, notification Notification, _ NotificationDelivery) error {
	if !s.config.WebSocket.Enabled {
		return fmt.Errorf("WebSocket notifications are disabled")
	}

	// Create WebSocket message
	message := WebSocketMessage{
		Type: string(notification.Type),
		Data: map[string]interface{}{
			"id":      notification.ID,
			"title":   notification.Title,
			"message": notification.Message,
			"data":    notification.Data,
		},
		Timestamp: time.Now(),
	}

	// Send to user if user-specific, otherwise broadcast
	if notification.UserID != nil {
		if err := s.websocketProvider.BroadcastToUser(ctx, *notification.UserID, message); err != nil {
			return fmt.Errorf("failed to send WebSocket message to user: %w", err)
		}
	} else {
		if err := s.websocketProvider.BroadcastToAll(ctx, message); err != nil {
			return fmt.Errorf("failed to broadcast WebSocket message: %w", err)
		}
	}

	// Record metrics
	if err := s.metrics.RecordNotificationSent(ctx, notification.Type, ChannelWebSocket); err != nil {
		log.Printf("Failed to record WebSocket metrics: %v", err)
	}

	return nil
}

// Helper methods

func (s *Service) getEnabledChannels(prefs NotificationPreference) []NotificationChannel {
	var channels []NotificationChannel
	if prefs.EmailEnabled {
		channels = append(channels, ChannelEmail)
	}
	if prefs.SMSEnabled {
		channels = append(channels, ChannelSMS)
	}
	if prefs.TelegramEnabled {
		channels = append(channels, ChannelTelegram)
	}
	if prefs.WebSocketEnabled {
		channels = append(channels, ChannelWebSocket)
	}
	if prefs.PushEnabled {
		channels = append(channels, ChannelPush)
	}
	return channels
}

func (s *Service) getDefaultPreferences() NotificationPreference {
	return NotificationPreference{
		EmailEnabled:     true,
		SMSEnabled:       false,
		TelegramEnabled:  false,
		WebSocketEnabled: true,
		PushEnabled:      false,
		Preferences:      make(map[NotificationType]bool),
		Timezone:         "UTC",
	}
}

func (s *Service) isChannelEnabled(channel NotificationChannel, prefs NotificationPreference) bool {
	switch channel {
	case ChannelEmail:
		return prefs.EmailEnabled
	case ChannelSMS:
		return prefs.SMSEnabled
	case ChannelTelegram:
		return prefs.TelegramEnabled
	case ChannelWebSocket:
		return prefs.WebSocketEnabled
	case ChannelPush:
		return prefs.PushEnabled
	default:
		return false
	}
}

func (s *Service) isNotificationTypeEnabled(notificationType NotificationType, prefs NotificationPreference) bool {
	if enabled, exists := prefs.Preferences[notificationType]; exists {
		return enabled
	}
	// Default to enabled if not specified
	return true
}

func (s *Service) isInQuietHours(prefs NotificationPreference) bool {
	if prefs.QuietHoursStart == nil || prefs.QuietHoursEnd == nil {
		return false
	}

	now := time.Now()
	// Simple implementation - in production, consider timezone
	start, err := time.Parse("15:04", *prefs.QuietHoursStart)
	if err != nil {
		return false
	}
	end, err := time.Parse("15:04", *prefs.QuietHoursEnd)
	if err != nil {
		return false
	}

	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	startTime := time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, time.UTC)
	endTime := time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)

	return currentTime.After(startTime) && currentTime.Before(endTime)
}

func (s *Service) getRecipient(_ Notification, channel NotificationChannel) string {
	// This is a simplified implementation
	// In production, you'd get the actual recipient from user data
	switch channel {
	case ChannelEmail:
		return "user@example.com" // Get from user data
	case ChannelSMS:
		return "+1234567890" // Get from user data
	case ChannelTelegram:
		return s.config.Telegram.ChatID // For admin notifications
	default:
		return ""
	}
}

func (s *Service) scheduleForLater(ctx context.Context, notification Notification, channel NotificationChannel) {
	// Implement scheduling logic
	// This could involve a job queue or database scheduling
}

func (s *Service) handleDeliveryFailure(ctx context.Context, delivery NotificationDelivery, err error) {
	// Check if should retry
	if s.retryHandler.ShouldRetry(ctx, delivery, err) {
		// Schedule retry
		retryDelay := s.retryHandler.GetRetryDelay(ctx, delivery)
		nextRetryAt := time.Now().Add(time.Duration(retryDelay) * time.Millisecond)

		updates := map[string]interface{}{
			"status":       StatusFailed,
			"errorMessage": err.Error(),
			"nextRetryAt":  nextRetryAt,
		}

		if updateErr := s.store.UpdateDelivery(ctx, delivery.ID, updates); updateErr != nil {
			log.Printf("Failed to update delivery for retry: %v", updateErr)
		}
	} else {
		// Mark as permanently failed
		errorMsg := err.Error()
		s.updateDeliveryStatus(ctx, delivery.ID, StatusFailed, &errorMsg)
	}
}

func (s *Service) updateDeliveryStatus(ctx context.Context, deliveryID string, status NotificationStatus, errorMessage *string) {
	updates := map[string]interface{}{
		"status": status,
	}

	if errorMessage != nil {
		updates["errorMessage"] = *errorMessage
	}

	if status == StatusSent || status == StatusDelivered {
		now := time.Now()
		updates["sentAt"] = now
		if status == StatusDelivered {
			updates["deliveredAt"] = now
		}
	}

	updates["updatedAt"] = time.Now()

	if err := s.store.UpdateDelivery(ctx, deliveryID, updates); err != nil {
		log.Printf("Failed to update delivery status: %v", err)
	}
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SendAdminAlert sends an alert to admin via Telegram
func (s *Service) SendAdminAlert(ctx context.Context, alert AdminAlert) error {
	if !s.config.Telegram.Enabled {
		return fmt.Errorf("Telegram notifications are disabled")
	}

	// Format the alert message
	message := s.formatAdminAlert(alert)

	// Send to Telegram
	if err := s.telegramProvider.SendMessage(ctx, s.config.Telegram.ChatID, message); err != nil {
		return fmt.Errorf("failed to send admin alert: %w", err)
	}

	// Record metrics
	if err := s.metrics.RecordNotificationSent(ctx, NotificationTypeSystemError, ChannelTelegram); err != nil {
		log.Printf("Failed to record admin alert metrics: %v", err)
	}

	return nil
}

// formatAdminAlert formats an admin alert for Telegram
func (s *Service) formatAdminAlert(alert AdminAlert) string {
	severityEmoji := s.getSeverityEmoji(alert.Severity)

	message := fmt.Sprintf("%s *Admin Alert*\n\n", severityEmoji)
	message += fmt.Sprintf("*Type:* %s\n", alert.Type)
	message += fmt.Sprintf("*Severity:* %s\n", alert.Severity)
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

// getSeverityEmoji returns emoji for severity level
func (s *Service) getSeverityEmoji(severity string) string {
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
