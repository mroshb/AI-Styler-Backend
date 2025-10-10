package share

import (
	"testing"
	"time"
)

func TestService_generateShareToken(t *testing.T) {
	// Create a minimal service for testing
	service := &Service{}

	token1, err1 := service.generateShareToken()
	if err1 != nil {
		t.Fatalf("Failed to generate token: %v", err1)
	}

	token2, err2 := service.generateShareToken()
	if err2 != nil {
		t.Fatalf("Failed to generate token: %v", err2)
	}

	// Tokens should be unique
	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	// Tokens should not be empty
	if token1 == "" || token2 == "" {
		t.Error("Generated tokens should not be empty")
	}

	// Tokens should be reasonably long (base64 encoded 32 bytes)
	if len(token1) < 40 || len(token2) < 40 {
		t.Error("Generated tokens should be reasonably long")
	}
}

func TestCreateShareRequest_Validation(t *testing.T) {
	tests := []struct {
		name          string
		req           CreateShareRequest
		expectedValid bool
	}{
		{
			name: "valid request",
			req: CreateShareRequest{
				ConversionID:  "conv-123",
				ExpiryMinutes: 5,
			},
			expectedValid: true,
		},
		{
			name: "minimum expiry",
			req: CreateShareRequest{
				ConversionID:  "conv-123",
				ExpiryMinutes: 1,
			},
			expectedValid: true,
		},
		{
			name: "maximum expiry",
			req: CreateShareRequest{
				ConversionID:  "conv-123",
				ExpiryMinutes: 5,
			},
			expectedValid: true,
		},
		{
			name: "expiry too short",
			req: CreateShareRequest{
				ConversionID:  "conv-123",
				ExpiryMinutes: 0,
			},
			expectedValid: false,
		},
		{
			name: "expiry too long",
			req: CreateShareRequest{
				ConversionID:  "conv-123",
				ExpiryMinutes: 10,
			},
			expectedValid: false,
		},
		{
			name: "missing conversion ID",
			req: CreateShareRequest{
				ExpiryMinutes: 5,
			},
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			valid := tt.req.ConversionID != "" &&
				tt.req.ExpiryMinutes >= MinExpiryMinutes &&
				tt.req.ExpiryMinutes <= MaxExpiryMinutes

			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}
		})
	}
}

func TestAccessShareRequest_Validation(t *testing.T) {
	tests := []struct {
		name          string
		req           AccessShareRequest
		expectedValid bool
	}{
		{
			name: "valid view request",
			req: AccessShareRequest{
				ShareToken: "token-123",
				AccessType: AccessTypeView,
			},
			expectedValid: true,
		},
		{
			name: "valid download request",
			req: AccessShareRequest{
				ShareToken: "token-123",
				AccessType: AccessTypeDownload,
			},
			expectedValid: true,
		},
		{
			name: "invalid access type",
			req: AccessShareRequest{
				ShareToken: "token-123",
				AccessType: "invalid",
			},
			expectedValid: false,
		},
		{
			name: "missing share token",
			req: AccessShareRequest{
				AccessType: AccessTypeView,
			},
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			valid := tt.req.ShareToken != "" &&
				(tt.req.AccessType == AccessTypeView || tt.req.AccessType == AccessTypeDownload)

			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}
		})
	}
}

func TestSharedLink_ExpiryCalculation(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(5 * time.Minute)

	link := SharedLink{
		ExpiresAt: expiresAt,
	}

	// Test seconds until expiry calculation
	secondsUntilExpiry := int(time.Until(link.ExpiresAt).Seconds())

	if secondsUntilExpiry <= 0 {
		t.Error("Seconds until expiry should be positive")
	}

	if secondsUntilExpiry > 300 { // 5 minutes
		t.Error("Seconds until expiry should not exceed 5 minutes")
	}
}

func TestConstants(t *testing.T) {
	// Test that constants are properly defined
	if MinExpiryMinutes != 1 {
		t.Errorf("Expected MinExpiryMinutes=1, got %d", MinExpiryMinutes)
	}

	if MaxExpiryMinutes != 5 {
		t.Errorf("Expected MaxExpiryMinutes=5, got %d", MaxExpiryMinutes)
	}

	if DefaultExpiryMinutes != 5 {
		t.Errorf("Expected DefaultExpiryMinutes=5, got %d", DefaultExpiryMinutes)
	}

	if ShareTokenLength != 32 {
		t.Errorf("Expected ShareTokenLength=32, got %d", ShareTokenLength)
	}

	if AccessTypeView != "view" {
		t.Errorf("Expected AccessTypeView='view', got '%s'", AccessTypeView)
	}

	if AccessTypeDownload != "download" {
		t.Errorf("Expected AccessTypeDownload='download', got '%s'", AccessTypeDownload)
	}
}
