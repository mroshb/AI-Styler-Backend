# Storage Architecture Documentation

## Overview

The AI Stayler storage architecture provides a comprehensive file storage system designed for image management with local server folders, database metadata tracking, signed URLs for secure access, and a backup & retention policy that keeps images forever.

## Architecture Components

### 1. Local Server Folder Structure

The storage system organizes files in a hierarchical folder structure:

```
/storage/
├── images/
│   ├── user/           # User uploaded images
│   │   └── {user_id}/
│   │       ├── images/
│   │       └── thumbnails/
│   ├── cloth/          # Clothing images
│   │   └── {vendor_id}/
│   │       ├── images/
│   │       └── thumbnails/
│   └── result/          # AI-generated result images
│       ├── {user_id}/
│       └── vendor/
│           └── {vendor_id}/
└── backups/            # Backup storage
    └── {date}/
        └── {files}
```

### 2. Database Metadata Tracking

Comprehensive metadata is tracked in PostgreSQL with the following tables:

- **storage_files**: Core file metadata
- **storage_access_logs**: Access tracking and analytics
- **storage_quotas**: Quota management per user/vendor
- **storage_backups**: Backup tracking and management
- **storage_signed_urls**: Signed URL generation and validation
- **storage_health_checks**: System health monitoring
- **storage_metrics**: Performance metrics

### 3. Signed URLs for Secure Access

- **HMAC-based signing**: Uses SHA256 HMAC for URL signing
- **Time-limited access**: Configurable TTL (default 1 hour)
- **Access type control**: Separate URLs for view vs download
- **Usage tracking**: Monitor signed URL usage and abuse
- **Automatic expiration**: Cleanup expired URLs

### 4. Backup & Retention Policy

- **Keep images forever**: No automatic deletion policy
- **Automatic backups**: Daily scheduled backups
- **Compression support**: Configurable compression levels
- **Retention management**: Configurable backup retention (default 1 year)
- **Integrity checking**: Checksum validation for backups

## Key Features

### File Management
- **Multi-type support**: User, cloth, and result images
- **Automatic thumbnails**: Generate thumbnails in multiple sizes
- **Checksum validation**: SHA256 checksums for integrity
- **Metadata tracking**: Comprehensive file metadata
- **Access counting**: Track file access frequency

### Security
- **Signed URLs**: Secure, time-limited access
- **Access logging**: Complete audit trail
- **Permission checking**: Owner-based access control
- **Rate limiting**: Prevent abuse and DoS attacks

### Performance
- **Efficient indexing**: Optimized database indexes
- **Caching support**: Redis integration for performance
- **Batch operations**: Bulk file operations
- **Health monitoring**: Real-time system health checks

### Scalability
- **Quota management**: Per-user/vendor limits
- **Storage statistics**: Usage analytics and reporting
- **Cleanup utilities**: Automated maintenance tasks
- **Backup scheduling**: Configurable backup policies

## API Endpoints

### File Operations
- `POST /storage/images` - Upload image
- `GET /storage/images/:id` - Get image metadata
- `PUT /storage/images/:id` - Update image metadata
- `DELETE /storage/images/:id` - Delete image
- `GET /storage/images` - List images with filtering

### Access Control
- `GET /storage/images/:id/access` - Generate access URL
- `GET /storage/images/:id/signed-url` - Generate signed URL
- `GET /storage/signed/:encodedPath` - Validate signed URL

### Search & Analytics
- `POST /storage/images/search` - Search images
- `GET /storage/quota` - Get storage quota
- `GET /storage/stats` - Get storage statistics
- `GET /storage/health` - Get system health

### Backup & Maintenance
- `POST /storage/backup` - Create backup
- `POST /storage/restore` - Restore from backup
- `DELETE /storage/backups/cleanup` - Cleanup old backups

## Configuration

### Storage Configuration
```yaml
storage:
  basePath: "./storage"
  backupPath: "./storage/backups"
  signedURLKey: "your-secret-key"
  maxFileSize: 52428800  # 50MB
  allowedTypes:
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "image/webp"
  thumbnailSizes:
    - name: "small"
      width: 150
      height: 150
    - name: "medium"
      width: 300
      height: 300
    - name: "large"
      width: 600
      height: 600
  retentionPolicy:
    keepImagesForever: true
    maxAge: 0
    cleanupSchedule: "0 2 * * *"  # Daily at 2 AM
  backupPolicy:
    enabled: true
    backupFrequency: "daily"
    retentionDays: 365
    compressionLevel: 6
```

### Server Configuration
```yaml
server:
  host: "localhost"
  port: 8080
  baseUrl: "http://localhost:8080"
  publicPath: "/api/storage/public"
  staticPath: "/api/storage/static"
```

## Database Schema

### Core Tables

#### storage_files
```sql
CREATE TABLE storage_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    checksum TEXT NOT NULL,
    storage_type TEXT NOT NULL CHECK (storage_type IN ('user', 'cloth', 'result')),
    owner_id UUID NOT NULL,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('user', 'vendor')),
    is_public BOOLEAN NOT NULL DEFAULT false,
    is_backed_up BOOLEAN NOT NULL DEFAULT false,
    backup_path TEXT,
    thumbnail_path TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed TIMESTAMPTZ,
    access_count INTEGER NOT NULL DEFAULT 0
);
```

