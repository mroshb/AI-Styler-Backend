# Conversion Service

The Conversion Service handles image conversion requests, managing user quotas, and processing conversions through background workers.

## Features

- **Conversion Management**: Create, retrieve, update, and delete conversion requests
- **Quota Management**: Check and enforce user conversion limits (2 free conversions)
- **Background Processing**: Queue conversion jobs for asynchronous processing
- **Status Tracking**: Track conversion status (pending, processing, completed, failed)
- **Rate Limiting**: Prevent abuse with rate limiting
- **Audit Logging**: Log all conversion activities
- **Metrics Collection**: Track conversion performance and usage

## API Endpoints

### Conversion Operations

- `POST /convert` - Create a new conversion request
- `GET /conversion/{id}` - Get conversion details
- `PUT /conversion/{id}` - Update conversion (status updates)
- `DELETE /conversion/{id}` - Delete conversion (pending/failed only)
- `POST /conversion/{id}/cancel` - Cancel a pending conversion
- `GET /conversion/{id}/status` - Get processing status

### List Operations

- `GET /conversions` - List user's conversions with pagination

### Quota & Metrics

- `GET /convert/quota` - Get user's quota status
- `GET /convert/metrics` - Get conversion metrics

## Models

### ConversionRequest
```json
{
  "userImageId": "uuid",
  "clothImageId": "uuid"
}
```

### ConversionResponse
```json
{
  "id": "uuid",
  "userId": "uuid",
  "userImageId": "uuid",
  "clothImageId": "uuid",
  "status": "pending|processing|completed|failed",
  "resultImageId": "uuid",
  "errorMessage": "string",
  "processingTimeMs": 1234,
  "createdAt": "2023-01-01T00:00:00Z",
  "updatedAt": "2023-01-01T00:00:00Z",
  "completedAt": "2023-01-01T00:00:00Z"
}
```

### QuotaCheck
```json
{
  "canConvert": true,
  "remainingFree": 1,
  "remainingPaid": 10,
  "totalRemaining": 11,
  "planName": "free",
  "monthlyLimit": 0
}
```

## Database Schema

### conversions table
- `id` - UUID primary key
- `user_id` - Foreign key to users table
- `user_image_id` - Foreign key to images table (user's image)
- `cloth_image_id` - Foreign key to images table (cloth image)
- `status` - Conversion status (pending, processing, completed, failed)
- `result_image_id` - Foreign key to images table (result image)
- `error_message` - Error message if failed
- `processing_time_ms` - Processing time in milliseconds
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp
- `completed_at` - Completion timestamp

### conversion_jobs table
- `id` - UUID primary key
- `conversion_id` - Foreign key to conversions table
- `status` - Job status (queued, processing, completed, failed, cancelled)
- `worker_id` - ID of the worker processing the job
- `priority` - Job priority (higher = more important)
- `retry_count` - Number of retries attempted
- `max_retries` - Maximum retries allowed
- `error_message` - Error message if failed
- `started_at` - When processing started
- `completed_at` - When processing completed
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### conversion_metrics table
- `id` - UUID primary key
- `conversion_id` - Foreign key to conversions table
- `user_id` - Foreign key to users table
- `processing_time_ms` - Processing time in milliseconds
- `input_file_size_bytes` - Size of input file
- `output_file_size_bytes` - Size of output file
- `success` - Whether conversion was successful
- `error_type` - Type of error if failed
- `created_at` - Creation timestamp

## Business Logic

### Quota Management
- Users get 2 free conversions by default
- Free conversions are used first
- Paid conversions require an active subscription plan
- Quota is checked before creating a conversion
- Quota is decremented immediately upon conversion creation

### Conversion Flow
1. User submits conversion request with user image ID and cloth image ID
2. System validates image access and cloth image availability
3. System checks user quota
4. System creates conversion record with status "pending"
5. System creates background job and enqueues it
6. Worker picks up job and updates status to "processing"
7. Worker processes the conversion
8. Worker updates status to "completed" or "failed"
9. System sends notification to user

### Status Transitions
- `pending` → `processing` (when worker starts)
- `processing` → `completed` (on success)
- `processing` → `failed` (on error)
- `pending` → `failed` (if cancelled)

## Error Handling

### Common Error Codes
- `quota_exceeded` - User has no remaining conversions
- `rate_limit_exceeded` - Too many requests
- `invalid_request` - Invalid input data
- `not_found` - Conversion not found
- `unauthorized` - User not authenticated

### Error Response Format
```json
{
  "error": {
    "code": "quota_exceeded",
    "message": "User quota exceeded: free=0, paid=0",
    "details": {}
  }
}
```

## Dependencies

The conversion service depends on:
- **Image Service**: For image validation and result storage
- **User Service**: For user quota management
- **Worker Service**: For background job processing
- **Notification Service**: For sending user notifications
- **Rate Limiter**: For preventing abuse
- **Audit Logger**: For logging activities
- **Metrics Collector**: For performance tracking

## Configuration

### Environment Variables
- `CONVERSION_RATE_LIMIT` - Requests per minute per user (default: 10)
- `CONVERSION_MAX_RETRIES` - Maximum job retries (default: 3)
- `CONVERSION_TIMEOUT_MS` - Processing timeout (default: 300000)

### Database Functions
- `create_conversion()` - Creates conversion and reserves quota
- `update_conversion_status()` - Updates conversion status
- `get_conversion_with_details()` - Gets conversion with image URLs
- `get_user_quota_status()` - Gets user's current quota status

## Testing

The service includes comprehensive tests for:
- Conversion creation and validation
- Quota checking and enforcement
- Status updates and transitions
- Error handling and edge cases
- Database operations
- API endpoints

Run tests with:
```bash
go test ./internal/conversion/...
```

## Monitoring

Key metrics to monitor:
- Conversion success rate
- Average processing time
- Quota usage patterns
- Error rates by type
- Queue depth and processing time
- User conversion patterns
