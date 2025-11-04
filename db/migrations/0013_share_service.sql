-- Share Service Database Migration
-- Creates tables for shared links and access logging

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- SHARE SERVICE TABLES
-- ============================================================================

-- Shared links table - Store shared conversion results
CREATE TABLE IF NOT EXISTS shared_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversion_id UUID NOT NULL REFERENCES conversions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_token TEXT UNIQUE NOT NULL,
    signed_url TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    max_access_count INTEGER,
    access_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Shared link access logs table - Track access attempts
-- Note: Table already exists from migration 0012, so we only add missing columns
CREATE TABLE IF NOT EXISTS shared_link_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shared_link_id UUID NOT NULL REFERENCES shared_links(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT
);

-- Add missing columns to existing shared_link_access_logs table
DO $$ 
BEGIN
    -- Add accessed_at if it doesn't exist (migration 0012 uses created_at)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'shared_link_access_logs' AND column_name = 'accessed_at') THEN
        -- Add accessed_at column and copy data from created_at if it exists
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'shared_link_access_logs' AND column_name = 'created_at') THEN
            ALTER TABLE shared_link_access_logs ADD COLUMN accessed_at TIMESTAMPTZ;
            UPDATE shared_link_access_logs SET accessed_at = created_at WHERE accessed_at IS NULL;
            ALTER TABLE shared_link_access_logs ALTER COLUMN accessed_at SET NOT NULL;
            ALTER TABLE shared_link_access_logs ALTER COLUMN accessed_at SET DEFAULT NOW();
        ELSE
            ALTER TABLE shared_link_access_logs ADD COLUMN accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
        END IF;
    END IF;
END $$;

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Shared links table indexes
CREATE INDEX IF NOT EXISTS idx_shared_links_conversion_id ON shared_links(conversion_id);
CREATE INDEX IF NOT EXISTS idx_shared_links_user_id ON shared_links(user_id);
CREATE INDEX IF NOT EXISTS idx_shared_links_share_token ON shared_links(share_token);
CREATE INDEX IF NOT EXISTS idx_shared_links_expires_at ON shared_links(expires_at);
CREATE INDEX IF NOT EXISTS idx_shared_links_is_active ON shared_links(is_active);
CREATE INDEX IF NOT EXISTS idx_shared_links_created_at ON shared_links(created_at DESC);

-- Shared link access logs table indexes
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_shared_link_id ON shared_link_access_logs(shared_link_id);
-- Only create accessed_at index if column exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'shared_link_access_logs' AND column_name = 'accessed_at') THEN
        CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_accessed_at ON shared_link_access_logs(accessed_at DESC);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_success ON shared_link_access_logs(success);
CREATE INDEX IF NOT EXISTS idx_shared_link_access_logs_ip_address ON shared_link_access_logs(ip_address);

-- ============================================================================
-- TRIGGERS FOR UPDATED_AT
-- ============================================================================

-- Trigger may already exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_shared_links_updated_at') THEN
        CREATE TRIGGER trg_shared_links_updated_at
        BEFORE UPDATE ON shared_links
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================

-- Function to cleanup expired shared links
CREATE OR REPLACE FUNCTION cleanup_expired_shared_links()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM shared_links 
    WHERE expires_at < NOW() AND is_active = true;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get shared link statistics
-- Note: Function may already exist with different return type from migration 0012
-- Drop and recreate if signature differs
DO $$ 
BEGIN
    -- Drop function if it exists with different signature
    DROP FUNCTION IF EXISTS get_shared_link_stats(UUID, UUID);
EXCEPTION 
    WHEN OTHERS THEN NULL;
END $$;

