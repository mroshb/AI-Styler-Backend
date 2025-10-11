# Image Service

The Image Service provides comprehensive image management functionality for the AI Stayler application, supporting user images, vendor images, and result images.

## Features

### Core Functionality
- **Image Upload**: Support for multiple image types (user, vendor, result)
- **Image Validation**: Type, size, and format validation
- **Image Processing**: Automatic resizing, thumbnail generation
- **Storage Management**: Local file storage with organized folder structure
- **Signed URLs**: Secure, time-limited access to images
- **Usage Tracking**: Complete audit trail of image usage
- **Quota Management**: Per-user and per-vendor image limits

### Image Types
- **User Images**: Personal images uploaded by users
- **Vendor Images**: Business images uploaded by vendors
- **Result Images**: AI-generated or processed images

### Storage Structure
```
uploads/
├── users/
│   └── {user_id}/
│       ├── images/
│       └── thumbnails/
├── vendors/
│   └── {vendor_id}/
│       ├── images/
│       └── thumbnails/
└── results/
    ├── {user_id}/
    └── vendor/
        └── {vendor_id}/
```

## API Endpoints

### Image Management
- `POST /images` - Upload a new image
- `GET /images` - List images with filtering and pagination
- `GET /images/{id}` - Get specific image details
- `PUT /images/{id}` - Update image metadata
- `DELETE /images/{id}` - Delete an image

### Signed URLs
- `POST /images/{id}/signed-url` - Generate signed URL for image access

### Usage Tracking
- `GET /images/{id}/usage` - Get image usage history

### Quota & Statistics
- `GET /quota` - Get current quota status
- `GET /stats` - Get image statistics

### Public Access
- `GET /public/images/{id}` - Get public image details
- `GET /public/images` - List public images

## Configuration

### Environment Variables
```bash
# Storage configuration
UPLOAD_MAX_SIZE=50MB
STORAGE_PATH=./uploads
SIGNED_URL_TTL=3600

# Quota limits
USER_IMAGE_LIMIT=100
VENDOR_IMAGE_LIMIT=1000
USER_FILE_SIZE_LIMIT=1073741824  # 1GB
VENDOR_FILE_SIZE_LIMIT=5368709120  # 5GB
```

### Supported Image Types
- JPEG (.jpg, .jpeg)
- PNG (.png)
- GIF (.gif)
- WebP (.webp)
- SVG (.svg)
- BMP (.bmp)
- TIFF (.tiff, .tif)

### File Size Limits
- **User Images**: 10MB per image, 1GB total
- **Vendor Images**: 50MB per image, 5GB total
- **Result Images**: 50MB per image, follows user/vendor limits

## Database Schema

### Images Table
```sql
CREATE TABLE images (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    vendor_id UUID REFERENCES vendors(id),
    type VARCHAR(20) NOT NULL, -- 'user', 'vendor', 'result'
    file_name TEXT NOT NULL,
    original_url TEXT NOT NULL,
    thumbnail_url TEXT,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    is_public BOOLEAN DEFAULT false,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Image Usage History Table
```sql
CREATE TABLE image_usage_history (
    id UUID PRIMARY KEY,
    image_id UUID REFERENCES images(id),
    user_id UUID REFERENCES users(id),
    action VARCHAR(50) NOT NULL, -- 'upload', 'view', 'download', 'delete', 'update'
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Quota Tracking Table
```sql
CREATE TABLE image_quota_tracking (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    vendor_id UUID REFERENCES vendors(id),
    year_month TEXT NOT NULL, -- 'YYYY-MM'
    user_images_used INTEGER DEFAULT 0,
    vendor_images_used INTEGER DEFAULT 0,
    result_images_used INTEGER DEFAULT 0,
    total_images_used INTEGER DEFAULT 0,
    total_file_size BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Usage Examples

### Upload an Image
```bash
curl -X POST http://localhost:8080/images \
  -H "Authorization: Bearer <token>" \
  -F "file=@image.jpg" \
  -F "type=user" \
  -F "isPublic=false" \
  -F "tags=profile,avatar"
```

### Generate Signed URL
```bash
curl -X POST http://localhost:8080/images/{image_id}/signed-url \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"accessType": "view"}'
```

### List Images
```bash
curl -X GET "http://localhost:8080/images?type=user&page=1&pageSize=20" \
  -H "Authorization: Bearer <token>"
```

## Security Features

### Access Control
- User images are private by default
- Vendor images can be public or private
- Result images follow user/vendor permissions
- Signed URLs provide time-limited access

### Validation
- File type validation based on MIME type and extension
- File size limits per image type
- Image dimension validation
- Malicious file detection

### Rate Limiting
- Upload rate limiting per user/vendor
- API call rate limiting
- Configurable limits and windows

## Monitoring & Analytics

### Usage Tracking
- Complete audit trail of all image operations
- IP address and user agent tracking
- Metadata storage for analytics

### Quota Monitoring
- Real-time quota status
- Automatic warnings when approaching limits
- Monthly quota reset

### Statistics
- Total images per user/vendor
- File size statistics
- Usage patterns and trends

## Error Handling

### Common Error Codes
- `400 Bad Request` - Invalid input or validation errors
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Image not found
- `413 Payload Too Large` - File size exceeds limit
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

### Error Response Format
```json
{
  "error": {
    "code": "validation_error",
    "message": "File size too large",
    "details": {
      "max_size": "10MB",
      "actual_size": "15MB"
    }
  }
}
```

## Testing

Run the test suite:
```bash
go test ./internal/image/...
```

Run with coverage:
```bash
go test -cover ./internal/image/...
```

## Dependencies

### Required Interfaces
- `Store` - Database operations
- `FileStorage` - File storage operations
- `ImageProcessor` - Image processing
- `UsageTracker` - Usage tracking
- `Cache` - Caching operations
- `NotificationService` - Notifications
- `AuditLogger` - Audit logging
- `RateLimiter` - Rate limiting

### External Dependencies
- Image processing library (e.g., imaging, gocv)
- File storage backend (local filesystem, S3, etc.)
- Database (PostgreSQL)
- Cache (Redis)
- Message queue (optional)

## Performance Considerations

### Caching
- Image metadata caching
- Signed URL caching
- Thumbnail caching

### Optimization
- Lazy thumbnail generation
- Background image processing
- CDN integration support

### Scalability
- Horizontal scaling support
- Database connection pooling
- File storage abstraction

## Future Enhancements

### Planned Features
- AI-powered image analysis
- Automatic tagging
- Image search and filtering
- Batch operations
- Image compression optimization
- CDN integration
- Image versioning
- Watermarking support
