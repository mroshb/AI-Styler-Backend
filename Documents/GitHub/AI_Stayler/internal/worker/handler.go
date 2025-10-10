package worker

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the worker service
type Handler struct {
	service WorkerService
}

// NewHandler creates a new worker handler
func NewHandler(service WorkerService) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers all worker routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	worker := r.Group("/worker")
	{
		// Worker management
		worker.POST("/start", h.StartWorker)
		worker.POST("/stop", h.StopWorker)
		worker.GET("/status", h.GetStatus)
		worker.GET("/health", h.GetHealth)

		// Job management
		worker.POST("/jobs", h.EnqueueJob)
		worker.GET("/jobs/:id", h.GetJob)
		worker.DELETE("/jobs/:id", h.CancelJob)
		worker.POST("/jobs/:id/process", h.ProcessJob)

		// Configuration
		worker.GET("/config", h.GetConfig)
		worker.PUT("/config", h.UpdateConfig)

		// Statistics and monitoring
		worker.GET("/stats", h.GetStats)
		worker.GET("/workers", h.GetWorkers)
	}
}

// StartWorker starts the worker service
func (h *Handler) StartWorker(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.service.Start(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start worker service",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Worker service started successfully",
		"timestamp": time.Now().Unix(),
	})
}

// StopWorker stops the worker service
func (h *Handler) StopWorker(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.service.Stop(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to stop worker service",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Worker service stopped successfully",
		"timestamp": time.Now().Unix(),
	})
}

// GetStatus returns the current status of the worker service
func (h *Handler) GetStatus(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.service.GetStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get worker status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetHealth returns the health status of the worker
func (h *Handler) GetHealth(c *gin.Context) {
	ctx := c.Request.Context()

	health, err := h.service.GetHealth(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get worker health",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

// EnqueueJobRequest represents the request to enqueue a job
type EnqueueJobRequest struct {
	Type         string     `json:"type" binding:"required"`
	ConversionID string     `json:"conversionId" binding:"required"`
	UserID       string     `json:"userId" binding:"required"`
	Priority     int        `json:"priority,omitempty"`
	Payload      JobPayload `json:"payload" binding:"required"`
}

// EnqueueJob enqueues a new job
func (h *Handler) EnqueueJob(c *gin.Context) {
	ctx := c.Request.Context()

	var req EnqueueJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set default priority if not provided
	if req.Priority == 0 {
		req.Priority = int(JobPriorityNormal)
	}

	if err := h.service.EnqueueJob(ctx, req.Type, req.ConversionID, req.UserID, req.Payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to enqueue job",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Job enqueued successfully",
		"timestamp": time.Now().Unix(),
	})
}

// GetJobRequest represents the request to get a job
type GetJobRequest struct {
	JobID string `uri:"id" binding:"required"`
}

// GetJob returns job details
func (h *Handler) GetJob(c *gin.Context) {
	var req GetJobRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid job ID",
			"details": err.Error(),
		})
		return
	}

	// This would need to be implemented in the service
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"jobId":   req.JobID,
		"message": "Job details not implemented yet",
	})
}

// CancelJobRequest represents the request to cancel a job
type CancelJobRequest struct {
	JobID string `uri:"id" binding:"required"`
}

// CancelJob cancels a job
func (h *Handler) CancelJob(c *gin.Context) {
	ctx := c.Request.Context()

	var req CancelJobRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid job ID",
			"details": err.Error(),
		})
		return
	}

	if err := h.service.CancelJob(ctx, req.JobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cancel job",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Job cancelled successfully",
		"timestamp": time.Now().Unix(),
	})
}

// ProcessJobRequest represents the request to process a job
type ProcessJobRequest struct {
	JobID string `uri:"id" binding:"required"`
}

// ProcessJob processes a specific job
func (h *Handler) ProcessJob(c *gin.Context) {
	var req ProcessJobRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid job ID",
			"details": err.Error(),
		})
		return
	}

	// This would need to be implemented in the service
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"jobId":   req.JobID,
		"message": "Job processing not implemented yet",
	})
}

