package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// PaymentStoreImpl implements the payment data operations
type PaymentStoreImpl struct {
	db *sql.DB
}

// NewPaymentStore creates a new payment store
func NewPaymentStore(db *sql.DB) *PaymentStoreImpl {
	return &PaymentStoreImpl{db: db}
}

// CreatePayment creates a new payment record
func (s *PaymentStoreImpl) CreatePayment(ctx context.Context, payment Payment) (Payment, error) {
	query := `
		INSERT INTO payments (
			id, user_id, plan_id, amount, currency, status, payment_method, 
			gateway, gateway_track_id, description, callback_url, return_url, 
			created_at, updated_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		) RETURNING id, user_id, plan_id, amount, currency, status, payment_method, 
			gateway, gateway_track_id, gateway_ref_number, gateway_card_number, 
			description, callback_url, return_url, created_at, updated_at, paid_at, expires_at`

	var result Payment
	err := s.db.QueryRowContext(ctx, query,
		payment.ID, payment.UserID, payment.PlanID, payment.Amount, payment.Currency,
		payment.Status, payment.PaymentMethod, payment.Gateway, payment.GatewayTrackID,
		payment.Description, payment.CallbackURL, payment.ReturnURL,
		payment.CreatedAt, payment.UpdatedAt, payment.ExpiresAt,
	).Scan(
		&result.ID, &result.UserID, &result.PlanID, &result.Amount, &result.Currency,
		&result.Status, &result.PaymentMethod, &result.Gateway, &result.GatewayTrackID,
		&result.GatewayRefNumber, &result.GatewayCardNumber, &result.Description,
		&result.CallbackURL, &result.ReturnURL, &result.CreatedAt, &result.UpdatedAt,
		&result.PaidAt, &result.ExpiresAt,
	)

	if err != nil {
		return Payment{}, fmt.Errorf("failed to create payment: %w", err)
	}

	return result, nil
}

// GetPayment retrieves a payment by ID
func (s *PaymentStoreImpl) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	query := `
		SELECT id, user_id, plan_id, amount, currency, status, payment_method, 
			gateway, gateway_track_id, gateway_ref_number, gateway_card_number, 
			description, callback_url, return_url, created_at, updated_at, paid_at, expires_at
		FROM payments 
		WHERE id = $1`

	var payment Payment
	err := s.db.QueryRowContext(ctx, query, paymentID).Scan(
		&payment.ID, &payment.UserID, &payment.PlanID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.PaymentMethod, &payment.Gateway, &payment.GatewayTrackID,
		&payment.GatewayRefNumber, &payment.GatewayCardNumber, &payment.Description,
		&payment.CallbackURL, &payment.ReturnURL, &payment.CreatedAt, &payment.UpdatedAt,
		&payment.PaidAt, &payment.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, fmt.Errorf("payment not found")
		}
		return Payment{}, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// GetPaymentByTrackID retrieves a payment by gateway track ID
func (s *PaymentStoreImpl) GetPaymentByTrackID(ctx context.Context, trackID string) (Payment, error) {
	query := `
		SELECT id, user_id, plan_id, amount, currency, status, payment_method, 
			gateway, gateway_track_id, gateway_ref_number, gateway_card_number, 
			description, callback_url, return_url, created_at, updated_at, paid_at, expires_at
		FROM payments 
		WHERE gateway_track_id = $1`

	var payment Payment
	err := s.db.QueryRowContext(ctx, query, trackID).Scan(
		&payment.ID, &payment.UserID, &payment.PlanID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.PaymentMethod, &payment.Gateway, &payment.GatewayTrackID,
		&payment.GatewayRefNumber, &payment.GatewayCardNumber, &payment.Description,
		&payment.CallbackURL, &payment.ReturnURL, &payment.CreatedAt, &payment.UpdatedAt,
		&payment.PaidAt, &payment.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, fmt.Errorf("payment not found")
		}
		return Payment{}, fmt.Errorf("failed to get payment by track ID: %w", err)
	}

	return payment, nil
}

