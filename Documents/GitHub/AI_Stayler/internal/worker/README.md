# Worker Service

The Worker Service provides background job processing for the AI Stayler application, specifically handling image conversion tasks using the Gemini API.

## Features

### Core Functionality
- **Job Queue Management**: In-memory and Redis-based job queues
- **Image Processing**: Integration with Gemini API for AI-powered image conversion
- **Retry Mechanism**: Exponential backoff with configurable retry policies
- **Worker Management**: Multiple worker instances with health monitoring
- **Status Tracking**: Real-time job status and progress tracking
- **Metrics Collection**: Comprehensive metrics for monitoring and alerting

### Job Types
- **Image Conversion**: Convert user images to wear clothing items using AI
- **Image Processing**: Resize, thumbnail generation, and format conversion
- **Batch Processing**: Handle multiple jobs concurrently

### Worker Architecture
- **Scalable Workers**: Configurable number of worker instances
- **Health Monitoring**: Worker health checks and automatic recovery
- **Load Balancing**: Intelligent job distribution across workers
- **Graceful Shutdown**: Clean shutdown with job completion

## API Endpoints

### Worker Management
- `POST /worker/start` - Start the worker service
- `POST /worker/stop` - Stop the worker service
- `GET /worker/status` - Get worker service status and statistics
- `GET /worker/health` - Get worker health status

### Job Management
- `POST /worker/jobs` - Enqueue a new job
- `GET /worker/jobs/{id}` - Get job details
- `DELETE /worker/jobs/{id}` - Cancel a job
- `POST /worker/jobs/{id}/process` - Process a specific job

### Configuration
- `GET /worker/config` - Get worker configuration
- `PUT /worker/config` - Update worker configuration

### Monitoring
- `GET /worker/stats` - Get worker statistics
- `GET /worker/workers` - Get list of active workers
- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint

## Configuration

### Environment Variables
```bash
# Worker configuration
WORKER_MAX_WORKERS=5
WORKER_JOB_TIMEOUT=600s
WORKER_RETRY_DELAY=30s
WORKER_MAX_RETRIES=3
WORKER_POLL_INTERVAL=5s
WORKER_CLEANUP_INTERVAL=1h
WORKER_HEALTH_CHECK_PORT=8081
WORKER_ENABLE_METRICS=true
WORKER_ENABLE_HEALTH_CHECK=true

# Gemini API configuration
GEMINI_API_KEY=your_api_key
GEMINI_BASE_URL=https://generativelanguage.googleapis.com
GEMINI_MODEL=gemini-1.5-pro
GEMINI_MAX_RETRIES=3
GEMINI_TIMEOUT=60

# Retry configuration
RETRY_MAX_RETRIES=3
RETRY_INITIAL_DELAY=5s
RETRY_MAX_DELAY=5m
RETRY_BACKOFF_FACTOR=2.0
RETRY_JITTER=true
```

## Usage Examples

### Starting the Worker Service
```go
// Create configuration
config := &worker.WorkerConfig{
    MaxWorkers:        5,
    JobTimeout:        10 * time.Minute,
    RetryDelay:        30 * time.Second,
    MaxRetries:        3,
    PollInterval:      5 * time.Second,
    CleanupInterval:   1 * time.Hour,
    EnableMetrics:     true,
    EnableHealthCheck: true,
}

// Create service
service := worker.NewService(
    config,
    jobQueue,
    imageProcessor,
    fileStorage,
    conversionStore,
    imageStore,
    geminiAPI,
    notifier,
    metricsCollector,
    healthChecker,
    retryHandler,
)

// Start service
ctx := context.Background()
if err := service.Start(ctx); err != nil {
    log.Fatal("Failed to start worker service:", err)
}
```

