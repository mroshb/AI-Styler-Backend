package sms

// Provider interface for SMS sending
type Provider interface {
	Send(code string, phone string) error
	IsMock() bool // Returns true if this is a mock provider (for development)
}

// NewProvider creates a new SMS provider based on configuration
func NewProvider(providerType, apiKey string, templateID int) Provider {
	return NewProviderWithParameter(providerType, apiKey, templateID, "Code")
}

// NewProviderWithParameter creates a new SMS provider with custom parameter name
func NewProviderWithParameter(providerType, apiKey string, templateID int, parameterName string) Provider {
	switch providerType {
	case "sms_ir":
		return NewSMSIrProviderWithParameter(apiKey, templateID, parameterName)
	case "mock":
		fallthrough
	default:
		return NewMockSMSProvider()
	}
}
