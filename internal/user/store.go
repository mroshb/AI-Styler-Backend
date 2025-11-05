package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// DBStore implements the Store interface using PostgreSQL
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new database store
func NewDBStore(db *sql.DB) Store {
	return &DBStore{db: db}
}

// GetProfile retrieves a user's profile
func (s *DBStore) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	query := `
		SELECT id, phone, name, avatar_url, bio, role, is_phone_verified, is_active,
		       free_conversions_used, free_conversions_limit, created_at, updated_at
		FROM users 
		WHERE id = $1`

	var profile UserProfile
	var name sql.NullString
	var avatarURL sql.NullString
	var bio sql.NullString
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID, &profile.Phone, &name, &avatarURL, &bio,
		&profile.Role, &profile.IsPhoneVerified, &profile.IsActive, &profile.FreeConversionsUsed,
		&profile.FreeConversionsLimit, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserProfile{}, fmt.Errorf("user not found")
		}
		return UserProfile{}, err
	}

	// Handle nullable fields
	if name.Valid {
		profile.Name = &name.String
	}
	if avatarURL.Valid {
		profile.AvatarURL = &avatarURL.String
	}
	if bio.Valid {
		profile.Bio = &bio.String
	}

	return profile, nil
}

// UpdateProfile updates a user's profile
func (s *DBStore) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error) {
	query := `
		UPDATE users 
		SET name = COALESCE($2, name),
		    avatar_url = COALESCE($3, avatar_url),
		    bio = COALESCE($4, bio),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, phone, name, avatar_url, bio, role, is_phone_verified, is_active,
		          free_conversions_used, free_conversions_limit, created_at, updated_at`

	var profile UserProfile
	var name sql.NullString
	var avatarURL sql.NullString
	var bio sql.NullString
	err := s.db.QueryRowContext(ctx, query, userID, req.Name, req.AvatarURL, req.Bio).Scan(
		&profile.ID, &profile.Phone, &name, &avatarURL, &bio,
		&profile.Role, &profile.IsPhoneVerified, &profile.IsActive, &profile.FreeConversionsUsed,
		&profile.FreeConversionsLimit, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserProfile{}, fmt.Errorf("user not found")
		}
		return UserProfile{}, err
	}

	// Handle nullable fields
	if name.Valid {
		profile.Name = &name.String
	}
	if avatarURL.Valid {
		profile.AvatarURL = &avatarURL.String
	}
	if bio.Valid {
		profile.Bio = &bio.String
	}

	return profile, nil
}

// CreateConversion creates a new conversion
func (s *DBStore) CreateConversion(ctx context.Context, userID string, req CreateConversionRequest) (UserConversion, error) {
	query := `
		INSERT INTO user_conversions (user_id, conversion_type, input_file_url, style_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, conversion_type, input_file_url, output_file_url, style_name,
		          status, error_message, processing_time_ms, file_size_bytes, created_at, completed_at`

	var conversion UserConversion
	err := s.db.QueryRowContext(ctx, query, userID, req.Type, req.InputFileURL, req.StyleName).Scan(
		&conversion.ID, &conversion.UserID, &conversion.ConversionType, &conversion.InputFileURL,
		&conversion.OutputFileURL, &conversion.StyleName, &conversion.Status, &conversion.ErrorMessage,
		&conversion.ProcessingTimeMs, &conversion.FileSizeBytes, &conversion.CreatedAt, &conversion.CompletedAt,
	)
	if err != nil {
		return UserConversion{}, err
	}

	return conversion, nil
}

// GetConversion retrieves a specific conversion
func (s *DBStore) GetConversion(ctx context.Context, conversionID string) (UserConversion, error) {
	query := `
		SELECT id, user_id, conversion_type, input_file_url, output_file_url, style_name,
		       status, error_message, processing_time_ms, file_size_bytes, created_at, completed_at
		FROM user_conversions 
		WHERE id = $1`

	var conversion UserConversion
	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conversion.ID, &conversion.UserID, &conversion.ConversionType, &conversion.InputFileURL,
		&conversion.OutputFileURL, &conversion.StyleName, &conversion.Status, &conversion.ErrorMessage,
		&conversion.ProcessingTimeMs, &conversion.FileSizeBytes, &conversion.CreatedAt, &conversion.CompletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserConversion{}, fmt.Errorf("conversion not found")
		}
		return UserConversion{}, err
	}

	return conversion, nil
}

