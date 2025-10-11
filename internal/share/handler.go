package share

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for share operations
type Handler struct {
	service *Service
}

// NewHandler creates a new share handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers share routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	share := router.Group("/share")
	{
		// Create shared link (requires authentication)
		share.POST("/create", h.CreateSharedLink)

		// Access shared link (public endpoint)
		share.GET("/:token", h.AccessSharedLink)

		// Deactivate shared link (requires authentication)
		share.DELETE("/:id", h.DeactivateSharedLink)

		// List user's shared links (requires authentication)
		share.GET("/", h.ListUserSharedLinks)

		// Get shared link statistics (requires authentication)
		share.GET("/stats", h.GetSharedLinkStats)

		// Cleanup expired links (admin endpoint)
		share.POST("/cleanup", h.CleanupExpiredLinks)
	}
}

// CreateSharedLink handles creating a new shared link
func (h *Handler) CreateSharedLink(c *gin.Context) {
	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Set default expiry if not provided
	if req.ExpiryMinutes == 0 {
		req.ExpiryMinutes = DefaultExpiryMinutes
	}

	response, err := h.service.CreateSharedLink(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// AccessSharedLink handles accessing a shared link
func (h *Handler) AccessSharedLink(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "share token is required"})
		return
	}

	// Get access type from query parameter
	accessType := c.DefaultQuery("type", AccessTypeView)
	if accessType != AccessTypeView && accessType != AccessTypeDownload {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid access type"})
		return
	}

	req := AccessShareRequest{
		ShareToken: token,
		AccessType: accessType,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Referer:    c.GetHeader("Referer"),
	}

	response, err := h.service.AccessSharedLink(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if !response.Success {
		c.JSON(http.StatusNotFound, response)
		return
	}

	// If successful, redirect to the result image URL
	if response.ResultImageURL != "" {
		c.Redirect(http.StatusFound, response.ResultImageURL)
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeactivateSharedLink handles deactivating a shared link
func (h *Handler) DeactivateSharedLink(c *gin.Context) {
	shareID := c.Param("id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "share ID is required"})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	err := h.service.DeactivateSharedLink(c.Request.Context(), shareID, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "shared link deactivated successfully"})
}

// ListUserSharedLinks handles listing user's shared links
func (h *Handler) ListUserSharedLinks(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	links, err := h.service.ListUserSharedLinks(c.Request.Context(), userID.(string), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shared links"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"links":  links,
		"limit":  limit,
		"offset": offset,
		"count":  len(links),
	})
}

// GetSharedLinkStats handles getting shared link statistics
func (h *Handler) GetSharedLinkStats(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get optional conversion ID filter
	conversionID := c.Query("conversion_id")

	stats, err := h.service.GetSharedLinkStats(c.Request.Context(), userID.(string), conversionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CleanupExpiredLinks handles cleanup of expired shared links
func (h *Handler) CleanupExpiredLinks(c *gin.Context) {
	// This endpoint should be protected by admin middleware
	// For now, we'll just check if user is authenticated
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	count, err := h.service.CleanupExpiredLinks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup expired links"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "cleanup completed",
		"count":   count,
	})
}
