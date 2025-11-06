package payment

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	BazaarPayBaseURL = "https://api.bazaar-pay.ir/badje/v1"
)

// BazaarPayService provides BazaarPay payment gateway functionality
type BazaarPayService struct {
	db          *sql.DB
	store       PaymentStore
	apiKey      string
	destination string
	redirectURL string
}

// NewBazaarPayService creates a new BazaarPay service
func NewBazaarPayService(db *sql.DB) *BazaarPayService {
	apiKey := os.Getenv("BAZAARPAY_API_KEY")
	destination := os.Getenv("BAZAARPAY_DESTINATION")
	if destination == "" {
		destination = "mynaa_bazaar" // مقدار پیش‌فرض
	}
	redirectURL := os.Getenv("BAZAARPAY_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "https://yourdomain.com/api/payments/bazaarpay/status"
	}

	return &BazaarPayService{
		db:          db,
		store:       NewPaymentStore(db),
		apiKey:      apiKey,
		destination: destination,
		redirectURL: redirectURL,
	}
}

// InitCheckout - ایجاد checkout token
func (s *BazaarPayService) InitCheckout(amount int64, serviceName string) (*BazaarCheckoutInitResponse, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("BAZAARPAY_API_KEY environment variable is not set")
	}

	// ساخت request body
	requestBody := BazaarCheckoutInitRequest{
		Amount:      amount,
		Destination: s.destination,
		ServiceName: serviceName,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// ارسال درخواست
	apiURL := BazaarPayBaseURL + "/checkout/init/"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bazaarpay request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var checkoutResp BazaarCheckoutInitResponse
	if err = json.Unmarshal(body, &checkoutResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &checkoutResp, nil
}

// BuildPaymentURL - ساخت URL پرداخت
func (s *BazaarPayService) BuildPaymentURL(checkoutToken, phone string) string {
	baseURL := "https://app.bazaar-pay.ir/payment"

	params := url.Values{}
	params.Add("token", checkoutToken)

	if phone != "" {
		params.Add("phone", phone)
	}

	if s.redirectURL != "" {
		params.Add("redirect_url", s.redirectURL)
	}

	return baseURL + "?" + params.Encode()
}

// TraceCheckout - پیگیری وضعیت
func (s *BazaarPayService) TraceCheckout(checkoutToken string) (*BazaarTraceResponse, error) {
	if checkoutToken == "" {
		return nil, fmt.Errorf("checkout token cannot be empty")
	}

	requestBody := BazaarTraceRequest{
		CheckoutToken: checkoutToken,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling trace request: %w", err)
	}

	apiURL := BazaarPayBaseURL + "/trace/"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating trace request: %w", err)
	}

	// ⚠️ trace نیاز به authorization ندارد
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending trace request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trace request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var traceResp BazaarTraceResponse
	if err = json.Unmarshal(body, &traceResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling trace response: %w", err)
	}

	return &traceResp, nil
}

