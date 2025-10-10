package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// postgresStore implements the Store interface using PostgreSQL
type postgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *sql.DB) Store {
	return &postgresStore{db: db}
}

// GetProfile retrieves a user's profile
func (s *postgresStore) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	query := `
		SELECT id, phone, name, avatar_url, bio, is_phone_verified, is_active,
		       last_login_at, created_at, updated_at
		FROM users 
		WHERE id = $1`

	var profile UserProfile
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID, &profile.Phone, &profile.Name, &profile.AvatarURL, &profile.Bio,
		&profile.IsPhoneVerified, &profile.IsActive, &profile.LastLoginAt,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserProfile{}, errors.New("user not found")
		}
		return UserProfile{}, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// GetUserByID retrieves a user by ID (alias for GetProfile)
func (s *postgresStore) GetUserByID(ctx context.Context, userID string) (UserProfile, error) {
	return s.GetProfile(ctx, userID)
}

// UpdateProfile updates a user's profile
func (s *postgresStore) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UserProfile, error) {
	// Build dynamic query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.AvatarURL != nil {
		setParts = append(setParts, fmt.Sprintf("avatar_url = $%d", argIndex))
		args = append(args, *req.AvatarURL)
		argIndex++
	}
	if req.Bio != nil {
		setParts = append(setParts, fmt.Sprintf("bio = $%d", argIndex))
		args = append(args, *req.Bio)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetProfile(ctx, userID)
	}

	query := fmt.Sprintf(`
		UPDATE users 
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING id, phone, name, avatar_url, bio, is_phone_verified, is_active,
		          last_login_at, created_at, updated_at`,
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf(`
			UPDATE users 
			SET %s, updated_at = NOW()
			WHERE id = $%d
			RETURNING id, phone, name, avatar_url, bio, is_phone_verified, is_active,
			          last_login_at, created_at, updated_at`,
			fmt.Sprintf("%s", setParts[i]), argIndex)
	}

	args = append(args, userID)

	var profile UserProfile
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&profile.ID, &profile.Phone, &profile.Name, &profile.AvatarURL, &profile.Bio,
		&profile.IsPhoneVerified, &profile.IsActive, &profile.LastLoginAt,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserProfile{}, errors.New("user not found")
		}
		return UserProfile{}, fmt.Errorf("failed to update profile: %w", err)
	}

	return profile, nil
}

// GetConversionHistory retrieves a user's conversion history
func (s *postgresStore) GetConversionHistory(ctx context.Context, userID string, req ConversionHistoryRequest) (ConversionHistoryResponse, error) {
	query := `
		SELECT id, conversion_type, input_file_url, output_file_url, style_name,
		       status, error_message, processing_time_ms, file_size_bytes,
		       created_at, completed_at
		FROM user_conversions 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	offset := (req.Page - 1) * req.PageSize
	rows, err := s.db.QueryContext(ctx, query, userID, req.PageSize, offset)
	if err != nil {
		return ConversionHistoryResponse{}, fmt.Errorf("failed to get conversion history: %w", err)
	}
	defer rows.Close()

	var conversions []UserConversion
	for rows.Next() {
		var conv UserConversion
		err := rows.Scan(
			&conv.ID, &conv.ConversionType, &conv.InputFileURL, &conv.OutputFileURL,
			&conv.StyleName, &conv.Status, &conv.ErrorMessage, &conv.ProcessingTimeMs,
			&conv.FileSizeBytes, &conv.CreatedAt, &conv.CompletedAt,
		)
		if err != nil {
			return ConversionHistoryResponse{}, fmt.Errorf("failed to scan conversion: %w", err)
		}
		conversions = append(conversions, conv)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM user_conversions WHERE user_id = $1`
	var total int
	err = s.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return ConversionHistoryResponse{}, fmt.Errorf("failed to get total count: %w", err)
	}

	return ConversionHistoryResponse{
		Conversions: conversions,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}, nil
}

// CanUserConvert checks if user can perform a conversion
func (s *postgresStore) CanUserConvert(ctx context.Context, userID string, conversionType string) (bool, error) {
	query := `SELECT can_user_convert($1, $2)`
	var canConvert bool
	err := s.db.QueryRowContext(ctx, query, userID, conversionType).Scan(&canConvert)
	if err != nil {
		return false, fmt.Errorf("failed to check conversion quota: %w", err)
	}
	return canConvert, nil
}

