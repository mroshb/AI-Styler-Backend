-- Storage Architecture Database Migration
-- Comprehensive metadata tracking for AI Stayler storage system
-- Implements local server folders with database metadata tracking

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- STORAGE METADATA TABLES
-- ============================================================================

-- storage_files table - Track all files in the storage system
CREATE TABLE IF NOT EXISTS storage_files (
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
    access_count INTEGER NOT NULL DEFAULT 0,
    
    -- Ensure owner_id references correct table based on owner_type
    CONSTRAINT chk_storage_owner_user CHECK (
        (owner_type = 'user' AND owner_id IN (SELECT id FROM users)) OR
        (owner_type = 'vendor' AND owner_id IN (SELECT id FROM vendors))
    )
);

-- storage_access_logs table - Track all file access
CREATE TABLE IF NOT EXISTS storage_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES storage_files(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    access_type TEXT NOT NULL CHECK (access_type IN ('view', 'download', 'upload', 'delete', 'update')),
    ip_address INET,
    user_agent TEXT,
    session_id TEXT,
    signed_url TEXT,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    response_time_ms INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure at least one of user_id or vendor_id is set
    CONSTRAINT chk_access_log_actor CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);

-- storage_quotas table - Track storage quotas per user/vendor
CREATE TABLE IF NOT EXISTS storage_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    quota_type TEXT NOT NULL CHECK (quota_type IN ('user', 'vendor', 'total')),
    max_file_size BIGINT NOT NULL DEFAULT 52428800, -- 50MB default
    max_files INTEGER NOT NULL DEFAULT 100,
    max_total_size BIGINT NOT NULL DEFAULT 5368709120, -- 5GB default
    current_file_count INTEGER NOT NULL DEFAULT 0,
    current_total_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure only one of user_id or vendor_id is set
    CONSTRAINT chk_quota_ownership CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    ),
    
    -- Ensure unique quota per entity per type
    UNIQUE(user_id, quota_type),
    UNIQUE(vendor_id, quota_type)
);

-- storage_backups table - Track backup operations
CREATE TABLE IF NOT EXISTS storage_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES storage_files(id) ON DELETE CASCADE,
    backup_path TEXT NOT NULL,
    backup_type TEXT NOT NULL CHECK (backup_type IN ('manual', 'scheduled', 'automatic')),
    backup_size BIGINT NOT NULL,
    backup_checksum TEXT NOT NULL,
    compression_level INTEGER DEFAULT 0,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}'
);

-- storage_signed_urls table - Track signed URL generation
CREATE TABLE IF NOT EXISTS storage_signed_urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES storage_files(id) ON DELETE CASCADE,
    signed_url TEXT NOT NULL,
    access_type TEXT NOT NULL CHECK (access_type IN ('view', 'download')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    usage_count INTEGER NOT NULL DEFAULT 0,
    max_usage INTEGER DEFAULT 1,
    metadata JSONB DEFAULT '{}'
);

-- storage_health_checks table - Track storage system health
CREATE TABLE IF NOT EXISTS storage_health_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_type TEXT NOT NULL CHECK (check_type IN ('disk_space', 'file_integrity', 'backup_status', 'quota_status')),
    status TEXT NOT NULL CHECK (status IN ('healthy', 'warning', 'critical', 'error')),
    message TEXT,
    disk_usage_percent DECIMAL(5,2),
    free_space_bytes BIGINT,
    total_space_bytes BIGINT,
    error_count INTEGER DEFAULT 0,
    warning_count INTEGER DEFAULT 0,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- storage_metrics table - Track storage performance metrics
CREATE TABLE IF NOT EXISTS storage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_type TEXT NOT NULL CHECK (metric_type IN ('upload', 'download', 'delete', 'backup', 'access')),
    metric_value BIGINT NOT NULL,
    metric_unit TEXT NOT NULL CHECK (metric_unit IN ('count', 'bytes', 'milliseconds', 'percentage')),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    
    -- Ensure at least one of user_id or vendor_id is set
    CONSTRAINT chk_metrics_actor CHECK (
        (user_id IS NOT NULL AND vendor_id IS NULL) OR
        (vendor_id IS NOT NULL AND user_id IS NULL)
    )
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- storage_files indexes
CREATE INDEX IF NOT EXISTS idx_storage_files_file_path ON storage_files(file_path);
CREATE INDEX IF NOT EXISTS idx_storage_files_owner_id ON storage_files(owner_id);
CREATE INDEX IF NOT EXISTS idx_storage_files_owner_type ON storage_files(owner_type);
CREATE INDEX IF NOT EXISTS idx_storage_files_storage_type ON storage_files(storage_type);
CREATE INDEX IF NOT EXISTS idx_storage_files_is_public ON storage_files(is_public);
CREATE INDEX IF NOT EXISTS idx_storage_files_is_backed_up ON storage_files(is_backed_up);
CREATE INDEX IF NOT EXISTS idx_storage_files_created_at ON storage_files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_storage_files_last_accessed ON storage_files(last_accessed DESC);
CREATE INDEX IF NOT EXISTS idx_storage_files_access_count ON storage_files(access_count DESC);
CREATE INDEX IF NOT EXISTS idx_storage_files_checksum ON storage_files(checksum);
CREATE INDEX IF NOT EXISTS idx_storage_files_mime_type ON storage_files(mime_type);
CREATE INDEX IF NOT EXISTS idx_storage_files_file_size ON storage_files(file_size);

