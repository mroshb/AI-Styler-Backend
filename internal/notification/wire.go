package notification

import (
	"database/sql"
)

// WireNotificationService creates a notification service with all dependencies
func WireNotificationService(db *sql.DB) (*Service, *Handler) {
	// Create store
	store := NewStore(db)

	// Create config first
	emailConfig := EmailConfig{
		Enabled:      true,
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "noreply@aistyler.com",
		SMTPPassword: "your-email-password",
		FromEmail:    "noreply@aistyler.com",
		FromName:     "AI Styler",
		Username:     "noreply@aistyler.com",
		Password:     "your-email-password",
	}

	smsConfig := SMSConfig{
		Enabled:    true,
		Provider:   "sms_ir",
		APIKey:     "your-sms-api-key",
		TemplateID: 1,
		RetryCount: 3,
	}

	telegramConfig := TelegramConfig{
		Enabled:      true,
		BotToken:     "your-telegram-bot-token",
		ChatID:       "your-telegram-chat-id",
		RetryCount:   3,
		RetryDelayMs: 1000,
	}

	websocketConfig := WebSocketConfig{
		Enabled:        true,
		Port:           8081,
		MaxConnections: 1000,
		PingInterval:   30,
	}

	retryConfig := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1000,  // 1 second
		MaxDelay:   30000, // 30 seconds
	}

	// Create providers with config
	emailProvider := NewEmailProvider(emailConfig)
	smsProvider := NewSMSProvider(smsConfig)
	telegramProvider := NewTelegramProvider(telegramConfig)
	websocketProvider := NewWebSocketProvider(websocketConfig)

	// Create template engine
	templateEngine := NewTemplateEngine()

	// Create services
	quotaService := NewQuotaService()
	userService := NewUserService()
	conversionService := NewConversionService()
	paymentService := NewPaymentService()

	// Create audit logger
	auditLogger := NewAuditLogger()

	// Create metrics collector
	metrics := NewMetricsCollector()

	// Create retry handler
	retryHandler := NewRetryHandler()

	// Create config
	config := NotificationConfig{
		Email:     emailConfig,
		SMS:       smsConfig,
		Telegram:  telegramConfig,
		WebSocket: websocketConfig,
		Retry:     retryConfig,
	}

	// Create service
	service := NewService(
		store,
		emailProvider,
		smsProvider,
		telegramProvider,
		websocketProvider,
		templateEngine,
		quotaService,
		userService,
		conversionService,
		paymentService,
		auditLogger,
		metrics,
		retryHandler,
		config,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// WireNotificationServiceWithMocks creates a notification service with mock dependencies for testing
func WireNotificationServiceWithMocks(store Store) (*Service, *Handler) {
	// Create mock dependencies
	emailProvider := NewMockEmailProvider()
	smsProvider := NewMockSMSProvider()
	telegramProvider := NewMockTelegramProvider()
	websocketProvider := NewMockWebSocketProvider()
	templateEngine := NewMockTemplateEngine()
	quotaService := NewMockQuotaService()
	userService := NewMockUserService()
	conversionService := NewMockConversionService()
	paymentService := NewMockPaymentService()
	auditLogger := NewMockAuditLogger()
	metrics := NewMockMetricsCollector()
	retryHandler := NewMockRetryHandler()

	// Create config
	config := NotificationConfig{
		Email: EmailConfig{
			Enabled: true,
		},
		SMS: SMSConfig{
			Enabled: true,
		},
		Telegram: TelegramConfig{
			Enabled: true,
		},
		WebSocket: WebSocketConfig{
			Enabled: true,
		},
		Retry: RetryConfig{
			MaxRetries: 3,
			BaseDelay:  1000,
			MaxDelay:   30000,
		},
	}

	// Create service
	service := NewService(
		store,
		emailProvider,
		smsProvider,
		telegramProvider,
		websocketProvider,
		templateEngine,
		quotaService,
		userService,
		conversionService,
		paymentService,
		auditLogger,
		metrics,
		retryHandler,
		config,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}
