package sms

import "log"

// MockSMSProvider is a mock implementation for testing
type MockSMSProvider struct{}

func NewMockSMSProvider() *MockSMSProvider {
	return &MockSMSProvider{}
}

func (m *MockSMSProvider) Send(code string, phone string) error {
	log.Printf("MOCK SMS: Sending code %s to phone %s", code, phone)
	log.Printf("MOCK SMS: In production, this would send real SMS via SMS.ir API")
	return nil
}

func (m *MockSMSProvider) IsMock() bool {
	return true
}
