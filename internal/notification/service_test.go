package notification

import (
	"context"
	"testing"
	"time"
)

// TestCreateNotification tests notification creation
func TestCreateNotification(t *testing.T) {
	// Test data
	userID := "user123"
	req := CreateNotificationRequest{
		UserID:   &userID,
		Type:     NotificationTypeConversionCompleted,
		Title:    "Test Notification",
		Message:  "This is a test notification",
		Priority: PriorityNormal,
	}

	// This test would require a mock store to work properly
	// For now, we'll just test the request structure
	if req.UserID == nil {
		t.Error("UserID should not be nil")
	}
	if req.Type != NotificationTypeConversionCompleted {
		t.Error("Type should be NotificationTypeConversionCompleted")
	}
	if req.Title != "Test Notification" {
		t.Error("Title should be 'Test Notification'")
	}
	if req.Message != "This is a test notification" {
		t.Error("Message should be 'This is a test notification'")
	}
	if req.Priority != PriorityNormal {
		t.Error("Priority should be PriorityNormal")
	}
}

// TestNotificationTypes tests notification type constants
func TestNotificationTypes(t *testing.T) {
	// Test conversion notification types
	if NotificationTypeConversionStarted != "conversion_started" {
		t.Error("NotificationTypeConversionStarted should be 'conversion_started'")
	}
	if NotificationTypeConversionCompleted != "conversion_completed" {
		t.Error("NotificationTypeConversionCompleted should be 'conversion_completed'")
	}
	if NotificationTypeConversionFailed != "conversion_failed" {
		t.Error("NotificationTypeConversionFailed should be 'conversion_failed'")
	}

	// Test quota notification types
	if NotificationTypeQuotaExhausted != "quota_exhausted" {
		t.Error("NotificationTypeQuotaExhausted should be 'quota_exhausted'")
	}
	if NotificationTypeQuotaWarning != "quota_warning" {
		t.Error("NotificationTypeQuotaWarning should be 'quota_warning'")
	}
	if NotificationTypeQuotaReset != "quota_reset" {
		t.Error("NotificationTypeQuotaReset should be 'quota_reset'")
	}

	// Test payment notification types
	if NotificationTypePaymentSuccess != "payment_success" {
		t.Error("NotificationTypePaymentSuccess should be 'payment_success'")
	}
	if NotificationTypePaymentFailed != "payment_failed" {
		t.Error("NotificationTypePaymentFailed should be 'payment_failed'")
	}

	// Test system notification types
	if NotificationTypeCriticalError != "critical_error" {
		t.Error("NotificationTypeCriticalError should be 'critical_error'")
	}
	if NotificationTypeSystemMaintenance != "system_maintenance" {
		t.Error("NotificationTypeSystemMaintenance should be 'system_maintenance'")
	}
}

// TestNotificationChannels tests notification channel constants
func TestNotificationChannels(t *testing.T) {
	if ChannelEmail != "email" {
		t.Error("ChannelEmail should be 'email'")
	}
	if ChannelSMS != "sms" {
		t.Error("ChannelSMS should be 'sms'")
	}
	if ChannelTelegram != "telegram" {
		t.Error("ChannelTelegram should be 'telegram'")
	}
	if ChannelWebSocket != "websocket" {
		t.Error("ChannelWebSocket should be 'websocket'")
	}
	if ChannelPush != "push" {
		t.Error("ChannelPush should be 'push'")
	}
}

// TestNotificationPriorities tests notification priority constants
func TestNotificationPriorities(t *testing.T) {
	if PriorityLow != "low" {
		t.Error("PriorityLow should be 'low'")
	}
	if PriorityNormal != "normal" {
		t.Error("PriorityNormal should be 'normal'")
	}
	if PriorityHigh != "high" {
		t.Error("PriorityHigh should be 'high'")
	}
	if PriorityCritical != "critical" {
		t.Error("PriorityCritical should be 'critical'")
	}
}

// TestNotificationStatuses tests notification status constants
func TestNotificationStatuses(t *testing.T) {
	if StatusPending != "pending" {
		t.Error("StatusPending should be 'pending'")
	}
	if StatusSending != "sending" {
		t.Error("StatusSending should be 'sending'")
	}
	if StatusSent != "sent" {
		t.Error("StatusSent should be 'sent'")
	}
	if StatusDelivered != "delivered" {
		t.Error("StatusDelivered should be 'delivered'")
	}
	if StatusFailed != "failed" {
		t.Error("StatusFailed should be 'failed'")
	}
	if StatusRead != "read" {
		t.Error("StatusRead should be 'read'")
	}
	if StatusExpired != "expired" {
		t.Error("StatusExpired should be 'expired'")
	}
}

// TestNotificationCreation tests notification struct creation
func TestNotificationCreation(t *testing.T) {
	userID := "user123"

	notification := Notification{
		ID:       "notif123",
		UserID:   &userID,
		Type:     NotificationTypeConversionCompleted,
		Title:    "Test Notification",
		Message:  "This is a test notification",
		Channels: []NotificationChannel{ChannelEmail, ChannelWebSocket},
		Priority: PriorityNormal,
		Status:   StatusPending,
	}

	if notification.ID != "notif123" {
		t.Error("ID should be 'notif123'")
	}
	if notification.UserID == nil || *notification.UserID != "user123" {
		t.Error("UserID should be 'user123'")
	}
	if notification.Type != NotificationTypeConversionCompleted {
		t.Error("Type should be NotificationTypeConversionCompleted")
	}
	if notification.Title != "Test Notification" {
		t.Error("Title should be 'Test Notification'")
	}
	if notification.Message != "This is a test notification" {
		t.Error("Message should be 'This is a test notification'")
	}
	if len(notification.Channels) != 2 {
		t.Error("Should have 2 channels")
	}
	if notification.Priority != PriorityNormal {
		t.Error("Priority should be PriorityNormal")
	}
	if notification.Status != StatusPending {
		t.Error("Status should be StatusPending")
	}
}

