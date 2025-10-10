-- Vendor Service Schema Migration
-- Creates vendor profiles, albums, and image management tables

BEGIN;

-- vendors table - vendor profiles and business information
CREATE TABLE IF NOT EXISTS vendors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    business_name TEXT NOT NULL,
    avatar_url TEXT,
    bio TEXT,
    contact_info JSONB NOT NULL DEFAULT '{}',
    social_links JSONB NOT NULL DEFAULT '{}',
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    free_images_used INTEGER NOT NULL DEFAULT 0,
    free_images_limit INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Create indexes for vendors table
CREATE INDEX IF NOT EXISTS idx_vendors_user_id ON vendors(user_id);
CREATE INDEX IF NOT EXISTS idx_vendors_business_name ON vendors(business_name);
CREATE INDEX IF NOT EXISTS idx_vendors_is_verified ON vendors(is_verified);
CREATE INDEX IF NOT EXISTS idx_vendors_is_active ON vendors(is_active);
CREATE INDEX IF NOT EXISTS idx_vendors_created_at ON vendors(created_at DESC);

-- Add trigger for vendors updated_at
CREATE TRIGGER trg_vendors_updated_at
BEFORE UPDATE ON vendors
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- albums table - vendor image albums/categories
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

-- Create indexes for albums table
CREATE INDEX IF NOT EXISTS idx_albums_vendor_id ON albums(vendor_id);
CREATE INDEX IF NOT EXISTS idx_albums_is_public ON albums(is_public);
CREATE INDEX IF NOT EXISTS idx_albums_created_at ON albums(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_albums_name ON albums(name);

-- Add trigger for albums updated_at
CREATE TRIGGER trg_albums_updated_at
BEFORE UPDATE ON albums
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- images table - vendor uploaded images
CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    album_id UUID REFERENCES albums(id) ON DELETE SET NULL,
    file_name TEXT NOT NULL,
    original_url TEXT NOT NULL,
    thumbnail_url TEXT,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    is_free BOOLEAN NOT NULL DEFAULT true,
    is_public BOOLEAN NOT NULL DEFAULT false,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for images table
CREATE INDEX IF NOT EXISTS idx_images_vendor_id ON images(vendor_id);
CREATE INDEX IF NOT EXISTS idx_images_album_id ON images(album_id);
CREATE INDEX IF NOT EXISTS idx_images_is_free ON images(is_free);
CREATE INDEX IF NOT EXISTS idx_images_is_public ON images(is_public);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_mime_type ON images(mime_type);
CREATE INDEX IF NOT EXISTS idx_images_tags ON images USING GIN(tags);

-- Add trigger for images updated_at
CREATE TRIGGER trg_images_updated_at
BEFORE UPDATE ON images
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- vendor_image_quotas table - track monthly image usage per vendor
CREATE TABLE IF NOT EXISTS vendor_image_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    year_month TEXT NOT NULL, -- Format: YYYY-MM
    free_images_used INTEGER NOT NULL DEFAULT 0,
    paid_images_used INTEGER NOT NULL DEFAULT 0,
    total_images_used INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(vendor_id, year_month)
);

-- Create indexes for vendor_image_quotas table
CREATE INDEX IF NOT EXISTS idx_vendor_image_quotas_vendor_id ON vendor_image_quotas(vendor_id);
CREATE INDEX IF NOT EXISTS idx_vendor_image_quotas_year_month ON vendor_image_quotas(year_month);

