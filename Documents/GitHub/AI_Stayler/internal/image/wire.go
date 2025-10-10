package image

import (
	"database/sql"
	"time"

	"AI_Styler/internal/storage"
)

// WireImageService creates an image service with all dependencies
func WireImageService(db *sql.DB) (*Service, *Handler) {
	// Create store
	store := NewDBStore(db)

	// Create file storage
	fileStorage, err := storage.NewStorageService(storage.StorageConfig{
		BasePath:     "./uploads",
		BackupPath:   "./backups",
		SignedURLKey: "default-key-change-in-production",
	})
	if err != nil {
		panic(err)
	}

	// Create image processor
	imageProcessor := NewImageProcessor()

	// Create usage tracker
	usageTracker := NewDBUsageTracker(db)

	// Create cache
	cache := NewRedisCache()

	// Create notification service
	notifier := NewNotificationService()

	// Create audit logger
	auditLogger := NewAuditLogger()

	// Create rate limiter
	rateLimiter := NewRateLimiter()

	// Create storage config
	config := StorageConfig{
		BasePath:      "./uploads",
		MaxFileSize:   MaxImageFileSize,
		AllowedTypes:  SupportedImageTypes,
		ThumbnailPath: "./uploads/thumbnails",
		SignedURLTTL:  int64(time.Hour.Seconds()),
	}

	// Create service
	service := NewService(
		store,
		fileStorage,
		imageProcessor,
		usageTracker,
		cache,
		notifier,
		auditLogger,
		rateLimiter,
		config,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// WireImageServiceWithMocks creates an image service with mock dependencies for testing
func WireImageServiceWithMocks(store Store) (*Service, *Handler) {
	// Create mock dependencies
	fileStorage := NewMockFileStorage()
	imageProcessor := NewMockImageProcessor()
	usageTracker := NewMockUsageTracker()
	cache := NewMockCache()
	notifier := NewMockNotificationService()
	auditLogger := NewMockAuditLogger()
	rateLimiter := NewMockRateLimiter()

	// Create storage config
	config := StorageConfig{
		BasePath:      "./uploads",
		MaxFileSize:   MaxImageFileSize,
		AllowedTypes:  SupportedImageTypes,
		ThumbnailPath: "./uploads/thumbnails",
		SignedURLTTL:  int64(time.Hour.Seconds()),
	}

	// Create service
	service := NewService(
		store,
		fileStorage,
		imageProcessor,
		usageTracker,
		cache,
		notifier,
		auditLogger,
		rateLimiter,
		config,
	)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}