// GetConfig returns the current configuration
func (h *Handler) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()

	config, err := h.service.GetConfig(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get configuration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfigRequest represents the request to update configuration
type UpdateConfigRequest struct {
	MaxWorkers        *int           `json:"maxWorkers,omitempty"`
	JobTimeout        *time.Duration `json:"jobTimeout,omitempty"`
	RetryDelay        *time.Duration `json:"retryDelay,omitempty"`
	MaxRetries        *int           `json:"maxRetries,omitempty"`
	PollInterval      *time.Duration `json:"pollInterval,omitempty"`
	CleanupInterval   *time.Duration `json:"cleanupInterval,omitempty"`
	HealthCheckPort   *int           `json:"healthCheckPort,omitempty"`
	EnableMetrics     *bool          `json:"enableMetrics,omitempty"`
	EnableHealthCheck *bool          `json:"enableHealthCheck,omitempty"`
}

// UpdateConfig updates the worker configuration
func (h *Handler) UpdateConfig(c *gin.Context) {
	ctx := c.Request.Context()

	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get current config
	currentConfig, err := h.service.GetConfig(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get current configuration",
			"details": err.Error(),
		})
		return
	}

	// Update config with provided values
	if req.MaxWorkers != nil {
		currentConfig.MaxWorkers = *req.MaxWorkers
	}
	if req.JobTimeout != nil {
		currentConfig.JobTimeout = *req.JobTimeout
	}
	if req.RetryDelay != nil {
		currentConfig.RetryDelay = *req.RetryDelay
	}
	if req.MaxRetries != nil {
		currentConfig.MaxRetries = *req.MaxRetries
	}
	if req.PollInterval != nil {
		currentConfig.PollInterval = *req.PollInterval
	}
	if req.CleanupInterval != nil {
		currentConfig.CleanupInterval = *req.CleanupInterval
	}
	if req.HealthCheckPort != nil {
		currentConfig.HealthCheckPort = *req.HealthCheckPort
	}
	if req.EnableMetrics != nil {
		currentConfig.EnableMetrics = *req.EnableMetrics
	}
	if req.EnableHealthCheck != nil {
		currentConfig.EnableHealthCheck = *req.EnableHealthCheck
	}

	// Apply the updated configuration
	if err := h.service.UpdateConfig(ctx, currentConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update configuration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Configuration updated successfully",
		"config":    currentConfig,
		"timestamp": time.Now().Unix(),
	})
}

// GetStats returns worker statistics
func (h *Handler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.service.GetStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetWorkers returns information about active workers
func (h *Handler) GetWorkers(c *gin.Context) {
	// This would need to be implemented in the service
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"workers": []gin.H{
			{
				"id":            "worker-1",
				"status":        "active",
				"jobsProcessed": 42,
				"uptime":        3600,
			},
		},
		"message": "Worker list not fully implemented yet",
	})
}

// HealthCheckHandler provides a simple health check endpoint
func (h *Handler) HealthCheckHandler(c *gin.Context) {
	ctx := c.Request.Context()

	health, err := h.service.GetHealth(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	status := "healthy"
	if health.Status != "healthy" {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        status,
		"workerId":      health.WorkerID,
		"uptime":        health.Uptime,
		"jobsProcessed": health.JobsProcessed,
		"timestamp":     time.Now().Unix(),
	})
}

// MetricsHandler provides metrics in Prometheus format
func (h *Handler) MetricsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.service.GetStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get metrics",
			"details": err.Error(),
		})
		return
	}

	// Generate Prometheus-style metrics
	metrics := []string{
		"# HELP worker_jobs_total Total number of jobs",
		"# TYPE worker_jobs_total counter",
		"worker_jobs_total " + strconv.FormatInt(stats.TotalJobs, 10),
		"",
		"# HELP worker_jobs_pending Number of pending jobs",
		"# TYPE worker_jobs_pending gauge",
		"worker_jobs_pending " + strconv.FormatInt(stats.PendingJobs, 10),
		"",
		"# HELP worker_jobs_processing Number of processing jobs",
		"# TYPE worker_jobs_processing gauge",
		"worker_jobs_processing " + strconv.FormatInt(stats.ProcessingJobs, 10),
		"",
		"# HELP worker_jobs_completed Number of completed jobs",
		"# TYPE worker_jobs_completed counter",
		"worker_jobs_completed " + strconv.FormatInt(stats.CompletedJobs, 10),
		"",
		"# HELP worker_jobs_failed Number of failed jobs",
		"# TYPE worker_jobs_failed counter",
		"worker_jobs_failed " + strconv.FormatInt(stats.FailedJobs, 10),
		"",
		"# HELP worker_active_workers Number of active workers",
		"# TYPE worker_active_workers gauge",
		"worker_active_workers " + strconv.Itoa(stats.ActiveWorkers),
		"",
		"# HELP worker_average_job_time Average job processing time in milliseconds",
		"# TYPE worker_average_job_time gauge",
		"worker_average_job_time " + strconv.FormatInt(stats.AverageJobTime, 10),
		"",
		"# HELP worker_success_rate Job success rate",
		"# TYPE worker_success_rate gauge",
		"worker_success_rate " + strconv.FormatFloat(stats.SuccessRate, 'f', 4, 64),
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%s\n", metrics)
}
