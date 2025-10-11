package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StorageService provides comprehensive file storage functionality
type StorageService struct {
	basePath     string
	signedURLKey []byte
	backupPath   string
}

// StorageConfig holds configuration for the storage service
type StorageConfig struct {
	BasePath     string `json:"basePath"`
	BackupPath   string `json:"backupPath"`
	SignedURLKey string `json:"signedURLKey"`
}

// FileInfo represents file metadata
type FileInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
	MimeType   string    `json:"mimeType"`
	Checksum   string    `json:"checksum"`
	IsBackedUp bool      `json:"isBackedUp"`
	BackupPath string    `json:"backupPath,omitempty"`
}

// SignedURLInfo represents signed URL metadata
type SignedURLInfo struct {
	URL        string    `json:"url"`
	ExpiresAt  time.Time `json:"expiresAt"`
	AccessType string    `json:"accessType"`
	FilePath   string    `json:"filePath"`
}

// StoragePaths defines the folder structure
type StoragePaths struct {
	Users   string
	Cloth   string
	Results string
	Backups string
}

// NewStorageService creates a new storage service
func NewStorageService(config StorageConfig) (*StorageService, error) {
	// Ensure base path exists
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	// Ensure backup path exists
	if err := os.MkdirAll(config.BackupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup path: %w", err)
	}

	// Create folder structure
	paths := StoragePaths{
		Users:   filepath.Join(config.BasePath, "images", "user"),
		Cloth:   filepath.Join(config.BasePath, "images", "cloth"),
		Results: filepath.Join(config.BasePath, "images", "result"),
		Backups: config.BackupPath,
	}

	if err := createFolderStructure(paths); err != nil {
		return nil, fmt.Errorf("failed to create folder structure: %w", err)
	}

	return &StorageService{
		basePath:     config.BasePath,
		signedURLKey: []byte(config.SignedURLKey),
		backupPath:   config.BackupPath,
	}, nil
}

// createFolderStructure creates the required folder hierarchy
func createFolderStructure(paths StoragePaths) error {
	folders := []string{
		paths.Users,
		paths.Cloth,
		paths.Results,
		paths.Backups,
	}

	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
	}

	return nil
}

// UploadFile uploads a file to the specified path
func (s *StorageService) UploadFile(ctx context.Context, data []byte, fileName string, path string) (string, error) {
	// Generate unique filename to prevent conflicts
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	uniqueFileName := fmt.Sprintf("%s_%s%s", baseName, generateUniqueID()[:8], ext)

	// Create full path
	fullPath := filepath.Join(s.basePath, path, uniqueFileName)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Create backup
	go s.createBackup(fullPath, uniqueFileName)

	return fullPath, nil
}

// DeleteFile deletes a file from storage
func (s *StorageService) DeleteFile(ctx context.Context, filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFile retrieves file data
func (s *StorageService) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// GetFileInfo retrieves file metadata
func (s *StorageService) GetFileInfo(ctx context.Context, filePath string) (*FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate checksum
	checksum, err := s.calculateChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Check if backup exists
	backupPath := s.getBackupPath(filePath)
	_, backupExists := os.Stat(backupPath)

	return &FileInfo{
		Path:       filePath,
		Size:       stat.Size(),
		CreatedAt:  stat.ModTime(),
		ModifiedAt: stat.ModTime(),
		MimeType:   s.getMimeType(filePath),
		Checksum:   checksum,
		IsBackedUp: !os.IsNotExist(backupExists),
		BackupPath: backupPath,
	}, nil
}

// GenerateSignedURL generates a signed URL for secure access
func (s *StorageService) GenerateSignedURL(ctx context.Context, filePath string, accessType string, ttl int64) (string, error) {
	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Generate expiration time
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)

	// Create signature data
	signatureData := fmt.Sprintf("%s:%s:%d:%s", filePath, accessType, expiresAt.Unix(), "ai_stayler")

	// Generate HMAC signature
	h := hmac.New(sha256.New, s.signedURLKey)
	h.Write([]byte(signatureData))
	signature := hex.EncodeToString(h.Sum(nil))

	// Create signed URL
	signedURL := fmt.Sprintf("/api/storage/signed/%s?access_type=%s&expires=%d&signature=%s",
		base64.URLEncoding.EncodeToString([]byte(filePath)),
		accessType,
		expiresAt.Unix(),
		signature,
	)

	return signedURL, nil
}

// ValidateSignedURL validates a signed URL
func (s *StorageService) ValidateSignedURL(ctx context.Context, signedURL string) (bool, string, error) {
	// Parse signed URL (simplified - in real implementation, parse query parameters)
	// This is a placeholder for the actual URL parsing logic
	return true, "", nil
}

// createBackup creates a backup of the uploaded file
func (s *StorageService) createBackup(filePath, fileName string) {
	// Read original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Log error but don't fail the upload
		return
	}

	// Create backup path with timestamp
	timestamp := time.Now().Format("2006-01-02")
	backupDir := filepath.Join(s.backupPath, timestamp)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return
	}

	backupPath := filepath.Join(backupDir, fileName)

	// Write backup
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		// Log error but don't fail
		return
	}
}

