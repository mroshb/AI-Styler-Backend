# Database Schema Design - AI Stayler

## Overview

This document describes the comprehensive PostgreSQL database schema for the AI Stayler application, designed with separate tables for users and vendors, along with shared tables for conversions, images, albums, payments, and rate limits.

## Schema Design Principles

1. **Separation of Concerns**: Users and vendors have separate tables with their own authentication and quota systems
2. **Shared Resources**: Common functionality like images, conversions, and payments are shared between users and vendors
3. **Scalability**: Proper indexing and constraints for performance at scale
4. **Data Integrity**: Foreign key constraints and check constraints ensure data consistency
5. **Audit Trail**: Comprehensive logging of all system activities

## Core Tables

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    free_quota_remaining INTEGER NOT NULL DEFAULT 2,
    plan_id UUID REFERENCES payment_plans(id) ON DELETE SET NULL,
    name TEXT,
    avatar_url TEXT,
    bio TEXT,
    is_phone_verified BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- Unique phone number authentication
- Free quota tracking (default: 2 conversions)
- Optional subscription plan
- Profile information (name, avatar, bio)
- Phone verification status
- Account status management

### Vendors Table
```sql
CREATE TABLE vendors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    free_gallery_remaining INTEGER NOT NULL DEFAULT 10,
    plan_id UUID REFERENCES payment_plans(id) ON DELETE SET NULL,
    profile_info JSONB NOT NULL DEFAULT '{}',
    business_name TEXT NOT NULL,
    avatar_url TEXT,
    bio TEXT,
    contact_info JSONB NOT NULL DEFAULT '{}',
    social_links JSONB NOT NULL DEFAULT '{}',
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Features:**
- Separate authentication system from users
- Gallery quota tracking (default: 10 images)
- Business profile information
- Contact and social media links
- Verification status for business accounts

## Shared Tables

### User Conversions
```sql
CREATE TABLE user_conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversion_type TEXT NOT NULL CHECK (conversion_type IN ('free', 'paid')),
    input_file_url TEXT NOT NULL,
    output_file_url TEXT,
    style_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    error_message TEXT,
    processing_time_ms INTEGER,
    file_size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
```

**Purpose:** Track all image conversion requests and their status

### Images
```sql
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('user', 'vendor', 'result')),
    file_name TEXT NOT NULL,
    original_url TEXT NOT NULL,
    thumbnail_url TEXT,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    is_public BOOLEAN NOT NULL DEFAULT false,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_image_ownership CHECK (
        (type = 'user' AND user_id IS NOT NULL AND vendor_id IS NULL) OR
        (type = 'vendor' AND vendor_id IS NOT NULL AND user_id IS NULL) OR
        (type = 'result' AND (user_id IS NOT NULL OR vendor_id IS NOT NULL))
    )
);
```

**Purpose:** Unified image storage for users, vendors, and conversion results

### Albums
```sql
CREATE TABLE albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false,
    image_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Purpose:** Organize vendor images into collections/categories

### Payments
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES payment_plans(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'IRR',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed', 'cancelled', 'expired')),
    payment_method TEXT NOT NULL DEFAULT 'zarinpal',
    gateway TEXT NOT NULL DEFAULT 'zarinpal',
    gateway_track_id TEXT,
    gateway_ref_number TEXT,
    gateway_card_number TEXT,
    description TEXT,
    callback_url TEXT NOT NULL,
    return_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    
    CONSTRAINT chk_payment_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);
```

**Purpose:** Handle payment transactions for both users and vendors

### Rate Limits
```sql
CREATE TABLE rate_limits (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    count INT NOT NULL,
    CONSTRAINT rate_limits_unique_key_window UNIQUE (key, window_start)
);
```

**Purpose:** Track API rate limiting (mirror of cache-based enforcement)

## Supporting Tables

### Payment Plans
```sql
CREATE TABLE payment_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    price_per_month_cents BIGINT NOT NULL DEFAULT 0,
    monthly_conversions_limit INTEGER NOT NULL DEFAULT 0,
    monthly_images_limit INTEGER NOT NULL DEFAULT 0,
    features TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Default Plans:**
- **Free Plan**: 2 conversions, 5 images
- **Basic Plan**: 20 conversions, 50 images (50,000 Rials/month)
- **Advanced Plan**: 100 conversions, 200 images (150,000 Rials/month)
- **Vendor Free Plan**: 10 images
- **Vendor Pro Plan**: 500 images (100,000 Rials/month)

### Sessions
```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    user_agent TEXT,
    ip INET,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    rotation_counter INT NOT NULL DEFAULT 0,
    
    CONSTRAINT chk_session_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);
```

**Purpose:** Manage authentication sessions for both users and vendors

### OTPs
```sql
CREATE TABLE otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone TEXT NOT NULL,
    code TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('phone_verify','password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Purpose:** Store one-time passwords for phone verification and password reset

### Audit Logs
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('system','user','vendor','admin')),
    action TEXT NOT NULL,
    resource TEXT,
    resource_id TEXT,
    metadata JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_audit_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL) OR
        (user_id IS NULL AND vendor_id IS NULL)
    )
);
```

**Purpose:** Comprehensive audit trail for all system activities

## Key Features

### 1. Quota Management
- **Users**: Free quota (2 conversions) + paid plan quotas
- **Vendors**: Free gallery quota (10 images) + paid plan quotas
- Automatic quota tracking and enforcement

### 2. Flexible Image Management
- Unified image table supporting users, vendors, and conversion results
- Ownership constraints ensure data integrity
- Support for metadata, tags, and public/private visibility

### 3. Payment System
- Support for both user and vendor payments
- Integration with ZarinPal gateway
- Comprehensive payment tracking and history

### 4. Security Features
- Separate authentication systems for users and vendors
- Session management with token rotation
- OTP-based phone verification
- Comprehensive audit logging

### 5. Performance Optimizations
- Strategic indexing for common query patterns
- Composite indexes for multi-column queries
- GIN indexes for JSONB and array fields
- Proper foreign key constraints

## Utility Functions

The schema includes several utility functions:

- `get_user_quota_status(user_id)` - Get user's current quota status
- `get_vendor_quota_status(vendor_id)` - Get vendor's current quota status
- `can_user_convert(user_id, type)` - Check if user can perform conversion
- `can_vendor_upload_image(vendor_id, is_free)` - Check if vendor can upload image
- `record_conversion(...)` - Record a new conversion with quota enforcement
- `record_image_upload(...)` - Record image upload with quota enforcement
- `get_system_stats()` - Get comprehensive system statistics

## Migration Strategy

The comprehensive schema migration (`0009_comprehensive_schema.sql`) includes:

1. **Core Tables**: Users, vendors, and all shared tables
2. **Indexes**: Performance-optimized indexes for all tables
3. **Constraints**: Data integrity constraints and check constraints
4. **Triggers**: Automatic `updated_at` timestamp management
5. **Default Data**: Pre-configured payment plans
6. **Utility Functions**: Helper functions for common operations

This schema provides a solid foundation for the AI Stayler application with clear separation between users and vendors while sharing common functionality efficiently.
