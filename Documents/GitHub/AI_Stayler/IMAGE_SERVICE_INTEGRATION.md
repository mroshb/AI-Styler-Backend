# Image Service Integration Guide

## Overview

The Image Service has been successfully designed and implemented as a comprehensive image management system for the AI Stayler application. This service provides centralized image handling for users, vendors, and AI-generated results.

## Architecture

### Service Components

1. **Models** (`internal/image/models.go`)
   - Image data structures
   - Request/response types
   - Constants and validation rules

2. **Interfaces** (`internal/image/interfaces.go`)
   - Service contracts
   - Dependency injection interfaces
   - Configuration types

3. **Service** (`internal/image/service.go`)
   - Core business logic
   - Image processing and validation
   - Quota management

4. **Handler** (`internal/image/handler.go`)
   - HTTP request handling
   - Input validation and parsing
   - Response formatting

5. **Routes** (`internal/image/routes.go`)
   - Route configuration
   - Public and private endpoints

6. **Context** (`internal/image/context.go`)
   - User/vendor context management
   - Authentication helpers

7. **Database Migration** (`db/migrations/0004_image_service.sql`)
   - Complete database schema
   - Quota tracking functions
   - Usage history tables

## Key Features

### ✅ Image Upload Service
- **Multi-type Support**: User, vendor, and result images
- **Validation**: Type, size, and format validation
- **Processing**: Automatic resizing and thumbnail generation
- **Storage**: Organized folder structure (users/, vendors/, results/)
- **Signed URLs**: Secure, time-limited access

### ✅ Usage Tracking
- **Complete Audit Trail**: All image operations logged
- **Usage History**: Per-image usage tracking
- **Analytics**: Statistics and reporting

### ✅ Quota Management
- **Per-User Limits**: 100 images, 1GB storage
- **Per-Vendor Limits**: 1000 images, 5GB storage
- **Monthly Tracking**: Automatic quota reset
- **Warning System**: Low quota notifications

### ✅ Database Tables
- **images**: Main image metadata
- **image_usage_history**: Usage tracking
- **image_quota_tracking**: Quota management

## Integration Points

### 1. Database Integration
The service uses the existing PostgreSQL database with new tables:
```sql
-- New tables added
CREATE TABLE images (...);
CREATE TABLE image_usage_history (...);
CREATE TABLE image_quota_tracking (...);
```

### 2. Storage Integration
- **Local Storage**: Files stored in organized directory structure
- **Path Structure**: `uploads/{type}/{owner_id}/`
- **Thumbnails**: Auto-generated in `thumbnails/` subdirectories

### 3. Authentication Integration
- Uses existing user/vendor context system
- Integrates with current authentication middleware
- Supports both user and vendor image ownership

### 4. Configuration Integration
Extends existing config system:
```go
type StorageConfig struct {
    UploadMaxSize   int64
    StoragePath     string
    SignedURLTTL    int64
    AllowedTypes    []string
}
```

## API Endpoints

### Image Management
```
POST   /images                    # Upload image
GET    /images                    # List images
GET    /images/{id}               # Get image details
PUT    /images/{id}               # Update image
DELETE /images/{id}               # Delete image
```

### Signed URLs
```
POST   /images/{id}/signed-url    # Generate signed URL
```

### Usage & Analytics
```
GET    /images/{id}/usage         # Usage history
GET    /quota                     # Quota status
GET    /stats                     # Image statistics
```

### Public Access
```
GET    /public/images/{id}        # Public image access
GET    /public/images             # List public images
```

## File Structure

```
internal/image/
├── models.go          # Data structures and types
├── interfaces.go      # Service contracts
├── service.go         # Core business logic
├── handler.go         # HTTP handlers
├── routes.go          # Route configuration
├── context.go         # Context management
├── wire.go           # Dependency injection
├── service_test.go   # Comprehensive tests
└── README.md         # Documentation

db/migrations/
└── 0004_image_service.sql  # Database schema
```

## Usage Examples

### Upload User Image
```bash
curl -X POST http://localhost:8080/images \
  -H "Authorization: Bearer <token>" \
  -F "file=@profile.jpg" \
  -F "type=user" \
  -F "isPublic=false" \
  -F "tags=profile,avatar"
```

### Generate Signed URL
```bash
curl -X POST http://localhost:8080/images/{id}/signed-url \
  -H "Authorization: Bearer <token>" \
  -d '{"accessType": "view"}'
```

### List Images with Filtering
```bash
curl -X GET "http://localhost:8080/images?type=user&page=1&pageSize=20" \
  -H "Authorization: Bearer <token>"
```

## Security Features

### Access Control
- **Private by Default**: User images are private
- **Public Option**: Images can be marked as public
- **Signed URLs**: Time-limited access tokens
- **Ownership Validation**: Users can only access their own images

### Validation
- **File Type**: MIME type and extension validation
- **File Size**: Configurable limits per image type
- **Image Integrity**: Malicious file detection
- **Rate Limiting**: Upload and API call limits

## Performance Optimizations

### Caching
- **Image Metadata**: Cached for fast retrieval
- **Signed URLs**: Cached to reduce generation overhead
- **Thumbnails**: Cached for quick access

### Processing
- **Lazy Thumbnails**: Generated on-demand
- **Background Processing**: Non-blocking image operations
- **Optimized Storage**: Efficient file organization

## Monitoring & Analytics

### Usage Tracking
- **Complete Audit Trail**: All operations logged
- **IP Tracking**: Request source tracking
- **User Agent**: Client information
- **Metadata**: Custom tracking data

### Quota Monitoring
- **Real-time Status**: Current quota usage
- **Automatic Warnings**: Low quota alerts
- **Monthly Reset**: Automatic quota refresh

## Testing

### Test Coverage
- **Unit Tests**: Service logic testing
- **Integration Tests**: End-to-end testing
- **Mock Implementations**: All dependencies mocked
- **Error Scenarios**: Comprehensive error testing

### Test Files
- `service_test.go`: Comprehensive test suite
- Mock implementations for all interfaces
- Test data and scenarios

## Future Enhancements

### Planned Features
- **AI Integration**: Automatic image analysis
- **CDN Support**: Content delivery network integration
- **Batch Operations**: Bulk image processing
- **Image Search**: Advanced search capabilities
- **Watermarking**: Automatic watermark application
- **Compression**: Advanced image optimization

### Scalability
- **Horizontal Scaling**: Multi-instance support
- **Database Sharding**: Large-scale data handling
- **File Storage**: Cloud storage integration
- **Caching**: Distributed caching support

## Dependencies

### Required Interfaces
- `Store`: Database operations
- `FileStorage`: File storage operations
- `ImageProcessor`: Image processing
- `UsageTracker`: Usage tracking
- `Cache`: Caching operations
- `NotificationService`: Notifications
- `AuditLogger`: Audit logging
- `RateLimiter`: Rate limiting

### External Dependencies
- Image processing library
- File storage backend
- Database (PostgreSQL)
- Cache (Redis)
- Message queue (optional)

## Conclusion

The Image Service provides a complete, production-ready image management solution that integrates seamlessly with the existing AI Stayler application. It offers comprehensive features for image handling, usage tracking, quota management, and security while maintaining high performance and scalability.

The service is designed with clean architecture principles, making it easy to extend and maintain. All components are thoroughly tested and documented, ensuring reliable operation in production environments.
