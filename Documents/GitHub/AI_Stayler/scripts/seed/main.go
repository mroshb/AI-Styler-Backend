package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"ai-styler/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go [seed|clear|status]")
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

	switch command {
	case "seed":
		if err := seedDatabase(db); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	case "clear":
		if err := clearSeedData(db); err != nil {
			log.Fatalf("Failed to clear seed data: %v", err)
		}
	case "status":
		if err := showSeedStatus(db); err != nil {
			log.Fatalf("Failed to show seed status: %v", err)
		}
	default:
		log.Fatal("Invalid command. Use: seed, clear, or status")
	}
}

func seedDatabase(db *sql.DB) error {
	fmt.Println("Starting database seeding...")

	// Seed admin users
	if err := seedAdminUsers(db); err != nil {
		return fmt.Errorf("failed to seed admin users: %v", err)
	}

	// Seed default plans
	if err := seedDefaultPlans(db); err != nil {
		return fmt.Errorf("failed to seed default plans: %v", err)
	}

	// Seed sample vendors
	if err := seedSampleVendors(db); err != nil {
		return fmt.Errorf("failed to seed sample vendors: %v", err)
	}

	// Seed system settings
	if err := seedSystemSettings(db); err != nil {
		return fmt.Errorf("failed to seed system settings: %v", err)
	}

	fmt.Println("Database seeding completed successfully!")
	return nil
}

func seedAdminUsers(db *sql.DB) error {
	fmt.Println("Seeding admin users...")

	adminUsers := []struct {
		Phone    string
		Name     string
		Email    string
		Role     string
		IsActive bool
	}{
		{
			Phone:    "+989123456789",
			Name:     "Super Admin",
			Email:    "admin@aistyler.com",
			Role:     "super_admin",
			IsActive: true,
		},
		{
			Phone:    "+989123456790",
			Name:     "System Admin",
			Email:    "system@aistyler.com",
			Role:     "admin",
			IsActive: true,
		},
	}

	for _, admin := range adminUsers {
		// Check if admin already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE phone = $1", admin.Phone).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			fmt.Printf("Admin user %s already exists, skipping...\n", admin.Phone)
			continue
		}

		// Insert admin user
		query := `
			INSERT INTO users (id, phone, name, email, role, is_active, is_verified, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())
		`

		userID := generateUUID()
		_, err = db.Exec(query, userID, admin.Phone, admin.Name, admin.Email, admin.Role, admin.IsActive)
		if err != nil {
			return err
		}

		fmt.Printf("Created admin user: %s (%s)\n", admin.Name, admin.Phone)
	}

	return nil
}

func seedDefaultPlans(db *sql.DB) error {
	fmt.Println("Seeding default plans...")

	plans := []struct {
		Name        string
		Description string
		Price       int64
		Duration    int
		Features    map[string]interface{}
		IsActive    bool
	}{
		{
			Name:        "Free Plan",
			Description: "Basic plan with limited features",
			Price:       0,
			Duration:    30,
			Features: map[string]interface{}{
				"max_conversions":   10,
				"max_images":        50,
				"storage_limit":     "100MB",
				"api_access":        false,
				"priority_support":  false,
				"watermark_removal": false,
			},
			IsActive: true,
		},
		{
			Name:        "Basic Plan",
			Description: "Standard plan for regular users",
			Price:       99000, // 99,000 IRR
			Duration:    30,
			Features: map[string]interface{}{
				"max_conversions":   100,
				"max_images":        500,
				"storage_limit":     "1GB",
				"api_access":        true,
				"priority_support":  false,
				"watermark_removal": true,
			},
			IsActive: true,
		},
		{
			Name:        "Premium Plan",
			Description: "Advanced plan for power users",
			Price:       199000, // 199,000 IRR
			Duration:    30,
			Features: map[string]interface{}{
				"max_conversions":   500,
				"max_images":        2500,
				"storage_limit":     "5GB",
				"api_access":        true,
				"priority_support":  true,
				"watermark_removal": true,
			},
			IsActive: true,
		},
		{
			Name:        "Enterprise Plan",
			Description: "Unlimited plan for businesses",
			Price:       499000, // 499,000 IRR
			Duration:    30,
			Features: map[string]interface{}{
				"max_conversions":    -1, // unlimited
				"max_images":         -1, // unlimited
				"storage_limit":      "50GB",
				"api_access":         true,
				"priority_support":   true,
				"watermark_removal":  true,
				"custom_integration": true,
			},
			IsActive: true,
		},
	}

	for _, plan := range plans {
		// Check if plan already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM plans WHERE name = $1", plan.Name).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			fmt.Printf("Plan %s already exists, skipping...\n", plan.Name)
			continue
		}

		// Insert plan
		query := `
			INSERT INTO plans (id, name, description, price, duration_days, features, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`

		planID := generateUUID()
		_, err = db.Exec(query, planID, plan.Name, plan.Description, plan.Price, plan.Duration, plan.Features, plan.IsActive)
		if err != nil {
			return err
		}

		fmt.Printf("Created plan: %s\n", plan.Name)
	}

	return nil
}

