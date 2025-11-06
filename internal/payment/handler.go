package payment

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for payment operations
type Handler struct {
	service        *Service
	bazaarPayService *BazaarPayService
}

// NewHandler creates a new payment handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// NewHandlerWithBazaarPay creates a new payment handler with BazaarPay service
func NewHandlerWithBazaarPay(service *Service, bazaarPayService *BazaarPayService) *Handler {
	return &Handler{
		service:        service,
		bazaarPayService: bazaarPayService,
	}
}

// CreatePayment handles payment creation
func (h *Handler) CreatePayment(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create payment
	resp, err := h.service.CreatePayment(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetPaymentStatus handles payment status retrieval
func (h *Handler) GetPaymentStatus(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment ID is required"})
		return
	}

	// Get payment status
	resp, err := h.service.GetPaymentStatus(c.Request.Context(), userID.(string), paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPaymentHistory handles payment history retrieval
func (h *Handler) GetPaymentHistory(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse query parameters
	var req PaymentHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get payment history
	resp, err := h.service.GetPaymentHistory(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPlans handles plans retrieval
func (h *Handler) GetPlans(c *gin.Context) {
	// Get plans
	plans, err := h.service.GetPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// GetUserActivePlan handles user active plan retrieval
func (h *Handler) GetUserActivePlan(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get user active plan
	plan, err := h.service.GetUserActivePlan(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plan": plan})
}

// CancelPayment handles payment cancellation
func (h *Handler) CancelPayment(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment ID is required"})
		return
	}

	// Cancel payment
	err := h.service.CancelPayment(c.Request.Context(), userID.(string), paymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment cancelled successfully"})
}

// WebhookHandler handles payment webhooks
func (h *Handler) WebhookHandler(c *gin.Context) {
	// Parse webhook data based on gateway
	gateway := c.Query("gateway")
	if gateway == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway parameter is required"})
		return
	}

	var webhook PaymentWebhook
	var err error

	switch gateway {
	case GatewayZarinpal:
		webhook, err = h.parseZarinpalWebhook(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported gateway"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process webhook
	err = h.service.VerifyPayment(c.Request.Context(), webhook)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}

// parseZarinpalWebhook parses Zarinpal webhook data
func (h *Handler) parseZarinpalWebhook(c *gin.Context) (PaymentWebhook, error) {
	var webhook PaymentWebhook

	// Zarinpal sends data as query parameters
	trackID := c.Query("trackId")
	if trackID == "" {
		return PaymentWebhook{}, errors.New("trackId parameter is required")
	}

	successStr := c.Query("success")
	success, err := strconv.ParseBool(successStr)
	if err != nil {
		success = false
	}

	statusStr := c.Query("status")
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		status = 0
	}

	orderID := c.Query("orderId")
	cardNumber := c.Query("cardNumber")

	webhook = PaymentWebhook{
		TrackID:    trackID,
		Success:    success,
		Status:     status,
		OrderID:    orderID,
		CardNumber: cardNumber,
	}

	return webhook, nil
}

// ZarinpalCallback handles Zarinpal payment callback
func (h *Handler) ZarinpalCallback(c *gin.Context) {
	// Parse callback data from query parameters
	trackID := c.Query("trackId")
	if trackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trackId parameter is required"})
		return
	}

	// Create webhook from callback data
	webhook := PaymentWebhook{
		TrackID: trackID,
		Success: c.Query("success") == "true",
		Status:  0, // Will be determined by verification
		OrderID: c.Query("orderId"),
		CardNumber: c.Query("cardNumber"),
	}

	// Verify and process payment
	err := h.service.VerifyPayment(c.Request.Context(), webhook)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Redirect to success page or return success response
	returnURL := c.Query("returnUrl")
	if returnURL != "" {
		c.Redirect(http.StatusFound, returnURL)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment verified successfully",
		"trackId": trackID,
	})
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "payment",
	})
}

// ================================================================
// Zarinpal Handlers
// ================================================================

// PayPlanZarinpal - ایجاد لینک پرداخت برای خرید پلن با زرین‌پال
func (h *Handler) PayPlanZarinpal(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		userID = c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}
	}

	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan id is required"})
		return
	}

	// Parse request body for optional return URL
	var req struct {
		ReturnURL   string `json:"returnUrl,omitempty"`
		Description string `json:"description,omitempty"`
	}
	_ = c.ShouldBindJSON(&req) // Ignore error, use defaults if not provided

	// Use default return URL if not provided
	if req.ReturnURL == "" {
		req.ReturnURL = "https://yourdomain.com/payment/success" // Default return URL
	}

	// Create payment request
	paymentReq := CreatePaymentRequest{
		PlanID:      planID,
		ReturnURL:   req.ReturnURL,
		Description: req.Description,
	}

	// Create payment using Zarinpal gateway
	resp, err := h.service.CreatePayment(c.Request.Context(), userID.(string), paymentReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"payment_id":  resp.PaymentID,
			"gateway_url": resp.GatewayURL,
			"track_id":    resp.TrackID,
			"expires_at":  resp.ExpiresAt,
		},
	})
}

// ================================================================
// BazaarPay Handlers
// ================================================================

// PayPlanBazaarPay - ایجاد لینک پرداخت برای خرید پلن
func (h *Handler) PayPlanBazaarPay(c *gin.Context) {
	if h.bazaarPayService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "BazaarPay service not initialized"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		userID = c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}
	}

	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan id is required"})
		return
	}

	// Get plan details using service
	plans, err := h.service.GetPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "failed to get plans"})
		return
	}

	var plan *PaymentPlan
	for i := range plans {
		if plans[i].ID == planID {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	// Get user phone (we'll need to get it from user service)
	// For now, we'll use empty string
	userPhone := ""

	// Create payment
	checkoutToken, paymentURL, err := h.bazaarPayService.PayForPlan(
		c.Request.Context(),
		userID.(string),
		planID,
		plan.DisplayName,
		plan.PricePerMonthCents,
		userPhone,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"checkout_token": checkoutToken,
			"payment_url":    paymentURL,
		},
	})
}

// BazaarPayStatusPage - صفحه HTML برای نمایش وضعیت
func (h *Handler) BazaarPayStatusPage(c *gin.Context) {
	c.HTML(http.StatusOK, "bazaarpay_status.html", nil)
}

// BazaarPayCheckStatus - API برای بررسی وضعیت (AJAX)
func (h *Handler) BazaarPayCheckStatus(c *gin.Context) {
	if h.bazaarPayService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "BazaarPay service not initialized"})
		return
	}

	var req struct {
		CheckoutToken string `json:"checkout_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "checkout_token is required",
		})
		return
	}

	// 1. بررسی وضعیت با Trace
	traceResp, err := h.bazaarPayService.TraceCheckout(req.CheckoutToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 2. اگر پرداخت موفق بود، پردازش کنیم
	if traceResp.Status == BazaarPayStatusPaidNotCommitted {
		if err := h.bazaarPayService.ProcessBazaarPayment(c.Request.Context(), req.CheckoutToken); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "payment processing failed: " + err.Error(),
			})
			return
		}
	}

	// 3. برگرداندن وضعیت
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":         traceResp.Status,
			"checkout_token": req.CheckoutToken,
			"is_paid":        traceResp.Status == BazaarPayStatusPaidCommitted || traceResp.Status == BazaarPayStatusPaidNotCommitted,
		},
	})
}
