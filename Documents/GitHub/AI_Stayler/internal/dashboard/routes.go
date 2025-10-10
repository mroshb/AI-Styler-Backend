package dashboard

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers dashboard routes
func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	dashboard := router.Group("/dashboard")
	{
		// Main dashboard endpoint
		dashboard.GET("", handler.GetDashboard)

		// Quota management
		dashboard.GET("/quota", handler.GetQuotaStatus)
		dashboard.POST("/quota/check", handler.CheckQuotaExceeded)

		// Conversion history
		dashboard.GET("/conversions", handler.GetConversionHistory)

		// Vendor gallery (public)
		dashboard.GET("/gallery", handler.GetVendorGallery)

		// Plan management
		dashboard.GET("/plan", handler.GetPlanStatus)

		// Statistics
		dashboard.GET("/statistics", handler.GetStatistics)

		// Recent activity
		dashboard.GET("/activity", handler.GetRecentActivity)

		// Cache management
		dashboard.POST("/cache/invalidate", handler.InvalidateCache)
	}
}

// RegisterPublicRoutes registers public dashboard routes
func RegisterPublicRoutes(router *gin.RouterGroup, handler *Handler) {
	public := router.Group("/public")
	{
		// Public vendor gallery
		public.GET("/gallery", handler.GetVendorGallery)
	}
}

// RegisterProtectedRoutes registers protected dashboard routes with authentication
func RegisterProtectedRoutes(router *gin.RouterGroup, handler *Handler, authMiddleware gin.HandlerFunc) {
	protected := router.Group("/dashboard")
	protected.Use(authMiddleware)
	{
		// All dashboard endpoints require authentication
		protected.GET("", handler.GetDashboard)
		protected.GET("/quota", handler.GetQuotaStatus)
		protected.POST("/quota/check", handler.CheckQuotaExceeded)
		protected.GET("/conversions", handler.GetConversionHistory)
		protected.GET("/plan", handler.GetPlanStatus)
		protected.GET("/statistics", handler.GetStatistics)
		protected.GET("/activity", handler.GetRecentActivity)
		protected.POST("/cache/invalidate", handler.InvalidateCache)
	}
}

// RegisterAdminRoutes registers admin dashboard routes
func RegisterAdminRoutes(router *gin.RouterGroup, handler *Handler, adminMiddleware gin.HandlerFunc) {
	admin := router.Group("/admin/dashboard")
	admin.Use(adminMiddleware)
	{
		// Admin-specific dashboard endpoints can be added here
		// For example: system statistics, user analytics, etc.
	}
}

// SetupRoutes sets up all dashboard routes with proper middleware
func SetupRoutes(router *gin.Engine, handler *Handler, authMiddleware gin.HandlerFunc) {
	// Public routes (no authentication required)
	publicGroup := router.Group("/api/v1")
	RegisterPublicRoutes(publicGroup, handler)

	// Protected routes (authentication required)
	protectedGroup := router.Group("/api/v1")
	RegisterProtectedRoutes(protectedGroup, handler, authMiddleware)

	// Admin routes (admin authentication required)
	adminGroup := router.Group("/api/v1")
	adminMiddleware := func(c *gin.Context) {
		// Simple admin check - in real implementation, this would be more sophisticated
		c.Next()
	}
	RegisterAdminRoutes(adminGroup, handler, adminMiddleware)
}
