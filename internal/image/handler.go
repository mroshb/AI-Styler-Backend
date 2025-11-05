package image

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"ai-styler/internal/common"
)

// Handler provides HTTP handlers for image operations
type Handler struct {
	service *Service
}

// NewHandler creates a new image handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// UploadImage handles POST /images
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Get user/vendor context
	userID := common.GetUserIDFromContext(r.Context())
	vendorID := common.GetVendorIDFromContext(r.Context())

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "failed to parse multipart form", nil)
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "file is required", nil)
		return
	}
	defer file.Close()

	// Parse other form fields
	req := parseUploadImageRequest(r)

	// Set file info
	req.FileName = header.Filename
	req.FileSize = header.Size
	req.MimeType = header.Header.Get("Content-Type")

	// Override MIME type based on file extension if it's generic or empty
	if req.MimeType == "" || req.MimeType == "application/octet-stream" {
		req.MimeType = getMimeTypeFromExtension(header.Filename)
	}

	// Set file reader
	req.File = file

	image, err := h.service.UploadImage(r.Context(), &userID, &vendorID, req)
	if err != nil {
		if strings.Contains(err.Error(), "rate limit") {
			common.WriteError(w, http.StatusTooManyRequests, "rate_limit", err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "quota exceeded") {
			common.WriteError(w, http.StatusForbidden, "quota_exceeded", "You have exceeded your free gallery upload limit. Please upgrade your plan to continue.", map[string]interface{}{
				"remaining_free":   0,
				"upgrade_required": true,
				"upgrade_url":      "/plans",
			})
			return
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "too long") || strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "invalid") {
			common.WriteError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", fmt.Sprintf("failed to upload image: %v", err), nil)
		return
	}

	common.WriteJSON(w, http.StatusCreated, image)
}

// GetImage handles GET /images/:id
func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request) {
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

	common.WriteJSON(w, http.StatusOK, image)
}

// UpdateImage handles PUT /images/:id
func (h *Handler) UpdateImage(w http.ResponseWriter, r *http.Request) {
	imageID := getImageIDFromPath(r.URL.Path)
	if imageID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "image ID required", nil)
		return
	}

	var req UpdateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON", nil)
		return
	}

	image, err := h.service.UpdateImage(r.Context(), imageID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "image not found", nil)
			return
		}
		if strings.Contains(err.Error(), "too long") || strings.Contains(err.Error(), "too many") {
			common.WriteError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to update image", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, image)
}

// DeleteImage handles DELETE /images/:id
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	imageID := getImageIDFromPath(r.URL.Path)
	if imageID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "image ID required", nil)
		return
	}

	err := h.service.DeleteImage(r.Context(), imageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "image not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to delete image", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "Image deleted successfully"})
}

// ListImages handles GET /images
func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request) {
	// Get user/vendor context to filter by owner
	userID := common.GetUserIDFromContext(r.Context())
	vendorID := common.GetVendorIDFromContext(r.Context())

	req := parseImageListRequest(r)

	// If user/vendor is authenticated, filter by their images by default
	// (unless explicitly requested otherwise via query params)
	if userID != "" && req.UserID == nil {
		req.UserID = &userID
	}
	if vendorID != "" && req.VendorID == nil {
		req.VendorID = &vendorID
	}

	response, err := h.service.ListImages(r.Context(), req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to list images", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, response)
}

// GenerateSignedURL handles POST /images/:id/signed-url
func (h *Handler) GenerateSignedURL(w http.ResponseWriter, r *http.Request) {
	imageID := getImageIDFromPath(r.URL.Path)
	if imageID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "image ID required", nil)
		return
	}

	var req struct {
		AccessType string `json:"accessType"`
		ExpiresIn  *int   `json:"expiresIn,omitempty"` // Optional expiration in seconds
	}
	// Body is optional - if not provided, use defaults
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON", nil)
			return
		}
	}

	if req.AccessType == "" {
		req.AccessType = AccessTypeView
	}

	response, err := h.service.GenerateSignedURL(r.Context(), imageID, req.AccessType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "image not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to generate signed URL", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, response)
}

// GetImageUsageHistory handles GET /images/:id/usage
func (h *Handler) GetImageUsageHistory(w http.ResponseWriter, r *http.Request) {
	imageID := getImageIDFromPath(r.URL.Path)
	if imageID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "image ID required", nil)
		return
	}

	req := parseImageUsageHistoryRequest(r)
	response, err := h.service.GetImageUsageHistory(r.Context(), imageID, req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get usage history", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, response)
}

// GetQuotaStatus handles GET /quota
func (h *Handler) GetQuotaStatus(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	vendorID := common.GetVendorIDFromContext(r.Context())

	status, err := h.service.GetQuotaStatus(r.Context(), &userID, &vendorID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get quota status", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, status)
}

// GetImageStats handles GET /stats
func (h *Handler) GetImageStats(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	vendorID := common.GetVendorIDFromContext(r.Context())

	stats, err := h.service.GetImageStats(r.Context(), &userID, &vendorID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", fmt.Sprintf("failed to get image stats: %v", err), nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, stats)
}

// Helper functions

func parseUploadImageRequest(r *http.Request) UploadImageRequest {
	req := UploadImageRequest{
		IsPublic: false,
	}

	if typeStr := r.FormValue("type"); typeStr != "" {
		req.Type = ImageType(typeStr)
	}

	if isPublicStr := r.FormValue("isPublic"); isPublicStr != "" {
		if isPublic, err := strconv.ParseBool(isPublicStr); err == nil {
			req.IsPublic = isPublic
		}
	}

	if tagsStr := r.FormValue("tags"); tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
	}

	if widthStr := r.FormValue("width"); widthStr != "" {
		if width, err := strconv.Atoi(widthStr); err == nil {
			req.Width = &width
		}
	}

	if heightStr := r.FormValue("height"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			req.Height = &height
		}
	}

	if metadataStr := r.FormValue("metadata"); metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
			req.Metadata = metadata
		}
	}

	return req
}

func parseImageListRequest(r *http.Request) ImageListRequest {
	req := ImageListRequest{
		Page:     1,
		PageSize: 20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			req.PageSize = pageSize
		}
	}

	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		imageType := ImageType(typeStr)
		req.Type = &imageType
	}

	if isPublicStr := r.URL.Query().Get("isPublic"); isPublicStr != "" {
		if isPublic, err := strconv.ParseBool(isPublicStr); err == nil {
			req.IsPublic = &isPublic
		}
	}

	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
	}

	if userID := r.URL.Query().Get("userId"); userID != "" {
		req.UserID = &userID
	}

	if vendorID := r.URL.Query().Get("vendorId"); vendorID != "" {
		req.VendorID = &vendorID
	}

	return req
}

func parseImageUsageHistoryRequest(r *http.Request) ImageUsageHistoryRequest {
	req := ImageUsageHistoryRequest{
		Page:     1,
		PageSize: 20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			req.PageSize = pageSize
		}
	}

	if action := r.URL.Query().Get("action"); action != "" {
		req.Action = action
	}

	return req
}

func getImageIDFromPath(path string) string {
	// Extract image ID from path like /api/images/123 or /images/123 or /api/images/123/signed-url
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	// Look for "images" in the path
	for i, part := range parts {
		if part == "images" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func getMimeTypeFromExtension(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "image/jpeg" // Default fallback
	}
}

// JSON helpers - now using common package
