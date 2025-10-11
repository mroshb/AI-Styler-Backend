package worker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ai-styler/internal/conversion"
	"ai-styler/internal/image"
)

// Service implements the WorkerService interface
type Service struct {
	config           *WorkerConfig
	jobQueue         JobQueue
	imageProcessor   ImageProcessor
	fileStorage      FileStorage
	conversionStore  ConversionStore
	imageStore       ImageStore
	geminiAPI        GeminiAPI
	notifier         NotificationService
	metricsCollector MetricsCollector
	healthChecker    HealthChecker
	retryHandler     RetryHandler

	// Worker state
	workers     map[string]*Worker
	workerMutex sync.RWMutex
	stopChan    chan struct{}
	workerID    string
	started     bool
	startMutex  sync.Mutex
}

// Worker represents a single worker instance
type Worker struct {
	ID            string
	Status        string
	CurrentJob    *WorkerJob
	JobsProcessed int64
	StartedAt     time.Time
	LastSeen      time.Time
}

// NewService creates a new worker service
func NewService(
	config *WorkerConfig,
	jobQueue JobQueue,
	imageProcessor ImageProcessor,
	fileStorage FileStorage,
	conversionStore ConversionStore,
	imageStore ImageStore,
	geminiAPI GeminiAPI,
	notifier NotificationService,
	metricsCollector MetricsCollector,
	healthChecker HealthChecker,
	retryHandler RetryHandler,
) *Service {
	if config == nil {
		config = getDefaultConfig()
	}

	return &Service{
		config:           config,
		jobQueue:         jobQueue,
		imageProcessor:   imageProcessor,
		fileStorage:      fileStorage,
		conversionStore:  conversionStore,
		imageStore:       imageStore,
		geminiAPI:        geminiAPI,
		notifier:         notifier,
		metricsCollector: metricsCollector,
		healthChecker:    healthChecker,
		retryHandler:     retryHandler,
		workers:          make(map[string]*Worker),
		stopChan:         make(chan struct{}),
		workerID:         generateWorkerID(),
	}
}

// Start starts the worker service
func (s *Service) Start(ctx context.Context) error {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()

	if s.started {
		return fmt.Errorf("worker service is already started")
	}

	log.Printf("Starting worker service with ID: %s", s.workerID)

	// Register this worker
	if err := s.healthChecker.RegisterWorker(ctx, s.workerID); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Start worker goroutines
	for i := 0; i < s.config.MaxWorkers; i++ {
		workerID := fmt.Sprintf("%s-%d", s.workerID, i)
		go s.workerLoop(ctx, workerID)
	}

	// Start cleanup goroutine
	go s.cleanupLoop(ctx)

	// Start health check goroutine
	if s.config.EnableHealthCheck {
		go s.healthCheckLoop(ctx)
	}

	s.started = true
	log.Printf("Worker service started with %d workers", s.config.MaxWorkers)

	return nil
}

// Stop stops the worker service
func (s *Service) Stop(ctx context.Context) error {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()

	if !s.started {
		return fmt.Errorf("worker service is not started")
	}

	log.Printf("Stopping worker service: %s", s.workerID)

	// Signal all workers to stop
	close(s.stopChan)

	// Unregister this worker
	if err := s.healthChecker.UnregisterWorker(ctx, s.workerID); err != nil {
		log.Printf("Failed to unregister worker: %v", err)
	}

	s.started = false
	log.Printf("Worker service stopped")

	return nil
}

