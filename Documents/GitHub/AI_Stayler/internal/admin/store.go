package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DBStore implements the Store interface using PostgreSQL
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new database store
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// User operations

// GetUsers retrieves a list of users with pagination and filtering
func (s *DBStore) GetUsers(ctx context.Context, req UserListRequest) (UserListResponse, error) {
	query := `
		SELECT 
			u.id, u.phone, u.name, u.avatar_url, u.bio, u.role, 
			u.is_phone_verified, u.free_conversions_used, u.free_conversions_limit,
			u.created_at, u.updated_at, u.is_active,
			COALESCE(s.last_used_at, u.created_at) as last_login_at
		FROM users u
		LEFT JOIN (
			SELECT DISTINCT ON (user_id) user_id, last_used_at
			FROM sessions 
			WHERE revoked_at IS NULL
			ORDER BY user_id, last_used_at DESC
		) s ON u.id = s.user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.Role != "" {
		query += " AND u.role = $" + strconv.Itoa(argIndex)
		args = append(args, req.Role)
		argIndex++
	}

	if req.Search != "" {
		query += fmt.Sprintf(" AND (u.phone ILIKE $%d OR u.name ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + req.Search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	if req.IsActive != nil {
		query += " AND u.is_active = $" + strconv.Itoa(argIndex)
		args = append(args, *req.IsActive)
		argIndex++
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT u.id, u.phone, u.name, u.avatar_url, u.bio, u.role, u.is_phone_verified, u.free_conversions_used, u.free_conversions_limit, u.created_at, u.updated_at, u.is_active, COALESCE(s.last_used_at, u.created_at) as last_login_at", "SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, "LEFT JOIN (", "", 1)
	countQuery = strings.Replace(countQuery, ") s ON u.id = s.user_id", "", 1)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return UserListResponse{}, fmt.Errorf("failed to count users: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY u.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return UserListResponse{}, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var user AdminUser
		var lastLoginAt sql.NullTime
		err := rows.Scan(
			&user.ID, &user.Phone, &user.Name, &user.AvatarURL, &user.Bio, &user.Role,
			&user.IsPhoneVerified, &user.FreeConversionsUsed, &user.FreeConversionsLimit,
			&user.CreatedAt, &user.UpdatedAt, &user.IsActive, &lastLoginAt,
		)
		if err != nil {
			return UserListResponse{}, fmt.Errorf("failed to scan user: %w", err)
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return UserListResponse{}, fmt.Errorf("error iterating users: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return UserListResponse{
		Users:      users,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUser retrieves a specific user by ID
func (s *DBStore) GetUser(ctx context.Context, userID string) (AdminUser, error) {
	query := `
		SELECT 
			u.id, u.phone, u.name, u.avatar_url, u.bio, u.role, 
			u.is_phone_verified, u.free_conversions_used, u.free_conversions_limit,
			u.created_at, u.updated_at, u.is_active,
			COALESCE(s.last_used_at, u.created_at) as last_login_at
		FROM users u
		LEFT JOIN (
			SELECT DISTINCT ON (user_id) user_id, last_used_at
			FROM sessions 
			WHERE revoked_at IS NULL
			ORDER BY user_id, last_used_at DESC
		) s ON u.id = s.user_id
		WHERE u.id = $1
	`

	var user AdminUser
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Phone, &user.Name, &user.AvatarURL, &user.Bio, &user.Role,
		&user.IsPhoneVerified, &user.FreeConversionsUsed, &user.FreeConversionsLimit,
		&user.CreatedAt, &user.UpdatedAt, &user.IsActive, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return AdminUser{}, fmt.Errorf("user not found")
		}
		return AdminUser{}, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// UpdateUser updates a user's information
func (s *DBStore) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (AdminUser, error) {
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

	if req.Role != nil {
		setParts = append(setParts, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, *req.Role)
		argIndex++
	}

	if req.IsPhoneVerified != nil {
		setParts = append(setParts, fmt.Sprintf("is_phone_verified = $%d", argIndex))
		args = append(args, *req.IsPhoneVerified)
		argIndex++
	}

	if req.FreeConversionsLimit != nil {
		setParts = append(setParts, fmt.Sprintf("free_conversions_limit = $%d", argIndex))
		args = append(args, *req.FreeConversionsLimit)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetUser(ctx, userID)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE users 
		SET %s
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return AdminUser{}, fmt.Errorf("failed to update user: %w", err)
	}

	return s.GetUser(ctx, userID)
}

// DeleteUser deletes a user
func (s *DBStore) DeleteUser(ctx context.Context, userID string) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// GetUserStats retrieves user statistics
func (s *DBStore) GetUserStats(ctx context.Context) (int, int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_active = true) as active
		FROM users
	`

	var total, active int
	err := s.db.QueryRowContext(ctx, query).Scan(&total, &active)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get user stats: %w", err)
	}

	return total, active, nil
}

