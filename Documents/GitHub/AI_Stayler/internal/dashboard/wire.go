package dashboard

import (
	"database/sql"

	"github.com/google/wire"
)

// ProviderSet is the dashboard service provider set
var ProviderSet = wire.NewSet(
	WireNewStoreImpl,
	WireNewDashboardService,
	WireNewDashboardHandler,
)

// WireNewStoreImpl creates a new dashboard store implementation for wire
func WireNewStoreImpl(db *sql.DB) Store {
	return &StoreImpl{db: db}
}

// WireNewDashboardService creates a new dashboard service for wire
func WireNewDashboardService(
	store Store,
	userService UserService,
	conversionService ConversionService,
	vendorService VendorService,
	paymentService PaymentService,
	cache Cache,
	metricsCollector MetricsCollector,
	auditLogger AuditLogger,
) *Service {
	return &Service{
		store:             store,
		userService:       userService,
		conversionService: conversionService,
		vendorService:     vendorService,
		paymentService:    paymentService,
		cache:             cache,
		metricsCollector:  metricsCollector,
		auditLogger:       auditLogger,
	}
}

// WireNewDashboardHandler creates a new dashboard handler for wire
func WireNewDashboardHandler(service *Service) *Handler {
	return &Handler{service: service}
}
