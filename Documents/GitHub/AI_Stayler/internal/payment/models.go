package payment

import (
	"time"
)

// Payment represents a payment transaction
type Payment struct {
	ID                string     `json:"id"`
	UserID            string     `json:"userId"`
	VendorID          *string    `json:"vendorId,omitempty"`
	PlanID            string     `json:"planId"`
	Amount            int64      `json:"amount"` // Amount in cents (Rials)
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	PaymentMethod     string     `json:"paymentMethod"`
	Gateway           string     `json:"gateway"`
	GatewayTrackID    *string    `json:"gatewayTrackId,omitempty"`
	GatewayRefNumber  *string    `json:"gatewayRefNumber,omitempty"`
	GatewayCardNumber *string    `json:"gatewayCardNumber,omitempty"`
	Description       string     `json:"description"`
	CallbackURL       string     `json:"callbackUrl"`
	ReturnURL         string     `json:"returnUrl"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	PaidAt            *time.Time `json:"paidAt,omitempty"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
}

// PaymentPlan represents available subscription plans
type PaymentPlan struct {
	ID                      string    `json:"id"`
	Name                    string    `json:"name"`
	DisplayName             string    `json:"displayName"`
	Description             string    `json:"description"`
	PricePerMonthCents      int64     `json:"pricePerMonthCents"`
	MonthlyConversionsLimit int       `json:"monthlyConversionsLimit"`
	MonthlyImagesLimit      int       `json:"monthlyImagesLimit"`
	Features                []string  `json:"features"`
	IsActive                bool      `json:"isActive"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

// PaymentWebhook represents webhook data from payment gateway
type PaymentWebhook struct {
	TrackID    string `json:"trackId"`
	Success    bool   `json:"success"`
	Status     int    `json:"status"`
	OrderID    string `json:"orderId"`
	CardNumber string `json:"cardNumber,omitempty"`
	Amount     int64  `json:"amount,omitempty"`
}

// CreatePaymentRequest represents the request to create a payment
type CreatePaymentRequest struct {
	PlanID      string `json:"planId" binding:"required"`
	ReturnURL   string `json:"returnUrl" binding:"required"`
	Description string `json:"description,omitempty"`
}

// CreatePaymentResponse represents the response for creating a payment
type CreatePaymentResponse struct {
	PaymentID  string    `json:"paymentId"`
	GatewayURL string    `json:"gatewayUrl"`
	TrackID    string    `json:"trackId"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// PaymentStatusResponse represents the response for payment status
type PaymentStatusResponse struct {
	PaymentID   string       `json:"paymentId"`
	Status      string       `json:"status"`
	Amount      int64        `json:"amount"`
	PlanName    string       `json:"planName"`
	PaidAt      *time.Time   `json:"paidAt,omitempty"`
	GatewayInfo *GatewayInfo `json:"gatewayInfo,omitempty"`
}

// GatewayInfo represents gateway-specific information
type GatewayInfo struct {
	TrackID    string `json:"trackId,omitempty"`
	RefNumber  string `json:"refNumber,omitempty"`
	CardNumber string `json:"cardNumber,omitempty"`
	Gateway    string `json:"gateway"`
}

// ZarinpalRequest represents Zarinpal payment request
type ZarinpalRequest struct {
	Merchant     string `json:"merchant"`
	Amount       int64  `json:"amount"`
	CallbackURL  string `json:"callbackUrl"`
	Description  string `json:"description,omitempty"`
	OrderID      string `json:"orderId,omitempty"`
	Mobile       string `json:"mobile,omitempty"`
	NationalCode string `json:"nationalCode,omitempty"`
}

// ZarinpalResponse represents Zarinpal payment response
type ZarinpalResponse struct {
	TrackID string `json:"trackId"`
	Result  int    `json:"result"`
	Message string `json:"message"`
}

// ZarinpalVerifyRequest represents Zarinpal verification request
type ZarinpalVerifyRequest struct {
	Merchant string `json:"merchant"`
	TrackID  string `json:"trackId"`
}

// ZarinpalVerifyResponse represents Zarinpal verification response
type ZarinpalVerifyResponse struct {
	PaidAt      string `json:"paidAt,omitempty"`
	CardNumber  string `json:"cardNumber,omitempty"`
	Status      int    `json:"status"`
	Amount      int64  `json:"amount"`
	RefNumber   string `json:"refNumber,omitempty"`
	Description string `json:"description,omitempty"`
	OrderID     string `json:"orderId,omitempty"`
	Result      int    `json:"result"`
	Message     string `json:"message"`
}

// PaymentHistoryRequest represents the request to get payment history
type PaymentHistoryRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Status   string `json:"status" form:"status"`
}

