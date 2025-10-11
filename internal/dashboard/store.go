package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// StoreImpl implements the dashboard data access layer
type StoreImpl struct {
	db *sql.DB
}

// NewStoreImpl creates a new dashboard store implementation
func NewStoreImpl(db *sql.DB) Store {
	return &StoreImpl{db: db}
}

// GetDashboardData retrieves comprehensive dashboard data
func (s *StoreImpl) GetDashboardData(ctx context.Context, userID string, req DashboardRequest) (DashboardData, error) {
	// This would typically aggregate data from multiple sources
	// For now, we'll implement individual methods and combine them

	userInfo, err := s.getUserInfo(ctx, userID)
	if err != nil {
		return DashboardData{}, err
	}

	quotaStatus, err := s.GetQuotaStatus(ctx, userID)
	if err != nil {
		return DashboardData{}, err
	}

	var conversionHistory ConversionHistory
	if req.IncludeHistory {
		conversionHistory, err = s.GetConversionHistory(ctx, userID, req.HistoryLimit)
		if err != nil {
			return DashboardData{}, err
		}
	}

	var vendorGallery VendorGallery
	if req.IncludeGallery {
		vendorGallery, err = s.GetVendorGallery(ctx, req.GalleryLimit)
		if err != nil {
			return DashboardData{}, err
		}
	}

	planStatus, err := s.GetPlanStatus(ctx, userID)
	if err != nil {
		return DashboardData{}, err
	}

	upgradePrompt, err := s.ShouldShowUpgradePrompt(ctx, userID)
	if err != nil {
		fmt.Printf("Failed to get upgrade prompt: %v\n", err)
	}

	var statistics DashboardStatistics
	if req.IncludeStatistics {
		statistics, err = s.GetDashboardStatistics(ctx, userID)
		if err != nil {
			return DashboardData{}, err
		}
	}

	recentActivity, err := s.GetRecentActivity(ctx, userID, 10)
	if err != nil {
		fmt.Printf("Failed to get recent activity: %v\n", err)
		recentActivity = []RecentActivity{}
	}

	return DashboardData{
		User:              userInfo,
		QuotaStatus:       quotaStatus,
		ConversionHistory: conversionHistory,
		VendorGallery:     vendorGallery,
		PlanStatus:        planStatus,
		UpgradePrompt:     upgradePrompt,
		Statistics:        statistics,
		RecentActivity:    recentActivity,
	}, nil
}

