-- Comprehensive Database Schema Migration
-- Unified schema for AI Stayler application with separate users & vendors tables
-- and shared tables for conversions, images, albums, payments, and rate limits

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Utility function for updated_at triggers
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- CORE TABLES: USERS & VENDORS
-- ============================================================================

-- Users table - Core user information
CREATE TABLE IF NOT EXISTS users (
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

-- Vendors table - Separate vendor information
CREATE TABLE IF NOT EXISTS vendors (
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

-- ============================================================================
-- SHARED TABLES
-- ============================================================================

-- User conversions table - Track all conversion activities
CREATE TABLE IF NOT EXISTS user_conversions (
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

-- Images table - Comprehensive image management for all types
CREATE TABLE IF NOT EXISTS images (
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
    
    -- Ensure at least one of user_id or vendor_id is set based on type
    CONSTRAINT chk_image_ownership CHECK (
        (type = 'user' AND user_id IS NOT NULL AND vendor_id IS NULL) OR
        (type = 'vendor' AND vendor_id IS NOT NULL AND user_id IS NULL) OR
        (type = 'result' AND (user_id IS NOT NULL OR vendor_id IS NOT NULL))
    )
);

-- Albums table - Vendor image albums/categories
CREATE TABLE IF NOT EXISTS albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false,
    image_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Payments table - Payment transactions
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES payment_plans(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL, -- Amount in cents (Rials)
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
    
    -- Ensure only one of user_id or vendor_id is set
    CONSTRAINT chk_payment_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);

-- Rate limits table - Track rate limiting
CREATE TABLE IF NOT EXISTS rate_limits (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    count INT NOT NULL,
    CONSTRAINT rate_limits_unique_key_window UNIQUE (key, window_start)
);

-- ============================================================================
-- SUPPORTING TABLES
-- ============================================================================

-- Payment plans table - Available subscription plans
CREATE TABLE IF NOT EXISTS payment_plans (
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

-- Sessions table - Store hashed refresh tokens
CREATE TABLE IF NOT EXISTS sessions (
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
    
    -- Ensure only one of user_id or vendor_id is set
    CONSTRAINT chk_session_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);

-- OTPs table - Store hashed codes
CREATE TABLE IF NOT EXISTS otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone TEXT NOT NULL,
    code TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('phone_verify','password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('system','user','vendor','admin')),
    action TEXT NOT NULL,
    resource TEXT,
    resource_id TEXT,
    metadata JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure only one of user_id or vendor_id is set
    CONSTRAINT chk_audit_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL) OR
        (user_id IS NULL AND vendor_id IS NULL)
    )
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Users table indexes
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_plan_id ON users(plan_id);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_is_phone_verified ON users(is_phone_verified);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at);

-- Vendors table indexes
CREATE INDEX IF NOT EXISTS idx_vendors_phone ON vendors(phone);
CREATE INDEX IF NOT EXISTS idx_vendors_plan_id ON vendors(plan_id);
CREATE INDEX IF NOT EXISTS idx_vendors_business_name ON vendors(business_name);
CREATE INDEX IF NOT EXISTS idx_vendors_is_verified ON vendors(is_verified);
CREATE INDEX IF NOT EXISTS idx_vendors_is_active ON vendors(is_active);
CREATE INDEX IF NOT EXISTS idx_vendors_created_at ON vendors(created_at);
CREATE INDEX IF NOT EXISTS idx_vendors_last_login_at ON vendors(last_login_at);

-- User conversions table indexes
CREATE INDEX IF NOT EXISTS idx_user_conversions_user_id ON user_conversions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_conversions_created_at ON user_conversions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_conversions_status ON user_conversions(status);
CREATE INDEX IF NOT EXISTS idx_user_conversions_type ON user_conversions(conversion_type);

-- Images table indexes
CREATE INDEX IF NOT EXISTS idx_images_user_id ON images(user_id);
CREATE INDEX IF NOT EXISTS idx_images_vendor_id ON images(vendor_id);
CREATE INDEX IF NOT EXISTS idx_images_type ON images(type);
CREATE INDEX IF NOT EXISTS idx_images_is_public ON images(is_public);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_mime_type ON images(mime_type);
CREATE INDEX IF NOT EXISTS idx_images_file_size ON images(file_size);
CREATE INDEX IF NOT EXISTS idx_images_tags ON images USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_images_metadata ON images USING GIN(metadata);
CREATE INDEX IF NOT EXISTS idx_images_user_type ON images(user_id, type);
CREATE INDEX IF NOT EXISTS idx_images_vendor_type ON images(vendor_id, type);
CREATE INDEX IF NOT EXISTS idx_images_type_public ON images(type, is_public);

-- Albums table indexes
CREATE INDEX IF NOT EXISTS idx_albums_vendor_id ON albums(vendor_id);
CREATE INDEX IF NOT EXISTS idx_albums_is_public ON albums(is_public);
CREATE INDEX IF NOT EXISTS idx_albums_created_at ON albums(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_albums_name ON albums(name);

-- Payments table indexes
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_vendor_id ON payments(vendor_id);
CREATE INDEX IF NOT EXISTS idx_payments_plan_id ON payments(plan_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_gateway_track_id ON payments(gateway_track_id);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_expires_at ON payments(expires_at);

-- Rate limits table indexes
CREATE INDEX IF NOT EXISTS idx_rate_limits_key ON rate_limits(key);
CREATE INDEX IF NOT EXISTS idx_rate_limits_window_start ON rate_limits(window_start);

-- Payment plans table indexes
CREATE INDEX IF NOT EXISTS idx_payment_plans_name ON payment_plans(name);
CREATE INDEX IF NOT EXISTS idx_payment_plans_is_active ON payment_plans(is_active);
CREATE INDEX IF NOT EXISTS idx_payment_plans_price ON payment_plans(price_per_month_cents);

-- Sessions table indexes
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_vendor_id ON sessions(vendor_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_active_user ON sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_active_vendor ON sessions(vendor_id) WHERE revoked_at IS NULL;

-- OTPs table indexes
CREATE INDEX IF NOT EXISTS idx_otps_phone_created_at ON otps(phone, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_otps_expires_at ON otps(expires_at);
CREATE INDEX IF NOT EXISTS idx_otps_phone_unconsumed ON otps(phone) WHERE consumed_at IS NULL;

-- Audit logs table indexes
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_vendor_id ON audit_logs(vendor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_type ON audit_logs(actor_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_metadata_gin ON audit_logs USING GIN (metadata);

-- ============================================================================
-- TRIGGERS FOR UPDATED_AT
-- ============================================================================

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_vendors_updated_at
BEFORE UPDATE ON vendors
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_images_updated_at
BEFORE UPDATE ON images
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_albums_updated_at
BEFORE UPDATE ON albums
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_payment_plans_updated_at
BEFORE UPDATE ON payment_plans
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- DEFAULT DATA
-- ============================================================================

-- Insert default payment plans
INSERT INTO payment_plans (id, name, display_name, description, price_per_month_cents, monthly_conversions_limit, monthly_images_limit, features, is_active) VALUES
    ('00000000-0000-0000-0000-000000000001', 'free', 'Free Plan', 'Basic free plan with limited conversions', 0, 2, 5, ARRAY['2 free conversions per month', '5 free images', 'Basic support'], true),
    ('00000000-0000-0000-0000-000000000002', 'basic', 'Basic Plan', 'Basic paid plan with more conversions', 50000, 20, 50, ARRAY['20 conversions per month', '50 images', 'Email support', 'Priority processing'], true),
    ('00000000-0000-0000-0000-000000000003', 'advanced', 'Advanced Plan', 'Advanced plan with unlimited conversions', 150000, 100, 200, ARRAY['100 conversions per month', '200 images', 'Priority support', 'Fast processing', 'Advanced features'], true),
    ('00000000-0000-0000-0000-000000000004', 'vendor_free', 'Vendor Free Plan', 'Free plan for vendors', 0, 0, 10, ARRAY['10 free images', 'Basic gallery'], true),
    ('00000000-0000-0000-0000-000000000005', 'vendor_pro', 'Vendor Pro Plan', 'Professional plan for vendors', 100000, 0, 500, ARRAY['500 images', 'Advanced gallery', 'Analytics', 'Priority support'], true)
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================

-- Function to get user quota status
CREATE OR REPLACE FUNCTION get_user_quota_status(p_user_id UUID)
RETURNS TABLE (
    free_conversions_remaining INTEGER,
    paid_conversions_remaining INTEGER,
    total_conversions_remaining INTEGER,
    plan_name TEXT,
    monthly_limit INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        GREATEST(0, u.free_quota_remaining) as free_conversions_remaining,
        GREATEST(0, COALESCE(pp.monthly_conversions_limit, 0)) as paid_conversions_remaining,
        GREATEST(0, u.free_quota_remaining) + GREATEST(0, COALESCE(pp.monthly_conversions_limit, 0)) as total_conversions_remaining,
        COALESCE(pp.name, 'free') as plan_name,
        COALESCE(pp.monthly_conversions_limit, 0) as monthly_limit
    FROM users u
    LEFT JOIN payment_plans pp ON u.plan_id = pp.id
    WHERE u.id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get vendor quota status
CREATE OR REPLACE FUNCTION get_vendor_quota_status(p_vendor_id UUID)
RETURNS TABLE (
    free_images_remaining INTEGER,
    paid_images_remaining INTEGER,
    total_images_remaining INTEGER,
    plan_name TEXT,
    monthly_limit INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        GREATEST(0, v.free_gallery_remaining) as free_images_remaining,
        GREATEST(0, COALESCE(pp.monthly_images_limit, 0)) as paid_images_remaining,
        GREATEST(0, v.free_gallery_remaining) + GREATEST(0, COALESCE(pp.monthly_images_limit, 0)) as total_images_remaining,
        COALESCE(pp.name, 'vendor_free') as plan_name,
        COALESCE(pp.monthly_images_limit, 0) as monthly_limit
    FROM vendors v
    LEFT JOIN payment_plans pp ON v.plan_id = pp.id
    WHERE v.id = p_vendor_id;
END;
$$ LANGUAGE plpgsql;

-- Function to check if user can perform conversion
CREATE OR REPLACE FUNCTION can_user_convert(p_user_id UUID, p_conversion_type TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    quota_status RECORD;
BEGIN
    SELECT * INTO quota_status FROM get_user_quota_status(p_user_id);
    
    IF p_conversion_type = 'free' THEN
        RETURN quota_status.free_conversions_remaining > 0;
    ELSIF p_conversion_type = 'paid' THEN
        RETURN quota_status.paid_conversions_remaining > 0;
    ELSE
        RETURN FALSE;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to check if vendor can upload image
CREATE OR REPLACE FUNCTION can_vendor_upload_image(p_vendor_id UUID, p_is_free BOOLEAN)
RETURNS BOOLEAN AS $$
DECLARE
    quota_status RECORD;
BEGIN
    SELECT * INTO quota_status FROM get_vendor_quota_status(p_vendor_id);
    
    IF p_is_free THEN
        RETURN quota_status.free_images_remaining > 0;
    ELSE
        RETURN quota_status.paid_images_remaining > 0;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to record a conversion
CREATE OR REPLACE FUNCTION record_conversion(
    p_user_id UUID,
    p_conversion_type TEXT,
    p_input_file_url TEXT,
    p_style_name TEXT
) RETURNS UUID AS $$
DECLARE
    conversion_id UUID;
BEGIN
    -- Check if user can convert
    IF NOT can_user_convert(p_user_id, p_conversion_type) THEN
        RAISE EXCEPTION 'User quota exceeded for conversion type: %', p_conversion_type;
    END IF;
    
    -- Create conversion record
    INSERT INTO user_conversions (user_id, conversion_type, input_file_url, style_name)
    VALUES (p_user_id, p_conversion_type, p_input_file_url, p_style_name)
    RETURNING id INTO conversion_id;
    
    -- Update user's free conversion count if it's a free conversion
    IF p_conversion_type = 'free' THEN
        UPDATE users 
        SET free_quota_remaining = GREATEST(0, free_quota_remaining - 1)
        WHERE id = p_user_id;
    END IF;
    
    RETURN conversion_id;
END;
$$ LANGUAGE plpgsql;

-- Function to record an image upload
CREATE OR REPLACE FUNCTION record_image_upload(
    p_user_id UUID,
    p_vendor_id UUID,
    p_type VARCHAR(20),
    p_file_name TEXT,
    p_original_url TEXT,
    p_thumbnail_url TEXT,
    p_file_size BIGINT,
    p_mime_type TEXT,
    p_width INTEGER,
    p_height INTEGER,
    p_is_public BOOLEAN,
    p_tags TEXT[],
    p_metadata JSONB
) RETURNS UUID AS $$
DECLARE
    image_id UUID;
BEGIN
    -- Check quota based on type
    IF p_type = 'user' AND p_user_id IS NOT NULL THEN
        -- For now, users can upload unlimited images (can be restricted later)
        NULL; -- Allow upload
    ELSIF p_type = 'vendor' AND p_vendor_id IS NOT NULL THEN
        IF NOT can_vendor_upload_image(p_vendor_id, true) THEN
            RAISE EXCEPTION 'Vendor image quota exceeded';
        END IF;
    END IF;
    
    -- Create image record
    INSERT INTO images (
        user_id, vendor_id, type, file_name, original_url, thumbnail_url,
        file_size, mime_type, width, height, is_public, tags, metadata
    )
    VALUES (
        p_user_id, p_vendor_id, p_type, p_file_name, p_original_url, p_thumbnail_url,
        p_file_size, p_mime_type, p_width, p_height, p_is_public, p_tags, p_metadata
    )
    RETURNING id INTO image_id;
    
    -- Update vendor's free image count if it's a vendor image
    IF p_type = 'vendor' AND p_vendor_id IS NOT NULL THEN
        UPDATE vendors 
        SET free_gallery_remaining = GREATEST(0, free_gallery_remaining - 1)
        WHERE id = p_vendor_id;
    END IF;
    
    RETURN image_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get system statistics
CREATE OR REPLACE FUNCTION get_system_stats()
RETURNS TABLE (
    total_users INTEGER,
    active_users INTEGER,
    total_vendors INTEGER,
    active_vendors INTEGER,
    total_conversions INTEGER,
    total_images INTEGER,
    total_payments BIGINT,
    total_revenue BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*)::INTEGER FROM users) as total_users,
        (SELECT COUNT(*)::INTEGER FROM users WHERE is_active = true) as active_users,
        (SELECT COUNT(*)::INTEGER FROM vendors) as total_vendors,
        (SELECT COUNT(*)::INTEGER FROM vendors WHERE is_active = true) as active_vendors,
        (SELECT COUNT(*)::INTEGER FROM user_conversions) as total_conversions,
        (SELECT COUNT(*)::INTEGER FROM images) as total_images,
        (SELECT COUNT(*) FROM payments) as total_payments,
        (SELECT COALESCE(SUM(amount) FILTER (WHERE status = 'completed'), 0) FROM payments) as total_revenue;
END;
$$ LANGUAGE plpgsql;

COMMIT;