// PaymentHistoryItem represents a single item in payment history
type PaymentHistoryItem struct {
	PaymentID        string     `json:"paymentId"`
	UserID           string     `json:"userId"`
	VendorID         *string    `json:"vendorId,omitempty"`
	PlanID           string     `json:"planId"`
	Amount           int64      `json:"amount"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	PaymentMethod    string     `json:"paymentMethod"`
	Gateway          string     `json:"gateway"`
	GatewayTrackID   *string    `json:"gatewayTrackId,omitempty"`
	GatewayRefNumber *string    `json:"gatewayRefNumber,omitempty"`
	Description      string     `json:"description"`
	CreatedAt        time.Time  `json:"createdAt"`
	PaidAt           *time.Time `json:"paidAt,omitempty"`
	PlanName         string     `json:"planName"`
	PlanDisplayName  string     `json:"planDisplayName"`
}

// PaymentHistoryResponse represents the response for payment history
type PaymentHistoryResponse struct {
	Payments   []PaymentHistoryItem `json:"payments"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalPages int                  `json:"totalPages"`
}

// Plan constants
const (
	PlanFree     = "free"
	PlanBasic    = "basic"
	PlanAdvanced = "advanced"
)

// Payment status constants
const (
	PaymentStatusPending   = "pending"
	PaymentStatusCompleted = "completed"
	PaymentStatusFailed    = "failed"
	PaymentStatusCancelled = "cancelled"
	PaymentStatusExpired   = "expired"
)

// Payment method constants
const (
	PaymentMethodZarinpal = "zarinpal"
)

// Gateway constants
const (
	GatewayZarinpal = "zarinpal"
)

// Currency constants
const (
	CurrencyIRR = "IRR"
)

// Zarinpal result codes
const (
	ZarinpalSuccess             = 100
	ZarinpalMerchantNotFound    = 102
	ZarinpalMerchantInactive    = 103
	ZarinpalMerchantInvalid     = 104
	ZarinpalAmountTooLow        = 105
	ZarinpalCallbackInvalid     = 106
	ZarinpalAmountExceeded      = 113
	ZarinpalNationalCodeInvalid = 114
	ZarinpalAlreadyVerified     = 201
	ZarinpalNotPaid             = 202
	ZarinpalTrackIDInvalid      = 203
)

// Zarinpal status codes
const (
	ZarinpalStatusPending             = -1
	ZarinpalStatusInternalErr         = -2
	ZarinpalStatusPaid                = 1
	ZarinpalStatusPaidUnverified      = 2
	ZarinpalStatusCancelled           = 3
	ZarinpalStatusCardInvalid         = 4
	ZarinpalStatusInsufficientFunds   = 5
	ZarinpalStatusWrongPin            = 6
	ZarinpalStatusTooManyRequests     = 7
	ZarinpalStatusDailyLimitExceeded  = 8
	ZarinpalStatusDailyAmountExceeded = 9
	ZarinpalStatusInvalidIssuer       = 10
	ZarinpalStatusSwitchError         = 11
	ZarinpalStatusCardUnavailable     = 12
	ZarinpalStatusRefunded            = 15
	ZarinpalStatusRefunding           = 16
	ZarinpalStatusReversed            = 18
)
