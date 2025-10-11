package notification

import (
	"context"
	"log"
	"time"
)

// IntegrationService provides notification integration for other services
type IntegrationService struct {
	notificationService NotificationService
}

// NewIntegrationService creates a new notification integration service
func NewIntegrationService(notificationService NotificationService) *IntegrationService {
	return &IntegrationService{
		notificationService: notificationService,
	}
}

// SendConversionStarted sends a conversion started notification
func (i *IntegrationService) SendConversionStarted(ctx context.Context, userID, conversionID string) error {
	if err := i.notificationService.SendConversionStarted(ctx, userID, conversionID); err != nil {
		log.Printf("Failed to send conversion started notification: %v", err)
		return err
	}
	return nil
}

// SendConversionCompleted sends a conversion completed notification
func (i *IntegrationService) SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error {
	if err := i.notificationService.SendConversionCompleted(ctx, userID, conversionID, resultImageID); err != nil {
		log.Printf("Failed to send conversion completed notification: %v", err)
		return err
	}
	return nil
}

// SendConversionFailed sends a conversion failed notification
func (i *IntegrationService) SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error {
	if err := i.notificationService.SendConversionFailed(ctx, userID, conversionID, errorMessage); err != nil {
		log.Printf("Failed to send conversion failed notification: %v", err)
		return err
	}
	return nil
}

// SendQuotaExhausted sends a quota exhausted notification
func (i *IntegrationService) SendQuotaExhausted(ctx context.Context, userID, quotaType string) error {
	if err := i.notificationService.SendQuotaExhausted(ctx, userID, quotaType); err != nil {
		log.Printf("Failed to send quota exhausted notification: %v", err)
		return err
	}
	return nil
}

// SendQuotaWarning sends a quota warning notification
func (i *IntegrationService) SendQuotaWarning(ctx context.Context, userID, quotaType string, remaining int) error {
	if err := i.notificationService.SendQuotaWarning(ctx, userID, quotaType, remaining); err != nil {
		log.Printf("Failed to send quota warning notification: %v", err)
		return err
	}
	return nil
}

// SendQuotaReset sends a quota reset notification
func (i *IntegrationService) SendQuotaReset(ctx context.Context, userID string) error {
	if err := i.notificationService.SendQuotaReset(ctx, userID); err != nil {
		log.Printf("Failed to send quota reset notification: %v", err)
		return err
	}
	return nil
}

// SendPaymentSuccess sends a payment success notification
func (i *IntegrationService) SendPaymentSuccess(ctx context.Context, userID, paymentID, planName string) error {
	if err := i.notificationService.SendPaymentSuccess(ctx, userID, paymentID, planName); err != nil {
		log.Printf("Failed to send payment success notification: %v", err)
		return err
	}
	return nil
}

// SendPaymentFailed sends a payment failed notification
func (i *IntegrationService) SendPaymentFailed(ctx context.Context, userID, paymentID, reason string) error {
	if err := i.notificationService.SendPaymentFailed(ctx, userID, paymentID, reason); err != nil {
		log.Printf("Failed to send payment failed notification: %v", err)
		return err
	}
	return nil
}

// SendPlanActivated sends a plan activated notification
func (i *IntegrationService) SendPlanActivated(ctx context.Context, userID, planName string) error {
	if err := i.notificationService.SendPlanActivated(ctx, userID, planName); err != nil {
		log.Printf("Failed to send plan activated notification: %v", err)
		return err
	}
	return nil
}

// SendPlanExpired sends a plan expired notification
func (i *IntegrationService) SendPlanExpired(ctx context.Context, userID, planName string) error {
	if err := i.notificationService.SendPlanExpired(ctx, userID, planName); err != nil {
		log.Printf("Failed to send plan expired notification: %v", err)
		return err
	}
	return nil
}

// SendCriticalError sends a critical error alert to Telegram
func (i *IntegrationService) SendCriticalError(ctx context.Context, errorType, message string, metadata map[string]interface{}) error {
	if err := i.notificationService.SendCriticalError(ctx, errorType, message, metadata); err != nil {
		log.Printf("Failed to send critical error notification: %v", err)
		return err
	}
	return nil
}

// SendSystemMaintenance sends a system maintenance notification
func (i *IntegrationService) SendSystemMaintenance(ctx context.Context, message string, scheduledFor *string) error {
	if err := i.notificationService.SendSystemMaintenance(ctx, message, scheduledFor); err != nil {
		log.Printf("Failed to send system maintenance notification: %v", err)
		return err
	}
	return nil
}

// BroadcastToUser broadcasts a message to a specific user via WebSocket
func (i *IntegrationService) BroadcastToUser(ctx context.Context, userID string, messageType string, data map[string]interface{}) error {
	message := WebSocketMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := i.notificationService.BroadcastToUser(ctx, userID, message); err != nil {
		log.Printf("Failed to broadcast message to user: %v", err)
		return err
	}
	return nil
}

// BroadcastToAll broadcasts a message to all connected users via WebSocket
func (i *IntegrationService) BroadcastToAll(ctx context.Context, messageType string, data map[string]interface{}) error {
	message := WebSocketMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := i.notificationService.BroadcastToAll(ctx, message); err != nil {
		log.Printf("Failed to broadcast message to all users: %v", err)
		return err
	}
	return nil
}
