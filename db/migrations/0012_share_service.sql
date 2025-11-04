-- Share Service Schema Migration
-- Creates tables for public sharing of conversion results with signed URLs and token-based access

BEGIN;

-- shared_links table - track all shared conversion links
CREATE TABLE IF NOT EXISTS shared_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversion_id UUID NOT NULL REFERENCES conversions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_token TEXT NOT NULL UNIQUE, -- Unique token for access
    signed_url TEXT NOT NULL, -- Pre-signed URL for direct access
    expires_at TIMESTAMPTZ NOT NULL, -- Link expiration time (1-5 minutes)
    access_count INTEGER NOT NULL DEFAULT 0, -- Track how many times accessed
    max_access_count INTEGER DEFAULT NULL, -- Optional access limit
    is_active BOOLEAN NOT NULL DEFAULT true, -- Can be deactivated before expiry
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- Note: PostgreSQL doesn't allow subqueries in CHECK constraints
    -- Validation will be done in application layer or via trigger
);

-- Create indexes for shared_links table
CREATE INDEX IF NOT EXISTS idx_shared_links_conversion_id ON shared_links(conversion_id);
CREATE INDEX IF NOT EXISTS idx_shared_links_user_id ON shared_links(user_id);
CREATE INDEX IF NOT EXISTS idx_shared_links_share_token ON shared_links(share_token);
CREATE INDEX IF NOT EXISTS idx_shared_links_expires_at ON shared_links(expires_at);
CREATE INDEX IF NOT EXISTS idx_shared_links_is_active ON shared_links(is_active);
CREATE INDEX IF NOT EXISTS idx_shared_links_created_at ON shared_links(created_at DESC);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_shared_links_active_expires ON shared_links(is_active, expires_at);
CREATE INDEX IF NOT EXISTS idx_shared_links_user_active ON shared_links(user_id, is_active);

