package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database   DatabaseConfig
	Server     ServerConfig
	JWT        JWTConfig
	Redis      RedisConfig
	SMS        SMSConfig
	Security   SecurityConfig
	RateLimit  RateLimitConfig
	Storage    StorageConfig
	Monitoring MonitoringConfig
	Gemini     GeminiConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type ServerConfig struct {
	HTTPAddr string
	GinMode  string
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type SMSConfig struct {
	Provider   string
	APIKey     string
	TemplateID int
}

type SecurityConfig struct {
	BCryptCost        int
	Argon2Memory      uint32
	Argon2Iterations  uint32
	Argon2Parallelism uint8
	Argon2SaltLength  uint32
	Argon2KeyLength   uint32
}

type RateLimitConfig struct {
	OTPPerPhone   int
	OTPPerIP      int
	LoginPerPhone int
	LoginPerIP    int
	Window        time.Duration
}

type StorageConfig struct {
	UploadMaxSize string
	StoragePath   string
	SignedURLTTL  time.Duration
}

type MonitoringConfig struct {
	TelegramBotToken string
	TelegramChatID   string
	LogLevel         string
	SentryDSN        string
	Environment      string
	Version          string
	HealthEnabled    bool
}

type GeminiConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Timeout    int
	MaxRetries int
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional, continue without it
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "styler"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
			GinMode:  getEnv("GIN_MODE", "debug"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessTTL:  getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getEnvAsDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		SMS: SMSConfig{
			Provider:   getEnv("SMS_PROVIDER", "mock"),
			APIKey:     getEnv("SMS_API_KEY", ""),
			TemplateID: getEnvAsInt("SMS_TEMPLATE_ID", 100000),
		},
		Security: SecurityConfig{
			BCryptCost:        getEnvAsInt("BCRYPT_COST", 12),
			Argon2Memory:      uint32(getEnvAsInt("ARGON2_MEMORY", 65536)),
			Argon2Iterations:  uint32(getEnvAsInt("ARGON2_ITERATIONS", 3)),
			Argon2Parallelism: uint8(getEnvAsInt("ARGON2_PARALLELISM", 2)),
			Argon2SaltLength:  uint32(getEnvAsInt("ARGON2_SALT_LENGTH", 16)),
			Argon2KeyLength:   uint32(getEnvAsInt("ARGON2_KEY_LENGTH", 32)),
		},
		RateLimit: RateLimitConfig{
			OTPPerPhone:   getEnvAsInt("RATE_LIMIT_OTP_PER_PHONE", 3),
			OTPPerIP:      getEnvAsInt("RATE_LIMIT_OTP_PER_IP", 100),
			LoginPerPhone: getEnvAsInt("RATE_LIMIT_LOGIN_PER_PHONE", 5),
			LoginPerIP:    getEnvAsInt("RATE_LIMIT_LOGIN_PER_IP", 10),
			Window:        getEnvAsDuration("RATE_LIMIT_WINDOW", time.Hour),
		},
		Storage: StorageConfig{
			UploadMaxSize: getEnv("UPLOAD_MAX_SIZE", "10MB"),
			StoragePath:   getEnv("STORAGE_PATH", "./uploads"),
			SignedURLTTL:  getEnvAsDuration("SIGNED_URL_TTL", time.Hour),
		},
		Monitoring: MonitoringConfig{
			TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
			LogLevel:         getEnv("LOG_LEVEL", "info"),
			SentryDSN:        getEnv("SENTRY_DSN", ""),
			Environment:      getEnv("ENVIRONMENT", "development"),
			Version:          getEnv("VERSION", "1.0.0"),
			HealthEnabled:    getEnvAsBool("HEALTH_ENABLED", true),
		},
		Gemini: GeminiConfig{
			APIKey:     getEnv("GEMINI_API_KEY", ""),
			BaseURL:    getEnv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
			Model:      getEnv("GEMINI_MODEL", "gemini-pro-vision"),
			Timeout:    getEnvAsInt("GEMINI_TIMEOUT", 300),
			MaxRetries: getEnvAsInt("GEMINI_MAX_RETRIES", 3),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
