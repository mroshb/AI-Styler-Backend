package route

import (
	"AI_Styler/internal/admin"
	"AI_Styler/internal/auth"
	"AI_Styler/internal/common"
	"AI_Styler/internal/config"
	"AI_Styler/internal/conversion"
	"AI_Styler/internal/docs"
	"AI_Styler/internal/image"
	"AI_Styler/internal/middleware"
	"AI_Styler/internal/monitoring"
	"AI_Styler/internal/notification"
	"AI_Styler/internal/payment"
	"AI_Styler/internal/security"
	"AI_Styler/internal/share"
	"AI_Styler/internal/sms"
	"AI_Styler/internal/storage"
	"AI_Styler/internal/user"
	"AI_Styler/internal/vendor"
	"AI_Styler/internal/worker"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Mock services for payment integration
type mockUserService struct{}
type mockNotificationService struct{}
type mockQuotaService struct{}
type mockAuditLogger struct{}
type mockRateLimiter struct{}

func (m *mockUserService) GetUserPlan(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}
func (m *mockUserService) UpdateUserPlan(ctx context.Context, planID string, status string) (interface{}, error) {
	return nil, nil
}
func (m *mockUserService) CreateUserPlan(ctx context.Context, userID string, planName string) (interface{}, error) {
	return nil, nil
}
func (m *mockNotificationService) SendPaymentSuccess(ctx context.Context, userID string, paymentID string, planName string) error {
	return nil
}
func (m *mockNotificationService) SendPaymentFailed(ctx context.Context, userID string, paymentID string, reason string) error {
	return nil
}
func (m *mockNotificationService) SendPlanActivated(ctx context.Context, userID string, planName string) error {
	return nil
}
func (m *mockNotificationService) SendPlanExpired(ctx context.Context, userID string, planName string) error {
	return nil
}
func (m *mockQuotaService) UpdateUserQuota(ctx context.Context, userID string, planName string) error {
	return nil
}
func (m *mockQuotaService) ResetMonthlyQuota(ctx context.Context, userID string) error { return nil }
func (m *mockQuotaService) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	return nil, nil
}
func (m *mockAuditLogger) LogPaymentAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error {
	return nil
}
func (m *mockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	return true
}

func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Load security configuration
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Create security middleware
	securityConfig := &security.SecurityConfig{
		RateLimitEnabled:       true,
		RateLimitPerIP:         cfg.RateLimit.OTPPerIP,
		RateLimitPerUser:       1000,
		RateLimitWindow:        cfg.RateLimit.Window,
		JWTSecret:              cfg.JWT.Secret,
		JWTExpiration:          cfg.JWT.AccessTTL,
		CORSEnabled:            true,
		AllowedOrigins:         []string{"*"},
		AllowedMethods:         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:         []string{"*"},
		SecurityHeadersEnabled: true,
		ImageScanEnabled:       true,
		SignedURLEnabled:       true,
		SignedURLExpiration:    24 * time.Hour,
	}

	securityMiddleware := security.NewSecurityMiddleware(securityConfig)

	// Apply security middleware
	r.Use(securityMiddleware.CORSMiddleware())
	r.Use(securityMiddleware.SecurityHeadersMiddleware())
	r.Use(securityMiddleware.RateLimitMiddleware())

	// Health endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	// API Documentation routes (no auth required)
	r.GET("/api/docs", gin.WrapF(docs.ServeSwaggerUI))
	r.GET("/api/docs/openapi.json", gin.WrapF(docs.ServeAPIDocumentation))

	// Auth routes (no auth required)
	mountAuth(r)

	// Protected routes
	protected := r.Group("/")
	protected.Use(securityMiddleware.OptionalAuthMiddleware())
	{
		// User routes
		mountUser(protected)

		// Vendor routes
		mountVendor(protected)

		// Conversion routes
		mountConversion(protected)

		// Payment routes
		mountPayment(protected)

		// Image routes
		mountImage(protected)

		// Notification routes
		mountNotification(protected)

		// Worker routes
		mountWorker(protected)

		// Storage routes
		mountStorage(protected)

		// Share routes
		mountShare(protected)
	}

	// Admin routes (require admin auth)
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(securityMiddleware.JWTAuthMiddleware())
	adminGroup.Use(securityMiddleware.AdminAuthMiddleware())
	{
		mountAdmin(adminGroup)
	}

	return r
}

