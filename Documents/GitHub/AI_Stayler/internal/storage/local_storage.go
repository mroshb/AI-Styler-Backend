package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LocalStorage implements file storage using local filesystem
type LocalStorage struct {
	basePath   string
	backupPath string
	logger     Logger
}

// Logger interface for logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath string, backupPath string, logger Logger) *LocalStorage {
	return &LocalStorage{
		basePath:   basePath,
		backupPath: backupPath,
		logger:     logger,
	}
}

// UploadFile uploads a file to local storage
func (s *LocalStorage) UploadFile(ctx context.Context, file *multipart.FileHeader, subPath string) (string, error) {
	// Create directory if it doesn't exist
	fullPath := filepath.Join(s.basePath, subPath)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate unique filename
	filename := s.generateUniqueFilename(file.Filename)
	filePath := filepath.Join(fullPath, filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	s.logger.Info("File uploaded successfully", "path", filePath, "size", file.Size)
	return filePath, nil
}

// GetFile retrieves a file from local storage
func (s *LocalStorage) GetFile(ctx context.Context, filePath string) (*os.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// DeleteFile deletes a file from local storage
func (s *LocalStorage) DeleteFile(ctx context.Context, filePath string) error {
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			s.logger.Debug("File already deleted", "path", filePath)
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Info("File deleted successfully", "path", filePath)
	return nil
}

// GenerateSignedURL generates a signed URL for file access
func (s *LocalStorage) GenerateSignedURL(ctx context.Context, filePath, accessType string, ttl time.Duration) (string, error) {
	// For local storage, we'll generate a simple signed URL
	// In production, you might want to use a more sophisticated approach

	expiresAt := time.Now().Add(ttl).Unix()

	// Create signature data
	signatureData := fmt.Sprintf("%s:%s:%d", filePath, accessType, expiresAt)

	// Generate HMAC signature (using a secret key from config)
	secretKey := "your-secret-key" // This should come from config
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(signatureData))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Build signed URL
	baseURL := "http://localhost:8080" // This should come from config
	signedURL := fmt.Sprintf("%s/api/storage/signed/%s?access_type=%s&expires=%d&signature=%s",
		baseURL, url.PathEscape(filePath), accessType, expiresAt, signature)

	return signedURL, nil
}

// ValidateSignedURL validates a signed URL
func (s *LocalStorage) ValidateSignedURL(ctx context.Context, signedURL string) (string, error) {
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return "", fmt.Errorf("invalid signed URL: %w", err)
	}

	// Extract parameters
	filePath := parsedURL.Path[strings.LastIndex(parsedURL.Path, "/")+1:]
	accessType := parsedURL.Query().Get("access_type")
	expiresStr := parsedURL.Query().Get("expires")
	signature := parsedURL.Query().Get("signature")

	if accessType == "" || expiresStr == "" || signature == "" {
		return "", fmt.Errorf("missing required parameters")
	}

	// Check expiration
	expires, err := time.Parse("", expiresStr)
	if err != nil {
		return "", fmt.Errorf("invalid expiration time: %w", err)
	}

	if time.Now().After(expires) {
		return "", fmt.Errorf("signed URL expired")
	}

	// Validate signature
	secretKey := "your-secret-key" // This should come from config
	signatureData := fmt.Sprintf("%s:%s:%s", filePath, accessType, expiresStr)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(signatureData))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if signature != expectedSignature {
		return "", fmt.Errorf("invalid signature")
	}

	return filePath, nil
}

// CreateThumbnail creates a thumbnail for an image
func (s *LocalStorage) CreateThumbnail(ctx context.Context, originalPath string, width, height int) (string, error) {
	// For now, we'll just copy the original file as thumbnail
	// In production, you'd use an image processing library like imaging or graphicsmagick

	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)
	name := strings.TrimSuffix(filepath.Base(originalPath), ext)
	thumbnailPath := filepath.Join(dir, fmt.Sprintf("%s_thumb_%dx%d%s", name, width, height, ext))

	// Copy original to thumbnail location
	src, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open original file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(thumbnailPath)
	if err != nil {
		return "", fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file for thumbnail: %w", err)
	}

	s.logger.Info("Thumbnail created", "original", originalPath, "thumbnail", thumbnailPath)
	return thumbnailPath, nil
}

// GetFileInfo returns information about a file
func (s *LocalStorage) GetFileInfo(ctx context.Context, filePath string) (*FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate checksum
	checksum, err := s.calculateChecksum(filePath)
	if err != nil {
		checksum = "" // Set empty checksum if calculation fails
	}

	// Get MIME type
	mimeType := s.getMimeType(filePath)

	// Check if backup exists
	backupPath := s.getBackupPath(filePath)
	_, backupExists := os.Stat(backupPath)

	return &FileInfo{
		Path:       filePath,
		Size:       stat.Size(),
		CreatedAt:  stat.ModTime(),
		ModifiedAt: stat.ModTime(),
		MimeType:   mimeType,
		Checksum:   checksum,
		IsBackedUp: !os.IsNotExist(backupExists),
		BackupPath: backupPath,
	}, nil
}