// CommitCheckout - تایید نهایی پرداخت
func (s *BazaarPayService) CommitCheckout(checkoutToken string) error {
	if checkoutToken == "" {
		return fmt.Errorf("checkout token cannot be empty")
	}

	if s.apiKey == "" {
		return fmt.Errorf("BAZAARPAY_API_KEY environment variable is not set")
	}

	requestBody := BazaarCommitRequest{
		CheckoutToken: checkoutToken,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling commit request: %w", err)
	}

	apiURL := BazaarPayBaseURL + "/commit/"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating commit request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending commit request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// ✅ 204 No Content = success
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("commit request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RefundCheckout - بازگشت پول
func (s *BazaarPayService) RefundCheckout(checkoutToken string, amount *int64) error {
	if checkoutToken == "" {
		return fmt.Errorf("checkout token cannot be empty")
	}

	if s.apiKey == "" {
		return fmt.Errorf("BAZAARPAY_API_KEY environment variable is not set")
	}

	requestBody := BazaarRefundRequest{
		CheckoutToken: checkoutToken,
		Amount:        amount, // nil = بازگشت کامل
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling refund request: %w", err)
	}

	apiURL := BazaarPayBaseURL + "/refund/"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating refund request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending refund request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refund request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreatePrePayment - ذخیره pre-payment
func (s *BazaarPayService) CreatePrePayment(ctx context.Context, userID, orderID, segment string, segmentID int, planID *string, amount *int64) error {
	query := `
		INSERT INTO pre_payments (user_id, order_id, segment, segment_id, plan_id, amount, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (order_id) DO NOTHING`

	_, err := s.db.ExecContext(ctx, query, userID, orderID, segment, segmentID, planID, amount)
	if err != nil {
		return fmt.Errorf("error saving pre-payment: %w", err)
	}

	return nil
}

// GetPrePayment - دریافت pre-payment
func (s *BazaarPayService) GetPrePayment(ctx context.Context, orderID string) (*PrePayment, error) {
	query := `
		SELECT id, user_id, order_id, segment, segment_id, plan_id, amount, created_at
		FROM pre_payments
		WHERE order_id = $1`

	var prePayment PrePayment
	err := s.db.QueryRowContext(ctx, query, orderID).Scan(
		&prePayment.ID,
		&prePayment.UserID,
		&prePayment.OrderID,
		&prePayment.Segment,
		&prePayment.SegmentID,
		&prePayment.PlanID,
		&prePayment.Amount,
		&prePayment.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pre-payment not found")
		}
		return nil, fmt.Errorf("error getting pre-payment: %w", err)
	}

	return &prePayment, nil
}

// CreatePlanPayment - ذخیره پرداخت موفق پلن
func (s *BazaarPayService) CreatePlanPayment(ctx context.Context, payment *PlanPayment) error {
	query := `
		INSERT INTO plan_payments (
			user_id, order_id, ref_number, amount, card_number,
			status, result, message, description, segment, segment_id, paid_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (order_id) DO NOTHING`

	_, err := s.db.ExecContext(ctx, query,
		payment.UserID,
		payment.OrderID,
		payment.RefNumber,
		payment.Amount,
		payment.CardNumber,
		payment.Status,
		payment.Result,
		payment.Message,
		payment.Description,
		payment.Segment,
		payment.SegmentID,
		payment.PaidAt,
	)
	if err != nil {
		return fmt.Errorf("error saving plan payment: %w", err)
	}

	return nil
}

// GetPlanPayment - دریافت پرداخت پلن
func (s *BazaarPayService) GetPlanPayment(ctx context.Context, orderID string) (*PlanPayment, error) {
	query := `
		SELECT id, user_id, order_id, ref_number, amount, card_number,
		       status, result, message, description, segment, segment_id, paid_at, created_at
		FROM plan_payments
		WHERE order_id = $1`

	var planPayment PlanPayment
	err := s.db.QueryRowContext(ctx, query, orderID).Scan(
		&planPayment.ID,
		&planPayment.UserID,
		&planPayment.OrderID,
		&planPayment.RefNumber,
		&planPayment.Amount,
		&planPayment.CardNumber,
		&planPayment.Status,
		&planPayment.Result,
		&planPayment.Message,
		&planPayment.Description,
		&planPayment.Segment,
		&planPayment.SegmentID,
		&planPayment.PaidAt,
		&planPayment.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("plan payment not found")
		}
		return nil, fmt.Errorf("error getting plan payment: %w", err)
	}

	return &planPayment, nil
}

// PayForPlan - ایجاد لینک پرداخت برای خرید پلن
func (s *BazaarPayService) PayForPlan(ctx context.Context, userID string, planID string, planName string, planPrice int64, userPhone string) (string, string, error) {
	// 1. ایجاد checkout
	serviceName := fmt.Sprintf("پلن %s - کاربر %s", planName, userID)
	checkout, err := s.InitCheckout(planPrice, serviceName)
	if err != nil {
		return "", "", fmt.Errorf("error creating checkout: %w", err)
	}

	// 2. ساخت URL پرداخت
	paymentURL := s.BuildPaymentURL(checkout.CheckoutToken, userPhone)

	// 3. ذخیره pre-payment
	segmentID := 0 // برای segment_id از 0 استفاده می‌کنیم
	err = s.CreatePrePayment(ctx, userID, checkout.CheckoutToken, "plan", segmentID, &planID, &planPrice)
	if err != nil {
		return "", "", fmt.Errorf("error saving pre-payment: %w", err)
	}

	return checkout.CheckoutToken, paymentURL, nil
}

// ProcessBazaarPayment - پردازش پرداخت موفق
func (s *BazaarPayService) ProcessBazaarPayment(ctx context.Context, checkoutToken string) error {
	// 1. یافتن pre-payment
	prePayment, err := s.GetPrePayment(ctx, checkoutToken)
	if err != nil {
		return fmt.Errorf("pre-payment not found: %w", err)
	}

	// 2. بررسی تکراری نبودن
	existingPayment, err := s.GetPlanPayment(ctx, checkoutToken)
	if err == nil && existingPayment != nil {
		return nil // قبلاً پردازش شده
	}

	// 3. دریافت amount از prePayment
	var amount int64
	if prePayment.Amount != nil {
		amount = *prePayment.Amount
	} else {
		return fmt.Errorf("amount not found in pre-payment")
	}

	// 4. ذخیره در دیتابیس (card_number از trace response در دسترس نیست)
	var cardNumber *string

	planPayment := &PlanPayment{
		UserID:     prePayment.UserID,
		OrderID:    checkoutToken,
		RefNumber:  &checkoutToken,
		Amount:     amount,
		CardNumber: cardNumber,
		Status:     100,
		Result:     100,
		Message:    stringPtrHelper("Successful"),
		Segment:    prePayment.Segment,
		SegmentID:  prePayment.SegmentID,
		PaidAt:     stringPtrHelper(time.Now().Format("2006-01-02 15:04:05")),
	}

	if err := s.CreatePlanPayment(ctx, planPayment); err != nil {
		return fmt.Errorf("error saving payment: %w", err)
	}

	// 5. فعال کردن plan اگر planID موجود باشد
	if prePayment.PlanID != nil && *prePayment.PlanID != "" {
		// ایجاد payment record در جدول payments برای سازگاری با سیستم موجود
		paymentID := generatePaymentID()
		payment := Payment{
			ID:                paymentID,
			UserID:            prePayment.UserID,
			PlanID:            *prePayment.PlanID,
			Amount:            amount,
			Currency:          CurrencyIRR,
			Status:            PaymentStatusCompleted,
			PaymentMethod:     GatewayBazaarPay,
			Gateway:           GatewayBazaarPay,
			GatewayTrackID:    &checkoutToken,
			GatewayRefNumber:  &checkoutToken,
			GatewayCardNumber: cardNumber,
			Description:       "BazaarPay payment",
			CallbackURL:       "",
			ReturnURL:         s.redirectURL,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			PaidAt:            timePtr(time.Now()),
		}

		_, err = s.store.CreatePayment(ctx, payment)
		if err != nil {
			// لاگ کردن خطا ولی ادامه بده
			fmt.Printf("Error creating payment record: %v\n", err)
		} else {
			// فعال کردن plan
			err = s.store.ActivateUserPlan(ctx, prePayment.UserID, *prePayment.PlanID, paymentID)
			if err != nil {
				// لاگ کردن خطا ولی ادامه بده
				fmt.Printf("Error activating user plan: %v\n", err)
			}
		}
	}

	// 6. Commit کردن تراکنش
	if err := s.CommitCheckout(checkoutToken); err != nil {
		// لاگ کردن خطا ولی ادامه بده
		fmt.Printf("Error committing checkout: %v\n", err)
	}

	return nil
}

// Helper function to convert string to *string
func stringPtrHelper(s string) *string {
	return &s
}

