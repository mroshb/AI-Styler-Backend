package vendor

import (
	"context"
	"database/sql"
	"fmt"
)

// Store defines the vendor store interface
type Store interface {
	GetVendors(ctx context.Context) ([]Vendor, error)
	GetVendor(ctx context.Context, id string) (*Vendor, error)
	CreateVendor(ctx context.Context, vendor *Vendor) (*Vendor, error)
	UpdateVendor(ctx context.Context, vendor *Vendor) (*Vendor, error)
	DeleteVendor(ctx context.Context, id string) error
}

// store implements the vendor store
type store struct {
	db *sql.DB
}

// NewStore creates a new vendor store
func NewStore(db *sql.DB) Store {
	return &store{
		db: db,
	}
}

// GetVendors retrieves all vendors
func (s *store) GetVendors(ctx context.Context) ([]Vendor, error) {
	query := `
		SELECT id, user_id, display_name, company_name, status, created_at, updated_at
		FROM vendors
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query vendors: %w", err)
	}
	defer rows.Close()

	var vendors []Vendor
	for rows.Next() {
		var vendor Vendor
		err := rows.Scan(
			&vendor.ID,
			&vendor.UserID,
			&vendor.DisplayName,
			&vendor.CompanyName,
			&vendor.Status,
			&vendor.CreatedAt,
			&vendor.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vendor: %w", err)
		}
		vendors = append(vendors, vendor)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vendors: %w", err)
	}

	return vendors, nil
}

// GetVendor retrieves a specific vendor by ID
func (s *store) GetVendor(ctx context.Context, id string) (*Vendor, error) {
	query := `
		SELECT id, user_id, display_name, company_name, status, created_at, updated_at
		FROM vendors
		WHERE id = $1
	`

	var vendor Vendor
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&vendor.ID,
		&vendor.UserID,
		&vendor.DisplayName,
		&vendor.CompanyName,
		&vendor.Status,
		&vendor.CreatedAt,
		&vendor.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vendor not found")
		}
		return nil, fmt.Errorf("failed to get vendor: %w", err)
	}

	return &vendor, nil
}

// CreateVendor creates a new vendor
func (s *store) CreateVendor(ctx context.Context, vendor *Vendor) (*Vendor, error) {
	query := `
		INSERT INTO vendors (user_id, display_name, company_name, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		vendor.UserID,
		vendor.DisplayName,
		vendor.CompanyName,
		vendor.Status,
	).Scan(&vendor.ID, &vendor.CreatedAt, &vendor.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create vendor: %w", err)
	}

	return vendor, nil
}

// UpdateVendor updates an existing vendor
func (s *store) UpdateVendor(ctx context.Context, vendor *Vendor) (*Vendor, error) {
	query := `
		UPDATE vendors
		SET user_id = $2, display_name = $3, company_name = $4, status = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		vendor.ID,
		vendor.UserID,
		vendor.DisplayName,
		vendor.CompanyName,
		vendor.Status,
	).Scan(&vendor.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vendor not found")
		}
		return nil, fmt.Errorf("failed to update vendor: %w", err)
	}

	return vendor, nil
}

// DeleteVendor deletes a vendor
func (s *store) DeleteVendor(ctx context.Context, id string) error {
	query := `DELETE FROM vendors WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete vendor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vendor not found")
	}

	return nil
}