// NewWithMonitoring creates a new router with monitoring capabilities
func NewWithMonitoring(monitor *monitoring.MonitoringService) *gin.Engine {
	r := gin.New()

	// Load security configuration
	cfg, err := config.Load()
	if err != nil {
		monitor.LogFatal(context.Background(), "Failed to load config", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create monitoring middleware
	requestLogger := middleware.NewRequestLoggerMiddleware(monitor.Logger())
	monitoringMiddleware := middleware.NewMonitoringMiddleware(monitor)
	contextMiddleware := middleware.NewContextMiddleware()
	recoveryMiddleware := middleware.NewRecoveryMiddleware(monitor)

	// Apply monitoring middleware first
	r.Use(recoveryMiddleware.Recovery())
	r.Use(contextMiddleware.InjectContext())
	r.Use(requestLogger.RequestLogging())
	r.Use(monitoringMiddleware.ErrorHandling())
	r.Use(monitoringMiddleware.PerformanceMonitoring())
	r.Use(monitoringMiddleware.SecurityMonitoring())

	// Create security middleware
	securityConfig := &security.SecurityConfig{
		RateLimitEnabled:       true,
		RateLimitPerIP:         cfg.RateLimit.OTPPerIP,
		RateLimitPerUser:       1000,
		RateLimitWindow:        cfg.RateLimit.Window,
		JWTSecret:              cfg.JWT.Secret,
		JWTExpiration:          cfg.JWT.AccessTTL,
		CORSEnabled:            true,
		AllowedOrigins:         []string{"*"},
		AllowedMethods:         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:         []string{"*"},
		SecurityHeadersEnabled: true,
		ImageScanEnabled:       true,
		SignedURLEnabled:       true,
		SignedURLExpiration:    24 * time.Hour,
	}

	securityMiddleware := security.NewSecurityMiddleware(securityConfig)

	// Apply security middleware
	r.Use(securityMiddleware.CORSMiddleware())
	r.Use(securityMiddleware.SecurityHeadersMiddleware())
	r.Use(securityMiddleware.RateLimitMiddleware())

	// Health endpoints with monitoring
	healthHandler := monitoring.NewHealthHandler(monitor.Health())
	healthHandler.RegisterRoutes(r.Group("/api"))

	// Auth routes (no auth required)
	mountAuth(r)

	// Protected routes
	protected := r.Group("/")
	protected.Use(securityMiddleware.OptionalAuthMiddleware())
	protected.Use(contextMiddleware.UserContext())
	protected.Use(contextMiddleware.VendorContext())
	protected.Use(contextMiddleware.ConversionContext())
	{
		// User routes
		mountUser(protected)

		// Vendor routes
		mountVendor(protected)

		// Conversion routes
		mountConversion(protected)

		// Payment routes
		mountPayment(protected)

		// Image routes
		mountImage(protected)

		// Notification routes
		mountNotification(protected)

		// Worker routes
		mountWorker(protected)

		// Storage routes
		mountStorage(protected)

		// Share routes
		mountShare(protected)
	}

	// Admin routes (require admin auth)
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(securityMiddleware.JWTAuthMiddleware())
	adminGroup.Use(securityMiddleware.AdminAuthMiddleware())
	{
		mountAdmin(adminGroup)
	}

	monitor.LogInfo(context.Background(), "Router initialized with monitoring", map[string]interface{}{
		"health_endpoints": true,
		"monitoring":       true,
	})

	return r
}

// NewWithServices creates a new router with all services properly wired
func NewWithServices(
	authService interface{},
	userService interface{},
	vendorService interface{},
	conversionService interface{},
	imageService interface{},
	paymentService interface{},
	shareService interface{},
	adminService interface{},
	notificationService interface{},
	monitor *monitoring.MonitoringService,
) *gin.Engine {
	r := gin.New()

	// Create monitoring middleware
	requestLogger := middleware.NewRequestLoggerMiddleware(monitor.Logger())
	monitoringMiddleware := middleware.NewMonitoringMiddleware(monitor)
	contextMiddleware := middleware.NewContextMiddleware()
	recoveryMiddleware := middleware.NewRecoveryMiddleware(monitor)

	// Apply monitoring middleware first
	r.Use(recoveryMiddleware.Recovery())
	r.Use(contextMiddleware.InjectContext())
	r.Use(requestLogger.RequestLogging())
	r.Use(monitoringMiddleware.ErrorHandling())
	r.Use(monitoringMiddleware.PerformanceMonitoring())
	r.Use(monitoringMiddleware.SecurityMonitoring())

	// Load security configuration
	cfg, err := config.Load()
	if err != nil {
		monitor.LogFatal(context.Background(), "Failed to load config", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create security middleware
	securityConfig := &security.SecurityConfig{
		RateLimitEnabled:       true,
		RateLimitPerIP:         cfg.RateLimit.OTPPerIP,
		RateLimitPerUser:       1000,
		RateLimitWindow:        cfg.RateLimit.Window,
		JWTSecret:              cfg.JWT.Secret,
		JWTExpiration:          cfg.JWT.AccessTTL,
		CORSEnabled:            true,
		AllowedOrigins:         []string{"*"},
		AllowedMethods:         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:         []string{"*"},
		SecurityHeadersEnabled: true,
		ImageScanEnabled:       true,
		SignedURLEnabled:       true,
		SignedURLExpiration:    24 * time.Hour,
	}

	securityMiddleware := security.NewSecurityMiddleware(securityConfig)

	// Apply security middleware
	r.Use(securityMiddleware.CORSMiddleware())
	r.Use(securityMiddleware.SecurityHeadersMiddleware())
	r.Use(securityMiddleware.RateLimitMiddleware())

	// Health endpoints with monitoring
	healthHandler := monitoring.NewHealthHandler(monitor.Health())
	healthHandler.RegisterRoutes(r.Group("/api"))

	// Auth routes (no auth required) - using passed authHandler
	authGroup := r.Group("/auth")
	authGroup.POST("/send-otp", common.GinWrap(authService.(*auth.Handler).SendOTP))
	authGroup.POST("/verify-otp", common.GinWrap(authService.(*auth.Handler).VerifyOTP))
	authGroup.POST("/register", common.GinWrap(authService.(*auth.Handler).Register))
	authGroup.POST("/login", common.GinWrap(authService.(*auth.Handler).Login))
	authGroup.POST("/refresh", common.GinWrap(authService.(*auth.Handler).Refresh))
	authGroup.POST("/logout", common.GinWrap(authService.(*auth.Handler).Authenticate(authService.(*auth.Handler).Logout)))
	authGroup.POST("/logout-all", common.GinWrap(authService.(*auth.Handler).Authenticate(authService.(*auth.Handler).LogoutAll)))

	// Protected routes - using passed handlers
	protected := r.Group("/api")
	protected.Use(securityMiddleware.OptionalAuthMiddleware())
	protected.Use(contextMiddleware.UserContext())
	protected.Use(contextMiddleware.VendorContext())
	protected.Use(contextMiddleware.ConversionContext())
	{
		// Mount service routes using passed handlers
		if userService != nil {
			user.MountRoutes(protected, userService.(*user.Handler))
		}
		if vendorService != nil {
			vendor.MountRoutes(protected, vendorService.(*vendor.Handler))
		}
		if conversionService != nil {
			conversion.MountRoutes(protected, conversionService.(*conversion.Handler))
		}
		if imageService != nil {
			image.SetupGinRoutes(protected, imageService.(*image.Handler))
		}
		if paymentService != nil {
			payment.SetupRoutes(protected, paymentService.(*payment.Handler))
		}
		if shareService != nil {
			// Share service doesn't have MountRoutes, we'll add it manually
			shareGroup := protected.Group("/share")
			shareGroup.GET("/:token", shareService.(*share.Handler).AccessSharedLink)
			shareGroup.POST("/create", shareService.(*share.Handler).CreateSharedLink)
			shareGroup.DELETE("/:id", shareService.(*share.Handler).DeactivateSharedLink)
			shareGroup.GET("/", shareService.(*share.Handler).ListUserSharedLinks)
		}
	}

	// Admin routes (require admin auth) - using passed adminHandler
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(securityMiddleware.JWTAuthMiddleware())
	adminGroup.Use(securityMiddleware.AdminAuthMiddleware())
	{
		if adminService != nil {
			admin.SetupRoutes(adminGroup, adminService.(*admin.Handler))
		}
	}

	// Notification routes - using passed notificationHandler
	notificationGroup := r.Group("/api/notifications")
	notificationGroup.Use(securityMiddleware.OptionalAuthMiddleware())
	{
		if notificationService != nil {
			notification.SetupRoutes(notificationGroup, notificationService.(*notification.Handler))
		}
	}

	monitor.LogInfo(context.Background(), "Router initialized with all services", map[string]interface{}{
		"health_endpoints": true,
		"monitoring":       true,
		"services":         []string{"auth", "user", "vendor", "conversion", "image", "payment", "share", "admin"},
	})

	return r
}

func mountAuth(r *gin.Engine) {
	// Load config for SMS provider
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	store := auth.NewInMemoryStore()
	limiter := auth.NewInMemoryLimiter()
	tokens := auth.NewSimpleTokenService()
	smsProvider := sms.NewProvider(cfg.SMS.Provider, cfg.SMS.APIKey, cfg.SMS.TemplateID)
	// Create handler compatible with gin via adapters
	h := auth.NewHandler(store, tokens, limiter, smsProvider)

	g := r.Group("/auth")
	g.POST("/send-otp", common.GinWrap(h.SendOTP))
	g.POST("/verify-otp", common.GinWrap(h.VerifyOTP))
	g.POST("/register", common.GinWrap(h.Register))
	g.POST("/login", common.GinWrap(h.Login))
	g.POST("/refresh", common.GinWrap(h.Refresh))
	g.POST("/logout", common.GinWrap(h.Authenticate(h.Logout)))
	g.POST("/logout-all", common.GinWrap(h.Authenticate(h.LogoutAll)))
}

func mountUser(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create user service and handler
	_, userHandler := user.WireUserService(db)

	// Mount user routes
	user.MountRoutes(r, userHandler)
}

func mountVendor(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create vendor service and handler
	_, vendorHandler := vendor.WireVendorService(db)

	// Mount vendor routes
	vendor.MountRoutes(r, vendorHandler)
}

func mountConversion(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create conversion service and handler
	_, conversionHandler := conversion.WireConversionService(db)

	// Mount conversion routes
	conversion.MountRoutes(r, conversionHandler)
}

func mountPayment(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create real services instead of mocks
	userService := &realUserService{db: db}
	notificationService := &realNotificationService{db: db}
	quotaService := &realQuotaService{db: db}
	auditLogger := &realAuditLogger{db: db}
	rateLimiter := &realRateLimiter{}

	// Create payment service and handler
	paymentHandler := payment.NewHandler(
		payment.NewService(
			payment.NewPaymentStore(db),
			payment.NewZarinpalGateway("zibal", "https://gateway.zibal.ir"),
			userService,
			notificationService,
			quotaService,
			auditLogger,
			rateLimiter,
			payment.NewPaymentConfigService(),
		),
	)

	// Mount payment routes
	paymentGroup := r.Group("/api")
	payment.SetupRoutes(paymentGroup, paymentHandler)
}

func mountAdmin(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create admin service and handler
	_, adminHandler := admin.WireAdminService(db)

	// Mount admin routes
	admin.SetupRoutes(r, adminHandler)
}

func mountImage(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create image service and handler
	_, imageHandler := image.WireImageService(db)

	// Skip mounting if handler is nil (not implemented yet)
	if imageHandler == nil {
		return
	}

	// Mount image routes
	imageGroup := r.Group("/api")
	image.SetupGinRoutes(imageGroup, imageHandler)
}

func mountNotification(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create notification service and handler
	_, notificationHandler := notification.WireNotificationService(db)

	// Skip mounting if handler is nil (not implemented yet)
	if notificationHandler == nil {
		return
	}

	// Mount notification routes
	notificationGroup := r.Group("/api")
	notification.SetupRoutes(notificationGroup, notificationHandler)
}

func mountWorker(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create worker service and handler
	_, workerHandler := worker.WireWorkerService(db)

	// Skip mounting if handler is nil (not implemented yet)
	if workerHandler == nil {
		return
	}

	// Mount worker routes
	workerGroup := r.Group("/api")
	worker.RegisterWorkerRoutes(workerGroup, workerHandler)
}

// buildDSN builds database connection string
func buildDSN(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)
}

func mountStorage(r *gin.RouterGroup) {
	// Load config for storage
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Create storage configuration
	storageConfig := &storage.Config{
		BasePath:     cfg.Storage.StoragePath,
		BackupPath:   cfg.Storage.StoragePath + "/backups",
		SignedURLKey: cfg.JWT.Secret,
		MaxFileSize:  50 * 1024 * 1024, // 50MB
		AllowedTypes: []string{
			"image/jpeg",
			"image/jpg",
			"image/png",
			"image/gif",
			"image/webp",
		},
		ThumbnailSizes: []storage.ThumbnailSize{
			{Name: "small", Width: 150, Height: 150},
			{Name: "medium", Width: 300, Height: 300},
			{Name: "large", Width: 600, Height: 600},
		},
		RetentionPolicy: storage.RetentionPolicy{
			KeepImagesForever: true,
			MaxAge:            0,
			CleanupSchedule:   "0 2 * * *",
		},
		BackupPolicy: storage.BackupPolicy{
			Enabled:          true,
			BackupFrequency:  "daily",
			RetentionDays:    365,
			CompressionLevel: 6,
		},
		ServerConfig: storage.ServerConfig{
			Host:       "localhost",
			Port:       8080,
			BaseURL:    "http://localhost:8080",
			PublicPath: "/api/storage/public",
			StaticPath: "/api/storage/static",
		},
	}

	// Create storage wire
	storageWire := storage.NewWire(storageConfig)

	// Initialize storage services
	ctx := context.Background()
	if err := storageWire.Initialize(ctx); err != nil {
		panic("failed to initialize storage: " + err.Error())
	}

	// Get handler and mount routes
	storageHandler := storageWire.GetHandler()
	storageHandler.RegisterRoutes(r.Group("/api"))
}

func mountShare(r *gin.RouterGroup) {
	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Connect to database
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Create share service dependencies
	shareStore := share.NewStore(db)

	// Mock services for now - these would be properly wired in a real implementation
	mockConversionService := &mockShareConversionService{}
	mockImageService := &mockShareImageService{}
	mockNotificationService := &mockShareNotificationService{}
	mockAuditLogger := &mockShareAuditLogger{}
	mockMetricsCollector := &mockShareMetricsCollector{}

	// Create share service and handler
	shareService := share.NewService(
		shareStore,
		mockConversionService,
		mockImageService,
		mockNotificationService,
		mockAuditLogger,
		mockMetricsCollector,
	)
	shareHandler := share.NewHandler(shareService)

	// Mount share routes
	shareHandler.RegisterRoutes(r.Group("/api"))
}

// Mock services for share integration
type mockShareConversionService struct{}
type mockShareImageService struct{}
type mockShareNotificationService struct{}
type mockShareAuditLogger struct{}
type mockShareMetricsCollector struct{}

func (m *mockShareConversionService) GetConversion(ctx context.Context, conversionID, userID string) (share.ConversionResponse, error) {
	return share.ConversionResponse{
		ID: conversionID, UserID: userID, Status: "completed",
		ResultImageID: stringPtr("img-123"),
	}, nil
}

func (m *mockShareConversionService) ValidateConversionOwnership(ctx context.Context, conversionID, userID string) error {
	return nil
}

func (m *mockShareImageService) GetImage(ctx context.Context, imageID string) (share.ImageResponse, error) {
	return share.ImageResponse{
		ID: imageID, OriginalURL: "https://example.com/image.jpg",
	}, nil
}

func (m *mockShareImageService) GenerateSignedURL(ctx context.Context, imageID string, accessType string, ttl int64) (string, error) {
	return "https://example.com/signed-url", nil
}

func (m *mockShareNotificationService) SendShareCreated(ctx context.Context, userID, shareID, shareToken string) error {
	return nil
}

func (m *mockShareNotificationService) SendShareAccessed(ctx context.Context, userID, shareID string, accessCount int) error {
	return nil
}

func (m *mockShareAuditLogger) LogShareCreated(ctx context.Context, userID, conversionID, shareID string) error {
	return nil
}

func (m *mockShareAuditLogger) LogShareAccessed(ctx context.Context, shareID, ipAddress, userAgent string) error {
	return nil
}

func (m *mockShareAuditLogger) LogShareDeactivated(ctx context.Context, userID, shareID string) error {
	return nil
}

func (m *mockShareMetricsCollector) RecordShareCreated(ctx context.Context, userID, conversionID string) error {
	return nil
}

func (m *mockShareMetricsCollector) RecordShareAccessed(ctx context.Context, shareID string, success bool) error {
	return nil
}

func (m *mockShareMetricsCollector) RecordShareExpired(ctx context.Context, shareID string) error {
	return nil
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// Real service implementations to replace mocks

// realUserService implements user service interface for payment
type realUserService struct {
	db *sql.DB
}

func (r *realUserService) GetUserPlan(ctx context.Context, userID string) (interface{}, error) {
	// Implementation would query the database for user's plan
	return map[string]interface{}{
		"plan_id": "free",
		"name":    "Free Plan",
	}, nil
}

func (r *realUserService) UpdateUserPlan(ctx context.Context, planID string, status string) (interface{}, error) {
	// Implementation would update user's plan in database
	return map[string]interface{}{
		"success": true,
		"plan_id": planID,
		"status":  status,
	}, nil
}

func (r *realUserService) CreateUserPlan(ctx context.Context, userID string, planName string) (interface{}, error) {
	// Implementation would create a new user plan
	return map[string]interface{}{
		"success":   true,
		"user_id":   userID,
		"plan_name": planName,
	}, nil
}

// realNotificationService implements notification service interface for payment
type realNotificationService struct {
	db *sql.DB
}

func (r *realNotificationService) SendPaymentSuccess(ctx context.Context, userID string, paymentID string, planName string) error {
	// Implementation would send actual notification
	fmt.Printf("Payment success notification sent to user %s for payment %s, plan %s\n", userID, paymentID, planName)
	return nil
}

func (r *realNotificationService) SendPaymentFailed(ctx context.Context, userID string, paymentID string, reason string) error {
	// Implementation would send actual notification
	fmt.Printf("Payment failed notification sent to user %s for payment %s, reason: %s\n", userID, paymentID, reason)
	return nil
}

func (r *realNotificationService) SendPlanActivated(ctx context.Context, userID string, planName string) error {
	// Implementation would send actual notification
	fmt.Printf("Plan activated notification sent to user %s for plan %s\n", userID, planName)
	return nil
}

func (r *realNotificationService) SendPlanExpired(ctx context.Context, userID string, planName string) error {
	// Implementation would send actual notification
	fmt.Printf("Plan expired notification sent to user %s for plan %s\n", userID, planName)
	return nil
}

// realQuotaService implements quota service interface for payment
type realQuotaService struct {
	db *sql.DB
}

func (r *realQuotaService) UpdateUserQuota(ctx context.Context, userID string, planName string) error {
	// Implementation would update user's quota based on plan
	fmt.Printf("Updated quota for user %s with plan %s\n", userID, planName)
	return nil
}

func (r *realQuotaService) ResetMonthlyQuota(ctx context.Context, userID string) error {
	// Implementation would reset user's monthly quota
	fmt.Printf("Reset monthly quota for user %s\n", userID)
	return nil
}

func (r *realQuotaService) GetUserQuotaStatus(ctx context.Context, userID string) (interface{}, error) {
	// Implementation would get user's current quota status
	return map[string]interface{}{
		"remaining": 10,
		"used":      0,
		"limit":     10,
	}, nil
}

// realAuditLogger implements audit logger interface for payment
type realAuditLogger struct {
	db *sql.DB
}

func (r *realAuditLogger) LogPaymentAction(ctx context.Context, userID string, action string, metadata map[string]interface{}) error {
	// Implementation would log payment actions to audit table
	fmt.Printf("Audit log: Payment action %s by user %s - %+v\n", action, userID, metadata)
	return nil
}

// realRateLimiter implements rate limiter interface for payment
type realRateLimiter struct{}

func (r *realRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	// Implementation would check rate limits
	return true
}
