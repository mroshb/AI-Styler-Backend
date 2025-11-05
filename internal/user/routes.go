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
	}
}

// authenticateMiddleware provides authentication middleware for user routes
func authenticateMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the Go context (set by UserContext middleware)
		// UserContext middleware extracts user ID from Gin context and sets it in Go context
		userID := common.GetUserIDFromContext(c.Request.Context())
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

		c.Next()
	}
}

// ginWrap - now using common package
