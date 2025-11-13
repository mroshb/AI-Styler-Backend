package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	// Check if wait parameter is present for long polling
	waitParam := r.URL.Query().Get("wait")
	if waitParam == "true" || waitParam == "1" {
		h.CreateConversionWithWait(w, r)
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

	// Validate that user image and cloth image are different
	if userImageID == clothImageID {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "user image and cloth image must be different", nil)
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
		if strings.Contains(err.Error(), "access denied") {
			common.WriteError(w, http.StatusForbidden, "access_denied", "You do not have permission to access one or more of the specified images", nil)
			return
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not accessible") || strings.Contains(err.Error(), "must be different") {
			common.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "chk_conversion_images") || strings.Contains(err.Error(), "violates check constraint") {
			common.WriteError(w, http.StatusBadRequest, "invalid_request", "user image and cloth image must be different", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to create conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusCreated, conversion)
}

// CreateConversionWithWait handles POST /convert?wait=true
// This endpoint creates a conversion and waits (long polling) until it's completed
func (h *Handler) CreateConversionWithWait(w http.ResponseWriter, r *http.Request) {
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

	// Validate required fields
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

	// Validate that user image and cloth image are different
	if userImageID == clothImageID {
		common.WriteError(w, http.StatusBadRequest, "invalid_request", "user image and cloth image must be different", nil)
		return
	}

	// Create a normalized request
	normalizedReq := ConversionRequest{
		UserImageID:  userImageID,
		ClothImageID: clothImageID,
		StyleName:    req.GetStyleName(),
	}

	// Create conversion
	conversion, err := h.service.CreateConversion(r.Context(), userID, normalizedReq)
	if err != nil {
		fmt.Printf("CreateConversionWithWait error: %v\n", err)
		
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
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not accessible") || strings.Contains(err.Error(), "must be different") {
			common.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to create conversion", nil)
		return
	}

	// Parse timeout and poll interval from query parameters
	// This must be done before setting headers to ensure proper error handling
	timeout := 5 * time.Minute // Default: 5 minutes
	if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
		if timeoutSeconds, err := strconv.Atoi(timeoutStr); err == nil && timeoutSeconds > 0 {
			timeout = time.Duration(timeoutSeconds) * time.Second
			// Maximum timeout: 30 minutes to prevent resource exhaustion
			if timeout > 30*time.Minute {
				timeout = 30 * time.Minute
			}
		}
	}

	pollInterval := 25 * time.Millisecond // Default: 25ms (very fast polling for immediate updates)
	if intervalStr := r.URL.Query().Get("poll_interval"); intervalStr != "" {
		if intervalMs, err := strconv.Atoi(intervalStr); err == nil && intervalMs > 0 {
			pollInterval = time.Duration(intervalMs) * time.Millisecond
			// Minimum 10ms, maximum 10 seconds
			if pollInterval < 10*time.Millisecond {
				pollInterval = 10 * time.Millisecond
			}
			if pollInterval > 10*time.Second {
				pollInterval = 10 * time.Second
			}
		}
	}

	// Set headers for long polling BEFORE any writes
	// This ensures headers are sent even if we return early
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering for long polling
	
	// If conversion is already completed or failed, return immediately
	if conversion.Status == ConversionStatusCompleted || conversion.Status == ConversionStatusFailed {
		common.WriteJSON(w, http.StatusOK, conversion)
		// Flush response to ensure it's sent immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// Give worker a tiny moment to process (especially if it's very fast)
	// But also check immediately before starting the watch loop
	time.Sleep(10 * time.Millisecond)
	
	// Quick check before starting watch loop - worker might have already finished
	quickCheck, err := h.service.GetConversion(ctx, conversion.ID, userID)
	if err == nil && (quickCheck.Status == ConversionStatusCompleted || quickCheck.Status == ConversionStatusFailed) {
		common.WriteJSON(w, http.StatusOK, quickCheck)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	// Start watching - WatchConversion will do immediate checks in a tight loop
	finalConversion, err := h.service.WatchConversion(ctx, conversion.ID, userID, timeout, pollInterval)
	if err != nil {
		// If context was cancelled due to timeout, return current status (should not happen due to improved error handling)
		if ctx.Err() == context.DeadlineExceeded {
			// finalConversion should contain the last known status
			if finalConversion.ID != "" {
				common.WriteJSON(w, http.StatusOK, finalConversion)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				return
			}
		}
		
		fmt.Printf("WatchConversion error: %v\n", err)
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "conversion not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to watch conversion", nil)
		return
	}

	// Return final conversion status
	common.WriteJSON(w, http.StatusOK, finalConversion)
	// Flush response to ensure it's sent immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
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
