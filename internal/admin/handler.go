package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for admin operations
type Handler struct {
	service AdminService
}

// NewHandler creates a new admin handler
func NewHandler(service AdminService) *Handler {
	return &Handler{service: service}
}

// User management handlers

// GetUsers handles GET /admin/users
func (h *Handler) GetUsers(c *gin.Context) {
	var req UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetUsers(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUser handles GET /admin/users/:id
func (h *Handler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles PUT /admin/users/:id
func (h *Handler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser handles DELETE /admin/users/:id
func (h *Handler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	err := h.service.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// SuspendUser handles POST /admin/users/:id/suspend
func (h *Handler) SuspendUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SuspendUser(c.Request.Context(), userID, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user suspended successfully"})
}

// ActivateUser handles POST /admin/users/:id/activate
func (h *Handler) ActivateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	err := h.service.ActivateUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user activated successfully"})
}

// Vendor management handlers

// GetVendors handles GET /admin/vendors
func (h *Handler) GetVendors(c *gin.Context) {
	var req VendorListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetVendors(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetVendor handles GET /admin/vendors/:id
func (h *Handler) GetVendor(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	vendor, err := h.service.GetVendor(c.Request.Context(), vendorID)
	if err != nil {
		if err.Error() == "vendor not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "vendor not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, vendor)
}

// UpdateVendor handles PUT /admin/vendors/:id
func (h *Handler) UpdateVendor(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	var req UpdateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vendor, err := h.service.UpdateVendor(c.Request.Context(), vendorID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, vendor)
}

// DeleteVendor handles DELETE /admin/vendors/:id
func (h *Handler) DeleteVendor(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	err := h.service.DeleteVendor(c.Request.Context(), vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vendor deleted successfully"})
}

// SuspendVendor handles POST /admin/vendors/:id/suspend
func (h *Handler) SuspendVendor(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SuspendVendor(c.Request.Context(), vendorID, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vendor suspended successfully"})
}

// ActivateVendor handles POST /admin/vendors/:id/activate
func (h *Handler) ActivateVendor(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	err := h.service.ActivateVendor(c.Request.Context(), vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vendor activated successfully"})
}

// VerifyVendor handles POST /admin/vendors/:id/verify
func (h *Handler) VerifyVendor(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	err := h.service.VerifyVendor(c.Request.Context(), vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vendor verified successfully"})
}

// Plan management handlers

// GetPlans handles GET /admin/plans
func (h *Handler) GetPlans(c *gin.Context) {
	var req PlanListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetPlans(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPlan handles GET /admin/plans/:id
func (h *Handler) GetPlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan ID is required"})
		return
	}

	plan, err := h.service.GetPlan(c.Request.Context(), planID)
	if err != nil {
		if err.Error() == "plan not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// CreatePlan handles POST /admin/plans
func (h *Handler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plan, err := h.service.CreatePlan(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, plan)
}

// UpdatePlan handles PUT /admin/plans/:id
func (h *Handler) UpdatePlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan ID is required"})
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plan, err := h.service.UpdatePlan(c.Request.Context(), planID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// DeletePlan handles DELETE /admin/plans/:id
func (h *Handler) DeletePlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan ID is required"})
		return
	}

	err := h.service.DeletePlan(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "plan deleted successfully"})
}

// Payment management handlers

// GetPayments handles GET /admin/payments
func (h *Handler) GetPayments(c *gin.Context) {
	var req PaymentListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetPayments(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPayment handles GET /admin/payments/:id
func (h *Handler) GetPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment ID is required"})
		return
	}

	payment, err := h.service.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		if err.Error() == "payment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// Conversion management handlers

// GetConversions handles GET /admin/conversions
func (h *Handler) GetConversions(c *gin.Context) {
	var req ConversionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetConversions(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetConversion handles GET /admin/conversions/:id
func (h *Handler) GetConversion(c *gin.Context) {
	conversionID := c.Param("id")
	if conversionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversion ID is required"})
		return
	}

	conversion, err := h.service.GetConversion(c.Request.Context(), conversionID)
	if err != nil {
		if err.Error() == "conversion not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversion not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conversion)
}

// Image management handlers

// GetImages handles GET /admin/images
func (h *Handler) GetImages(c *gin.Context) {
	var req ImageListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetImages(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetImage handles GET /admin/images/:id
func (h *Handler) GetImage(c *gin.Context) {
	imageID := c.Param("id")
	if imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image ID is required"})
		return
	}

	image, err := h.service.GetImage(c.Request.Context(), imageID)
	if err != nil {
		if err.Error() == "image not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, image)
}

// Audit trail handlers

// GetAuditLogs handles GET /admin/audit-logs
func (h *Handler) GetAuditLogs(c *gin.Context) {
	var req AuditLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.service.GetAuditLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Quota management handlers

// RevokeUserQuota handles POST /admin/users/:id/revoke-quota
func (h *Handler) RevokeUserQuota(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	var req RevokeQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Override user ID from URL
	req.UserID = userID

	err := h.service.RevokeUserQuota(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quota revoked successfully"})
}

// RevokeVendorQuota handles POST /admin/vendors/:id/revoke-quota
func (h *Handler) RevokeVendorQuota(c *gin.Context) {
	vendorID := c.Param("id")
	if vendorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor ID is required"})
		return
	}

	var req struct {
		QuotaType string `json:"quotaType" binding:"required,oneof=free paid"`
		Amount    int    `json:"amount" binding:"required,min=1"`
		Reason    string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.RevokeVendorQuota(c.Request.Context(), vendorID, req.QuotaType, req.Amount, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quota revoked successfully"})
}

// RevokeUserPlan handles POST /admin/users/:id/revoke-plan
func (h *Handler) RevokeUserPlan(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	var req RevokePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Override user ID from URL
	req.UserID = userID

	err := h.service.RevokeUserPlan(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "plan revoked successfully"})
}

// Statistics handlers

// GetSystemStats handles GET /admin/stats
func (h *Handler) GetSystemStats(c *gin.Context) {
	stats, err := h.service.GetSystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserStats handles GET /admin/stats/users
func (h *Handler) GetUserStats(c *gin.Context) {
	total, active, err := h.service.GetUserStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  total,
		"active": active,
	})
}

// GetVendorStats handles GET /admin/stats/vendors
func (h *Handler) GetVendorStats(c *gin.Context) {
	total, active, err := h.service.GetVendorStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  total,
		"active": active,
	})
}

// GetPaymentStats handles GET /admin/stats/payments
func (h *Handler) GetPaymentStats(c *gin.Context) {
	total, revenue, err := h.service.GetPaymentStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"revenue": revenue,
	})
}

// GetConversionStats handles GET /admin/stats/conversions
func (h *Handler) GetConversionStats(c *gin.Context) {
	total, pending, failed, err := h.service.GetConversionStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"pending": pending,
		"failed":  failed,
	})
}

// GetImageStats handles GET /admin/stats/images
func (h *Handler) GetImageStats(c *gin.Context) {
	total, err := h.service.GetImageStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
	})
}
