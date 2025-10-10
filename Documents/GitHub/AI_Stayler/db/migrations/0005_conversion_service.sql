-- Conversion Service Schema Migration
-- Creates tables for conversion requests and management

BEGIN;

-- conversions table - track all conversion requests
CREATE TABLE IF NOT EXISTS conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    cloth_image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    result_image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    error_message TEXT,
    processing_time_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Create indexes for conversions table
CREATE INDEX IF NOT EXISTS idx_conversions_user_id ON conversions(user_id);
CREATE INDEX IF NOT EXISTS idx_conversions_created_at ON conversions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversions_status ON conversions(status);
CREATE INDEX IF NOT EXISTS idx_conversions_user_image_id ON conversions(user_image_id);
CREATE INDEX IF NOT EXISTS idx_conversions_cloth_image_id ON conversions(cloth_image_id);
CREATE INDEX IF NOT EXISTS idx_conversions_result_image_id ON conversions(result_image_id);

-- Add trigger for conversions updated_at
CREATE TRIGGER trg_conversions_updated_at
BEFORE UPDATE ON conversions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- conversion_jobs table - track background processing jobs
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

-- Create indexes for conversion_jobs table
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_conversion_id ON conversion_jobs(conversion_id);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_status ON conversion_jobs(status);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_priority ON conversion_jobs(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_worker_id ON conversion_jobs(worker_id);
CREATE INDEX IF NOT EXISTS idx_conversion_jobs_created_at ON conversion_jobs(created_at);

-- Add trigger for conversion_jobs updated_at
CREATE TRIGGER trg_conversion_jobs_updated_at
BEFORE UPDATE ON conversion_jobs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- conversion_metrics table - track conversion performance metrics
CREATE TABLE IF NOT EXISTS conversion_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversion_id UUID NOT NULL REFERENCES conversions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    processing_time_ms INTEGER NOT NULL,
    input_file_size_bytes BIGINT,
    output_file_size_bytes BIGINT,
    success BOOLEAN NOT NULL,
    error_type TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for conversion_metrics table
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_conversion_id ON conversion_metrics(conversion_id);
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_user_id ON conversion_metrics(user_id);
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_created_at ON conversion_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_conversion_metrics_success ON conversion_metrics(success);

-- Function to create a conversion request
CREATE OR REPLACE FUNCTION create_conversion(
    p_user_id UUID,
    p_user_image_id UUID,
    p_cloth_image_id UUID
) RETURNS UUID AS $$
DECLARE
    conversion_id UUID;
    user_quota RECORD;
BEGIN
    -- Check user quota
    SELECT * INTO user_quota FROM get_user_quota_status(p_user_id);
    
    IF user_quota.total_conversions_remaining <= 0 THEN
        RAISE EXCEPTION 'User quota exceeded. Free: %, Paid: %', 
            user_quota.free_conversions_remaining, 
            user_quota.paid_conversions_remaining;
    END IF;
    
    -- Create conversion record
    INSERT INTO conversions (user_id, user_image_id, cloth_image_id)
    VALUES (p_user_id, p_user_image_id, p_cloth_image_id)
    RETURNING id INTO conversion_id;
    
    -- Create conversion job
    INSERT INTO conversion_jobs (conversion_id, priority)
    VALUES (conversion_id, 0);
    
    -- Update user quota (use free conversions first)
    IF user_quota.free_conversions_remaining > 0 THEN
        UPDATE users 
        SET free_conversions_used = free_conversions_used + 1
        WHERE id = p_user_id;
        
        -- Update monthly quota
        INSERT INTO conversion_quotas (user_id, year_month, free_conversions_used, total_conversions_used)
        VALUES (p_user_id, TO_CHAR(NOW(), 'YYYY-MM'), 1, 1)
        ON CONFLICT (user_id, year_month) 
        DO UPDATE SET
            free_conversions_used = conversion_quotas.free_conversions_used + 1,
            total_conversions_used = conversion_quotas.total_conversions_used + 1,
            updated_at = NOW();
    ELSE
        -- Use paid conversions
        UPDATE user_plans 
        SET conversions_used_this_month = conversions_used_this_month + 1
        WHERE user_id = p_user_id AND status = 'active';
        
        -- Update monthly quota
        INSERT INTO conversion_quotas (user_id, year_month, paid_conversions_used, total_conversions_used)
        VALUES (p_user_id, TO_CHAR(NOW(), 'YYYY-MM'), 1, 1)
        ON CONFLICT (user_id, year_month) 
        DO UPDATE SET
            paid_conversions_used = conversion_quotas.paid_conversions_used + 1,
            total_conversions_used = conversion_quotas.total_conversions_used + 1,
            updated_at = NOW();
    END IF;
    
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
        completed_at = CASE WHEN p_status IN ('completed', 'failed') THEN NOW() ELSE completed_at END,
        updated_at = NOW()
    WHERE id = p_conversion_id;
    
    -- Update job status
    UPDATE conversion_jobs 
    SET 
        status = CASE 
            WHEN p_status = 'processing' THEN 'processing'
            WHEN p_status IN ('completed', 'failed') THEN p_status
            ELSE status
        END,
        completed_at = CASE WHEN p_status IN ('completed', 'failed') THEN NOW() ELSE completed_at END,
        updated_at = NOW()
    WHERE conversion_id = p_conversion_id;
    
    -- Record metrics if completed
    IF p_status IN ('completed', 'failed') THEN
        INSERT INTO conversion_metrics (
            conversion_id, 
            user_id, 
            processing_time_ms, 
            success, 
            error_type
        ) VALUES (
            p_conversion_id,
            conversion_record.user_id,
            COALESCE(p_processing_time_ms, 0),
            p_status = 'completed',
            CASE WHEN p_status = 'failed' THEN 'conversion_failed' ELSE NULL END
        );
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to get conversion with details
CREATE OR REPLACE FUNCTION get_conversion_with_details(p_conversion_id UUID)
RETURNS TABLE (
    id UUID,
    user_id UUID,
    user_image_id UUID,
    cloth_image_id UUID,
    status TEXT,
    result_image_id UUID,
    error_message TEXT,
    processing_time_ms INTEGER,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    user_image_url TEXT,
    cloth_image_url TEXT,
    result_image_url TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c.id,
        c.user_id,
        c.user_image_id,
        c.cloth_image_id,
        c.status,
        c.result_image_id,
        c.error_message,
        c.processing_time_ms,
        c.created_at,
        c.updated_at,
        c.completed_at,
        ui.original_url as user_image_url,
        ci.original_url as cloth_image_url,
        ri.original_url as result_image_url
    FROM conversions c
    LEFT JOIN images ui ON c.user_image_id = ui.id
    LEFT JOIN images ci ON c.cloth_image_id = ci.id
    LEFT JOIN images ri ON c.result_image_id = ri.id
    WHERE c.id = p_conversion_id;
END;
$$ LANGUAGE plpgsql;

COMMIT;