// GetQuotaStatus retrieves current quota status
func (s *StoreImpl) GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error) {
	query := `
		SELECT 
			free_conversions_remaining,
			paid_conversions_remaining,
			total_conversions_remaining,
			plan_name,
			monthly_limit
		FROM get_user_quota_status($1)
	`

	var quota QuotaStatus
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&quota.FreeConversionsRemaining,
		&quota.PaidConversionsRemaining,
		&quota.TotalConversionsRemaining,
		&quota.PlanName,
		&quota.MonthlyLimit,
	)
	if err != nil {
		return QuotaStatus{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	// Get additional quota information
	quota.FreeConversionsUsed = 2 - quota.FreeConversionsRemaining // Assuming 2 free conversions
	quota.FreeConversionsLimit = 2
	quota.MonthlyUsage = quota.MonthlyLimit - quota.TotalConversionsRemaining
	quota.UsagePercentage = calculateUsagePercentage(quota.MonthlyUsage, quota.MonthlyLimit)

	// Calculate reset date (next month)
	now := time.Now()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	quota.ResetDate = &nextMonth

	return quota, nil
}

// GetConversionHistory retrieves conversion history
func (s *StoreImpl) GetConversionHistory(ctx context.Context, userID string, limit int) (ConversionHistory, error) {
	query := `
		SELECT 
			id,
			status,
			style_name,
			input_file_url,
			result_file_url,
			processing_time_ms,
			created_at,
			completed_at,
			error_message
		FROM user_conversions 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return ConversionHistory{}, fmt.Errorf("failed to query conversion history: %w", err)
	}
	defer rows.Close()

	var conversions []ConversionSummary
	var totalConversions, successfulConversions, failedConversions int
	var totalProcessingTime int
	var lastConversionAt *time.Time

	for rows.Next() {
		var conv ConversionSummary
		var resultFileURL sql.NullString
		var processingTimeMs sql.NullInt32
		var completedAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&conv.ID,
			&conv.Status,
			&conv.StyleName,
			&conv.InputImageURL,
			&resultFileURL,
			&processingTimeMs,
			&conv.CreatedAt,
			&completedAt,
			&errorMessage,
		)
		if err != nil {
			return ConversionHistory{}, fmt.Errorf("failed to scan conversion: %w", err)
		}

		if resultFileURL.Valid {
			conv.ResultImageURL = &resultFileURL.String
		}
		if processingTimeMs.Valid {
			timeMs := int(processingTimeMs.Int32)
			conv.ProcessingTimeMs = &timeMs
			totalProcessingTime += timeMs
		}
		if completedAt.Valid {
			conv.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			conv.ErrorMessage = &errorMessage.String
		}

		conversions = append(conversions, conv)
		totalConversions++

		if conv.Status == "completed" {
			successfulConversions++
		} else if conv.Status == "failed" {
			failedConversions++
		}

		if lastConversionAt == nil || conv.CreatedAt.After(*lastConversionAt) {
			lastConversionAt = &conv.CreatedAt
		}
	}

	var averageProcessingTime *int
	if totalConversions > 0 {
		avg := totalProcessingTime / totalConversions
		averageProcessingTime = &avg
	}

	return ConversionHistory{
		RecentConversions:     conversions,
		TotalConversions:      totalConversions,
		SuccessfulConversions: successfulConversions,
		FailedConversions:     failedConversions,
		AverageProcessingTime: averageProcessingTime,
		LastConversionAt:      lastConversionAt,
	}, nil
}

// GetVendorGallery retrieves vendor gallery information
func (s *StoreImpl) GetVendorGallery(ctx context.Context, limit int) (VendorGallery, error) {
	// Get featured albums
	albumsQuery := `
		SELECT 
			a.id,
			a.vendor_id,
			a.name,
			a.description,
			a.image_count,
			a.is_public,
			a.created_at,
			i.original_url as cover_image_url
		FROM albums a
		LEFT JOIN images i ON a.id = i.album_id AND i.is_public = true
		WHERE a.is_public = true
		ORDER BY a.image_count DESC, a.created_at DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, albumsQuery, limit/2)
	if err != nil {
		return VendorGallery{}, fmt.Errorf("failed to query albums: %w", err)
	}
	defer rows.Close()

	var albums []AlbumSummary
	for rows.Next() {
		var album AlbumSummary
		var description sql.NullString
		var coverImageURL sql.NullString

		err := rows.Scan(
			&album.ID,
			&album.VendorID,
			&album.Name,
			&description,
			&album.ImageCount,
			&album.IsPublic,
			&album.CreatedAt,
			&coverImageURL,
		)
		if err != nil {
			return VendorGallery{}, fmt.Errorf("failed to scan album: %w", err)
		}

		if description.Valid {
			album.Description = &description.String
		}
		if coverImageURL.Valid {
			album.CoverImageURL = &coverImageURL.String
		}

		albums = append(albums, album)
	}

	// Get recent images
	imagesQuery := `
		SELECT 
			i.id,
			i.vendor_id,
			i.album_id,
			i.file_name,
			i.thumbnail_url,
			i.original_url,
			i.is_public,
			i.tags,
			i.created_at
		FROM images i
		WHERE i.is_public = true AND i.type = 'vendor'
		ORDER BY i.created_at DESC
		LIMIT $1
	`

	rows, err = s.db.QueryContext(ctx, imagesQuery, limit)
	if err != nil {
		return VendorGallery{}, fmt.Errorf("failed to query images: %w", err)
	}
	defer rows.Close()

	var images []ImageSummary
	for rows.Next() {
		var img ImageSummary
		var albumID sql.NullString
		var tagsJSON sql.NullString

		err := rows.Scan(
			&img.ID,
			&img.VendorID,
			&albumID,
			&img.FileName,
			&img.ThumbnailURL,
			&img.OriginalURL,
			&img.IsPublic,
			&tagsJSON,
			&img.CreatedAt,
		)
		if err != nil {
			return VendorGallery{}, fmt.Errorf("failed to scan image: %w", err)
		}

		if albumID.Valid {
			img.AlbumID = &albumID.String
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &img.Tags)
		}

		images = append(images, img)
	}

	// Get total counts
	var totalAlbums, totalImages int
	countQuery := `
		SELECT 
			(SELECT COUNT(*) FROM albums WHERE is_public = true),
			(SELECT COUNT(*) FROM images WHERE is_public = true AND type = 'vendor')
	`
	err = s.db.QueryRowContext(ctx, countQuery).Scan(&totalAlbums, &totalImages)
	if err != nil {
		fmt.Printf("Failed to get total counts: %v\n", err)
	}

	// Get popular styles (simplified)
	popularStyles := []StyleInfo{
		{Name: "casual", DisplayName: "Casual", ImageCount: 25, Popularity: 85, PreviewURL: ""},
		{Name: "formal", DisplayName: "Formal", ImageCount: 20, Popularity: 75, PreviewURL: ""},
		{Name: "sporty", DisplayName: "Sporty", ImageCount: 15, Popularity: 65, PreviewURL: ""},
	}

	return VendorGallery{
		FeaturedAlbums: albums,
		RecentImages:   images,
		TotalAlbums:    totalAlbums,
		TotalImages:    totalImages,
		PopularStyles:  popularStyles,
	}, nil
}