-- Composite indexes for storage_files
CREATE INDEX IF NOT EXISTS idx_storage_files_owner_type_id ON storage_files(owner_type, owner_id);
CREATE INDEX IF NOT EXISTS idx_storage_files_type_public ON storage_files(storage_type, is_public);
CREATE INDEX IF NOT EXISTS idx_storage_files_owner_storage_type ON storage_files(owner_id, owner_type, storage_type);

-- storage_access_logs indexes
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_file_id ON storage_access_logs(file_id);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_user_id ON storage_access_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_vendor_id ON storage_access_logs(vendor_id);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_access_type ON storage_access_logs(access_type);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_created_at ON storage_access_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_ip_address ON storage_access_logs(ip_address);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_success ON storage_access_logs(success);

-- Composite indexes for storage_access_logs
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_file_access ON storage_access_logs(file_id, access_type);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_user_access ON storage_access_logs(user_id, access_type);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_vendor_access ON storage_access_logs(vendor_id, access_type);
CREATE INDEX IF NOT EXISTS idx_storage_access_logs_access_date ON storage_access_logs(access_type, created_at DESC);

-- storage_quotas indexes
CREATE INDEX IF NOT EXISTS idx_storage_quotas_user_id ON storage_quotas(user_id);
CREATE INDEX IF NOT EXISTS idx_storage_quotas_vendor_id ON storage_quotas(vendor_id);
CREATE INDEX IF NOT EXISTS idx_storage_quotas_quota_type ON storage_quotas(quota_type);
CREATE INDEX IF NOT EXISTS idx_storage_quotas_updated_at ON storage_quotas(updated_at DESC);

