-- Migration: Notification Service
-- Description: Creates tables for the notification system

-- Create notification types enum
CREATE TYPE notification_type AS ENUM (
    'conversion_started',
    'conversion_completed', 
    'conversion_failed',
    'quota_exhausted',
    'quota_warning',
    'quota_reset',
    'payment_success',
    'payment_failed',
    'plan_activated',
    'plan_expired',
    'system_maintenance',
    'system_error',
    'critical_error',
    'welcome',
    'profile_updated',
    'password_changed'
);

-- Create notification channel enum
CREATE TYPE notification_channel AS ENUM (
    'email',
    'sms',
    'telegram',
    'websocket',
    'push'
);

-- Create notification priority enum
CREATE TYPE notification_priority AS ENUM (
    'low',
    'normal',
    'high',
    'critical'
);

-- Create notification status enum
CREATE TYPE notification_status AS ENUM (
    'pending',
    'sending',
    'sent',
    'delivered',
    'failed',
    'read',
    'expired'
);

-- Create notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type notification_type NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    data JSONB,
    channels notification_channel[] NOT NULL,
    priority notification_priority NOT NULL DEFAULT 'normal',
    status notification_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    scheduled_for TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Create notification_deliveries table
CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel notification_channel NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status notification_status NOT NULL DEFAULT 'pending',
    error_message TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create notification_preferences table
CREATE TABLE notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_enabled BOOLEAN NOT NULL DEFAULT true,
    sms_enabled BOOLEAN NOT NULL DEFAULT false,
    telegram_enabled BOOLEAN NOT NULL DEFAULT false,
    websocket_enabled BOOLEAN NOT NULL DEFAULT true,
    push_enabled BOOLEAN NOT NULL DEFAULT false,
    preferences JSONB NOT NULL DEFAULT '{}',
    quiet_hours_start VARCHAR(5), -- Format: HH:MM
    quiet_hours_end VARCHAR(5),   -- Format: HH:MM
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create notification_templates table
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type notification_type NOT NULL,
    channel notification_channel NOT NULL,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    variables TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(type, channel)
);

-- Create indexes for better performance
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
CREATE INDEX idx_notifications_scheduled_for ON notifications(scheduled_for);

CREATE INDEX idx_notification_deliveries_notification_id ON notification_deliveries(notification_id);
CREATE INDEX idx_notification_deliveries_channel ON notification_deliveries(channel);
CREATE INDEX idx_notification_deliveries_status ON notification_deliveries(status);
CREATE INDEX idx_notification_deliveries_created_at ON notification_deliveries(created_at);
CREATE INDEX idx_notification_deliveries_next_retry_at ON notification_deliveries(next_retry_at);

CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);

CREATE INDEX idx_notification_templates_type_channel ON notification_templates(type, channel);
CREATE INDEX idx_notification_templates_is_active ON notification_templates(is_active);

-- Create function to update notification status
CREATE OR REPLACE FUNCTION update_notification_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Update notification status based on delivery statuses
    UPDATE notifications 
    SET status = CASE 
        WHEN EXISTS (SELECT 1 FROM notification_deliveries WHERE notification_id = NEW.notification_id AND status = 'delivered') 
        THEN 'delivered'::notification_status
        WHEN EXISTS (SELECT 1 FROM notification_deliveries WHERE notification_id = NEW.notification_id AND status = 'sent') 
        THEN 'sent'::notification_status
        WHEN EXISTS (SELECT 1 FROM notification_deliveries WHERE notification_id = NEW.notification_id AND status = 'failed') 
        THEN 'failed'::notification_status
        ELSE 'sending'::notification_status
    END
    WHERE id = NEW.notification_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to update notification status when delivery status changes
CREATE TRIGGER trigger_update_notification_status
    AFTER UPDATE ON notification_deliveries
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_status();

-- Create function to clean up expired notifications
CREATE OR REPLACE FUNCTION cleanup_expired_notifications()
RETURNS void AS $$
BEGIN
    -- Mark expired notifications as expired
    UPDATE notifications 
    SET status = 'expired'::notification_status
    WHERE expires_at IS NOT NULL 
    AND expires_at < NOW() 
    AND status IN ('pending', 'sending');
    
    -- Delete old failed deliveries (older than 30 days)
    DELETE FROM notification_deliveries 
    WHERE status = 'failed' 
    AND created_at < NOW() - INTERVAL '30 days';
    
    -- Delete old read notifications (older than 90 days)
    DELETE FROM notifications 
    WHERE status = 'read' 
    AND read_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Create function to get notification statistics
