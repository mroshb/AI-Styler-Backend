package vendor

import (
	"context"
	"errors"
)

// Service defines the vendor service interface
type Service interface {
	GetVendors(ctx context.Context) ([]Vendor, error)
	GetVendor(ctx context.Context, id string) (*Vendor, error)
	CreateVendor(ctx context.Context, req CreateVendorRequest) (*Vendor, error)
	UpdateVendor(ctx context.Context, id string, req UpdateVendorRequest) (*Vendor, error)
	DeleteVendor(ctx context.Context, id string) error
}

// service implements the vendor service
type service struct {
	store Store
}

// NewService creates a new vendor service
func NewService(store Store) Service {
	return &service{
		store: store,
	}
}

// GetVendors retrieves all vendors
func (s *service) GetVendors(ctx context.Context) ([]Vendor, error) {
	return s.store.GetVendors(ctx)
}

// GetVendor retrieves a specific vendor by ID
func (s *service) GetVendor(ctx context.Context, id string) (*Vendor, error) {
	if id == "" {
		return nil, errors.New("vendor ID is required")
	}

	return s.store.GetVendor(ctx, id)
}

// CreateVendor creates a new vendor
func (s *service) CreateVendor(ctx context.Context, req CreateVendorRequest) (*Vendor, error) {
	if req.UserID == "" {
		return nil, errors.New("user ID is required")
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	vendor := &Vendor{
		UserID:      req.UserID,
		DisplayName: req.DisplayName,
		CompanyName: req.CompanyName,
		Status:      status,
	}

	return s.store.CreateVendor(ctx, vendor)
}

// UpdateVendor updates an existing vendor
func (s *service) UpdateVendor(ctx context.Context, id string, req UpdateVendorRequest) (*Vendor, error) {
	if id == "" {
		return nil, errors.New("vendor ID is required")
	}

	// Get existing vendor
	vendor, err := s.store.GetVendor(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.DisplayName != nil {
		vendor.DisplayName = req.DisplayName
	}
	if req.CompanyName != nil {
		vendor.CompanyName = req.CompanyName
	}
	if req.Status != nil {
		vendor.Status = *req.Status
	}

	return s.store.UpdateVendor(ctx, vendor)
}

// DeleteVendor deletes a vendor
func (s *service) DeleteVendor(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("vendor ID is required")
	}

	return s.store.DeleteVendor(ctx, id)
}
