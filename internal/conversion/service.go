package conversion

import (
	"context"
	"fmt"
	"time"
)

// Service provides conversion management functionality
type Service struct {
	store        Store
	imageService ImageService
	processor    ConversionProcessor
	notifier     NotificationService
	rateLimiter  RateLimiter
	auditLogger  AuditLogger
	worker       WorkerService
	metrics      MetricsCollector
}

// NewService creates a new conversion service
func NewService(
	store Store,
	imageService ImageService,
	processor ConversionProcessor,
	notifier NotificationService,
	rateLimiter RateLimiter,
	auditLogger AuditLogger,
	worker WorkerService,
	metrics MetricsCollector,
) *Service {
	return &Service{
		store:        store,
		imageService: imageService,
		processor:    processor,
		notifier:     notifier,
		rateLimiter:  rateLimiter,
		auditLogger:  auditLogger,
		worker:       worker,
		metrics:      metrics,
	}
}

// CreateConversion creates a new conversion request
func (s *Service) CreateConversion(ctx context.Context, userID string, req ConversionRequest) (ConversionResponse, error) {
	// Check rate limit
	allowed, err := s.rateLimiter.CheckRateLimit(ctx, userID)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		return ConversionResponse{}, fmt.Errorf("rate limit exceeded")
	}

	// Validate that user_image_id and cloth_image_id are different
	userImageID := req.GetUserImageID()
	clothImageID := req.GetClothImageID()
	
	if userImageID == clothImageID {
		return ConversionResponse{}, fmt.Errorf("user image and cloth image must be different")
	}

	// Validate image access
	if err := s.imageService.ValidateImageAccess(ctx, userImageID, userID); err != nil {
		return ConversionResponse{}, fmt.Errorf("invalid user image access: %w", err)
	}

	// Validate cloth image exists and is accessible
	// Cloth image can be:
	// 1. Public image (is_public = true)
	// 2. Vendor image (type = 'vendor')
	// 3. User's own image (belongs to the same user)
	clothImage, err := s.imageService.GetImage(ctx, clothImageID)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("invalid cloth image: %w", err)
	}
	
	// Check if cloth image belongs to the user (allow using own images)
	isOwnImage := (clothImage.UserID != "" && clothImage.UserID == userID) ||
	              (clothImage.VendorID != "" && clothImage.VendorID == userID)
	
	// Allow if: own image, public, or vendor type
	// Note: SQL function will also validate this, but we check early for better error messages
	if !isOwnImage && !clothImage.IsPublic && clothImage.Type != "vendor" {
		return ConversionResponse{}, fmt.Errorf("cloth image is not accessible: must be public, vendor image, or your own image")
	}

	// Check user quota and create conversion (handled by database function)
	quota, err := s.store.CheckUserQuota(ctx, userID)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("failed to check quota: %w", err)
	}
	if !quota.CanConvert {
		return ConversionResponse{}, fmt.Errorf("quota exceeded: free=%d, paid=%d", quota.RemainingFree, quota.RemainingPaid)
	}

	// Create conversion (this will also update quota counters)
	styleName := req.GetStyleName()
	conversionID, err := s.store.CreateConversion(ctx, userID, userImageID, clothImageID, styleName)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("failed to create conversion: %w", err)
	}

	// Record request
	if err := s.rateLimiter.RecordRequest(ctx, userID); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to record request: %v\n", err)
	}

	// Log audit
	if err := s.auditLogger.LogConversionRequest(ctx, userID, conversionID, req); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to log audit: %v\n", err)
	}

	// Record metrics
	if err := s.metrics.RecordConversionStart(ctx, conversionID, userID); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to record metrics: %v\n", err)
	}

	// Enqueue job for processing
	if err := s.worker.EnqueueConversion(ctx, conversionID); err != nil {
		// Log but don't fail the request - conversion is created
		fmt.Printf("Failed to enqueue conversion: %v\n", err)
	}

	// Send notification
	if err := s.notifier.SendConversionStarted(ctx, userID, conversionID); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to send notification: %v\n", err)
	}

	// Get the created conversion
	conversion, err := s.store.GetConversionWithDetails(ctx, conversionID)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("failed to get created conversion: %w", err)
	}

	return conversion, nil
}

// GetConversion retrieves a conversion by ID
func (s *Service) GetConversion(ctx context.Context, conversionID, userID string) (ConversionResponse, error) {
	conversion, err := s.store.GetConversionWithDetails(ctx, conversionID)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	// Check if user owns this conversion
	if conversion.UserID != userID {
		return ConversionResponse{}, fmt.Errorf("conversion not found")
	}

	return conversion, nil
}