func seedSampleVendors(db *sql.DB) error {
	fmt.Println("Seeding sample vendors...")

	vendors := []struct {
		Name        string
		Description string
		Website     string
		ContactInfo map[string]interface{}
		IsActive    bool
	}{
		{
			Name:        "AI Style Pro",
			Description: "Professional AI styling services",
			Website:     "https://aistylepro.com",
			ContactInfo: map[string]interface{}{
				"email":   "contact@aistylepro.com",
				"phone":   "+989123456789",
				"address": "Tehran, Iran",
			},
			IsActive: true,
		},
		{
			Name:        "Style Master",
			Description: "Expert styling and design services",
			Website:     "https://stylemaster.ir",
			ContactInfo: map[string]interface{}{
				"email":   "info@stylemaster.ir",
				"phone":   "+989123456790",
				"address": "Isfahan, Iran",
			},
			IsActive: true,
		},
	}

	for _, vendor := range vendors {
		// Check if vendor already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM vendors WHERE name = $1", vendor.Name).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			fmt.Printf("Vendor %s already exists, skipping...\n", vendor.Name)
			continue
		}

		// Insert vendor
		query := `
			INSERT INTO vendors (id, name, description, website, contact_info, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		`

		vendorID := generateUUID()
		_, err = db.Exec(query, vendorID, vendor.Name, vendor.Description, vendor.Website, vendor.ContactInfo, vendor.IsActive)
		if err != nil {
			return err
		}

		fmt.Printf("Created vendor: %s\n", vendor.Name)
	}

	return nil
}

func seedSystemSettings(db *sql.DB) error {
	fmt.Println("Seeding system settings...")

	settings := []struct {
		Key   string
		Value interface{}
		Type  string
	}{
		{"app_name", "AI Styler", "string"},
		{"app_version", "1.0.0", "string"},
		{"maintenance_mode", false, "boolean"},
		{"max_file_size", "50MB", "string"},
		{"allowed_file_types", []string{"jpg", "jpeg", "png", "gif", "webp"}, "array"},
		{"default_plan_id", "", "string"}, // Will be set after plans are created
		{"smtp_enabled", false, "boolean"},
		{"sms_enabled", true, "boolean"},
		{"payment_enabled", true, "boolean"},
		{"analytics_enabled", true, "boolean"},
		{"backup_enabled", true, "boolean"},
		{"backup_frequency", "daily", "string"},
		{"retention_days", 365, "integer"},
		{"rate_limit_enabled", true, "boolean"},
		{"rate_limit_per_minute", 60, "integer"},
		{"security_scan_enabled", true, "boolean"},
		{"watermark_enabled", true, "boolean"},
		{"api_rate_limit", 1000, "integer"},
		{"max_concurrent_conversions", 10, "integer"},
		{"conversion_timeout", 300, "integer"}, // 5 minutes
	}

	for _, setting := range settings {
		// Check if setting already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM system_settings WHERE key = $1", setting.Key).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			fmt.Printf("Setting %s already exists, skipping...\n", setting.Key)
			continue
		}

		// Insert setting
		query := `
			INSERT INTO system_settings (id, key, value, type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`

		settingID := generateUUID()
		_, err = db.Exec(query, settingID, setting.Key, setting.Value, setting.Type)
		if err != nil {
			return err
		}

		fmt.Printf("Created setting: %s\n", setting.Key)
	}

	// Set default plan ID after plans are created
	var defaultPlanID string
	err := db.QueryRow("SELECT id FROM plans WHERE name = 'Free Plan' LIMIT 1").Scan(&defaultPlanID)
	if err == nil {
		_, err = db.Exec("UPDATE system_settings SET value = $1 WHERE key = 'default_plan_id'", defaultPlanID)
		if err != nil {
			return err
		}
		fmt.Printf("Set default plan ID: %s\n", defaultPlanID)
	}

	return nil
}

func clearSeedData(db *sql.DB) error {
	fmt.Println("Clearing seed data...")

	tables := []string{
		"system_settings",
		"plans",
		"vendors",
		"users",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		result, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %v", table, err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Cleared %d rows from %s\n", rowsAffected, table)
	}

	fmt.Println("Seed data cleared successfully!")
	return nil
}

func showSeedStatus(db *sql.DB) error {
	fmt.Println("Seed Data Status:")
	fmt.Println("=================")

	// Check users
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return err
	}
	fmt.Printf("Users: %d\n", userCount)

	// Check plans
	var planCount int
	err = db.QueryRow("SELECT COUNT(*) FROM plans").Scan(&planCount)
	if err != nil {
		return err
	}
	fmt.Printf("Plans: %d\n", planCount)

	// Check vendors
	var vendorCount int
	err = db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&vendorCount)
	if err != nil {
		return err
	}
	fmt.Printf("Vendors: %d\n", vendorCount)

	// Check system settings
	var settingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM system_settings").Scan(&settingCount)
	if err != nil {
		return err
	}
	fmt.Printf("System Settings: %d\n", settingCount)

	return nil
}

// Simple UUID generator (for demo purposes)
func generateUUID() string {
	// In production, use a proper UUID library
	return fmt.Sprintf("uuid-%d", len(fmt.Sprintf("%p", generateUUID)))
}