### Enqueuing a Job
```go
// Create job payload
payload := worker.JobPayload{
    UserImageID:  "user-img-123",
    ClothImageID: "cloth-img-456",
    Options: map[string]interface{}{
        "style": "casual",
        "quality": "high",
    },
}

// Enqueue job
err := service.EnqueueJob(
    ctx,
    "image_conversion",
    "conv-789",
    "user-123",
    payload,
)
if err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

### Monitoring Worker Health
```go
// Get worker health
health, err := service.GetHealth(ctx)
if err != nil {
    log.Printf("Failed to get health: %v", err)
    return
}

log.Printf("Worker %s status: %s", health.WorkerID, health.Status)
log.Printf("Jobs processed: %d", health.JobsProcessed)
log.Printf("Uptime: %d seconds", health.Uptime)
```

## Job Processing Flow

1. **Job Creation**: Job is created and enqueued with pending status
2. **Job Pickup**: Available worker picks up the job and marks it as processing
3. **Image Download**: Worker downloads user and cloth images from storage
4. **AI Processing**: Images are sent to Gemini API for conversion
5. **Result Processing**: Converted image is processed and optimized
6. **Storage Upload**: Result image is uploaded to storage
7. **Database Update**: Conversion record is updated with result
8. **Notification**: User is notified of completion or failure
9. **Cleanup**: Job is marked as completed and cleaned up

## Retry Mechanism

### Retry Policies
- **Exponential Backoff**: Delay increases exponentially with each retry
- **Jitter**: Random variation in delay to prevent thundering herd
- **Max Retries**: Configurable maximum retry attempts
- **Error Classification**: Automatic classification of retryable vs non-retryable errors

### Retryable Errors
- Network timeouts
- Connection errors
- Rate limiting
- Temporary service unavailability
- Server errors (5xx)

### Non-Retryable Errors
- Invalid input data
- Authentication failures
- Permission denied
- File not found
- Quota exceeded

## Metrics and Monitoring

### Prometheus Metrics
- `worker_jobs_total` - Total number of jobs
- `worker_jobs_pending` - Number of pending jobs
- `worker_jobs_processing` - Number of processing jobs
- `worker_jobs_completed` - Number of completed jobs
- `worker_jobs_failed` - Number of failed jobs
- `worker_active_workers` - Number of active workers
- `worker_average_job_time` - Average job processing time
- `worker_success_rate` - Job success rate

### Health Checks
- Worker availability
- Job queue status
- External service connectivity
- Resource utilization

## Error Handling

### Job-Level Errors
- Individual job failures don't affect other jobs
- Detailed error logging and tracking
- Automatic retry with exponential backoff
- Dead letter queue for permanently failed jobs

### Service-Level Errors
- Graceful degradation when external services are unavailable
- Circuit breaker pattern for API calls
- Automatic recovery and restart
- Alerting for critical failures

## Performance Considerations

### Scalability
- Horizontal scaling with multiple worker instances
- Load balancing across workers
- Queue-based architecture for decoupling
- Efficient resource utilization

### Optimization
- Connection pooling for external APIs
- Image caching and optimization
- Batch processing where possible
- Memory-efficient image handling

## Security

### API Security
- Secure API key management
- Request validation and sanitization
- Rate limiting and abuse prevention
- Audit logging for all operations

### Data Protection
- Secure image storage and transmission
- Data encryption in transit and at rest
- Access control and permissions
- Privacy-compliant data handling

## Testing

### Unit Tests
- Service logic testing
- Mock external dependencies
- Error scenario testing
- Performance testing

### Integration Tests
- End-to-end job processing
- External service integration
- Database operations
- Queue operations

### Load Testing
- High-volume job processing
- Worker scaling tests
- Memory and CPU usage
- Queue performance under load

## Deployment

### Docker Support
- Containerized worker service
- Environment-based configuration
- Health check endpoints
- Graceful shutdown handling

### Kubernetes Support
- Deployment manifests
- Service discovery
- ConfigMaps and Secrets
- Horizontal Pod Autoscaling

### Monitoring Integration
- Prometheus metrics export
- Grafana dashboards
- Alerting rules
- Log aggregation
