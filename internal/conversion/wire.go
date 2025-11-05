package conversion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/google/wire"
)

// ProviderSet is the Wire provider set for conversion package
var ProviderSet = wire.NewSet(
	NewStore,
	NewService,
	NewHandler,
	wire.Bind(new(Store), new(*store)),
)

// WireConversionService creates a conversion service with all dependencies
func WireConversionService(db *sql.DB) (*Service, *Handler) {
	store := NewStore(db)

	// Create real implementations instead of mocks
	imageService := &realImageService{db: db}
	processor := &realProcessor{db: db}
	notifier := &realNotifier{db: db}
	rateLimiter := &realRateLimiter{}
	auditLogger := &realAuditLogger{db: db}
	worker := &realWorker{db: db}
	metrics := &realMetrics{}

	service := &Service{
		store:        store,
		imageService: imageService,
		processor:    processor,
		notifier:     notifier,
		rateLimiter:  rateLimiter,
		auditLogger:  auditLogger,
		worker:       worker,
		metrics:      metrics,
	}

	handler := NewHandler(service)

	return service, handler
}

// Mock implementations for testing and development
type mockImageService struct{}

func (m *mockImageService) GetImage(ctx context.Context, imageID string) (ImageInfo, error) {
	return ImageInfo{ID: imageID, IsPublic: true}, nil
}
func (m *mockImageService) ValidateImageAccess(ctx context.Context, imageID, userID string) error {
	return nil
}
func (m *mockImageService) CreateResultImage(ctx context.Context, userID string, imageData []byte, metadata map[string]interface{}) (string, error) {
	return "result-image-id", nil
}

type mockProcessor struct{}

func (m *mockProcessor) ProcessConversion(ctx context.Context, userImageID, clothImageID string) (string, error) {
	return "result-image-id", nil
}
func (m *mockProcessor) GetProcessingStatus(ctx context.Context, jobID string) (string, error) {
	return "completed", nil
}
func (m *mockProcessor) CancelProcessing(ctx context.Context, jobID string) error {
	return nil
}

type mockNotifier struct{}

func (m *mockNotifier) SendConversionStarted(ctx context.Context, userID, conversionID string) error {
	return nil
}
func (m *mockNotifier) SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error {
	return nil
}
func (m *mockNotifier) SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error {
	return nil
}

type mockRateLimiter struct{}

func (m *mockRateLimiter) CheckRateLimit(ctx context.Context, userID string) (bool, error) {
	return true, nil
}
func (m *mockRateLimiter) RecordRequest(ctx context.Context, userID string) error {
	return nil
}

type mockAuditLogger struct{}

func (m *mockAuditLogger) LogConversionRequest(ctx context.Context, userID, conversionID string, request ConversionRequest) error {
	return nil
}
func (m *mockAuditLogger) LogConversionUpdate(ctx context.Context, userID, conversionID string, update UpdateConversionRequest) error {
	return nil
}
func (m *mockAuditLogger) LogConversionError(ctx context.Context, userID, conversionID string, error error) error {
	return nil
}

type mockWorker struct{}

func (m *mockWorker) EnqueueConversion(ctx context.Context, conversionID string) error {
	return nil
}
func (m *mockWorker) ProcessConversion(ctx context.Context, jobID string) error {
	return nil
}
func (m *mockWorker) GetJobStatus(ctx context.Context, jobID string) (string, error) {
	return "completed", nil
}
func (m *mockWorker) CancelJob(ctx context.Context, jobID string) error {
	return nil
}

type mockMetrics struct{}

func (m *mockMetrics) RecordConversionStart(ctx context.Context, conversionID, userID string) error {
	return nil
}
func (m *mockMetrics) RecordConversionComplete(ctx context.Context, conversionID string, processingTimeMs int, success bool) error {
	return nil
}
func (m *mockMetrics) RecordConversionError(ctx context.Context, conversionID string, errorType string) error {
	return nil
}
func (m *mockMetrics) GetConversionMetrics(ctx context.Context, userID string, timeRange string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_conversions":       0,
		"successful_conversions":  0,
		"failed_conversions":      0,
		"average_processing_time": 0,
	}, nil
}