-- Add trigger for shared_links updated_at
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_shared_links_updated_at') THEN
        CREATE TRIGGER trg_shared_links_updated_at
        BEFORE UPDATE ON shared_links
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- shared_link_access_logs table - track all access attempts to shared links
CREATE TABLE IF NOT EXISTS shared_link_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shared_link_id UUID NOT NULL REFERENCES shared_links(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    access_type VARCHAR(20) NOT NULL DEFAULT 'view' CHECK (access_type IN ('view', 'download')),
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for shared_link_access_logs table
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_shared_link_id ON shared_link_access_logs(shared_link_id);
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_ip_address ON shared_link_access_logs(ip_address);
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_access_type ON shared_link_access_logs(access_type);
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_success ON shared_link_access_logs(success);
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_created_at ON shared_link_access_logs(created_at DESC);

-- Composite indexes for analytics
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_link_success ON shared_link_access_logs(shared_link_id, success);
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_type_date ON shared_link_access_logs(access_type, created_at DESC);

-- Function to create a shared link for a conversion
CREATE OR REPLACE FUNCTION create_shared_link(
    p_conversion_id UUID,
    p_user_id UUID,
    p_expiry_minutes INTEGER DEFAULT 5,
    p_max_access_count INTEGER DEFAULT NULL
) RETURNS TABLE (
    share_id UUID,
    share_token TEXT,
    signed_url TEXT,
    expires_at TIMESTAMPTZ
) AS $$
DECLARE
    conversion_record RECORD;
    new_share_id UUID;
    new_share_token TEXT;
    new_signed_url TEXT;
    expiry_time TIMESTAMPTZ;
BEGIN
    -- Validate conversion exists and is completed
    SELECT c.*, i.original_url as result_image_url
    INTO conversion_record
    FROM conversions c
    LEFT JOIN images i ON c.result_image_id = i.id
    WHERE c.id = p_conversion_id 
    AND c.user_id = p_user_id 
    AND c.status = 'completed' 
    AND c.result_image_id IS NOT NULL;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Conversion not found, not completed, or not owned by user';
    END IF;
    
    -- Validate expiry time (1-5 minutes)
    IF p_expiry_minutes < 1 OR p_expiry_minutes > 5 THEN
        RAISE EXCEPTION 'Expiry time must be between 1 and 5 minutes';
    END IF;
    
    -- Calculate expiry time
    expiry_time := NOW() + INTERVAL '1 minute' * p_expiry_minutes;
    
    -- Generate unique share token (base64 encoded UUID + timestamp)
    new_share_token := encode(gen_random_bytes(32), 'base64url');
    
    -- Generate signed URL (this would be implemented in the application layer)
    -- For now, we'll create a placeholder URL structure
    new_signed_url := '/api/share/' || new_share_token;
    
    -- Create shared link record
    INSERT INTO shared_links (
        conversion_id, user_id, share_token, signed_url, 
        expires_at, max_access_count
    )
    VALUES (
        p_conversion_id, p_user_id, new_share_token, new_signed_url,
        expiry_time, p_max_access_count
    )
    RETURNING id INTO new_share_id;
    
    -- Return the created link details
    RETURN QUERY
    SELECT 
        new_share_id,
        new_share_token,
        new_signed_url,
        expiry_time;
END;
$$ LANGUAGE plpgsql;

-- Function to validate and access a shared link
CREATE OR REPLACE FUNCTION access_shared_link(
    p_share_token TEXT,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_referer TEXT DEFAULT NULL,
    p_access_type VARCHAR(20) DEFAULT 'view'
) RETURNS TABLE (
    success BOOLEAN,
    conversion_id UUID,
    result_image_url TEXT,
    error_message TEXT
) AS $$
DECLARE
    link_record RECORD;
    conversion_record RECORD;
    access_log_id UUID;
BEGIN
    -- Get shared link details
    SELECT sl.*, c.result_image_id
    INTO link_record
    FROM shared_links sl
    LEFT JOIN conversions c ON sl.conversion_id = c.id
    WHERE sl.share_token = p_share_token;
    
    IF NOT FOUND THEN
        -- Log failed access attempt
        INSERT INTO shared_link_access_logs (
            shared_link_id, ip_address, user_agent, referer, 
            access_type, success, error_message
        )
        VALUES (
            NULL, p_ip_address, p_user_agent, p_referer,
            p_access_type, false, 'Share token not found'
        )
        RETURNING id INTO access_log_id;
        
        RETURN QUERY SELECT false, NULL::UUID, NULL::TEXT, 'Share token not found';
        RETURN;
    END IF;
    
    -- Check if link is active
    IF NOT link_record.is_active THEN
        -- Log failed access attempt
        INSERT INTO shared_link_access_logs (
            shared_link_id, ip_address, user_agent, referer, 
            access_type, success, error_message
        )
        VALUES (
            link_record.id, p_ip_address, p_user_agent, p_referer,
            p_access_type, false, 'Share link is inactive'
        )
        RETURNING id INTO access_log_id;
        
        RETURN QUERY SELECT false, NULL::UUID, NULL::TEXT, 'Share link is inactive';
        RETURN;
    END IF;
    
    -- Check if link has expired
    IF link_record.expires_at < NOW() THEN
        -- Log failed access attempt
        INSERT INTO shared_link_access_logs (
            shared_link_id, ip_address, user_agent, referer, 
            access_type, success, error_message
        )
        VALUES (
            link_record.id, p_ip_address, p_user_agent, p_referer,
            p_access_type, false, 'Share link has expired'
        )
        RETURNING id INTO access_log_id;
        
        RETURN QUERY SELECT false, NULL::UUID, NULL::TEXT, 'Share link has expired';
        RETURN;
    END IF;
    
    -- Check access count limit
    IF link_record.max_access_count IS NOT NULL 
    AND link_record.access_count >= link_record.max_access_count THEN
        -- Log failed access attempt
        INSERT INTO shared_link_access_logs (
            shared_link_id, ip_address, user_agent, referer, 
            access_type, success, error_message
        )
        VALUES (
            link_record.id, p_ip_address, p_user_agent, p_referer,
            p_access_type, false, 'Share link access limit exceeded'
        )
        RETURNING id INTO access_log_id;
        
        RETURN QUERY SELECT false, NULL::UUID, NULL::TEXT, 'Share link access limit exceeded';
        RETURN;
    END IF;
    
    -- Get conversion and image details
    SELECT c.id, i.original_url as result_image_url
    INTO conversion_record
    FROM conversions c
    LEFT JOIN images i ON c.result_image_id = i.id
    WHERE c.id = link_record.conversion_id;
    
    -- Update access count
    UPDATE shared_links 
    SET access_count = access_count + 1, updated_at = NOW()
    WHERE id = link_record.id;
    
    -- Log successful access
    INSERT INTO shared_link_access_logs (
        shared_link_id, ip_address, user_agent, referer, 
        access_type, success, metadata
    )
    VALUES (
        link_record.id, p_ip_address, p_user_agent, p_referer,
        p_access_type, true, jsonb_build_object('access_count', link_record.access_count + 1)
    )
    RETURNING id INTO access_log_id;
    
    RETURN QUERY SELECT true, conversion_record.id, conversion_record.result_image_url, NULL::TEXT;
END;
$$ LANGUAGE plpgsql;

-- Function to deactivate a shared link
CREATE OR REPLACE FUNCTION deactivate_shared_link(
    p_share_id UUID,
    p_user_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    link_record RECORD;
BEGIN
    -- Get shared link details
    SELECT * INTO link_record
    FROM shared_links
    WHERE id = p_share_id AND user_id = p_user_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Deactivate the link
    UPDATE shared_links 
    SET is_active = false, updated_at = NOW()
    WHERE id = p_share_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to get shared link statistics
CREATE OR REPLACE FUNCTION get_shared_link_stats(
    p_user_id UUID DEFAULT NULL,
    p_conversion_id UUID DEFAULT NULL
) RETURNS TABLE (
    total_links INTEGER,
    active_links INTEGER,
    expired_links INTEGER,
    total_access_count BIGINT,
    unique_ip_addresses BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_links,
        COUNT(CASE WHEN is_active = true AND expires_at > NOW() THEN 1 END)::INTEGER as active_links,
        COUNT(CASE WHEN expires_at <= NOW() THEN 1 END)::INTEGER as expired_links,
        COALESCE(SUM(access_count), 0) as total_access_count,
        COALESCE(COUNT(DISTINCT sla.ip_address), 0) as unique_ip_addresses
    FROM shared_links sl
    LEFT JOIN shared_link_access_logs sla ON sl.id = sla.shared_link_id AND sla.success = true
    WHERE (p_user_id IS NULL OR sl.user_id = p_user_id)
    AND (p_conversion_id IS NULL OR sl.conversion_id = p_conversion_id);
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup expired shared links (can be run periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_shared_links()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired shared links and their access logs
    WITH deleted_links AS (
        DELETE FROM shared_links 
        WHERE expires_at < NOW() - INTERVAL '1 hour' -- Keep for 1 hour after expiry for analytics
        RETURNING id
    )
    DELETE FROM shared_link_access_logs 
    WHERE shared_link_id IN (SELECT id FROM deleted_links);
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a view for active shared links with conversion details
CREATE OR REPLACE VIEW active_shared_links AS
SELECT 
    sl.id as share_id,
    sl.conversion_id,
    sl.user_id,
    sl.share_token,
    sl.signed_url,
    sl.expires_at,
    sl.access_count,
    sl.max_access_count,
    sl.created_at,
    c.status as conversion_status,
    c.result_image_id,
    i.original_url as result_image_url,
    i.file_name as result_image_name,
    i.file_size as result_image_size,
    i.mime_type as result_image_mime_type,
    EXTRACT(EPOCH FROM (sl.expires_at - NOW())) as seconds_until_expiry
FROM shared_links sl
LEFT JOIN conversions c ON sl.conversion_id = c.id
LEFT JOIN images i ON c.result_image_id = i.id
WHERE sl.is_active = true 
AND sl.expires_at > NOW();

-- Grant necessary permissions (only if app_user role exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON shared_links TO app_user;
        GRANT SELECT, INSERT ON shared_link_access_logs TO app_user;
        GRANT SELECT ON active_shared_links TO app_user;
        GRANT EXECUTE ON FUNCTION create_shared_link TO app_user;
        GRANT EXECUTE ON FUNCTION access_shared_link TO app_user;
        GRANT EXECUTE ON FUNCTION deactivate_shared_link TO app_user;
        GRANT EXECUTE ON FUNCTION get_shared_link_stats TO app_user;
        GRANT EXECUTE ON FUNCTION cleanup_expired_shared_links TO app_user;
    END IF;
END $$;

COMMIT;
