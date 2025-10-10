package payment

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for payment operations
type Handler struct {
	service *Service
}

// NewHandler creates a new payment handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "payment",
	})
}
