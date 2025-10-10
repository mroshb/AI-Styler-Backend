package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestHandler_GetUsers(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test users
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	router := setupTestRouter()
	router.GET("/admin/users", handler.GetUsers)

	req, _ := http.NewRequest("GET", "/admin/users?page=1&pageSize=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response UserListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if response.Total != 1 {
		t.Fatalf("Expected total 1, got %d", response.Total)
	}
	if len(response.Users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(response.Users))
	}
}

func TestHandler_GetUser(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	router := setupTestRouter()
	router.GET("/admin/users/:id", handler.GetUser)

	req, _ := http.NewRequest("GET", "/admin/users/user1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var user AdminUser
	err := json.Unmarshal(w.Body.Bytes(), &user)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user.ID != "user1" {
		t.Fatalf("Expected user ID user1, got %s", user.ID)
	}
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	router := setupTestRouter()
	router.GET("/admin/users/:id", handler.GetUser)

	req, _ := http.NewRequest("GET", "/admin/users/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", w.Code)
	}
}

func TestHandler_UpdateUser(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	router := setupTestRouter()
	router.PUT("/admin/users/:id", handler.UpdateUser)

	updateReq := UpdateUserRequest{
		Name: stringPtr("Updated Name"),
		Role: stringPtr("admin"),
	}

	jsonData, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/admin/users/user1", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var user AdminUser
	err := json.Unmarshal(w.Body.Bytes(), &user)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user.Name == nil || *user.Name != "Updated Name" {
		t.Fatalf("Expected name 'Updated Name', got %v", user.Name)
	}
	if user.Role != "admin" {
		t.Fatalf("Expected role 'admin', got %s", user.Role)
	}
}

func TestHandler_UpdateUser_InvalidRole(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	router := setupTestRouter()
	router.PUT("/admin/users/:id", handler.UpdateUser)

	updateReq := UpdateUserRequest{
		Role: stringPtr("invalid_role"),
	}

	jsonData, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/admin/users/user1", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", w.Code)
	}
}

func TestHandler_DeleteUser(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	router := setupTestRouter()
	router.DELETE("/admin/users/:id", handler.DeleteUser)

	req, _ := http.NewRequest("DELETE", "/admin/users/user1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify user is deleted
	_, exists := store.users["user1"]
	if exists {
		t.Fatal("Expected user to be deleted")
	}
}

func TestHandler_SuspendUser(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:       "user1",
		Phone:    "1234567890",
		Role:     "user",
		IsActive: true,
	}

	router := setupTestRouter()
	router.POST("/admin/users/:id/suspend", handler.SuspendUser)

	suspendReq := struct {
		Reason string `json:"reason"`
	}{
		Reason: "Test suspension",
	}

	jsonData, _ := json.Marshal(suspendReq)
	req, _ := http.NewRequest("POST", "/admin/users/user1/suspend", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify user is suspended
	user := store.users["user1"]
	if user.IsActive {
		t.Fatal("Expected user to be suspended")
	}
}

func TestHandler_ActivateUser(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:       "user1",
		Phone:    "1234567890",
		Role:     "user",
		IsActive: false,
	}

	router := setupTestRouter()
	router.POST("/admin/users/:id/activate", handler.ActivateUser)

	req, _ := http.NewRequest("POST", "/admin/users/user1/activate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify user is activated
	user := store.users["user1"]
	if !user.IsActive {
		t.Fatal("Expected user to be activated")
	}
}

func TestHandler_CreatePlan(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	router := setupTestRouter()
	router.POST("/admin/plans", handler.CreatePlan)

	createReq := CreatePlanRequest{
		Name:                    "test-plan",
		DisplayName:             "Test Plan",
		Description:             "A test plan",
		PricePerMonthCents:      1000,
		MonthlyConversionsLimit: 10,
		Features:                []string{"feature1", "feature2"},
		IsActive:                true,
	}

	jsonData, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/admin/plans", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", w.Code)
	}

	var plan AdminPlan
	err := json.Unmarshal(w.Body.Bytes(), &plan)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if plan.Name != "test-plan" {
		t.Fatalf("Expected plan name 'test-plan', got %s", plan.Name)
	}
	if plan.DisplayName != "Test Plan" {
		t.Fatalf("Expected display name 'Test Plan', got %s", plan.DisplayName)
	}
}

func TestHandler_CreatePlan_InvalidPrice(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	router := setupTestRouter()
	router.POST("/admin/plans", handler.CreatePlan)

	createReq := CreatePlanRequest{
		Name:                    "test-plan",
		DisplayName:             "Test Plan",
		Description:             "A test plan",
		PricePerMonthCents:      -1000, // Invalid negative price
		MonthlyConversionsLimit: 10,
		Features:                []string{"feature1", "feature2"},
		IsActive:                true,
	}

	jsonData, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/admin/plans", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", w.Code)
	}
}

func TestHandler_GetSystemStats(t *testing.T) {
	store := NewMockStore()
	_, handler := WireAdminServiceWithMocks(store)

	// Set up mock stats
	store.systemStats = AdminStats{
		TotalUsers:         100,
		ActiveUsers:        80,
		TotalVendors:       20,
		ActiveVendors:      15,
		TotalConversions:   500,
		TotalPayments:      200,
		TotalRevenue:       10000,
		TotalImages:        1000,
		PendingConversions: 10,
		FailedConversions:  5,
	}

	router := setupTestRouter()
	router.GET("/admin/stats", handler.GetSystemStats)

	req, _ := http.NewRequest("GET", "/admin/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var stats AdminStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if stats.TotalUsers != 100 {
		t.Fatalf("Expected total users 100, got %d", stats.TotalUsers)
	}
	if stats.ActiveUsers != 80 {
		t.Fatalf("Expected active users 80, got %d", stats.ActiveUsers)
	}
}
