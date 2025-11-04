-- Image Service Schema Migration
-- Creates comprehensive image management tables for users, vendors, and results

BEGIN;

-- images table - comprehensive image management for all types
-- Note: Table may already exist from migration 0003, so we alter it if needed
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

-- Alter existing images table to add new columns if they don't exist
DO $$ 
BEGIN
    -- Add user_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'images' AND column_name = 'user_id') THEN
        ALTER TABLE images ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    END IF;
    
    -- Add type column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'images' AND column_name = 'type') THEN
        ALTER TABLE images ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'vendor';
    END IF;
    
    -- Add metadata column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'images' AND column_name = 'metadata') THEN
        ALTER TABLE images ADD COLUMN metadata JSONB DEFAULT '{}';
    END IF;
    
    -- Make vendor_id nullable if it's currently NOT NULL (for backward compatibility)
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'images' AND column_name = 'vendor_id' AND is_nullable = 'NO') THEN
        ALTER TABLE images ALTER COLUMN vendor_id DROP NOT NULL;
    END IF;
    
    -- Drop old constraint if it exists and add new one
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_image_ownership') THEN
        ALTER TABLE images DROP CONSTRAINT chk_image_ownership;
    END IF;
    
    -- Add new constraint
    ALTER TABLE images ADD CONSTRAINT chk_image_ownership CHECK (
        (type = 'user' AND user_id IS NOT NULL AND vendor_id IS NULL) OR
        (type = 'vendor' AND vendor_id IS NOT NULL AND user_id IS NULL) OR
        (type = 'result' AND (user_id IS NOT NULL OR vendor_id IS NOT NULL))
    );
END $$;

-- Create indexes for images table
CREATE INDEX IF NOT EXISTS idx_images_user_id ON images(user_id);
CREATE INDEX IF NOT EXISTS idx_images_vendor_id ON images(vendor_id);
CREATE INDEX IF NOT EXISTS idx_images_type ON images(type);
CREATE INDEX IF NOT EXISTS idx_images_is_public ON images(is_public);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_mime_type ON images(mime_type);
CREATE INDEX IF NOT EXISTS idx_images_file_size ON images(file_size);
CREATE INDEX IF NOT EXISTS idx_images_tags ON images USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_images_metadata ON images USING GIN(metadata);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_images_user_type ON images(user_id, type);
CREATE INDEX IF NOT EXISTS idx_images_vendor_type ON images(vendor_id, type);
CREATE INDEX IF NOT EXISTS idx_images_type_public ON images(type, is_public);

-- Add trigger for images updated_at (only if it doesn't exist)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_images_updated_at') THEN
        CREATE TRIGGER trg_images_updated_at
        BEFORE UPDATE ON images
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- image_usage_history table - track all image usage
CREATE TABLE IF NOT EXISTS image_usage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('upload', 'view', 'download', 'delete', 'update', 'share')),
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for image_usage_history table
CREATE INDEX IF NOT EXISTS idx_image_usage_history_image_id ON image_usage_history(image_id);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_user_id ON image_usage_history(user_id);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_action ON image_usage_history(action);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_created_at ON image_usage_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_ip_address ON image_usage_history(ip_address);

-- Composite indexes for analytics
CREATE INDEX IF NOT EXISTS idx_image_usage_history_image_action ON image_usage_history(image_id, action);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_user_action ON image_usage_history(user_id, action);
CREATE INDEX IF NOT EXISTS idx_image_usage_history_action_date ON image_usage_history(action, created_at DESC);

-- image_quota_tracking table - track image quotas per user/vendor
CREATE TABLE IF NOT EXISTS image_quota_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    year_month TEXT NOT NULL, -- Format: YYYY-MM
    user_images_used INTEGER NOT NULL DEFAULT 0,
    vendor_images_used INTEGER NOT NULL DEFAULT 0,
    result_images_used INTEGER NOT NULL DEFAULT 0,
    total_images_used INTEGER NOT NULL DEFAULT 0,
    total_file_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure only one of user_id or vendor_id is set
    CONSTRAINT chk_quota_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    ),
    
    -- Ensure unique quota per entity per month
    UNIQUE(user_id, year_month),
    UNIQUE(vendor_id, year_month)
);

