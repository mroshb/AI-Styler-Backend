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
		images.GET("", common.GinWrap(handler.ListImages))           // GET /images
		images.GET("/:id", handler.GetImageGin)                      // GET /images/:id
		images.PUT("/:id", handler.UpdateImageGin)                   // PUT /images/:id
		images.DELETE("/:id", handler.DeleteImageGin)                // DELETE /images/:id
		images.POST("/:id/signed-url", handler.GenerateSignedURLGin) // POST /images/:id/signed-url
		images.GET("/:id/usage", handler.GetImageUsageHistoryGin)    // GET /images/:id/usage
	}

	// Quota and statistics
	router.GET("/quota", common.GinWrap(handler.GetQuotaStatus)) // GET /quota
	router.GET("/stats", common.GinWrap(handler.GetImageStats))  // GET /stats
}

// Gin handler wrappers for handlers that need path parameters

// UploadImageGin handles POST /images with multipart form
func (h *Handler) UploadImageGin(c *gin.Context) {
	// Gin automatically handles multipart form, so we can use it directly
	h.UploadImage(c.Writer, c.Request)
}

// GetImageGin handles GET /images/:id
func (h *Handler) GetImageGin(c *gin.Context) {
	// Extract path parameter and set it in request URL for handler
	imageID := c.Param("id")
	if imageID != "" {
		// Update the request path to include the ID so handler can extract it
		c.Request.URL.Path = "/api/images/" + imageID
	}
	h.GetImage(c.Writer, c.Request)
}

// UpdateImageGin handles PUT /images/:id
func (h *Handler) UpdateImageGin(c *gin.Context) {
	// Extract path parameter and set it in request URL
	imageID := c.Param("id")
	if imageID != "" {
		c.Request.URL.Path = "/api/images/" + imageID
	}
	h.UpdateImage(c.Writer, c.Request)
}

// DeleteImageGin handles DELETE /images/:id
func (h *Handler) DeleteImageGin(c *gin.Context) {
	// Extract path parameter and set it in request URL
	imageID := c.Param("id")
	if imageID != "" {
		c.Request.URL.Path = "/api/images/" + imageID
	}
	h.DeleteImage(c.Writer, c.Request)
}

// GenerateSignedURLGin handles POST /images/:id/signed-url
func (h *Handler) GenerateSignedURLGin(c *gin.Context) {
	// Extract path parameter and set it in request URL
	imageID := c.Param("id")
	if imageID != "" {
		c.Request.URL.Path = "/api/images/" + imageID + "/signed-url"
	}
	h.GenerateSignedURL(c.Writer, c.Request)
}

// GetImageUsageHistoryGin handles GET /images/:id/usage
func (h *Handler) GetImageUsageHistoryGin(c *gin.Context) {
	// Extract path parameter and set it in request URL
	imageID := c.Param("id")
	if imageID != "" {
		c.Request.URL.Path = "/api/images/" + imageID + "/usage"
	}
	h.GetImageUsageHistory(c.Writer, c.Request)
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
