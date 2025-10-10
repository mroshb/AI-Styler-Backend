package user

import (
	"testing"

	"AI_Styler/internal/common"
)

func TestUserService_Integration(t *testing.T) {
	// Skip if database is not available
	common.SkipIfNoDB(t)

	// Create test database
	tdb := common.MustNewTestDB(t)
	defer tdb.Close()
	defer tdb.CleanupTestDB()

	// Test database connection
	if err := tdb.DB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Log("Database connection successful")

	// Test basic database operations
	t.Run("DatabaseConnection", func(t *testing.T) {
		var result int
		err := tdb.DB.QueryRow("SELECT 1").Scan(&result)
		if err != nil {
			t.Fatalf("Failed to execute test query: %v", err)
		}

		if result != 1 {
			t.Errorf("Expected 1, got %d", result)
		}
	})

	t.Run("TableExists", func(t *testing.T) {
		// Check if users table exists
		var exists bool
		err := tdb.DB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = 'users'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check if users table exists: %v", err)
		}

		if !exists {
			t.Error("Users table does not exist")
		}
	})
}

func TestUserService_HTTP_Integration(t *testing.T) {
	// Skip if database is not available
	common.SkipIfNoDB(t)

	// Create test database
	tdb := common.MustNewTestDB(t)
	defer tdb.Close()
	defer tdb.CleanupTestDB()

	// Test database connection
	if err := tdb.DB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Log("HTTP integration test database connection successful")
}