CREATE OR REPLACE FUNCTION get_notification_stats(time_range INTERVAL DEFAULT INTERVAL '24 hours')
RETURNS TABLE (
    total_sent BIGINT,
    total_delivered BIGINT,
    total_failed BIGINT,
    total_read BIGINT,
    by_channel JSONB,
    by_type JSONB,
    by_priority JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_sent,
        COUNT(CASE WHEN nd.status = 'delivered' THEN 1 END) as total_delivered,
        COUNT(CASE WHEN nd.status = 'failed' THEN 1 END) as total_failed,
        COUNT(CASE WHEN n.status = 'read' THEN 1 END) as total_read,
        (
            SELECT jsonb_object_agg(channel, count)
            FROM (
                SELECT channel::text, COUNT(*) as count
                FROM notification_deliveries nd2
                WHERE nd2.created_at >= NOW() - time_range
                GROUP BY channel
            ) channel_stats
        ) as by_channel,
        (
            SELECT jsonb_object_agg(type, count)
            FROM (
                SELECT n2.type::text, COUNT(*) as count
                FROM notifications n2
                WHERE n2.created_at >= NOW() - time_range
                GROUP BY n2.type
            ) type_stats
        ) as by_type,
        (
            SELECT jsonb_object_agg(priority, count)
            FROM (
                SELECT n3.priority::text, COUNT(*) as count
                FROM notifications n3
                WHERE n3.created_at >= NOW() - time_range
                GROUP BY n3.priority
            ) priority_stats
        ) as by_priority
    FROM notification_deliveries nd
    JOIN notifications n ON nd.notification_id = n.id
    WHERE nd.created_at >= NOW() - time_range;
END;
$$ LANGUAGE plpgsql;

-- Insert default notification templates
INSERT INTO notification_templates (type, channel, subject, body, variables, is_active) VALUES
-- Email templates
('conversion_started', 'email', 'Conversion Started', 
 'Your image conversion has started.\n\nConversion ID: {{.conversionId}}\nStatus: {{.status}}', 
 ARRAY['conversionId', 'status'], true),

('conversion_completed', 'email', 'Conversion Completed', 
 'Your image conversion has completed successfully!\n\nConversion ID: {{.conversionId}}\nResult Image ID: {{.resultImageId}}\nStatus: {{.status}}', 
 ARRAY['conversionId', 'resultImageId', 'status'], true),

('conversion_failed', 'email', 'Conversion Failed', 
 'Your image conversion failed.\n\nConversion ID: {{.conversionId}}\nError: {{.errorMessage}}\nStatus: {{.status}}', 
 ARRAY['conversionId', 'errorMessage', 'status'], true),

('quota_exhausted', 'email', 'Quota Exhausted', 
 'Your {{.quotaType}} quota has been exhausted. Please upgrade your plan to continue.', 
 ARRAY['quotaType'], true),

('quota_warning', 'email', 'Quota Warning', 
 'You have {{.remaining}} {{.quotaType}} conversions remaining this month.', 
 ARRAY['remaining', 'quotaType'], true),

-- SMS templates
('conversion_started', 'sms', '', 
 'Conversion started. ID: {{.conversionId}}', 
 ARRAY['conversionId'], true),

('conversion_completed', 'sms', '', 
 'Conversion completed! ID: {{.conversionId}}', 
 ARRAY['conversionId'], true),

('conversion_failed', 'sms', '', 
 'Conversion failed. ID: {{.conversionId}}', 
 ARRAY['conversionId'], true),

('quota_exhausted', 'sms', '', 
 'Quota exhausted. Upgrade your plan.', 
 ARRAY[], true),

('quota_warning', 'sms', '', 
 '{{.remaining}} {{.quotaType}} conversions remaining.', 
 ARRAY['remaining', 'quotaType'], true),

-- Telegram templates
('conversion_started', 'telegram', '', 
 '*Conversion Started*\n\nYour image conversion has started.\n\n*Conversion ID:* {{.conversionId}}\n*Status:* {{.status}}', 
 ARRAY['conversionId', 'status'], true),

('conversion_completed', 'telegram', '', 
 '*Conversion Completed* ✅\n\nYour image conversion has completed successfully!\n\n*Conversion ID:* {{.conversionId}}\n*Result Image ID:* {{.resultImageId}}\n*Status:* {{.status}}', 
 ARRAY['conversionId', 'resultImageId', 'status'], true),

('conversion_failed', 'telegram', '', 
 '*Conversion Failed* ❌\n\nYour image conversion failed.\n\n*Conversion ID:* {{.conversionId}}\n*Error:* {{.errorMessage}}\n*Status:* {{.status}}', 
 ARRAY['conversionId', 'errorMessage', 'status'], true),

('quota_exhausted', 'telegram', '', 
 '*Quota Exhausted* ⚠️\n\nYour {{.quotaType}} quota has been exhausted.\n\n[Upgrade your plan](/upgrade)', 
 ARRAY['quotaType'], true),

('quota_warning', 'telegram', '', 
 '*Quota Warning* ⚠️\n\nYou have {{.remaining}} {{.quotaType}} conversions remaining this month.', 
 ARRAY['remaining', 'quotaType'], true),

('critical_error', 'telegram', '', 
 '*Critical Error* 🚨\n\n{{.message}}\n\n*Error Type:* {{.errorType}}\n*Timestamp:* {{.timestamp}}', 
 ARRAY['message', 'errorType', 'timestamp'], true);

-- Create default notification preferences for existing users
INSERT INTO notification_preferences (user_id, email_enabled, sms_enabled, telegram_enabled, websocket_enabled, push_enabled, preferences)
SELECT 
    id,
    true,  -- email enabled by default
    false, -- sms disabled by default
    false, -- telegram disabled by default
    true,  -- websocket enabled by default
    false, -- push disabled by default
    '{}'   -- empty preferences (all types enabled by default)
FROM users
WHERE id NOT IN (SELECT user_id FROM notification_preferences);

-- Add comments
COMMENT ON TABLE notifications IS 'Stores all notifications in the system';
COMMENT ON TABLE notification_deliveries IS 'Tracks delivery attempts for each notification channel';
COMMENT ON TABLE notification_preferences IS 'User preferences for notification delivery';
COMMENT ON TABLE notification_templates IS 'Templates for different notification types and channels';

COMMENT ON COLUMN notifications.data IS 'Additional data for the notification in JSON format';
COMMENT ON COLUMN notifications.channels IS 'Array of channels to deliver the notification through';
COMMENT ON COLUMN notification_preferences.preferences IS 'Per-notification-type preferences (type -> enabled)';
COMMENT ON COLUMN notification_preferences.quiet_hours_start IS 'Start time for quiet hours in HH:MM format';
COMMENT ON COLUMN notification_preferences.quiet_hours_end IS 'End time for quiet hours in HH:MM format';
COMMENT ON COLUMN notification_templates.variables IS 'List of template variables that can be used in the template';