// ListConversions lists user's conversions
func (s *Service) ListConversions(ctx context.Context, userID string, req ConversionListRequest) (ConversionListResponse, error) {
	req.UserID = userID // Ensure user can only see their own conversions

	conversions, err := s.store.ListConversions(ctx, req)
	if err != nil {
		return ConversionListResponse{}, fmt.Errorf("failed to list conversions: %w", err)
	}

	return conversions, nil
}

// UpdateConversion updates a conversion (typically for status updates)
func (s *Service) UpdateConversion(ctx context.Context, conversionID, userID string, req UpdateConversionRequest) error {
	// Get conversion to verify ownership
	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion: %w", err)
	}

	if conversion.UserID != userID {
		return fmt.Errorf("conversion not found")
	}

	// Update conversion
	if err := s.store.UpdateConversion(ctx, conversionID, req); err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}

	// Log audit
	if err := s.auditLogger.LogConversionUpdate(ctx, userID, conversionID, req); err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to log audit: %v\n", err)
	}

	return nil
}

// DeleteConversion deletes a conversion
func (s *Service) DeleteConversion(ctx context.Context, conversionID, userID string) error {
	// Get conversion to verify ownership
	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion: %w", err)
	}

	if conversion.UserID != userID {
		return fmt.Errorf("conversion not found")
	}

	// Only allow deletion of pending or failed conversions
	if conversion.Status != ConversionStatusPending && conversion.Status != ConversionStatusFailed {
		return fmt.Errorf("cannot delete conversion with status: %s", conversion.Status)
	}

	// Delete conversion
	if err := s.store.DeleteConversion(ctx, conversionID); err != nil {
		return fmt.Errorf("failed to delete conversion: %w", err)
	}

	return nil
}

// GetQuotaStatus gets user's quota status
func (s *Service) GetQuotaStatus(ctx context.Context, userID string) (QuotaCheck, error) {
	quota, err := s.store.CheckUserQuota(ctx, userID)
	if err != nil {
		return QuotaCheck{}, fmt.Errorf("failed to get quota status: %w", err)
	}

	return quota, nil
}

// ProcessConversion processes a conversion (called by worker)
func (s *Service) ProcessConversion(ctx context.Context, conversionID string) error {
	// Get conversion
	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion: %w", err)
	}

	// Update status to processing
	if err := s.store.UpdateConversion(ctx, conversionID, UpdateConversionRequest{
		Status: stringPtr(ConversionStatusProcessing),
	}); err != nil {
		return fmt.Errorf("failed to update conversion status: %w", err)
	}

	startTime := time.Now()

	// Process conversion
	resultImageID, err := s.processor.ProcessConversion(ctx, conversion.UserImageID, conversion.ClothImageID)
	if err != nil {
		// Update status to failed
		updateReq := UpdateConversionRequest{
			Status:       stringPtr(ConversionStatusFailed),
			ErrorMessage: stringPtr(err.Error()),
		}
		if updateErr := s.store.UpdateConversion(ctx, conversionID, updateReq); updateErr != nil {
			fmt.Printf("Failed to update conversion status to failed: %v\n", updateErr)
		}

		// Send failure notification
		if notifyErr := s.notifier.SendConversionFailed(ctx, conversion.UserID, conversionID, err.Error()); notifyErr != nil {
			fmt.Printf("Failed to send failure notification: %v\n", notifyErr)
		}

		// Record error metrics
		if metricsErr := s.metrics.RecordConversionError(ctx, conversionID, "processing_failed"); metricsErr != nil {
			fmt.Printf("Failed to record error metrics: %v\n", metricsErr)
		}

		// Log audit
		if auditErr := s.auditLogger.LogConversionError(ctx, conversion.UserID, conversionID, err); auditErr != nil {
			fmt.Printf("Failed to log audit: %v\n", auditErr)
		}

		return fmt.Errorf("conversion processing failed: %w", err)
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	// Update status to completed
	updateReq := UpdateConversionRequest{
		Status:           stringPtr(ConversionStatusCompleted),
		ResultImageID:    stringPtr(resultImageID),
		ProcessingTimeMs: intPtr(processingTime),
	}
	if err := s.store.UpdateConversion(ctx, conversionID, updateReq); err != nil {
		return fmt.Errorf("failed to update conversion status: %w", err)
	}

	// Send success notification
	if err := s.notifier.SendConversionCompleted(ctx, conversion.UserID, conversionID, resultImageID); err != nil {
		fmt.Printf("Failed to send success notification: %v\n", err)
	}

	// Record success metrics
	if err := s.metrics.RecordConversionComplete(ctx, conversionID, processingTime, true); err != nil {
		fmt.Printf("Failed to record success metrics: %v\n", err)
	}

	return nil
}

