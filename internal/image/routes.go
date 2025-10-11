package image

import (
	"net/http"
	"strings"

	"ai-styler/internal/common"

	"github.com/gin-gonic/gin"
)

// SetupGinRoutes configures the image service routes for Gin
func SetupGinRoutes(router *gin.RouterGroup, handler *Handler) {
	// Image management routes
	images := router.Group("/images")
	{
		images.POST("", handler.UploadImageGin)                      // POST /images
		images.GET("", handler.ListImagesGin)                        // GET /images
		images.GET("/:id", handler.GetImageGin)                      // GET /images/:id
		images.PUT("/:id", handler.UpdateImageGin)                   // PUT /images/:id
		images.DELETE("/:id", handler.DeleteImageGin)                // DELETE /images/:id
		images.POST("/:id/signed-url", handler.GenerateSignedURLGin) // POST /images/:id/signed-url
		images.GET("/:id/usage", handler.GetImageUsageHistoryGin)    // GET /images/:id/usage
	}

	// Quota and statistics
	router.GET("/quota", handler.GetQuotaStatusGin) // GET /quota
	router.GET("/stats", handler.GetImageStatsGin)  // GET /stats
}

// Gin handler wrappers
func (h *Handler) UploadImageGin(c *gin.Context) {
	// Convert Gin context to http.Request/ResponseWriter
	// This is a simplified implementation
	c.JSON(200, gin.H{"message": "Upload image endpoint"})
}

func (h *Handler) ListImagesGin(c *gin.Context) {
	c.JSON(200, gin.H{"message": "List images endpoint"})
}

func (h *Handler) GetImageGin(c *gin.Context) {
	imageID := c.Param("id")
	c.JSON(200, gin.H{"message": "Get image endpoint", "id": imageID})
}

func (h *Handler) UpdateImageGin(c *gin.Context) {
	imageID := c.Param("id")
	c.JSON(200, gin.H{"message": "Update image endpoint", "id": imageID})
}

func (h *Handler) DeleteImageGin(c *gin.Context) {
	imageID := c.Param("id")
	c.JSON(200, gin.H{"message": "Delete image endpoint", "id": imageID})
}

func (h *Handler) GenerateSignedURLGin(c *gin.Context) {
	imageID := c.Param("id")
	c.JSON(200, gin.H{"message": "Generate signed URL endpoint", "id": imageID})
}

func (h *Handler) GetImageUsageHistoryGin(c *gin.Context) {
	imageID := c.Param("id")
	c.JSON(200, gin.H{"message": "Get image usage history endpoint", "id": imageID})
}

func (h *Handler) GetQuotaStatusGin(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get quota status endpoint"})
}

func (h *Handler) GetImageStatsGin(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get image stats endpoint"})
}

// SetupRoutes configures the image service routes
func SetupRoutes(handler *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Image management routes
	mux.HandleFunc("POST /images", handler.UploadImage)
	mux.HandleFunc("GET /images", handler.ListImages)
	mux.HandleFunc("GET /images/{id}", handler.GetImage)
	mux.HandleFunc("PUT /images/{id}", handler.UpdateImage)
	mux.HandleFunc("DELETE /images/{id}", handler.DeleteImage)

	// Signed URL generation
	mux.HandleFunc("POST /images/{id}/signed-url", handler.GenerateSignedURL)

	// Usage tracking
	mux.HandleFunc("GET /images/{id}/usage", handler.GetImageUsageHistory)

	// Quota and statistics
	mux.HandleFunc("GET /quota", handler.GetQuotaStatus)
	mux.HandleFunc("GET /stats", handler.GetImageStats)

	return mux
}

// SetupPublicRoutes configures public image routes (no authentication required)
func SetupPublicRoutes(handler *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Public image access (for public images only)
	mux.HandleFunc("GET /public/images/{id}", handler.GetPublicImage)
	mux.HandleFunc("GET /public/images", handler.ListPublicImages)

	return mux
}

// GetPublicImage handles GET /public/images/:id
func (h *Handler) GetPublicImage(w http.ResponseWriter, r *http.Request) {
	imageID := getImageIDFromPath(r.URL.Path)
	if imageID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "image ID required", nil)
		return
	}

	image, err := h.service.GetImage(r.Context(), imageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "image not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get image", nil)
		return
	}

	// Check if image is public
	if !image.IsPublic {
		common.WriteError(w, http.StatusForbidden, "forbidden", "image is not public", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, image)
}

// ListPublicImages handles GET /public/images
func (h *Handler) ListPublicImages(w http.ResponseWriter, r *http.Request) {
	req := parseImageListRequest(r)
	req.IsPublic = boolPtr(true) // Only public images

	response, err := h.service.ListImages(r.Context(), req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to list public images", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, response)
}
