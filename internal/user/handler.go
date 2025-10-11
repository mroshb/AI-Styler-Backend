package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ai-styler/internal/common"
)

// Handler provides HTTP handlers for user operations
type Handler struct {
	service *Service
}

// NewHandler creates a new user handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetProfile handles GET /profile
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get profile", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, profile)
}

// UpdateProfile handles PUT /profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON", nil)
		return
	}

	profile, err := h.service.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "too long") {
			common.WriteError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to update profile", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, profile)
}

// GetConversionHistory handles GET /conversions
func (h *Handler) GetConversionHistory(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	// Parse query parameters
	req := ConversionHistoryRequest{
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

	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = status
	}

	if conversionType := r.URL.Query().Get("type"); conversionType != "" {
		req.Type = conversionType
	}

	history, err := h.service.GetConversionHistory(r.Context(), userID, req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get conversion history", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, history)
}

// CreateConversion handles POST /conversions
func (h *Handler) CreateConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	var req CreateConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON", nil)
		return
	}

	conversion, err := h.service.CreateConversion(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "quota exceeded") || strings.Contains(err.Error(), "rate limit") {
			common.WriteError(w, http.StatusTooManyRequests, "quota_exceeded", err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			common.WriteError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to create conversion", nil)
		return
	}

	common.WriteJSON(w, http.StatusCreated, conversion)
}

// GetConversion handles GET /conversions/:id
func (h *Handler) GetConversion(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	conversionID := getConversionIDFromPath(r.URL.Path)
	if conversionID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "conversion ID required", nil)
		return
	}

	conversion, err := h.service.GetConversion(r.Context(), userID, conversionID)
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

// GetQuotaStatus handles GET /quota
func (h *Handler) GetQuotaStatus(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	status, err := h.service.GetQuotaStatus(r.Context(), userID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get quota status", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, status)
}

// GetUserPlan handles GET /plan
func (h *Handler) GetUserPlan(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	plan, err := h.service.GetUserPlan(r.Context(), userID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to get user plan", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, plan)
}

// CreateUserPlan handles POST /plan
func (h *Handler) CreateUserPlan(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	var req struct {
		PlanName string `json:"planName" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON", nil)
		return
	}

	plan, err := h.service.CreateUserPlan(r.Context(), userID, req.PlanName)
	if err != nil {
		if strings.Contains(err.Error(), "invalid plan name") {
			common.WriteError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to create user plan", nil)
		return
	}

	common.WriteJSON(w, http.StatusCreated, plan)
}

// UpdateUserPlan handles PUT /plan/:id
func (h *Handler) UpdateUserPlan(w http.ResponseWriter, r *http.Request) {
	userID := common.GetUserIDFromContext(r.Context())
	if userID == "" {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not authenticated", nil)
		return
	}

	planID := getPlanIDFromPath(r.URL.Path)
	if planID == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "plan ID required", nil)
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON", nil)
		return
	}

	plan, err := h.service.UpdateUserPlan(r.Context(), planID, req.Status)
	if err != nil {
		if strings.Contains(err.Error(), "invalid plan status") {
			common.WriteError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			common.WriteError(w, http.StatusNotFound, "not_found", "plan not found", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "failed to update user plan", nil)
		return
	}

	common.WriteJSON(w, http.StatusOK, plan)
}

// Helper functions

func getConversionIDFromPath(path string) string {
	// Extract conversion ID from path like /conversions/123
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[1] == "conversions" {
		return parts[2]
	}
	return ""
}

func getPlanIDFromPath(path string) string {
	// Extract plan ID from path like /plan/123
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[1] == "plan" {
		return parts[2]
	}
	return ""
}

// JSON helpers - now using common package
