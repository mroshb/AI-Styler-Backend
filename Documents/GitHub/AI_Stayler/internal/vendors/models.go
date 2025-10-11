package vendor

import (
	"time"
)

// Vendor represents a vendor entity
type Vendor struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	DisplayName *string    `json:"display_name,omitempty" db:"display_name"`
	CompanyName *string    `json:"company_name,omitempty" db:"company_name"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateVendorRequest represents the request to create a vendor
type CreateVendorRequest struct {
	UserID      string  `json:"user_id" binding:"required"`
	DisplayName *string `json:"display_name,omitempty"`
	CompanyName *string `json:"company_name,omitempty"`
	Status      string  `json:"status"`
}

// UpdateVendorRequest represents the request to update a vendor
type UpdateVendorRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	CompanyName *string `json:"company_name,omitempty"`
	Status      *string `json:"status,omitempty"`
}
