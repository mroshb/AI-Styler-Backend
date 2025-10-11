package admin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// MockStore implements Store interface for testing
type MockStore struct {
	users           map[string]AdminUser
	vendors         map[string]AdminVendor
	plans           map[string]AdminPlan
	payments        map[string]AdminPayment
	conversions     map[string]AdminConversion
	images          map[string]AdminImage
	auditLogs       []AuditLog
	userStats       [2]int   // total, active
	vendorStats     [2]int   // total, active
	paymentStats    [2]int64 // total, revenue
	conversionStats [3]int   // total, pending, failed
	imageStats      int
	systemStats     AdminStats
}

// NewMockStore creates a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		users:       make(map[string]AdminUser),
		vendors:     make(map[string]AdminVendor),
		plans:       make(map[string]AdminPlan),
		payments:    make(map[string]AdminPayment),
		conversions: make(map[string]AdminConversion),
		images:      make(map[string]AdminImage),
		auditLogs:   make([]AuditLog, 0),
	}
}

// User operations
func (m *MockStore) GetUsers(ctx context.Context, req UserListRequest) (UserListResponse, error) {
	users := make([]AdminUser, 0)
	for _, user := range m.users {
		users = append(users, user)
	}

	return UserListResponse{
		Users:      users,
		Total:      len(users),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockStore) GetUser(ctx context.Context, userID string) (AdminUser, error) {
	user, exists := m.users[userID]
	if !exists {
		return AdminUser{}, errors.New("user not found")
	}
	return user, nil
}

func (m *MockStore) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (AdminUser, error) {
	user, exists := m.users[userID]
	if !exists {
		return AdminUser{}, errors.New("user not found")
	}

	if req.Name != nil {
		user.Name = req.Name
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.IsPhoneVerified != nil {
		user.IsPhoneVerified = *req.IsPhoneVerified
	}
	if req.FreeConversionsLimit != nil {
		user.FreeConversionsLimit = *req.FreeConversionsLimit
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	user.UpdatedAt = time.Now()
	m.users[userID] = user
	return user, nil
}

func (m *MockStore) DeleteUser(ctx context.Context, userID string) error {
	delete(m.users, userID)
	return nil
}

func (m *MockStore) GetUserStats(ctx context.Context) (int, int, error) {
	return m.userStats[0], m.userStats[1], nil
}

// Vendor operations
func (m *MockStore) GetVendors(ctx context.Context, req VendorListRequest) (VendorListResponse, error) {
	vendors := make([]AdminVendor, 0)
	for _, vendor := range m.vendors {
		vendors = append(vendors, vendor)
	}

	return VendorListResponse{
		Vendors:    vendors,
		Total:      len(vendors),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockStore) GetVendor(ctx context.Context, vendorID string) (AdminVendor, error) {
	vendor, exists := m.vendors[vendorID]
	if !exists {
		return AdminVendor{}, errors.New("vendor not found")
	}
	return vendor, nil
}

func (m *MockStore) UpdateVendor(ctx context.Context, vendorID string, req UpdateVendorRequest) (AdminVendor, error) {
	vendor, exists := m.vendors[vendorID]
	if !exists {
		return AdminVendor{}, errors.New("vendor not found")
	}

	if req.BusinessName != nil {
		vendor.BusinessName = *req.BusinessName
	}
	if req.AvatarURL != nil {
		vendor.AvatarURL = req.AvatarURL
	}
	if req.Bio != nil {
		vendor.Bio = req.Bio
	}
	if req.IsVerified != nil {
		vendor.IsVerified = *req.IsVerified
	}
	if req.IsActive != nil {
		vendor.IsActive = *req.IsActive
	}
	if req.FreeImagesLimit != nil {
		vendor.FreeImagesLimit = *req.FreeImagesLimit
	}

	vendor.UpdatedAt = time.Now()
	m.vendors[vendorID] = vendor
	return vendor, nil
}

func (m *MockStore) DeleteVendor(ctx context.Context, vendorID string) error {
	delete(m.vendors, vendorID)
	return nil
}

func (m *MockStore) GetVendorStats(ctx context.Context) (int, int, error) {
	return m.vendorStats[0], m.vendorStats[1], nil
}

// Plan operations
func (m *MockStore) GetPlans(ctx context.Context, req PlanListRequest) (PlanListResponse, error) {
	plans := make([]AdminPlan, 0)
	for _, plan := range m.plans {
		plans = append(plans, plan)
	}

	return PlanListResponse{
		Plans:      plans,
		Total:      len(plans),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockStore) GetPlan(ctx context.Context, planID string) (AdminPlan, error) {
	plan, exists := m.plans[planID]
	if !exists {
		return AdminPlan{}, errors.New("plan not found")
	}
	return plan, nil
}

func (m *MockStore) CreatePlan(ctx context.Context, req CreatePlanRequest) (AdminPlan, error) {
	plan := AdminPlan{
		ID:                      "plan-" + req.Name,
		Name:                    req.Name,
		DisplayName:             req.DisplayName,
		Description:             req.Description,
		PricePerMonthCents:      req.PricePerMonthCents,
		MonthlyConversionsLimit: req.MonthlyConversionsLimit,
		Features:                req.Features,
		IsActive:                req.IsActive,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
		SubscriberCount:         0,
	}

	m.plans[plan.ID] = plan
	return plan, nil
}

func (m *MockStore) UpdatePlan(ctx context.Context, planID string, req UpdatePlanRequest) (AdminPlan, error) {
	plan, exists := m.plans[planID]
	if !exists {
		return AdminPlan{}, errors.New("plan not found")
	}

	if req.DisplayName != nil {
		plan.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		plan.Description = *req.Description
	}
	if req.PricePerMonthCents != nil {
		plan.PricePerMonthCents = *req.PricePerMonthCents
	}
	if req.MonthlyConversionsLimit != nil {
		plan.MonthlyConversionsLimit = *req.MonthlyConversionsLimit
	}
	if req.Features != nil {
		plan.Features = req.Features
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	plan.UpdatedAt = time.Now()
	m.plans[planID] = plan
	return plan, nil
}

func (m *MockStore) DeletePlan(ctx context.Context, planID string) error {
	delete(m.plans, planID)
	return nil
}

// Payment operations
func (m *MockStore) GetPayments(ctx context.Context, req PaymentListRequest) (PaymentListResponse, error) {
	payments := make([]AdminPayment, 0)
	for _, payment := range m.payments {
		payments = append(payments, payment)
	}

	return PaymentListResponse{
		Payments:   payments,
		Total:      len(payments),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockStore) GetPayment(ctx context.Context, paymentID string) (AdminPayment, error) {
	payment, exists := m.payments[paymentID]
	if !exists {
		return AdminPayment{}, errors.New("payment not found")
	}
	return payment, nil
}

func (m *MockStore) GetPaymentStats(ctx context.Context) (int, int64, error) {
	return int(m.paymentStats[0]), m.paymentStats[1], nil
}

// Conversion operations
func (m *MockStore) GetConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error) {
	conversions := make([]AdminConversion, 0)
	for _, conversion := range m.conversions {
		conversions = append(conversions, conversion)
	}

	return ConversionListResponse{
		Conversions: conversions,
		Total:       len(conversions),
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  1,
	}, nil
}

func (m *MockStore) GetConversion(ctx context.Context, conversionID string) (AdminConversion, error) {
	conversion, exists := m.conversions[conversionID]
	if !exists {
		return AdminConversion{}, errors.New("conversion not found")
	}
	return conversion, nil
}

func (m *MockStore) GetConversionStats(ctx context.Context) (int, int, int, error) {
	return m.conversionStats[0], m.conversionStats[1], m.conversionStats[2], nil
}

// Image operations
func (m *MockStore) GetImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	images := make([]AdminImage, 0)
	for _, image := range m.images {
		images = append(images, image)
	}

	return ImageListResponse{
		Images:     images,
		Total:      len(images),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockStore) GetImage(ctx context.Context, imageID string) (AdminImage, error) {
	image, exists := m.images[imageID]
	if !exists {
		return AdminImage{}, errors.New("image not found")
	}
	return image, nil
}

func (m *MockStore) GetImageStats(ctx context.Context) (int, error) {
	return m.imageStats, nil
}

// Audit log operations
func (m *MockStore) GetAuditLogs(ctx context.Context, req AuditLogListRequest) (AuditLogListResponse, error) {
	return AuditLogListResponse{
		AuditLogs:  m.auditLogs,
		Total:      len(m.auditLogs),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockStore) CreateAuditLog(ctx context.Context, log AuditLog) error {
	m.auditLogs = append(m.auditLogs, log)
	return nil
}

// Quota operations
func (m *MockStore) RevokeUserQuota(ctx context.Context, userID string, quotaType string, amount int, reason string) error {
	// Mock implementation
	return nil
}

func (m *MockStore) RevokeVendorQuota(ctx context.Context, vendorID string, quotaType string, amount int, reason string) error {
	// Mock implementation
	return nil
}

func (m *MockStore) RevokeUserPlan(ctx context.Context, userID string, reason string) error {
	// Mock implementation
	return nil
}

// Statistics
func (m *MockStore) GetSystemStats(ctx context.Context) (AdminStats, error) {
	return m.systemStats, nil
}

// Test cases

func TestAdminService_GetUsers(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test users
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	req := UserListRequest{Page: 1, PageSize: 10}
	response, err := service.GetUsers(context.Background(), req)

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

func TestAdminService_GetUser(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	user, err := service.GetUser(context.Background(), "user1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if user.ID != "user1" {
		t.Fatalf("Expected user ID user1, got %s", user.ID)
	}
}

func TestAdminService_GetUser_NotFound(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	_, err := service.GetUser(context.Background(), "nonexistent")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("Expected error containing 'user not found', got %v", err)
	}
}

func TestAdminService_UpdateUser(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	req := UpdateUserRequest{
		Name: stringPtr("Updated Name"),
		Role: stringPtr("admin"),
	}

	user, err := service.UpdateUser(context.Background(), "user1", req)

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

func TestAdminService_UpdateUser_InvalidRole(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	req := UpdateUserRequest{
		Role: stringPtr("invalid_role"),
	}

	_, err := service.UpdateUser(context.Background(), "user1", req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != "invalid role" {
		t.Fatalf("Expected 'invalid role' error, got %v", err)
	}
}

func TestAdminService_DeleteUser(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:    "user1",
		Phone: "1234567890",
		Role:  "user",
	}

	err := service.DeleteUser(context.Background(), "user1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify user is deleted
	_, exists := store.users["user1"]
	if exists {
		t.Fatal("Expected user to be deleted")
	}
}

func TestAdminService_SuspendUser(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:       "user1",
		Phone:    "1234567890",
		Role:     "user",
		IsActive: true,
	}

	err := service.SuspendUser(context.Background(), "user1", "Test suspension")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	user := store.users["user1"]
	if user.IsActive {
		t.Fatal("Expected user to be suspended")
	}
}

func TestAdminService_ActivateUser(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Add test user
	store.users["user1"] = AdminUser{
		ID:       "user1",
		Phone:    "1234567890",
		Role:     "user",
		IsActive: false,
	}

	err := service.ActivateUser(context.Background(), "user1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	user := store.users["user1"]
	if !user.IsActive {
		t.Fatal("Expected user to be activated")
	}
}

func TestAdminService_CreatePlan(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	req := CreatePlanRequest{
		Name:                    "test-plan",
		DisplayName:             "Test Plan",
		Description:             "A test plan",
		PricePerMonthCents:      1000,
		MonthlyConversionsLimit: 10,
		Features:                []string{"feature1", "feature2"},
		IsActive:                true,
	}

	plan, err := service.CreatePlan(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if plan.Name != "test-plan" {
		t.Fatalf("Expected plan name 'test-plan', got %s", plan.Name)
	}

	if plan.PricePerMonthCents != 1000 {
		t.Fatalf("Expected price 1000, got %d", plan.PricePerMonthCents)
	}
}

func TestAdminService_CreatePlan_InvalidPrice(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	req := CreatePlanRequest{
		Name:                    "test-plan",
		DisplayName:             "Test Plan",
		Description:             "A test plan",
		PricePerMonthCents:      -1000, // Invalid negative price
		MonthlyConversionsLimit: 10,
		Features:                []string{"feature1", "feature2"},
		IsActive:                true,
	}

	_, err := service.CreatePlan(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != "price cannot be negative" {
		t.Fatalf("Expected 'price cannot be negative' error, got %v", err)
	}
}

func TestAdminService_RevokeUserQuota(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	req := RevokeQuotaRequest{
		UserID:    "user1",
		QuotaType: "free",
		Amount:    5,
		Reason:    "Test revocation",
	}

	err := service.RevokeUserQuota(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestAdminService_RevokeUserQuota_InvalidQuotaType(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	req := RevokeQuotaRequest{
		UserID:    "user1",
		QuotaType: "invalid",
		Amount:    5,
		Reason:    "Test revocation",
	}

	err := service.RevokeUserQuota(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != "invalid quota type" {
		t.Fatalf("Expected 'invalid quota type' error, got %v", err)
	}
}

func TestAdminService_GetSystemStats(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

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

	stats, err := service.GetSystemStats(context.Background())

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

func TestAdminService_GetUserStats(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Set up mock stats
	store.userStats = [2]int{100, 80}

	total, active, err := service.GetUserStats(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 100 {
		t.Fatalf("Expected total 100, got %d", total)
	}

	if active != 80 {
		t.Fatalf("Expected active 80, got %d", active)
	}
}

func TestAdminService_GetVendorStats(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Set up mock stats
	store.vendorStats = [2]int{20, 15}

	total, active, err := service.GetVendorStats(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 20 {
		t.Fatalf("Expected total 20, got %d", total)
	}

	if active != 15 {
		t.Fatalf("Expected active 15, got %d", active)
	}
}

func TestAdminService_GetPaymentStats(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Set up mock stats
	store.paymentStats = [2]int64{200, 10000}

	total, revenue, err := service.GetPaymentStats(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 200 {
		t.Fatalf("Expected total 200, got %d", total)
	}

	if revenue != 10000 {
		t.Fatalf("Expected revenue 10000, got %d", revenue)
	}
}

func TestAdminService_GetConversionStats(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Set up mock stats
	store.conversionStats = [3]int{500, 10, 5}

	total, pending, failed, err := service.GetConversionStats(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 500 {
		t.Fatalf("Expected total 500, got %d", total)
	}

	if pending != 10 {
		t.Fatalf("Expected pending 10, got %d", pending)
	}

	if failed != 5 {
		t.Fatalf("Expected failed 5, got %d", failed)
	}
}

func TestAdminService_GetImageStats(t *testing.T) {
	store := NewMockStore()
	service, _ := WireAdminServiceWithMocks(store)

	// Set up mock stats
	store.imageStats = 1000

	total, err := service.GetImageStats(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 1000 {
		t.Fatalf("Expected total 1000, got %d", total)
	}
}
