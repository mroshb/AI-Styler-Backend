package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ai-styler/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go [up|down|status]")
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	switch command {
	case "up":
		if err := runMigrations(db, "up"); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
	case "down":
		if err := runMigrations(db, "down"); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
	case "status":
		if err := showMigrationStatus(db); err != nil {
			log.Fatalf("Failed to show migration status: %v", err)
		}
	default:
		log.Fatal("Invalid command. Use: up, down, or status")
	}
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

func runMigrations(db *sql.DB, direction string) error {
	migrationsDir := "db/migrations"

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}

	// Sort files by name
	sort.Strings(files)

	if direction == "down" {
		// Reverse order for down migrations
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	for _, file := range files {
		filename := filepath.Base(file)
		version := strings.TrimSuffix(filename, ".sql")

		// Check if migration is already applied
		if direction == "up" {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
			if err != nil {
				return err
			}
			if count > 0 {
				fmt.Printf("Skipping %s (already applied)\n", filename)
				continue
			}
		} else {
			// For down migrations, check if it's applied
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
			if err != nil {
				return err
			}
			if count == 0 {
				fmt.Printf("Skipping %s (not applied)\n", filename)
				continue
			}
		}

		fmt.Printf("Running %s migration: %s\n", direction, filename)

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		// Execute migration (file already contains BEGIN/COMMIT)
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %v", filename, err)
		}

		// Update migrations table in a separate transaction
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if direction == "up" {
			_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING", version)
		} else {
			_, err = tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
		}

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update migrations table for %s: %v", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		fmt.Printf("Successfully applied %s migration: %s\n", direction, filename)
	}

	return nil
}

func showMigrationStatus(db *sql.DB) error {
	migrationsDir := "db/migrations"

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}

	sort.Strings(files)

	fmt.Println("Migration Status:")
	fmt.Println("================")

	for _, file := range files {
		filename := filepath.Base(file)
		version := strings.TrimSuffix(filename, ".sql")

		var appliedAt sql.NullString
		err := db.QueryRow("SELECT applied_at FROM schema_migrations WHERE version = $1", version).Scan(&appliedAt)

		if err == sql.ErrNoRows {
			fmt.Printf("❌ %s (not applied)\n", filename)
		} else if err != nil {
			return err
		} else {
			fmt.Printf("✅ %s (applied at %s)\n", filename, appliedAt.String)
		}
	}

	return nil
}
