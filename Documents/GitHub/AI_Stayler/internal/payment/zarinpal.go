package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ZarinpalGateway implements PaymentGateway interface for Zarinpal
type ZarinpalGateway struct {
	merchantID string
	baseURL    string
	httpClient *http.Client
}

// NewZarinpalGateway creates a new Zarinpal gateway instance
func NewZarinpalGateway(merchantID, baseURL string) *ZarinpalGateway {
	return &ZarinpalGateway{
		merchantID: merchantID,
		baseURL:    baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreatePayment creates a payment request with Zarinpal
func (z *ZarinpalGateway) CreatePayment(ctx context.Context, req ZarinpalRequest) (ZarinpalResponse, error) {
	// Set merchant ID
	req.Merchant = z.merchantID

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", z.baseURL+"/v1/request", bytes.NewBuffer(jsonData))
	if err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	resp, err := z.httpClient.Do(httpReq)
	if err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return ZarinpalResponse{}, fmt.Errorf("zarinpal API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var zarinpalResp ZarinpalResponse
	if err := json.Unmarshal(body, &zarinpalResp); err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check Zarinpal result code
	if zarinpalResp.Result != ZarinpalSuccess {
		return ZarinpalResponse{}, fmt.Errorf("zarinpal error %d: %s", zarinpalResp.Result, zarinpalResp.Message)
	}

	return zarinpalResp, nil
}

// VerifyPayment verifies a payment with Zarinpal
func (z *ZarinpalGateway) VerifyPayment(ctx context.Context, req ZarinpalVerifyRequest) (ZarinpalVerifyResponse, error) {
	// Set merchant ID
	req.Merchant = z.merchantID

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", z.baseURL+"/v1/verify", bytes.NewBuffer(jsonData))
	if err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	resp, err := z.httpClient.Do(httpReq)
	if err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return ZarinpalVerifyResponse{}, fmt.Errorf("zarinpal API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var zarinpalResp ZarinpalVerifyResponse
	if err := json.Unmarshal(body, &zarinpalResp); err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check Zarinpal result code
	if zarinpalResp.Result != ZarinpalSuccess {
		return ZarinpalVerifyResponse{}, fmt.Errorf("zarinpal error %d: %s", zarinpalResp.Result, zarinpalResp.Message)
	}

	return zarinpalResp, nil
}

// GetPaymentURL returns the payment URL for a given track ID
func (z *ZarinpalGateway) GetPaymentURL(trackID string) string {
	return fmt.Sprintf("%s/start/%s", z.baseURL, trackID)
}

// GetGatewayName returns the gateway name
func (z *ZarinpalGateway) GetGatewayName() string {
	return GatewayZarinpal
}

// ZarinpalError represents a Zarinpal-specific error
type ZarinpalError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *ZarinpalError) Error() string {
	return fmt.Sprintf("zarinpal error %d: %s", e.Code, e.Message)
}

// GetZarinpalErrorMessage returns a human-readable error message for Zarinpal result codes
func GetZarinpalErrorMessage(resultCode int) string {
	switch resultCode {
	case ZarinpalSuccess:
		return "Payment successful"
	case ZarinpalMerchantNotFound:
		return "Merchant not found"
	case ZarinpalMerchantInactive:
		return "Merchant is inactive or contract not signed"
	case ZarinpalMerchantInvalid:
		return "Invalid merchant"
	case ZarinpalAmountTooLow:
		return "Amount must be greater than 1,000 Rials"
	case ZarinpalCallbackInvalid:
		return "Invalid callback URL (must start with http or https)"
	case ZarinpalAmountExceeded:
		return "Amount exceeds transaction limit"
	case ZarinpalNationalCodeInvalid:
		return "Invalid national code"
	case ZarinpalAlreadyVerified:
		return "Payment already verified"
	case ZarinpalNotPaid:
		return "Payment not completed or failed"
	case ZarinpalTrackIDInvalid:
		return "Invalid track ID"
	default:
		return "Unknown error occurred"
	}
}

// GetZarinpalStatusMessage returns a human-readable message for Zarinpal status codes
func GetZarinpalStatusMessage(statusCode int) string {
	switch statusCode {
	case ZarinpalStatusPending:
		return "Payment pending"
	case ZarinpalStatusInternalErr:
		return "Internal error"
	case ZarinpalStatusPaid:
		return "Payment completed and verified"
	case ZarinpalStatusPaidUnverified:
		return "Payment completed but not verified"
	case ZarinpalStatusCancelled:
		return "Payment cancelled by user"
	case ZarinpalStatusCardInvalid:
		return "Invalid card number"
	case ZarinpalStatusInsufficientFunds:
		return "Insufficient funds"
	case ZarinpalStatusWrongPin:
		return "Wrong PIN"
	case ZarinpalStatusTooManyRequests:
		return "Too many requests"
	case ZarinpalStatusDailyLimitExceeded:
		return "Daily transaction limit exceeded"
	case ZarinpalStatusDailyAmountExceeded:
		return "Daily amount limit exceeded"
	case ZarinpalStatusInvalidIssuer:
		return "Invalid card issuer"
	case ZarinpalStatusSwitchError:
		return "Switch error"
	case ZarinpalStatusCardUnavailable:
		return "Card unavailable"
	case ZarinpalStatusRefunded:
		return "Payment refunded"
	case ZarinpalStatusRefunding:
		return "Payment being refunded"
	case ZarinpalStatusReversed:
		return "Payment reversed"
	default:
		return "Unknown status"
	}
}
