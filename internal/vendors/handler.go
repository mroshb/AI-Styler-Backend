package vendors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles vendor-related HTTP requests
type Handler struct {
	service Service
}

// NewHandler creates a new vendor handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers vendor routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	vendor := r.Group("/vendors")
	{
		vendor.GET("", h.GetVendors)
		vendor.GET("/:id", h.GetVendor)
		vendor.POST("", h.CreateVendor)
		vendor.PUT("/:id", h.UpdateVendor)
		vendor.DELETE("/:id", h.DeleteVendor)
	}
}

// GetVendors retrieves all vendors
func (h *Handler) GetVendors(c *gin.Context) {
	vendors, err := h.service.GetVendors(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vendors": vendors})
}

// GetVendor retrieves a specific vendor by ID
func (h *Handler) GetVendor(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	vendor, err := h.service.GetVendor(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vendor": vendor})
}

// CreateVendor creates a new vendor
func (h *Handler) CreateVendor(c *gin.Context) {
	var req CreateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vendor, err := h.service.CreateVendor(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"vendor": vendor})
}

// UpdateVendor updates an existing vendor
func (h *Handler) UpdateVendor(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	var req UpdateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vendor, err := h.service.UpdateVendor(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vendor": vendor})
}

// DeleteVendor deletes a vendor
func (h *Handler) DeleteVendor(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	err := h.service.DeleteVendor(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
