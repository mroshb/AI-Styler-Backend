package user

import (
	"net/http"

	"ai-styler/internal/common"

	"github.com/gin-gonic/gin"
)

// MountRoutes mounts user service routes to the gin engine
func MountRoutes(r *gin.RouterGroup, handler *Handler) {
	// User profile routes (protected)
	userGroup := r.Group("/user")
	userGroup.Use(authenticateMiddleware())
	{
		// Profile management
		userGroup.GET("/profile", common.GinWrap(handler.GetProfile))
		userGroup.PUT("/profile", common.GinWrap(handler.UpdateProfile))

		// Conversion management
		userGroup.GET("/conversions", common.GinWrap(handler.GetConversionHistory))
		userGroup.POST("/conversions", common.GinWrap(handler.CreateConversion))
		userGroup.GET("/conversions/:id", common.GinWrap(handler.GetConversion))

		// Quota management
		userGroup.GET("/quota", common.GinWrap(handler.GetQuotaStatus))

		// Plan management
		userGroup.GET("/plan", common.GinWrap(handler.GetUserPlan))
		userGroup.POST("/plan", common.GinWrap(handler.CreateUserPlan))
		userGroup.PUT("/plan/:id", common.GinWrap(handler.UpdateUserPlan))
	}
}

// authenticateMiddleware provides authentication middleware for user routes
func authenticateMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the context (set by auth middleware)
		// This assumes the auth middleware has already validated the token
		// and set the user ID in the context
		userID := c.GetString("userID")
		if userID == "" {
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
		ctx := common.SetUserIDInContext(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// ginWrap - now using common package
