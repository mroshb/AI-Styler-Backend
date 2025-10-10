package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Duration    time.Duration          `json:"duration"`
	LastChecked time.Time              `json:"last_checked"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    HealthStatus  `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Version   string        `json:"version"`
	Uptime    time.Duration `json:"uptime"`
	Checks    []HealthCheck `json:"checks"`
	Summary   HealthSummary `json:"summary"`
}

// HealthSummary provides a summary of health checks
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string     `json:"go_version"`
	OS           string     `json:"os"`
	Architecture string     `json:"architecture"`
	NumCPU       int        `json:"num_cpu"`
	NumGoroutine int        `json:"num_goroutine"`
	Memory       MemoryInfo `json:"memory"`
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// HealthMonitor provides health monitoring capabilities
type HealthMonitor struct {
	startTime   time.Time
	version     string
	environment string
	checks      map[string]HealthChecker
}

// HealthChecker interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) HealthCheck
}

// DatabaseHealthChecker checks database health
type DatabaseHealthChecker struct {
	db *sql.DB
}

// RedisHealthChecker checks Redis health
type RedisHealthChecker struct {
	client *redis.Client
}

// SystemHealthChecker checks system health
type SystemHealthChecker struct{}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(version string, environment string) *HealthMonitor {
	return &HealthMonitor{
		startTime:   time.Now(),
		version:     version,
		environment: environment,
		checks:      make(map[string]HealthChecker),
	}
}

// AddChecker adds a health checker
func (h *HealthMonitor) AddChecker(name string, checker HealthChecker) {
	h.checks[name] = checker
}

// GetHealth returns the overall health status
func (h *HealthMonitor) GetHealth(ctx context.Context) HealthResponse {
	checks := make([]HealthCheck, 0, len(h.checks))
	summary := HealthSummary{}

	for name, checker := range h.checks {
		check := checker.Check(ctx)
		check.Name = name
		checks = append(checks, check)

		summary.Total++
		switch check.Status {
		case HealthStatusHealthy:
			summary.Healthy++
		case HealthStatusDegraded:
			summary.Degraded++
		case HealthStatusUnhealthy:
			summary.Unhealthy++
		}
	}

	// Determine overall status
	var status HealthStatus
	if summary.Unhealthy > 0 {
		status = HealthStatusUnhealthy
	} else if summary.Degraded > 0 {
		status = HealthStatusDegraded
	} else {
		status = HealthStatusHealthy
	}

	return HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		Checks:    checks,
		Summary:   summary,
	}
}

// GetSystemInfo returns system information
func (h *HealthMonitor) GetSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		Memory: MemoryInfo{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
		},
	}
}

// Check performs a health check for database
func (d *DatabaseHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	if d.db == nil {
		return HealthCheck{
			Status:      HealthStatusUnhealthy,
			Message:     "Database connection not initialized",
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Set a timeout for the ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := d.db.PingContext(pingCtx)
	duration := time.Since(start)

	if err != nil {
		return HealthCheck{
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Database ping failed: %v", err),
			Duration:    duration,
			LastChecked: time.Now(),
		}
	}

	// Get connection stats
	stats := d.db.Stats()
	details := map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}

	return HealthCheck{
		Status:      HealthStatusHealthy,
		Message:     "Database connection healthy",
		Duration:    duration,
		LastChecked: time.Now(),
		Details:     details,
	}
}

// Check performs a health check for Redis
func (r *RedisHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	if r.client == nil {
		return HealthCheck{
			Status:      HealthStatusUnhealthy,
			Message:     "Redis client not initialized",
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Set a timeout for the ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.client.Ping(pingCtx).Err()
	duration := time.Since(start)

	if err != nil {
		return HealthCheck{
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Redis ping failed: %v", err),
			Duration:    duration,
			LastChecked: time.Now(),
		}
	}

	// Get Redis info
	info, err := r.client.Info(ctx).Result()
	var details map[string]interface{}
	if err == nil {
		details = map[string]interface{}{
			"info_length": len(info),
			"connected":   true,
		}
	} else {
		details = map[string]interface{}{
			"connected":  true,
			"info_error": err.Error(),
		}
	}

	return HealthCheck{
		Status:      HealthStatusHealthy,
		Message:     "Redis connection healthy",
		Duration:    duration,
		LastChecked: time.Now(),
		Details:     details,
	}
}

// Check performs a health check for system
func (s *SystemHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Check memory usage
	memUsagePercent := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	goroutineCount := runtime.NumGoroutine()

	var status HealthStatus
	var message string
	details := map[string]interface{}{
		"memory_usage_percent": memUsagePercent,
		"goroutine_count":      goroutineCount,
		"num_cpu":              runtime.NumCPU(),
		"go_version":           runtime.Version(),
	}

	if memUsagePercent > 90 {
		status = HealthStatusDegraded
		message = "High memory usage detected"
	} else if goroutineCount > 1000 {
		status = HealthStatusDegraded
		message = "High goroutine count detected"
	} else {
		status = HealthStatusHealthy
		message = "System resources healthy"
	}

	return HealthCheck{
		Status:      status,
		Message:     message,
		Duration:    time.Since(start),
		LastChecked: time.Now(),
		Details:     details,
	}
}

// HealthHandler handles health check endpoints
type HealthHandler struct {
	monitor *HealthMonitor
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(monitor *HealthMonitor) *HealthHandler {
	return &HealthHandler{
		monitor: monitor,
	}
}

// Health returns the health status
func (h *HealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	health := h.monitor.GetHealth(ctx)

	statusCode := http.StatusOK
	switch health.Status {
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still return 200 for degraded
	case HealthStatusHealthy:
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, health)
}

// Readiness returns the readiness status
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx := c.Request.Context()
	health := h.monitor.GetHealth(ctx)

	// For readiness, we only care about critical components
	ready := true
	for _, check := range health.Checks {
		if check.Status == HealthStatusUnhealthy {
			ready = false
			break
		}
	}

	if ready {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
	}
}

// Liveness returns the liveness status
func (h *HealthHandler) Liveness(c *gin.Context) {
	// Liveness check is simple - if the service is running, it's alive
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
		"uptime":    time.Since(h.monitor.startTime).String(),
	})
}

// SystemInfo returns system information
func (h *HealthHandler) SystemInfo(c *gin.Context) {
	info := h.monitor.GetSystemInfo()
	c.JSON(http.StatusOK, info)
}

// Metrics returns basic metrics
func (h *HealthHandler) Metrics(c *gin.Context) {
	ctx := c.Request.Context()
	health := h.monitor.GetHealth(ctx)
	systemInfo := h.monitor.GetSystemInfo()

	metrics := map[string]interface{}{
		"health":      health,
		"system":      systemInfo,
		"timestamp":   time.Now(),
		"environment": h.monitor.environment,
	}

	c.JSON(http.StatusOK, metrics)
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(r *gin.RouterGroup) {
	health := r.Group("/health")
	{
		health.GET("/", h.Health)
		health.GET("/ready", h.Readiness)
		health.GET("/live", h.Liveness)
		health.GET("/system", h.SystemInfo)
		health.GET("/metrics", h.Metrics)
	}
}
