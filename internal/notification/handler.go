package notification

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles notification HTTP requests
type Handler struct {
	service NotificationService
}

// NewHandler creates a new notification handler
func NewHandler(service NotificationService) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateNotification creates a new notification
func (h *Handler) CreateNotification(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, err := h.service.CreateNotification(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

// GetNotification gets a notification by ID
func (h *Handler) GetNotification(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification ID is required"})
		return
	}

	notification, err := h.service.GetNotification(c.Request.Context(), notificationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// ListNotifications lists notifications
func (h *Handler) ListNotifications(c *gin.Context) {
	var req NotificationListRequest

	// Parse query parameters
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			req.PageSize = pageSize
		}
	}
	if userID := c.Query("userId"); userID != "" {
		req.UserID = &userID
	}
	if notificationType := c.Query("type"); notificationType != "" {
		nt := NotificationType(notificationType)
		req.Type = &nt
	}
	if status := c.Query("status"); status != "" {
		ns := NotificationStatus(status)
		req.Status = &ns
	}
	if channel := c.Query("channel"); channel != "" {
		nc := NotificationChannel(channel)
		req.Channel = &nc
	}
	if priority := c.Query("priority"); priority != "" {
		np := NotificationPriority(priority)
		req.Priority = &np
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.ListNotifications(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// MarkAsRead marks a notification as read
func (h *Handler) MarkAsRead(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification ID is required"})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), notificationID, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

// DeleteNotification deletes a notification
func (h *Handler) DeleteNotification(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification ID is required"})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.DeleteNotification(c.Request.Context(), notificationID, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification deleted"})
}

// GetNotificationPreferences gets user notification preferences
func (h *Handler) GetNotificationPreferences(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	preferences, err := h.service.GetNotificationPreferences(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// UpdateNotificationPreferences updates user notification preferences
func (h *Handler) UpdateNotificationPreferences(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateNotificationPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateNotificationPreferences(c.Request.Context(), userIDStr, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
}

// GetNotificationStats gets notification statistics
func (h *Handler) GetNotificationStats(c *gin.Context) {
	timeRange := c.DefaultQuery("timeRange", "24h")

	stats, err := h.service.GetNotificationStats(c.Request.Context(), timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// WebSocketHandler handles WebSocket connections
func (h *Handler) WebSocketHandler(c *gin.Context) {
	// This would be implemented by the WebSocket provider
	// The actual WebSocket handling is done in the WebSocket provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "WebSocket handler not implemented"})
}

// SendTestNotification sends a test notification (for testing purposes)
func (h *Handler) SendTestNotification(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Create a test notification
	req := CreateNotificationRequest{
		UserID:  &userIDStr,
		Type:    NotificationTypeSystemMaintenance,
		Title:   "Test Notification",
		Message: "This is a test notification to verify the notification system is working.",
		Data: map[string]interface{}{
			"test": true,
		},
		Priority: PriorityNormal,
	}

	notification, err := h.service.CreateNotification(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "test notification sent",
		"notification": notification,
	})
}