// EnqueueJob enqueues a new job for processing
func (s *Service) EnqueueJob(ctx context.Context, jobType string, conversionID, userID string, payload JobPayload) error {
	job := &WorkerJob{
		ID:           generateJobID(),
		Type:         jobType,
		ConversionID: conversionID,
		UserID:       userID,
		Priority:     JobPriorityNormal,
		Status:       JobStatusPending,
		RetryCount:   0,
		MaxRetries:   s.config.MaxRetries,
		Payload:      payload,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.jobQueue.EnqueueJob(ctx, job); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Record metrics
	if s.metricsCollector != nil {
		s.metricsCollector.RecordJobStart(ctx, job.ID, job.Type)
	}

	log.Printf("Enqueued job %s of type %s for conversion %s", job.ID, jobType, conversionID)
	return nil
}

// ProcessJob processes a single job
func (s *Service) ProcessJob(ctx context.Context, job *WorkerJob) error {
	startTime := time.Now()

	log.Printf("Processing job %s of type %s", job.ID, job.Type)

	// Update job status to processing
	if err := s.jobQueue.UpdateJobStatus(ctx, job.ID, JobStatusProcessing, s.workerID); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Process based on job type
	var result interface{}
	var err error

	switch job.Type {
	case "image_conversion":
		result, err = s.processImageConversion(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	processingTime := time.Since(startTime)

	if err != nil {
		log.Printf("Job %s failed after %v: %v", job.ID, processingTime, err)

		// Check if we should retry
		if s.retryHandler.ShouldRetry(ctx, job, err) && job.RetryCount < job.MaxRetries {
			// Increment retry count and reschedule
			job.RetryCount++
			job.Status = JobStatusPending
			job.ErrorMessage = err.Error()
			job.UpdatedAt = time.Now()

			// Calculate retry delay
			delay := s.retryHandler.GetRetryDelay(ctx, job)

			// Reschedule job with delay
			go func() {
				time.Sleep(delay)
				if err := s.jobQueue.EnqueueJob(ctx, job); err != nil {
					log.Printf("Failed to reschedule job %s: %v", job.ID, err)
				}
			}()

			log.Printf("Job %s scheduled for retry %d/%d in %v", job.ID, job.RetryCount, job.MaxRetries, delay)
			return nil
		}

		// Mark job as failed
		if err := s.jobQueue.FailJob(ctx, job.ID, err.Error()); err != nil {
			log.Printf("Failed to mark job %s as failed: %v", job.ID, err)
		}

		// Update conversion status
		if err := s.updateConversionStatus(ctx, job.ConversionID, "failed", nil, err.Error(), int(processingTime.Milliseconds())); err != nil {
			log.Printf("Failed to update conversion status: %v", err)
		}

		// Send failure notification
		if s.notifier != nil {
			if err := s.notifier.SendConversionFailed(ctx, job.UserID, job.ConversionID, err.Error()); err != nil {
				log.Printf("Failed to send failure notification: %v", err)
			}
		}

		// Record error metrics
		if s.metricsCollector != nil {
			s.metricsCollector.RecordJobError(ctx, job.ID, "processing_error")
		}

		return err
	}

	// Mark job as completed
	if err := s.jobQueue.CompleteJob(ctx, job.ID, result); err != nil {
		log.Printf("Failed to mark job %s as completed: %v", job.ID, err)
	}

	// Update conversion status
	if err := s.updateConversionStatus(ctx, job.ConversionID, "completed", result, "", int(processingTime.Milliseconds())); err != nil {
		log.Printf("Failed to update conversion status: %v", err)
	}

	// Send success notification
	if s.notifier != nil {
		if resultImageID, ok := result.(string); ok {
			if err := s.notifier.SendConversionCompleted(ctx, job.UserID, job.ConversionID, resultImageID); err != nil {
				log.Printf("Failed to send success notification: %v", err)
			}
		}
	}

	// Record success metrics
	if s.metricsCollector != nil {
		s.metricsCollector.RecordJobComplete(ctx, job.ID, int(processingTime.Milliseconds()), true)
	}

	log.Printf("Job %s completed successfully in %v", job.ID, processingTime)
	return nil
}

// processImageConversion processes an image conversion job with comprehensive error handling
func (s *Service) processImageConversion(ctx context.Context, job *WorkerJob) (interface{}, error) {
	// Get conversion details
	conversion, err := s.conversionStore.GetConversion(ctx, job.ConversionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion: %w", err)
	}

	// Get user image
	userImage, err := s.imageStore.GetImage(ctx, conversion.UserImageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user image: %w", err)
	}

	// Get cloth image
	clothImage, err := s.imageStore.GetImage(ctx, conversion.ClothImageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloth image: %w", err)
	}

	// Download images with retry logic
	userImageData, err := s.downloadImageWithRetry(ctx, userImage.OriginalURL, "user image")
	if err != nil {
		return nil, fmt.Errorf("failed to download user image: %w", err)
	}

	clothImageData, err := s.downloadImageWithRetry(ctx, clothImage.OriginalURL, "cloth image")
	if err != nil {
		return nil, fmt.Errorf("failed to download cloth image: %w", err)
	}

	// Validate downloaded images
	if err := s.validateImages(ctx, userImageData, clothImageData); err != nil {
		return nil, fmt.Errorf("image validation failed: %w", err)
	}

	// Call Gemini API for conversion with timeout
	resultImageData, err := s.convertImageWithTimeout(ctx, userImageData, clothImageData, job.Payload.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to convert image with Gemini: %w", err)
	}

	// Process the result image
	processedData, width, height, err := s.imageProcessor.ProcessImage(ctx, resultImageData, "converted_"+userImage.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to process result image: %w", err)
	}

	// Generate storage path for result
	storagePath := fmt.Sprintf("results/%s/%s", job.UserID, job.ConversionID)

	// Upload result image with retry
	resultURL, err := s.uploadFileWithRetry(ctx, processedData, "converted_"+userImage.FileName, storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload result image: %w", err)
	}

	// Generate thumbnail
	thumbnailData, err := s.imageProcessor.GenerateThumbnail(ctx, processedData, "converted_"+userImage.FileName, 300, 300)
	if err != nil {
		log.Printf("Failed to generate thumbnail: %v", err)
		// Continue without thumbnail
	}

	var thumbnailURL *string
	if thumbnailData != nil {
		thumbURL, err := s.fileStorage.UploadFile(ctx, thumbnailData, "thumb_"+userImage.FileName, storagePath+"/thumbnails")
		if err != nil {
			log.Printf("Failed to upload thumbnail: %v", err)
		} else {
			thumbnailURL = &thumbURL
		}
	}

	// Create result image record
	createReq := image.CreateImageRequest{
		UserID:       &job.UserID,
		Type:         image.ImageTypeResult,
		FileName:     "converted_" + userImage.FileName,
		OriginalURL:  resultURL,
		ThumbnailURL: thumbnailURL,
		FileSize:     int64(len(processedData)),
		MimeType:     userImage.MimeType,
		Width:        &width,
		Height:       &height,
		IsPublic:     false,
		Tags:         []string{"converted", "ai-generated"},
		Metadata: map[string]interface{}{
			"conversion_id":  job.ConversionID,
			"user_image_id":  conversion.UserImageID,
			"cloth_image_id": conversion.ClothImageID,
			"processed_at":   time.Now().Unix(),
		},
	}

	resultImage, err := s.imageStore.CreateImage(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create result image record: %w", err)
	}

	return resultImage.ID, nil
}

// updateConversionStatus updates the conversion status in the database
func (s *Service) updateConversionStatus(ctx context.Context, conversionID, status string, result interface{}, errorMessage string, processingTimeMs int) error {
	updateReq := conversion.UpdateConversionRequest{
		Status:           &status,
		ProcessingTimeMs: &processingTimeMs,
	}

	if result != nil {
		if resultImageID, ok := result.(string); ok {
			updateReq.ResultImageID = &resultImageID
		}
	}

	if errorMessage != "" {
		updateReq.ErrorMessage = &errorMessage
	}

	return s.conversionStore.UpdateConversion(ctx, conversionID, updateReq)
}

// workerLoop is the main worker loop
func (s *Service) workerLoop(ctx context.Context, workerID string) {
	worker := &Worker{
		ID:        workerID,
		Status:    "idle",
		StartedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	s.workerMutex.Lock()
	s.workers[workerID] = worker
	s.workerMutex.Unlock()

	defer func() {
		s.workerMutex.Lock()
		delete(s.workers, workerID)
		s.workerMutex.Unlock()
	}()

	log.Printf("Worker %s started", workerID)

	for {
		select {
		case <-s.stopChan:
			log.Printf("Worker %s stopping", workerID)
			return
		case <-ctx.Done():
			log.Printf("Worker %s stopping due to context cancellation", workerID)
			return
		default:
			// Try to get a job
			job, err := s.jobQueue.DequeueJob(ctx, workerID)
			if err != nil {
				log.Printf("Failed to dequeue job: %v", err)
				time.Sleep(s.config.PollInterval)
				continue
			}

			if job == nil {
				// No jobs available, wait
				time.Sleep(s.config.PollInterval)
				continue
			}

			// Update worker status
			worker.Status = "processing"
			worker.CurrentJob = job
			worker.LastSeen = time.Now()

			// Process the job
			if err := s.ProcessJob(ctx, job); err != nil {
				log.Printf("Worker %s failed to process job %s: %v", workerID, job.ID, err)
			}

			// Update worker status
			worker.Status = "idle"
			worker.CurrentJob = nil
			worker.JobsProcessed++
			worker.LastSeen = time.Now()
		}
	}
}

// cleanupLoop periodically cleans up old jobs
func (s *Service) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Clean up jobs older than 24 hours
			cutoff := time.Now().Add(-24 * time.Hour)
			if err := s.jobQueue.CleanupOldJobs(ctx, cutoff); err != nil {
				log.Printf("Failed to cleanup old jobs: %v", err)
			}
		}
	}
}

// healthCheckLoop periodically updates worker health
func (s *Service) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update health status
			if s.metricsCollector != nil {
				s.metricsCollector.RecordWorkerHealth(ctx, s.workerID, "healthy")
			}
		}
	}
}