-- storage_backups indexes
CREATE INDEX IF NOT EXISTS idx_storage_backups_file_id ON storage_backups(file_id);
CREATE INDEX IF NOT EXISTS idx_storage_backups_backup_type ON storage_backups(backup_type);
CREATE INDEX IF NOT EXISTS idx_storage_backups_created_at ON storage_backups(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_storage_backups_expires_at ON storage_backups(expires_at);
CREATE INDEX IF NOT EXISTS idx_storage_backups_is_active ON storage_backups(is_active);

-- storage_signed_urls indexes
CREATE INDEX IF NOT EXISTS idx_storage_signed_urls_file_id ON storage_signed_urls(file_id);
CREATE INDEX IF NOT EXISTS idx_storage_signed_urls_expires_at ON storage_signed_urls(expires_at);
CREATE INDEX IF NOT EXISTS idx_storage_signed_urls_is_active ON storage_signed_urls(is_active);
CREATE INDEX IF NOT EXISTS idx_storage_signed_urls_created_at ON storage_signed_urls(created_at DESC);

-- storage_health_checks indexes
CREATE INDEX IF NOT EXISTS idx_storage_health_checks_check_type ON storage_health_checks(check_type);
CREATE INDEX IF NOT EXISTS idx_storage_health_checks_status ON storage_health_checks(status);
CREATE INDEX IF NOT EXISTS idx_storage_health_checks_checked_at ON storage_health_checks(checked_at DESC);

-- storage_metrics indexes
CREATE INDEX IF NOT EXISTS idx_storage_metrics_metric_type ON storage_metrics(metric_type);
CREATE INDEX IF NOT EXISTS idx_storage_metrics_user_id ON storage_metrics(user_id);
CREATE INDEX IF NOT EXISTS idx_storage_metrics_vendor_id ON storage_metrics(vendor_id);
CREATE INDEX IF NOT EXISTS idx_storage_metrics_recorded_at ON storage_metrics(recorded_at DESC);

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

-- Triggers for updated_at
CREATE TRIGGER trg_storage_files_updated_at
BEFORE UPDATE ON storage_files
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_storage_quotas_updated_at
BEFORE UPDATE ON storage_quotas
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- TRIGGERS FOR ACCESS COUNTING
-- ============================================================================

-- Function to update file access count
CREATE OR REPLACE FUNCTION update_file_access_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE storage_files 
    SET 
        access_count = access_count + 1,
        last_accessed = NOW()
    WHERE id = NEW.file_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for access count updates
CREATE TRIGGER trg_update_file_access_count
AFTER INSERT ON storage_access_logs
FOR EACH ROW EXECUTE FUNCTION update_file_access_count();

-- ============================================================================
-- TRIGGERS FOR QUOTA UPDATES
-- ============================================================================

-- Function to update storage quotas
CREATE OR REPLACE FUNCTION update_storage_quota()
RETURNS TRIGGER AS $$
DECLARE
    quota_user_id UUID;
    quota_vendor_id UUID;
BEGIN
    -- Determine quota tracking entity
    IF NEW.owner_type = 'user' THEN
        quota_user_id := NEW.owner_id;
        quota_vendor_id := NULL;
    ELSIF NEW.owner_type = 'vendor' THEN
        quota_user_id := NULL;
        quota_vendor_id := NEW.owner_id;
    END IF;
    
    -- Update quota counts
    IF TG_OP = 'INSERT' THEN
        INSERT INTO storage_quotas (
            user_id, vendor_id, quota_type,
            current_file_count, current_total_size
        )
        VALUES (
            quota_user_id, quota_vendor_id, 'total',
            1, NEW.file_size
        )
        ON CONFLICT (user_id, quota_type) 
        DO UPDATE SET
            current_file_count = storage_quotas.current_file_count + 1,
            current_total_size = storage_quotas.current_total_size + NEW.file_size,
            updated_at = NOW()
        ON CONFLICT (vendor_id, quota_type)
        DO UPDATE SET
            current_file_count = storage_quotas.current_file_count + 1,
            current_total_size = storage_quotas.current_total_size + NEW.file_size,
            updated_at = NOW();
            
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE storage_quotas 
        SET 
            current_file_count = GREATEST(0, current_file_count - 1),
            current_total_size = GREATEST(0, current_total_size - OLD.file_size),
            updated_at = NOW()
        WHERE (user_id = quota_user_id AND quota_type = 'total') OR
              (vendor_id = quota_vendor_id AND quota_type = 'total');
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger for quota updates
CREATE TRIGGER trg_update_storage_quota
AFTER INSERT OR DELETE ON storage_files
FOR EACH ROW EXECUTE FUNCTION update_storage_quota();

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================

-- Function to create a storage file record
CREATE OR REPLACE FUNCTION create_storage_file(
    p_file_path TEXT,
    p_file_name TEXT,
    p_file_size BIGINT,
    p_mime_type TEXT,
    p_checksum TEXT,
    p_storage_type TEXT,
    p_owner_id UUID,
    p_owner_type TEXT,
    p_is_public BOOLEAN DEFAULT false,
    p_metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    file_id UUID;
BEGIN
    INSERT INTO storage_files (
        file_path, file_name, file_size, mime_type, checksum,
        storage_type, owner_id, owner_type, is_public, metadata
    )
    VALUES (
        p_file_path, p_file_name, p_file_size, p_mime_type, p_checksum,
        p_storage_type, p_owner_id, p_owner_type, p_is_public, p_metadata
    )
    RETURNING id INTO file_id;
    
    RETURN file_id;
END;
$$ LANGUAGE plpgsql;

-- Function to record file access
CREATE OR REPLACE FUNCTION record_file_access(
    p_file_id UUID,
    p_user_id UUID,
    p_vendor_id UUID,
    p_access_type TEXT,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_session_id TEXT DEFAULT NULL,
    p_signed_url TEXT DEFAULT NULL,
    p_success BOOLEAN DEFAULT true,
    p_error_message TEXT DEFAULT NULL,
    p_response_time_ms INTEGER DEFAULT NULL,
    p_metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    access_id UUID;
BEGIN
    INSERT INTO storage_access_logs (
        file_id, user_id, vendor_id, access_type, ip_address, user_agent,
        session_id, signed_url, success, error_message, response_time_ms, metadata
    )
    VALUES (
        p_file_id, p_user_id, p_vendor_id, p_access_type, p_ip_address, p_user_agent,
        p_session_id, p_signed_url, p_success, p_error_message, p_response_time_ms, p_metadata
    )
    RETURNING id INTO access_id;
    
    RETURN access_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a signed URL record
CREATE OR REPLACE FUNCTION create_signed_url(
    p_file_id UUID,
    p_signed_url TEXT,
    p_access_type TEXT,
    p_expires_at TIMESTAMPTZ,
    p_created_by UUID DEFAULT NULL,
    p_max_usage INTEGER DEFAULT 1,
    p_metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    url_id UUID;
BEGIN
    INSERT INTO storage_signed_urls (
        file_id, signed_url, access_type, expires_at, created_by, max_usage, metadata
    )
    VALUES (
        p_file_id, p_signed_url, p_access_type, p_expires_at, p_created_by, p_max_usage, p_metadata
    )
    RETURNING id INTO url_id;
    
    RETURN url_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get storage quota status
CREATE OR REPLACE FUNCTION get_storage_quota_status(p_user_id UUID DEFAULT NULL, p_vendor_id UUID DEFAULT NULL)
RETURNS TABLE (
    quota_type TEXT,
    current_files INTEGER,
    max_files INTEGER,
    current_size BIGINT,
    max_size BIGINT,
    usage_percent DECIMAL(5,2),
    remaining_files INTEGER,
    remaining_size BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        q.quota_type,
        q.current_file_count,
        q.max_files,
        q.current_total_size,
        q.max_total_size,
        CASE 
            WHEN q.max_total_size > 0 THEN (q.current_total_size::DECIMAL / q.max_total_size::DECIMAL) * 100
            ELSE 0
        END as usage_percent,
        GREATEST(0, q.max_files - q.current_file_count) as remaining_files,
        GREATEST(0, q.max_total_size - q.current_total_size) as remaining_size
    FROM storage_quotas q
    WHERE (p_user_id IS NULL OR q.user_id = p_user_id)
    AND (p_vendor_id IS NULL OR q.vendor_id = p_vendor_id)
    ORDER BY q.quota_type;
END;
$$ LANGUAGE plpgsql;

-- Function to get storage statistics
CREATE OR REPLACE FUNCTION get_storage_stats(
    p_user_id UUID DEFAULT NULL,
    p_vendor_id UUID DEFAULT NULL,
    p_storage_type TEXT DEFAULT NULL
) RETURNS TABLE (
    total_files BIGINT,
    total_size BIGINT,
    user_files BIGINT,
    cloth_files BIGINT,
    result_files BIGINT,
    public_files BIGINT,
    private_files BIGINT,
    backed_up_files BIGINT,
    average_file_size NUMERIC,
    most_accessed_file TEXT,
    oldest_file TIMESTAMPTZ,
    newest_file TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_files,
        COALESCE(SUM(sf.file_size), 0) as total_size,
        COUNT(*) FILTER (WHERE sf.storage_type = 'user') as user_files,
        COUNT(*) FILTER (WHERE sf.storage_type = 'cloth') as cloth_files,
        COUNT(*) FILTER (WHERE sf.storage_type = 'result') as result_files,
        COUNT(*) FILTER (WHERE sf.is_public = true) as public_files,
        COUNT(*) FILTER (WHERE sf.is_public = false) as private_files,
        COUNT(*) FILTER (WHERE sf.is_backed_up = true) as backed_up_files,
        COALESCE(AVG(sf.file_size), 0) as average_file_size,
        (SELECT file_name FROM storage_files WHERE access_count = (SELECT MAX(access_count) FROM storage_files) LIMIT 1) as most_accessed_file,
        MIN(sf.created_at) as oldest_file,
        MAX(sf.created_at) as newest_file
    FROM storage_files sf
    WHERE (p_user_id IS NULL OR sf.owner_id = p_user_id)
    AND (p_vendor_id IS NULL OR sf.owner_id = p_vendor_id)
    AND (p_storage_type IS NULL OR sf.storage_type = p_storage_type);
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup expired signed URLs
CREATE OR REPLACE FUNCTION cleanup_expired_signed_urls()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    UPDATE storage_signed_urls 
    SET is_active = false 
    WHERE expires_at < NOW() AND is_active = true;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup old access logs
CREATE OR REPLACE FUNCTION cleanup_old_access_logs(p_days_to_keep INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM storage_access_logs 
    WHERE created_at < NOW() - INTERVAL '1 day' * p_days_to_keep;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup expired backups
CREATE OR REPLACE FUNCTION cleanup_expired_backups()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM storage_backups 
    WHERE expires_at < NOW() AND is_active = true;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL DATA SETUP
-- ============================================================================

-- Insert default quotas for existing users
INSERT INTO storage_quotas (user_id, quota_type, max_file_size, max_files, max_total_size)
SELECT 
    id, 
    'total',
    52428800, -- 50MB
    100,      -- 100 files
    1073741824 -- 1GB
FROM users
WHERE NOT EXISTS (
    SELECT 1 FROM storage_quotas WHERE user_id = users.id AND quota_type = 'total'
);

-- Insert default quotas for existing vendors
INSERT INTO storage_quotas (vendor_id, quota_type, max_file_size, max_files, max_total_size)
SELECT 
    id, 
    'total',
    52428800, -- 50MB
    1000,     -- 1000 files
    5368709120 -- 5GB
FROM vendors
WHERE NOT EXISTS (
    SELECT 1 FROM storage_quotas WHERE vendor_id = vendors.id AND quota_type = 'total'
);

COMMIT;
