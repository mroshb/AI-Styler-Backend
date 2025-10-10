package dashboard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles dashboard HTTP requests
type Handler struct {
	service *Service
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetDashboard retrieves comprehensive dashboard data
// @Summary Get dashboard data
// @Description Retrieves comprehensive dashboard data including quota status, conversion history, vendor gallery, and plan information
// @Tags dashboard
// @Accept json
// @Produce json
// @Param includeHistory query bool false "Include conversion history" default(true)
// @Param includeGallery query bool false "Include vendor gallery" default(true)
// @Param includeStatistics query bool false "Include statistics" default(true)
// @Param historyLimit query int false "Limit for conversion history" default(10)
// @Param galleryLimit query int false "Limit for gallery items" default(20)
// @Success 200 {object} DashboardData
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard [get]
// @Security BearerAuth
func (h *Handler) GetDashboard(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	// Parse query parameters
	req := DashboardRequest{
		IncludeHistory:    true,
		IncludeGallery:    true,
		IncludeStatistics: true,
		HistoryLimit:      DefaultHistoryLimit,
		GalleryLimit:      DefaultGalleryLimit,
	}

	if includeHistory := c.Query("includeHistory"); includeHistory != "" {
		if val, err := strconv.ParseBool(includeHistory); err == nil {
			req.IncludeHistory = val
		}
	}

	if includeGallery := c.Query("includeGallery"); includeGallery != "" {
		if val, err := strconv.ParseBool(includeGallery); err == nil {
			req.IncludeGallery = val
		}
	}

	if includeStatistics := c.Query("includeStatistics"); includeStatistics != "" {
		if val, err := strconv.ParseBool(includeStatistics); err == nil {
			req.IncludeStatistics = val
		}
	}

	if historyLimit := c.Query("historyLimit"); historyLimit != "" {
		if val, err := strconv.Atoi(historyLimit); err == nil && val > 0 {
			req.HistoryLimit = val
		}
	}

	if galleryLimit := c.Query("galleryLimit"); galleryLimit != "" {
		if val, err := strconv.Atoi(galleryLimit); err == nil && val > 0 {
			req.GalleryLimit = val
		}
	}

	// Validate limits
	if req.HistoryLimit > MaxHistoryLimit {
		req.HistoryLimit = MaxHistoryLimit
	}
	if req.GalleryLimit > MaxGalleryLimit {
		req.GalleryLimit = MaxGalleryLimit
	}

	dashboardData, err := h.service.GetDashboardData(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dashboard data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboardData)
}

// GetQuotaStatus retrieves current quota status
// @Summary Get quota status
// @Description Retrieves current quota status and upgrade recommendations
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} QuotaCheckResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/quota [get]
// @Security BearerAuth
func (h *Handler) GetQuotaStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	quotaResponse, err := h.service.CheckQuota(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check quota: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, quotaResponse)
}

// GetConversionHistory retrieves conversion history
// @Summary Get conversion history
// @Description Retrieves user's conversion history with pagination
// @Tags dashboard
// @Accept json
// @Produce json
// @Param limit query int false "Number of conversions to retrieve" default(10)
// @Success 200 {object} ConversionHistory
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/conversions [get]
// @Security BearerAuth
func (h *Handler) GetConversionHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	limit := DefaultHistoryLimit
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	if limit > MaxHistoryLimit {
		limit = MaxHistoryLimit
	}

	history, err := h.service.GetConversionHistory(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversion history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetVendorGallery retrieves vendor gallery information
// @Summary Get vendor gallery
// @Description Retrieves featured vendor albums and recent images
// @Tags dashboard
// @Accept json
// @Produce json
// @Param limit query int false "Number of gallery items to retrieve" default(20)
// @Success 200 {object} VendorGallery
// @Failure 400 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/gallery [get]
func (h *Handler) GetVendorGallery(c *gin.Context) {
	limit := DefaultGalleryLimit
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	if limit > MaxGalleryLimit {
		limit = MaxGalleryLimit
	}

	gallery, err := h.service.GetVendorGallery(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vendor gallery: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gallery)
}

// GetPlanStatus retrieves current plan status
// @Summary Get plan status
// @Description Retrieves current subscription plan information and available upgrades
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} PlanStatus
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/plan [get]
// @Security BearerAuth
func (h *Handler) GetPlanStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	planStatus, err := h.service.GetPlanStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get plan status: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, planStatus)
}

// GetStatistics retrieves dashboard statistics
// @Summary Get dashboard statistics
// @Description Retrieves user's conversion statistics and trends
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} DashboardStatistics
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/statistics [get]
// @Security BearerAuth
func (h *Handler) GetStatistics(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	statistics, err := h.service.GetDashboardStatistics(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get statistics: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, statistics)
}

// GetRecentActivity retrieves recent user activity
// @Summary Get recent activity
// @Description Retrieves recent user activity including conversions, logins, and plan changes
// @Tags dashboard
// @Accept json
// @Produce json
// @Param limit query int false "Number of activities to retrieve" default(10)
// @Success 200 {array} RecentActivity
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/activity [get]
// @Security BearerAuth
func (h *Handler) GetRecentActivity(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	if limit > 50 {
		limit = 50
	}

	activity, err := h.service.GetRecentActivity(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recent activity: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, activity)
}

// InvalidateCache invalidates dashboard cache
// @Summary Invalidate dashboard cache
// @Description Invalidates cached dashboard data for the current user
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/cache/invalidate [post]
// @Security BearerAuth
func (h *Handler) InvalidateCache(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	err := h.service.InvalidateCache(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to invalidate cache: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache invalidated successfully"})
}

// CheckQuotaExceeded checks if user has exceeded quota and returns appropriate response
// @Summary Check quota exceeded
// @Description Checks if user has exceeded their conversion quota
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} QuotaCheckResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 500 {object} common.ErrorResponse
// @Router /dashboard/quota/check [post]
// @Security BearerAuth
func (h *Handler) CheckQuotaExceeded(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	quotaResponse, err := h.service.CheckQuota(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check quota: " + err.Error()})
		return
	}

	// If quota exceeded, return 403 with upgrade information
	if !quotaResponse.CanConvert {
		c.JSON(http.StatusForbidden, gin.H{
			"error":              "quota_exceeded",
			"message":            "You have exceeded your conversion quota. Please upgrade your plan to continue.",
			"quota_status":       quotaResponse.QuotaStatus,
			"upgrade_prompt":     quotaResponse.UpgradePrompt,
			"recommended_action": quotaResponse.RecommendedAction,
			"upgrade_url":        "/plans",
		})
		return
	}

	c.JSON(http.StatusOK, quotaResponse)
}