// GetProcessingStatus gets the processing status of a conversion
func (s *Service) GetProcessingStatus(ctx context.Context, conversionID, userID string) (string, error) {
	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return "", fmt.Errorf("failed to get conversion: %w", err)
	}

	if conversion.UserID != userID {
		return "", fmt.Errorf("conversion not found")
	}

	return conversion.Status, nil
}

// CancelConversion cancels a pending conversion
func (s *Service) CancelConversion(ctx context.Context, conversionID, userID string) error {
	conversion, err := s.store.GetConversion(ctx, conversionID)
	if err != nil {
		return fmt.Errorf("failed to get conversion: %w", err)
	}

	if conversion.UserID != userID {
		return fmt.Errorf("conversion not found")
	}

	if conversion.Status != ConversionStatusPending {
		return fmt.Errorf("cannot cancel conversion with status: %s", conversion.Status)
	}

	// Update status to failed with cancellation message
	updateReq := UpdateConversionRequest{
		Status:       stringPtr(ConversionStatusFailed),
		ErrorMessage: stringPtr("cancelled by user"),
	}
	if err := s.store.UpdateConversion(ctx, conversionID, updateReq); err != nil {
		return fmt.Errorf("failed to cancel conversion: %w", err)
	}

	return nil
}

// GetConversionMetrics gets conversion metrics for a user
func (s *Service) GetConversionMetrics(ctx context.Context, userID, timeRange string) (map[string]interface{}, error) {
	metrics, err := s.metrics.GetConversionMetrics(ctx, userID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion metrics: %w", err)
	}

	return metrics, nil
}

// WatchConversion waits for a conversion to complete and returns the result
// This function polls the conversion status until it's completed or failed
// The ctx parameter should already have a timeout applied by the caller
// pollInterval: interval between status checks (default: 100ms)
func (s *Service) WatchConversion(ctx context.Context, conversionID, userID string, timeout, pollInterval time.Duration) (ConversionResponse, error) {
	// Set defaults for poll interval (faster for better responsiveness)
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}

	// Create ticker for polling - start immediately
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Store the last known conversion status
	var lastKnownConversion ConversionResponse

	// Initial check - but don't return immediately if pending, start polling right away
	conversion, err := s.store.GetConversionWithDetails(ctx, conversionID)
	if err != nil {
		return ConversionResponse{}, fmt.Errorf("failed to get conversion: %w", err)
	}

	// Check ownership
	if conversion.UserID != userID {
		return ConversionResponse{}, fmt.Errorf("conversion not found")
	}

	// If already completed or failed, return immediately
	if conversion.Status == ConversionStatusCompleted || conversion.Status == ConversionStatusFailed {
		return conversion, nil
	}

	// Store last known status
	lastKnownConversion = conversion

	// Poll until completed or timeout
	// We'll check immediately in the loop, no need to wait for first tick
	for {
		// Immediate check (before waiting for ticker) to catch rapid status changes
		current, err := s.store.GetConversionWithDetails(ctx, conversionID)
		if err == nil {
			lastKnownConversion = current
			// If status changed to completed or failed, return immediately
			if current.Status == ConversionStatusCompleted || current.Status == ConversionStatusFailed {
				return current, nil
			}
		}

		select {
		case <-ctx.Done():
			// Timeout or context cancelled - return last known status
			if ctx.Err() == context.DeadlineExceeded {
				// Try to get latest status with a fresh context (with short timeout)
				// This allows us to get the current status even if the main context timed out
				statusCtx, statusCancel := context.WithTimeout(context.Background(), 2*time.Second)
				current, err := s.store.GetConversionWithDetails(statusCtx, conversionID)
				statusCancel()
				
				if err == nil {
					// Successfully retrieved current status
					return current, nil
				}
				// If we can't get the latest status, return the last known status
				if lastKnownConversion.ID != "" {
					return lastKnownConversion, nil
				}
				return ConversionResponse{}, fmt.Errorf("timeout waiting for conversion: %w", err)
			}
			// Context was cancelled for other reasons
			if lastKnownConversion.ID != "" {
				return lastKnownConversion, nil
			}
			return ConversionResponse{}, fmt.Errorf("context cancelled: %w", ctx.Err())

		case <-ticker.C:
			// Check status again (this will be after the initial immediate check)
			current, err := s.store.GetConversionWithDetails(ctx, conversionID)
			if err != nil {
				// If context is done, return last known status
				if ctx.Err() != nil {
					if lastKnownConversion.ID != "" {
						return lastKnownConversion, nil
					}
					return ConversionResponse{}, fmt.Errorf("context error: %w", ctx.Err())
				}
				// If it's a temporary error, continue polling but keep last known status
				continue
			}

			// Update last known conversion
			lastKnownConversion = current

			// If status changed to completed or failed, return immediately
			if current.Status == ConversionStatusCompleted || current.Status == ConversionStatusFailed {
				return current, nil
			}
		}
	}
}