// Vendor operations

// GetVendors retrieves a list of vendors with pagination and filtering
func (s *DBStore) GetVendors(ctx context.Context, req VendorListRequest) (VendorListResponse, error) {
	query := `
		SELECT 
			v.id, v.user_id, v.business_name, v.avatar_url, v.bio,
			v.contact_info, v.social_links, v.is_verified, v.is_active,
			v.free_images_used, v.free_images_limit, v.created_at, v.updated_at,
			COALESCE(s.last_used_at, v.created_at) as last_login_at
		FROM vendors v
		JOIN users u ON v.user_id = u.id
		LEFT JOIN (
			SELECT DISTINCT ON (user_id) user_id, last_used_at
			FROM sessions 
			WHERE revoked_at IS NULL
			ORDER BY user_id, last_used_at DESC
		) s ON v.user_id = s.user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.Search != "" {
		query += fmt.Sprintf(" AND (v.business_name ILIKE $%d OR u.phone ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + req.Search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	if req.IsActive != nil {
		query += fmt.Sprintf(" AND v.is_active = $%d", argIndex)
		args = append(args, *req.IsActive)
		argIndex++
	}

	if req.IsVerified != nil {
		query += fmt.Sprintf(" AND v.is_verified = $%d", argIndex)
		args = append(args, *req.IsVerified)
		argIndex++
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT v.id, v.user_id, v.business_name, v.avatar_url, v.bio, v.contact_info, v.social_links, v.is_verified, v.is_active, v.free_images_used, v.free_images_limit, v.created_at, v.updated_at, COALESCE(s.last_used_at, v.created_at) as last_login_at", "SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, "LEFT JOIN (", "", 1)
	countQuery = strings.Replace(countQuery, ") s ON v.user_id = s.user_id", "", 1)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return VendorListResponse{}, fmt.Errorf("failed to count vendors: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY v.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return VendorListResponse{}, fmt.Errorf("failed to query vendors: %w", err)
	}
	defer rows.Close()

	var vendors []AdminVendor
	for rows.Next() {
		var vendor AdminVendor
		var lastLoginAt sql.NullTime
		var contactInfoJSON, socialLinksJSON []byte

		err := rows.Scan(
			&vendor.ID, &vendor.UserID, &vendor.BusinessName, &vendor.AvatarURL, &vendor.Bio,
			&contactInfoJSON, &socialLinksJSON, &vendor.IsVerified, &vendor.IsActive,
			&vendor.FreeImagesUsed, &vendor.FreeImagesLimit, &vendor.CreatedAt, &vendor.UpdatedAt, &lastLoginAt,
		)
		if err != nil {
			return VendorListResponse{}, fmt.Errorf("failed to scan vendor: %w", err)
		}

		// Parse JSON fields
		if len(contactInfoJSON) > 0 {
			// In a real implementation, you'd use json.Unmarshal here
			// For now, we'll create empty structs
			vendor.ContactInfo = ContactInfo{}
		}

		if len(socialLinksJSON) > 0 {
			// In a real implementation, you'd use json.Unmarshal here
			// For now, we'll create empty structs
			vendor.SocialLinks = SocialLinks{}
		}

		if lastLoginAt.Valid {
			vendor.LastLoginAt = &lastLoginAt.Time
		}

		vendors = append(vendors, vendor)
	}

	if err = rows.Err(); err != nil {
		return VendorListResponse{}, fmt.Errorf("error iterating vendors: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return VendorListResponse{
		Vendors:    vendors,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetVendor retrieves a specific vendor by ID
func (s *DBStore) GetVendor(ctx context.Context, vendorID string) (AdminVendor, error) {
	query := `
		SELECT 
			v.id, v.user_id, v.business_name, v.avatar_url, v.bio,
			v.contact_info, v.social_links, v.is_verified, v.is_active,
			v.free_images_used, v.free_images_limit, v.created_at, v.updated_at,
			COALESCE(s.last_used_at, v.created_at) as last_login_at
		FROM vendors v
		JOIN users u ON v.user_id = u.id
		LEFT JOIN (
			SELECT DISTINCT ON (user_id) user_id, last_used_at
			FROM sessions 
			WHERE revoked_at IS NULL
			ORDER BY user_id, last_used_at DESC
		) s ON v.user_id = s.user_id
		WHERE v.id = $1
	`

	var vendor AdminVendor
	var lastLoginAt sql.NullTime
	var contactInfoJSON, socialLinksJSON []byte

	err := s.db.QueryRowContext(ctx, query, vendorID).Scan(
		&vendor.ID, &vendor.UserID, &vendor.BusinessName, &vendor.AvatarURL, &vendor.Bio,
		&contactInfoJSON, &socialLinksJSON, &vendor.IsVerified, &vendor.IsActive,
		&vendor.FreeImagesUsed, &vendor.FreeImagesLimit, &vendor.CreatedAt, &vendor.UpdatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return AdminVendor{}, fmt.Errorf("vendor not found")
		}
		return AdminVendor{}, fmt.Errorf("failed to get vendor: %w", err)
	}

	// Parse JSON fields
	if len(contactInfoJSON) > 0 {
		// In a real implementation, you'd use json.Unmarshal here
		vendor.ContactInfo = ContactInfo{}
	}

	if len(socialLinksJSON) > 0 {
		// In a real implementation, you'd use json.Unmarshal here
		vendor.SocialLinks = SocialLinks{}
	}

	if lastLoginAt.Valid {
		vendor.LastLoginAt = &lastLoginAt.Time
	}

	return vendor, nil
}

// UpdateVendor updates a vendor's information
func (s *DBStore) UpdateVendor(ctx context.Context, vendorID string, req UpdateVendorRequest) (AdminVendor, error) {
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.BusinessName != nil {
		setParts = append(setParts, fmt.Sprintf("business_name = $%d", argIndex))
		args = append(args, *req.BusinessName)
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

	if req.IsVerified != nil {
		setParts = append(setParts, fmt.Sprintf("is_verified = $%d", argIndex))
		args = append(args, *req.IsVerified)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if req.FreeImagesLimit != nil {
		setParts = append(setParts, fmt.Sprintf("free_images_limit = $%d", argIndex))
		args = append(args, *req.FreeImagesLimit)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetVendor(ctx, vendorID)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, vendorID)

	query := fmt.Sprintf(`
		UPDATE vendors 
		SET %s
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return AdminVendor{}, fmt.Errorf("failed to update vendor: %w", err)
	}

	return s.GetVendor(ctx, vendorID)
}

