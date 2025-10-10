-- User Service Schema Migration
-- Extends the existing auth schema with user profiles, conversions, and plans

BEGIN;

-- Extend users table with profile fields
ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS free_conversions_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS free_conversions_limit INTEGER NOT NULL DEFAULT 2;

-- Create indexes for new profile fields
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
CREATE INDEX IF NOT EXISTS idx_users_free_conversions ON users(free_conversions_used, free_conversions_limit);

-- user_conversions table - track all conversion activities
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

CREATE INDEX IF NOT EXISTS idx_user_conversions_user_id ON user_conversions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_conversions_created_at ON user_conversions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_conversions_status ON user_conversions(status);
CREATE INDEX IF NOT EXISTS idx_user_conversions_type ON user_conversions(conversion_type);

-- user_plans table - track user subscription plans
CREATE TABLE IF NOT EXISTS user_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_name TEXT NOT NULL CHECK (plan_name IN ('free', 'basic', 'premium', 'enterprise')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'expired', 'suspended')),
    monthly_conversions_limit INTEGER NOT NULL DEFAULT 0,
    conversions_used_this_month INTEGER NOT NULL DEFAULT 0,
    price_per_month_cents INTEGER NOT NULL DEFAULT 0,
    billing_cycle_start_date DATE,
    billing_cycle_end_date DATE,
    auto_renew BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_plans_user_id ON user_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_user_plans_status ON user_plans(status);
CREATE INDEX IF NOT EXISTS idx_user_plans_plan_name ON user_plans(plan_name);
CREATE INDEX IF NOT EXISTS idx_user_plans_expires_at ON user_plans(expires_at);

-- Add trigger for user_plans updated_at
CREATE TRIGGER trg_user_plans_updated_at
BEFORE UPDATE ON user_plans
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- user_plan_history table - track plan changes and billing
CREATE TABLE IF NOT EXISTS user_plan_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_name TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('created', 'upgraded', 'downgraded', 'cancelled', 'renewed', 'expired')),
    previous_plan_name TEXT,
    price_per_month_cents INTEGER NOT NULL,
    effective_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_plan_history_user_id ON user_plan_history(user_id);
CREATE INDEX IF NOT EXISTS idx_user_plan_history_effective_date ON user_plan_history(effective_date DESC);

-- conversion_quotas table - track monthly conversion usage per user
CREATE TABLE IF NOT EXISTS conversion_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    year_month TEXT NOT NULL, -- Format: YYYY-MM
    free_conversions_used INTEGER NOT NULL DEFAULT 0,
    paid_conversions_used INTEGER NOT NULL DEFAULT 0,
    total_conversions_used INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, year_month)
);

CREATE INDEX IF NOT EXISTS idx_conversion_quotas_user_id ON conversion_quotas(user_id);
CREATE INDEX IF NOT EXISTS idx_conversion_quotas_year_month ON conversion_quotas(year_month);

-- Add trigger for conversion_quotas updated_at
CREATE TRIGGER trg_conversion_quotas_updated_at
BEFORE UPDATE ON conversion_quotas
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Function to get current user quota status
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
        GREATEST(0, u.free_conversions_limit - u.free_conversions_used) as free_conversions_remaining,
        GREATEST(0, COALESCE(up.monthly_conversions_limit, 0) - COALESCE(up.conversions_used_this_month, 0)) as paid_conversions_remaining,
        GREATEST(0, COALESCE(up.monthly_conversions_limit, 0) - COALESCE(up.conversions_used_this_month, 0)) + GREATEST(0, u.free_conversions_limit - u.free_conversions_used) as total_conversions_remaining,
        COALESCE(up.plan_name, 'free') as plan_name,
        COALESCE(up.monthly_conversions_limit, 0) as monthly_limit
    FROM users u
    LEFT JOIN user_plans up ON u.id = up.user_id AND up.status = 'active'
    WHERE u.id = p_user_id;
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

-- Function to record a conversion
CREATE OR REPLACE FUNCTION record_conversion(
    p_user_id UUID,
    p_conversion_type TEXT,
    p_input_file_url TEXT,
    p_style_name TEXT
) RETURNS UUID AS $$
DECLARE
    conversion_id UUID;
    current_month TEXT;
BEGIN
    -- Check if user can convert
    IF NOT can_user_convert(p_user_id, p_conversion_type) THEN
        RAISE EXCEPTION 'User quota exceeded for conversion type: %', p_conversion_type;
    END IF;
    
    -- Get current month
    current_month := TO_CHAR(NOW(), 'YYYY-MM');
    
    -- Create conversion record
    INSERT INTO user_conversions (user_id, conversion_type, input_file_url, style_name)
    VALUES (p_user_id, p_conversion_type, p_input_file_url, p_style_name)
    RETURNING id INTO conversion_id;
    
    -- Update user's free conversion count if it's a free conversion
    IF p_conversion_type = 'free' THEN
        UPDATE users 
        SET free_conversions_used = free_conversions_used + 1
        WHERE id = p_user_id;
    END IF;
    
    -- Update or create monthly quota record
    INSERT INTO conversion_quotas (user_id, year_month, free_conversions_used, paid_conversions_used, total_conversions_used)
    VALUES (
        p_user_id, 
        current_month,
        CASE WHEN p_conversion_type = 'free' THEN 1 ELSE 0 END,
        CASE WHEN p_conversion_type = 'paid' THEN 1 ELSE 0 END,
        1
    )
    ON CONFLICT (user_id, year_month) 
    DO UPDATE SET
        free_conversions_used = conversion_quotas.free_conversions_used + CASE WHEN p_conversion_type = 'free' THEN 1 ELSE 0 END,
        paid_conversions_used = conversion_quotas.paid_conversions_used + CASE WHEN p_conversion_type = 'paid' THEN 1 ELSE 0 END,
        total_conversions_used = conversion_quotas.total_conversions_used + 1,
        updated_at = NOW();
    
    -- Update user plan conversions if it's a paid conversion
    IF p_conversion_type = 'paid' THEN
        UPDATE user_plans 
        SET conversions_used_this_month = conversions_used_this_month + 1
        WHERE user_id = p_user_id AND status = 'active';
    END IF;
    
    RETURN conversion_id;
END;
$$ LANGUAGE plpgsql;

COMMIT;
