package telegram

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the Telegram bot
type Config struct {
	Telegram TelegramConfig
	API      APIConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Security SecurityConfig
	Server   ServerConfig
	RateLimit RateLimitConfig
}

// TelegramConfig holds Telegram-specific configuration
type TelegramConfig struct {
	BotToken string
	Env      string // development or production
}

// APIConfig holds backend API configuration
type APIConfig struct {
	BaseURL    string
	APIKey     string // Optional API key for bot-to-API auth
	Timeout    time.Duration
	RetryCount int
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	DSN string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Host     string
	Port     int
	Password string
	DB       int
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	MaxUploadSize int64 // in bytes
	AllowedMIME   []string
}

// ServerConfig holds webhook server configuration
type ServerConfig struct {
	WebhookURL string
	WebhookPort int
	HealthPort  int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MessagesPerMinute    int
	ConversionsPerHour   int
	WindowDuration       time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			Env:      getEnv("BOT_ENV", "development"),
		},
		API: APIConfig{
			BaseURL:    getEnv("API_BASE_URL", "http://localhost:8080"),
			APIKey:     getEnv("API_KEY_FOR_BOT", ""),
			Timeout:    getEnvAsDuration("API_TIMEOUT", 30*time.Second),
			RetryCount: getEnvAsInt("API_RETRY_COUNT", 3),
		},
		Database: DatabaseConfig{
			DSN: getEnv("POSTGRES_DSN", ""),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Security: SecurityConfig{
			MaxUploadSize: parseSize(getEnv("MAX_UPLOAD_SIZE", "10MB")),
			AllowedMIME:   []string{"image/jpeg", "image/png", "image/webp", "image/jpg"},
		},
		Server: ServerConfig{
			WebhookURL:  getEnv("WEBHOOK_URL", ""),
			WebhookPort: getEnvAsInt("WEBHOOK_PORT", 8443),
			HealthPort:  getEnvAsInt("HEALTH_PORT", 8081),
		},
		RateLimit: RateLimitConfig{
			MessagesPerMinute:  getEnvAsInt("RATE_LIMIT_MESSAGES", 10),
			ConversionsPerHour: getEnvAsInt("RATE_LIMIT_CONVERSIONS", 5),
			WindowDuration:     getEnvAsDuration("RATE_LIMIT_WINDOW", time.Minute),
		},
	}

	// Build Redis URL if not provided
	if cfg.Redis.URL == "" {
		cfg.Redis.URL = buildRedisURL(cfg.Redis)
	}

	// Build PostgreSQL DSN if not provided
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = buildPostgresDSN()
	}

	return cfg, nil
}

// Helper functions
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

func parseSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	
	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "B") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 10 * 1024 * 1024 // Default 10MB
	}

	return size * multiplier
}

func buildRedisURL(redis RedisConfig) string {
	if redis.Password != "" {
		return "redis://:" + redis.Password + "@" + redis.Host + ":" + strconv.Itoa(redis.Port) + "/" + strconv.Itoa(redis.DB)
	}
	return "redis://" + redis.Host + ":" + strconv.Itoa(redis.Port) + "/" + strconv.Itoa(redis.DB)
}

func buildPostgresDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "styler")
	sslmode := getEnv("DB_SSLMODE", "disable")

	if password != "" {
		return "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=" + sslmode
	}
	return "host=" + host + " port=" + port + " user=" + user + " dbname=" + dbname + " sslmode=" + sslmode
}