// GetStatus returns the current status of the worker service
func (s *Service) GetStatus(ctx context.Context) (*WorkerStats, error) {
	return s.jobQueue.GetQueueStats(ctx)
}

// GetHealth returns the health status of this worker
func (s *Service) GetHealth(ctx context.Context) (*WorkerHealth, error) {
	s.workerMutex.RLock()
	worker, exists := s.workers[s.workerID]
	s.workerMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("worker not found")
	}

	uptime := time.Since(worker.StartedAt).Seconds()

	return &WorkerHealth{
		WorkerID:      s.workerID,
		Status:        worker.Status,
		LastSeen:      worker.LastSeen,
		JobsProcessed: worker.JobsProcessed,
		CurrentJob: func() *string {
			if worker.CurrentJob != nil {
				return &worker.CurrentJob.ID
			} else {
				return nil
			}
		}(),
		Uptime: int64(uptime),
	}, nil
}

// CancelJob cancels a job
func (s *Service) CancelJob(ctx context.Context, jobID string) error {
	job, err := s.jobQueue.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status != JobStatusPending {
		return fmt.Errorf("job %s is not in pending status", jobID)
	}

	return s.jobQueue.UpdateJobStatus(ctx, jobID, JobStatusCancelled, s.workerID)
}

// UpdateConfig updates the worker configuration
func (s *Service) UpdateConfig(ctx context.Context, config *WorkerConfig) error {
	s.config = config
	return nil
}

