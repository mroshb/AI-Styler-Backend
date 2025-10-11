package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailProviderImpl implements EmailProvider interface
type EmailProviderImpl struct {
	config EmailConfig
}

// NewEmailProvider creates a new email provider
func NewEmailProvider(config EmailConfig) EmailProvider {
	return &EmailProviderImpl{
		config: config,
	}
}

// SendEmail sends an email
func (e *EmailProviderImpl) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	if !e.config.Enabled {
		return fmt.Errorf("email notifications are disabled")
	}

	// Validate email
	if !e.ValidateEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	// Prepare message
	message := e.prepareMessage(to, subject, body, isHTML)

	// Send email
	auth := smtp.PlainAuth("", e.config.SMTPUsername, e.config.SMTPPassword, e.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	if err := smtp.SendMail(addr, auth, e.config.FromEmail, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendTemplateEmail sends an email using a template
func (e *EmailProviderImpl) SendTemplateEmail(ctx context.Context, to, templateID string, data map[string]interface{}) error {
	// This would integrate with a template engine
	// For now, we'll use a simple approach
	subject := "Notification"
	body := "You have a new notification"

	return e.SendEmail(ctx, to, subject, body, true)
}

// ValidateEmail validates an email address
func (e *EmailProviderImpl) ValidateEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// prepareMessage prepares the email message
func (e *EmailProviderImpl) prepareMessage(to, subject, body string, isHTML bool) string {
	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	message := fmt.Sprintf("From: %s <%s>\r\n", e.config.FromName, e.config.FromEmail)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType)
	message += "\r\n"
	message += body

	return message
}