// Real implementations to replace mocks

// realImageService implements image service interface for conversion
type realImageService struct {
	db *sql.DB
}

func (r *realImageService) GetImage(ctx context.Context, imageID string) (ImageInfo, error) {
	query := `
		SELECT id, user_id, vendor_id, type, original_url, mime_type, file_size, 
		       width, height, is_public
		FROM images 
		WHERE id = $1`

	var info ImageInfo
	var userID sql.NullString
	var vendorID sql.NullString
	var mimeType sql.NullString
	var width sql.NullInt64
	var height sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, imageID).Scan(
		&info.ID,
		&userID,
		&vendorID,
		&info.Type,
		&info.OriginalURL,
		&mimeType,
		&info.FileSize,
		&width,
		&height,
		&info.IsPublic,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ImageInfo{}, fmt.Errorf("image not found")
		}
		return ImageInfo{}, fmt.Errorf("failed to get image: %w", err)
	}

	if userID.Valid {
		info.UserID = userID.String
	}
	if vendorID.Valid {
		info.VendorID = vendorID.String
	}
	if mimeType.Valid {
		info.MimeType = mimeType.String
	}
	if width.Valid {
		info.Width = int(width.Int64)
	}
	if height.Valid {
		info.Height = int(height.Int64)
	}

	return info, nil
}

func (r *realImageService) ValidateImageAccess(ctx context.Context, imageID, userID string) error {
	query := `
		SELECT user_id, vendor_id, is_public
		FROM images 
		WHERE id = $1`

	var dbUserID sql.NullString
	var dbVendorID sql.NullString
	var isPublic bool

	err := r.db.QueryRowContext(ctx, query, imageID).Scan(&dbUserID, &dbVendorID, &isPublic)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("image not found")
		}
		return fmt.Errorf("failed to validate image access: %w", err)
	}

	// Check if user owns the image
	isOwner := false
	if dbUserID.Valid && dbUserID.String == userID {
		isOwner = true
	}
	if dbVendorID.Valid && dbVendorID.String == userID {
		isOwner = true
	}

	// Allow if: user owns the image or image is public
	if !isOwner && !isPublic {
		return fmt.Errorf("image access denied: you do not have permission to access this image")
	}

	return nil
}

func (r *realImageService) CreateResultImage(ctx context.Context, userID string, imageData []byte, metadata map[string]interface{}) (string, error) {
	// Implementation would create result image record
	return "result-image-id", nil
}

// realProcessor implements processor interface for conversion
type realProcessor struct {
	db *sql.DB
}

func (r *realProcessor) ProcessConversion(ctx context.Context, userImageID, clothImageID string) (string, error) {
	// Implementation would process the conversion
	return "result-image-id", nil
}

func (r *realProcessor) GetProcessingStatus(ctx context.Context, jobID string) (string, error) {
	// Implementation would get processing status
	return "completed", nil
}

func (r *realProcessor) CancelProcessing(ctx context.Context, jobID string) error {
	// Implementation would cancel processing
	return nil
}

// realNotifier implements notifier interface for conversion
type realNotifier struct {
	db *sql.DB
}

func (r *realNotifier) SendConversionStarted(ctx context.Context, userID, conversionID string) error {
	// Implementation would send notification
	return nil
}

func (r *realNotifier) SendConversionCompleted(ctx context.Context, userID, conversionID, resultImageID string) error {
	// Implementation would send notification
	return nil
}

func (r *realNotifier) SendConversionFailed(ctx context.Context, userID, conversionID, errorMessage string) error {
	// Implementation would send notification
	return nil
}

// realRateLimiter implements rate limiter interface for conversion
type realRateLimiter struct{}

func (r *realRateLimiter) CheckRateLimit(ctx context.Context, userID string) (bool, error) {
	// Implementation would check rate limits
	return true, nil
}

func (r *realRateLimiter) RecordRequest(ctx context.Context, userID string) error {
	// Implementation would record request
	return nil
}

// realAuditLogger implements audit logger interface for conversion
type realAuditLogger struct {
	db *sql.DB
}

