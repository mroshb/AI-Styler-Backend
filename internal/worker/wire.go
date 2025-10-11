package worker

import (
	"database/sql"
	"time"

	"ai-styler/internal/conversion"
	"ai-styler/internal/image"
	"ai-styler/internal/notification"
	"ai-styler/internal/storage"
)

// WireWorkerService creates a worker service with all dependencies
func WireWorkerService(db *sql.DB) (*Service, *Handler) {
	// Create configuration
	config := &WorkerConfig{
		MaxWorkers:        5,
		JobTimeout:        10 * time.Minute,
		RetryDelay:        5 * time.Second,
		MaxRetries:        3,
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

	// Create file storage
	fileStorage, err := storage.NewStorageService(storage.StorageConfig{
		BasePath:     "./uploads",
		BackupPath:   "./backups",
		SignedURLKey: "default-key-change-in-production",
	})
	if err != nil {
		panic(err)
	}

	// Create stores
	conversionStore := conversion.NewStore(db)
	imageStore := image.NewDBStore(db)

	// Create Gemini API client
	geminiConfig := &GeminiConfig{
		APIKey:     "your-gemini-api-key",
		BaseURL:    "https://generativelanguage.googleapis.com",
		Model:      "gemini-1.5-pro",
		MaxRetries: 3,
		Timeout:    60,
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

// WireWorkerServiceWithMocks creates a worker service with mock dependencies for testing
func WireWorkerServiceWithMocks() (*Service, *Handler) {
	// Create configuration
	config := &WorkerConfig{
		MaxWorkers:        2,
		JobTimeout:        5 * time.Minute,
		RetryDelay:        1 * time.Second,
		MaxRetries:        2,
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
