package worker

import (
	"github.com/gin-gonic/gin"
)

// RegisterWorkerRoutes registers all worker-related routes
func RegisterWorkerRoutes(r *gin.RouterGroup, handler *Handler) {
	// Worker management routes
	worker := r.Group("/worker")
	{
		// Basic worker operations
		worker.POST("/start", handler.StartWorker)
		worker.POST("/stop", handler.StopWorker)
		worker.GET("/status", handler.GetStatus)
		worker.GET("/health", handler.GetHealth)

		// Job management
		worker.POST("/jobs", handler.EnqueueJob)
		worker.GET("/jobs/:id", handler.GetJob)
		worker.DELETE("/jobs/:id", handler.CancelJob)
		worker.POST("/jobs/:id/process", handler.ProcessJob)

		// Configuration management
		worker.GET("/config", handler.GetConfig)
		worker.PUT("/config", handler.UpdateConfig)

		// Monitoring and statistics
		worker.GET("/stats", handler.GetStats)
		worker.GET("/workers", handler.GetWorkers)
	}

	// Health check endpoint (separate from worker group)
	r.GET("/health", handler.HealthCheckHandler)

	// Metrics endpoint for Prometheus
	r.GET("/metrics", handler.MetricsHandler)
}

// RegisterWorkerRoutesWithAuth registers worker routes with authentication middleware
func RegisterWorkerRoutesWithAuth(r *gin.RouterGroup, handler *Handler, authMiddleware gin.HandlerFunc) {
	// Apply authentication to all worker routes
	worker := r.Group("/worker")
	worker.Use(authMiddleware)
	{
		// Worker management
		worker.POST("/start", handler.StartWorker)
		worker.POST("/stop", handler.StopWorker)
		worker.GET("/status", handler.GetStatus)
		worker.GET("/health", handler.GetHealth)

		// Job management
		worker.POST("/jobs", handler.EnqueueJob)
		worker.GET("/jobs/:id", handler.GetJob)
		worker.DELETE("/jobs/:id", handler.CancelJob)
		worker.POST("/jobs/:id/process", handler.ProcessJob)

		// Configuration management
		worker.GET("/config", handler.GetConfig)
		worker.PUT("/config", handler.UpdateConfig)

		// Monitoring and statistics
		worker.GET("/stats", handler.GetStats)
		worker.GET("/workers", handler.GetWorkers)
	}

	// Public health check endpoint (no auth required)
	r.GET("/health", handler.HealthCheckHandler)

	// Public metrics endpoint (no auth required)
	r.GET("/metrics", handler.MetricsHandler)
}

// RegisterWorkerRoutesWithRateLimit registers worker routes with rate limiting
func RegisterWorkerRoutesWithRateLimit(r *gin.RouterGroup, handler *Handler, rateLimitMiddleware gin.HandlerFunc) {
	// Apply rate limiting to all worker routes
	worker := r.Group("/worker")
	worker.Use(rateLimitMiddleware)
	{
		// Worker management
		worker.POST("/start", handler.StartWorker)
		worker.POST("/stop", handler.StopWorker)
		worker.GET("/status", handler.GetStatus)
		worker.GET("/health", handler.GetHealth)

		// Job management
		worker.POST("/jobs", handler.EnqueueJob)
		worker.GET("/jobs/:id", handler.GetJob)
		worker.DELETE("/jobs/:id", handler.CancelJob)
		worker.POST("/jobs/:id/process", handler.ProcessJob)

		// Configuration management
		worker.GET("/config", handler.GetConfig)
		worker.PUT("/config", handler.UpdateConfig)

		// Monitoring and statistics
		worker.GET("/stats", handler.GetStats)
		worker.GET("/workers", handler.GetWorkers)
	}

	// Public endpoints (no rate limiting)
	r.GET("/health", handler.HealthCheckHandler)
	r.GET("/metrics", handler.MetricsHandler)
}

// RegisterWorkerRoutesWithMiddleware registers worker routes with custom middleware
func RegisterWorkerRoutesWithMiddleware(r *gin.RouterGroup, handler *Handler, middlewares ...gin.HandlerFunc) {
	// Apply all provided middlewares to worker routes
	worker := r.Group("/worker")
	worker.Use(middlewares...)
	{
		// Worker management
		worker.POST("/start", handler.StartWorker)
		worker.POST("/stop", handler.StopWorker)
		worker.GET("/status", handler.GetStatus)
		worker.GET("/health", handler.GetHealth)

		// Job management
		worker.POST("/jobs", handler.EnqueueJob)
		worker.GET("/jobs/:id", handler.GetJob)
		worker.DELETE("/jobs/:id", handler.CancelJob)
		worker.POST("/jobs/:id/process", handler.ProcessJob)

		// Configuration management
		worker.GET("/config", handler.GetConfig)
		worker.PUT("/config", handler.UpdateConfig)

		// Monitoring and statistics
		worker.GET("/stats", handler.GetStats)
		worker.GET("/workers", handler.GetWorkers)
	}

	// Public endpoints (no middleware)
	r.GET("/health", handler.HealthCheckHandler)
	r.GET("/metrics", handler.MetricsHandler)
}

// GetWorkerRouteInfo returns information about available worker routes
func GetWorkerRouteInfo() map[string]interface{} {
	return map[string]interface{}{
		"worker_routes": map[string]interface{}{
			"POST /worker/start": map[string]string{
				"description":   "Start the worker service",
				"auth_required": "true",
			},
			"POST /worker/stop": map[string]string{
				"description":   "Stop the worker service",
				"auth_required": "true",
			},
			"GET /worker/status": map[string]string{
				"description":   "Get worker service status and statistics",
				"auth_required": "false",
			},
			"GET /worker/health": map[string]string{
				"description":   "Get worker health status",
				"auth_required": "false",
			},
			"POST /worker/jobs": map[string]string{
				"description":   "Enqueue a new job",
				"auth_required": "true",
			},
			"GET /worker/jobs/:id": map[string]string{
				"description":   "Get job details",
				"auth_required": "true",
			},
			"DELETE /worker/jobs/:id": map[string]string{
				"description":   "Cancel a job",
				"auth_required": "true",
			},
			"POST /worker/jobs/:id/process": map[string]string{
				"description":   "Process a specific job",
				"auth_required": "true",
			},
			"GET /worker/config": map[string]string{
				"description":   "Get worker configuration",
				"auth_required": "true",
			},
			"PUT /worker/config": map[string]string{
				"description":   "Update worker configuration",
				"auth_required": "true",
			},
			"GET /worker/stats": map[string]string{
				"description":   "Get worker statistics",
				"auth_required": "false",
			},
			"GET /worker/workers": map[string]string{
				"description":   "Get list of active workers",
				"auth_required": "false",
			},
		},
		"public_routes": map[string]interface{}{
			"GET /health": map[string]string{
				"description":   "Health check endpoint",
				"auth_required": "false",
			},
			"GET /metrics": map[string]string{
				"description":   "Prometheus metrics endpoint",
				"auth_required": "false",
			},
		},
	}
}
