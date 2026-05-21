-- ═══════════════════════════════════════════════════════════════════════════
-- PHASE 3B: NOTIFICATIONS, WEBHOOKS & EXTERNAL INTEGRATIONS
-- PostgreSQL VERSION - CORRECTED DATA TYPES (TEXT for FK, not UUID)
-- Date: 2026-05-18
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS notifications (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    device_id TEXT,
    notification_type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    icon_url TEXT,
    parcel_id TEXT,
    related_activity_id UUID,
    action_url TEXT,
    send_push_notification BOOLEAN DEFAULT TRUE,
    send_email BOOLEAN DEFAULT FALSE,
    send_sms BOOLEAN DEFAULT FALSE,
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days'),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS notifications_created_at ON notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS notifications_is_read ON notifications(is_read);
CREATE INDEX IF NOT EXISTS notifications_expires_at ON notifications(expires_at);

CREATE TABLE IF NOT EXISTS push_notification_tokens (
    token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    fcm_token TEXT UNIQUE,
    apns_token TEXT UNIQUE,
    platform TEXT NOT NULL,
    token_valid BOOLEAN DEFAULT TRUE,
    last_validated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS push_notification_tokens_device_id ON push_notification_tokens(device_id);
CREATE INDEX IF NOT EXISTS push_notification_tokens_user_id ON push_notification_tokens(user_id);
CREATE INDEX IF NOT EXISTS push_notification_tokens_token_valid ON push_notification_tokens(token_valid);

CREATE TABLE IF NOT EXISTS notification_preferences (
    preference_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL UNIQUE,
    receive_parcel_comments BOOLEAN DEFAULT TRUE,
    receive_sync_updates BOOLEAN DEFAULT TRUE,
    receive_rtk_alerts BOOLEAN DEFAULT TRUE,
    receive_collaboration_presence BOOLEAN DEFAULT TRUE,
    receive_edit_conflicts BOOLEAN DEFAULT TRUE,
    push_enabled BOOLEAN DEFAULT TRUE,
    email_enabled BOOLEAN DEFAULT FALSE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    batch_notifications BOOLEAN DEFAULT FALSE,
    batch_interval_minutes INTEGER DEFAULT 30,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS notification_preferences_user_id ON notification_preferences(user_id);

CREATE TABLE IF NOT EXISTS webhooks (
    webhook_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    project_id TEXT,
    webhook_url TEXT NOT NULL,
    webhook_secret TEXT NOT NULL,
    webhook_name TEXT,
    webhook_description TEXT,
    subscribe_parcel_created BOOLEAN DEFAULT FALSE,
    subscribe_parcel_updated BOOLEAN DEFAULT FALSE,
    subscribe_parcel_deleted BOOLEAN DEFAULT FALSE,
    subscribe_sync_completed BOOLEAN DEFAULT FALSE,
    subscribe_rtk_status_changed BOOLEAN DEFAULT FALSE,
    subscribe_collaboration_event BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_triggered TIMESTAMP WITH TIME ZONE,
    delivery_count BIGINT DEFAULT 0,
    failed_count BIGINT DEFAULT 0,
    average_latency_ms DECIMAL(10,2),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS webhooks_user_id ON webhooks(user_id);
CREATE INDEX IF NOT EXISTS webhooks_is_active ON webhooks(is_active);
CREATE INDEX IF NOT EXISTS webhooks_webhook_url ON webhooks(webhook_url);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    delivery_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    event_data JSONB NOT NULL,
    delivery_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    attempt_count INTEGER DEFAULT 1,
    response_status_code INTEGER,
    response_body TEXT,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    is_delivered BOOLEAN DEFAULT FALSE,
    delivery_time_ms INTEGER,
    FOREIGN KEY (webhook_id) REFERENCES webhooks(webhook_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS webhook_deliveries_delivery_timestamp ON webhook_deliveries(delivery_timestamp DESC);
CREATE INDEX IF NOT EXISTS webhook_deliveries_is_delivered ON webhook_deliveries(is_delivered);
CREATE INDEX IF NOT EXISTS webhook_deliveries_next_retry_at ON webhook_deliveries(next_retry_at);

CREATE TABLE IF NOT EXISTS email_queue (
    email_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    notification_id UUID,
    recipient_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,
    status TEXT DEFAULT 'queued',
    sent_at TIMESTAMP WITH TIME ZONE,
    opened_at TIMESTAMP WITH TIME ZONE,
    attempt_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '5 minutes'),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (notification_id) REFERENCES notifications(notification_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS email_queue_user_id ON email_queue(user_id);
CREATE INDEX IF NOT EXISTS email_queue_status ON email_queue(status);
CREATE INDEX IF NOT EXISTS email_queue_next_retry_at ON email_queue(next_retry_at);
CREATE INDEX IF NOT EXISTS email_queue_created_at ON email_queue(created_at DESC);

CREATE TABLE IF NOT EXISTS sms_queue (
    sms_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    notification_id UUID,
    recipient_phone TEXT NOT NULL,
    message_text TEXT NOT NULL,
    message_length_chars INTEGER,
    status TEXT DEFAULT 'queued',
    sent_at TIMESTAMP WITH TIME ZONE,
    attempt_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '10 minutes'),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (notification_id) REFERENCES notifications(notification_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS sms_queue_user_id ON sms_queue(user_id);
CREATE INDEX IF NOT EXISTS sms_queue_status ON sms_queue(status);
CREATE INDEX IF NOT EXISTS sms_queue_next_retry_at ON sms_queue(next_retry_at);
CREATE INDEX IF NOT EXISTS sms_queue_created_at ON sms_queue(created_at DESC);

CREATE TABLE IF NOT EXISTS audit_log (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT,
    device_id TEXT,
    action_type TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    action_description TEXT,
    change_details JSONB,
    ip_address INET,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    action_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS audit_log_action_type ON audit_log(action_type);
CREATE INDEX IF NOT EXISTS audit_log_action_timestamp ON audit_log(action_timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_log_resource_type ON audit_log(resource_type);

INSERT INTO public.schema_migrations (version, description)
VALUES (
    '005_phase_3b_notifications_and_webhooks',
    'Phase 3B: Notifications, Webhooks, Email/SMS Queues, Audit Logging & External Integrations'
)
ON CONFLICT DO NOTHING;
