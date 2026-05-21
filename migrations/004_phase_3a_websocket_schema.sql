-- ═══════════════════════════════════════════════════════════════════════════
-- PHASE 3A: REAL-TIME COLLABORATION & WEBSOCKET INFRASTRUCTURE
-- PostgreSQL VERSION - CORRECTED DATA TYPES (TEXT for FK, not UUID)
-- Date: 2026-05-18
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS websocket_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id TEXT NOT NULL,
    user_id TEXT,
    connection_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disconnection_time TIMESTAMP WITH TIME ZONE,
    client_version TEXT,
    platform TEXT,
    browser_user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    session_token TEXT UNIQUE,
    current_parcel_id TEXT,
    current_view_mode TEXT,
    latency_ms INTEGER,
    packet_loss_percent DECIMAL(5,2) DEFAULT 0,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS websocket_sessions_device_id ON websocket_sessions(device_id);
CREATE INDEX IF NOT EXISTS websocket_sessions_user_id ON websocket_sessions(user_id);
CREATE INDEX IF NOT EXISTS websocket_sessions_is_active ON websocket_sessions(is_active);
CREATE INDEX IF NOT EXISTS websocket_sessions_connection_time ON websocket_sessions(connection_time DESC);

CREATE TABLE IF NOT EXISTS collaborative_cursors (
    cursor_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    device_id TEXT NOT NULL,
    user_id TEXT,
    parcel_id TEXT,
    field_path TEXT,
    cursor_position INTEGER,
    cursor_color TEXT,
    cursor_emoji TEXT,
    last_update TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES websocket_sessions(session_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS collaborative_cursors_session_id ON collaborative_cursors(session_id);
CREATE INDEX IF NOT EXISTS collaborative_cursors_parcel_id ON collaborative_cursors(parcel_id);
CREATE INDEX IF NOT EXISTS collaborative_cursors_last_update ON collaborative_cursors(last_update DESC);

CREATE TABLE IF NOT EXISTS realtime_activity_log (
    activity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    device_id TEXT NOT NULL,
    user_id TEXT,
    activity_type TEXT NOT NULL,
    parcel_id TEXT,
    action_description TEXT,
    before_value JSONB,
    after_value JSONB,
    execution_time_ms INTEGER,
    activity_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    broadcast_time TIMESTAMP WITH TIME ZONE,
    is_broadcast BOOLEAN DEFAULT FALSE,
    broadcast_count INTEGER DEFAULT 0,
    FOREIGN KEY (session_id) REFERENCES websocket_sessions(session_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS realtime_activity_log_session_id ON realtime_activity_log(session_id);
CREATE INDEX IF NOT EXISTS realtime_activity_log_parcel_id ON realtime_activity_log(parcel_id);
CREATE INDEX IF NOT EXISTS realtime_activity_log_timestamp ON realtime_activity_log(activity_timestamp DESC);
CREATE INDEX IF NOT EXISTS realtime_activity_log_is_broadcast ON realtime_activity_log(is_broadcast);

CREATE TABLE IF NOT EXISTS parcel_edit_locks (
    lock_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id TEXT NOT NULL UNIQUE,
    session_id UUID NOT NULL,
    device_id TEXT NOT NULL,
    user_id TEXT,
    lock_type TEXT DEFAULT 'exclusive',
    lock_reason TEXT,
    acquired_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 minutes'),
    released_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (session_id) REFERENCES websocket_sessions(session_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS parcel_edit_locks_parcel_id ON parcel_edit_locks(parcel_id);
CREATE INDEX IF NOT EXISTS parcel_edit_locks_session_id ON parcel_edit_locks(session_id);
CREATE INDEX IF NOT EXISTS parcel_edit_locks_is_active ON parcel_edit_locks(is_active);
CREATE INDEX IF NOT EXISTS parcel_edit_locks_expires_at ON parcel_edit_locks(expires_at);

CREATE TABLE IF NOT EXISTS operational_transforms (
    ot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    device_id TEXT NOT NULL,
    user_id TEXT,
    parcel_id TEXT,
    operation_type TEXT,
    operation_path JSONB,
    operation_value JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lamport_clock BIGINT NOT NULL,
    vector_clock JSONB,
    conflicts_resolved INTEGER DEFAULT 0,
    is_committed BOOLEAN DEFAULT FALSE,
    commit_time TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (session_id) REFERENCES websocket_sessions(session_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS operational_transforms_session_id ON operational_transforms(session_id);
CREATE INDEX IF NOT EXISTS operational_transforms_parcel_id ON operational_transforms(parcel_id);
CREATE INDEX IF NOT EXISTS operational_transforms_timestamp ON operational_transforms(timestamp DESC);
CREATE INDEX IF NOT EXISTS operational_transforms_lamport_clock ON operational_transforms(lamport_clock DESC);
CREATE INDEX IF NOT EXISTS operational_transforms_is_committed ON operational_transforms(is_committed);

CREATE TABLE IF NOT EXISTS presence_state (
    presence_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    device_id TEXT NOT NULL,
    user_id TEXT,
    online_status TEXT DEFAULT 'online',
    status_message TEXT,
    current_activity TEXT,
    current_parcel_id TEXT,
    current_field TEXT,
    is_connected BOOLEAN DEFAULT TRUE,
    battery_percent INTEGER,
    network_type TEXT,
    signal_strength INTEGER,
    connected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disconnected_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (session_id) REFERENCES websocket_sessions(session_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS presence_state_session_id ON presence_state(session_id);
CREATE INDEX IF NOT EXISTS presence_state_user_id ON presence_state(user_id);
CREATE INDEX IF NOT EXISTS presence_state_online_status ON presence_state(online_status);
CREATE INDEX IF NOT EXISTS presence_state_last_activity_at ON presence_state(last_activity_at DESC);

CREATE TABLE IF NOT EXISTS broadcast_queue (
    message_id BIGSERIAL PRIMARY KEY,
    message_type TEXT NOT NULL,
    source_session_id UUID,
    target_sessions JSONB,
    payload JSONB NOT NULL,
    priority INTEGER DEFAULT 0,
    target_parcel_id TEXT,
    broadcast_scope TEXT DEFAULT 'parcel',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delivered_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    is_processed BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS broadcast_queue_message_type ON broadcast_queue(message_type);
CREATE INDEX IF NOT EXISTS broadcast_queue_created_at ON broadcast_queue(created_at DESC);
CREATE INDEX IF NOT EXISTS broadcast_queue_is_processed ON broadcast_queue(is_processed);
CREATE INDEX IF NOT EXISTS broadcast_queue_target_parcel_id ON broadcast_queue(target_parcel_id);

CREATE TABLE IF NOT EXISTS collaboration_metrics (
    metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    active_sessions INTEGER,
    total_users INTEGER,
    total_parcels_edited INTEGER,
    total_events_processed BIGINT,
    avg_latency_ms DECIMAL(10,2),
    max_latency_ms INTEGER,
    avg_event_broadcast_time_ms DECIMAL(10,2),
    concurrent_editors_max INTEGER,
    average_session_duration_seconds INTEGER,
    total_conflicts_detected INTEGER,
    conflicts_resolved_ot INTEGER,
    conflicts_resolved_lww INTEGER,
    conflicts_manual_intervention INTEGER,
    UNIQUE(period_start, period_end)
);

CREATE INDEX IF NOT EXISTS collaboration_metrics_period_start ON collaboration_metrics(period_start DESC);

INSERT INTO public.schema_migrations (version, description)
VALUES (
    '004_phase_3a_websocket_schema',
    'Phase 3A: WebSocket, Collaborative Editing, Real-time Presence & Activity Tracking'
)
ON CONFLICT DO NOTHING;
