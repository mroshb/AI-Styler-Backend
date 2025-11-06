package payment

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up payment routes
func SetupRoutes(router *gin.RouterGroup, handler *Handler) {
	// Payment routes (require authentication)
	payments := router.Group("/payments")
	{
		// Generic payment routes
		payments.POST("/create", handler.CreatePayment)
		payments.GET("/:id/status", handler.GetPaymentStatus)
		payments.GET("/history", handler.GetPaymentHistory)
		payments.DELETE("/:id/cancel", handler.CancelPayment)

		// Zarinpal routes
		zarinpal := payments.Group("/zarinpal")
		{
			// Payment endpoints (require authentication)
			zarinpal.POST("/plan/:id", handler.PayPlanZarinpal)
		}

		// Callback route for Zarinpal (public, no auth required)
		payments.GET("/callback", handler.ZarinpalCallback)

		// BazaarPay routes
		bazaarpay := payments.Group("/bazaarpay")
		{
			// Payment endpoints (require authentication)
			bazaarpay.POST("/plan/:id", handler.PayPlanBazaarPay)
			// bazaarpay.POST("/order/:id", handler.OrderPayBazaarPay)
			// bazaarpay.POST("/shop/:id", handler.PayShopBazaarPay)

			// Status page and API (public)
			bazaarpay.GET("/status", handler.BazaarPayStatusPage)
			bazaarpay.POST("/check-status", handler.BazaarPayCheckStatus)
		}
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
