package conversion

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Mock implementations for testing
type mockStore struct {
	conversions map[string]Conversion
	quota       map[string]QuotaCheck
}

func newMockStore() *mockStore {
	return &mockStore{
		conversions: make(map[string]Conversion),
		quota:       make(map[string]QuotaCheck),
	}
}

func (m *mockStore) CreateConversion(ctx context.Context, userID, userImageID, clothImageID string) (string, error) {
	conversionID := "test-conversion-id"
	conversion := Conversion{
		ID:           conversionID,
		UserID:       userID,
		UserImageID:  userImageID,
		ClothImageID: clothImageID,
		Status:       ConversionStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.conversions[conversionID] = conversion
	return conversionID, nil
}

func (m *mockStore) GetConversion(ctx context.Context, conversionID string) (Conversion, error) {
	conv, exists := m.conversions[conversionID]
	if !exists {
		return Conversion{}, fmt.Errorf("conversion not found")
	}
	return conv, nil
}

func (m *mockStore) GetConversionWithDetails(ctx context.Context, conversionID string) (ConversionResponse, error) {
	conv, exists := m.conversions[conversionID]
	if !exists {
		return ConversionResponse{}, fmt.Errorf("conversion not found")
	}

	response := ConversionResponse{
		ID:           conv.ID,
		UserID:       conv.UserID,
		UserImageID:  conv.UserImageID,
		ClothImageID: conv.ClothImageID,
		Status:       conv.Status,
		CreatedAt:    conv.CreatedAt,
		UpdatedAt:    conv.UpdatedAt,
	}

	if conv.ResultImageID != nil {
		response.ResultImageID = conv.ResultImageID
	}
	if conv.ErrorMessage != nil {
		response.ErrorMessage = conv.ErrorMessage
	}
	if conv.ProcessingTimeMs != nil {
		response.ProcessingTimeMs = conv.ProcessingTimeMs
	}
	if conv.CompletedAt != nil {
		response.CompletedAt = conv.CompletedAt
	}

	return response, nil
}

func (m *mockStore) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) error {
	conv, exists := m.conversions[conversionID]
	if !exists {
		return fmt.Errorf("conversion not found")
	}

	if req.Status != nil {
		conv.Status = *req.Status
	}
	if req.ResultImageID != nil {
		conv.ResultImageID = req.ResultImageID
	}
	if req.ErrorMessage != nil {
		conv.ErrorMessage = req.ErrorMessage
	}
	if req.ProcessingTimeMs != nil {
		conv.ProcessingTimeMs = req.ProcessingTimeMs
	}

	conv.UpdatedAt = time.Now()
	m.conversions[conversionID] = conv
	return nil
}

func (m *mockStore) ListConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error) {
	// Simple implementation for testing
	conversions := make([]ConversionResponse, 0)
	for _, conv := range m.conversions {
		if conv.UserID == req.UserID {
			response := ConversionResponse{
				ID:           conv.ID,
				UserID:       conv.UserID,
				UserImageID:  conv.UserImageID,
				ClothImageID: conv.ClothImageID,
				Status:       conv.Status,
				CreatedAt:    conv.CreatedAt,
				UpdatedAt:    conv.UpdatedAt,
			}
			conversions = append(conversions, response)
		}
	}

	return ConversionListResponse{
		Conversions: conversions,
		Total:       len(conversions),
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  1,
	}, nil
}

func (m *mockStore) DeleteConversion(ctx context.Context, conversionID string) error {
	delete(m.conversions, conversionID)
	return nil
}

func (m *mockStore) CheckUserQuota(ctx context.Context, userID string) (QuotaCheck, error) {
	quota, exists := m.quota[userID]
	if !exists {
		quota = QuotaCheck{
			CanConvert:     true,
			RemainingFree:  2,
			RemainingPaid:  0,
			TotalRemaining: 2,
			PlanName:       "free",
			MonthlyLimit:   0,
		}
		m.quota[userID] = quota
	}
	return quota, nil
}

func (m *mockStore) ReserveQuota(ctx context.Context, userID string) error {
	return nil
}

func (m *mockStore) ReleaseQuota(ctx context.Context, userID string) error {
	return nil
}

func (m *mockStore) CreateConversionJob(ctx context.Context, conversionID string) error {
	return nil
}

func (m *mockStore) GetNextJob(ctx context.Context) (*ConversionJob, error) {
	return nil, nil
}

func (m *mockStore) UpdateJobStatus(ctx context.Context, jobID, status, workerID string) error {
	return nil
}

func (m *mockStore) CompleteJob(ctx context.Context, jobID, resultImageID string, processingTimeMs int) error {
	return nil
}

func (m *mockStore) FailJob(ctx context.Context, jobID, errorMessage string) error {
	return nil
}

func TestCreateConversion(t *testing.T) {
	// Create mock service
	store := newMockStore()
	service := &Service{
		store:        store,
		imageService: &mockImageService{},
		processor:    &mockProcessor{},
		notifier:     &mockNotifier{},
		rateLimiter:  &mockRateLimiter{},
		auditLogger:  &mockAuditLogger{},
		worker:       &mockWorker{},
		metrics:      &mockMetrics{},
	}

	ctx := context.Background()
	userID := "test-user-id"
	req := ConversionRequest{
		UserImageID:  "user-image-id",
		ClothImageID: "cloth-image-id",
	}

	// Test successful conversion creation
	response, err := service.CreateConversion(ctx, userID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.ID == "" {
		t.Error("Expected conversion ID to be set")
	}
	if response.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, response.UserID)
	}
	if response.UserImageID != req.UserImageID {
		t.Errorf("Expected user image ID %s, got %s", req.UserImageID, response.UserImageID)
	}
	if response.ClothImageID != req.ClothImageID {
		t.Errorf("Expected cloth image ID %s, got %s", req.ClothImageID, response.ClothImageID)
	}
	if response.Status != ConversionStatusPending {
		t.Errorf("Expected status %s, got %s", ConversionStatusPending, response.Status)
	}
}

func TestGetConversion(t *testing.T) {
	store := newMockStore()
	service := &Service{
		store: store,
	}

	ctx := context.Background()
	userID := "test-user-id"
	conversionID := "test-conversion-id"

	// Create a test conversion
	store.conversions[conversionID] = Conversion{
		ID:           conversionID,
		UserID:       userID,
		UserImageID:  "user-image-id",
		ClothImageID: "cloth-image-id",
		Status:       ConversionStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Test getting conversion
	response, err := service.GetConversion(ctx, conversionID, userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.ID != conversionID {
		t.Errorf("Expected conversion ID %s, got %s", conversionID, response.ID)
	}
	if response.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, response.UserID)
	}
}

func TestGetQuotaStatus(t *testing.T) {
	store := newMockStore()
	service := &Service{
		store: store,
	}

	ctx := context.Background()
	userID := "test-user-id"

	// Test getting quota status
	quota, err := service.GetQuotaStatus(ctx, userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !quota.CanConvert {
		t.Error("Expected user to be able to convert")
	}
	if quota.RemainingFree != 2 {
		t.Errorf("Expected 2 free conversions remaining, got %d", quota.RemainingFree)
	}
	if quota.PlanName != "free" {
		t.Errorf("Expected plan name 'free', got %s", quota.PlanName)
	}
}
