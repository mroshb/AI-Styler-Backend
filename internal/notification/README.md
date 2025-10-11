# Notification Service

A comprehensive notification system for the AI Stayler application that supports multiple delivery channels and real-time updates.

## Features

- **Multiple Delivery Channels**: Email, SMS, Telegram, WebSocket, and Push notifications
- **Real-time Updates**: WebSocket support for instant notifications
- **User Preferences**: Customizable notification preferences per user
- **Template System**: Flexible template system for different notification types
- **Quota Monitoring**: Automatic quota monitoring and notifications
- **Retry Logic**: Built-in retry mechanism for failed deliveries
- **Statistics**: Comprehensive notification statistics and metrics
- **Quiet Hours**: Support for user-defined quiet hours

## Architecture

### Core Components

1. **NotificationService**: Main service interface for notification operations
2. **NotificationStore**: Data access layer for notifications
3. **Providers**: Channel-specific providers (Email, SMS, Telegram, WebSocket)
4. **TemplateEngine**: Template processing for different channels
5. **QuotaMonitor**: Monitors user quotas and sends notifications
6. **IntegrationService**: Provides easy integration for other services

### Notification Types

- **Conversion Notifications**: Started, completed, failed
- **Quota Notifications**: Exhausted, warning, reset
- **Payment Notifications**: Success, failed, plan activated/expired
- **System Notifications**: Maintenance, errors, critical alerts
- **User Notifications**: Welcome, profile updates, password changes

### Delivery Channels

- **Email**: SMTP-based email delivery with HTML templates
- **SMS**: SMS delivery via SMS providers (SMS.ir, etc.)
- **Telegram**: Telegram bot notifications for admin alerts
- **WebSocket**: Real-time notifications for web clients
- **Push**: Mobile push notifications (future implementation)

## Usage

### Basic Notification Creation

```go
// Create a notification
req := CreateNotificationRequest{
    UserID: &userID,
    Type:   NotificationTypeConversionCompleted,
    Title:  "Conversion Completed",
    Message: "Your image conversion has completed successfully!",
    Data: map[string]interface{}{
        "conversionId":  "conv_123",
        "resultImageId": "img_456",
    },
    Priority: PriorityNormal,
}

notification, err := notificationService.CreateNotification(ctx, req)
```

### Integration with Other Services

```go
// In your service, inject the notification integration service
type MyService struct {
    notificationIntegration *notification.IntegrationService
}

// Send notifications
func (s *MyService) ProcessConversion(ctx context.Context, conversionID string) {
    // ... processing logic ...
    
    // Send notification
    if err := s.notificationIntegration.SendConversionCompleted(ctx, userID, conversionID, resultImageID); err != nil {
        log.Printf("Failed to send notification: %v", err)
    }
}
```

### WebSocket Real-time Updates

```javascript
// Client-side WebSocket connection
const ws = new WebSocket('ws://localhost:8080/api/notifications/ws');

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received notification:', message);
    
    // Handle different notification types
    switch(message.type) {
        case 'conversion_completed':
            showSuccessNotification(message.data);
            break;
        case 'quota_exhausted':
            showQuotaWarning(message.data);
            break;
        // ... other cases
    }
};
```

## Configuration

### Email Configuration

```go
emailConfig := EmailConfig{
    SMTPHost:     "smtp.gmail.com",
    SMTPPort:     587,
    SMTPUsername: "your-email@gmail.com",
    SMTPPassword: "your-password",
    FromEmail:    "noreply@aistayler.com",
    FromName:     "AI Stayler",
    Enabled:      true,
}
```

### Telegram Configuration

```go
telegramConfig := TelegramConfig{
    BotToken:     "your-bot-token",
    ChatID:       "admin-chat-id",
    Enabled:      true,
    RetryCount:   3,
    RetryDelayMs: 1000,
}
```

### WebSocket Configuration

```go
websocketConfig := WebSocketConfig{
    Enabled:        true,
    Port:           8080,
    MaxConnections: 1000,
    PingInterval:   30, // seconds
}
```

## Database Schema

The notification system uses several database tables:

- `notifications`: Main notifications table
- `notification_deliveries`: Delivery tracking for each channel
- `notification_preferences`: User notification preferences
- `notification_templates`: Templates for different notification types

See `db/migrations/0008_notification_service.sql` for the complete schema.

## API Endpoints

### Notifications

- `POST /api/notifications` - Create notification
- `GET /api/notifications` - List notifications
- `GET /api/notifications/:id` - Get notification
- `PUT /api/notifications/:id/read` - Mark as read
- `DELETE /api/notifications/:id` - Delete notification
- `POST /api/notifications/test` - Send test notification

### Preferences

- `GET /api/notifications/preferences` - Get user preferences
- `PUT /api/notifications/preferences` - Update preferences

### Statistics

- `GET /api/notifications/stats` - Get notification statistics

### WebSocket

- `GET /api/notifications/ws` - WebSocket connection

## Templates

The system supports templates for different notification types and channels. Templates use Go's template syntax with variables.

### Example Email Template

```html
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Conversion ID: {{.notification.data.conversionId}}</p>
<p>Status: {{.notification.data.status}}</p>
```

### Example SMS Template

```
{{.notification.title}}: {{.notification.message}} (ID: {{.notification.data.conversionId}})
```

### Example Telegram Template

```
*{{.notification.title}}* ✅

{{.notification.message}}

*Conversion ID:* {{.notification.data.conversionId}}
*Status:* {{.notification.data.status}}
```

## Monitoring and Statistics

The system provides comprehensive statistics:

- Total notifications sent/delivered/failed/read
- Statistics by channel, type, and priority
- Average delivery times
- Error rates and retry counts

## Error Handling

- Failed deliveries are automatically retried based on retry policies
- Critical errors are sent to Telegram for admin alerts
- All errors are logged for debugging
- Graceful degradation when providers are unavailable

## Security Considerations

- WebSocket connections require authentication
- User preferences are isolated per user
- Sensitive data is not logged
- Rate limiting prevents abuse

## Future Enhancements

- Mobile push notifications
- Advanced template editor
- A/B testing for notifications
- Advanced analytics and reporting
- Notification scheduling
- Bulk notification sending
- Notification digests
