package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// postgresStore implements the PaymentStore interface using PostgreSQL
type postgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *sql.DB) PaymentStore {
	return &postgresStore{db: db}
}

// CreatePayment creates a new payment record
func (s *postgresStore) CreatePayment(ctx context.Context, payment Payment) (Payment, error) {
	query := `
		INSERT INTO payments (id, user_id, vendor_id, plan_id, amount, currency, status,
		                     payment_method, gateway, gateway_track_id, description,
		                     callback_url, return_url, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, user_id, vendor_id, plan_id, amount, currency, status,
		          payment_method, gateway, gateway_track_id, gateway_ref_number,
		          gateway_card_number, description, callback_url, return_url,
		          created_at, updated_at, paid_at, expires_at`

	var createdPayment Payment
	err := s.db.QueryRowContext(ctx, query,
		payment.ID, payment.UserID, payment.VendorID, payment.PlanID, payment.Amount,
		payment.Currency, payment.Status, payment.PaymentMethod, payment.Gateway,
		payment.GatewayTrackID, payment.Description, payment.CallbackURL,
		payment.ReturnURL, payment.ExpiresAt,
	).Scan(
		&createdPayment.ID, &createdPayment.UserID, &createdPayment.VendorID,
		&createdPayment.PlanID, &createdPayment.Amount, &createdPayment.Currency,
		&createdPayment.Status, &createdPayment.PaymentMethod, &createdPayment.Gateway,
		&createdPayment.GatewayTrackID, &createdPayment.GatewayRefNumber,
		&createdPayment.GatewayCardNumber, &createdPayment.Description,
		&createdPayment.CallbackURL, &createdPayment.ReturnURL,
		&createdPayment.CreatedAt, &createdPayment.UpdatedAt,
		&createdPayment.PaidAt, &createdPayment.ExpiresAt,
	)
	if err != nil {
		return Payment{}, fmt.Errorf("failed to create payment: %w", err)
	}

	return createdPayment, nil
}

// GetPayment retrieves a payment by ID
func (s *postgresStore) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	query := `
		SELECT id, user_id, vendor_id, plan_id, amount, currency, status,
		       payment_method, gateway, gateway_track_id, gateway_ref_number,
		       gateway_card_number, description, callback_url, return_url,
		       created_at, updated_at, paid_at, expires_at
		FROM payments 
		WHERE id = $1`

	var payment Payment
	err := s.db.QueryRowContext(ctx, query, paymentID).Scan(
		&payment.ID, &payment.UserID, &payment.VendorID, &payment.PlanID,
		&payment.Amount, &payment.Currency, &payment.Status, &payment.PaymentMethod,
		&payment.Gateway, &payment.GatewayTrackID, &payment.GatewayRefNumber,
		&payment.GatewayCardNumber, &payment.Description, &payment.CallbackURL,
		&payment.ReturnURL, &payment.CreatedAt, &payment.UpdatedAt,
		&payment.PaidAt, &payment.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, errors.New("payment not found")
		}
		return Payment{}, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// GetPaymentByTrackID retrieves a payment by gateway track ID
func (s *postgresStore) GetPaymentByTrackID(ctx context.Context, trackID string) (Payment, error) {
	query := `
		SELECT id, user_id, vendor_id, plan_id, amount, currency, status,
		       payment_method, gateway, gateway_track_id, gateway_ref_number,
		       gateway_card_number, description, callback_url, return_url,
		       created_at, updated_at, paid_at, expires_at
		FROM payments 
		WHERE gateway_track_id = $1`

	var payment Payment
	err := s.db.QueryRowContext(ctx, query, trackID).Scan(
		&payment.ID, &payment.UserID, &payment.VendorID, &payment.PlanID,
		&payment.Amount, &payment.Currency, &payment.Status, &payment.PaymentMethod,
		&payment.Gateway, &payment.GatewayTrackID, &payment.GatewayRefNumber,
		&payment.GatewayCardNumber, &payment.Description, &payment.CallbackURL,
		&payment.ReturnURL, &payment.CreatedAt, &payment.UpdatedAt,
		&payment.PaidAt, &payment.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, errors.New("payment not found")
		}
		return Payment{}, fmt.Errorf("failed to get payment by track ID: %w", err)
	}

	return payment, nil
}

