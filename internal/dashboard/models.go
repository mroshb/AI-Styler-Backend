package dashboard

import (
	"time"
)

// DashboardData represents the complete dashboard information for a user
type DashboardData struct {
	User              UserInfo            `json:"user"`
	QuotaStatus       QuotaStatus         `json:"quotaStatus"`
	ConversionHistory ConversionHistory   `json:"conversionHistory"`
	VendorGallery     VendorGallery       `json:"vendorGallery"`
	PlanStatus        PlanStatus          `json:"planStatus"`
	UpgradePrompt     *UpgradePrompt      `json:"upgradePrompt,omitempty"`
	Statistics        DashboardStatistics `json:"statistics"`
	RecentActivity    []RecentActivity    `json:"recentActivity"`
}

// UserInfo represents basic user information for dashboard
type UserInfo struct {
	ID              string     `json:"id"`
	Name            *string    `json:"name,omitempty"`
	AvatarURL       *string    `json:"avatarUrl,omitempty"`
	Phone           string     `json:"phone"`
	IsPhoneVerified bool       `json:"isPhoneVerified"`
	Role            string     `json:"role"`
	CreatedAt       time.Time  `json:"createdAt"`
	LastLoginAt     *time.Time `json:"lastLoginAt,omitempty"`
}

// QuotaStatus represents current quota information
type QuotaStatus struct {
	FreeConversionsRemaining  int        `json:"freeConversionsRemaining"`
	PaidConversionsRemaining  int        `json:"paidConversionsRemaining"`
	TotalConversionsRemaining int        `json:"totalConversionsRemaining"`
	FreeConversionsUsed       int        `json:"freeConversionsUsed"`
	FreeConversionsLimit      int        `json:"freeConversionsLimit"`
	PlanName                  string     `json:"planName"`
	MonthlyLimit              int        `json:"monthlyLimit"`
	MonthlyUsage              int        `json:"monthlyUsage"`
	UsagePercentage           float64    `json:"usagePercentage"`
	ResetDate                 *time.Time `json:"resetDate,omitempty"`
}

// ConversionHistory represents user's conversion history
type ConversionHistory struct {
	RecentConversions     []ConversionSummary `json:"recentConversions"`
	TotalConversions      int                 `json:"totalConversions"`
	SuccessfulConversions int                 `json:"successfulConversions"`
	FailedConversions     int                 `json:"failedConversions"`
	AverageProcessingTime *int                `json:"averageProcessingTime,omitempty"`
	LastConversionAt      *time.Time          `json:"lastConversionAt,omitempty"`
}

// ConversionSummary represents a summary of a conversion for dashboard display
type ConversionSummary struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	StyleName        string     `json:"styleName"`
	InputImageURL    string     `json:"inputImageUrl"`
	ResultImageURL   *string    `json:"resultImageUrl,omitempty"`
	ProcessingTimeMs *int       `json:"processingTimeMs,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
}

// VendorGallery represents vendor gallery information
type VendorGallery struct {
	FeaturedAlbums []AlbumSummary `json:"featuredAlbums"`
	RecentImages   []ImageSummary `json:"recentImages"`
	TotalAlbums    int            `json:"totalAlbums"`
	TotalImages    int            `json:"totalImages"`
	PopularStyles  []StyleInfo    `json:"popularStyles"`
}

// AlbumSummary represents a summary of a vendor album
type AlbumSummary struct {
	ID            string    `json:"id"`
	VendorID      string    `json:"vendorId"`
	Name          string    `json:"name"`
	Description   *string   `json:"description,omitempty"`
	ImageCount    int       `json:"imageCount"`
	CoverImageURL *string   `json:"coverImageUrl,omitempty"`
	IsPublic      bool      `json:"isPublic"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ImageSummary represents a summary of a vendor image
