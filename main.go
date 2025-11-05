package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-styler/internal/admin"
	"ai-styler/internal/auth"
	"ai-styler/internal/config"
	"ai-styler/internal/conversion"
	"ai-styler/internal/image"
	"ai-styler/internal/logging"
	"ai-styler/internal/migration"
	"ai-styler/internal/monitoring"
	"ai-styler/internal/notification"
	"ai-styler/internal/payment"
	"ai-styler/internal/route"
	"ai-styler/internal/security"
	"ai-styler/internal/share"
	"ai-styler/internal/sms"
	"ai-styler/internal/storage"
	"ai-styler/internal/user"
	"ai-styler/internal/vendors"
	"ai-styler/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

// SimpleLogger implements the storage.Logger interface
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	log.Println("[INFO]", msg, fields)
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	log.Println("[ERROR]", msg, fields)
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	log.Println("[DEBUG]", msg, fields)
}

// MockWorkerService is a mock implementation for testing
type MockWorkerService struct{}

func (m *MockWorkerService) Start(ctx context.Context) error {
	log.Println("Mock worker service started")
	return nil
}

func (m *MockWorkerService) Stop(ctx context.Context) error {
	log.Println("Mock worker service stopped")
	return nil
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger (needed for migration logs)
	logger := logging.NewStructuredLogger(logging.LoggerConfig{
		Level:  logging.ParseLogLevel(cfg.Monitoring.LogLevel),
		Format: "json",
	})
	logger.Info(context.Background(), "Starting AI Styler backend service", nil)

	// Initialize database connection
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run database migrations if enabled
	if cfg.Database.AutoMigrate {
		logger.Info(context.Background(), "Running database migrations...", nil)
		if err := migration.RunMigrations(db, cfg.Database.MigrationsDir); err != nil {
			log.Fatalf("failed to run database migrations: %v", err)
		}
		logger.Info(context.Background(), "Database migrations completed", nil)
	}

	// Initialize Redis connection
	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Printf("failed to initialize Redis: %v", err)
		redisClient = nil // Continue without Redis
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Initialize monitoring service
	monitorConfig := monitoring.MonitoringConfig{
		Sentry: monitoring.SentryConfig{
			DSN:              cfg.Monitoring.SentryDSN,
			Environment:      cfg.Monitoring.Environment,
			Release:          cfg.Monitoring.Version,
			Debug:            cfg.Monitoring.Environment == "development",
			SampleRate:       1.0,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			MaxBreadcrumbs:   50,
		},
		Telegram: monitoring.TelegramConfig{
			BotToken: cfg.Monitoring.TelegramBotToken,
			ChatID:   cfg.Monitoring.TelegramChatID,
			Enabled:  cfg.Monitoring.TelegramBotToken != "" && cfg.Monitoring.TelegramChatID != "",
			Timeout:  10 * time.Second,
		},
		Logging: logging.LoggerConfig{
			Level:       logging.ParseLogLevel(cfg.Monitoring.LogLevel),
			Format:      "json",
			Output:      "stdout",
			Service:     "ai-stayler",
			Version:     cfg.Monitoring.Version,
			Environment: cfg.Monitoring.Environment,
		},
		Health: monitoring.HealthConfig{
			Enabled:       cfg.Monitoring.HealthEnabled,
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}

	monitor, err := monitoring.NewMonitoringService(monitorConfig, db, redisClient)
	if err != nil {
		log.Fatalf("failed to initialize monitoring service: %v", err)
	}
	defer monitor.Close()

	// Initialize storage
	storageLogger := &SimpleLogger{}
	backupPath := cfg.Storage.StoragePath + "/backup"
	localStorage := storage.NewLocalStorage(cfg.Storage.StoragePath, backupPath, storageLogger)
	_ = localStorage // Use localStorage to avoid unused variable error

	// Initialize stores
	authStore := auth.NewPostgresStore(db)

	// Initialize security components
	rateLimiter := auth.NewInMemoryLimiter()
	
	// Use ProductionTokenService with PostgreSQL session store for persistent sessions
	jwtSigner := security.NewProductionJWTSigner(cfg.JWT.Secret, "ai-styler")
	sessionStore := auth.NewPostgresSessionStore(db)
	accessTTL := cfg.JWT.AccessTTL
	refreshTTL := cfg.JWT.RefreshTTL
	if refreshTTL == 0 {
		refreshTTL = 30 * 24 * time.Hour // Default: 30 days (720 hours)
	}
	productionTokenService := auth.NewProductionTokenService(jwtSigner, sessionStore, accessTTL, refreshTTL)
	tokenService := auth.NewTokenServiceAdapter(productionTokenService)

	// Initialize SMS provider from configuration
	smsProvider := sms.NewProviderWithParameter(cfg.SMS.Provider, cfg.SMS.APIKey, cfg.SMS.TemplateID, cfg.SMS.ParameterName)

	// Initialize services with dependencies
	authHandler := auth.NewHandler(authStore, tokenService, rateLimiter, smsProvider)

	// Initialize all services
	_, userHandler := user.WireUserService(db)
	_, vendorHandler := vendors.WireVendorService(db)
	_, conversionHandler := conversion.WireConversionService(db)
	_, imageHandler := image.WireImageService(db)
	_, paymentHandler := payment.WirePaymentService(db)
	_, shareHandler := share.WireShareService(db)
	_, adminHandler := admin.WireAdminService(db)
	_, notificationHandler := notification.WireNotificationService(db)

	// Initialize worker service with config
	workerService, _ := worker.WireWorkerService(db, cfg)

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Create router with all services
	r := route.NewWithServices(
		authHandler,
		userHandler,
		vendorHandler,
		conversionHandler,
		imageHandler,
		paymentHandler,
		shareHandler,
		adminHandler,
		notificationHandler,
		monitor,
	)

	// Start worker service in background
	go func() {
		logger.Info(context.Background(), "Starting worker service", nil)
		if err := workerService.Start(context.Background()); err != nil {
			logger.Error(context.Background(), "Worker service failed", map[string]interface{}{"error": err})
		}
	}()

	// Start server in a goroutine
	server := &http.Server{
		Addr:    cfg.Server.HTTPAddr,
		Handler: r,
	}

	go func() {
		monitor.LogInfo(context.Background(), "Server starting", map[string]interface{}{
			"addr": cfg.Server.HTTPAddr,
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			monitor.LogFatal(context.Background(), "Server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(context.Background(), "Shutting down server...", nil)

	// Stop worker service
	logger.Info(context.Background(), "Stopping worker service", nil)
	if err := workerService.Stop(context.Background()); err != nil {
		logger.Error(context.Background(), "Failed to stop worker service", map[string]interface{}{"error": err})
	}

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal(context.Background(), "Server forced to shutdown", map[string]interface{}{"error": err})
	}

	logger.Info(context.Background(), "Server exited", nil)
}

// initDatabase initializes database connection
func initDatabase(cfg *config.Config) (*sql.DB, error) {
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
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// initRedis initializes Redis connection
func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