// UpdatePayment updates a payment
func (s *postgresStore) UpdatePayment(ctx context.Context, paymentID string, updates map[string]interface{}) (Payment, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetPayment(ctx, paymentID)
	}

	query := fmt.Sprintf(`
		UPDATE payments 
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING id, user_id, vendor_id, plan_id, amount, currency, status,
		          payment_method, gateway, gateway_track_id, gateway_ref_number,
		          gateway_card_number, description, callback_url, return_url,
		          created_at, updated_at, paid_at, expires_at`,
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf(`
			UPDATE payments 
			SET %s, updated_at = NOW()
			WHERE id = $%d
			RETURNING id, user_id, vendor_id, plan_id, amount, currency, status,
			          payment_method, gateway, gateway_track_id, gateway_ref_number,
			          gateway_card_number, description, callback_url, return_url,
			          created_at, updated_at, paid_at, expires_at`,
			fmt.Sprintf("%s", setParts[i]), argIndex)
	}

	args = append(args, paymentID)

	var payment Payment
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&payment.ID, &payment.UserID, &payment.VendorID, &payment.PlanID,
		&payment.Amount, &payment.Currency, &payment.Status, &payment.PaymentMethod,
		&payment.Gateway, &payment.GatewayTrackID, &payment.GatewayRefNumber,
		&payment.GatewayCardNumber, &payment.Description, &payment.CallbackURL,
		&payment.ReturnURL, &payment.CreatedAt, &payment.UpdatedAt,
		&payment.PaidAt, &payment.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, errors.New("payment not found")
		}
		return Payment{}, fmt.Errorf("failed to update payment: %w", err)
	}

	return payment, nil
}

// GetPaymentHistory retrieves payment history for a user
func (s *postgresStore) GetPaymentHistory(ctx context.Context, userID string, req PaymentHistoryRequest) (PaymentHistoryResponse, error) {
	query := `
		SELECT p.id, p.user_id, p.vendor_id, p.plan_id, p.amount, p.currency, p.status,
		       p.payment_method, p.gateway, p.gateway_track_id, p.gateway_ref_number,
		       p.description, p.created_at, p.paid_at, pp.name as plan_name,
		       pp.display_name as plan_display_name
		FROM payments p
		LEFT JOIN payment_plans pp ON p.plan_id = pp.id
		WHERE p.user_id = $1 OR p.vendor_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`

	offset := (req.Page - 1) * req.PageSize
	rows, err := s.db.QueryContext(ctx, query, userID, req.PageSize, offset)
	if err != nil {
		return PaymentHistoryResponse{}, fmt.Errorf("failed to get payment history: %w", err)
	}
	defer rows.Close()

	var payments []PaymentHistoryItem
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
		payments = append(payments, item)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM payments WHERE user_id = $1 OR vendor_id = $1`
	var total int
	err = s.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return PaymentHistoryResponse{}, fmt.Errorf("failed to get total count: %w", err)
	}

	return PaymentHistoryResponse{
		Payments: payments,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetPlan retrieves a plan by ID
func (s *postgresStore) GetPlan(ctx context.Context, planID string) (PaymentPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_per_month_cents,
		       monthly_conversions_limit, monthly_images_limit, features, is_active,
		       created_at, updated_at
		FROM payment_plans 
		WHERE id = $1`

	var plan PaymentPlan
	err := s.db.QueryRowContext(ctx, query, planID).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description,
		&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit,
		&plan.MonthlyImagesLimit, pq.Array(&plan.Features), &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentPlan{}, errors.New("plan not found")
		}
		return PaymentPlan{}, fmt.Errorf("failed to get plan: %w", err)
	}

	return plan, nil
}

// GetAllPlans retrieves all available plans
func (s *postgresStore) GetAllPlans(ctx context.Context) ([]PaymentPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_per_month_cents,
		       monthly_conversions_limit, monthly_images_limit, features, is_active,
		       created_at, updated_at
		FROM payment_plans 
		WHERE is_active = true
		ORDER BY price_per_month_cents ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}
	defer rows.Close()

	var plans []PaymentPlan
	for rows.Next() {
		var plan PaymentPlan
		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description,
			&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit,
			&plan.MonthlyImagesLimit, pq.Array(&plan.Features), &plan.IsActive,
			&plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

// CreatePlan creates a new payment plan
func (s *postgresStore) CreatePlan(ctx context.Context, plan PaymentPlan) (PaymentPlan, error) {
	query := `
		INSERT INTO payment_plans (id, name, display_name, description, price_per_month_cents,
		                         monthly_conversions_limit, monthly_images_limit, features, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, display_name, description, price_per_month_cents,
		          monthly_conversions_limit, monthly_images_limit, features, is_active,
		          created_at, updated_at`

	var createdPlan PaymentPlan
	err := s.db.QueryRowContext(ctx, query,
		plan.ID, plan.Name, plan.DisplayName, plan.Description, plan.PricePerMonthCents,
		plan.MonthlyConversionsLimit, plan.MonthlyImagesLimit, pq.Array(plan.Features), plan.IsActive,
	).Scan(
		&createdPlan.ID, &createdPlan.Name, &createdPlan.DisplayName, &createdPlan.Description,
		&createdPlan.PricePerMonthCents, &createdPlan.MonthlyConversionsLimit,
		&createdPlan.MonthlyImagesLimit, pq.Array(&createdPlan.Features), &createdPlan.IsActive,
		&createdPlan.CreatedAt, &createdPlan.UpdatedAt,
	)
	if err != nil {
		return PaymentPlan{}, fmt.Errorf("failed to create plan: %w", err)
	}

	return createdPlan, nil
}

// UpdatePlan updates a payment plan
func (s *postgresStore) UpdatePlan(ctx context.Context, planID string, updates map[string]interface{}) (PaymentPlan, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetPlan(ctx, planID)
	}

	query := fmt.Sprintf(`
		UPDATE payment_plans 
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING id, name, display_name, description, price_per_month_cents,
		          monthly_conversions_limit, monthly_images_limit, features, is_active,
		          created_at, updated_at`,
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf(`
			UPDATE payment_plans 
			SET %s, updated_at = NOW()
			WHERE id = $%d
			RETURNING id, name, display_name, description, price_per_month_cents,
			          monthly_conversions_limit, monthly_images_limit, features, is_active,
			          created_at, updated_at`,
			fmt.Sprintf("%s", setParts[i]), argIndex)
	}

	args = append(args, planID)

	var plan PaymentPlan
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description,
		&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit,
		&plan.MonthlyImagesLimit, pq.Array(&plan.Features), &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentPlan{}, errors.New("plan not found")
		}
		return PaymentPlan{}, fmt.Errorf("failed to update plan: %w", err)
	}

	return plan, nil
}

