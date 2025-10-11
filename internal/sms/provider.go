package sms

// Provider interface for SMS sending
type Provider interface {
	Send(code string, phone string) error
}

// NewProvider creates a new SMS provider based on configuration
func NewProvider(providerType, apiKey string, templateID int) Provider {
	switch providerType {
	case "sms_ir":
		return NewSMSIrProvider(apiKey, templateID)
	case "mock":
		fallthrough
	default:
		return NewMockSMSProvider()
	}
}
