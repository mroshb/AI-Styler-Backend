-- Fix create_conversion function to allow user's own images as cloth images
-- This script updates the SQL function to match the updated validation logic

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
    
    -- Validate cloth image (can be public vendor image, public image, or user's own image)
    -- Cloth image is accessible if:
    -- 1. It's a vendor image (type = 'vendor')
    -- 2. It's public (is_public = true)
    -- 3. It belongs to the user (user_id = p_user_id for user images)
    IF NOT EXISTS (
        SELECT 1 FROM images 
        WHERE id = p_cloth_image_id 
        AND (
            type = 'vendor' 
            OR is_public = true
            OR (p_user_id IS NOT NULL AND user_id = p_user_id AND type = 'user')
        )
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
    
    -- Update quota (handled by conversion_quotas table)
    -- This is a simplified version - adjust based on your quota logic
    
    RETURN conversion_id;
END;
$$ LANGUAGE plpgsql;

