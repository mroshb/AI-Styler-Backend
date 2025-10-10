package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramProviderImpl implements TelegramProvider interface
type TelegramProviderImpl struct {
	config     TelegramConfig
	httpClient *http.Client
	baseURL    string
}

// NewTelegramProvider creates a new Telegram provider
func NewTelegramProvider(config TelegramConfig) TelegramProvider {
	return &TelegramProviderImpl{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org/bot" + config.BotToken,
	}
}

// SendMessage sends a text message to Telegram
func (t *TelegramProviderImpl) SendMessage(ctx context.Context, chatID, message string) error {
	if !t.config.Enabled {
		return fmt.Errorf("Telegram notifications are disabled")
	}

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	return t.sendRequest(ctx, "sendMessage", payload)
}

// SendTemplateMessage sends a template message to Telegram
func (t *TelegramProviderImpl) SendTemplateMessage(ctx context.Context, chatID, templateID string, data map[string]interface{}) error {
	// This would integrate with a template engine
	// For now, we'll use a simple approach
	message := "You have a new notification"

	return t.SendMessage(ctx, chatID, message)
}

// SendPhoto sends a photo to Telegram
func (t *TelegramProviderImpl) SendPhoto(ctx context.Context, chatID, photoURL, caption string) error {
	if !t.config.Enabled {
		return fmt.Errorf("Telegram notifications are disabled")
	}

	payload := map[string]string{
		"chat_id":    chatID,
		"photo":      photoURL,
		"caption":    caption,
		"parse_mode": "Markdown",
	}

	return t.sendRequest(ctx, "sendPhoto", payload)
}

// SendDocument sends a document to Telegram
func (t *TelegramProviderImpl) SendDocument(ctx context.Context, chatID, documentURL, caption string) error {
	if !t.config.Enabled {
		return fmt.Errorf("Telegram notifications are disabled")
	}

	payload := map[string]string{
		"chat_id":    chatID,
		"document":   documentURL,
		"caption":    caption,
		"parse_mode": "Markdown",
	}

	return t.sendRequest(ctx, "sendDocument", payload)
}

// SetWebhook sets the webhook for Telegram bot
func (t *TelegramProviderImpl) SetWebhook(ctx context.Context, webhookURL string) error {
	if !t.config.Enabled {
		return fmt.Errorf("Telegram notifications are disabled")
	}

	payload := map[string]string{
		"url": webhookURL,
	}

	return t.sendRequest(ctx, "setWebhook", payload)
}

// GetUpdates gets updates from Telegram
func (t *TelegramProviderImpl) GetUpdates(ctx context.Context) ([]TelegramUpdate, error) {
	if !t.config.Enabled {
		return nil, fmt.Errorf("Telegram notifications are disabled")
	}

	payload := map[string]interface{}{
		"offset": 0,
		"limit":  100,
	}

	var response struct {
		OK     bool             `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}

	if err := t.sendRequestWithResponse(ctx, "getUpdates", payload, &response); err != nil {
		return nil, err
	}

	if !response.OK {
		return nil, fmt.Errorf("telegram API returned error")
	}

	return response.Result, nil
}

// sendRequest sends a request to Telegram API
func (t *TelegramProviderImpl) sendRequest(ctx context.Context, method string, payload interface{}) error {
	var response struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	return t.sendRequestWithResponse(ctx, method, payload, &response)
}

// sendRequestWithResponse sends a request to Telegram API and parses response
func (t *TelegramProviderImpl) sendRequestWithResponse(ctx context.Context, method string, payload interface{}, response interface{}) error {
	url := fmt.Sprintf("%s/%s", t.baseURL, method)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
