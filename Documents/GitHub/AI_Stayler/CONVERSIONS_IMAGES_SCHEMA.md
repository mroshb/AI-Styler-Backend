# Database Schema: Conversions & Images

## Overview

This document describes the comprehensive database schema for conversions and images in the AI Stayler application. The schema supports both user and vendor conversions with detailed tracking and analytics.

## Core Tables

### 1. Conversions Table

```sql
CREATE TABLE conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    user_image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    cloth_image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    result_image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    error_message TEXT,
    processing_time_ms INTEGER,
    conversion_type TEXT NOT NULL DEFAULT 'free' CHECK (conversion_type IN ('free', 'paid')),
    style_name TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- **Flexible Ownership**: Supports both user and vendor conversions
- **Image Relationships**: Links user image, cloth image, and result image
- **Status Tracking**: Complete conversion lifecycle management
- **Performance Metrics**: Processing time tracking
- **Metadata Support**: Flexible JSON metadata storage

### 2. Images Table (Enhanced)

```sql
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL, -- Can be user_id or vendor_id
    owner_type TEXT NOT NULL CHECK (owner_type IN ('user', 'vendor')),
    album_id UUID REFERENCES albums(id) ON DELETE SET NULL,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    original_url TEXT NOT NULL,
    thumbnail_url TEXT,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    type TEXT NOT NULL CHECK (type IN ('user', 'cloth', 'result')),
    is_public BOOLEAN NOT NULL DEFAULT false,
    is_free BOOLEAN NOT NULL DEFAULT true,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- **Flexible Ownership**: Supports both users and vendors as owners
- **Image Types**: Distinguishes between user, cloth, and result images
- **Album Organization**: Images can be organized into albums
- **Public/Private**: Visibility controls
- **Rich Metadata**: Tags and JSON metadata support
- **File Management**: Complete file information tracking

### 3. Image Usage History Table

```sql
CREATE TABLE image_usage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    conversion_id UUID REFERENCES conversions(id) ON DELETE SET NULL,
    action TEXT NOT NULL CHECK (action IN ('upload', 'view', 'download', 'delete', 'update', 'share', 'convert', 'use_in_conversion')),
    ip_address INET,
    user_agent TEXT,
    session_id TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- **Complete Tracking**: All image interactions tracked
- **Conversion Linking**: Links usage to specific conversions
- **Audit Trail**: IP address, user agent, and session tracking
- **Action Types**: Comprehensive action categorization
- **Analytics Ready**: Structured for analytics and reporting

## Supporting Tables

### 4. Albums Table (Enhanced)

```sql
CREATE TABLE albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('user', 'vendor')),
    name TEXT NOT NULL,
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false,
    image_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- **Flexible Ownership**: Both users and vendors can create albums
- **Automatic Counting**: Image count automatically maintained
- **Public/Private**: Album visibility controls
- **Unique Names**: Album names unique per owner

### 5. Conversion Jobs Table