// DeactivateUserPlan deactivates a user's plan
func (s *postgresStore) DeactivateUserPlan(ctx context.Context, userID string) error {
	query := `
		UPDATE user_plans 
		SET status = 'cancelled', updated_at = NOW()
		WHERE (user_id = $1 OR vendor_id = $1) AND status = 'active'`

	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user plan: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("no active plan found")
	}

	return nil
}

// GetUserActivePlan retrieves the user's active plan
func (s *postgresStore) GetUserActivePlan(ctx context.Context, userID string) (PaymentPlan, error) {
	query := `
		SELECT pp.id, pp.name, pp.display_name, pp.description, pp.price_per_month_cents,
		       pp.monthly_conversions_limit, pp.monthly_images_limit, pp.features,
		       pp.is_active, pp.created_at, pp.updated_at
		FROM payment_plans pp
		JOIN user_plans up ON pp.id = up.plan_id
		WHERE up.user_id = $1 AND up.status = 'active'
		UNION ALL
		SELECT pp.id, pp.name, pp.display_name, pp.description, pp.price_per_month_cents,
		       pp.monthly_conversions_limit, pp.monthly_images_limit, pp.features,
		       pp.is_active, pp.created_at, pp.updated_at
		FROM payment_plans pp
		JOIN user_plans up ON pp.id = up.plan_id
		WHERE up.vendor_id = $1 AND up.status = 'active'
		LIMIT 1`

	var plan PaymentPlan
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Description,
		&plan.PricePerMonthCents, &plan.MonthlyConversionsLimit,
		&plan.MonthlyImagesLimit, pq.Array(&plan.Features), &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentPlan{}, errors.New("no active plan found")
		}
		return PaymentPlan{}, fmt.Errorf("failed to get user active plan: %w", err)
	}

	return plan, nil
}

// ActivateUserPlan activates a user's plan
func (s *postgresStore) ActivateUserPlan(ctx context.Context, userID, planID, paymentID string) error {
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if user already has an active plan
	checkQuery := `
		SELECT id FROM user_plans 
		WHERE (user_id = $1 OR vendor_id = $1) AND status = 'active'`
	var existingPlanID sql.NullString
	err = tx.QueryRowContext(ctx, checkQuery, userID).Scan(&existingPlanID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing plan: %w", err)
	}

	if existingPlanID.Valid {
		// Deactivate existing plan
		deactivateQuery := `
			UPDATE user_plans 
			SET status = 'cancelled', updated_at = NOW()
			WHERE id = $1`
		_, err = tx.ExecContext(ctx, deactivateQuery, existingPlanID.String)
		if err != nil {
			return fmt.Errorf("failed to deactivate existing plan: %w", err)
		}
	}

	// Create new user plan
	insertQuery := `
		INSERT INTO user_plans (user_id, vendor_id, plan_id, payment_id, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (user_id, plan_id) 
		DO UPDATE SET status = 'active', payment_id = $4, updated_at = NOW()`

	// Determine if this is a user or vendor
	userQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	var isUser bool
	err = tx.QueryRowContext(ctx, userQuery, userID).Scan(&isUser)
	if err != nil {
		return fmt.Errorf("failed to check user type: %w", err)
	}

	if isUser {
		_, err = tx.ExecContext(ctx, insertQuery, userID, nil, planID, paymentID)
	} else {
		_, err = tx.ExecContext(ctx, insertQuery, nil, userID, planID, paymentID)
	}
	if err != nil {
		return fmt.Errorf("failed to activate user plan: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetPaymentStats retrieves payment statistics
func (s *postgresStore) GetPaymentStats(ctx context.Context) (int, int64, error) {
	query := `
		SELECT 
			COUNT(*) as total_payments,
			COALESCE(SUM(amount) FILTER (WHERE status = 'completed'), 0) as total_revenue
		FROM payments`

	var totalPayments int
	var totalRevenue int64
	err := s.db.QueryRowContext(ctx, query).Scan(&totalPayments, &totalRevenue)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get payment stats: %w", err)
	}

	return totalPayments, totalRevenue, nil
}
