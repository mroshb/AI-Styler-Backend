package vendors

import (
	"database/sql"
)

// WireVendorService wires up the vendor service dependencies
func WireVendorService(db *sql.DB) (Service, *Handler) {
	store := NewStore(db)
	service := NewService(store)
	handler := NewHandler(service)
	return service, handler
}