// CreateConversion creates a new conversion
func (s *postgresStore) CreateConversion(ctx context.Context, userID string, req CreateConversionRequest) (UserConversion, error) {
	query := `
		SELECT record_conversion($1, $2, $3, $4)`

	var conversionID string
	err := s.db.QueryRowContext(ctx, query, userID, req.Type, req.InputFileURL, req.StyleName).Scan(&conversionID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "P0001" {
			return UserConversion{}, errors.New("conversion quota exceeded")
		}
		return UserConversion{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	// Get the created conversion
	conversion, err := s.GetConversion(ctx, conversionID)
	if err != nil {
		return UserConversion{}, fmt.Errorf("failed to get created conversion: %w", err)
	}

	return conversion, nil
}

// GetConversion retrieves a specific conversion
func (s *postgresStore) GetConversion(ctx context.Context, conversionID string) (UserConversion, error) {
	query := `
		SELECT id, user_id, conversion_type, input_file_url, output_file_url,
		       style_name, status, error_message, processing_time_ms, file_size_bytes,
		       created_at, completed_at
		FROM user_conversions 
		WHERE id = $1`

	var conv UserConversion
	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conv.ID, &conv.UserID, &conv.ConversionType, &conv.InputFileURL, &conv.OutputFileURL,
		&conv.StyleName, &conv.Status, &conv.ErrorMessage, &conv.ProcessingTimeMs,
		&conv.FileSizeBytes, &conv.CreatedAt, &conv.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserConversion{}, errors.New("conversion not found")
		}
		return UserConversion{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	return conv, nil
}

// UpdateConversion updates a conversion
func (s *postgresStore) UpdateConversion(ctx context.Context, conversionID string, req UpdateConversionRequest) (UserConversion, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}
	if req.OutputFileURL != nil {
		setParts = append(setParts, fmt.Sprintf("output_file_url = $%d", argIndex))
		args = append(args, *req.OutputFileURL)
		argIndex++
	}
	if req.ErrorMessage != nil {
		setParts = append(setParts, fmt.Sprintf("error_message = $%d", argIndex))
		args = append(args, *req.ErrorMessage)
		argIndex++
	}
	if req.ProcessingTimeMs != nil {
		setParts = append(setParts, fmt.Sprintf("processing_time_ms = $%d", argIndex))
		args = append(args, *req.ProcessingTimeMs)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetConversion(ctx, conversionID)
	}

	// Add completed_at if status is completed
	if req.Status != nil && *req.Status == ConversionStatusCompleted {
		setParts = append(setParts, "completed_at = NOW()")
	}

	query := fmt.Sprintf(`
		UPDATE user_conversions 
		SET %s
		WHERE id = $%d
		RETURNING id, user_id, conversion_type, input_file_url, output_file_url,
		          style_name, status, error_message, processing_time_ms, file_size_bytes,
		          created_at, completed_at`,
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf(`
			UPDATE user_conversions 
			SET %s
			WHERE id = $%d
			RETURNING id, user_id, conversion_type, input_file_url, output_file_url,
			          style_name, status, error_message, processing_time_ms, file_size_bytes,
			          created_at, completed_at`,
			fmt.Sprintf("%s", setParts[i]), argIndex)
	}

	args = append(args, conversionID)

	var conv UserConversion
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&conv.ID, &conv.UserID, &conv.ConversionType, &conv.InputFileURL, &conv.OutputFileURL,
		&conv.StyleName, &conv.Status, &conv.ErrorMessage, &conv.ProcessingTimeMs,
		&conv.FileSizeBytes, &conv.CreatedAt, &conv.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserConversion{}, errors.New("conversion not found")
		}
		return UserConversion{}, fmt.Errorf("failed to update conversion: %w", err)
	}

	return conv, nil
}

