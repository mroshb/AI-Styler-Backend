-- Payment Service Schema Migration
-- Creates payment tables and related functionality

BEGIN;

-- payment_plans table - available subscription plans
CREATE TABLE IF NOT EXISTS payment_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    price_per_month_cents BIGINT NOT NULL DEFAULT 0,
    monthly_conversions_limit INTEGER NOT NULL DEFAULT 0,
    features TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_plans_name ON payment_plans(name);
CREATE INDEX IF NOT EXISTS idx_payment_plans_is_active ON payment_plans(is_active);
CREATE INDEX IF NOT EXISTS idx_payment_plans_price ON payment_plans(price_per_month_cents);

-- Add trigger for payment_plans updated_at
CREATE TRIGGER trg_payment_plans_updated_at
BEFORE UPDATE ON payment_plans
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- payments table - payment transactions
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES payment_plans(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL, -- Amount in cents (Rials)
    currency TEXT NOT NULL DEFAULT 'IRR',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed', 'cancelled', 'expired')),
    payment_method TEXT NOT NULL DEFAULT 'zarinpal',
    gateway TEXT NOT NULL DEFAULT 'zarinpal',
    gateway_track_id TEXT,
    gateway_ref_number TEXT,
    gateway_card_number TEXT,
    description TEXT,
    callback_url TEXT NOT NULL,
    return_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_plan_id ON payments(plan_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_gateway_track_id ON payments(gateway_track_id);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_expires_at ON payments(expires_at);

-- Add trigger for payments updated_at
CREATE TRIGGER trg_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- payment_history table - track payment status changes
CREATE TABLE IF NOT EXISTS payment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status_from TEXT,
    status_to TEXT NOT NULL,
    gateway_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_history_payment_id ON payment_history(payment_id);
CREATE INDEX IF NOT EXISTS idx_payment_history_user_id ON payment_history(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_history_created_at ON payment_history(created_at DESC);

-- Insert default payment plans
INSERT INTO payment_plans (id, name, display_name, description, price_per_month_cents, monthly_conversions_limit, features, is_active) VALUES
    ('00000000-0000-0000-0000-000000000001', 'free', 'Free Plan', 'Basic free plan with limited conversions', 0, 2, ARRAY['2 free conversions per month', 'Basic support'], true),
    ('00000000-0000-0000-0000-000000000002', 'basic', 'Basic Plan', 'Basic paid plan with more conversions', 50000, 20, ARRAY['20 conversions per month', 'Email support', 'Priority processing'], true),
    ('00000000-0000-0000-0000-000000000003', 'advanced', 'Advanced Plan', 'Advanced plan with unlimited conversions', 150000, 100, ARRAY['100 conversions per month', 'Priority support', 'Fast processing', 'Advanced features'], true)
ON CONFLICT (name) DO NOTHING;

-- Function to create payment
CREATE OR REPLACE FUNCTION create_payment(
    p_user_id UUID,
    p_plan_id UUID,
    p_amount BIGINT,
    p_currency TEXT,
    p_payment_method TEXT,
    p_gateway TEXT,
    p_description TEXT,
    p_callback_url TEXT,
    p_return_url TEXT,
    p_expires_at TIMESTAMPTZ
) RETURNS UUID AS $$
DECLARE
    payment_id UUID;
BEGIN
    -- Generate payment ID
    payment_id := gen_random_uuid();
    
    -- Insert payment record
    INSERT INTO payments (
        id, user_id, plan_id, amount, currency, payment_method, 
        gateway, description, callback_url, return_url, expires_at
    ) VALUES (
        payment_id, p_user_id, p_plan_id, p_amount, p_currency, p_payment_method,
        p_gateway, p_description, p_callback_url, p_return_url, p_expires_at
    );
    
    -- Log payment creation
    INSERT INTO payment_history (payment_id, user_id, status_to)
    VALUES (payment_id, p_user_id, 'pending');
    
    RETURN payment_id;
END;
$$ LANGUAGE plpgsql;

-- Function to update payment status
CREATE OR REPLACE FUNCTION update_payment_status(
    p_payment_id UUID,
    p_status TEXT,
    p_gateway_track_id TEXT DEFAULT NULL,
    p_gateway_ref_number TEXT DEFAULT NULL,
    p_gateway_card_number TEXT DEFAULT NULL,
    p_gateway_response JSONB DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    current_status TEXT;
    p_user_id UUID;
BEGIN
    -- Get current status and user ID
    SELECT status, user_id INTO current_status, p_user_id
    FROM payments 
    WHERE id = p_payment_id;
    
    -- Check if payment exists
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Update payment
    UPDATE payments 
    SET 
        status = p_status,
        gateway_track_id = COALESCE(p_gateway_track_id, gateway_track_id),
        gateway_ref_number = COALESCE(p_gateway_ref_number, gateway_ref_number),
        gateway_card_number = COALESCE(p_gateway_card_number, gateway_card_number),
        paid_at = CASE WHEN p_status = 'completed' THEN NOW() ELSE paid_at END,
        updated_at = NOW()
    WHERE id = p_payment_id;
    
    -- Log status change
    INSERT INTO payment_history (payment_id, user_id, status_from, status_to, gateway_response)
    VALUES (p_payment_id, p_user_id, current_status, p_status, p_gateway_response);
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to get user payment summary
CREATE OR REPLACE FUNCTION get_user_payment_summary(p_user_id UUID)
RETURNS TABLE (
    total_payments BIGINT,
    successful_payments BIGINT,
    failed_payments BIGINT,
    total_amount_paid BIGINT,
    last_payment_date TIMESTAMPTZ,
    current_plan_name TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_payments,
        COUNT(*) FILTER (WHERE p.status = 'completed') as successful_payments,
        COUNT(*) FILTER (WHERE p.status = 'failed') as failed_payments,
        COALESCE(SUM(p.amount) FILTER (WHERE p.status = 'completed'), 0) as total_amount_paid,
        MAX(p.created_at) FILTER (WHERE p.status = 'completed') as last_payment_date,
        COALESCE(pp.name, 'free') as current_plan_name
    FROM payments p
    LEFT JOIN user_plans up ON up.user_id = p_user_id AND up.status = 'active'
    LEFT JOIN payment_plans pp ON up.plan_id = pp.id
    WHERE p.user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired payments
CREATE OR REPLACE FUNCTION cleanup_expired_payments()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    -- Update expired payments
    UPDATE payments 
    SET status = 'expired', updated_at = NOW()
    WHERE status = 'pending' 
        AND expires_at IS NOT NULL 
        AND expires_at < NOW();
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    
    -- Log expired payments
    INSERT INTO payment_history (payment_id, user_id, status_from, status_to)
    SELECT id, user_id, 'pending', 'expired'
    FROM payments 
    WHERE status = 'expired' 
        AND updated_at > NOW() - INTERVAL '1 minute';
    
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Create a scheduled job to clean up expired payments (if pg_cron is available)
-- This would typically be set up in the application or via a cron job
-- SELECT cron.schedule('cleanup-expired-payments', '*/5 * * * *', 'SELECT cleanup_expired_payments();');

COMMIT;
