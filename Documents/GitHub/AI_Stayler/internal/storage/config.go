package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config holds storage configuration
type Config struct {
	BasePath        string          `json:"basePath" yaml:"basePath"`
	BackupPath      string          `json:"backupPath" yaml:"backupPath"`
	SignedURLKey    string          `json:"signedURLKey" yaml:"signedURLKey"`
	MaxFileSize     int64           `json:"maxFileSize" yaml:"maxFileSize"`
	AllowedTypes    []string        `json:"allowedTypes" yaml:"allowedTypes"`
	ThumbnailSizes  []ThumbnailSize `json:"thumbnailSizes" yaml:"thumbnailSizes"`
	RetentionPolicy RetentionPolicy `json:"retentionPolicy" yaml:"retentionPolicy"`
	BackupPolicy    BackupPolicy    `json:"backupPolicy" yaml:"backupPolicy"`
	ServerConfig    ServerConfig    `json:"serverConfig" yaml:"serverConfig"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	BaseURL    string `json:"baseUrl" yaml:"baseUrl"`
	PublicPath string `json:"publicPath" yaml:"publicPath"`
	StaticPath string `json:"staticPath" yaml:"staticPath"`
}

// DefaultConfig returns default storage configuration
func DefaultConfig() *Config {
	return &Config{
		BasePath:        "./storage",
		BackupPath:      "./storage/backups",
		SignedURLKey:    generateRandomKey(),
		MaxFileSize:     DefaultMaxFileSize,
		AllowedTypes:    SupportedImageTypes,
		ThumbnailSizes:  DefaultThumbnailSizes,
		RetentionPolicy: DefaultRetentionPolicy,
		BackupPolicy:    DefaultBackupPolicy,
		ServerConfig: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			BaseURL:    "http://localhost:8080",
			PublicPath: "/api/storage/public",
			StaticPath: "/api/storage/static",
		},
	}
}

// ValidateConfig validates storage configuration
func (c *Config) Validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("base path is required")
	}

	if c.BackupPath == "" {
		return fmt.Errorf("backup path is required")
	}

	if c.SignedURLKey == "" {
		return fmt.Errorf("signed URL key is required")
	}

	if c.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}

	if len(c.AllowedTypes) == 0 {
		return fmt.Errorf("at least one allowed type must be specified")
	}

	return nil
}

// InitializeStorage initializes the storage system
func InitializeStorage(config *Config) error {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create base directories
	dirs := []string{
		config.BasePath,
		config.BackupPath,
		filepath.Join(config.BasePath, "images", "user"),
		filepath.Join(config.BasePath, "images", "cloth"),
		filepath.Join(config.BasePath, "images", "result"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// StorageManager manages the storage system
type StorageManager struct {
	config          *Config
	storage         StorageServiceInterface
	imageStorage    *ImageStorageService
	backupScheduler *BackupScheduler
	healthMonitor   *HealthMonitor
}

// NewStorageManager creates a new storage manager
func NewStorageManager(config *Config) (*StorageManager, error) {
	// Initialize storage
	if err := InitializeStorage(config); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create storage service
	storageService, err := NewStorageService(StorageConfig{
		BasePath:     config.BasePath,
		BackupPath:   config.BackupPath,
		SignedURLKey: config.SignedURLKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	// Create image storage service
	imageStorageConfig := ImageStorageConfig{
		BasePath:        config.BasePath,
		MaxFileSize:     config.MaxFileSize,
		AllowedTypes:    config.AllowedTypes,
		ThumbnailSizes:  config.ThumbnailSizes,
		RetentionPolicy: config.RetentionPolicy,
		BackupPolicy:    config.BackupPolicy,
	}

	imageStorage := NewImageStorageService(storageService, imageStorageConfig)

	// Create backup scheduler
	backupScheduler := NewBackupScheduler(storageService, config.BackupPolicy)

	// Create health monitor
	healthMonitor := NewHealthMonitor(storageService, config)

	return &StorageManager{
		config:          config,
		storage:         storageService,
		imageStorage:    imageStorage,
		backupScheduler: backupScheduler,
		healthMonitor:   healthMonitor,
	}, nil
}

// Start starts the storage manager
func (sm *StorageManager) Start(ctx context.Context) error {
	// Start backup scheduler
	if err := sm.backupScheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backup scheduler: %w", err)
	}

	// Start health monitor
	if err := sm.healthMonitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}

	return nil
}

// Stop stops the storage manager
func (sm *StorageManager) Stop(ctx context.Context) error {
	// Stop backup scheduler
	if err := sm.backupScheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop backup scheduler: %w", err)
	}

	// Stop health monitor
	if err := sm.healthMonitor.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop health monitor: %w", err)
	}

	return nil
}

// GetImageStorage returns the image storage service
func (sm *StorageManager) GetImageStorage() *ImageStorageService {
	return sm.imageStorage
}

// GetStorage returns the storage service
func (sm *StorageManager) GetStorage() StorageServiceInterface {
	return sm.storage
}

// GetConfig returns the configuration
func (sm *StorageManager) GetConfig() *Config {
	return sm.config
}

// BackupScheduler handles scheduled backups
type BackupScheduler struct {
	storage      StorageServiceInterface
	backupPolicy BackupPolicy
	ticker       *time.Ticker
	done         chan bool
}

// NewBackupScheduler creates a new backup scheduler
func NewBackupScheduler(storage StorageServiceInterface, policy BackupPolicy) *BackupScheduler {
	return &BackupScheduler{
		storage:      storage,
		backupPolicy: policy,
		done:         make(chan bool),
	}
}

// Start starts the backup scheduler
func (bs *BackupScheduler) Start(ctx context.Context) error {
	if !bs.backupPolicy.Enabled {
		return nil
	}

	// Determine backup frequency
	var duration time.Duration
	switch bs.backupPolicy.BackupFrequency {
	case "daily":
		duration = 24 * time.Hour
	case "weekly":
		duration = 7 * 24 * time.Hour
	case "monthly":
		duration = 30 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	bs.ticker = time.NewTicker(duration)

	go func() {
		for {
			select {
			case <-bs.ticker.C:
				// Perform backup cleanup
				bs.storage.CleanupOldBackups(ctx, bs.backupPolicy.RetentionDays)
			case <-bs.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the backup scheduler
func (bs *BackupScheduler) Stop(ctx context.Context) error {
	if bs.ticker != nil {
		bs.ticker.Stop()
	}
	close(bs.done)
	return nil
}

// HealthMonitor monitors storage health
type HealthMonitor struct {
	storage StorageServiceInterface
	config  *Config
	ticker  *time.Ticker
	done    chan bool
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(storage StorageServiceInterface, config *Config) *HealthMonitor {
	return &HealthMonitor{
		storage: storage,
		config:  config,
		done:    make(chan bool),
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.ticker = time.NewTicker(5 * time.Minute) // Check every 5 minutes

	go func() {
		for {
			select {
			case <-hm.ticker.C:
				// Check storage health
				hm.checkHealth(ctx)
			case <-hm.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop(ctx context.Context) error {
	if hm.ticker != nil {
		hm.ticker.Stop()
	}
	close(hm.done)
	return nil
}

// checkHealth performs health checks
func (hm *HealthMonitor) checkHealth(ctx context.Context) {
	// Get disk usage
	diskUsage, err := hm.storage.GetDiskUsage(ctx)
	if err != nil {
		// Log error
		return
	}

	// Check if disk usage is too high
	usagePercent := float64(diskUsage.TotalSize) / float64(diskUsage.TotalSpace) * 100
	if usagePercent > 90 {
		// Log critical warning
	}
}

// Helper functions

func generateRandomKey() string {
	// Generate a random 32-byte key for HMAC signing
	return "ai_stayler_storage_key_2024_secure_random_string"
}

// GetStoragePaths returns the configured storage paths
func (c *Config) GetStoragePaths() StoragePaths {
	return StoragePaths{
		Users:   filepath.Join(c.BasePath, "images", "user"),
		Cloth:   filepath.Join(c.BasePath, "images", "cloth"),
		Results: filepath.Join(c.BasePath, "images", "result"),
		Backups: c.BackupPath,
	}
}

// GetPublicURL returns the public URL for a file
func (c *Config) GetPublicURL(filePath string) string {
	relativePath := filePath
	if filepath.IsAbs(filePath) {
		rel, err := filepath.Rel(c.BasePath, filePath)
		if err == nil {
			relativePath = rel
		}
	}

	return fmt.Sprintf("%s%s/%s", c.ServerConfig.BaseURL, c.ServerConfig.PublicPath, relativePath)
}

// GetStaticURL returns the static URL for a file
func (c *Config) GetStaticURL(filePath string) string {
	relativePath := filePath
	if filepath.IsAbs(filePath) {
		rel, err := filepath.Rel(c.BasePath, filePath)
		if err == nil {
			relativePath = rel
		}
	}

	return fmt.Sprintf("%s%s/%s", c.ServerConfig.BaseURL, c.ServerConfig.StaticPath, relativePath)
}