-- Add trigger for vendor_image_quotas updated_at
CREATE TRIGGER trg_vendor_image_quotas_updated_at
BEFORE UPDATE ON vendor_image_quotas
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Function to get vendor image quota status
CREATE OR REPLACE FUNCTION get_vendor_image_quota_status(p_vendor_id UUID)
RETURNS TABLE (
    free_images_remaining INTEGER,
    paid_images_remaining INTEGER,
    total_images_remaining INTEGER,
    free_images_used INTEGER,
    free_images_limit INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        GREATEST(0, v.free_images_limit - v.free_images_used) as free_images_remaining,
        0 as paid_images_remaining, -- No paid images for now
        GREATEST(0, v.free_images_limit - v.free_images_used) as total_images_remaining,
        v.free_images_used as free_images_used,
        v.free_images_limit as free_images_limit
    FROM vendors v
    WHERE v.id = p_vendor_id;
END;
$$ LANGUAGE plpgsql;

-- Function to check if vendor can upload image
CREATE OR REPLACE FUNCTION can_vendor_upload_image(p_vendor_id UUID, p_is_free BOOLEAN)
RETURNS BOOLEAN AS $$
DECLARE
    quota_status RECORD;
BEGIN
    SELECT * INTO quota_status FROM get_vendor_image_quota_status(p_vendor_id);
    
    IF p_is_free THEN
        RETURN quota_status.free_images_remaining > 0;
    ELSE
        -- For now, only free images are supported
        RETURN FALSE;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to record an image upload
CREATE OR REPLACE FUNCTION record_image_upload(
    p_vendor_id UUID,
    p_album_id UUID,
    p_file_name TEXT,
    p_original_url TEXT,
    p_thumbnail_url TEXT,
    p_file_size BIGINT,
    p_mime_type TEXT,
    p_width INTEGER,
    p_height INTEGER,
    p_is_free BOOLEAN,
    p_is_public BOOLEAN,
    p_tags TEXT[]
) RETURNS UUID AS $$
DECLARE
    image_id UUID;
    current_month TEXT;
BEGIN
    -- Check if vendor can upload image
    IF NOT can_vendor_upload_image(p_vendor_id, p_is_free) THEN
        RAISE EXCEPTION 'Vendor image quota exceeded';
    END IF;
    
    -- Get current month
    current_month := TO_CHAR(NOW(), 'YYYY-MM');
    
    -- Create image record
    INSERT INTO images (
        vendor_id, album_id, file_name, original_url, thumbnail_url,
        file_size, mime_type, width, height, is_free, is_public, tags
    )
    VALUES (
        p_vendor_id, p_album_id, p_file_name, p_original_url, p_thumbnail_url,
        p_file_size, p_mime_type, p_width, p_height, p_is_free, p_is_public, p_tags
    )
    RETURNING id INTO image_id;
    
    -- Update vendor's free image count if it's a free image
    IF p_is_free THEN
        UPDATE vendors 
        SET free_images_used = free_images_used + 1
        WHERE id = p_vendor_id;
    END IF;
    
    -- Update album image count
    IF p_album_id IS NOT NULL THEN
        UPDATE albums 
        SET image_count = image_count + 1
        WHERE id = p_album_id;
    END IF;
    
    -- Update or create monthly quota record
    INSERT INTO vendor_image_quotas (vendor_id, year_month, free_images_used, paid_images_used, total_images_used)
    VALUES (
        p_vendor_id, 
        current_month,
        CASE WHEN p_is_free THEN 1 ELSE 0 END,
        CASE WHEN p_is_free THEN 0 ELSE 1 END,
        1
    )
    ON CONFLICT (vendor_id, year_month) 
    DO UPDATE SET
        free_images_used = vendor_image_quotas.free_images_used + CASE WHEN p_is_free THEN 1 ELSE 0 END,
        paid_images_used = vendor_image_quotas.paid_images_used + CASE WHEN p_is_free THEN 0 ELSE 1 END,
        total_images_used = vendor_image_quotas.total_images_used + 1,
        updated_at = NOW();
    
    RETURN image_id;
END;
$$ LANGUAGE plpgsql;

-- Function to delete an image and update counts
CREATE OR REPLACE FUNCTION delete_vendor_image(p_image_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    img_record RECORD;
BEGIN
    -- Get image details
    SELECT vendor_id, album_id, is_free INTO img_record
    FROM images 
    WHERE id = p_image_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Delete the image
    DELETE FROM images WHERE id = p_image_id;
    
    -- Update vendor's free image count if it was a free image
    IF img_record.is_free THEN
        UPDATE vendors 
        SET free_images_used = GREATEST(0, free_images_used - 1)
        WHERE id = img_record.vendor_id;
    END IF;
    
    -- Update album image count
    IF img_record.album_id IS NOT NULL THEN
        UPDATE albums 
        SET image_count = GREATEST(0, image_count - 1)
        WHERE id = img_record.album_id;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to get vendor statistics
CREATE OR REPLACE FUNCTION get_vendor_stats(p_vendor_id UUID)
RETURNS TABLE (
    total_images INTEGER,
    free_images_used INTEGER,
    free_images_limit INTEGER,
    paid_images_used INTEGER,
    total_albums INTEGER,
    public_albums INTEGER,
    public_images INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*)::INTEGER FROM images WHERE vendor_id = p_vendor_id) as total_images,
        v.free_images_used as free_images_used,
        v.free_images_limit as free_images_limit,
        0 as paid_images_used, -- No paid images for now
        (SELECT COUNT(*)::INTEGER FROM albums WHERE vendor_id = p_vendor_id) as total_albums,
        (SELECT COUNT(*)::INTEGER FROM albums WHERE vendor_id = p_vendor_id AND is_public = true) as public_albums,
        (SELECT COUNT(*)::INTEGER FROM images WHERE vendor_id = p_vendor_id AND is_public = true) as public_images
    FROM vendors v
    WHERE v.id = p_vendor_id;
END;
$$ LANGUAGE plpgsql;

COMMIT;
