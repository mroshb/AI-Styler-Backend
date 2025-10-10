package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Wire provides dependency injection for storage services
type Wire struct {
	config       *Config
	storage      StorageServiceInterface
	imageStorage *ImageStorageService
	handler      *Handler
	manager      *StorageManager
}

// NewWire creates a new storage wire
func NewWire(config *Config) *Wire {
	return &Wire{
		config: config,
	}
}

// Initialize initializes all storage services
func (w *Wire) Initialize(ctx context.Context) error {
	// Create storage manager
	manager, err := NewStorageManager(w.config)
	if err != nil {
		return fmt.Errorf("failed to create storage manager: %w", err)
	}

	w.manager = manager
	w.storage = manager.GetStorage()
	w.imageStorage = manager.GetImageStorage()

	// Create handler
	w.handler = NewHandler(w.imageStorage, w.storage)

	// Start storage manager
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start storage manager: %w", err)
	}

	return nil
}

// GetStorage returns the storage service
func (w *Wire) GetStorage() StorageServiceInterface {
	return w.storage
}

// GetImageStorage returns the image storage service
func (w *Wire) GetImageStorage() *ImageStorageService {
	return w.imageStorage
}

// GetHandler returns the HTTP handler
func (w *Wire) GetHandler() *Handler {
	return w.handler
}

// GetManager returns the storage manager
func (w *Wire) GetManager() *StorageManager {
	return w.manager
}

// GetConfig returns the configuration
func (w *Wire) GetConfig() *Config {
	return w.config
}

// Shutdown gracefully shuts down the storage services
func (w *Wire) Shutdown(ctx context.Context) error {
	if w.manager != nil {
		return w.manager.Stop(ctx)
	}
	return nil
}

// StorageRepository provides database operations for storage
type StorageRepository struct {
	db *sql.DB
}

// NewStorageRepository creates a new storage repository
func NewStorageRepository(db *sql.DB) *StorageRepository {
	return &StorageRepository{db: db}
}

// CreateStorageFile creates a storage file record
func (r *StorageRepository) CreateStorageFile(ctx context.Context, req CreateStorageFileRequest) (string, error) {
	query := `
		SELECT create_storage_file($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var fileID string
	err := r.db.QueryRowContext(ctx, query,
		req.FilePath,
		req.FileName,
		req.FileSize,
		req.MimeType,
		req.Checksum,
		req.StorageType,
		req.OwnerID,
		req.OwnerType,
		req.IsPublic,
		req.Metadata,
	).Scan(&fileID)

	if err != nil {
		return "", fmt.Errorf("failed to create storage file: %w", err)
	}

	return fileID, nil
}

// GetStorageFile retrieves a storage file record
func (r *StorageRepository) GetStorageFile(ctx context.Context, fileID string) (*StorageFile, error) {
	query := `
		SELECT id, file_path, file_name, file_size, mime_type, checksum,
		       storage_type, owner_id, owner_type, is_public, is_backed_up,
		       backup_path, thumbnail_path, metadata, created_at, updated_at,
		       last_accessed, access_count
		FROM storage_files
		WHERE id = $1
	`

	var file StorageFile
	var createdAt, updatedAt, lastAccessed sql.NullTime

	err := r.db.QueryRowContext(ctx, query, fileID).Scan(
		&file.ID,
		&file.FilePath,
		&file.FileName,
		&file.FileSize,
		&file.MimeType,
		&file.Checksum,
		&file.StorageType,
		&file.OwnerID,
		&file.OwnerType,
		&file.IsPublic,
		&file.IsBackedUp,
		&file.BackupPath,
		&file.ThumbnailPath,
		&file.Metadata,
		&createdAt,
		&updatedAt,
		&lastAccessed,
		&file.AccessCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get storage file: %w", err)
	}

	if createdAt.Valid {
		file.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		file.UpdatedAt = updatedAt.Time
	}
	if lastAccessed.Valid {
		file.LastAccessed = &lastAccessed.Time
	}

	return &file, nil
}

// UpdateStorageFile updates a storage file record
func (r *StorageRepository) UpdateStorageFile(ctx context.Context, fileID string, req UpdateStorageFileRequest) error {
	query := `
		UPDATE storage_files
		SET is_public = COALESCE($2, is_public),
		    metadata = COALESCE($3, metadata),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, fileID, req.IsPublic, req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to update storage file: %w", err)
	}

	return nil
}

