package notification

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up notification routes
func SetupRoutes(router *gin.RouterGroup, handler *Handler) {
	// Notification routes
	notifications := router.Group("/notifications")
	{
		notifications.POST("", handler.CreateNotification)        // POST /notifications
		notifications.GET("", handler.ListNotifications)          // GET /notifications
		notifications.GET("/:id", handler.GetNotification)        // GET /notifications/:id
		notifications.PUT("/:id/read", handler.MarkAsRead)        // PUT /notifications/:id/read
		notifications.DELETE("/:id", handler.DeleteNotification)  // DELETE /notifications/:id
		notifications.POST("/test", handler.SendTestNotification) // POST /notifications/test
	}

	// Notification preferences routes
	preferences := router.Group("/notifications/preferences")
	{
		preferences.GET("", handler.GetNotificationPreferences)    // GET /notifications/preferences
		preferences.PUT("", handler.UpdateNotificationPreferences) // PUT /notifications/preferences
	}

	// Statistics routes
	stats := router.Group("/notifications/stats")
	{
		stats.GET("", handler.GetNotificationStats) // GET /notifications/stats
	}

	// WebSocket routes
	ws := router.Group("/notifications/ws")
	{
		ws.GET("", handler.WebSocketHandler) // GET /notifications/ws
	}
}