#### storage_quotas
```sql
CREATE TABLE storage_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    quota_type TEXT NOT NULL CHECK (quota_type IN ('user', 'vendor', 'total')),
    max_file_size BIGINT NOT NULL DEFAULT 52428800,
    max_files INTEGER NOT NULL DEFAULT 100,
    max_total_size BIGINT NOT NULL DEFAULT 5368709120,
    current_file_count INTEGER NOT NULL DEFAULT 0,
    current_total_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Usage Examples

### Upload Image
```go
// Create upload request
req := ImageUploadRequest{
    File:        fileReader,
    FileName:    "profile.jpg",
    ContentType: "image/jpeg",
    Size:        1024000,
    ImageType:   "user",
    OwnerID:     "user-123",
    IsPublic:    false,
    Tags:        []string{"profile", "avatar"},
    Metadata:    map[string]interface{}{
        "category": "profile",
        "source": "upload",
    },
}

// Upload image
response, err := imageStorage.UploadImage(ctx, req)
if err != nil {
    return err
}

// Response contains file ID, paths, and signed URL
fmt.Printf("Image uploaded: %s\n", response.ImageID)
fmt.Printf("File path: %s\n", response.FilePath)
fmt.Printf("Access URL: %s\n", response.OriginalURL)
```

### Generate Signed URL
```go
// Create access request
req := ImageAccessRequest{
    ImageID:     "image-123",
    AccessType:  "view",
    TTL:         3600, // 1 hour
    RequesterID: "user-123",
}

// Generate signed URL
response, err := imageStorage.GetImageAccess(ctx, req)
if err != nil {
    return err
}

// Use signed URL
fmt.Printf("Signed URL: %s\n", response.SignedURL)
fmt.Printf("Expires at: %s\n", response.ExpiresAt)
```

### Search Images
```go
// Create search request
req := ImageSearchRequest{
    Query:       "profile",
    ImageType:   "user",
    OwnerID:     "user-123",
    IsPublic:    &[]bool{false}[0],
    Tags:        []string{"avatar"},
    Page:        1,
    PageSize:    20,
    SortBy:      "date",
    SortOrder:   "desc",
}

// Search images
response, err := imageStorage.SearchImages(ctx, req)
if err != nil {
    return err
}

// Process results
for _, image := range response.Images {
    fmt.Printf("Found image: %s (%s)\n", image.FileName, image.ImageType)
}
```

## Security Considerations

### Access Control
- **Owner-based permissions**: Users can only access their own files
- **Public/private flags**: Control file visibility
- **Signed URL expiration**: Time-limited access
- **Rate limiting**: Prevent abuse

### Data Integrity
- **Checksum validation**: SHA256 checksums for all files
- **Backup verification**: Checksum validation for backups
- **Access logging**: Complete audit trail
- **Health monitoring**: System integrity checks

### Privacy
- **Metadata encryption**: Sensitive metadata can be encrypted
- **Access logging**: Track who accesses what and when
- **Retention policies**: Configurable data retention
- **Secure deletion**: Proper file cleanup

## Performance Optimization

### Database Optimization
- **Comprehensive indexing**: Optimized for common queries
- **Partitioning support**: For large datasets
- **Connection pooling**: Efficient database connections
- **Query optimization**: Optimized SQL queries

### File System Optimization
- **Efficient file operations**: Optimized I/O operations
- **Caching layer**: Redis integration for performance
- **Batch operations**: Bulk file operations
- **Compression**: Configurable compression levels

### Monitoring
- **Health checks**: Real-time system monitoring
- **Performance metrics**: Track system performance
- **Usage analytics**: Storage usage patterns
- **Alert system**: Proactive issue detection

## Maintenance

### Automated Tasks
- **Backup scheduling**: Daily automatic backups
- **Cleanup operations**: Remove expired data
- **Health monitoring**: Continuous system monitoring
- **Quota enforcement**: Automatic quota management

### Manual Operations
- **Backup restoration**: Restore from specific backups
- **Quota adjustments**: Modify user/vendor quotas
- **System maintenance**: Manual cleanup operations
- **Health diagnostics**: Detailed system analysis

## Troubleshooting

### Common Issues
1. **Disk space**: Monitor disk usage and cleanup old backups
2. **Quota exceeded**: Check and adjust user quotas
3. **Signed URL expiration**: Regenerate URLs as needed
4. **File corruption**: Use checksum validation to detect issues

### Monitoring Tools
- **Health endpoint**: `/storage/health` for system status
- **Statistics endpoint**: `/storage/stats` for usage metrics
- **Logs**: Comprehensive access and error logging
- **Metrics**: Performance and usage metrics

## Future Enhancements

### Planned Features
- **Cloud storage integration**: AWS S3, Google Cloud Storage
- **CDN integration**: Global content delivery
- **Advanced compression**: Better compression algorithms
- **Machine learning**: Intelligent file organization

### Scalability Improvements
- **Distributed storage**: Multi-node storage clusters
- **Load balancing**: Distribute load across servers
- **Caching layers**: Multiple caching strategies
- **Database sharding**: Horizontal database scaling

This storage architecture provides a robust, scalable, and secure foundation for the AI Stayler image management system, ensuring reliable file storage with comprehensive metadata tracking and backup capabilities.