type ImageSummary struct {
	ID           string    `json:"id"`
	VendorID     string    `json:"vendorId"`
	AlbumID      *string   `json:"albumId,omitempty"`
	FileName     string    `json:"fileName"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	OriginalURL  string    `json:"originalUrl"`
	IsPublic     bool      `json:"isPublic"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"createdAt"`
}

// StyleInfo represents popular style information
type StyleInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	ImageCount  int    `json:"imageCount"`
	Popularity  int    `json:"popularity"`
	PreviewURL  string `json:"previewUrl"`
}

// PlanStatus represents current plan information
type PlanStatus struct {
	CurrentPlan      string     `json:"currentPlan"`
	PlanDisplayName  string     `json:"planDisplayName"`
	Status           string     `json:"status"`
	MonthlyLimit     int        `json:"monthlyLimit"`
	PricePerMonth    int64      `json:"pricePerMonth"`
	BillingCycleEnd  *time.Time `json:"billingCycleEnd,omitempty"`
	AutoRenew        bool       `json:"autoRenew"`
	Features         []string   `json:"features"`
	UpgradeAvailable bool       `json:"upgradeAvailable"`
	AvailablePlans   []PlanInfo `json:"availablePlans"`
}

// PlanInfo represents available plan information
type PlanInfo struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	DisplayName             string   `json:"displayName"`
	Description             string   `json:"description"`
	PricePerMonthCents      int64    `json:"pricePerMonthCents"`
	MonthlyConversionsLimit int      `json:"monthlyConversionsLimit"`
	Features                []string `json:"features"`
	IsRecommended           bool     `json:"isRecommended"`
	IsCurrent               bool     `json:"isCurrent"`
}

// UpgradePrompt represents upgrade prompt information
type UpgradePrompt struct {
	ShowPrompt      bool   `json:"showPrompt"`
	Title           string `json:"title"`
	Message         string `json:"message"`
	ActionText      string `json:"actionText"`
	ActionURL       string `json:"actionUrl"`
	UrgencyLevel    string `json:"urgencyLevel"` // "low", "medium", "high"
	RemainingQuota  int    `json:"remainingQuota"`
	RecommendedPlan string `json:"recommendedPlan"`
}

// DashboardStatistics represents overall dashboard statistics
type DashboardStatistics struct {
	TotalConversions      int         `json:"totalConversions"`
	SuccessfulRate        float64     `json:"successfulRate"`
	AverageProcessingTime *int        `json:"averageProcessingTime,omitempty"`
	TotalProcessingTime   int         `json:"totalProcessingTime"`
	FavoriteStyle         *string     `json:"favoriteStyle,omitempty"`
	MostUsedStyle         *string     `json:"mostUsedStyle,omitempty"`
	ConversionTrends      []TrendData `json:"conversionTrends"`
}

// TrendData represents trend information
type TrendData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// RecentActivity represents recent user activity
type RecentActivity struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "conversion", "login", "plan_change", "payment"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DashboardRequest represents the request to get dashboard data
type DashboardRequest struct {
	IncludeHistory    bool `json:"includeHistory" form:"includeHistory"`
	IncludeGallery    bool `json:"includeGallery" form:"includeGallery"`
	IncludeStatistics bool `json:"includeStatistics" form:"includeStatistics"`
	HistoryLimit      int  `json:"historyLimit" form:"historyLimit"`
	GalleryLimit      int  `json:"galleryLimit" form:"galleryLimit"`
}

// QuotaCheckRequest represents the request to check quota status
type QuotaCheckRequest struct {
	UserID string `json:"userId" binding:"required"`
}

// QuotaCheckResponse represents the response for quota check
type QuotaCheckResponse struct {
	CanConvert        bool           `json:"canConvert"`
	QuotaStatus       QuotaStatus    `json:"quotaStatus"`
	UpgradePrompt     *UpgradePrompt `json:"upgradePrompt,omitempty"`
	RecommendedAction string         `json:"recommendedAction"`
}

// Constants
const (
	// Urgency levels for upgrade prompts
	UrgencyLow    = "low"
	UrgencyMedium = "medium"
	UrgencyHigh   = "high"

	// Activity types
	ActivityTypeConversion = "conversion"
	ActivityTypeLogin      = "login"
	ActivityTypePlanChange = "plan_change"
	ActivityTypePayment    = "payment"

	// Default limits
	DefaultHistoryLimit = 10
	DefaultGalleryLimit = 20
	MaxHistoryLimit     = 50
	MaxGalleryLimit     = 100
)

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}

// Helper function for creating time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
