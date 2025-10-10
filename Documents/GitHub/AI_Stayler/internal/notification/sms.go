package notification

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SMSProviderImpl implements SMSProvider interface
type SMSProviderImpl struct {
	config     SMSConfig
	httpClient *http.Client
}

// NewSMSProvider creates a new SMS provider
func NewSMSProvider(config SMSConfig) SMSProvider {
	return &SMSProviderImpl{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendSMS sends an SMS message
func (s *SMSProviderImpl) SendSMS(ctx context.Context, phone, message string) error {
	if !s.config.Enabled {
		return fmt.Errorf("SMS notifications are disabled")
	}

	// Validate phone number
	if !s.ValidatePhone(phone) {
		return fmt.Errorf("invalid phone number: %s", phone)
	}

	// Send SMS based on provider
	switch s.config.Provider {
	case "sms_ir":
		return s.sendViaSMSIR(ctx, phone, message)
	case "kavenegar":
		return s.sendViaKavenegar(ctx, phone, message)
	default:
		return fmt.Errorf("unsupported SMS provider: %s", s.config.Provider)
	}
}

// SendTemplateSMS sends an SMS using a template
func (s *SMSProviderImpl) SendTemplateSMS(ctx context.Context, phone, templateID string, data map[string]interface{}) error {
	// This would integrate with a template engine
	// For now, we'll use a simple approach
	message := "You have a new notification"

	return s.SendSMS(ctx, phone, message)
}

// ValidatePhone validates a phone number
func (s *SMSProviderImpl) ValidatePhone(phone string) bool {
	// Simple validation - in production, use a proper phone validation library
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	return len(phone) >= 10 && len(phone) <= 15 && strings.HasPrefix(phone, "+")
}

// sendViaSMSIR sends SMS via SMS.ir
func (s *SMSProviderImpl) sendViaSMSIR(ctx context.Context, phone, message string) error {
	// SMS.ir API implementation
	apiURL := "https://api.sms.ir/v1/send/verify"

	data := url.Values{}
	data.Set("mobile", phone)
	data.Set("templateId", fmt.Sprintf("%d", s.config.TemplateID))
	data.Set("parameter", message)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-API-KEY", s.config.APIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendViaKavenegar sends SMS via Kavenegar
func (s *SMSProviderImpl) sendViaKavenegar(ctx context.Context, phone, message string) error {
	// Kavenegar API implementation
	apiURL := fmt.Sprintf("https://api.kavenegar.com/v1/%s/sms/send.json", s.config.APIKey)

	data := url.Values{}
	data.Set("receptor", phone)
	data.Set("message", message)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d", resp.StatusCode)
	}

	return nil
}
