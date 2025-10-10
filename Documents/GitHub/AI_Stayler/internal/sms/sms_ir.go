package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SMSIrProvider struct {
	APIKey     string
	TemplateID int
	BaseURL    string
	HTTPClient *http.Client
}

type VerifySendModel struct {
	Mobile     string                     `json:"mobile"`
	TemplateID int                        `json:"templateId"`
	Parameters []VerifySendParameterModel `json:"parameters"`
}

type VerifySendParameterModel struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SMSIrResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    struct {
		MessageID int     `json:"messageId"`
		Cost      float64 `json:"cost"`
	} `json:"data"`
}

func NewSMSIrProvider(apiKey string, templateID int) *SMSIrProvider {
	return &SMSIrProvider{
		APIKey:     apiKey,
		TemplateID: templateID,
		BaseURL:    "https://api.sms.ir/v1/send/verify",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SMSIrProvider) Send(code string, phone string) error {
	// Remove + from phone number if present
	if len(phone) > 0 && phone[0] == '+' {
		phone = phone[1:]
	}

	// Create request payload
	payload := VerifySendModel{
		Mobile:     phone,
		TemplateID: s.TemplateID,
		Parameters: []VerifySendParameterModel{
			{
				Name:  "Code",
				Value: code,
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SMS request: %w", err)
	}

	// Log the request being sent
	fmt.Printf("SMS API Request: %s\n", string(jsonData))

	// Create HTTP request
	req, err := http.NewRequest("POST", s.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.APIKey)

	// Send request
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for logging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response
	fmt.Printf("SMS API Response Status: %d\n", resp.StatusCode)
	fmt.Printf("SMS API Response Body: %s\n", string(bodyBytes))

	// Parse response
	var smsResp SMSIrResponse
	if err := json.Unmarshal(bodyBytes, &smsResp); err != nil {
		return fmt.Errorf("failed to decode SMS response: %w", err)
	}

	// Check if successful
	if smsResp.Status != 1 {
		return fmt.Errorf("SMS send failed: %s", smsResp.Message)
	}

	fmt.Printf("SMS sent successfully! MessageID: %d, Cost: %.2f\n", smsResp.Data.MessageID, smsResp.Data.Cost)
	return nil
}