// GetPlanStatus retrieves current plan status
func (s *StoreImpl) GetPlanStatus(ctx context.Context, userID string) (PlanStatus, error) {
	query := `
		SELECT 
			COALESCE(up.plan_name, 'free') as current_plan,
			COALESCE(up.status, 'active') as status,
			COALESCE(up.monthly_conversions_limit, 0) as monthly_limit,
			COALESCE(up.price_per_month_cents, 0) as price_per_month,
			up.billing_cycle_end_date,
			COALESCE(up.auto_renew, false) as auto_renew
		FROM users u
		LEFT JOIN user_plans up ON u.id = up.user_id AND up.status = 'active'
		WHERE u.id = $1
	`

	var plan PlanStatus
	var billingCycleEnd sql.NullTime

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&plan.CurrentPlan,
		&plan.Status,
		&plan.MonthlyLimit,
		&plan.PricePerMonth,
		&billingCycleEnd,
		&plan.AutoRenew,
	)
	if err != nil {
		return PlanStatus{}, fmt.Errorf("failed to get plan status: %w", err)
	}

	if billingCycleEnd.Valid {
		plan.BillingCycleEnd = &billingCycleEnd.Time
	}

	// Set plan display name and features
	switch plan.CurrentPlan {
	case "free":
		plan.PlanDisplayName = "Free Plan"
		plan.Features = []string{"2 free conversions", "Basic support"}
	case "basic":
		plan.PlanDisplayName = "Basic Plan"
		plan.Features = []string{"50 conversions/month", "Priority support", "Advanced styles"}
	case "premium":
		plan.PlanDisplayName = "Premium Plan"
		plan.Features = []string{"200 conversions/month", "Premium support", "All styles", "Priority processing"}
	case "enterprise":
		plan.PlanDisplayName = "Enterprise Plan"
		plan.Features = []string{"Unlimited conversions", "24/7 support", "Custom styles", "API access"}
	}

	// Check if upgrade is available
	plan.UpgradeAvailable = plan.CurrentPlan != "enterprise"

	// Get available plans
	plan.AvailablePlans = s.getAvailablePlans(plan.CurrentPlan)

	return plan, nil
}

// ShouldShowUpgradePrompt determines if upgrade prompt should be shown
func (s *StoreImpl) ShouldShowUpgradePrompt(ctx context.Context, userID string) (*UpgradePrompt, error) {
	quotaStatus, err := s.GetQuotaStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Don't show prompt if user has plenty of quota
	if quotaStatus.TotalConversionsRemaining > 5 {
		return nil, nil
	}

	urgencyLevel := getUrgencyLevel(quotaStatus.TotalConversionsRemaining)

	var title, message, actionText string
	var recommendedPlan string

	if quotaStatus.TotalConversionsRemaining <= 0 {
		title = "Quota Exceeded"
		message = "You've used all your conversions. Upgrade to continue using AI Styler."
		actionText = "Upgrade Now"
		recommendedPlan = "basic"
	} else if quotaStatus.TotalConversionsRemaining <= 2 {
		title = "Low Quota Warning"
		message = "You're running low on conversions. Consider upgrading for more capacity."
		actionText = "View Plans"
		recommendedPlan = "premium"
	} else {
		title = "Upgrade Available"
		message = "Unlock more features and conversions with our premium plans."
		actionText = "Learn More"
		recommendedPlan = "basic"
	}

	return &UpgradePrompt{
		ShowPrompt:      true,
		Title:           title,
		Message:         message,
		ActionText:      actionText,
		ActionURL:       "/plans",
		UrgencyLevel:    urgencyLevel,
		RemainingQuota:  quotaStatus.TotalConversionsRemaining,
		RecommendedPlan: recommendedPlan,
	}, nil
}