-- Create indexes for image_quota_tracking table
CREATE INDEX IF NOT EXISTS idx_image_quota_tracking_user_id ON image_quota_tracking(user_id);
CREATE INDEX IF NOT EXISTS idx_image_quota_tracking_vendor_id ON image_quota_tracking(vendor_id);
CREATE INDEX IF NOT EXISTS idx_image_quota_tracking_year_month ON image_quota_tracking(year_month);

-- Add trigger for image_quota_tracking updated_at
CREATE TRIGGER trg_image_quota_tracking_updated_at
BEFORE UPDATE ON image_quota_tracking
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Function to get image quota status for user
CREATE OR REPLACE FUNCTION get_user_image_quota_status(p_user_id UUID)
RETURNS TABLE (
    user_images_remaining INTEGER,
    total_images_remaining INTEGER,
    user_images_used INTEGER,
    user_images_limit INTEGER,
    total_file_size BIGINT,
    file_size_limit BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        GREATEST(0, 100 - COALESCE(q.user_images_used, 0)) as user_images_remaining, -- Default limit of 100
        GREATEST(0, 100 - COALESCE(q.total_images_used, 0)) as total_images_remaining,
        COALESCE(q.user_images_used, 0) as user_images_used,
        100 as user_images_limit, -- Default limit
        COALESCE(q.total_file_size, 0) as total_file_size,
        1073741824 as file_size_limit -- 1GB default limit
    FROM image_quota_tracking q
    WHERE q.user_id = p_user_id 
    AND q.year_month = TO_CHAR(NOW(), 'YYYY-MM')
    UNION ALL
    SELECT 
        100 as user_images_remaining,
        100 as total_images_remaining,
        0 as user_images_used,
        100 as user_images_limit,
        0 as total_file_size,
        1073741824 as file_size_limit
    WHERE NOT EXISTS (
        SELECT 1 FROM image_quota_tracking 
        WHERE user_id = p_user_id 
        AND year_month = TO_CHAR(NOW(), 'YYYY-MM')
    )
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to get image quota status for vendor
CREATE OR REPLACE FUNCTION get_vendor_image_quota_status(p_vendor_id UUID)
RETURNS TABLE (
    vendor_images_remaining INTEGER,
    total_images_remaining INTEGER,
    vendor_images_used INTEGER,
    vendor_images_limit INTEGER,
    total_file_size BIGINT,
    file_size_limit BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        GREATEST(0, 1000 - COALESCE(q.vendor_images_used, 0)) as vendor_images_remaining, -- Default limit of 1000
        GREATEST(0, 1000 - COALESCE(q.total_images_used, 0)) as total_images_remaining,
        COALESCE(q.vendor_images_used, 0) as vendor_images_used,
        1000 as vendor_images_limit, -- Default limit
        COALESCE(q.total_file_size, 0) as total_file_size,
        5368709120 as file_size_limit -- 5GB default limit
    FROM image_quota_tracking q
    WHERE q.vendor_id = p_vendor_id 
    AND q.year_month = TO_CHAR(NOW(), 'YYYY-MM')
    UNION ALL
    SELECT 
        1000 as vendor_images_remaining,
        1000 as total_images_remaining,
        0 as vendor_images_used,
        1000 as vendor_images_limit,
        0 as total_file_size,
        5368709120 as file_size_limit
    WHERE NOT EXISTS (
        SELECT 1 FROM image_quota_tracking 
        WHERE vendor_id = p_vendor_id 
        AND year_month = TO_CHAR(NOW(), 'YYYY-MM')
    )
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to check if user can upload image
CREATE OR REPLACE FUNCTION can_user_upload_image(p_user_id UUID, p_type VARCHAR(20), p_file_size BIGINT)
RETURNS BOOLEAN AS $$
DECLARE
    quota_status RECORD;
BEGIN
    SELECT * INTO quota_status FROM get_user_image_quota_status(p_user_id);
    
    -- Check image count limit
    IF quota_status.user_images_remaining <= 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Check file size limit
    IF quota_status.total_file_size + p_file_size > quota_status.file_size_limit THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to check if vendor can upload image
CREATE OR REPLACE FUNCTION can_vendor_upload_image(p_vendor_id UUID, p_type VARCHAR(20), p_file_size BIGINT)
RETURNS BOOLEAN AS $$
DECLARE
    quota_status RECORD;
BEGIN
    SELECT * INTO quota_status FROM get_vendor_image_quota_status(p_vendor_id);
    
    -- Check image count limit
    IF quota_status.vendor_images_remaining <= 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Check file size limit
    IF quota_status.total_file_size + p_file_size > quota_status.file_size_limit THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
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
    current_month TEXT;
    quota_user_id UUID;
    quota_vendor_id UUID;
BEGIN
    -- Determine quota tracking entity
    IF p_type = 'user' THEN
        quota_user_id := p_user_id;
        quota_vendor_id := NULL;
    ELSIF p_type = 'vendor' THEN
        quota_user_id := NULL;
        quota_vendor_id := p_vendor_id;
    ELSIF p_type = 'result' THEN
        -- For result images, use the appropriate entity
        IF p_user_id IS NOT NULL THEN
            quota_user_id := p_user_id;
            quota_vendor_id := NULL;
        ELSE
            quota_user_id := NULL;
            quota_vendor_id := p_vendor_id;
        END IF;
    END IF;
    
    -- Check quota
    IF quota_user_id IS NOT NULL THEN
        IF NOT can_user_upload_image(quota_user_id, p_type, p_file_size) THEN
            RAISE EXCEPTION 'User image quota exceeded';
        END IF;
    ELSIF quota_vendor_id IS NOT NULL THEN
        IF NOT can_vendor_upload_image(quota_vendor_id, p_type, p_file_size) THEN
            RAISE EXCEPTION 'Vendor image quota exceeded';
        END IF;
    END IF;
    
    -- Get current month
    current_month := TO_CHAR(NOW(), 'YYYY-MM');
    
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
    
    -- Update or create quota record
    INSERT INTO image_quota_tracking (
        user_id, vendor_id, year_month, 
        user_images_used, vendor_images_used, result_images_used, 
        total_images_used, total_file_size
    )
    VALUES (
        quota_user_id, quota_vendor_id, current_month,
        CASE WHEN p_type = 'user' THEN 1 ELSE 0 END,
        CASE WHEN p_type = 'vendor' THEN 1 ELSE 0 END,
        CASE WHEN p_type = 'result' THEN 1 ELSE 0 END,
        1, p_file_size
    )
    ON CONFLICT (user_id, year_month) 
    DO UPDATE SET
        user_images_used = image_quota_tracking.user_images_used + CASE WHEN p_type = 'user' THEN 1 ELSE 0 END,
        result_images_used = image_quota_tracking.result_images_used + CASE WHEN p_type = 'result' THEN 1 ELSE 0 END,
        total_images_used = image_quota_tracking.total_images_used + 1,
        total_file_size = image_quota_tracking.total_file_size + p_file_size,
        updated_at = NOW()
    ON CONFLICT (vendor_id, year_month)
    DO UPDATE SET
        vendor_images_used = image_quota_tracking.vendor_images_used + CASE WHEN p_type = 'vendor' THEN 1 ELSE 0 END,
        result_images_used = image_quota_tracking.result_images_used + CASE WHEN p_type = 'result' THEN 1 ELSE 0 END,
        total_images_used = image_quota_tracking.total_images_used + 1,
        total_file_size = image_quota_tracking.total_file_size + p_file_size,
        updated_at = NOW();
    
    -- Record usage history
    INSERT INTO image_usage_history (image_id, user_id, action, metadata)
    VALUES (image_id, p_user_id, 'upload', jsonb_build_object('file_size', p_file_size, 'mime_type', p_mime_type));
    
    RETURN image_id;
END;
$$ LANGUAGE plpgsql;

-- Function to record image usage
CREATE OR REPLACE FUNCTION record_image_usage(
    p_image_id UUID,
    p_user_id UUID,
    p_action VARCHAR(50),
    p_ip_address INET,
    p_user_agent TEXT,
    p_metadata JSONB
) RETURNS UUID AS $$
DECLARE
    usage_id UUID;
BEGIN
    INSERT INTO image_usage_history (
        image_id, user_id, action, ip_address, user_agent, metadata
    )
    VALUES (
        p_image_id, p_user_id, p_action, p_ip_address, p_user_agent, p_metadata
    )
    RETURNING id INTO usage_id;
    
    RETURN usage_id;
END;
$$ LANGUAGE plpgsql;

-- Function to delete an image and update counts
CREATE OR REPLACE FUNCTION delete_image(p_image_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    img_record RECORD;
    quota_user_id UUID;
    quota_vendor_id UUID;
    current_month TEXT;
BEGIN
    -- Get image details
    SELECT user_id, vendor_id, type, file_size INTO img_record
    FROM images 
    WHERE id = p_image_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Determine quota tracking entity
    IF img_record.type = 'user' THEN
        quota_user_id := img_record.user_id;
        quota_vendor_id := NULL;
    ELSIF img_record.type = 'vendor' THEN
        quota_user_id := NULL;
        quota_vendor_id := img_record.vendor_id;
    ELSIF img_record.type = 'result' THEN
        IF img_record.user_id IS NOT NULL THEN
            quota_user_id := img_record.user_id;
            quota_vendor_id := NULL;
        ELSE
            quota_user_id := NULL;
            quota_vendor_id := img_record.vendor_id;
        END IF;
    END IF;
    
    -- Delete the image
    DELETE FROM images WHERE id = p_image_id;
    
    -- Update quota tracking
    current_month := TO_CHAR(NOW(), 'YYYY-MM');
    
    IF quota_user_id IS NOT NULL THEN
        UPDATE image_quota_tracking 
        SET 
            user_images_used = GREATEST(0, user_images_used - CASE WHEN img_record.type = 'user' THEN 1 ELSE 0 END),
            result_images_used = GREATEST(0, result_images_used - CASE WHEN img_record.type = 'result' THEN 1 ELSE 0 END),
            total_images_used = GREATEST(0, total_images_used - 1),
            total_file_size = GREATEST(0, total_file_size - img_record.file_size),
            updated_at = NOW()
        WHERE user_id = quota_user_id AND year_month = current_month;
    ELSIF quota_vendor_id IS NOT NULL THEN
        UPDATE image_quota_tracking 
        SET 
            vendor_images_used = GREATEST(0, vendor_images_used - CASE WHEN img_record.type = 'vendor' THEN 1 ELSE 0 END),
            result_images_used = GREATEST(0, result_images_used - CASE WHEN img_record.type = 'result' THEN 1 ELSE 0 END),
            total_images_used = GREATEST(0, total_images_used - 1),
            total_file_size = GREATEST(0, total_file_size - img_record.file_size),
            updated_at = NOW()
        WHERE vendor_id = quota_vendor_id AND year_month = current_month;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to get image statistics
CREATE OR REPLACE FUNCTION get_image_stats(p_user_id UUID DEFAULT NULL, p_vendor_id UUID DEFAULT NULL)
RETURNS TABLE (
    total_images INTEGER,
    user_images INTEGER,
    vendor_images INTEGER,
    result_images INTEGER,
    public_images INTEGER,
    private_images INTEGER,
    total_file_size BIGINT,
    average_file_size BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_images,
        COUNT(CASE WHEN type = 'user' THEN 1 END)::INTEGER as user_images,
        COUNT(CASE WHEN type = 'vendor' THEN 1 END)::INTEGER as vendor_images,
        COUNT(CASE WHEN type = 'result' THEN 1 END)::INTEGER as result_images,
        COUNT(CASE WHEN is_public = true THEN 1 END)::INTEGER as public_images,
        COUNT(CASE WHEN is_public = false THEN 1 END)::INTEGER as private_images,
        COALESCE(SUM(file_size), 0) as total_file_size,
        COALESCE(AVG(file_size), 0)::BIGINT as average_file_size
    FROM images 
    WHERE (p_user_id IS NULL OR user_id = p_user_id)
    AND (p_vendor_id IS NULL OR vendor_id = p_vendor_id);
END;
$$ LANGUAGE plpgsql;

COMMIT;