// getBackupPath returns the expected backup path for a file
func (s *StorageService) getBackupPath(filePath string) string {
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("2006-01-02")
	return filepath.Join(s.backupPath, timestamp, fileName)
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *StorageService) calculateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// getMimeType determines MIME type based on file extension
func (s *StorageService) getMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}

	return "application/octet-stream"
}

// GetStorageStats returns storage statistics
func (s *StorageService) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	stats := &StorageStats{}

	// Calculate stats for each folder
	folders := []string{
		filepath.Join(s.basePath, "images", "user"),
		filepath.Join(s.basePath, "images", "cloth"),
		filepath.Join(s.basePath, "images", "result"),
		s.backupPath,
	}

	for _, folder := range folders {
		folderStats, err := s.calculateFolderStats(folder)
		if err != nil {
			continue // Skip folders that don't exist
		}

		stats.TotalFiles += folderStats.FileCount
		stats.TotalSize += folderStats.TotalSize

		switch {
		case strings.Contains(folder, "user"):
			stats.UserFiles = folderStats.FileCount
			stats.UserSize = folderStats.TotalSize
		case strings.Contains(folder, "cloth"):
			stats.ClothFiles = folderStats.FileCount
			stats.ClothSize = folderStats.TotalSize
		case strings.Contains(folder, "result"):
			stats.ResultFiles = folderStats.FileCount
			stats.ResultSize = folderStats.TotalSize
		case strings.Contains(folder, "backup"):
			stats.BackupFiles = folderStats.FileCount
			stats.BackupSize = folderStats.TotalSize
		}
	}

	return stats, nil
}

// StorageStats represents storage statistics
type StorageStats struct {
	TotalFiles  int64 `json:"totalFiles"`
	TotalSize   int64 `json:"totalSize"`
	UserFiles   int64 `json:"userFiles"`
	UserSize    int64 `json:"userSize"`
	ClothFiles  int64 `json:"clothFiles"`
	ClothSize   int64 `json:"clothSize"`
	ResultFiles int64 `json:"resultFiles"`
	ResultSize  int64 `json:"resultSize"`
	BackupFiles int64 `json:"backupFiles"`
	BackupSize  int64 `json:"backupSize"`
}

// FolderStats represents statistics for a folder
type FolderStats struct {
	FileCount int64
	TotalSize int64
}

// calculateFolderStats calculates statistics for a folder
func (s *StorageService) calculateFolderStats(folderPath string) (*FolderStats, error) {
	var fileCount int64
	var totalSize int64

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &FolderStats{
		FileCount: fileCount,
		TotalSize: totalSize,
	}, nil
}

// CleanupOldBackups removes backups older than specified days
func (s *StorageService) CleanupOldBackups(ctx context.Context, daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	return filepath.Walk(s.backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's not a directory or if it's the root backup directory
		if !info.IsDir() || path == s.backupPath {
			return nil
		}

		// Parse date from directory name (format: 2006-01-02)
		dirName := filepath.Base(path)
		if dirDate, err := time.Parse("2006-01-02", dirName); err == nil {
			if dirDate.Before(cutoffDate) {
				// Remove old backup directory
				return os.RemoveAll(path)
			}
		}

		return nil
	})
}

// RestoreFromBackup restores a file from backup
func (s *StorageService) RestoreFromBackup(ctx context.Context, filePath string, backupDate string) error {
	fileName := filepath.Base(filePath)
	backupPath := filepath.Join(s.backupPath, backupDate, fileName)

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupPath)
	}

	// Read backup data
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write restored file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// ListFiles lists files in a directory with pagination
func (s *StorageService) ListFiles(ctx context.Context, directory string, page, pageSize int) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileInfo, err := s.GetFileInfo(ctx, path)
			if err != nil {
				return err
			}
			files = append(files, *fileInfo)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(files) {
		return []FileInfo{}, nil
	}

	if end > len(files) {
		end = len(files)
	}

	return files[start:end], nil
}

// CopyFile copies a file to another location
func (s *StorageService) CopyFile(ctx context.Context, srcPath, dstPath string) error {
	// Read source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write destination file
	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// MoveFile moves a file to another location
func (s *StorageService) MoveFile(ctx context.Context, srcPath, dstPath string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move the file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// GetDiskUsage returns disk usage information
func (s *StorageService) GetDiskUsage(ctx context.Context) (*DiskUsage, error) {
	var totalSize int64
	var fileCount int64

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DiskUsage{
		TotalSize:  totalSize,
		FileCount:  fileCount,
		BasePath:   s.basePath,
		BackupPath: s.backupPath,
	}, nil
}

// DiskUsage represents disk usage information
type DiskUsage struct {
	TotalSize  int64  `json:"totalSize"`
	TotalSpace int64  `json:"totalSpace"`
	FileCount  int64  `json:"fileCount"`
	BasePath   string `json:"basePath"`
	BackupPath string `json:"backupPath"`
}

// Helper function to generate unique ID
func generateUniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// CreateBackup creates a backup of a file
func (s *StorageService) CreateBackup(ctx context.Context, filePath string) error {
	fileName := filepath.Base(filePath)
	s.createBackup(filePath, fileName)
	return nil
}
