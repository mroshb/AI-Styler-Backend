-- Database Schema Migration: Conversions & Images
-- Comprehensive schema for AI Stayler conversion and image management
-- Extends existing schema with proper relationships and tracking

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- CONVERSIONS TABLE
-- ============================================================================

-- conversions table - Track all image conversion requests
-- Note: Table already exists from migration 0005, so we only add missing columns
CREATE TABLE IF NOT EXISTS conversions (
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure only one of user_id or vendor_id is set
    CONSTRAINT chk_conversion_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    ),
    
    -- Ensure different image types for input
    CONSTRAINT chk_conversion_images CHECK (
        user_image_id != cloth_image_id
    )
);

-- Add missing columns to existing conversions table
DO $$ 
BEGIN
    -- Add vendor_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'conversions' AND column_name = 'vendor_id') THEN
        ALTER TABLE conversions ADD COLUMN vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE;
        -- Make user_id nullable if it's currently NOT NULL
        ALTER TABLE conversions ALTER COLUMN user_id DROP NOT NULL;
    END IF;
    
    -- Add conversion_type if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'conversions' AND column_name = 'conversion_type') THEN
        ALTER TABLE conversions ADD COLUMN conversion_type TEXT NOT NULL DEFAULT 'free';
        ALTER TABLE conversions ADD CONSTRAINT chk_conversion_type CHECK (conversion_type IN ('free', 'paid'));
    END IF;
    
    -- Add style_name if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'conversions' AND column_name = 'style_name') THEN
        ALTER TABLE conversions ADD COLUMN style_name TEXT;
    END IF;
    
    -- Add metadata if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'conversions' AND column_name = 'metadata') THEN
        ALTER TABLE conversions ADD COLUMN metadata JSONB DEFAULT '{}';
    END IF;
    
    -- Update status constraint to include 'cancelled'
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname LIKE '%status%' AND conrelid = 'conversions'::regclass) THEN
        -- Drop old constraint and add new one
        ALTER TABLE conversions DROP CONSTRAINT IF EXISTS conversions_status_check;
        ALTER TABLE conversions ADD CONSTRAINT conversions_status_check 
            CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled'));
    END IF;
    
    -- Add ownership constraint if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversion_ownership') THEN
        ALTER TABLE conversions ADD CONSTRAINT chk_conversion_ownership CHECK (
            (user_id IS NOT NULL AND vendor_id IS NULL) OR
            (vendor_id IS NOT NULL AND user_id IS NULL)
        );
    END IF;
    
    -- Add images constraint if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversion_images') THEN
        ALTER TABLE conversions ADD CONSTRAINT chk_conversion_images CHECK (
            user_image_id != cloth_image_id
        );
    END IF;
END $$;

-- ============================================================================
-- IMAGES TABLE (Enhanced)
-- ============================================================================

-- images table - Comprehensive image management
CREATE TABLE IF NOT EXISTS images (
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure owner_id references correct table based on owner_type
    CONSTRAINT chk_image_owner_user CHECK (
        (owner_type = 'user' AND owner_id IN (SELECT id FROM users)) OR
        (owner_type = 'vendor' AND owner_id IN (SELECT id FROM vendors))
    )
);

-- ============================================================================
-- IMAGE USAGE HISTORY TABLE
-- ============================================================================

-- image_usage_history table - Track all image usage and interactions
CREATE TABLE IF NOT EXISTS image_usage_history (
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure at least one of user_id or vendor_id is set
    CONSTRAINT chk_usage_history_actor CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);

-- Add missing columns to existing image_usage_history table
DO $$ 
BEGIN
    -- Add vendor_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'image_usage_history' AND column_name = 'vendor_id') THEN
        ALTER TABLE image_usage_history ADD COLUMN vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL;
    END IF;
    
    -- Add conversion_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'image_usage_history' AND column_name = 'conversion_id') THEN
        ALTER TABLE image_usage_history ADD COLUMN conversion_id UUID REFERENCES conversions(id) ON DELETE SET NULL;
    END IF;
    
    -- Add session_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'image_usage_history' AND column_name = 'session_id') THEN
        ALTER TABLE image_usage_history ADD COLUMN session_id TEXT;
    END IF;
    
    -- Update action constraint to include new actions if it doesn't match
    -- Note: We'll drop and recreate the constraint if it exists
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname LIKE '%action%' AND conrelid = 'image_usage_history'::regclass) THEN
        ALTER TABLE image_usage_history DROP CONSTRAINT IF EXISTS image_usage_history_action_check;
        ALTER TABLE image_usage_history ADD CONSTRAINT image_usage_history_action_check 
            CHECK (action IN ('upload', 'view', 'download', 'delete', 'update', 'share', 'convert', 'use_in_conversion'));
    END IF;
    
    -- Add ownership constraint if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_usage_history_actor') THEN
        ALTER TABLE image_usage_history ADD CONSTRAINT chk_usage_history_actor CHECK (
            (user_id IS NOT NULL AND vendor_id IS NULL) OR
            (vendor_id IS NOT NULL AND user_id IS NULL)
        );
    END IF;
