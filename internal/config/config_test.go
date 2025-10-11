package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Test with default values
	cfg, err := Load()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test default values
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", cfg.Database.Port)
	}
	if cfg.Server.HTTPAddr != ":8080" {
		t.Errorf("Expected HTTP addr ':8080', got '%s'", cfg.Server.HTTPAddr)
	}
	if cfg.SMS.Provider != "mock" {
		t.Errorf("Expected SMS provider 'mock', got '%s'", cfg.SMS.Provider)
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("DB_HOST", "test-host")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("DB_USER", "test-user")
	os.Setenv("DB_PASSWORD", "test-password")
	os.Setenv("DB_NAME", "test-db")
	os.Setenv("HTTP_ADDR", ":9090")
	os.Setenv("SMS_PROVIDER", "sms_ir")
	os.Setenv("SMS_API_KEY", "test-api-key")
	os.Setenv("SMS_TEMPLATE_ID", "123456")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_ACCESS_TTL", "30m")
	os.Setenv("JWT_REFRESH_TTL", "168h")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("HTTP_ADDR")
		os.Unsetenv("SMS_PROVIDER")
		os.Unsetenv("SMS_API_KEY")
		os.Unsetenv("SMS_TEMPLATE_ID")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_ACCESS_TTL")
		os.Unsetenv("JWT_REFRESH_TTL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test environment variable values
	if cfg.Database.Host != "test-host" {
		t.Errorf("Expected database host 'test-host', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Expected database port 3306, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "test-user" {
		t.Errorf("Expected database user 'test-user', got '%s'", cfg.Database.User)
	}
	if cfg.Database.Password != "test-password" {
		t.Errorf("Expected database password 'test-password', got '%s'", cfg.Database.Password)
	}
	if cfg.Database.Name != "test-db" {
		t.Errorf("Expected database name 'test-db', got '%s'", cfg.Database.Name)
	}
	if cfg.Server.HTTPAddr != ":9090" {
		t.Errorf("Expected HTTP addr ':9090', got '%s'", cfg.Server.HTTPAddr)
	}
	if cfg.SMS.Provider != "sms_ir" {
		t.Errorf("Expected SMS provider 'sms_ir', got '%s'", cfg.SMS.Provider)
	}
	if cfg.SMS.APIKey != "test-api-key" {
		t.Errorf("Expected SMS API key 'test-api-key', got '%s'", cfg.SMS.APIKey)
	}
	if cfg.SMS.TemplateID != 123456 {
		t.Errorf("Expected SMS template ID 123456, got %d", cfg.SMS.TemplateID)
	}
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("Expected JWT secret 'test-secret', got '%s'", cfg.JWT.Secret)
	}
	if cfg.JWT.AccessTTL != 30*time.Minute {
		t.Errorf("Expected JWT access TTL 30m, got %v", cfg.JWT.AccessTTL)
	}
	expectedRefreshTTL := 7 * 24 * time.Hour
	if cfg.JWT.RefreshTTL != expectedRefreshTTL {
		t.Errorf("Expected JWT refresh TTL %v, got %v", expectedRefreshTTL, cfg.JWT.RefreshTTL)
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	value := getEnv("TEST_VAR", "default")
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", value)
	}

	// Test with non-existing environment variable
	value = getEnv("NON_EXISTING_VAR", "default")
	if value != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	value := getEnvAsInt("TEST_INT", 0)
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}

	// Test with invalid integer
	os.Setenv("TEST_INVALID_INT", "not-a-number")
	defer os.Unsetenv("TEST_INVALID_INT")

	value = getEnvAsInt("TEST_INVALID_INT", 10)
	if value != 10 {
		t.Errorf("Expected 10, got %d", value)
	}

	// Test with non-existing variable
	value = getEnvAsInt("NON_EXISTING_INT", 5)
	if value != 5 {
		t.Errorf("Expected 5, got %d", value)
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	// Test with valid duration
	os.Setenv("TEST_DURATION", "2h30m")
	defer os.Unsetenv("TEST_DURATION")

	value := getEnvAsDuration("TEST_DURATION", time.Hour)
	expected := 2*time.Hour + 30*time.Minute
	if value != expected {
		t.Errorf("Expected %v, got %v", expected, value)
	}

	// Test with invalid duration
	os.Setenv("TEST_INVALID_DURATION", "not-a-duration")
	defer os.Unsetenv("TEST_INVALID_DURATION")

	value = getEnvAsDuration("TEST_INVALID_DURATION", time.Minute)
	if value != time.Minute {
		t.Errorf("Expected %v, got %v", time.Minute, value)
	}

	// Test with non-existing variable
	value = getEnvAsDuration("NON_EXISTING_DURATION", time.Second)
	if value != time.Second {
		t.Errorf("Expected %v, got %v", time.Second, value)
	}
}