// GetConfig returns the current configuration
func (s *Service) GetConfig(ctx context.Context) (*WorkerConfig, error) {
	return s.config, nil
}

// Helper functions

func getDefaultConfig() *WorkerConfig {
	return &WorkerConfig{
		MaxWorkers:        DefaultMaxWorkers,
		JobTimeout:        DefaultJobTimeout,
		RetryDelay:        DefaultRetryDelay,
		MaxRetries:        DefaultMaxRetries,
		PollInterval:      DefaultPollInterval,
		CleanupInterval:   DefaultCleanupInterval,
		HealthCheckPort:   DefaultHealthCheckPort,
		EnableMetrics:     true,
		EnableHealthCheck: true,
	}
}

func generateWorkerID() string {
	return fmt.Sprintf("worker-%d", time.Now().UnixNano())
}

func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}

// downloadImageWithRetry downloads an image with retry logic
func (s *Service) downloadImageWithRetry(ctx context.Context, url, description string) ([]byte, error) {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		data, err := s.fileStorage.GetFile(ctx, url)
		if err == nil {
			return data, nil
		}

		lastErr = err
		log.Printf("Failed to download %s (attempt %d/%d): %v", description, attempt+1, maxRetries, err)

		// Check if error is retryable
		if !s.isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error downloading %s: %w", description, err)
		}

		// Wait before retry
		if attempt < maxRetries-1 {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("failed to download %s after %d attempts: %w", description, maxRetries, lastErr)
}

