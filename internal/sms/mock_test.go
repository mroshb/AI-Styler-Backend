package sms

import (
	"testing"
)

func TestMockSMSProvider_Send(t *testing.T) {
	provider := NewMockSMSProvider()

	// Test successful send
	err := provider.Send("123456", "+9123456789")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestNewProvider_Mock(t *testing.T) {
	provider := NewProvider("mock", "", 0)

	// Should return MockSMSProvider
	if _, ok := provider.(*MockSMSProvider); !ok {
		t.Error("Expected MockSMSProvider for 'mock' provider type")
	}
}

func TestNewProvider_SMSIr(t *testing.T) {
	provider := NewProvider("sms_ir", "test-key", 100000)

	// Should return SMSIrProvider
	if _, ok := provider.(*SMSIrProvider); !ok {
		t.Error("Expected SMSIrProvider for 'sms_ir' provider type")
	}
}

func TestNewProvider_Default(t *testing.T) {
	provider := NewProvider("unknown", "", 0)

	// Should return MockSMSProvider as default
	if _, ok := provider.(*MockSMSProvider); !ok {
		t.Error("Expected MockSMSProvider for unknown provider type")
	}
}