// DeleteVendor deletes a vendor
func (s *DBStore) DeleteVendor(ctx context.Context, vendorID string) error {
	query := "DELETE FROM vendors WHERE id = $1"
	_, err := s.db.ExecContext(ctx, query, vendorID)
	if err != nil {
		return fmt.Errorf("failed to delete vendor: %w", err)
	}
	return nil
}

// GetVendorStats retrieves vendor statistics
func (s *DBStore) GetVendorStats(ctx context.Context) (int, int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_active = true) as active
		FROM vendors
	`

	var total, active int
	err := s.db.QueryRowContext(ctx, query).Scan(&total, &active)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get vendor stats: %w", err)
	}

	return total, active, nil
}

// Plan operations

// GetPlans retrieves a list of plans with pagination and filtering
func (s *DBStore) GetPlans(ctx context.Context, req PlanListRequest) (PlanListResponse, error) {
	query := `
		SELECT 
			p.id, p.name, p.display_name, p.description, p.price_per_month_cents,
			p.monthly_conversions_limit, p.features, p.is_active, p.created_at, p.updated_at,
			COUNT(up.id) as subscriber_count
		FROM payment_plans p
		LEFT JOIN user_plans up ON p.id = up.plan_id AND up.status = 'active'
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.IsActive != nil {
		query += fmt.Sprintf(" AND p.is_active = $%d", argIndex)
		args = append(args, *req.IsActive)
		argIndex++
	}

	query += " GROUP BY p.id, p.name, p.display_name, p.description, p.price_per_month_cents, p.monthly_conversions_limit, p.features, p.is_active, p.created_at, p.updated_at"

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM payment_plans p
		WHERE 1=1
	`
	if req.IsActive != nil {
		countQuery += " AND p.is_active = $1"
	}

	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args[:argIndex-1]...).Scan(&total)
	if err != nil {
		return PlanListResponse{}, fmt.Errorf("failed to count plans: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY p.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return PlanListResponse{}, fmt.Errorf("failed to query plans: %w", err)
	}
	defer rows.Close()

	var plans []AdminPlan
	for rows.Next() {
		var plan AdminPlan
		var featuresJSON []byte

		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
			&plan.MonthlyConversionsLimit, &featuresJSON, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt, &plan.SubscriberCount,
		)
		if err != nil {
			return PlanListResponse{}, fmt.Errorf("failed to scan plan: %w", err)
		}

		// Parse features JSON
		if len(featuresJSON) > 0 {
			// In a real implementation, you'd use json.Unmarshal here
			// For now, we'll create an empty slice
			plan.Features = []string{}
		}

		plans = append(plans, plan)
	}

	if err = rows.Err(); err != nil {
		return PlanListResponse{}, fmt.Errorf("error iterating plans: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return PlanListResponse{
		Plans:      plans,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetPlan retrieves a specific plan by ID
func (s *DBStore) GetPlan(ctx context.Context, planID string) (AdminPlan, error) {
	query := `
		SELECT 
			p.id, p.name, p.display_name, p.description, p.price_per_month_cents,
			p.monthly_conversions_limit, p.features, p.is_active, p.created_at, p.updated_at,
			COUNT(up.id) as subscriber_count
		FROM payment_plans p
		LEFT JOIN user_plans up ON p.id = up.plan_id AND up.status = 'active'
		WHERE p.id = $1
		GROUP BY p.id, p.name, p.display_name, p.description, p.price_per_month_cents, p.monthly_conversions_limit, p.features, p.is_active, p.created_at, p.updated_at
	`

	var plan AdminPlan
	var featuresJSON []byte

	err := s.db.QueryRowContext(ctx, query, planID).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
		&plan.MonthlyConversionsLimit, &featuresJSON, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt, &plan.SubscriberCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return AdminPlan{}, fmt.Errorf("plan not found")
		}
		return AdminPlan{}, fmt.Errorf("failed to get plan: %w", err)
	}

	// Parse features JSON
	if len(featuresJSON) > 0 {
		// In a real implementation, you'd use json.Unmarshal here
		plan.Features = []string{}
	}

	return plan, nil
}

// CreatePlan creates a new subscription plan
func (s *DBStore) CreatePlan(ctx context.Context, req CreatePlanRequest) (AdminPlan, error) {
	query := `
		INSERT INTO payment_plans (name, display_name, description, price_per_month_cents, monthly_conversions_limit, features, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, display_name, description, price_per_month_cents, monthly_conversions_limit, features, is_active, created_at, updated_at
	`

	var plan AdminPlan
	// Convert features to JSON
	// In a real implementation, you'd use json.Marshal here
	featuresJSON := []byte("[]")

	err := s.db.QueryRowContext(ctx, query, req.Name, req.DisplayName, req.Description, req.PricePerMonthCents, req.MonthlyConversionsLimit, featuresJSON, req.IsActive).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
		&plan.MonthlyConversionsLimit, &featuresJSON, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return AdminPlan{}, fmt.Errorf("failed to create plan: %w", err)
	}

	// Parse features JSON
	if len(featuresJSON) > 0 {
		// In a real implementation, you'd use json.Unmarshal here
		plan.Features = []string{}
	}

	plan.SubscriberCount = 0

	return plan, nil
}

// UpdatePlan updates a subscription plan
func (s *DBStore) UpdatePlan(ctx context.Context, planID string, req UpdatePlanRequest) (AdminPlan, error) {
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.DisplayName != nil {
		setParts = append(setParts, fmt.Sprintf("display_name = $%d", argIndex))
		args = append(args, *req.DisplayName)
		argIndex++
	}

	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.PricePerMonthCents != nil {
		setParts = append(setParts, fmt.Sprintf("price_per_month_cents = $%d", argIndex))
		args = append(args, *req.PricePerMonthCents)
		argIndex++
	}

	if req.MonthlyConversionsLimit != nil {
		setParts = append(setParts, fmt.Sprintf("monthly_conversions_limit = $%d", argIndex))
		args = append(args, *req.MonthlyConversionsLimit)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetPlan(ctx, planID)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, planID)

	query := fmt.Sprintf(`
		UPDATE payment_plans 
		SET %s
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return AdminPlan{}, fmt.Errorf("failed to update plan: %w", err)
	}

	return s.GetPlan(ctx, planID)
}

