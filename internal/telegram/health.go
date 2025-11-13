package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HealthServer provides health check endpoints
type HealthServer struct {
	db    *sql.DB
	redis *redis.Client
	port  int
}

// NewHealthServer creates a new health server
func NewHealthServer(db *sql.DB, redis *redis.Client, port int) *HealthServer {
	return &HealthServer{
		db:    db,
		redis: redis,
		port:  port,
	}
}

// Start starts the health server
func (hs *HealthServer) Start() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", hs.healthHandler)
	mux.HandleFunc("/health/ready", hs.readinessHandler)
	mux.HandleFunc("/health/live", hs.livenessHandler)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", hs.port),
		Handler: mux,
	}

	return server.ListenAndServe()
}

// healthHandler handles general health check
func (hs *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "telegram-bot",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// readinessHandler handles readiness check
func (hs *HealthServer) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ready := true
	checks := map[string]string{}

	// Check database
	if hs.db != nil {
		if err := hs.db.PingContext(ctx); err != nil {
			ready = false
			checks["database"] = "unhealthy: " + err.Error()
		} else {
			checks["database"] = "healthy"
		}
	} else {
		checks["database"] = "not configured"
	}

	// Check Redis
	if hs.redis != nil {
		if err := hs.redis.Ping(ctx).Err(); err != nil {
			ready = false
			checks["redis"] = "unhealthy: " + err.Error()
		} else {
			checks["redis"] = "healthy"
		}
	} else {
		checks["redis"] = "not configured"
	}

	status := map[string]interface{}{
		"ready":  ready,
		"checks": checks,
	}

	w.Header().Set("Content-Type", "application/json")
	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(status)
}

// livenessHandler handles liveness check
func (hs *HealthServer) livenessHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"alive":    true,
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