// UpdatePayment updates a payment record
func (s *PaymentStoreImpl) UpdatePayment(ctx context.Context, paymentID string, updates map[string]interface{}) (Payment, error) {
	if len(updates) == 0 {
		return s.GetPayment(ctx, paymentID)
	}

	// Whitelist of allowed payment fields to prevent SQL injection
	allowedFields := map[string]bool{
		"status":              true,
		"gateway_track_id":    true,
		"gateway_ref_number":  true,
		"gateway_card_number": true,
		"description":         true,
		"callback_url":        true,
		"return_url":          true,
		"paid_at":             true,
		"expires_at":          true,
	}

	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		// Validate field name to prevent SQL injection
		if !allowedFields[field] {
			return Payment{}, fmt.Errorf("invalid field name: %s", field)
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add payment ID to args
	args = append(args, paymentID)

	// Join all set parts with commas
	setClause := strings.Join(setParts, ", ")

	query := fmt.Sprintf(`
		UPDATE payments 
		SET %s
		WHERE id = $%d
		RETURNING id, user_id, plan_id, amount, currency, status, payment_method, 
			gateway, gateway_track_id, gateway_ref_number, gateway_card_number, 
			description, callback_url, return_url, created_at, updated_at, paid_at, expires_at`,
		setClause, argIndex)

	var payment Payment
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&payment.ID, &payment.UserID, &payment.PlanID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.PaymentMethod, &payment.Gateway, &payment.GatewayTrackID,
		&payment.GatewayRefNumber, &payment.GatewayCardNumber, &payment.Description,
		&payment.CallbackURL, &payment.ReturnURL, &payment.CreatedAt, &payment.UpdatedAt,
		&payment.PaidAt, &payment.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, fmt.Errorf("payment not found")
		}
		return Payment{}, fmt.Errorf("failed to update payment: %w", err)
	}

	return payment, nil
}

// GetPaymentHistory retrieves payment history for a user
func (s *PaymentStoreImpl) GetPaymentHistory(ctx context.Context, userID string, req PaymentHistoryRequest) (PaymentHistoryResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	// Build WHERE clause
	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM payments %s", whereClause)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return PaymentHistoryResponse{}, fmt.Errorf("failed to count payments: %w", err)
	}

	// Get payments with plan information
	query := fmt.Sprintf(`
		SELECT p.id, p.user_id, p.vendor_id, p.plan_id, p.amount, p.currency, p.status, 
			p.payment_method, p.gateway, p.gateway_track_id, p.gateway_ref_number, 
			p.description, p.created_at, p.paid_at,
			pp.name as plan_name, pp.display_name as plan_display_name
		FROM payments p
		LEFT JOIN payment_plans pp ON p.plan_id = pp.id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, req.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return PaymentHistoryResponse{}, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	var historyItems []PaymentHistoryItem
	for rows.Next() {
		var item PaymentHistoryItem
		err := rows.Scan(
			&item.PaymentID, &item.UserID, &item.VendorID, &item.PlanID,
			&item.Amount, &item.Currency, &item.Status, &item.PaymentMethod,
			&item.Gateway, &item.GatewayTrackID, &item.GatewayRefNumber,
			&item.Description, &item.CreatedAt, &item.PaidAt,
			&item.PlanName, &item.PlanDisplayName,
		)
		if err != nil {
			return PaymentHistoryResponse{}, fmt.Errorf("failed to scan payment: %w", err)
		}
		historyItems = append(historyItems, item)
	}

	if err = rows.Err(); err != nil {
		return PaymentHistoryResponse{}, fmt.Errorf("failed to iterate payments: %w", err)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return PaymentHistoryResponse{
		Payments:   historyItems,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetPlan retrieves a payment plan by ID
func (s *PaymentStoreImpl) GetPlan(ctx context.Context, planID string) (PaymentPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_per_month_cents, 
			monthly_conversions_limit, features, is_active, created_at, updated_at
		FROM payment_plans 
		WHERE id = $1`

	var plan PaymentPlan
	err := s.db.QueryRowContext(ctx, query, planID).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
		&plan.MonthlyConversionsLimit, pq.Array(&plan.Features), &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentPlan{}, fmt.Errorf("plan not found")
		}
		return PaymentPlan{}, fmt.Errorf("failed to get plan: %w", err)
	}

	return plan, nil
}

// GetAllPlans retrieves all active payment plans
func (s *PaymentStoreImpl) GetAllPlans(ctx context.Context) ([]PaymentPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_per_month_cents, 
			monthly_conversions_limit, features, is_active, created_at, updated_at
		FROM payment_plans 
		WHERE is_active = true
		ORDER BY price_per_month_cents ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query plans: %w", err)
	}
	defer rows.Close()

	var plans []PaymentPlan
	for rows.Next() {
		var plan PaymentPlan
		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
			&plan.MonthlyConversionsLimit, pq.Array(&plan.Features), &plan.IsActive,
			&plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, plan)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate plans: %w", err)
	}

	return plans, nil
}

