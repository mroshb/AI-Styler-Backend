package payment

import (
	"os"
	"strconv"
)

// PaymentConfigServiceImpl implements payment configuration
type PaymentConfigServiceImpl struct{}

// NewPaymentConfigService creates a new config service
func NewPaymentConfigService() *PaymentConfigServiceImpl {
	return &PaymentConfigServiceImpl{}
}

// GetZarinpalMerchantID returns the Zarinpal merchant ID
func (c *PaymentConfigServiceImpl) GetZarinpalMerchantID() string {
	merchantID := os.Getenv("ZARINPAL_MERCHANT_ID")
	if merchantID == "" {
		return "zibal" // Test merchant ID
	}
	return merchantID
}

// GetZarinpalBaseURL returns the Zarinpal base URL
func (c *PaymentConfigServiceImpl) GetZarinpalBaseURL() string {
	baseURL := os.Getenv("ZARINPAL_BASE_URL")
	if baseURL == "" {
		return "https://gateway.zibal.ir"
	}
	return baseURL
}

// GetPaymentCallbackURL returns the payment callback URL
func (c *PaymentConfigServiceImpl) GetPaymentCallbackURL() string {
	callbackURL := os.Getenv("PAYMENT_CALLBACK_URL")
	if callbackURL == "" {
		return "https://your-domain.com/api/payments/webhooks/notify?gateway=zarinpal"
	}
	return callbackURL
}

// GetPaymentReturnURL returns the default payment return URL
func (c *PaymentConfigServiceImpl) GetPaymentReturnURL() string {
	returnURL := os.Getenv("PAYMENT_RETURN_URL")
	if returnURL == "" {
		return "https://your-domain.com/payment/success"
	}
	return returnURL
}

// GetPaymentExpiryMinutes returns the payment expiry time in minutes
func (c *PaymentConfigServiceImpl) GetPaymentExpiryMinutes() int {
	expiryStr := os.Getenv("PAYMENT_EXPIRY_MINUTES")
	if expiryStr == "" {
		return 30 // Default 30 minutes
	}

	expiry, err := strconv.Atoi(expiryStr)
	if err != nil {
		return 30
	}
	return expiry
}
