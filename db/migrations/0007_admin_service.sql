-- Admin Service Schema Migration
-- Adds admin-specific functionality and audit trail enhancements

BEGIN;

-- Add is_active column to users table if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- Create index for is_active column
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Add last_login_at column to users table if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Create index for last_login_at column
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at);

-- Add is_active column to vendors table if it doesn't exist
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- Create index for is_active column
CREATE INDEX IF NOT EXISTS idx_vendors_is_active ON vendors(is_active);

-- Add last_login_at column to vendors table (via users table join)
-- This will be handled in the application layer

-- Add subscriber_count column to payment_plans table if it doesn't exist
-- This will be calculated dynamically in the application layer

-- Enhance audit_logs table with additional fields for admin service
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS resource TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS resource_id TEXT;

-- Create indexes for new audit_logs columns
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);

-- Add admin-specific audit log entries
INSERT INTO audit_logs (id, user_id, actor_type, action, resource, resource_id, metadata, created_at) VALUES
    ('00000000-0000-0000-0000-000000000001', NULL, 'system', 'migration', 'admin_service', '0007_admin_service', '{"migration": "admin_service_setup"}', NOW())
ON CONFLICT (id) DO NOTHING;

-- Create function to get admin user statistics
CREATE OR REPLACE FUNCTION get_admin_user_stats()
RETURNS TABLE (
    total_users INTEGER,
    active_users INTEGER,
    total_vendors INTEGER,
    active_vendors INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*)::INTEGER FROM users) as total_users,
        (SELECT COUNT(*)::INTEGER FROM users WHERE is_active = true) as active_users,
        (SELECT COUNT(*)::INTEGER FROM vendors) as total_vendors,
        (SELECT COUNT(*)::INTEGER FROM vendors WHERE is_active = true) as active_vendors;
END;
$$ LANGUAGE plpgsql;

