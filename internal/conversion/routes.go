package conversion

import (
	"net/http"

	"ai-styler/internal/common"

	"github.com/gin-gonic/gin"
)

// MountRoutes mounts conversion service routes to the gin engine
func MountRoutes(r *gin.RouterGroup, handler *Handler) {
	// Conversion routes (protected)
	conversionGroup := r.Group("/convert")
	conversionGroup.Use(authenticateMiddleware())
	{
		// Create conversion
		conversionGroup.POST("", common.GinWrap(handler.CreateConversion))

		// Get quota status
		conversionGroup.GET("/quota", common.GinWrap(handler.GetQuotaStatus))

		// Get conversion metrics
		conversionGroup.GET("/metrics", common.GinWrap(handler.GetConversionMetrics))
	}

	// Individual conversion routes (protected)
	conversionIDGroup := r.Group("/conversion")
	conversionIDGroup.Use(authenticateMiddleware())
	{
		// Get conversion by ID
		conversionIDGroup.GET("/:id", common.GinWrap(handler.GetConversion))

		// Update conversion
		conversionIDGroup.PUT("/:id", common.GinWrap(handler.UpdateConversion))

		// Delete conversion
		conversionIDGroup.DELETE("/:id", common.GinWrap(handler.DeleteConversion))

		// Cancel conversion
		conversionIDGroup.POST("/:id/cancel", common.GinWrap(handler.CancelConversion))

		// Get processing status
		conversionIDGroup.GET("/:id/status", common.GinWrap(handler.GetProcessingStatus))
	}

	// List conversions (protected)
	conversionsGroup := r.Group("/conversions")
	conversionsGroup.Use(authenticateMiddleware())
	{
		// List user's conversions
		conversionsGroup.GET("", common.GinWrap(handler.ListConversions))
	}
}

// authenticateMiddleware provides authentication middleware for conversion routes
func authenticateMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the context (set by auth middleware)
		// This assumes the auth middleware has already validated the token
		// and set the user ID in the context
		// Try both "user_id" (snake_case) and "userID" (camelCase) for compatibility
		userID, exists := c.Get("user_id")
		if !exists || userID == "" {
			// Fallback to camelCase
			userID = c.GetString("userID")
		}
		
		// Convert to string if it's not already
		userIDStr := ""
		if userID != nil {
			if str, ok := userID.(string); ok {
				userIDStr = str
			}
		}
		
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "unauthorized",
					"message": "user not authenticated",
				},
			})
			c.Abort()
			return
		}

		// Set user ID in context for handlers using proper context key
		ctx := common.SetUserIDInContext(c.Request.Context(), userIDStr)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// ginWrap - now using common package