// GetDashboardStatistics retrieves dashboard statistics
func (s *StoreImpl) GetDashboardStatistics(ctx context.Context, userID string) (DashboardStatistics, error) {
	// Get conversion statistics
	statsQuery := `
		SELECT 
			COUNT(*) as total_conversions,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_conversions,
			AVG(CASE WHEN processing_time_ms IS NOT NULL THEN processing_time_ms END) as avg_processing_time,
			SUM(CASE WHEN processing_time_ms IS NOT NULL THEN processing_time_ms ELSE 0 END) as total_processing_time
		FROM user_conversions 
		WHERE user_id = $1
	`

	var stats DashboardStatistics
	var avgProcessingTime sql.NullFloat64

	var successfulConversions int
	err := s.db.QueryRowContext(ctx, statsQuery, userID).Scan(
		&stats.TotalConversions,
		&successfulConversions,
		&avgProcessingTime,
		&stats.TotalProcessingTime,
	)
	if err != nil {
		return DashboardStatistics{}, fmt.Errorf("failed to get conversion statistics: %w", err)
	}

	// Calculate success rate
	if stats.TotalConversions > 0 {
		stats.SuccessfulRate = float64(successfulConversions) / float64(stats.TotalConversions) * 100
	}

	if avgProcessingTime.Valid {
		avg := int(avgProcessingTime.Float64)
		stats.AverageProcessingTime = &avg
	}

	// Get favorite and most used styles
	styleQuery := `
		SELECT 
			style_name,
			COUNT(*) as usage_count
		FROM user_conversions 
		WHERE user_id = $1 AND status = 'completed'
		GROUP BY style_name
		ORDER BY usage_count DESC
		LIMIT 2
	`

	rows, err := s.db.QueryContext(ctx, styleQuery, userID)
	if err != nil {
		fmt.Printf("Failed to get style statistics: %v\n", err)
	} else {
		defer rows.Close()

		var styles []string
		for rows.Next() {
			var styleName string
			var usageCount int
			rows.Scan(&styleName, &usageCount)
			styles = append(styles, styleName)
		}

		if len(styles) > 0 {
			stats.FavoriteStyle = &styles[0]
		}
		if len(styles) > 1 {
			stats.MostUsedStyle = &styles[1]
		}
	}

	// Get conversion trends (last 7 days)
	trendsQuery := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as count
		FROM user_conversions 
		WHERE user_id = $1 AND created_at >= NOW() - INTERVAL '7 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err = s.db.QueryContext(ctx, trendsQuery, userID)
	if err != nil {
		fmt.Printf("Failed to get conversion trends: %v\n", err)
		stats.ConversionTrends = []TrendData{}
	} else {
		defer rows.Close()

		var trends []TrendData
		for rows.Next() {
			var trend TrendData
			var date time.Time
			rows.Scan(&date, &trend.Count)
			trend.Date = date.Format("2006-01-02")
			trends = append(trends, trend)
		}
		stats.ConversionTrends = trends
	}

	return stats, nil
}

