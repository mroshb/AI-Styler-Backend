package migration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

// RunMigrations runs pending database migrations
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name
	sort.Strings(files)

	var appliedCount int
	var skippedCount int

	for _, file := range files {
		filename := filepath.Base(file)
		version := strings.TrimSuffix(filename, ".sql")

		// Check if migration is already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", filename, err)
		}

		if count > 0 {
			skippedCount++
			continue
		}

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute migration (file already contains BEGIN/COMMIT)
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		// Update migrations table
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", filename, err)
		}

		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update migrations table for %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration record for %s: %w", filename, err)
		}

		appliedCount++
	}

	if appliedCount > 0 {
		fmt.Printf("Applied %d migration(s), skipped %d already applied\n", appliedCount, skippedCount)
	}

	return nil
}

// GetMigrationStatus returns the status of all migrations
func GetMigrationStatus(db *sql.DB, migrationsDir string) (map[string]bool, error) {
	status := make(map[string]bool)

	// Get all migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name
	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)
		version := strings.TrimSuffix(filename, ".sql")

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to check migration status for %s: %w", filename, err)
		}

		status[filename] = count > 0
	}

	return status, nil
}

func createMigrationsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	return err
}