// CreatePlan creates a new payment plan
func (s *PaymentStoreImpl) CreatePlan(ctx context.Context, plan PaymentPlan) (PaymentPlan, error) {
	query := `
		INSERT INTO payment_plans (
			id, name, display_name, description, price_per_month_cents, 
			monthly_conversions_limit, features, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id, name, display_name, description, price_per_month_cents, 
			monthly_conversions_limit, features, is_active, created_at, updated_at`

	var result PaymentPlan
	err := s.db.QueryRowContext(ctx, query,
		plan.ID, plan.Name, plan.DisplayName, plan.Description, plan.PricePerMonthCents,
		plan.MonthlyConversionsLimit, pq.Array(plan.Features), plan.IsActive,
		plan.CreatedAt, plan.UpdatedAt,
	).Scan(
		&result.ID, &result.Name, &result.DisplayName, &result.Description, &result.PricePerMonthCents,
		&result.MonthlyConversionsLimit, pq.Array(&result.Features), &result.IsActive,
		&result.CreatedAt, &result.UpdatedAt,
	)

	if err != nil {
		return PaymentPlan{}, fmt.Errorf("failed to create plan: %w", err)
	}

	return result, nil
}

// UpdatePlan updates a payment plan
func (s *PaymentStoreImpl) UpdatePlan(ctx context.Context, planID string, updates map[string]interface{}) (PaymentPlan, error) {
	if len(updates) == 0 {
		return s.GetPlan(ctx, planID)
	}

	// Whitelist of allowed plan fields to prevent SQL injection
	allowedFields := map[string]bool{
		"name":                      true,
		"display_name":              true,
		"description":               true,
		"price_per_month_cents":     true,
		"monthly_conversions_limit": true,
		"features":                  true,
		"is_active":                 true,
	}

	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		// Validate field name to prevent SQL injection
		if !allowedFields[field] {
			return PaymentPlan{}, fmt.Errorf("invalid field name: %s", field)
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add plan ID to args
	args = append(args, planID)

	// Join all set parts with commas
	setClause := strings.Join(setParts, ", ")

	query := fmt.Sprintf(`
		UPDATE payment_plans 
		SET %s
		WHERE id = $%d
		RETURNING id, name, display_name, description, price_per_month_cents, 
			monthly_conversions_limit, features, is_active, created_at, updated_at`,
		setClause, argIndex)

	var plan PaymentPlan
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
		&plan.MonthlyConversionsLimit, pq.Array(&plan.Features), &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentPlan{}, fmt.Errorf("plan not found")
		}
		return PaymentPlan{}, fmt.Errorf("failed to update plan: %w", err)
	}

	return plan, nil
}

// GetUserActivePlan retrieves the user's active plan
func (s *PaymentStoreImpl) GetUserActivePlan(ctx context.Context, userID string) (PaymentPlan, error) {
	query := `
		SELECT p.id, p.name, p.display_name, p.description, p.price_per_month_cents, 
			p.monthly_conversions_limit, p.features, p.is_active, p.created_at, p.updated_at
		FROM payment_plans p
		JOIN user_plans up ON p.id = up.plan_id
		WHERE up.user_id = $1 AND up.status = 'active'`

	var plan PaymentPlan
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description, &plan.PricePerMonthCents,
		&plan.MonthlyConversionsLimit, pq.Array(&plan.Features), &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentPlan{}, fmt.Errorf("no active plan found")
		}
		return PaymentPlan{}, fmt.Errorf("failed to get user active plan: %w", err)
	}

	return plan, nil
}

// ActivateUserPlan activates a plan for a user
func (s *PaymentStoreImpl) ActivateUserPlan(ctx context.Context, userID string, planID string, paymentID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Deactivate current plan
	_, err = tx.ExecContext(ctx, `
		UPDATE user_plans 
		SET status = 'cancelled', updated_at = NOW() 
		WHERE user_id = $1 AND status = 'active'`, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate current plan: %w", err)
	}

	// Create new active plan
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_plans (
			user_id, plan_id, status, monthly_conversions_limit, 
			conversions_used_this_month, price_per_month_cents, 
			billing_cycle_start_date, billing_cycle_end_date, 
			auto_renew, created_at, updated_at
		) VALUES (
			$1, $2, 'active', $3, 0, $4, 
			CURRENT_DATE, CURRENT_DATE + INTERVAL '1 month', 
			true, NOW(), NOW()
		)`, userID, planID, 0, 0) // Will be updated with actual plan details
	if err != nil {
		return fmt.Errorf("failed to activate new plan: %w", err)
	}

	// Update with actual plan details
	_, err = tx.ExecContext(ctx, `
		UPDATE user_plans 
		SET monthly_conversions_limit = p.monthly_conversions_limit,
			price_per_month_cents = p.price_per_month_cents
		FROM payment_plans p
		WHERE user_plans.plan_id = p.id 
			AND user_plans.user_id = $1 
			AND user_plans.status = 'active'`, userID)
	if err != nil {
		return fmt.Errorf("failed to update plan details: %w", err)
	}

	return tx.Commit()
}

// DeactivateUserPlan deactivates the user's current plan
func (s *PaymentStoreImpl) DeactivateUserPlan(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_plans 
		SET status = 'cancelled', updated_at = NOW() 
		WHERE user_id = $1 AND status = 'active'`, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user plan: %w", err)
	}
	return nil
}
