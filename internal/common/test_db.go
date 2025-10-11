package common

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestDBConfig represents test database configuration
type TestDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetTestDBConfig returns test database configuration from environment or defaults
func GetTestDBConfig() *TestDBConfig {
	return &TestDBConfig{
		Host:     getEnvOrDefault("TEST_DB_HOST", getEnvOrDefault("DB_HOST", "localhost")),
		Port:     getEnvOrDefault("TEST_DB_PORT", getEnvOrDefault("DB_PORT", "5432")),
		User:     getEnvOrDefault("TEST_DB_USER", getEnvOrDefault("DB_USER", "postgres")),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", getEnvOrDefault("DB_PASSWORD", "")),
		DBName:   getEnvOrDefault("TEST_DB_NAME", "styler"),
		SSLMode:  getEnvOrDefault("TEST_DB_SSLMODE", getEnvOrDefault("DB_SSLMODE", "disable")),
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetTestDSN returns the test database connection string
func GetTestDSN() string {
	config := GetTestDBConfig()
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
}

// TestDB represents a test database connection
type TestDB struct {
	DB     *sql.DB
	Config *TestDBConfig
}

// NewTestDB creates a new test database connection
func NewTestDB() (*TestDB, error) {
	config := GetTestDBConfig()
	dsn := GetTestDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &TestDB{
		DB:     db,
		Config: config,
	}, nil
}

// SetupTestDB sets up the test database with required tables
func (tdb *TestDB) SetupTestDB() error {
	// Create test database if it doesn't exist
	createDBQuery := fmt.Sprintf("CREATE DATABASE %s", tdb.Config.DBName)

	// Connect to postgres database to create test database
	postgresDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		tdb.Config.Host, tdb.Config.Port, tdb.Config.User, tdb.Config.Password, tdb.Config.SSLMode)

	postgresDB, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer postgresDB.Close()

	// Try to create database (ignore if it already exists)
	_, err = postgresDB.Exec(createDBQuery)
	if err != nil {
		// Check if database already exists
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)"
		err = postgresDB.QueryRow(checkQuery, tdb.Config.DBName).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("failed to create test database: %w", err)
		}
	}

	// Run migrations on test database
	return tdb.runMigrations()
}

// runMigrations runs database migrations
func (tdb *TestDB) runMigrations() error {
	// Read and execute migration files
	migrations := []string{
		// Auth migrations
		`CREATE TABLE IF NOT EXISTS otp_verifications (
			id SERIAL PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			code VARCHAR(10) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'user',
			is_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User service migrations
		`CREATE TABLE IF NOT EXISTS user_profiles (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			email VARCHAR(255),
			avatar_url VARCHAR(500),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS user_conversions (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			user_image_id VARCHAR(36) NOT NULL,
			cloth_image_id VARCHAR(36) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			result_image_id VARCHAR(36),
			error_message TEXT,
			processing_time_ms INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS user_plans (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			plan_name VARCHAR(50) NOT NULL,
			monthly_limit INTEGER NOT NULL DEFAULT 0,
			remaining_conversions INTEGER NOT NULL DEFAULT 0,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS conversion_quotas (
			user_id VARCHAR(36) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			free_conversions_used INTEGER DEFAULT 0,
			paid_conversions_used INTEGER DEFAULT 0,
			last_reset_date DATE DEFAULT CURRENT_DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Vendor service migrations
		`CREATE TABLE IF NOT EXISTS vendor_profiles (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			business_name VARCHAR(255) NOT NULL,
			description TEXT,
			website VARCHAR(500),
			phone VARCHAR(20),
			email VARCHAR(255),
			address TEXT,
			logo_url VARCHAR(500),
			is_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS vendor_albums (
			id VARCHAR(36) PRIMARY KEY,
			vendor_id VARCHAR(36) NOT NULL REFERENCES vendor_profiles(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			is_public BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS vendor_images (
			id VARCHAR(36) PRIMARY KEY,
			vendor_id VARCHAR(36) NOT NULL REFERENCES vendor_profiles(id) ON DELETE CASCADE,
			album_id VARCHAR(36) REFERENCES vendor_albums(id) ON DELETE SET NULL,
			file_name VARCHAR(255) NOT NULL,
			original_url VARCHAR(500) NOT NULL,
			thumbnail_url VARCHAR(500),
			file_size BIGINT NOT NULL,
			mime_type VARCHAR(100) NOT NULL,
			width INTEGER,
			height INTEGER,
			is_public BOOLEAN DEFAULT TRUE,
			tags TEXT[],
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Image service migrations
		`CREATE TABLE IF NOT EXISTS images (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) REFERENCES users(id) ON DELETE CASCADE,
			vendor_id VARCHAR(36) REFERENCES vendor_profiles(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			original_url VARCHAR(500) NOT NULL,
			thumbnail_url VARCHAR(500),
			file_size BIGINT NOT NULL,
			mime_type VARCHAR(100) NOT NULL,
			width INTEGER,
			height INTEGER,
			is_public BOOLEAN DEFAULT FALSE,
			tags TEXT[],
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS image_usage_history (
			id VARCHAR(36) PRIMARY KEY,
			image_id VARCHAR(36) NOT NULL REFERENCES images(id) ON DELETE CASCADE,
			user_id VARCHAR(36) REFERENCES users(id) ON DELETE CASCADE,
			action VARCHAR(50) NOT NULL,
			ip_address INET,
			user_agent TEXT,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, migration := range migrations {
		if _, err := tdb.DB.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}

// CleanupTestDB cleans up test data
func (tdb *TestDB) CleanupTestDB() error {
	// Clean up test data
	queries := []string{
		"DELETE FROM image_usage_history WHERE user_id LIKE 'test-%'",
		"DELETE FROM images WHERE user_id LIKE 'test-%' OR vendor_id IN (SELECT id FROM vendor_profiles WHERE user_id LIKE 'test-%')",
		"DELETE FROM vendor_images WHERE vendor_id IN (SELECT id FROM vendor_profiles WHERE user_id LIKE 'test-%')",
		"DELETE FROM vendor_albums WHERE vendor_id IN (SELECT id FROM vendor_profiles WHERE user_id LIKE 'test-%')",
		"DELETE FROM vendor_profiles WHERE user_id LIKE 'test-%'",
		"DELETE FROM conversion_quotas WHERE user_id LIKE 'test-%'",
		"DELETE FROM user_plans WHERE user_id LIKE 'test-%'",
		"DELETE FROM user_conversions WHERE user_id LIKE 'test-%'",
		"DELETE FROM user_profiles WHERE user_id LIKE 'test-%'",
		"DELETE FROM users WHERE id LIKE 'test-%'",
		"DELETE FROM otp_verifications WHERE phone LIKE 'test-%'",
	}

	for _, query := range queries {
		if _, err := tdb.DB.Exec(query); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Warning: failed to cleanup test data: %v\n", err)
		}
	}

	return nil
}

// Close closes the test database connection
func (tdb *TestDB) Close() error {
	return tdb.DB.Close()
}

// SkipIfNoDB skips the test if database is not available
func SkipIfNoDB(t *testing.T) {
	tdb, err := NewTestDB()
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
	}
	tdb.Close()
}

// MustNewTestDB creates a new test database or fails the test
func MustNewTestDB(t *testing.T) *TestDB {
	tdb, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := tdb.SetupTestDB(); err != nil {
		tdb.Close()
		t.Fatalf("Failed to setup test database: %v", err)
	}

	return tdb
}