// DeletePlan deletes a subscription plan
func (s *DBStore) DeletePlan(ctx context.Context, planID string) error {
	query := "DELETE FROM payment_plans WHERE id = $1"
	_, err := s.db.ExecContext(ctx, query, planID)
	if err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}
	return nil
}

// Payment operations

// GetPayments retrieves a list of payments with pagination and filtering
func (s *DBStore) GetPayments(ctx context.Context, req PaymentListRequest) (PaymentListResponse, error) {
	query := `
		SELECT 
			p.id, p.user_id, u.phone, p.plan_id, pp.name as plan_name,
			p.amount, p.currency, p.status, p.payment_method, p.gateway,
			p.gateway_track_id, p.gateway_ref_number, p.gateway_card_number,
			p.description, p.created_at, p.updated_at, p.paid_at, p.expires_at
		FROM payments p
		JOIN users u ON p.user_id = u.id
		JOIN payment_plans pp ON p.plan_id = pp.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND p.status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.UserID != "" {
		query += fmt.Sprintf(" AND p.user_id = $%d", argIndex)
		args = append(args, req.UserID)
		argIndex++
	}

	if req.PlanID != "" {
		query += fmt.Sprintf(" AND p.plan_id = $%d", argIndex)
		args = append(args, req.PlanID)
		argIndex++
	}

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND p.created_at >= $%d", argIndex)
		args = append(args, req.DateFrom)
		argIndex++
	}

	if req.DateTo != "" {
		query += fmt.Sprintf(" AND p.created_at <= $%d", argIndex)
		args = append(args, req.DateTo)
		argIndex++
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT p.id, p.user_id, u.phone, p.plan_id, pp.name as plan_name, p.amount, p.currency, p.status, p.payment_method, p.gateway, p.gateway_track_id, p.gateway_ref_number, p.gateway_card_number, p.description, p.created_at, p.updated_at, p.paid_at, p.expires_at", "SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, "JOIN users u ON p.user_id = u.id", "", 1)
	countQuery = strings.Replace(countQuery, "JOIN payment_plans pp ON p.plan_id = pp.id", "", 1)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return PaymentListResponse{}, fmt.Errorf("failed to count payments: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY p.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return PaymentListResponse{}, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	var payments []AdminPayment
	for rows.Next() {
		var payment AdminPayment

		err := rows.Scan(
			&payment.ID, &payment.UserID, &payment.UserPhone, &payment.PlanID, &payment.PlanName,
			&payment.Amount, &payment.Currency, &payment.Status, &payment.PaymentMethod, &payment.Gateway,
			&payment.GatewayTrackID, &payment.GatewayRefNumber, &payment.GatewayCardNumber,
			&payment.Description, &payment.CreatedAt, &payment.UpdatedAt, &payment.PaidAt, &payment.ExpiresAt,
		)
		if err != nil {
			return PaymentListResponse{}, fmt.Errorf("failed to scan payment: %w", err)
		}

		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return PaymentListResponse{}, fmt.Errorf("error iterating payments: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return PaymentListResponse{
		Payments:   payments,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetPayment retrieves a specific payment by ID
func (s *DBStore) GetPayment(ctx context.Context, paymentID string) (AdminPayment, error) {
	query := `
		SELECT 
			p.id, p.user_id, u.phone, p.plan_id, pp.name as plan_name,
			p.amount, p.currency, p.status, p.payment_method, p.gateway,
			p.gateway_track_id, p.gateway_ref_number, p.gateway_card_number,
			p.description, p.created_at, p.updated_at, p.paid_at, p.expires_at
		FROM payments p
		JOIN users u ON p.user_id = u.id
		JOIN payment_plans pp ON p.plan_id = pp.id
		WHERE p.id = $1
	`

	var payment AdminPayment
	err := s.db.QueryRowContext(ctx, query, paymentID).Scan(
		&payment.ID, &payment.UserID, &payment.UserPhone, &payment.PlanID, &payment.PlanName,
		&payment.Amount, &payment.Currency, &payment.Status, &payment.PaymentMethod, &payment.Gateway,
		&payment.GatewayTrackID, &payment.GatewayRefNumber, &payment.GatewayCardNumber,
		&payment.Description, &payment.CreatedAt, &payment.UpdatedAt, &payment.PaidAt, &payment.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return AdminPayment{}, fmt.Errorf("payment not found")
		}
		return AdminPayment{}, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// GetPaymentStats retrieves payment statistics
func (s *DBStore) GetPaymentStats(ctx context.Context) (int, int64, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(amount) FILTER (WHERE status = 'completed'), 0) as revenue
		FROM payments
	`

	var total int
	var revenue int64
	err := s.db.QueryRowContext(ctx, query).Scan(&total, &revenue)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get payment stats: %w", err)
	}

	return total, revenue, nil
}

