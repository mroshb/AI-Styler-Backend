package sms

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSMSIrProvider_Send(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Errorf("Expected API key 'test-api-key', got '%s'", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Parse request body
		var req VerifySendModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request structure
		if req.Mobile != "9123456789" {
			t.Errorf("Expected mobile '9123456789', got '%s'", req.Mobile)
		}
		if req.TemplateID != 100000 {
			t.Errorf("Expected template ID 100000, got %d", req.TemplateID)
		}
		if len(req.Parameters) != 1 {
			t.Errorf("Expected 1 parameter, got %d", len(req.Parameters))
		}
		if req.Parameters[0].Name != "Code" {
			t.Errorf("Expected parameter name 'Code', got '%s'", req.Parameters[0].Name)
		}
		if req.Parameters[0].Value != "123456" {
			t.Errorf("Expected parameter value '123456', got '%s'", req.Parameters[0].Value)
		}

		// Send success response
		response := SMSIrResponse{
			Status:  1,
			Message: "موفق",
			Data: struct {
				MessageID int     `json:"messageId"`
				Cost      float64 `json:"cost"`
			}{
				MessageID: 12345,
				Cost:      1.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider := &SMSIrProvider{
		APIKey:     "test-api-key",
		TemplateID: 100000,
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	// Test successful send
	err := provider.Send("123456", "+9123456789")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestSMSIrProvider_Send_ErrorResponse(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SMSIrResponse{
			Status:  0,
			Message: "خطا در ارسال",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := &SMSIrProvider{
		APIKey:     "test-api-key",
		TemplateID: 100000,
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	// Test error response
	err := provider.Send("123456", "+9123456789")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "SMS send failed: خطا در ارسال" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestSMSIrProvider_Send_PhoneFormatting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req VerifySendModel
		json.NewDecoder(r.Body).Decode(&req)

		// Should remove + from phone number
		if req.Mobile != "9123456789" {
			t.Errorf("Expected mobile without +, got '%s'", req.Mobile)
		}

		response := SMSIrResponse{Status: 1, Message: "موفق"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := &SMSIrProvider{
		APIKey:     "test-api-key",
		TemplateID: 100000,
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	err := provider.Send("123456", "+9123456789")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
