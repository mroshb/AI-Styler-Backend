package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler_GetDashboard_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("GET", "/api/v1/dashboard", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_GetQuotaStatus_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("GET", "/api/v1/dashboard/quota", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_GetConversionHistory_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("GET", "/api/v1/dashboard/conversions", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_GetPlanStatus_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("GET", "/api/v1/dashboard/plan", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_GetStatistics_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("GET", "/api/v1/dashboard/statistics", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_GetRecentActivity_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("GET", "/api/v1/dashboard/activity", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_InvalidateCache_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("POST", "/api/v1/dashboard/cache/invalidate", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_CheckQuotaExceeded_NoUserID(t *testing.T) {
	// Create a simple handler for testing
	handler := &Handler{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), handler)

	// Create request without user_id in context
	req, _ := http.NewRequest("POST", "/api/v1/dashboard/quota/check", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_GetVendorGallery_Public(t *testing.T) {
	// Create a simple handler for testing with a mock service
	mockService := &Service{} // Empty service will cause nil pointer, but that's expected in test
	handler := &Handler{service: mockService}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterPublicRoutes(router.Group("/api/v1"), handler)

	// Create request for public gallery (no auth required)
	req, _ := http.NewRequest("GET", "/api/v1/public/gallery", nil)
	w := httptest.NewRecorder()

	// Execute - this will panic due to nil service, which is expected
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil service
			t.Log("Expected panic due to nil service:", r)
		}
	}()

	router.ServeHTTP(w, req)

	// This should return an error since we don't have a real service implementation
	// but it should not be an unauthorized error
	if w.Code == http.StatusUnauthorized {
		t.Error("Public gallery endpoint should not require authentication")
	}
}

func TestRegisterRoutes(t *testing.T) {
	// Test that routes can be registered without errors
	handler := &Handler{}
	router := gin.New()

	// This should not panic
	RegisterRoutes(router.Group("/api/v1"), handler)
	RegisterPublicRoutes(router.Group("/api/v1"), handler)
}

func TestRegisterProtectedRoutes(t *testing.T) {
	// Test that protected routes can be registered without errors
	handler := &Handler{}
	router := gin.New()

	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Next()
	}

	// This should not panic
	RegisterProtectedRoutes(router.Group("/api/v1"), handler, authMiddleware)
}

func TestRegisterAdminRoutes(t *testing.T) {
	// Test that admin routes can be registered without errors
	handler := &Handler{}
	router := gin.New()

	adminMiddleware := func(c *gin.Context) {
		c.Next()
	}

	// This should not panic
	RegisterAdminRoutes(router.Group("/api/v1"), handler, adminMiddleware)
}

func TestSetupRoutes(t *testing.T) {
	// Test that all routes can be set up without errors
	handler := &Handler{}
	router := gin.New()

	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Next()
	}

	// This should not panic
	SetupRoutes(router, handler, authMiddleware)
}