// DeleteStorageFile deletes a storage file record
func (r *StorageRepository) DeleteStorageFile(ctx context.Context, fileID string) error {
	query := `DELETE FROM storage_files WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete storage file: %w", err)
	}

	return nil
}

// RecordFileAccess records file access
func (r *StorageRepository) RecordFileAccess(ctx context.Context, req RecordFileAccessRequest) (string, error) {
	query := `
		SELECT record_file_access($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	var accessID string
	err := r.db.QueryRowContext(ctx, query,
		req.FileID,
		req.UserID,
		req.VendorID,
		req.AccessType,
		req.IPAddress,
		req.UserAgent,
		req.SessionID,
		req.SignedURL,
		req.Success,
		req.ErrorMessage,
		req.ResponseTimeMs,
		req.Metadata,
	).Scan(&accessID)

	if err != nil {
		return "", fmt.Errorf("failed to record file access: %w", err)
	}

	return accessID, nil
}

// CreateSignedURL creates a signed URL record
func (r *StorageRepository) CreateSignedURL(ctx context.Context, req CreateSignedURLRequest) (string, error) {
	query := `
		SELECT create_signed_url($1, $2, $3, $4, $5, $6, $7)
	`

	var urlID string
	err := r.db.QueryRowContext(ctx, query,
		req.FileID,
		req.SignedURL,
		req.AccessType,
		req.ExpiresAt,
		req.CreatedBy,
		req.MaxUsage,
		req.Metadata,
	).Scan(&urlID)

	if err != nil {
		return "", fmt.Errorf("failed to create signed URL: %w", err)
	}

	return urlID, nil
}

// GetStorageQuotaStatus gets storage quota status
func (r *StorageRepository) GetStorageQuotaStatus(ctx context.Context, userID *string, vendorID *string) ([]QuotaStatus, error) {
	query := `SELECT * FROM get_storage_quota_status($1, $2)`

	rows, err := r.db.QueryContext(ctx, query, userID, vendorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage quota status: %w", err)
	}
	defer rows.Close()

	var quotas []QuotaStatus
	for rows.Next() {
		var quota QuotaStatus
		err := rows.Scan(
			&quota.QuotaType,
			&quota.CurrentFiles,
			&quota.MaxFiles,
			&quota.CurrentSize,
			&quota.MaxSize,
			&quota.UsagePercent,
			&quota.RemainingFiles,
			&quota.RemainingSize,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quota status: %w", err)
		}
		quotas = append(quotas, quota)
	}

	return quotas, nil
}

// GetStorageStats gets storage statistics
func (r *StorageRepository) GetStorageStats(ctx context.Context, userID *string, vendorID *string, storageType *string) (*StorageStats, error) {
	query := `SELECT * FROM get_storage_stats($1, $2, $3)`

	var stats StorageStats
	var createdAt, updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID, vendorID, storageType).Scan(
		&stats.TotalFiles,
		&stats.TotalSize,
		&stats.UserFiles,
		&stats.ClothFiles,
		&stats.ResultFiles,
		&createdAt, // public_files
		&createdAt, // private_files
		&createdAt, // backed_up_files
		&createdAt, // average_file_size
		&createdAt, // most_accessed_file
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	// Set timestamps
	if createdAt.Valid {
		// stats.OldestFile = &createdAt.Time
	}
	if updatedAt.Valid {
		// stats.NewestFile = &updatedAt.Time
	}

	return &stats, nil
}

// Request/Response types for repository operations

type CreateStorageFileRequest struct {
	FilePath    string                 `json:"filePath"`
	FileName    string                 `json:"fileName"`
	FileSize    int64                  `json:"fileSize"`
	MimeType    string                 `json:"mimeType"`
	Checksum    string                 `json:"checksum"`
	StorageType string                 `json:"storageType"`
	OwnerID     string                 `json:"ownerId"`
	OwnerType   string                 `json:"ownerType"`
	IsPublic    bool                   `json:"isPublic"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type UpdateStorageFileRequest struct {
	IsPublic *bool                  `json:"isPublic,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type RecordFileAccessRequest struct {
	FileID         string                 `json:"fileId"`
	UserID         *string                `json:"userId,omitempty"`
	VendorID       *string                `json:"vendorId,omitempty"`
	AccessType     string                 `json:"accessType"`
	IPAddress      *string                `json:"ipAddress,omitempty"`
	UserAgent      *string                `json:"userAgent,omitempty"`
	SessionID      *string                `json:"sessionId,omitempty"`
	SignedURL      *string                `json:"signedUrl,omitempty"`
	Success        bool                   `json:"success"`
	ErrorMessage   *string                `json:"errorMessage,omitempty"`
	ResponseTimeMs *int                   `json:"responseTimeMs,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type CreateSignedURLRequest struct {
	FileID     string                 `json:"fileId"`
	SignedURL  string                 `json:"signedUrl"`
	AccessType string                 `json:"accessType"`
	ExpiresAt  time.Time              `json:"expiresAt"`
	CreatedBy  *string                `json:"createdBy,omitempty"`
	MaxUsage   int                    `json:"maxUsage"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type StorageFile struct {
	ID            string                 `json:"id"`
	FilePath      string                 `json:"filePath"`
	FileName      string                 `json:"fileName"`
	FileSize      int64                  `json:"fileSize"`
	MimeType      string                 `json:"mimeType"`
	Checksum      string                 `json:"checksum"`
	StorageType   string                 `json:"storageType"`
	OwnerID       string                 `json:"ownerId"`
	OwnerType     string                 `json:"ownerType"`
	IsPublic      bool                   `json:"isPublic"`
	IsBackedUp    bool                   `json:"isBackedUp"`
	BackupPath    *string                `json:"backupPath,omitempty"`
	ThumbnailPath *string                `json:"thumbnailPath,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	LastAccessed  *time.Time             `json:"lastAccessed,omitempty"`
	AccessCount   int                    `json:"accessCount"`
}

type QuotaStatus struct {
	QuotaType      string  `json:"quotaType"`
	CurrentFiles   int     `json:"currentFiles"`
	MaxFiles       int     `json:"maxFiles"`
	CurrentSize    int64   `json:"currentSize"`
	MaxSize        int64   `json:"maxSize"`
	UsagePercent   float64 `json:"usagePercent"`
	RemainingFiles int     `json:"remainingFiles"`
	RemainingSize  int64   `json:"remainingSize"`
}
