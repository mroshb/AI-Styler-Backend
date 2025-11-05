package worker

import (
	"database/sql"
	"time"

	"ai-styler/internal/config"
	"ai-styler/internal/conversion"
	"ai-styler/internal/image"
	"ai-styler/internal/notification"
	"ai-styler/internal/storage"
)

// WireWorkerService creates a worker service with all dependencies
func WireWorkerService(db *sql.DB, cfg *config.Config) (*Service, *Handler) {
	// Create worker configuration
	workerConfig := &WorkerConfig{
		MaxWorkers:        5,
		JobTimeout:        10 * time.Minute,
		RetryDelay:        5 * time.Second,
		MaxRetries:        1,
		PollInterval:      1 * time.Second,
		CleanupInterval:   1 * time.Hour,
		HealthCheckPort:   8082,
		EnableMetrics:     true,
		EnableHealthCheck: true,
	}

	// Create job queue
	jobQueue := NewDBJobQueue(db)

	// Create image processor
	imageProcessor := image.NewImageProcessor()

	// Create file storage using config
	backupPath := cfg.Storage.StoragePath + "/backup"
	fileStorage, err := storage.NewStorageService(storage.StorageConfig{
		BasePath:     cfg.Storage.StoragePath,
		BackupPath:   backupPath,
		SignedURLKey: "default-key-change-in-production", // TODO: Move to config
	})
	if err != nil {
		panic(err)
	}

	// Create stores
	conversionStore := conversion.NewStore(db)
	imageStore := image.NewDBStore(db)

	// Create Gemini API client using config
	geminiConfig := &GeminiConfig{
		APIKey:               cfg.Gemini.APIKey,
		BaseURL:              cfg.Gemini.BaseURL,
		Model:                cfg.Gemini.Model,
		MaxRetries:           cfg.Gemini.MaxRetries,
		Timeout:              cfg.Gemini.Timeout,
		PreprocessNoiseLevel: cfg.Gemini.PreprocessNoiseLevel,
		PreprocessJpegQuality: cfg.Gemini.PreprocessJpegQuality,
	}
	geminiAPI := NewGeminiClient(geminiConfig)

	// Create notification service
	notifier, _ := notification.WireNotificationService(db)

	// Create metrics collector
	metricsCollector := NewMetricsCollector()

	// Create health checker
	healthChecker := NewHealthChecker()

	// Create retry handler
	retryHandler := NewRetryHandler()

	// Create service
	service := NewService(
		workerConfig,
		jobQueue,
		imageProcessor,
		fileStorage,
		conversionStore,
		imageStore,
		geminiAPI,
		notifier,
		metricsCollector,
		healthChecker,
		retryHandler,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// WireWorkerServiceWithMocks creates a worker service with mock dependencies for testing
func WireWorkerServiceWithMocks() (*Service, *Handler) {
	// Create configuration
	config := &WorkerConfig{
		MaxWorkers:        2,
		JobTimeout:        5 * time.Minute,
		RetryDelay:        1 * time.Second,
		MaxRetries:        1,
		PollInterval:      500 * time.Millisecond,
		CleanupInterval:   30 * time.Minute,
		HealthCheckPort:   8082,
		EnableMetrics:     true,
		EnableHealthCheck: true,
	}

	// Create mock dependencies
	jobQueue := NewMockJobQueue()
	imageProcessor := image.NewMockImageProcessor()
	fileStorage := NewMockFileStorage()
	conversionStore := NewMockConversionStore()
	imageStore := NewMockImageStore()
	geminiAPI := NewMockGeminiAPI()
	notifier := NewMockNotificationService()
	metricsCollector := NewMockMetricsCollector()
	healthChecker := NewMockHealthChecker()
	retryHandler := NewMockRetryHandler()

	// Create service
	service := NewService(
		config,
		jobQueue,
		imageProcessor,
		fileStorage,
		conversionStore,
		imageStore,
		geminiAPI,
		notifier,
		metricsCollector,
		healthChecker,
		retryHandler,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}
