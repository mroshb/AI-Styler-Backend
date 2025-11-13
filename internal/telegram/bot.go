package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents the Telegram bot service
type Bot struct {
	api          *tgbotapi.BotAPI
	config       *Config
	handlers     *Handlers
	ctx          context.Context
	cancel       context.CancelFunc
	webhookURL   string
	webhookPort  int
}

// NewBot creates a new bot instance
func NewBot(config *Config, handlers *Handlers) (*Bot, error) {
	log.Printf("Creating bot with token: %s...", maskToken(config.Telegram.BotToken))
	
	bot, err := tgbotapi.NewBotAPI(config.Telegram.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = config.Telegram.Env == "development"

	// Verify bot connection
	botInfo, err := bot.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to verify bot token: %w", err)
	}

	log.Printf("Bot authenticated successfully! Username: @%s (ID: %d)", botInfo.UserName, botInfo.ID)

	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		api:         bot,
		config:      config,
		handlers:    handlers,
		ctx:         ctx,
		cancel:      cancel,
		webhookURL:  config.Server.WebhookURL,
		webhookPort: config.Server.WebhookPort,
	}, nil
}

// maskToken masks the token for logging (shows only first 10 and last 4 characters)
func maskToken(token string) string {
	if len(token) <= 14 {
		return "***"
	}
	return token[:10] + "..." + token[len(token)-4:]
}

// Start starts the bot in polling or webhook mode
func (b *Bot) Start() error {
	if b.config.Telegram.Env == "production" && b.webhookURL != "" {
		return b.startWebhook()
	}
	return b.startPolling()
}

// startPolling starts the bot in long polling mode (for development)
func (b *Bot) startPolling() error {
	log.Printf("Starting bot in polling mode...")
	
	// Clear any pending updates
	_, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		log.Printf("Warning: Failed to clear pending updates: %v", err)
	}
	
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Printf("Getting updates channel...")
	updates := b.api.GetUpdatesChan(u)
	log.Printf("✅ Bot is now listening for updates! Send /start to test.")

	for {
		select {
		case <-b.ctx.Done():
			log.Printf("Polling context cancelled, stopping...")
			return nil
		case update := <-updates:
			log.Printf("Received update: Message=%v, CallbackQuery=%v", update.Message != nil, update.CallbackQuery != nil)
			go b.handleUpdate(update)
		}
	}
}

// startWebhook starts the bot in webhook mode (for production)
func (b *Bot) startWebhook() error {
	log.Printf("Starting bot in webhook mode...")

	// Set webhook
	wh, err := tgbotapi.NewWebhook(b.webhookURL + "/webhook")
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	_, err = b.api.Request(wh)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	// Start webhook server
	http.HandleFunc("/webhook", b.webhookHandler)
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", b.webhookPort),
		Handler: nil,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Webhook server error: %v", err)
		}
	}()

	// Wait for shutdown
	<-b.ctx.Done()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return server.Shutdown(ctx)
}

// webhookHandler handles webhook requests
func (b *Bot) webhookHandler(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("Failed to decode update: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	go b.handleUpdate(update)
	w.WriteHeader(http.StatusOK)
}

// handleUpdate processes a Telegram update
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		RecordProcessingDuration("update", duration)
	}()

	RecordUpdate("update")

	if update.Message != nil {
		b.handlers.HandleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handlers.HandleCallbackQuery(update.CallbackQuery)
	}
}

// Stop stops the bot
func (b *Bot) Stop() {
	log.Printf("Stopping bot...")
	b.cancel()
	
	// Remove webhook if in production
	if b.config.Telegram.Env == "production" {
		deleteWebhook := tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true}
		_, _ = b.api.Request(deleteWebhook)
	}
}

// GetBot returns the underlying bot API instance
func (b *Bot) GetBot() *tgbotapi.BotAPI {
	return b.api
}