// UpdateConversion updates a conversion
func (s *DBStore) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) (UserConversion, error) {
	query := `
		UPDATE user_conversions 
		SET output_file_url = COALESCE($2, output_file_url),
		    status = COALESCE($3, status),
		    error_message = COALESCE($4, error_message),
		    processing_time_ms = COALESCE($5, processing_time_ms),
		    file_size_bytes = COALESCE($6, file_size_bytes),
		    completed_at = CASE WHEN $3 = 'completed' OR $3 = 'failed' THEN NOW() ELSE completed_at END
		WHERE id = $1
		RETURNING id, user_id, conversion_type, input_file_url, output_file_url, style_name,
		          status, error_message, processing_time_ms, file_size_bytes, created_at, completed_at`

	var conversion UserConversion
	err := s.db.QueryRowContext(ctx, query, conversionID, req.OutputFileURL, req.Status,
		req.ErrorMessage, req.ProcessingTimeMs, req.FileSizeBytes).Scan(
		&conversion.ID, &conversion.UserID, &conversion.ConversionType, &conversion.InputFileURL,
		&conversion.OutputFileURL, &conversion.StyleName, &conversion.Status, &conversion.ErrorMessage,
		&conversion.ProcessingTimeMs, &conversion.FileSizeBytes, &conversion.CreatedAt, &conversion.CompletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserConversion{}, fmt.Errorf("conversion not found")
		}
		return UserConversion{}, err
	}

	return conversion, nil
}

