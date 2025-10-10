package notification

import (
	"context"
)

// Mock implementations for testing

// MockEmailProvider implements EmailProvider interface
type MockEmailProvider struct{}

func NewMockEmailProvider() EmailProvider {
	return &MockEmailProvider{}
}

func (m *MockEmailProvider) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	return nil
}

func (m *MockEmailProvider) SendTemplateEmail(ctx context.Context, to, templateID string, data map[string]interface{}) error {
	return nil
}

func (m *MockEmailProvider) ValidateEmail(email string) bool {
	return true
}

// MockSMSProvider implements SMSProvider interface
type MockSMSProvider struct{}

func NewMockSMSProvider() SMSProvider {
	return &MockSMSProvider{}
}

func (m *MockSMSProvider) SendSMS(ctx context.Context, phone, message string) error {
	return nil
}

func (m *MockSMSProvider) SendTemplateSMS(ctx context.Context, phone, templateID string, data map[string]interface{}) error {
	return nil
}

func (m *MockSMSProvider) ValidatePhone(phone string) bool {
	return true
}

// MockTelegramProvider implements TelegramProvider interface
type MockTelegramProvider struct{}

func NewMockTelegramProvider() TelegramProvider {
	return &MockTelegramProvider{}
}

func (m *MockTelegramProvider) SendMessage(ctx context.Context, chatID, message string) error {
	return nil
}

func (m *MockTelegramProvider) SendTemplateMessage(ctx context.Context, chatID, templateID string, data map[string]interface{}) error {
	return nil
}

func (m *MockTelegramProvider) SendPhoto(ctx context.Context, chatID, photoURL, caption string) error {
	return nil
}

func (m *MockTelegramProvider) SendDocument(ctx context.Context, chatID, documentURL, caption string) error {
	return nil
}

func (m *MockTelegramProvider) SetWebhook(ctx context.Context, webhookURL string) error {
	return nil
}

func (m *MockTelegramProvider) GetUpdates(ctx context.Context) ([]TelegramUpdate, error) {
	return []TelegramUpdate{}, nil
}

// MockWebSocketProvider implements WebSocketProvider interface
type MockWebSocketProvider struct{}

func NewMockWebSocketProvider() WebSocketProvider {
	return &MockWebSocketProvider{}
}

func (m *MockWebSocketProvider) BroadcastToUser(ctx context.Context, userID string, message WebSocketMessage) error {
	return nil
}

func (m *MockWebSocketProvider) BroadcastToAll(ctx context.Context, message WebSocketMessage) error {
	return nil
}

func (m *MockWebSocketProvider) GetConnectedUsers(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockWebSocketProvider) IsUserConnected(ctx context.Context, userID string) bool {
	return true
}

func (m *MockWebSocketProvider) CloseUserConnection(ctx context.Context, userID string) error {
	return nil
}

// MockTemplateEngine implements TemplateEngine interface
type MockTemplateEngine struct{}

func NewMockTemplateEngine() TemplateEngine {
	return &MockTemplateEngine{}
}

func (m *MockTemplateEngine) ProcessTemplate(template string, data map[string]interface{}) (string, error) {
	return "Mock template result", nil
}

func (m *MockTemplateEngine) ProcessEmailTemplate(templateID string, data map[string]interface{}) (subject, body string, err error) {
	return "Mock Subject", "Mock Body", nil
}

func (m *MockTemplateEngine) ProcessSMSTemplate(templateID string, data map[string]interface{}) (string, error) {
	return "Mock SMS", nil
}

func (m *MockTemplateEngine) ProcessTelegramTemplate(templateID string, data map[string]interface{}) (string, error) {
	return "Mock Telegram", nil
}

// MockQuotaService implements QuotaService interface
type MockQuotaService struct{}

func NewMockQuotaService() QuotaService {
	return &MockQuotaService{}
}

func (m *MockQuotaService) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{"remaining": 100}, nil
}

func (m *MockQuotaService) CheckUserQuota(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{"allowed": true}, nil
}

// MockUserService implements UserService interface
type MockUserService struct{}

func NewMockUserService() UserService {
	return &MockUserService{}
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{"id": userID, "email": "test@example.com"}, nil
}

func (m *MockUserService) GetUserByPhone(ctx context.Context, phone string) (interface{}, error) {
	return map[string]interface{}{"phone": phone}, nil
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (interface{}, error) {
	return map[string]interface{}{"email": email}, nil
}

// MockConversionService implements ConversionService interface
type MockConversionService struct{}

func NewMockConversionService() ConversionService {
	return &MockConversionService{}
}

func (m *MockConversionService) GetConversion(ctx context.Context, conversionID string) (interface{}, error) {
	return map[string]interface{}{"id": conversionID}, nil
}

func (m *MockConversionService) GetConversionWithDetails(ctx context.Context, conversionID string) (interface{}, error) {
	return map[string]interface{}{"id": conversionID, "status": "completed"}, nil
}

// MockPaymentService implements PaymentService interface
type MockPaymentService struct{}

func NewMockPaymentService() PaymentService {
	return &MockPaymentService{}
}

func (m *MockPaymentService) GetPayment(ctx context.Context, paymentID string) (interface{}, error) {
	return map[string]interface{}{"id": paymentID}, nil
}

func (m *MockPaymentService) GetUserPayments(ctx context.Context, userID string) (interface{}, error) {
	return []interface{}{}, nil
}

// MockAuditLogger implements AuditLogger interface
type MockAuditLogger struct{}

func NewMockAuditLogger() AuditLogger {
	return &MockAuditLogger{}
}

func (m *MockAuditLogger) LogNotificationSent(ctx context.Context, userID *string, notificationType NotificationType, channel NotificationChannel, success bool, errorMessage *string) error {
	return nil
}

func (m *MockAuditLogger) LogNotificationRead(ctx context.Context, userID, notificationID string) error {
	return nil
}

func (m *MockAuditLogger) LogNotificationDeleted(ctx context.Context, userID, notificationID string) error {
	return nil
}

// MockMetricsCollector implements MetricsCollector interface
type MockMetricsCollector struct{}

func NewMockMetricsCollector() MetricsCollector {
	return &MockMetricsCollector{}
}

func (m *MockMetricsCollector) RecordNotificationSent(ctx context.Context, notificationType NotificationType, channel NotificationChannel) error {
	return nil
}

func (m *MockMetricsCollector) RecordNotificationDelivered(ctx context.Context, notificationType NotificationType, channel NotificationChannel, deliveryTimeMs int64) error {
	return nil
}

func (m *MockMetricsCollector) RecordNotificationFailed(ctx context.Context, notificationType NotificationType, channel NotificationChannel, errorType string) error {
	return nil
}

func (m *MockMetricsCollector) RecordNotificationRead(ctx context.Context, notificationType NotificationType, channel NotificationChannel) error {
	return nil
}

func (m *MockMetricsCollector) GetNotificationMetrics(ctx context.Context, timeRange string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// MockRetryHandler implements RetryHandler interface
type MockRetryHandler struct{}

func NewMockRetryHandler() RetryHandler {
	return &MockRetryHandler{}
}

func (m *MockRetryHandler) ShouldRetry(ctx context.Context, delivery NotificationDelivery, err error) bool {
	return delivery.RetryCount < 3
}

func (m *MockRetryHandler) GetRetryDelay(ctx context.Context, delivery NotificationDelivery) int64 {
	return 1000 // 1 second
}

func (m *MockRetryHandler) IncrementRetryCount(ctx context.Context, deliveryID string) error {
	return nil
}
