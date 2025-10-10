package dashboard

import (
	"testing"
	"time"
)

// Simple test functions without external dependencies
func TestCalculateUsagePercentage(t *testing.T) {
	tests := []struct {
		name     string
		used     int
		limit    int
		expected float64
	}{
		{
			name:     "Normal usage",
			used:     50,
			limit:    100,
			expected: 50.0,
		},
		{
			name:     "Zero limit",
			used:     10,
			limit:    0,
			expected: 0.0,
		},
		{
			name:     "Over limit",
			used:     150,
			limit:    100,
			expected: 100.0,
		},
		{
			name:     "No usage",
			used:     0,
			limit:    100,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateUsagePercentage(tt.used, tt.limit)
			if result != tt.expected {
				t.Errorf("calculateUsagePercentage(%d, %d) = %f, want %f", tt.used, tt.limit, result, tt.expected)
			}
		})
	}
}

func TestGetUrgencyLevel(t *testing.T) {
	tests := []struct {
		name          string
		remaining     int
		expectedLevel string
	}{
		{
			name:          "No remaining",
			remaining:     0,
			expectedLevel: UrgencyHigh,
		},
		{
			name:          "Low remaining",
			remaining:     2,
			expectedLevel: UrgencyMedium,
		},
		{
			name:          "High remaining",
			remaining:     10,
			expectedLevel: UrgencyLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUrgencyLevel(tt.remaining)
			if result != tt.expectedLevel {
				t.Errorf("getUrgencyLevel(%d) = %s, want %s", tt.remaining, result, tt.expectedLevel)
			}
		})
	}
}

func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := stringPtr(s)
	if ptr == nil {
		t.Error("stringPtr returned nil")
	}
	if *ptr != s {
		t.Errorf("stringPtr(%s) = %s, want %s", s, *ptr, s)
	}
}

func TestIntPtr(t *testing.T) {
	i := 42
	ptr := intPtr(i)
	if ptr == nil {
		t.Error("intPtr returned nil")
	}
	if *ptr != i {
		t.Errorf("intPtr(%d) = %d, want %d", i, *ptr, i)
	}
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := timePtr(now)
	if ptr == nil {
		t.Error("timePtr returned nil")
	}
	if !ptr.Equal(now) {
		t.Errorf("timePtr(%v) = %v, want %v", now, *ptr, now)
	}
}

func TestDashboardData_Validation(t *testing.T) {
	// Test that DashboardData can be created with valid data
	data := DashboardData{
		User: UserInfo{
			ID:              "test-user",
			Name:            stringPtr("Test User"),
			Phone:           "1234567890",
			IsPhoneVerified: true,
			Role:            "user",
			CreatedAt:       time.Now(),
		},
		QuotaStatus: QuotaStatus{
			FreeConversionsRemaining:  1,
			PaidConversionsRemaining:  0,
			TotalConversionsRemaining: 1,
			FreeConversionsUsed:       1,
			FreeConversionsLimit:      2,
			PlanName:                  "free",
			MonthlyLimit:              0,
			MonthlyUsage:              0,
			UsagePercentage:           50.0,
		},
		ConversionHistory: ConversionHistory{
			RecentConversions:     []ConversionSummary{},
			TotalConversions:      0,
			SuccessfulConversions: 0,
			FailedConversions:     0,
		},
		VendorGallery: VendorGallery{
			FeaturedAlbums: []AlbumSummary{},
			RecentImages:   []ImageSummary{},
			TotalAlbums:    0,
			TotalImages:    0,
			PopularStyles:  []StyleInfo{},
		},
		PlanStatus: PlanStatus{
			CurrentPlan:      "free",
			PlanDisplayName:  "Free Plan",
			Status:           "active",
			MonthlyLimit:     0,
			PricePerMonth:    0,
			AutoRenew:        false,
			Features:         []string{"2 free conversions"},
			UpgradeAvailable: true,
			AvailablePlans:   []PlanInfo{},
		},
		Statistics: DashboardStatistics{
			TotalConversions:    0,
			SuccessfulRate:      0.0,
			TotalProcessingTime: 0,
			ConversionTrends:    []TrendData{},
		},
		RecentActivity: []RecentActivity{},
	}

	// Basic validation
	if data.User.ID == "" {
		t.Error("User ID should not be empty")
	}
	if data.QuotaStatus.TotalConversionsRemaining < 0 {
		t.Error("Total conversions remaining should not be negative")
	}
	if data.PlanStatus.CurrentPlan == "" {
		t.Error("Current plan should not be empty")
	}
}

func TestQuotaStatus_Validation(t *testing.T) {
	quota := QuotaStatus{
		FreeConversionsRemaining:  1,
		PaidConversionsRemaining:  0,
		TotalConversionsRemaining: 1,
		FreeConversionsUsed:       1,
		FreeConversionsLimit:      2,
		PlanName:                  "free",
		MonthlyLimit:              0,
		MonthlyUsage:              0,
		UsagePercentage:           50.0,
	}

	if quota.TotalConversionsRemaining != quota.FreeConversionsRemaining+quota.PaidConversionsRemaining {
		t.Error("Total conversions remaining should equal free + paid remaining")
	}
	if quota.UsagePercentage < 0 || quota.UsagePercentage > 100 {
		t.Error("Usage percentage should be between 0 and 100")
	}
}

func TestUpgradePrompt_Validation(t *testing.T) {
	prompt := &UpgradePrompt{
		ShowPrompt:      true,
		Title:           "Test Title",
		Message:         "Test Message",
		ActionText:      "Test Action",
		ActionURL:       "/test",
		UrgencyLevel:    UrgencyMedium,
		RemainingQuota:  1,
		RecommendedPlan: "basic",
	}

	if prompt.Title == "" {
		t.Error("Upgrade prompt title should not be empty")
	}
	if prompt.Message == "" {
		t.Error("Upgrade prompt message should not be empty")
	}
	if prompt.ActionText == "" {
		t.Error("Upgrade prompt action text should not be empty")
	}
	if prompt.ActionURL == "" {
		t.Error("Upgrade prompt action URL should not be empty")
	}
	if prompt.UrgencyLevel != UrgencyLow && prompt.UrgencyLevel != UrgencyMedium && prompt.UrgencyLevel != UrgencyHigh {
		t.Error("Upgrade prompt urgency level should be valid")
	}
}

func TestConstants(t *testing.T) {
	// Test that constants are properly defined
	if UrgencyLow == "" {
		t.Error("UrgencyLow constant should be defined")
	}
	if UrgencyMedium == "" {
		t.Error("UrgencyMedium constant should be defined")
	}
	if UrgencyHigh == "" {
		t.Error("UrgencyHigh constant should be defined")
	}
	if ActivityTypeConversion == "" {
		t.Error("ActivityTypeConversion constant should be defined")
	}
	if ActivityTypeLogin == "" {
		t.Error("ActivityTypeLogin constant should be defined")
	}
	if ActivityTypePlanChange == "" {
		t.Error("ActivityTypePlanChange constant should be defined")
	}
	if ActivityTypePayment == "" {
		t.Error("ActivityTypePayment constant should be defined")
	}
	if DefaultHistoryLimit <= 0 {
		t.Error("DefaultHistoryLimit should be positive")
	}
	if DefaultGalleryLimit <= 0 {
		t.Error("DefaultGalleryLimit should be positive")
	}
	if MaxHistoryLimit <= 0 {
		t.Error("MaxHistoryLimit should be positive")
	}
	if MaxGalleryLimit <= 0 {
		t.Error("MaxGalleryLimit should be positive")
	}
}