// ListFiles lists files in a directory
func (s *LocalStorage) ListFiles(ctx context.Context, directory string, page, pageSize int) ([]FileInfo, error) {
	fullPath := filepath.Join(s.basePath, directory)

	var files []FileInfo
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileInfo, err := s.GetFileInfo(ctx, path)
			if err != nil {
				s.logger.Error("Failed to get file info", "path", path, "error", err)
				return nil // Continue processing other files
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

// CleanupExpiredFiles removes expired files
func (s *LocalStorage) CleanupExpiredFiles(ctx context.Context, maxAge time.Duration) (int, error) {
	var count int

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && time.Since(info.ModTime()) > maxAge {
			if err := os.Remove(path); err != nil {
				s.logger.Error("Failed to remove expired file", "path", path, "error", err)
				return nil // Continue processing other files
			}
			count++
			s.logger.Debug("Removed expired file", "path", path)
		}

		return nil
	})

	if err != nil {
		return count, fmt.Errorf("failed to cleanup expired files: %w", err)
	}

	s.logger.Info("Cleanup completed", "removed_files", count)
	return count, nil
}

// generateUniqueFilename generates a unique filename
func (s *LocalStorage) generateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *LocalStorage) calculateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// getMimeType determines MIME type based on file extension
func (s *LocalStorage) getMimeType(filePath string) string {
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

// getBackupPath returns the expected backup path for a file
func (s *LocalStorage) getBackupPath(filePath string) string {
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("2006-01-02")
	return filepath.Join(s.backupPath, timestamp, fileName)
}

// RegisterRoutes registers storage routes with Gin router
func (s *LocalStorage) RegisterRoutes(r *gin.RouterGroup) {
	storageGroup := r.Group("/storage")
	{
		storageGroup.GET("/signed/*filepath", s.handleSignedURL)
		storageGroup.GET("/public/*filepath", s.handlePublicFile)
		storageGroup.POST("/upload", s.handleUpload)
		storageGroup.DELETE("/:filepath", s.handleDelete)
		storageGroup.GET("/info/:filepath", s.handleGetInfo)
		storageGroup.GET("/list", s.handleListFiles)
	}
}

// handleSignedURL handles signed URL requests
func (s *LocalStorage) handleSignedURL(c *gin.Context) {
	filePath := c.Param("filepath")

	// Validate signed URL
	validatedPath, err := s.ValidateSignedURL(c.Request.Context(), c.Request.URL.String())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired signed URL"})
		return
	}

	if validatedPath != filePath {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Path mismatch"})
		return
	}

	// Serve file
	fullPath := filepath.Join(s.basePath, filePath)
	file, err := s.GetFile(c.Request.Context(), fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	c.File(fullPath)
}

// handlePublicFile handles public file requests
func (s *LocalStorage) handlePublicFile(c *gin.Context) {
	filePath := c.Param("filepath")
	fullPath := filepath.Join(s.basePath, "public", filePath)

	file, err := s.GetFile(c.Request.Context(), fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	c.File(fullPath)
}

// handleUpload handles file upload requests
func (s *LocalStorage) handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	subPath := c.DefaultQuery("path", "uploads")
	uploadedPath, err := s.UploadFile(c.Request.Context(), file, subPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"path":    uploadedPath,
	})
}

// handleDelete handles file deletion requests
func (s *LocalStorage) handleDelete(c *gin.Context) {
	filePath := c.Param("filepath")
	fullPath := filepath.Join(s.basePath, filePath)

	err := s.DeleteFile(c.Request.Context(), fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// handleGetInfo handles file info requests
func (s *LocalStorage) handleGetInfo(c *gin.Context) {
	filePath := c.Param("filepath")
	fullPath := filepath.Join(s.basePath, filePath)

	info, err := s.GetFileInfo(c.Request.Context(), fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// handleListFiles handles file listing requests
func (s *LocalStorage) handleListFiles(c *gin.Context) {
	dirPath := c.DefaultQuery("path", "")
	page := 1
	pageSize := 50

	// Parse pagination parameters
	if p := c.Query("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if parsedPageSize, err := strconv.Atoi(ps); err == nil && parsedPageSize > 0 && parsedPageSize <= 100 {
			pageSize = parsedPageSize
		}
	}

	files, err := s.ListFiles(c.Request.Context(), dirPath, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files":    files,
		"page":     page,
		"pageSize": pageSize,
		"total":    len(files),
	})
}