// Conversion operations

// GetConversions retrieves a list of conversions with pagination and filtering
func (s *DBStore) GetConversions(ctx context.Context, req ConversionListRequest) (ConversionListResponse, error) {
	query := `
		SELECT 
			uc.id, uc.user_id, u.phone, uc.conversion_type, uc.input_file_url,
			uc.output_file_url, uc.style_name, uc.status, uc.error_message,
			uc.processing_time_ms, uc.file_size_bytes, uc.created_at, uc.completed_at
		FROM user_conversions uc
		JOIN users u ON uc.user_id = u.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND uc.status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.UserID != "" {
		query += fmt.Sprintf(" AND uc.user_id = $%d", argIndex)
		args = append(args, req.UserID)
		argIndex++
	}

	if req.Type != "" {
		query += fmt.Sprintf(" AND uc.conversion_type = $%d", argIndex)
		args = append(args, req.Type)
		argIndex++
	}

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND uc.created_at >= $%d", argIndex)
		args = append(args, req.DateFrom)
		argIndex++
	}

	if req.DateTo != "" {
		query += fmt.Sprintf(" AND uc.created_at <= $%d", argIndex)
		args = append(args, req.DateTo)
		argIndex++
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT uc.id, uc.user_id, u.phone, uc.conversion_type, uc.input_file_url, uc.output_file_url, uc.style_name, uc.status, uc.error_message, uc.processing_time_ms, uc.file_size_bytes, uc.created_at, uc.completed_at", "SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, "JOIN users u ON uc.user_id = u.id", "", 1)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to count conversions: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY uc.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to query conversions: %w", err)
	}
	defer rows.Close()

	var conversions []AdminConversion
	for rows.Next() {
		var conversion AdminConversion

		err := rows.Scan(
			&conversion.ID, &conversion.UserID, &conversion.UserPhone, &conversion.ConversionType,
			&conversion.InputFileURL, &conversion.OutputFileURL, &conversion.StyleName, &conversion.Status,
			&conversion.ErrorMessage, &conversion.ProcessingTimeMs, &conversion.FileSizeBytes,
			&conversion.CreatedAt, &conversion.CompletedAt,
		)
		if err != nil {
			return ConversionListResponse{}, fmt.Errorf("failed to scan conversion: %w", err)
		}

		conversions = append(conversions, conversion)
	}

	if err = rows.Err(); err != nil {
		return ConversionListResponse{}, fmt.Errorf("error iterating conversions: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return ConversionListResponse{
		Conversions: conversions,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
	}, nil
}

// GetConversion retrieves a specific conversion by ID
func (s *DBStore) GetConversion(ctx context.Context, conversionID string) (AdminConversion, error) {
	query := `
		SELECT 
			uc.id, uc.user_id, u.phone, uc.conversion_type, uc.input_file_url,
			uc.output_file_url, uc.style_name, uc.status, uc.error_message,
			uc.processing_time_ms, uc.file_size_bytes, uc.created_at, uc.completed_at
		FROM user_conversions uc
		JOIN users u ON uc.user_id = u.id
		WHERE uc.id = $1
	`

	var conversion AdminConversion
	err := s.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conversion.ID, &conversion.UserID, &conversion.UserPhone, &conversion.ConversionType,
		&conversion.InputFileURL, &conversion.OutputFileURL, &conversion.StyleName, &conversion.Status,
		&conversion.ErrorMessage, &conversion.ProcessingTimeMs, &conversion.FileSizeBytes,
		&conversion.CreatedAt, &conversion.CompletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return AdminConversion{}, fmt.Errorf("conversion not found")
		}
		return AdminConversion{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	return conversion, nil
}

// GetConversionStats retrieves conversion statistics
func (s *DBStore) GetConversionStats(ctx context.Context) (int, int, int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'pending' OR status = 'processing') as pending,
			COUNT(*) FILTER (WHERE status = 'failed') as failed
		FROM user_conversions
	`

	var total, pending, failed int
	err := s.db.QueryRowContext(ctx, query).Scan(&total, &pending, &failed)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get conversion stats: %w", err)
	}

	return total, pending, failed, nil
}