// GetConversionHistory retrieves conversion history with pagination
func (s *DBStore) GetConversionHistory(ctx context.Context, userID string, req ConversionHistoryRequest) (ConversionHistoryResponse, error) {
	// Build WHERE clause
	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.Type != "" {
		whereClause += fmt.Sprintf(" AND conversion_type = $%d", argIndex)
		args = append(args, req.Type)
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_conversions %s", whereClause)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return ConversionHistoryResponse{}, err
	}

	// Calculate pagination
	offset := (req.Page - 1) * req.PageSize
	totalPages := (total + req.PageSize - 1) / req.PageSize

	// Get conversions
	query := fmt.Sprintf(`
		SELECT id, user_id, conversion_type, input_file_url, output_file_url, style_name,
		       status, error_message, processing_time_ms, file_size_bytes, created_at, completed_at
		FROM user_conversions 
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, req.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ConversionHistoryResponse{}, err
	}
	defer rows.Close()

	var conversions []UserConversion
	for rows.Next() {
		var conversion UserConversion
		err := rows.Scan(
			&conversion.ID, &conversion.UserID, &conversion.ConversionType, &conversion.InputFileURL,
			&conversion.OutputFileURL, &conversion.StyleName, &conversion.Status, &conversion.ErrorMessage,
			&conversion.ProcessingTimeMs, &conversion.FileSizeBytes, &conversion.CreatedAt, &conversion.CompletedAt,
		)
		if err != nil {
			return ConversionHistoryResponse{}, err
		}
		conversions = append(conversions, conversion)
	}

	return ConversionHistoryResponse{
		Conversions: conversions,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
	}, nil
}

// GetUserPlan retrieves a user's current plan
func (s *DBStore) GetUserPlan(ctx context.Context, userID string) (UserPlan, error) {
	query := `
		SELECT id, user_id, plan_name, status, monthly_conversions_limit, conversions_used_this_month,
		       price_per_month_cents, billing_cycle_start_date, billing_cycle_end_date, auto_renew,
		       created_at, updated_at, expires_at
		FROM user_plans 
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1`

	var plan UserPlan
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&plan.ID, &plan.UserID, &plan.PlanName, &plan.Status, &plan.MonthlyConversionsLimit,
		&plan.ConversionsUsedThisMonth, &plan.PricePerMonthCents, &plan.BillingCycleStartDate,
		&plan.BillingCycleEndDate, &plan.AutoRenew, &plan.CreatedAt, &plan.UpdatedAt, &plan.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default free plan
			return UserPlan{
				ID:                       "",
				UserID:                   userID,
				PlanName:                 PlanFree,
				Status:                   PlanStatusActive,
				MonthlyConversionsLimit:  0,
				ConversionsUsedThisMonth: 0,
				PricePerMonthCents:       0,
				AutoRenew:                true,
				CreatedAt:                time.Now(),
				UpdatedAt:                time.Now(),
			}, nil
		}
		return UserPlan{}, err
	}

	return plan, nil
}

// CreateUserPlan creates a new plan for a user
func (s *DBStore) CreateUserPlan(ctx context.Context, userID string, planName string) (UserPlan, error) {
	// Set plan limits based on plan name
	var monthlyLimit, priceCents int
	switch planName {
	case PlanFree:
		monthlyLimit = 0
		priceCents = 0
	case PlanBasic:
		monthlyLimit = 10
		priceCents = 999
	case PlanPremium:
		monthlyLimit = 50
		priceCents = 2999
	case PlanEnterprise:
		monthlyLimit = 200
		priceCents = 9999
	default:
		return UserPlan{}, fmt.Errorf("invalid plan name: %s", planName)
	}

	// Cancel existing active plans
	_, err := s.db.ExecContext(ctx, "UPDATE user_plans SET status = 'cancelled' WHERE user_id = $1 AND status = 'active'", userID)
	if err != nil {
		return UserPlan{}, err
	}

	// Create new plan
	query := `
		INSERT INTO user_plans (user_id, plan_name, status, monthly_conversions_limit, price_per_month_cents, auto_renew)
		VALUES ($1, $2, 'active', $3, $4, true)
		RETURNING id, user_id, plan_name, status, monthly_conversions_limit, conversions_used_this_month,
		          price_per_month_cents, billing_cycle_start_date, billing_cycle_end_date, auto_renew,
		          created_at, updated_at, expires_at`

	var plan UserPlan
	err = s.db.QueryRowContext(ctx, query, userID, planName, monthlyLimit, priceCents).Scan(
		&plan.ID, &plan.UserID, &plan.PlanName, &plan.Status, &plan.MonthlyConversionsLimit,
		&plan.ConversionsUsedThisMonth, &plan.PricePerMonthCents, &plan.BillingCycleStartDate,
		&plan.BillingCycleEndDate, &plan.AutoRenew, &plan.CreatedAt, &plan.UpdatedAt, &plan.ExpiresAt,
	)
	if err != nil {
		return UserPlan{}, err
	}

	return plan, nil
}

// UpdateUserPlan updates a user's plan status
func (s *DBStore) UpdateUserPlan(ctx context.Context, planID string, status string) (UserPlan, error) {
	query := `
		UPDATE user_plans 
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, plan_name, status, monthly_conversions_limit, conversions_used_this_month,
		          price_per_month_cents, billing_cycle_start_date, billing_cycle_end_date, auto_renew,
		          created_at, updated_at, expires_at`

	var plan UserPlan
	err := s.db.QueryRowContext(ctx, query, planID, status).Scan(
		&plan.ID, &plan.UserID, &plan.PlanName, &plan.Status, &plan.MonthlyConversionsLimit,
		&plan.ConversionsUsedThisMonth, &plan.PricePerMonthCents, &plan.BillingCycleStartDate,
		&plan.BillingCycleEndDate, &plan.AutoRenew, &plan.CreatedAt, &plan.UpdatedAt, &plan.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserPlan{}, fmt.Errorf("plan not found")
		}
		return UserPlan{}, err
	}

	return plan, nil
}

// GetQuotaStatus retrieves current quota status for a user
func (s *DBStore) GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error) {
	query := `SELECT * FROM get_user_quota_status($1)`

	var status QuotaStatus
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&status.FreeConversionsRemaining, &status.PaidConversionsRemaining,
		&status.TotalConversionsRemaining, &status.PlanName, &status.MonthlyLimit,
	)
	if err != nil {
		return QuotaStatus{}, err
	}

	return status, nil
}

// CanUserConvert checks if a user can perform a conversion
func (s *DBStore) CanUserConvert(ctx context.Context, userID string, conversionType string) (bool, error) {
	query := `SELECT can_user_convert($1, $2)`

	var canConvert bool
	err := s.db.QueryRowContext(ctx, query, userID, conversionType).Scan(&canConvert)
	if err != nil {
		return false, err
	}

	return canConvert, nil
}

// RecordConversion records a conversion using the database function
func (s *DBStore) RecordConversion(ctx context.Context, userID string, conversionType string, inputFileURL string, styleName string) (string, error) {
	query := `SELECT record_conversion($1, $2, $3, $4)`

	var conversionID string
	err := s.db.QueryRowContext(ctx, query, userID, conversionType, inputFileURL, styleName).Scan(&conversionID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "P0001" {
			return "", fmt.Errorf("quota exceeded")
		}
		return "", err
	}

	return conversionID, nil
}

// GetUserByID retrieves a user by ID (for compatibility)
func (s *DBStore) GetUserByID(ctx context.Context, userID string) (UserProfile, error) {
	return s.GetProfile(ctx, userID)
}