END $$;

-- ============================================================================
-- ALBUMS TABLE (Enhanced for both users and vendors)
-- ============================================================================

-- albums table - Image organization for both users and vendors
CREATE TABLE IF NOT EXISTS albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('user', 'vendor')),
    name TEXT NOT NULL,
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false,
    image_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure owner_id references correct table based on owner_type
    CONSTRAINT chk_album_owner_user CHECK (
        (owner_type = 'user' AND owner_id IN (SELECT id FROM users)) OR
        (owner_type = 'vendor' AND owner_id IN (SELECT id FROM vendors))
    ),
    
    -- Unique album names per owner
    CONSTRAINT chk_album_name_unique UNIQUE (owner_id, owner_type, name)
);

-- ============================================================================
-- CONVERSION JOBS TABLE
-- ============================================================================

-- conversion_jobs table - Track background processing jobs
CREATE TABLE IF NOT EXISTS conversion_jobs (
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

-- ============================================================================
-- CONVERSION METRICS TABLE
-- ============================================================================

-- conversion_metrics table - Track conversion performance metrics
-- Note: Table already exists from migration 0005, so we only add missing columns
CREATE TABLE IF NOT EXISTS conversion_metrics (
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure at least one of user_id or vendor_id is set
    CONSTRAINT chk_metrics_actor CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);

-- Add missing columns to existing conversion_metrics table
DO $$ 
BEGIN
    -- Add vendor_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'conversion_metrics' AND column_name = 'vendor_id') THEN
        ALTER TABLE conversion_metrics ADD COLUMN vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE;
        -- Make user_id nullable if it's currently NOT NULL
        ALTER TABLE conversion_metrics ALTER COLUMN user_id DROP NOT NULL;
    END IF;
    
    -- Add ai_model_version if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'conversion_metrics' AND column_name = 'ai_model_version') THEN
        ALTER TABLE conversion_metrics ADD COLUMN ai_model_version TEXT;
    END IF;
    
    -- Add ownership constraint if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_metrics_actor') THEN
        ALTER TABLE conversion_metrics ADD CONSTRAINT chk_metrics_actor CHECK (
            (user_id IS NOT NULL AND vendor_id IS NULL) OR
            (vendor_id IS NOT NULL AND user_id IS NULL)
        );
    END IF;
END $$;

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Conversions table indexes
CREATE INDEX IF NOT EXISTS idx_conversions_user_id ON conversions(user_id);
-- Only create vendor_id index if column exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'conversions' AND column_name = 'vendor_id') THEN
CREATE INDEX IF NOT EXISTS idx_conversions_vendor_id ON conversions(vendor_id);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_conversions_status ON conversions(status);
CREATE INDEX IF NOT EXISTS idx_conversions_created_at ON conversions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversions_user_image_id ON conversions(user_image_id);
CREATE INDEX IF NOT EXISTS idx_conversions_cloth_image_id ON conversions(cloth_image_id);
CREATE INDEX IF NOT EXISTS idx_conversions_result_image_id ON conversions(result_image_id);
CREATE INDEX IF NOT EXISTS idx_conversions_conversion_type ON conversions(conversion_type);
CREATE INDEX IF NOT EXISTS idx_conversions_style_name ON conversions(style_name);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_conversions_user_status ON conversions(user_id, status);
-- Only create vendor_status index if vendor_id exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'conversions' AND column_name = 'vendor_id') THEN
CREATE INDEX IF NOT EXISTS idx_conversions_vendor_status ON conversions(vendor_id, status);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_conversions_status_created ON conversions(status, created_at DESC);