-- Create function to get admin payment statistics
CREATE OR REPLACE FUNCTION get_admin_payment_stats()
RETURNS TABLE (
    total_payments BIGINT,
    total_revenue BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_payments,
        COALESCE(SUM(amount) FILTER (WHERE status = 'completed'), 0) as total_revenue
    FROM payments;
END;
$$ LANGUAGE plpgsql;

-- Create function to get admin conversion statistics
CREATE OR REPLACE FUNCTION get_admin_conversion_stats()
RETURNS TABLE (
    total_conversions INTEGER,
    pending_conversions INTEGER,
    failed_conversions INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_conversions,
        COUNT(*) FILTER (WHERE status = 'pending' OR status = 'processing')::INTEGER as pending_conversions,
        COUNT(*) FILTER (WHERE status = 'failed')::INTEGER as failed_conversions
    FROM user_conversions;
END;
$$ LANGUAGE plpgsql;

-- Create function to get admin image statistics
CREATE OR REPLACE FUNCTION get_admin_image_stats()
RETURNS TABLE (
    total_images INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT COUNT(*)::INTEGER as total_images
    FROM images;
END;
$$ LANGUAGE plpgsql;

-- Create function to get comprehensive admin statistics
CREATE OR REPLACE FUNCTION get_admin_system_stats()
RETURNS TABLE (
    total_users INTEGER,
    active_users INTEGER,
    total_vendors INTEGER,
    active_vendors INTEGER,
    total_conversions INTEGER,
    total_payments BIGINT,
    total_revenue BIGINT,
    total_images INTEGER,
    pending_conversions INTEGER,
    failed_conversions INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*)::INTEGER FROM users) as total_users,
        (SELECT COUNT(*)::INTEGER FROM users WHERE is_active = true) as active_users,
        (SELECT COUNT(*)::INTEGER FROM vendors) as total_vendors,
        (SELECT COUNT(*)::INTEGER FROM vendors WHERE is_active = true) as active_vendors,
        (SELECT COUNT(*)::INTEGER FROM user_conversions) as total_conversions,
        (SELECT COUNT(*) FROM payments) as total_payments,
        (SELECT COALESCE(SUM(amount) FILTER (WHERE status = 'completed'), 0) FROM payments) as total_revenue,
        (SELECT COUNT(*)::INTEGER FROM images) as total_images,
        (SELECT COUNT(*) FILTER (WHERE status = 'pending' OR status = 'processing')::INTEGER FROM user_conversions) as pending_conversions,
        (SELECT COUNT(*) FILTER (WHERE status = 'failed')::INTEGER FROM user_conversions) as failed_conversions;
END;
$$ LANGUAGE plpgsql;

-- Create function to revoke user quota
CREATE OR REPLACE FUNCTION revoke_user_quota(
    p_user_id UUID,
    p_quota_type TEXT,
    p_amount INTEGER,
    p_reason TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    current_quota INTEGER;
BEGIN
    -- Get current quota based on type
    IF p_quota_type = 'free' THEN
        SELECT free_conversions_used INTO current_quota
        FROM users 
        WHERE id = p_user_id;
        
        IF NOT FOUND THEN
            RETURN FALSE;
        END IF;
        
        -- Reduce free conversions used
        UPDATE users 
        SET free_conversions_used = GREATEST(0, free_conversions_used - p_amount)
        WHERE id = p_user_id;
        
    ELSIF p_quota_type = 'paid' THEN
        -- For paid conversions, we would need to update user_plans
        -- This is a simplified implementation
        UPDATE user_plans 
        SET conversions_used_this_month = GREATEST(0, conversions_used_this_month - p_amount)
        WHERE user_id = p_user_id AND status = 'active';
        
        IF NOT FOUND THEN
            RETURN FALSE;
        END IF;
    ELSE
        RETURN FALSE;
    END IF;
    
    -- Log the quota revocation
    INSERT INTO audit_logs (user_id, actor_type, action, resource, resource_id, metadata)
    VALUES (p_user_id, 'admin', 'revoke', 'quota', p_user_id::TEXT, 
            json_build_object('quota_type', p_quota_type, 'amount', p_amount, 'reason', p_reason));
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to revoke vendor quota
CREATE OR REPLACE FUNCTION revoke_vendor_quota(
    p_vendor_id UUID,
    p_quota_type TEXT,
    p_amount INTEGER,
    p_reason TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    current_quota INTEGER;
BEGIN
    -- Get current quota based on type
    IF p_quota_type = 'free' THEN
        SELECT free_images_used INTO current_quota
        FROM vendors 
        WHERE id = p_vendor_id;
        
        IF NOT FOUND THEN
            RETURN FALSE;
        END IF;
        
        -- Reduce free images used
        UPDATE vendors 
        SET free_images_used = GREATEST(0, free_images_used - p_amount)
        WHERE id = p_vendor_id;
        
    ELSE
        RETURN FALSE;
    END IF;
    
    -- Log the quota revocation
    INSERT INTO audit_logs (user_id, actor_type, action, resource, resource_id, metadata)
    VALUES (NULL, 'admin', 'revoke', 'quota', p_vendor_id::TEXT, 
            json_build_object('quota_type', p_quota_type, 'amount', p_amount, 'reason', p_reason));
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to revoke user plan
CREATE OR REPLACE FUNCTION revoke_user_plan(
    p_user_id UUID,
    p_reason TEXT
) RETURNS BOOLEAN AS $$
BEGIN
    -- Update user plan status to cancelled
    UPDATE user_plans 
    SET status = 'cancelled', updated_at = NOW()
    WHERE user_id = p_user_id AND status = 'active';
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Log the plan revocation
    INSERT INTO audit_logs (user_id, actor_type, action, resource, resource_id, metadata)
    VALUES (p_user_id, 'admin', 'revoke', 'plan', p_user_id::TEXT, 
            json_build_object('reason', p_reason));
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to update user last login
CREATE OR REPLACE FUNCTION update_user_last_login(p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE users 
    SET last_login_at = NOW()
    WHERE id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to update last_login_at when user logs in
-- This would be called from the application when a user successfully logs in

COMMIT;
