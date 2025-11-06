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

// ZibalGateway implements PaymentGateway interface for Zibal
type ZibalGateway struct {
	merchantID string
	baseURL    string
	httpClient *http.Client
}

// NewZibalGateway creates a new Zibal gateway instance
func NewZibalGateway(merchantID, baseURL string) *ZibalGateway {
	if baseURL == "" {
		baseURL = "https://gateway.zibal.ir"
	}
	return &ZibalGateway{
		merchantID: merchantID,
		baseURL:    baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreatePayment creates a payment request with Zibal
func (z *ZibalGateway) CreatePayment(ctx context.Context, req ZarinpalRequest) (ZarinpalResponse, error) {
	// Set merchant ID
	req.Merchant = z.merchantID

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request - Zibal uses /v1/request endpoint
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
		return ZarinpalResponse{}, fmt.Errorf("zibal API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var zibalResp ZarinpalResponse
	if err := json.Unmarshal(body, &zibalResp); err != nil {
		return ZarinpalResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check Zibal result code (same as Zarinpal)
	if zibalResp.Result != ZarinpalSuccess {
		return ZarinpalResponse{}, fmt.Errorf("zibal error %d: %s", zibalResp.Result, zibalResp.Message)
	}

	return zibalResp, nil
}

// VerifyPayment verifies a payment with Zibal
func (z *ZibalGateway) VerifyPayment(ctx context.Context, req ZarinpalVerifyRequest) (ZarinpalVerifyResponse, error) {
	// Set merchant ID
	req.Merchant = z.merchantID

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request - Zibal uses /v1/verify endpoint
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
		return ZarinpalVerifyResponse{}, fmt.Errorf("zibal API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var zibalResp ZarinpalVerifyResponse
	if err := json.Unmarshal(body, &zibalResp); err != nil {
		return ZarinpalVerifyResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check Zibal result code (same as Zarinpal)
	if zibalResp.Result != ZarinpalSuccess {
		return ZarinpalVerifyResponse{}, fmt.Errorf("zibal error %d: %s", zibalResp.Result, zibalResp.Message)
	}

	return zibalResp, nil
}

// GetPaymentURL returns the payment URL for a given track ID
func (z *ZibalGateway) GetPaymentURL(trackID string) string {
	return fmt.Sprintf("%s/start/%s", z.baseURL, trackID)
}

// GetGatewayName returns the gateway name
func (z *ZibalGateway) GetGatewayName() string {
	return GatewayZibal
}