// GetQuotaStatus retrieves current quota status for a user
func (s *postgresStore) GetQuotaStatus(ctx context.Context, userID string) (QuotaStatus, error) {
	query := `SELECT * FROM get_user_quota_status($1)`

	var status QuotaStatus
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&status.FreeConversionsRemaining, &status.PaidConversionsRemaining,
		&status.TotalConversionsRemaining, &status.PlanName, &status.MonthlyLimit,
	)
	if err != nil {
		return QuotaStatus{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	return status, nil
}

// GetUserPlan retrieves a user's current plan
func (s *postgresStore) GetUserPlan(ctx context.Context, userID string) (UserPlan, error) {
	query := `
		SELECT up.id, up.user_id, up.plan_id, pp.name, pp.display_name, pp.description,
		       pp.price_per_month_cents, pp.monthly_conversions_limit,
		       pp.features, up.status, up.created_at, up.updated_at
		FROM user_plans up
		JOIN payment_plans pp ON up.plan_id = pp.id
		WHERE up.user_id = $1 AND up.status = 'active'`

	var plan UserPlan
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&plan.ID, &plan.UserID, &plan.PlanID, &plan.PlanName, &plan.DisplayName, &plan.Description,
		&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit,
		pq.Array(&plan.Features), &plan.Status, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return free plan if no active plan
			return UserPlan{
				ID:                      "free-plan-" + userID,
				PlanID:                  "free",
				UserID:                  userID,
				PlanName:                "free",
				DisplayName:             "Free Plan",
				Description:             "Basic free plan",
				PricePerMonthCents:      0,
				MonthlyConversionsLimit: 2,
				Features:                []string{"2 free conversions per month"},
				Status:                  "active",
			}, nil
		}
		return UserPlan{}, fmt.Errorf("failed to get user plan: %w", err)
	}

	return plan, nil
}

// CreateUserPlan creates a new plan for a user
func (s *postgresStore) CreateUserPlan(ctx context.Context, userID string, planName string) (UserPlan, error) {
	// Get plan details
	planQuery := `SELECT id, name, display_name, description, price_per_month_cents, 
	                     monthly_conversions_limit, features
	              FROM payment_plans WHERE name = $1 AND is_active = true`

	var planID string
	var plan UserPlan
	err := s.db.QueryRowContext(ctx, planQuery, planName).Scan(
		&planID, &plan.PlanName, &plan.DisplayName, &plan.Description,
		&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit, pq.Array(&plan.Features),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserPlan{}, errors.New("plan not found")
		}
		return UserPlan{}, fmt.Errorf("failed to get plan details: %w", err)
	}

	// Set the PlanID field
	plan.PlanID = planID

	// Create user plan
	insertQuery := `
		INSERT INTO user_plans (user_id, plan_id, status)
		VALUES ($1, $2, 'active')
		RETURNING id, created_at, updated_at`

	err = s.db.QueryRowContext(ctx, insertQuery, userID, planID).Scan(
		&plan.ID, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return UserPlan{}, errors.New("user already has an active plan")
		}
		return UserPlan{}, fmt.Errorf("failed to create user plan: %w", err)
	}

	plan.UserID = userID
	plan.Status = "active"

	return plan, nil
}

// UpdateUserPlan updates a user's plan status
func (s *postgresStore) UpdateUserPlan(ctx context.Context, planID string, status string) (UserPlan, error) {
	query := `
		UPDATE user_plans 
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, user_id, plan_id, status, created_at, updated_at`

	var plan UserPlan
	err := s.db.QueryRowContext(ctx, query, status, planID).Scan(
		&plan.ID, &plan.UserID, &plan.PlanID, &plan.Status, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserPlan{}, errors.New("user plan not found")
		}
		return UserPlan{}, fmt.Errorf("failed to update user plan: %w", err)
	}

	// Get plan details
	planQuery := `SELECT name, display_name, description, price_per_month_cents, 
	                     monthly_conversions_limit, features
	              FROM payment_plans WHERE id = $1`

	err = s.db.QueryRowContext(ctx, planQuery, plan.PlanID).Scan(
		&plan.PlanName, &plan.DisplayName, &plan.Description,
		&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit, pq.Array(&plan.Features),
	)
	if err != nil {
		return UserPlan{}, fmt.Errorf("failed to get plan details: %w", err)
	}

	return plan, nil
}

// RecordConversion records a conversion for quota tracking
func (s *postgresStore) RecordConversion(ctx context.Context, userID string, conversionType string, inputFileURL string, styleName string) (string, error) {
	// Create conversion record
	conversionReq := CreateConversionRequest{
		Type:         conversionType,
		InputFileURL: inputFileURL,
		StyleName:    styleName,
	}

	conversion, err := s.CreateConversion(ctx, userID, conversionReq)
	if err != nil {
		return "", fmt.Errorf("failed to record conversion: %w", err)
	}

	// Update quota usage
	updateQuery := `
		UPDATE conversion_quotas 
		SET conversions_used = conversions_used + 1,
		    updated_at = NOW()
		WHERE user_id = $1 AND year = EXTRACT(YEAR FROM NOW()) AND month = EXTRACT(MONTH FROM NOW())`

	_, err = s.db.ExecContext(ctx, updateQuery, userID)
	if err != nil {
		return "", fmt.Errorf("failed to update quota: %w", err)
	}

	return conversion.ID, nil
}
