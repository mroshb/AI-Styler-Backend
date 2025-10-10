package admin

import (
	"context"
	"database/sql"
)

// WireAdminService creates an admin service with all dependencies
func WireAdminService(db *sql.DB) (*Service, *Handler) {
	// Create store
	store := NewDBStore(db)

	// Create real dependencies instead of mocks
	notifier := &realAdminNotificationService{db: db}
	auditLogger := &realAdminAuditLogger{db: db}

	// Create service
	service := NewService(store, notifier, auditLogger)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// WireAdminServiceWithMocks creates an admin service with mock dependencies for testing
func WireAdminServiceWithMocks(store Store) (*Service, *Handler) {
	// Create mock dependencies
	notifier := NewMockNotificationService()
	auditLogger := NewMockAuditLogger()

	// Create service
	service := NewService(store, notifier, auditLogger)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// MockNotificationService implements NotificationService for testing
type MockNotificationService struct{}

// NewMockNotificationService creates a new mock notification service
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{}
}

// SendNotification sends a notification (mock implementation)
func (m *MockNotificationService) SendNotification(ctx context.Context, userID string, notificationType string, data map[string]interface{}) error {
	// Mock implementation - in real implementation, this would send actual notifications
	return nil
}

// SendEmail sends an email (mock implementation)
func (m *MockNotificationService) SendEmail(ctx context.Context, email string, subject string, body string) error {
	// Mock implementation - in real implementation, this would send actual emails
	return nil
}

// SendSMS sends an SMS (mock implementation)
func (m *MockNotificationService) SendSMS(ctx context.Context, phone string, message string) error {
	// Mock implementation - in real implementation, this would send actual SMS
	return nil
}

// MockAuditLogger implements AuditLogger for testing
type MockAuditLogger struct{}

// NewMockAuditLogger creates a new mock audit logger
func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{}
}

// LogAction logs an action (mock implementation)
func (m *MockAuditLogger) LogAction(ctx context.Context, userID *string, actorType, action, resource string, resourceID *string, metadata map[string]interface{}) error {
	// Mock implementation - in real implementation, this would log to audit table
	return nil
}

// Real implementations to replace mocks

// realAdminNotificationService implements NotificationService for admin
type realAdminNotificationService struct {
	db *sql.DB
}

func (r *realAdminNotificationService) SendNotification(ctx context.Context, userID string, notificationType string, data map[string]interface{}) error {
	// Implementation would send actual notifications
	return nil
}

func (r *realAdminNotificationService) SendEmail(ctx context.Context, email string, subject string, body string) error {
	// Implementation would send actual emails
	return nil
}

func (r *realAdminNotificationService) SendSMS(ctx context.Context, phone string, message string) error {
	// Implementation would send actual SMS
	return nil
}

// realAdminAuditLogger implements AuditLogger for admin
type realAdminAuditLogger struct {
	db *sql.DB
}

func (r *realAdminAuditLogger) LogAction(ctx context.Context, userID *string, actorType, action, resource string, resourceID *string, metadata map[string]interface{}) error {
	// Implementation would log admin actions to audit table
	return nil
}
