package user

import (
	"encoding/json"
	"net/http"
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

// JSON helpers - now using common package