-- Images table indexes
-- Note: Existing images table uses user_id/vendor_id, not owner_id/owner_type
-- Skip owner_id/owner_type indexes as columns don't exist
-- CREATE INDEX IF NOT EXISTS idx_images_owner_id ON images(owner_id); -- Skipped: column doesn't exist
-- CREATE INDEX IF NOT EXISTS idx_images_owner_type ON images(owner_type); -- Skipped: column doesn't exist
CREATE INDEX IF NOT EXISTS idx_images_album_id ON images(album_id);
CREATE INDEX IF NOT EXISTS idx_images_type ON images(type);
CREATE INDEX IF NOT EXISTS idx_images_is_public ON images(is_public);
CREATE INDEX IF NOT EXISTS idx_images_is_free ON images(is_free);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_mime_type ON images(mime_type);
CREATE INDEX IF NOT EXISTS idx_images_file_size ON images(file_size);
CREATE INDEX IF NOT EXISTS idx_images_tags ON images USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_images_metadata ON images USING GIN(metadata);

-- Composite indexes for images
-- Note: Existing images table uses user_id/vendor_id, not owner_id/owner_type
-- CREATE INDEX IF NOT EXISTS idx_images_owner_type_id ON images(owner_type, owner_id); -- Skipped: columns don't exist
CREATE INDEX IF NOT EXISTS idx_images_type_public ON images(type, is_public);
-- CREATE INDEX IF NOT EXISTS idx_images_owner_album ON images(owner_id, owner_type, album_id); -- Skipped: columns don't exist

-- Image usage history indexes
CREATE INDEX IF NOT EXISTS idx_image_usage_history_image_id ON image_usage_history(image_id);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_user_id ON image_usage_history(user_id);
-- Only create vendor_id index if column exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'image_usage_history' AND column_name = 'vendor_id') THEN
CREATE INDEX IF NOT EXISTS idx_image_usage_history_vendor_id ON image_usage_history(vendor_id);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_image_usage_history_conversion_id ON image_usage_history(conversion_id);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_action ON image_usage_history(action);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_created_at ON image_usage_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_ip_address ON image_usage_history(ip_address);

-- Composite indexes for usage history
CREATE INDEX IF NOT EXISTS idx_image_usage_history_image_action ON image_usage_history(image_id, action);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_user_action ON image_usage_history(user_id, action);
-- Only create vendor_action index if vendor_id exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'image_usage_history' AND column_name = 'vendor_id') THEN
CREATE INDEX IF NOT EXISTS idx_image_usage_history_vendor_action ON image_usage_history(vendor_id, action);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_image_usage_history_action_date ON image_usage_history(action, created_at DESC);

