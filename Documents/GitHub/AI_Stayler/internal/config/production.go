package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// ProductionConfig represents production-ready configuration
type ProductionConfig struct {
	Config
	Security   SecurityProductionConfig
	Database   DatabaseProductionConfig
	Redis      RedisProductionConfig
	Monitoring MonitoringProductionConfig
}

// SecurityProductionConfig represents production security configuration
type SecurityProductionConfig struct {
	SecurityConfig
	JWTSecretLength  int
	SessionTimeout   time.Duration
	MaxLoginAttempts int
	LockoutDuration  time.Duration
}

// DatabaseProductionConfig represents production database configuration
type DatabaseProductionConfig struct {
	DatabaseConfig
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SSLMode         string
}

// RedisProductionConfig represents production Redis configuration
type RedisProductionConfig struct {
	RedisConfig
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
}

// MonitoringProductionConfig represents production monitoring configuration
type MonitoringProductionConfig struct {
	MonitoringConfig
	MetricsEnabled   bool
	TracingEnabled   bool
	ProfilingEnabled bool
	LogRetentionDays int
	AlertThresholds  map[string]float64
}

// LoadProduction loads production configuration with enhanced settings
func LoadProduction() (*ProductionConfig, error) {
	// Load base configuration
	baseConfig, err := Load()
	if err != nil {
		return nil, err
	}

	config := &ProductionConfig{
		Config: *baseConfig,
		Security: SecurityProductionConfig{
			SecurityConfig:   baseConfig.Security,
			JWTSecretLength:  getEnvAsInt("JWT_SECRET_LENGTH", 64),
			SessionTimeout:   getEnvAsDuration("SESSION_TIMEOUT", 24*time.Hour),
			MaxLoginAttempts: getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:  getEnvAsDuration("LOCKOUT_DURATION", 15*time.Minute),
		},
		Database: DatabaseProductionConfig{
			DatabaseConfig:  baseConfig.Database,
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 1*time.Minute),
			SSLMode:         getEnv("DB_SSLMODE", "require"),
		},
		Redis: RedisProductionConfig{
			RedisConfig:  baseConfig.Redis,
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 100),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 10),
			MaxRetries:   getEnvAsInt("REDIS_MAX_RETRIES", 3),
			DialTimeout:  getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolTimeout:  getEnvAsDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
		},
		Monitoring: MonitoringProductionConfig{
			MonitoringConfig: baseConfig.Monitoring,
			MetricsEnabled:   getEnvAsBool("METRICS_ENABLED", true),
			TracingEnabled:   getEnvAsBool("TRACING_ENABLED", true),
			ProfilingEnabled: getEnvAsBool("PROFILING_ENABLED", false),
			LogRetentionDays: getEnvAsInt("LOG_RETENTION_DAYS", 30),
			AlertThresholds: map[string]float64{
				"error_rate":    getEnvAsFloat("ALERT_ERROR_RATE", 0.05),
				"response_time": getEnvAsFloat("ALERT_RESPONSE_TIME", 2.0),
				"cpu_usage":     getEnvAsFloat("ALERT_CPU_USAGE", 0.80),
				"memory_usage":  getEnvAsFloat("ALERT_MEMORY_USAGE", 0.85),
				"disk_usage":    getEnvAsFloat("ALERT_DISK_USAGE", 0.90),
			},
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the production configuration
func (c *ProductionConfig) Validate() error {
	// Validate JWT secret
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	// Validate database configuration
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate Redis configuration
	if c.Redis.Host == "" {
		return fmt.Errorf("Redis host is required")
	}
	if c.Redis.Port <= 0 || c.Redis.Port > 65535 {
		return fmt.Errorf("invalid Redis port: %d", c.Redis.Port)
	}

	// Validate security configuration
	if c.Security.BCryptCost < 10 || c.Security.BCryptCost > 15 {
		return fmt.Errorf("BCrypt cost must be between 10 and 15")
	}

	// Validate rate limiting configuration
	if c.RateLimit.OTPPerPhone <= 0 {
		return fmt.Errorf("OTP per phone limit must be positive")
	}
	if c.RateLimit.OTPPerIP <= 0 {
		return fmt.Errorf("OTP per IP limit must be positive")
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *ProductionConfig) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisOptions returns Redis client options
func (c *ProductionConfig) GetRedisOptions() *redis.Options {
	return &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port),
		Password:     c.Redis.Password,
		DB:           c.Redis.DB,
		PoolSize:     c.Redis.PoolSize,
		MinIdleConns: c.Redis.MinIdleConns,
		MaxRetries:   c.Redis.MaxRetries,
		DialTimeout:  c.Redis.DialTimeout,
		ReadTimeout:  c.Redis.ReadTimeout,
		WriteTimeout: c.Redis.WriteTimeout,
		PoolTimeout:  c.Redis.PoolTimeout,
	}
}

// IsProduction returns true if running in production mode
func (c *ProductionConfig) IsProduction() bool {
	return c.Monitoring.Environment == "production"
}

// IsDevelopment returns true if running in development mode
func (c *ProductionConfig) IsDevelopment() bool {
	return c.Monitoring.Environment == "development"
}

// IsTesting returns true if running in testing mode
func (c *ProductionConfig) IsTesting() bool {
	return c.Monitoring.Environment == "testing"
}

// GetLogLevel returns the appropriate log level
func (c *ProductionConfig) GetLogLevel() string {
	if c.IsProduction() {
		return "info"
	}
	return c.Monitoring.LogLevel
}

// GetGinMode returns the appropriate Gin mode
func (c *ProductionConfig) GetGinMode() string {
	if c.IsProduction() {
		return "release"
	}
	return c.Server.GinMode
}

// Helper function to get environment variable as float
func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