```sql
CREATE TABLE conversion_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversion_id UUID NOT NULL REFERENCES conversions(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'cancelled')),
    worker_id TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- **Job Queue Management**: Background job processing
- **Retry Logic**: Built-in retry mechanism
- **Priority Support**: Job prioritization
- **Worker Tracking**: Worker assignment and monitoring

### 6. Conversion Metrics Table

```sql
CREATE TABLE conversion_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversion_id UUID NOT NULL REFERENCES conversions(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    processing_time_ms INTEGER NOT NULL,
    input_file_size_bytes BIGINT,
    output_file_size_bytes BIGINT,
    success BOOLEAN NOT NULL,
    error_type TEXT,
    ai_model_version TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- **Performance Analytics**: Detailed performance metrics
- **File Size Tracking**: Input/output file size monitoring
- **Error Classification**: Error type categorization
- **AI Model Tracking**: Model version tracking

## Key Features

### 1. Flexible Ownership Model

The schema supports both users and vendors as owners of images and conversions:

- **Users**: Can upload user images and create conversions
- **Vendors**: Can upload cloth images and create conversions
- **Shared Resources**: Both can access public cloth images

### 2. Comprehensive Tracking

- **Image Usage**: Every image interaction is tracked
- **Conversion Lifecycle**: Complete conversion process tracking
- **Performance Metrics**: Detailed performance analytics
- **Audit Trail**: Complete audit trail for compliance

### 3. Scalability Features

- **Efficient Indexing**: Strategic indexes for common queries
- **Partitioning Ready**: Schema designed for future partitioning
- **JSON Support**: Flexible metadata storage
- **Array Support**: Tag arrays for efficient searching

### 4. Data Integrity

- **Foreign Key Constraints**: Proper referential integrity
- **Check Constraints**: Data validation at database level
- **Unique Constraints**: Prevents duplicate data
- **Cascade Deletes**: Proper cleanup on deletion

## Utility Functions

### 1. create_conversion()

Creates a new conversion with proper validation:

```sql
SELECT create_conversion(
    user_id := 'user-uuid',
    vendor_id := NULL,
    user_image_id := 'image-uuid-1',
    cloth_image_id := 'image-uuid-2',
    conversion_type := 'free',
    style_name := 'casual'
);
```

### 2. update_conversion_status()

Updates conversion status and records metrics:

```sql
SELECT update_conversion_status(
    conversion_id := 'conversion-uuid',
    status := 'completed',
    result_image_id := 'result-image-uuid',
    processing_time_ms := 5000
);
```

### 3. record_image_usage()

Records image usage for analytics:

```sql
SELECT record_image_usage(
    image_id := 'image-uuid',
    user_id := 'user-uuid',
    vendor_id := NULL,
    action := 'view',
    ip_address := '192.168.1.1',
    user_agent := 'Mozilla/5.0...'
);
```

### 4. get_conversion_stats()

Retrieves conversion statistics:

```sql
SELECT * FROM get_conversion_stats(
    p_user_id := 'user-uuid',
    p_date_from := '2025-01-01',
    p_date_to := '2025-01-31'
);
```

### 5. get_image_stats()

Retrieves image statistics:

```sql
SELECT * FROM get_image_stats(
    p_owner_id := 'user-uuid',
    p_owner_type := 'user',
    p_image_type := 'user'
);
```

## Indexes and Performance

### Primary Indexes

- **Conversions**: user_id, vendor_id, status, created_at
- **Images**: owner_id, owner_type, type, is_public
- **Usage History**: image_id, user_id, vendor_id, action, created_at

### Composite Indexes

- **Conversions**: (user_id, status), (vendor_id, status), (status, created_at)
- **Images**: (owner_type, owner_id), (type, is_public), (owner_id, owner_type, album_id)
- **Usage History**: (image_id, action), (user_id, action), (action, created_at)

### Specialized Indexes

- **GIN Indexes**: tags, metadata for efficient JSON/array queries
- **Partial Indexes**: Active records, public content
- **Covering Indexes**: Frequently accessed columns

## Migration Strategy

The migration file `0010_conversions_images_schema.sql` includes:

1. **Table Creation**: All tables with proper constraints
2. **Index Creation**: Performance-optimized indexes
3. **Trigger Setup**: Automatic timestamp and count updates
4. **Function Creation**: Utility functions for common operations
5. **Data Validation**: Check constraints and foreign keys

## Usage Examples

### Creating a User Conversion

```sql
-- 1. Upload user image
INSERT INTO images (owner_id, owner_type, file_path, type, ...)
VALUES ('user-uuid', 'user', '/uploads/user.jpg', 'user', ...);

-- 2. Create conversion
SELECT create_conversion(
    user_id := 'user-uuid',
    user_image_id := 'user-image-uuid',
    cloth_image_id := 'cloth-image-uuid',
    conversion_type := 'free'
);
```

### Tracking Image Usage

```sql
-- Record image view
SELECT record_image_usage(
    image_id := 'image-uuid',
    user_id := 'user-uuid',
    action := 'view',
    ip_address := '192.168.1.1'
);
```

### Getting Analytics

```sql
-- Get user conversion stats
SELECT * FROM get_conversion_stats(p_user_id := 'user-uuid');

-- Get image usage analytics
SELECT 
    action,
    COUNT(*) as count,
    DATE(created_at) as date
FROM image_usage_history 
WHERE image_id = 'image-uuid'
GROUP BY action, DATE(created_at)
ORDER BY date DESC;
```

This schema provides a robust foundation for the AI Stayler application's conversion and image management needs, with comprehensive tracking, analytics, and scalability features.
