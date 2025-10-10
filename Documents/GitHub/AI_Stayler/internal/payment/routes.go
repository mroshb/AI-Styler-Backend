package payment

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up payment routes
func SetupRoutes(router *gin.RouterGroup, handler *Handler) {
	// Payment routes (require authentication)
	payments := router.Group("/payments")
	{
		payments.POST("/create", handler.CreatePayment)
		payments.GET("/:id/status", handler.GetPaymentStatus)
		payments.GET("/history", handler.GetPaymentHistory)
		payments.DELETE("/:id/cancel", handler.CancelPayment)
	}

	// Plan routes (public for viewing, auth for user-specific)
	plans := router.Group("/plans")
	{
		plans.GET("/", handler.GetPlans)
		plans.GET("/active", handler.GetUserActivePlan) // requires auth
	}

	// Webhook routes (public, no auth required)
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/notify", handler.WebhookHandler)
	}

	// Health check
	router.GET("/health", handler.HealthCheck)
}