-- Albums table indexes
-- Note: Existing albums table uses vendor_id, not owner_id/owner_type
-- CREATE INDEX IF NOT EXISTS idx_albums_owner_id ON albums(owner_id); -- Skipped: column doesn't exist
-- CREATE INDEX IF NOT EXISTS idx_albums_owner_type ON albums(owner_type); -- Skipped: column doesn't exist
CREATE INDEX IF NOT EXISTS idx_albums_is_public ON albums(is_public);
CREATE INDEX IF NOT EXISTS idx_albums_created_at ON albums(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_albums_name ON albums(name);

-- Composite indexes for albums
-- Note: Existing albums table uses vendor_id, not owner_id/owner_type
-- CREATE INDEX IF NOT EXISTS idx_albums_owner_type_id ON albums(owner_type, owner_id); -- Skipped: columns don't exist
-- CREATE INDEX IF NOT EXISTS idx_albums_owner_public ON albums(owner_id, owner_type, is_public); -- Skipped: columns don't exist

-- Conversion jobs indexes
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_conversion_id ON conversion_jobs(conversion_id);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_status ON conversion_jobs(status);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_priority ON conversion_jobs(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_worker_id ON conversion_jobs(worker_id);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_created_at ON conversion_jobs(created_at);

-- Conversion metrics indexes
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_conversion_id ON conversion_metrics(conversion_id);
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_user_id ON conversion_metrics(user_id);
-- Only create vendor_id index if column exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'conversion_metrics' AND column_name = 'vendor_id') THEN
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_vendor_id ON conversion_metrics(vendor_id);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_created_at ON conversion_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_success ON conversion_metrics(success);

-- ============================================================================
-- TRIGGERS FOR UPDATED_AT
-- ============================================================================

-- Utility function for updated_at triggers
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at (may already exist)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_conversions_updated_at') THEN
CREATE TRIGGER trg_conversions_updated_at
BEFORE UPDATE ON conversions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_images_updated_at') THEN
CREATE TRIGGER trg_images_updated_at
BEFORE UPDATE ON images
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_albums_updated_at') THEN
CREATE TRIGGER trg_albums_updated_at
BEFORE UPDATE ON albums
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_conversion_jobs_updated_at') THEN
CREATE TRIGGER trg_conversion_jobs_updated_at
BEFORE UPDATE ON conversion_jobs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- ============================================================================
-- TRIGGERS FOR COUNT UPDATES
-- ============================================================================

-- Function to update album image count
CREATE OR REPLACE FUNCTION update_album_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.album_id IS NOT NULL THEN
            UPDATE albums 
            SET image_count = image_count + 1
            WHERE id = NEW.album_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.album_id IS NOT NULL THEN
            UPDATE albums 
            SET image_count = GREATEST(0, image_count - 1)
            WHERE id = OLD.album_id;
        END IF;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Handle album change
        IF OLD.album_id IS DISTINCT FROM NEW.album_id THEN
            -- Decrease old album count
            IF OLD.album_id IS NOT NULL THEN
                UPDATE albums 
                SET image_count = GREATEST(0, image_count - 1)
                WHERE id = OLD.album_id;
            END IF;
            -- Increase new album count
            IF NEW.album_id IS NOT NULL THEN
                UPDATE albums 
                SET image_count = image_count + 1
                WHERE id = NEW.album_id;
            END IF;
        END IF;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger for album image count updates
CREATE TRIGGER trg_update_album_image_count
AFTER INSERT OR UPDATE OR DELETE ON images
FOR EACH ROW EXECUTE FUNCTION update_album_image_count();

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================

-- Function to create a conversion
CREATE OR REPLACE FUNCTION create_conversion(
    p_user_id UUID,
    p_vendor_id UUID,
    p_user_image_id UUID,
    p_cloth_image_id UUID,
    p_conversion_type TEXT DEFAULT 'free',
    p_style_name TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    conversion_id UUID;
    owner_type TEXT;
    owner_id UUID;
BEGIN
    -- Determine owner
    IF p_user_id IS NOT NULL THEN
        owner_type := 'user';
        owner_id := p_user_id;
    ELSIF p_vendor_id IS NOT NULL THEN
        owner_type := 'vendor';
        owner_id := p_vendor_id;
    ELSE
        RAISE EXCEPTION 'Either user_id or vendor_id must be provided';
    END IF;
    
    -- Validate images exist and belong to owner
    -- Note: images table uses user_id/vendor_id, not owner_id/owner_type
    IF p_user_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM images 
            WHERE id = p_user_image_id 
            AND user_id = p_user_id
            AND type IN ('user', 'result')
        ) THEN
            RAISE EXCEPTION 'User image not found or does not belong to user';
        END IF;
    ELSIF p_vendor_id IS NOT NULL THEN
    IF NOT EXISTS (
        SELECT 1 FROM images 
        WHERE id = p_user_image_id 
            AND vendor_id = p_vendor_id
            AND type IN ('vendor', 'result')
    ) THEN
            RAISE EXCEPTION 'Image not found or does not belong to vendor';
        END IF;
    END IF;
    
    -- Validate cloth image (can be public vendor image)
    IF NOT EXISTS (
        SELECT 1 FROM images 
        WHERE id = p_cloth_image_id 
        AND (type = 'vendor' OR is_public = true)
    ) THEN
        RAISE EXCEPTION 'Cloth image not found or not accessible';
    END IF;
    
    -- Create conversion record
    INSERT INTO conversions (
        user_id, vendor_id, user_image_id, cloth_image_id, 
        conversion_type, style_name
    )
    VALUES (
        p_user_id, p_vendor_id, p_user_image_id, p_cloth_image_id,
        p_conversion_type, p_style_name
    )
    RETURNING id INTO conversion_id;
    
    -- Record usage history
    INSERT INTO image_usage_history (
        image_id, user_id, vendor_id, conversion_id, action
    )
    VALUES (
        p_user_image_id, p_user_id, p_vendor_id, conversion_id, 'use_in_conversion'
    );
    
    INSERT INTO image_usage_history (
        image_id, user_id, vendor_id, conversion_id, action
    )
    VALUES (
        p_cloth_image_id, p_user_id, p_vendor_id, conversion_id, 'use_in_conversion'
    );
    
    RETURN conversion_id;
END;
$$ LANGUAGE plpgsql;

-- Function to update conversion status
CREATE OR REPLACE FUNCTION update_conversion_status(
    p_conversion_id UUID,
    p_status TEXT,
    p_result_image_id UUID DEFAULT NULL,
    p_error_message TEXT DEFAULT NULL,
    p_processing_time_ms INTEGER DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    conversion_record RECORD;
BEGIN
    -- Get conversion details
    SELECT * INTO conversion_record FROM conversions WHERE id = p_conversion_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Update conversion
    UPDATE conversions 
    SET 
        status = p_status,
        result_image_id = COALESCE(p_result_image_id, result_image_id),
        error_message = COALESCE(p_error_message, error_message),
        processing_time_ms = COALESCE(p_processing_time_ms, processing_time_ms),
        updated_at = NOW()
    WHERE id = p_conversion_id;
    
    -- Record metrics if completed or failed
    IF p_status IN ('completed', 'failed') THEN
        INSERT INTO conversion_metrics (
            conversion_id, 
            user_id, 
            vendor_id,
            processing_time_ms, 
            success, 
            error_type
        ) VALUES (
            p_conversion_id,
            conversion_record.user_id,
            conversion_record.vendor_id,
            COALESCE(p_processing_time_ms, 0),
            p_status = 'completed',
            CASE WHEN p_status = 'failed' THEN 'conversion_failed' ELSE NULL END
        );
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to record image usage
CREATE OR REPLACE FUNCTION record_image_usage(
    p_image_id UUID,
    p_user_id UUID,
    p_vendor_id UUID,
    p_action TEXT,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_session_id TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    usage_id UUID;
BEGIN
    INSERT INTO image_usage_history (
        image_id, user_id, vendor_id, action, ip_address, user_agent, session_id, metadata
    )
    VALUES (
        p_image_id, p_user_id, p_vendor_id, p_action, p_ip_address, p_user_agent, p_session_id, p_metadata
    )
    RETURNING id INTO usage_id;
    
    RETURN usage_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get conversion statistics
CREATE OR REPLACE FUNCTION get_conversion_stats(
    p_user_id UUID DEFAULT NULL,
    p_vendor_id UUID DEFAULT NULL,
    p_date_from TIMESTAMPTZ DEFAULT NULL,
    p_date_to TIMESTAMPTZ DEFAULT NULL
) RETURNS TABLE (
    total_conversions BIGINT,
    completed_conversions BIGINT,
    failed_conversions BIGINT,
    pending_conversions BIGINT,
    average_processing_time_ms NUMERIC,
    total_processing_time_ms BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_conversions,
        COUNT(*) FILTER (WHERE c.status = 'completed') as completed_conversions,
        COUNT(*) FILTER (WHERE c.status = 'failed') as failed_conversions,
        COUNT(*) FILTER (WHERE c.status IN ('pending', 'processing')) as pending_conversions,
        COALESCE(AVG(c.processing_time_ms), 0) as average_processing_time_ms,
        COALESCE(SUM(c.processing_time_ms), 0) as total_processing_time_ms
    FROM conversions c
    WHERE (p_user_id IS NULL OR c.user_id = p_user_id)
    AND (p_vendor_id IS NULL OR c.vendor_id = p_vendor_id)
    AND (p_date_from IS NULL OR c.created_at >= p_date_from)
    AND (p_date_to IS NULL OR c.created_at <= p_date_to);
END;
$$ LANGUAGE plpgsql;

-- Function to get image statistics
-- Note: Adapted for existing images table structure (user_id/vendor_id)
CREATE OR REPLACE FUNCTION get_image_stats(
    p_user_id UUID DEFAULT NULL,
    p_vendor_id UUID DEFAULT NULL,
    p_image_type TEXT DEFAULT NULL
) RETURNS TABLE (
    total_images BIGINT,
    user_images BIGINT,
    vendor_images BIGINT,
    result_images BIGINT,
    public_images BIGINT,
    private_images BIGINT,
    total_file_size BIGINT,
    average_file_size NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_images,
        COUNT(*) FILTER (WHERE i.type = 'user') as user_images,
        COUNT(*) FILTER (WHERE i.type = 'vendor') as vendor_images,
        COUNT(*) FILTER (WHERE i.type = 'result') as result_images,
        COUNT(*) FILTER (WHERE i.is_public = true) as public_images,
        COUNT(*) FILTER (WHERE i.is_public = false) as private_images,
        COALESCE(SUM(i.file_size), 0) as total_file_size,
        COALESCE(AVG(i.file_size), 0) as average_file_size
    FROM images i
    WHERE (p_user_id IS NULL OR i.user_id = p_user_id)
    AND (p_vendor_id IS NULL OR i.vendor_id = p_vendor_id)
    AND (p_image_type IS NULL OR i.type = p_image_type);
END;
$$ LANGUAGE plpgsql;

COMMIT;
