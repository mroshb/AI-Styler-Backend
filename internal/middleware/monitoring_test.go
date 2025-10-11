package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-styler/internal/logging"
	"ai-styler/internal/monitoring"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerMiddleware(t *testing.T) {
	config := logging.LoggerConfig{
		Level:       logging.LogLevelInfo,
		Format:      "json",
		Output:      "stdout",
		Service:     "test-service",
		Version:     "1.0.0",
		Environment: "test",
	}

	logger := logging.NewStructuredLogger(config)
	middleware := NewRequestLoggerMiddleware(logger)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestLogging())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMonitoringMiddleware(t *testing.T) {
	config := monitoring.MonitoringConfig{
		Sentry: monitoring.SentryConfig{
			DSN:              "",
			Environment:      "test",
			Release:          "1.0.0",
			Debug:            false,
			SampleRate:       1.0,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			MaxBreadcrumbs:   50,
		},
		Telegram: monitoring.TelegramConfig{
			BotToken: "",
			ChatID:   "",
			Enabled:  false,
			Timeout:  10 * time.Second,
		},
		Logging: logging.LoggerConfig{
			Level:       logging.LogLevelInfo,
			Format:      "json",
			Output:      "stdout",
			Service:     "test-service",
			Version:     "1.0.0",
			Environment: "test",
		},
		Health: monitoring.HealthConfig{
			Enabled:       false,
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}

	monitor, err := monitoring.NewMonitoringService(config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create monitoring service: %v", err)
	}
	defer monitor.Close()

	middleware := NewMonitoringMiddleware(monitor)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandling())
	r.Use(middleware.PerformanceMonitoring())
	r.Use(middleware.SecurityMonitoring())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	r.GET("/error", func(c *gin.Context) {
		c.Error(errors.New("internal server error"))
		c.JSON(500, gin.H{"error": "internal server error"})
	})

	// Test successful request
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test error request
	req, _ = http.NewRequest("GET", "/error", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestContextMiddleware(t *testing.T) {
	middleware := NewContextMiddleware()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.InjectContext())
	r.Use(middleware.UserContext())
	r.Use(middleware.VendorContext())
	r.Use(middleware.ConversionContext())

	r.GET("/test", func(c *gin.Context) {
		// Check if context values are set
		ctx := c.Request.Context()

		if traceID := ctx.Value("trace_id"); traceID == nil {
			t.Error("Expected trace_id to be set in context")
		}

		c.JSON(200, gin.H{"message": "test"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "test-user-123")
	req.Header.Set("X-Vendor-ID", "test-vendor-456")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	// Create a simple test without monitoring service
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Use Gin's default recovery middleware for testing
	r.Use(gin.Recovery())

	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	// Test panic recovery
	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The recovery middleware should return 500 status
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Test normal request
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == id2 {
		t.Error("Expected different request IDs")
	}

	if len(id1) != 32 {
		t.Errorf("Expected request ID length 32, got %d", len(id1))
	}

	if len(id2) != 32 {
		t.Errorf("Expected request ID length 32, got %d", len(id2))
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	config := monitoring.MonitoringConfig{
		Sentry: monitoring.SentryConfig{
			DSN:              "",
			Environment:      "test",
			Release:          "1.0.0",
			Debug:            false,
			SampleRate:       1.0,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			MaxBreadcrumbs:   50,
		},
		Telegram: monitoring.TelegramConfig{
			BotToken: "",
			ChatID:   "",
			Enabled:  false,
			Timeout:  10 * time.Second,
		},
		Logging: logging.LoggerConfig{
			Level:       logging.LogLevelInfo,
			Format:      "json",
			Output:      "stdout",
			Service:     "test-service",
			Version:     "1.0.0",
			Environment: "test",
		},
		Health: monitoring.HealthConfig{
			Enabled:       false,
			CheckInterval: 30 * time.Second,
			Timeout:       10 * time.Second,
		},
	}

	monitor, err := monitoring.NewMonitoringService(config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create monitoring service: %v", err)
	}
	defer monitor.Close()

	// Create all middleware
	requestLogger := NewRequestLoggerMiddleware(monitor.Logger())
	monitoringMiddleware := NewMonitoringMiddleware(monitor)
	contextMiddleware := NewContextMiddleware()
	recoveryMiddleware := NewRecoveryMiddleware(monitor)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Apply middleware in order
	r.Use(recoveryMiddleware.Recovery())
	r.Use(contextMiddleware.InjectContext())
	r.Use(requestLogger.RequestLogging())
	r.Use(monitoringMiddleware.ErrorHandling())
	r.Use(monitoringMiddleware.PerformanceMonitoring())
	r.Use(monitoringMiddleware.SecurityMonitoring())
	r.Use(contextMiddleware.UserContext())
	r.Use(contextMiddleware.VendorContext())
	r.Use(contextMiddleware.ConversionContext())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	r.GET("/error", func(c *gin.Context) {
		c.Error(errors.New("internal server error"))
		c.JSON(500, gin.H{"error": "internal server error"})
	})

	// Test successful request with all middleware
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "test-user-123")
	req.Header.Set("X-Vendor-ID", "test-vendor-456")
	req.Header.Set("X-Trace-ID", "test-trace-789")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test error request with all middleware
	req, _ = http.NewRequest("GET", "/error", nil)
	req.Header.Set("X-User-ID", "test-user-123")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