// Image operations

// GetImages retrieves a list of images with pagination and filtering
func (s *DBStore) GetImages(ctx context.Context, req ImageListRequest) (ImageListResponse, error) {
	query := `
		SELECT 
			i.id, i.vendor_id, v.business_name, i.album_id, a.name as album_name,
			i.file_name, i.original_url, i.thumbnail_url, i.file_size, i.mime_type,
			i.width, i.height, i.is_free, i.is_public, i.tags, i.created_at, i.updated_at
		FROM images i
		JOIN vendors v ON i.vendor_id = v.id
		LEFT JOIN albums a ON i.album_id = a.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.VendorID != "" {
		query += fmt.Sprintf(" AND i.vendor_id = $%d", argIndex)
		args = append(args, req.VendorID)
		argIndex++
	}

	if req.IsPublic != nil {
		query += fmt.Sprintf(" AND i.is_public = $%d", argIndex)
		args = append(args, *req.IsPublic)
		argIndex++
	}

	if req.IsFree != nil {
		query += fmt.Sprintf(" AND i.is_free = $%d", argIndex)
		args = append(args, *req.IsFree)
		argIndex++
	}

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND i.created_at >= $%d", argIndex)
		args = append(args, req.DateFrom)
		argIndex++
	}

	if req.DateTo != "" {
		query += fmt.Sprintf(" AND i.created_at <= $%d", argIndex)
		args = append(args, req.DateTo)
		argIndex++
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT i.id, i.vendor_id, v.business_name, i.album_id, a.name as album_name, i.file_name, i.original_url, i.thumbnail_url, i.file_size, i.mime_type, i.width, i.height, i.is_free, i.is_public, i.tags, i.created_at, i.updated_at", "SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, "JOIN vendors v ON i.vendor_id = v.id", "", 1)
	countQuery = strings.Replace(countQuery, "LEFT JOIN albums a ON i.album_id = a.id", "", 1)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to count images: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY i.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ImageListResponse{}, fmt.Errorf("failed to query images: %w", err)
	}
	defer rows.Close()

	var images []AdminImage
	for rows.Next() {
		var image AdminImage
		var albumName sql.NullString
		var tagsJSON []byte

		err := rows.Scan(
			&image.ID, &image.VendorID, &image.VendorName, &image.AlbumID, &albumName,
			&image.FileName, &image.OriginalURL, &image.ThumbnailURL, &image.FileSize, &image.MimeType,
			&image.Width, &image.Height, &image.IsFree, &image.IsPublic, &tagsJSON,
			&image.CreatedAt, &image.UpdatedAt,
		)
		if err != nil {
			return ImageListResponse{}, fmt.Errorf("failed to scan image: %w", err)
		}

		if albumName.Valid {
			image.AlbumName = &albumName.String
		}

		// Parse tags JSON
		if len(tagsJSON) > 0 {
			// In a real implementation, you'd use json.Unmarshal here
			image.Tags = []string{}
		}

		images = append(images, image)
	}

	if err = rows.Err(); err != nil {
		return ImageListResponse{}, fmt.Errorf("error iterating images: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return ImageListResponse{
		Images:     images,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetImage retrieves a specific image by ID
func (s *DBStore) GetImage(ctx context.Context, imageID string) (AdminImage, error) {
	query := `
		SELECT 
			i.id, i.vendor_id, v.business_name, i.album_id, a.name as album_name,
			i.file_name, i.original_url, i.thumbnail_url, i.file_size, i.mime_type,
			i.width, i.height, i.is_free, i.is_public, i.tags, i.created_at, i.updated_at
		FROM images i
		JOIN vendors v ON i.vendor_id = v.id
		LEFT JOIN albums a ON i.album_id = a.id
		WHERE i.id = $1
	`

	var image AdminImage
	var albumName sql.NullString
	var tagsJSON []byte

	err := s.db.QueryRowContext(ctx, query, imageID).Scan(
		&image.ID, &image.VendorID, &image.VendorName, &image.AlbumID, &albumName,
		&image.FileName, &image.OriginalURL, &image.ThumbnailURL, &image.FileSize, &image.MimeType,
		&image.Width, &image.Height, &image.IsFree, &image.IsPublic, &tagsJSON,
		&image.CreatedAt, &image.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return AdminImage{}, fmt.Errorf("image not found")
		}
		return AdminImage{}, fmt.Errorf("failed to get image: %w", err)
	}

	if albumName.Valid {
		image.AlbumName = &albumName.String
	}

	// Parse tags JSON
	if len(tagsJSON) > 0 {
		// In a real implementation, you'd use json.Unmarshal here
		image.Tags = []string{}
	}

	return image, nil
}

// GetImageStats retrieves image statistics
func (s *DBStore) GetImageStats(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM images"

	var total int
	err := s.db.QueryRowContext(ctx, query).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get image stats: %w", err)
	}

	return total, nil
}

// Audit log operations

// GetAuditLogs retrieves a list of audit logs with pagination and filtering
func (s *DBStore) GetAuditLogs(ctx context.Context, req AuditLogListRequest) (AuditLogListResponse, error) {
	query := `
		SELECT 
			id, user_id, actor_type, action, resource, resource_id, metadata, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if req.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, req.UserID)
		argIndex++
	}

	if req.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, req.Action)
		argIndex++
	}

	if req.Resource != "" {
		query += fmt.Sprintf(" AND resource = $%d", argIndex)
		args = append(args, req.Resource)
		argIndex++
	}

	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, req.DateFrom)
		argIndex++
	}

	if req.DateTo != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, req.DateTo)
		argIndex++
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT id, user_id, actor_type, action, resource, resource_id, metadata, created_at", "SELECT COUNT(*)", 1)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return AuditLogListResponse{}, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return AuditLogListResponse{}, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var auditLogs []AuditLog
	for rows.Next() {
		var auditLog AuditLog
		var userID sql.NullString
		var resourceID sql.NullString
		var metadataJSON []byte

		err := rows.Scan(
			&auditLog.ID, &userID, &auditLog.ActorType, &auditLog.Action, &auditLog.Resource,
			&resourceID, &metadataJSON, &auditLog.CreatedAt,
		)
		if err != nil {
			return AuditLogListResponse{}, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if userID.Valid {
			auditLog.UserID = &userID.String
		}

		if resourceID.Valid {
			auditLog.ResourceID = &resourceID.String
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			// In a real implementation, you'd use json.Unmarshal here
			auditLog.Metadata = map[string]interface{}{}
		}

		auditLogs = append(auditLogs, auditLog)
	}

	if err = rows.Err(); err != nil {
		return AuditLogListResponse{}, fmt.Errorf("error iterating audit logs: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return AuditLogListResponse{
		AuditLogs:  auditLogs,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateAuditLog creates a new audit log entry
func (s *DBStore) CreateAuditLog(ctx context.Context, log AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, user_id, actor_type, action, resource, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Convert metadata to JSON
	// In a real implementation, you'd use json.Marshal here
	metadataJSON := []byte("{}")

	_, err := s.db.ExecContext(ctx, query, log.ID, log.UserID, log.ActorType, log.Action, log.Resource, log.ResourceID, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// Quota operations

// RevokeUserQuota revokes quota from a user
func (s *DBStore) RevokeUserQuota(ctx context.Context, userID string, quotaType string, amount int, reason string) error {
	// This is a simplified implementation
	// In a real implementation, you'd need to handle quota revocation logic
	// and update the appropriate tables

	// Log the quota revocation
	auditLog := AuditLog{
		ID:         fmt.Sprintf("quota_revoke_%d", time.Now().UnixNano()),
		UserID:     &userID,
		ActorType:  ActorTypeAdmin,
		Action:     ActionRevoke,
		Resource:   ResourceQuota,
		ResourceID: &userID,
		Metadata: map[string]interface{}{
			"quota_type": quotaType,
			"amount":     amount,
			"reason":     reason,
		},
		CreatedAt: time.Now(),
	}

	return s.CreateAuditLog(ctx, auditLog)
}

// RevokeVendorQuota revokes quota from a vendor
func (s *DBStore) RevokeVendorQuota(ctx context.Context, vendorID string, quotaType string, amount int, reason string) error {
	// This is a simplified implementation
	// In a real implementation, you'd need to handle quota revocation logic
	// and update the appropriate tables

	// Log the quota revocation
	auditLog := AuditLog{
		ID:         fmt.Sprintf("quota_revoke_%d", time.Now().UnixNano()),
		UserID:     nil, // Vendor quota revocation
		ActorType:  ActorTypeAdmin,
		Action:     ActionRevoke,
		Resource:   ResourceQuota,
		ResourceID: &vendorID,
		Metadata: map[string]interface{}{
			"quota_type": quotaType,
			"amount":     amount,
			"reason":     reason,
		},
		CreatedAt: time.Now(),
	}

	return s.CreateAuditLog(ctx, auditLog)
}

// RevokeUserPlan revokes a user's subscription plan
func (s *DBStore) RevokeUserPlan(ctx context.Context, userID string, reason string) error {
	// Update user plan status to cancelled
	query := `
		UPDATE user_plans 
		SET status = 'cancelled', updated_at = NOW()
		WHERE user_id = $1 AND status = 'active'
	`

	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user plan: %w", err)
	}

	// Log the plan revocation
	auditLog := AuditLog{
		ID:         fmt.Sprintf("plan_revoke_%d", time.Now().UnixNano()),
		UserID:     &userID,
		ActorType:  ActorTypeAdmin,
		Action:     ActionRevoke,
		Resource:   ResourcePlan,
		ResourceID: &userID,
		Metadata: map[string]interface{}{
			"reason": reason,
		},
		CreatedAt: time.Now(),
	}

	return s.CreateAuditLog(ctx, auditLog)
}

// Statistics

// GetSystemStats retrieves system-wide statistics
func (s *DBStore) GetSystemStats(ctx context.Context) (AdminStats, error) {
	// Get user stats
	userTotal, userActive, err := s.GetUserStats(ctx)
	if err != nil {
		return AdminStats{}, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Get vendor stats
	vendorTotal, vendorActive, err := s.GetVendorStats(ctx)
	if err != nil {
		return AdminStats{}, fmt.Errorf("failed to get vendor stats: %w", err)
	}

	// Get conversion stats
	conversionTotal, conversionPending, conversionFailed, err := s.GetConversionStats(ctx)
	if err != nil {
		return AdminStats{}, fmt.Errorf("failed to get conversion stats: %w", err)
	}

	// Get payment stats
	paymentTotal, revenue, err := s.GetPaymentStats(ctx)
	if err != nil {
		return AdminStats{}, fmt.Errorf("failed to get payment stats: %w", err)
	}

	// Get image stats
	imageTotal, err := s.GetImageStats(ctx)
	if err != nil {
		return AdminStats{}, fmt.Errorf("failed to get image stats: %w", err)
	}

	return AdminStats{
		TotalUsers:         userTotal,
		ActiveUsers:        userActive,
		TotalVendors:       vendorTotal,
		ActiveVendors:      vendorActive,
		TotalConversions:   conversionTotal,
		TotalPayments:      paymentTotal,
		TotalRevenue:       revenue,
		TotalImages:        imageTotal,
		PendingConversions: conversionPending,
		FailedConversions:  conversionFailed,
	}, nil
}