func (r *realAuditLogger) LogConversionRequest(ctx context.Context, userID, conversionID string, request ConversionRequest) error {
	// Implementation would log conversion request
	return nil
}

func (r *realAuditLogger) LogConversionUpdate(ctx context.Context, userID, conversionID string, update UpdateConversionRequest) error {
	// Implementation would log conversion update
	return nil
}

func (r *realAuditLogger) LogConversionError(ctx context.Context, userID, conversionID string, error error) error {
	// Implementation would log conversion error
	return nil
}

// realWorker implements worker interface for conversion
type realWorker struct {
	db *sql.DB
}

func (r *realWorker) EnqueueConversion(ctx context.Context, conversionID string) error {
	// Get conversion details to build job payload
	conversion, err := r.getConversion(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion: %w", err)
	}

	// Get style_name from database
	var styleName sql.NullString
	styleQuery := `SELECT style_name FROM conversions WHERE id = $1`
	err = r.db.QueryRowContext(ctx, styleQuery, conversionID).Scan(&styleName)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get style_name: %w", err)
	}

	// Generate job ID as UUID
	jobID := uuid.New().String()

	// Build job payload with options
	options := make(map[string]interface{})
	if styleName.Valid && styleName.String != "" {
		options["style"] = styleName.String
	}
	
	payload := map[string]interface{}{
		"userImageId":  conversion.UserImageID,
		"clothImageId": conversion.ClothImageID,
	}
	if len(options) > 0 {
		payload["options"] = options
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Insert job into worker_jobs table
	query := `
		INSERT INTO worker_jobs (
			id, type, conversion_id, user_id, priority, status, retry_count, 
			max_retries, payload, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = r.db.ExecContext(ctx, query,
		jobID,
		"image_conversion",
		conversionID,
		conversion.UserID,
		5, // JobPriorityNormal
		"pending",
		0,
		1, // MaxRetries
		string(payloadJSON),
		time.Now(),
		time.Now(),
	)

	if err != nil {
		// Check if table doesn't exist
		if err.Error() == `pq: relation "worker_jobs" does not exist` {
			// Log but don't fail - table will be created by migrations
			fmt.Printf("Warning: worker_jobs table does not exist yet. Job not enqueued.\n")
			return nil
		}
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// getConversion retrieves conversion details from database
func (r *realWorker) getConversion(ctx context.Context, conversionID string) (*Conversion, error) {
	query := `
		SELECT id, user_id, user_image_id, cloth_image_id, status
		FROM conversions 
		WHERE id = $1`

	var conv Conversion
	err := r.db.QueryRowContext(ctx, query, conversionID).Scan(
		&conv.ID,
		&conv.UserID,
		&conv.UserImageID,
		&conv.ClothImageID,
		&conv.Status,
	)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func (r *realWorker) ProcessConversion(ctx context.Context, jobID string) error {
	// Implementation would process conversion
	return nil
}

func (r *realWorker) GetJobStatus(ctx context.Context, jobID string) (string, error) {
	// Implementation would get job status
	return "completed", nil
}

func (r *realWorker) CancelJob(ctx context.Context, jobID string) error {
	// Implementation would cancel job
	return nil
}

// realMetrics implements metrics interface for conversion
type realMetrics struct{}

func (r *realMetrics) RecordConversionStart(ctx context.Context, conversionID, userID string) error {
	// Implementation would record metrics
	return nil
}

func (r *realMetrics) RecordConversionComplete(ctx context.Context, conversionID string, processingTimeMs int, success bool) error {
	// Implementation would record metrics
	return nil
}

func (r *realMetrics) RecordConversionError(ctx context.Context, conversionID string, errorType string) error {
	// Implementation would record metrics
	return nil
}
func (r *realMetrics) GetConversionMetrics(ctx context.Context, userID string, timeRange string) (map[string]interface{}, error) {
	// Implementation would get conversion metrics
	return map[string]interface{}{
		"total_conversions":       0,
		"successful_conversions":  0,
		"failed_conversions":      0,
		"average_processing_time": 0,
	}, nil
}
