package admin

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up admin routes
func SetupRoutes(router *gin.RouterGroup, handler *Handler) {
	// Admin middleware - ensure only admin users can access these routes
	adminGroup := router.Group("/admin")
	adminGroup.Use(AdminAuthMiddleware())

	// User management routes
	users := adminGroup.Group("/users")
	{
		users.GET("", handler.GetUsers)                          // GET /admin/users
		users.GET("/:id", handler.GetUser)                       // GET /admin/users/:id
		users.PUT("/:id", handler.UpdateUser)                    // PUT /admin/users/:id
		users.DELETE("/:id", handler.DeleteUser)                 // DELETE /admin/users/:id
		users.POST("/:id/suspend", handler.SuspendUser)          // POST /admin/users/:id/suspend
		users.POST("/:id/activate", handler.ActivateUser)        // POST /admin/users/:id/activate
		users.POST("/:id/revoke-quota", handler.RevokeUserQuota) // POST /admin/users/:id/revoke-quota
		users.POST("/:id/revoke-plan", handler.RevokeUserPlan)   // POST /admin/users/:id/revoke-plan
	}

	// Vendor management routes
	vendors := adminGroup.Group("/vendors")
	{
		vendors.GET("", handler.GetVendors)                          // GET /admin/vendors
		vendors.GET("/:id", handler.GetVendor)                       // GET /admin/vendors/:id
		vendors.PUT("/:id", handler.UpdateVendor)                    // PUT /admin/vendors/:id
		vendors.DELETE("/:id", handler.DeleteVendor)                 // DELETE /admin/vendors/:id
		vendors.POST("/:id/suspend", handler.SuspendVendor)          // POST /admin/vendors/:id/suspend
		vendors.POST("/:id/activate", handler.ActivateVendor)        // POST /admin/vendors/:id/activate
		vendors.POST("/:id/verify", handler.VerifyVendor)            // POST /admin/vendors/:id/verify
		vendors.POST("/:id/revoke-quota", handler.RevokeVendorQuota) // POST /admin/vendors/:id/revoke-quota
	}

	// Plan management routes
	plans := adminGroup.Group("/plans")
	{
		plans.GET("", handler.GetPlans)          // GET /admin/plans
		plans.GET("/:id", handler.GetPlan)       // GET /admin/plans/:id
		plans.POST("", handler.CreatePlan)       // POST /admin/plans
		plans.PUT("/:id", handler.UpdatePlan)    // PUT /admin/plans/:id
		plans.DELETE("/:id", handler.DeletePlan) // DELETE /admin/plans/:id
	}

	// Payment management routes
	payments := adminGroup.Group("/payments")
	{
		payments.GET("", handler.GetPayments)    // GET /admin/payments
		payments.GET("/:id", handler.GetPayment) // GET /admin/payments/:id
	}

	// Conversion management routes
	conversions := adminGroup.Group("/conversions")
	{
		conversions.GET("", handler.GetConversions)    // GET /admin/conversions
		conversions.GET("/:id", handler.GetConversion) // GET /admin/conversions/:id
	}

	// Image management routes
	images := adminGroup.Group("/images")
	{
		images.GET("", handler.GetImages)    // GET /admin/images
		images.GET("/:id", handler.GetImage) // GET /admin/images/:id
	}

	// Audit trail routes
	auditLogs := adminGroup.Group("/audit-logs")
	{
		auditLogs.GET("", handler.GetAuditLogs) // GET /admin/audit-logs
	}

	// Statistics routes
	stats := adminGroup.Group("/stats")
	{
		stats.GET("", handler.GetSystemStats)                 // GET /admin/stats
		stats.GET("/users", handler.GetUserStats)             // GET /admin/stats/users
		stats.GET("/vendors", handler.GetVendorStats)         // GET /admin/stats/vendors
		stats.GET("/payments", handler.GetPaymentStats)       // GET /admin/stats/payments
		stats.GET("/conversions", handler.GetConversionStats) // GET /admin/stats/conversions
		stats.GET("/images", handler.GetImageStats)           // GET /admin/stats/images
	}
}

// AdminAuthMiddleware ensures only admin users can access admin routes
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Get user role from context (set by auth middleware)
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Check if user is admin
		if userRole != "admin" {
			c.JSON(403, gin.H{"error": "forbidden - admin access required"})
			c.Abort()
			return
		}

		// Add admin context
		c.Set("admin_user_id", userID)
		c.Next()
	}
}

// AdminRateLimitMiddleware applies rate limiting to admin routes
func AdminRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a real implementation, you would check rate limits here
		// For now, we'll just pass through
		c.Next()
	}
}

// AdminAuditMiddleware logs admin actions
func AdminAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the admin action
		// In a real implementation, you would log the action here
		c.Next()
	}
}
