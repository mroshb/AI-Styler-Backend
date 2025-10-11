package vendor

import (
	"github.com/gin-gonic/gin"
)

// MountRoutes registers all vendor routes
func MountRoutes(r *gin.RouterGroup, handler *Handler) {
	vendor := r.Group("/vendors")
	{
		vendor.GET("", handler.GetVendors)
		vendor.GET("/:id", handler.GetVendor)
		vendor.POST("", handler.CreateVendor)
		vendor.PUT("/:id", handler.UpdateVendor)
		vendor.DELETE("/:id", handler.DeleteVendor)
	}
}
