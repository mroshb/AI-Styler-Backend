package user

import (
	"database/sql"
)

// WireUserService creates a user service with all dependencies
func WireUserService(db *sql.DB) (*Service, *Handler) {
	// Create store
	store := NewDBStore(db)

	// Create mock dependencies (replace with real implementations in production)
	auditLogger := NewMockAuditLogger()

	// Create service
	service := NewService(store, auditLogger)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}

// WireUserServiceWithMocks creates a user service with mock dependencies for testing
func WireUserServiceWithMocks(store Store) (*Service, *Handler) {
	// Create mock dependencies
	auditLogger := NewMockAuditLogger()

	// Create service
	service := NewService(store, auditLogger)

	// Create handler
	handler := NewHandler(service)

	return service, handler
}
