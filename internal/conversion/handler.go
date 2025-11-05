package conversion

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ai-styler/internal/common"
)

// Handler provides HTTP handlers for conversion operations
type Handler struct {
	service *Service
}

// NewHandler creates a new conversion handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateConversion handles POST /convert
func (h *Handler) CreateConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	var req ConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body", nil)
		return
	}

	// Validate required fields using helper methods that support both formats
	userImageID := req.GetUserImageID()
	clothImageID := req.GetClothImageID()
	
	if userImageID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "userImageId or user_image_id is required", nil)
		return
	}
	if clothImageID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "clothImageId or cloth_image_id is required", nil)
		return
	}

	// Create a normalized request with the extracted values
	normalizedReq := ConversionRequest{
		UserImageID:  userImageID,
		ClothImageID: clothImageID,
		StyleName:    req.GetStyleName(),
	}

	conversion, err := h.service.CreateConversion(r.Context(), userID, normalizedReq)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("CreateConversion error: %v\n", err)
		
		if strings.Contains(err.Error(), "quota exceeded") {
			common.WriteError(w, http.StatusForbidden, "quota_exceeded", "You have exceeded your free conversion limit. Please upgrade your plan to continue.", map[string]interface{}{
				"remaining_free":   0,
				"upgrade_required": true,
				"upgrade_url":      "/plans",
			})
			return
		}
		if strings.Contains(err.Error(), "rate limit") {
			common.WriteError(w, http.StatusTooManyRequests, "rate_limit_exceeded", err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not accessible") {
			common.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to create conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusCreated, conversion)
}

// GetConversion handles GET /conversion/{id}
func (h *Handler) GetConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	conversionID := getPathParam(r, "id")
	if conversionID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "conversion ID is required", nil)
		return
	}

	conversion, err := h.service.GetConversion(r.Context(), conversionID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "conversion not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, conversion)
}

// ListConversions handles GET /conversions
func (h *Handler) ListConversions(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	// Parse query parameters
	req := ConversionListRequest{
		Page:     1,
		PageSize: DefaultPageSize,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= MaxPageSize {
			req.PageSize = pageSize
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = status
	}

	conversions, err := h.service.ListConversions(r.Context(), userID, req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to list conversions", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, conversions)
}

// UpdateConversion handles PUT /conversion/{id}
func (h *Handler) UpdateConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	conversionID := getPathParam(r, "id")
	if conversionID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "conversion ID is required", nil)
		return
	}

	var req UpdateConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body", nil)
		return
	}

	err := h.service.UpdateConversion(r.Context(), conversionID, userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "conversion not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to update conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "conversion updated successfully"})
}

// DeleteConversion handles DELETE /conversion/{id}
func (h *Handler) DeleteConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	conversionID := getPathParam(r, "id")
	if conversionID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "conversion ID is required", nil)
		return
	}

	err := h.service.DeleteConversion(r.Context(), conversionID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "conversion not found", nil)
			return
		}
		if strings.Contains(err.Error(), "cannot delete") {
			common.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to delete conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "conversion deleted successfully"})
}

// GetQuotaStatus handles GET /quota
func (h *Handler) GetQuotaStatus(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	quota, err := h.service.GetQuotaStatus(r.Context(), userID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get quota status", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, quota)
}

// CancelConversion handles POST /conversion/{id}/cancel
func (h *Handler) CancelConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	conversionID := getPathParam(r, "id")
	if conversionID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "conversion ID is required", nil)
		return
	}

	err := h.service.CancelConversion(r.Context(), conversionID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "conversion not found", nil)
			return
		}
		if strings.Contains(err.Error(), "cannot cancel") {
			common.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to cancel conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "conversion cancelled successfully"})
}

// GetProcessingStatus handles GET /conversion/{id}/status
func (h *Handler) GetProcessingStatus(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	conversionID := getPathParam(r, "id")
	if conversionID == "" {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "conversion ID is required", nil)
		return
	}

	status, err := h.service.GetProcessingStatus(r.Context(), conversionID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "conversion not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get processing status", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"status": status})
}

// GetConversionMetrics handles GET /conversions/metrics
func (h *Handler) GetConversionMetrics(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "30d" // Default to last 30 days
	}

	metrics, err := h.service.GetConversionMetrics(r.Context(), userID, timeRange)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get conversion metrics", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, metrics)
}

// Helper functions - now using common package

// getPathParam extracts a path parameter from the request
// First tries to get it from context (set by GinWrap), then falls back to parsing the URL
func getPathParam(r *http.Request, param string) string {
	// First, try to get from context (set by GinWrap)
	if val := r.Context().Value("path_param_" + param); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	
	// Fallback: parse from URL path
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// For routes like /api/conversion/:id or /api/conversion/:id/status
	// We look for "conversion" and return the segment immediately after it
	if param == "id" {
		for i, part := range parts {
			if part == "conversion" && i+1 < len(parts) {
				// Return the next segment (which is the ID)
				return parts[i+1]
			}
		}
	}
	
	// Fallback: look for the parameter name directly (for other routes)
	for i, part := range parts {
		if part == param && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}
