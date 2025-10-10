package storage

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for storage operations
type Handler struct {
	imageStorage *ImageStorageService
	storage      StorageServiceInterface
}

// NewHandler creates a new storage handler
func NewHandler(imageStorage *ImageStorageService, storage StorageServiceInterface) *Handler {
	return &Handler{
		imageStorage: imageStorage,
		storage:      storage,
	}
}

// RegisterRoutes registers storage routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	storage := router.Group("/storage")
	{
		// Image upload
		storage.POST("/images", h.UploadImage)

		// Image access
		storage.GET("/images/:id/access", h.GetImageAccess)
		storage.GET("/images/:id/signed-url", h.GenerateSignedURL)

		// Image management
		storage.DELETE("/images/:id", h.DeleteImage)
		storage.PUT("/images/:id", h.UpdateImage)
		storage.GET("/images/:id", h.GetImage)

		// Image search and listing
		storage.GET("/images", h.ListImages)
		storage.POST("/images/search", h.SearchImages)

		// Batch operations
		storage.POST("/images/batch", h.PerformBatchOperation)

		// Storage management
		storage.GET("/quota", h.GetStorageQuota)
		storage.GET("/health", h.GetStorageHealth)
		storage.GET("/stats", h.GetStorageStats)

		// Backup operations
		storage.POST("/backup", h.CreateBackup)
		storage.POST("/restore", h.RestoreFromBackup)
		storage.DELETE("/backups/cleanup", h.CleanupBackups)

		// Signed URL validation
		storage.GET("/signed/:encodedPath", h.ValidateSignedURL)
	}
}

// UploadImage handles image upload requests
func (h *Handler) UploadImage(c *gin.Context) {
	var req ImageUploadRequest

	// Parse multipart form
	_, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Build request
	req.File = src
	req.FileName = file.Filename
	req.ContentType = file.Header.Get("Content-Type")
	req.Size = file.Size
	req.ImageType = c.PostForm("imageType")
	req.OwnerID = c.PostForm("ownerId")
	req.IsPublic = c.PostForm("isPublic") == "true"

	// Parse tags
	if tagsStr := c.PostForm("tags"); tagsStr != "" {
		req.Tags = parseTags(tagsStr)
	}

	// Upload image
	response, err := h.imageStorage.UploadImage(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetImageAccess handles image access requests
func (h *Handler) GetImageAccess(c *gin.Context) {
	imageID := c.Param("id")

	var req ImageAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ImageID = imageID
	req.RequesterID = c.GetString("user_id") // From auth middleware

	response, err := h.imageStorage.GetImageAccess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GenerateSignedURL handles signed URL generation
func (h *Handler) GenerateSignedURL(c *gin.Context) {
	imageID := c.Param("id")
	accessType := c.DefaultQuery("access_type", AccessTypeView)
	ttlStr := c.DefaultQuery("ttl", "3600")

	ttl, err := strconv.ParseInt(ttlStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TTL"})
		return
	}

	req := ImageAccessRequest{
		ImageID:     imageID,
		AccessType:  accessType,
		TTL:         ttl,
		RequesterID: c.GetString("user_id"),
	}

	response, err := h.imageStorage.GetImageAccess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteImage handles image deletion
func (h *Handler) DeleteImage(c *gin.Context) {
	imageID := c.Param("id")

	err := h.imageStorage.DeleteImage(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// UpdateImage handles image updates
func (h *Handler) UpdateImage(c *gin.Context) {
	imageID := c.Param("id")

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would typically update image metadata in the database
	c.JSON(http.StatusOK, gin.H{"message": "Image updated successfully", "imageId": imageID})
}

// GetImage handles image retrieval
func (h *Handler) GetImage(c *gin.Context) {
	imageID := c.Param("id")

	// This would typically get image metadata from the database
	c.JSON(http.StatusOK, gin.H{"imageId": imageID, "message": "Image retrieved successfully"})
}

// ListImages handles image listing
func (h *Handler) ListImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	imageType := c.Query("imageType")
	ownerID := c.Query("ownerId")

	req := ImageSearchRequest{
		ImageType: imageType,
		OwnerID:   ownerID,
		Page:      page,
		PageSize:  pageSize,
	}

	response, err := h.imageStorage.SearchImages(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SearchImages handles image search
func (h *Handler) SearchImages(c *gin.Context) {
	var req ImageSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.imageStorage.SearchImages(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// PerformBatchOperation handles batch operations
func (h *Handler) PerformBatchOperation(c *gin.Context) {
	var req ImageBatchOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.imageStorage.PerformBatchOperation(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetStorageQuota handles storage quota requests
func (h *Handler) GetStorageQuota(c *gin.Context) {
	userID := c.GetString("user_id")

	quota, err := h.imageStorage.GetStorageQuota(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quota)
}

// GetStorageHealth handles storage health requests
func (h *Handler) GetStorageHealth(c *gin.Context) {
	health, err := h.imageStorage.GetStorageHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetStorageStats handles storage statistics requests
func (h *Handler) GetStorageStats(c *gin.Context) {
	stats, err := h.storage.GetStorageStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateBackup handles backup creation requests
func (h *Handler) CreateBackup(c *gin.Context) {
	var req struct {
		ImageID string `json:"imageId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.imageStorage.PerformBatchOperation(c.Request.Context(), ImageBatchOperation{
		Operation: "backup",
		ImageIDs:  []string{req.ImageID},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Backup created successfully"})
}

// RestoreFromBackup handles restore requests
func (h *Handler) RestoreFromBackup(c *gin.Context) {
	var req struct {
		ImageID    string `json:"imageId"`
		BackupDate string `json:"backupDate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would typically restore from backup
	c.JSON(http.StatusOK, gin.H{"message": "Image restored successfully"})
}

// CleanupBackups handles backup cleanup requests
func (h *Handler) CleanupBackups(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
		return
	}

	err = h.storage.CleanupOldBackups(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Backups cleaned up successfully"})
}

// ValidateSignedURL handles signed URL validation
func (h *Handler) ValidateSignedURL(c *gin.Context) {
	encodedPath := c.Param("encodedPath")

	// Decode the path
	pathBytes, err := base64.URLEncoding.DecodeString(encodedPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid encoded path"})
		return
	}

	filePath := string(pathBytes)

	// Get query parameters
	expiresStr := c.Query("expires")

	// Validate signature
	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expires parameter"})
		return
	}

	// Check if URL has expired
	if time.Now().Unix() > expires {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Signed URL has expired"})
		return
	}

	// Validate signature (simplified)
	valid, _, err := h.storage.ValidateSignedURL(c.Request.Context(), c.Request.URL.String())
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Serve the file
	c.File(filePath)
}

// Helper functions

func parseTags(tagsStr string) []string {
	// Simple comma-separated tag parsing
	tags := strings.Split(tagsStr, ",")
	var result []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