// uploadFileWithRetry uploads a file with retry logic
func (s *Service) uploadFileWithRetry(ctx context.Context, data []byte, filename, path string) (string, error) {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		url, err := s.fileStorage.UploadFile(ctx, data, filename, path)
		if err == nil {
			return url, nil
		}

		lastErr = err
		log.Printf("Failed to upload file %s (attempt %d/%d): %v", filename, attempt+1, maxRetries, err)

		// Check if error is retryable
		if !s.isRetryableError(err) {
			return "", fmt.Errorf("non-retryable error uploading %s: %w", filename, err)
		}

		// Wait before retry
		if attempt < maxRetries-1 {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(delay)
		}
	}

	return "", fmt.Errorf("failed to upload %s after %d attempts: %w", filename, maxRetries, lastErr)
}

// validateImages validates downloaded images
func (s *Service) validateImages(ctx context.Context, userImageData, clothImageData []byte) error {
	// Check if images are empty
	if len(userImageData) == 0 {
		return fmt.Errorf("user image is empty")
	}
	if len(clothImageData) == 0 {
		return fmt.Errorf("cloth image is empty")
	}

	// Check file sizes
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if int64(len(userImageData)) > maxSize {
		return fmt.Errorf("user image too large: %d bytes (max: %d)", len(userImageData), maxSize)
	}
	if int64(len(clothImageData)) > maxSize {
		return fmt.Errorf("cloth image too large: %d bytes (max: %d)", len(clothImageData), maxSize)
	}

	// Basic format validation
	if err := s.validateImageFormat(userImageData, "user image"); err != nil {
		return err
	}
	if err := s.validateImageFormat(clothImageData, "cloth image"); err != nil {
		return err
	}

	return nil
}

// validateImageFormat validates image format
func (s *Service) validateImageFormat(data []byte, description string) error {
	if len(data) < 4 {
		return fmt.Errorf("%s is too small to be a valid image", description)
	}

	// Check for common image formats
	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return nil
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return nil
	}

	// WebP
	if len(data) >= 12 &&
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return nil
	}

	// GIF
	if len(data) >= 6 &&
		data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return nil
	}

	return fmt.Errorf("%s has unsupported format", description)
}

// convertImageWithTimeout converts image with timeout
func (s *Service) convertImageWithTimeout(ctx context.Context, userImageData, clothImageData []byte, options map[string]interface{}) ([]byte, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Use a channel to handle the conversion
	resultChan := make(chan struct {
		data []byte
		err  error
	}, 1)

	go func() {
		data, err := s.geminiAPI.ConvertImage(timeoutCtx, userImageData, clothImageData, options)
		resultChan <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	select {
	case result := <-resultChan:
		return result.data, result.err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("image conversion timed out after 5 minutes")
	}
}

// isRetryableError checks if an error is retryable
func (s *Service) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retryable errors
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network unreachable",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"server error",
		"internal server error",
		"bad gateway",
		"gateway timeout",
		"request timeout",
		"context deadline exceeded",
	}

	for _, retryableError := range retryableErrors {
		if containsService(errStr, retryableError) {
			return true
		}
	}

	return false
}

// containsService checks if a string contains a substring (case-insensitive)
func containsService(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstringService(s, substr)))
}

// containsSubstringService performs case-insensitive substring search
func containsSubstringService(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