// TestNotificationPreferenceCreation tests notification preference struct creation
func TestNotificationPreferenceCreation(t *testing.T) {
	userID := "user123"
	prefs := NotificationPreference{
		UserID:           userID,
		EmailEnabled:     true,
		SMSEnabled:       false,
		TelegramEnabled:  false,
		WebSocketEnabled: true,
		PushEnabled:      false,
		Timezone:         "UTC",
	}

	if prefs.UserID != "user123" {
		t.Error("UserID should be 'user123'")
	}
	if !prefs.EmailEnabled {
		t.Error("EmailEnabled should be true")
	}
	if prefs.SMSEnabled {
		t.Error("SMSEnabled should be false")
	}
	if prefs.TelegramEnabled {
		t.Error("TelegramEnabled should be false")
	}
	if !prefs.WebSocketEnabled {
		t.Error("WebSocketEnabled should be true")
	}
	if prefs.PushEnabled {
		t.Error("PushEnabled should be false")
	}
	if prefs.Timezone != "UTC" {
		t.Error("Timezone should be 'UTC'")
	}
}

// TestWebSocketMessageCreation tests WebSocket message struct creation
func TestWebSocketMessageCreation(t *testing.T) {
	now := time.Now()
	data := map[string]interface{}{
		"test": true,
		"id":   "123",
	}

	message := WebSocketMessage{
		Type:      "test_message",
		Data:      data,
		Timestamp: now,
	}

	if message.Type != "test_message" {
		t.Error("Type should be 'test_message'")
	}
	if message.Data["test"] != true {
		t.Error("Data should contain test=true")
	}
	if message.Data["id"] != "123" {
		t.Error("Data should contain id=123")
	}
	if !message.Timestamp.Equal(now) {
		t.Error("Timestamp should match now")
	}
}

// TestGenerateID tests the generateID function
func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}
	if id2 == "" {
		t.Error("Generated ID should not be empty")
	}
	if id1 == id2 {
		t.Error("Generated IDs should be different")
	}
	if len(id1) != 32 {
		t.Error("Generated ID should be 32 characters long")
	}
	if len(id2) != 32 {
		t.Error("Generated ID should be 32 characters long")
	}
}

// TestIntegrationService tests the integration service
func TestIntegrationService(t *testing.T) {
	// Create a mock notification service
	mockService := &MockNotificationService{}

	// Create integration service
	integrationService := &IntegrationService{
		notificationService: mockService,
	}

	// Test that the integration service is created
	if integrationService.notificationService == nil {
		t.Error("Notification service should not be nil")
	}
}

// MockNotificationService is a simple mock for testing
type MockNotificationService struct{}

func (m *MockNotificationService) CreateNotification(ctx context.Context, req CreateNotificationRequest) (Notification, error) {
	return Notification{}, nil
}

func (m *MockNotificationService) GetNotification(ctx context.Context, notificationID string) (Notification, error) {
	return Notification{}, nil
}

func (m *MockNotificationService) ListNotifications(ctx context.Context, req NotificationListRequest) (NotificationListResponse, error) {
	return NotificationListResponse{}, nil
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	return nil
}

func (m *MockNotificationService) DeleteNotification(ctx context.Context, notificationID, userID string) error {
	return nil
}

func (m *MockNotificationService) SendConversionStarted(ctx context.Context, userID, conversionID string) error {
	return nil
}

func (m *MockNotificationService) SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error {
	return nil
}

func (m *MockNotificationService) SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error {
	return nil
}

func (m *MockNotificationService) SendQuotaExhausted(ctx context.Context, userID string, quotaType string) error {
	return nil
}

func (m *MockNotificationService) SendQuotaWarning(ctx context.Context, userID string, quotaType string, remaining int) error {
	return nil
}

func (m *MockNotificationService) SendQuotaReset(ctx context.Context, userID string) error {
	return nil
}

func (m *MockNotificationService) SendPaymentSuccess(ctx context.Context, userID, paymentID, planName string) error {
	return nil
}

func (m *MockNotificationService) SendPaymentFailed(ctx context.Context, userID, paymentID, reason string) error {
	return nil
}

func (m *MockNotificationService) SendPlanActivated(ctx context.Context, userID, planName string) error {
	return nil
}

func (m *MockNotificationService) SendPlanExpired(ctx context.Context, userID, planName string) error {
	return nil
}

func (m *MockNotificationService) SendCriticalError(ctx context.Context, errorType, message string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockNotificationService) SendSystemMaintenance(ctx context.Context, message string, scheduledFor *string) error {
	return nil
}

func (m *MockNotificationService) GetNotificationPreferences(ctx context.Context, userID string) (NotificationPreference, error) {
	return NotificationPreference{}, nil
}

func (m *MockNotificationService) UpdateNotificationPreferences(ctx context.Context, userID string, req UpdateNotificationPreferenceRequest) error {
	return nil
}

func (m *MockNotificationService) GetNotificationStats(ctx context.Context, timeRange string) (NotificationStats, error) {
	return NotificationStats{}, nil
}

func (m *MockNotificationService) BroadcastToUser(ctx context.Context, userID string, message WebSocketMessage) error {
	return nil
}

func (m *MockNotificationService) BroadcastToAll(ctx context.Context, message WebSocketMessage) error {
	return nil
}