// GetRecentActivity retrieves recent user activity
func (s *StoreImpl) GetRecentActivity(ctx context.Context, userID string, limit int) ([]RecentActivity, error) {
	// This would typically query an activity log table
	// For now, we'll create some mock data based on conversions

	query := `
		SELECT 
			id,
			'conversion' as type,
			'Image Conversion' as title,
			'Converted image with ' || style_name as description,
			status,
			created_at
		FROM user_conversions 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return []RecentActivity{}, fmt.Errorf("failed to get recent activity: %w", err)
	}
	defer rows.Close()

	var activities []RecentActivity
	for rows.Next() {
		var activity RecentActivity
		err := rows.Scan(
			&activity.ID,
			&activity.Type,
			&activity.Title,
			&activity.Description,
			&activity.Status,
			&activity.CreatedAt,
		)
		if err != nil {
			return []RecentActivity{}, fmt.Errorf("failed to scan activity: %w", err)
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// CheckQuota checks if user can perform conversions
func (s *StoreImpl) CheckQuota(ctx context.Context, userID string) (QuotaCheckResponse, error) {
	quotaStatus, err := s.GetQuotaStatus(ctx, userID)
	if err != nil {
		return QuotaCheckResponse{}, err
	}

	canConvert := quotaStatus.TotalConversionsRemaining > 0
	upgradePrompt, _ := s.ShouldShowUpgradePrompt(ctx, userID)

	recommendedAction := "continue_normal"
	if !canConvert {
		recommendedAction = "upgrade_plan"
	} else if quotaStatus.TotalConversionsRemaining <= 2 {
		recommendedAction = "consider_upgrade"
	}

	return QuotaCheckResponse{
		CanConvert:        canConvert,
		QuotaStatus:       quotaStatus,
		UpgradePrompt:     upgradePrompt,
		RecommendedAction: recommendedAction,
	}, nil
}

// Cache operations
func (s *StoreImpl) GetCachedDashboardData(ctx context.Context, userID string) (*DashboardData, error) {
	// This would implement actual caching logic
	return nil, fmt.Errorf("cache not implemented")
}

func (s *StoreImpl) SetCachedDashboardData(ctx context.Context, userID string, data DashboardData, ttl int) error {
	// This would implement actual caching logic
	return nil
}

func (s *StoreImpl) InvalidateDashboardCache(ctx context.Context, userID string) error {
	// This would implement actual caching logic
	return nil
}

// Helper functions
func (s *StoreImpl) getUserInfo(ctx context.Context, userID string) (UserInfo, error) {
	query := `
		SELECT 
			id,
			name,
			avatar_url,
			phone,
			is_phone_verified,
			role,
			created_at
		FROM users 
		WHERE id = $1
	`

	var userInfo UserInfo
	var name, avatarURL sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&userInfo.ID,
		&name,
		&avatarURL,
		&userInfo.Phone,
		&userInfo.IsPhoneVerified,
		&userInfo.Role,
		&userInfo.CreatedAt,
	)
	if err != nil {
		return UserInfo{}, fmt.Errorf("failed to get user info: %w", err)
	}

	if name.Valid {
		userInfo.Name = &name.String
	}
	if avatarURL.Valid {
		userInfo.AvatarURL = &avatarURL.String
	}

	return userInfo, nil
}

func (s *StoreImpl) getAvailablePlans(currentPlan string) []PlanInfo {
	plans := []PlanInfo{
		{
			ID:                      "free",
			Name:                    "free",
			DisplayName:             "Free Plan",
			Description:             "Perfect for trying out AI Styler",
			PricePerMonthCents:      0,
			MonthlyConversionsLimit: 2,
			Features:                []string{"2 free conversions", "Basic support"},
			IsCurrent:               currentPlan == "free",
		},
		{
			ID:                      "basic",
			Name:                    "basic",
			DisplayName:             "Basic Plan",
			Description:             "Great for regular users",
			PricePerMonthCents:      50000, // 50,000 Rials
			MonthlyConversionsLimit: 50,
			Features:                []string{"50 conversions/month", "Priority support", "Advanced styles"},
			IsRecommended:           currentPlan == "free",
			IsCurrent:               currentPlan == "basic",
		},
		{
			ID:                      "premium",
			Name:                    "premium",
			DisplayName:             "Premium Plan",
			Description:             "For power users and professionals",
			PricePerMonthCents:      150000, // 150,000 Rials
			MonthlyConversionsLimit: 200,
			Features:                []string{"200 conversions/month", "Premium support", "All styles", "Priority processing"},
			IsRecommended:           currentPlan == "basic",
			IsCurrent:               currentPlan == "premium",
		},
		{
			ID:                      "enterprise",
			Name:                    "enterprise",
			DisplayName:             "Enterprise Plan",
			Description:             "For businesses and teams",
			PricePerMonthCents:      500000, // 500,000 Rials
			MonthlyConversionsLimit: -1,     // Unlimited
			Features:                []string{"Unlimited conversions", "24/7 support", "Custom styles", "API access"},
			IsCurrent:               currentPlan == "enterprise",
		},
	}

	// Filter out current plan if it's not free
	if currentPlan != "free" {
		var filteredPlans []PlanInfo
		for _, plan := range plans {
			if plan.Name != currentPlan {
				filteredPlans = append(filteredPlans, plan)
			}
		}
		return filteredPlans
	}

	return plans
}