CREATE OR REPLACE FUNCTION get_shared_link_stats(p_user_id UUID, p_conversion_id UUID)
RETURNS TABLE (
    total_shared_links INTEGER,
    active_shared_links INTEGER,
    total_access_count INTEGER,
    unique_access_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_shared_links,
        COUNT(*) FILTER (WHERE is_active = true AND expires_at > NOW())::INTEGER as active_shared_links,
        COALESCE(SUM(access_count), 0)::INTEGER as total_access_count,
        COUNT(DISTINCT sla.ip_address)::INTEGER as unique_access_count
    FROM shared_links sl
    LEFT JOIN shared_link_access_logs sla ON sl.id = sla.shared_link_id AND sla.success = true
    WHERE sl.user_id = p_user_id 
      AND (p_conversion_id IS NULL OR sl.conversion_id = p_conversion_id);
END;
$$ LANGUAGE plpgsql;

-- Function to validate shared link access
CREATE OR REPLACE FUNCTION validate_shared_link_access(p_share_token TEXT)
RETURNS TABLE (
    shared_link_id UUID,
    conversion_id UUID,
    user_id UUID,
    signed_url TEXT,
    expires_at TIMESTAMPTZ,
    max_access_count INTEGER,
    current_access_count INTEGER,
    is_valid BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        sl.id as shared_link_id,
        sl.conversion_id,
        sl.user_id,
        sl.signed_url,
        sl.expires_at,
        sl.max_access_count,
        sl.access_count as current_access_count,
        (sl.is_active = true AND sl.expires_at > NOW() AND 
         (sl.max_access_count IS NULL OR sl.access_count < sl.max_access_count)) as is_valid
    FROM shared_links sl
    WHERE sl.share_token = p_share_token;
END;
$$ LANGUAGE plpgsql;

-- Function to record shared link access
CREATE OR REPLACE FUNCTION record_shared_link_access(
    p_shared_link_id UUID,
    p_ip_address INET,
    p_user_agent TEXT,
    p_success BOOLEAN,
    p_error_message TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    access_log_id UUID;
BEGIN
    -- Insert access log
    INSERT INTO shared_link_access_logs (shared_link_id, ip_address, user_agent, success, error_message)
    VALUES (p_shared_link_id, p_ip_address, p_user_agent, p_success, p_error_message)
    RETURNING id INTO access_log_id;
    
    -- Update access count if successful
    IF p_success THEN
        UPDATE shared_links 
        SET access_count = access_count + 1, updated_at = NOW()
        WHERE id = p_shared_link_id;
    END IF;
    
    RETURN access_log_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- CONSTRAINTS AND VALIDATIONS
-- ============================================================================

-- Ensure share_token is unique and not empty
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_share_token_not_empty') THEN
        ALTER TABLE shared_links ADD CONSTRAINT chk_share_token_not_empty CHECK (LENGTH(share_token) > 0);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_expires_at_future') THEN
        ALTER TABLE shared_links ADD CONSTRAINT chk_expires_at_future CHECK (expires_at > created_at);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_max_access_count_positive') THEN
        ALTER TABLE shared_links ADD CONSTRAINT chk_max_access_count_positive CHECK (max_access_count IS NULL OR max_access_count > 0);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_access_count_non_negative') THEN
        ALTER TABLE shared_links ADD CONSTRAINT chk_access_count_non_negative CHECK (access_count >= 0);
    END IF;
END $$;

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE shared_links IS 'Stores shared links for conversion results with access control';
COMMENT ON TABLE shared_link_access_logs IS 'Logs all access attempts to shared links for analytics and security';

COMMENT ON COLUMN shared_links.share_token IS 'Unique token used in public URLs for accessing shared content';
COMMENT ON COLUMN shared_links.signed_url IS 'Pre-signed URL for accessing the actual image content';
COMMENT ON COLUMN shared_links.max_access_count IS 'Maximum number of times the link can be accessed (NULL = unlimited)';
COMMENT ON COLUMN shared_links.access_count IS 'Current number of successful accesses';
COMMENT ON COLUMN shared_links.is_active IS 'Whether the shared link is currently active';

COMMENT ON COLUMN shared_link_access_logs.ip_address IS 'IP address of the user accessing the shared link';
COMMENT ON COLUMN shared_link_access_logs.user_agent IS 'User agent string from the request';
COMMENT ON COLUMN shared_link_access_logs.success IS 'Whether the access attempt was successful';
COMMENT ON COLUMN shared_link_access_logs.error_message IS 'Error message if access failed';

COMMIT;
