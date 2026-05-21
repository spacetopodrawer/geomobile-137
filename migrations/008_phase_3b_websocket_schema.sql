-- Phase 3B: WebSocket Real-Time Updates & Conflict Visualization
-- Migration 008: Database schema for real-time synchronization

-- WebSocket session tracking
CREATE TABLE IF NOT EXISTS websocket_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    ip_address INET NOT NULL,
    connected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disconnected_at TIMESTAMP,
    total_messages_sent INT DEFAULT 0,
    total_messages_received INT DEFAULT 0,
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    viewport_min_x FLOAT8,
    viewport_max_x FLOAT8,
    viewport_min_y FLOAT8,
    viewport_max_y FLOAT8,
    status VARCHAR(50) DEFAULT 'active', -- active, idle, disconnected
    CONSTRAINT websocket_sessions_user_fk FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_websocket_sessions_device ON websocket_sessions(device_id);
CREATE INDEX idx_websocket_sessions_user ON websocket_sessions(user_id);
CREATE INDEX idx_websocket_sessions_status ON websocket_sessions(status);
CREATE INDEX idx_websocket_sessions_connected ON websocket_sessions(connected_at);

-- Immutable transaction log for temporal replay and audit trail
CREATE TABLE IF NOT EXISTS transaction_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    operation VARCHAR(50) NOT NULL, -- CREATE, UPDATE, DELETE
    before_state JSONB,
    after_state JSONB,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    conflict_resolved_by VARCHAR(50), -- Strategy: last_write_wins, custom_rule, user_choice
    transaction_hash VARCHAR(64), -- SHA256 of the transaction for integrity
    CONSTRAINT transaction_log_parcel_fk FOREIGN KEY (parcel_id) REFERENCES parcels(parcel_id),
    CONSTRAINT transaction_log_user_fk FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_transaction_log_parcel ON transaction_log(parcel_id);
CREATE INDEX idx_transaction_log_timestamp ON transaction_log(timestamp);
CREATE INDEX idx_transaction_log_parcel_timestamp ON transaction_log(parcel_id, timestamp);
CREATE INDEX idx_transaction_log_device ON transaction_log(device_id);
CREATE INDEX idx_transaction_log_user ON transaction_log(user_id);
CREATE INDEX idx_transaction_log_operation ON transaction_log(operation);

-- Conflict resolution history
CREATE TABLE IF NOT EXISTS conflict_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id VARCHAR(255) NOT NULL,
    device_1 VARCHAR(255) NOT NULL,
    device_2 VARCHAR(255) NOT NULL,
    user_1_id UUID,
    user_2_id UUID,
    edit_1 JSONB NOT NULL,
    edit_2 JSONB NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    winning_edit JSONB,
    winning_device VARCHAR(255),
    strategy VARCHAR(50) NOT NULL, -- last_write_wins, custom_rule, user_choice
    resolved_by_user_id UUID,
    resolution_notes TEXT,
    CONSTRAINT conflict_log_parcel_fk FOREIGN KEY (parcel_id) REFERENCES parcels(parcel_id),
    CONSTRAINT conflict_log_user1_fk FOREIGN KEY (user_1_id) REFERENCES users(id),
    CONSTRAINT conflict_log_user2_fk FOREIGN KEY (user_2_id) REFERENCES users(id),
    CONSTRAINT conflict_log_resolver_fk FOREIGN KEY (resolved_by_user_id) REFERENCES users(id)
);

CREATE INDEX idx_conflict_log_parcel ON conflict_log(parcel_id);
CREATE INDEX idx_conflict_log_timestamp ON conflict_log(timestamp);
CREATE INDEX idx_conflict_log_resolved ON conflict_log(resolved_at);
CREATE INDEX idx_conflict_log_strategy ON conflict_log(strategy);

-- Parcel edit history with temporal tracking
CREATE TABLE IF NOT EXISTS parcel_edit_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    edit_sequence INT NOT NULL, -- Logical clock for ordering
    before_geometry GEOMETRY,
    after_geometry GEOMETRY,
    before_attributes JSONB,
    after_attributes JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT parcel_edit_history_parcel_fk FOREIGN KEY (parcel_id) REFERENCES parcels(parcel_id),
    CONSTRAINT parcel_edit_history_user_fk FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_parcel_edit_history_parcel ON parcel_edit_history(parcel_id);
CREATE INDEX idx_parcel_edit_history_user ON parcel_edit_history(user_id);
CREATE INDEX idx_parcel_edit_history_created ON parcel_edit_history(created_at);
CREATE INDEX idx_parcel_edit_history_sequence ON parcel_edit_history(parcel_id, edit_sequence);

-- Real-time presence tracking (periodically updated)
CREATE TABLE IF NOT EXISTS user_presence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    viewport_min_x FLOAT8,
    viewport_max_x FLOAT8,
    viewport_min_y FLOAT8,
    viewport_max_y FLOAT8,
    last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active', -- active, idle, away
    CONSTRAINT user_presence_user_fk FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_user_presence_user ON user_presence(user_id);
CREATE INDEX idx_user_presence_device ON user_presence(device_id);
CREATE INDEX idx_user_presence_status ON user_presence(status);

-- Function to calculate transaction hash (for integrity verification)
CREATE OR REPLACE FUNCTION calculate_transaction_hash(
    parcel_id_param VARCHAR,
    device_id_param VARCHAR,
    before_state JSONB,
    after_state JSONB,
    timestamp_param TIMESTAMP
) RETURNS VARCHAR AS $$
BEGIN
    RETURN encode(
        digest(
            parcel_id_param || device_id_param ||
            COALESCE(before_state::TEXT, '') ||
            COALESCE(after_state::TEXT, '') ||
            timestamp_param::TEXT,
            'sha256'
        ),
        'hex'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to get transaction timeline for a parcel
CREATE OR REPLACE FUNCTION get_parcel_transaction_timeline(
    parcel_id_param VARCHAR,
    start_time TIMESTAMP,
    end_time TIMESTAMP
) RETURNS TABLE (
    transaction_id UUID,
    device_id VARCHAR,
    operation VARCHAR,
    timestamp TIMESTAMP,
    conflict BOOLEAN,
    strategy VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        tl.id,
        tl.device_id,
        tl.operation,
        tl.timestamp,
        CASE WHEN cl.id IS NOT NULL THEN TRUE ELSE FALSE END AS conflict,
        COALESCE(cl.strategy, tl.conflict_resolved_by) AS strategy
    FROM transaction_log tl
    LEFT JOIN conflict_log cl ON tl.parcel_id = cl.parcel_id
        AND tl.timestamp >= cl.timestamp
        AND tl.timestamp <= COALESCE(cl.resolved_at, CURRENT_TIMESTAMP)
    WHERE tl.parcel_id = parcel_id_param
        AND tl.timestamp >= start_time
        AND tl.timestamp <= end_time
    ORDER BY tl.timestamp ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to replay parcel to a specific timestamp
CREATE OR REPLACE FUNCTION replay_parcel_state(
    parcel_id_param VARCHAR,
    target_timestamp TIMESTAMP
) RETURNS TABLE (
    parcel_id VARCHAR,
    geometry GEOMETRY,
    attributes JSONB,
    as_of_timestamp TIMESTAMP,
    last_editor_id UUID,
    last_device_id VARCHAR
) AS $$
DECLARE
    final_state JSONB;
    final_geom GEOMETRY;
    final_timestamp TIMESTAMP;
    final_user UUID;
    final_device VARCHAR;
BEGIN
    -- Get the latest state as of target_timestamp
    SELECT
        after_state,
        after_geometry,
        timestamp,
        user_id,
        device_id
    INTO final_state, final_geom, final_timestamp, final_user, final_device
    FROM transaction_log
    WHERE parcel_id = parcel_id_param
        AND timestamp <= target_timestamp
    ORDER BY timestamp DESC
    LIMIT 1;

    -- Return the reconstructed state
    RETURN QUERY SELECT
        parcel_id_param,
        final_geom,
        final_state,
        final_timestamp,
        final_user,
        final_device;
END;
$$ LANGUAGE plpgsql;

-- Function to detect conflicts in a time window
CREATE OR REPLACE FUNCTION detect_conflicts_in_window(
    parcel_id_param VARCHAR,
    window_start TIMESTAMP,
    window_end TIMESTAMP
) RETURNS TABLE (
    conflict_id UUID,
    device_1 VARCHAR,
    device_2 VARCHAR,
    timestamp TIMESTAMP,
    strategy VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        cl.id,
        cl.device_1,
        cl.device_2,
        cl.timestamp,
        cl.strategy
    FROM conflict_log cl
    WHERE cl.parcel_id = parcel_id_param
        AND cl.timestamp >= window_start
        AND cl.timestamp <= window_end
    ORDER BY cl.timestamp DESC;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-hash transactions
CREATE OR REPLACE FUNCTION trigger_set_transaction_hash()
RETURNS TRIGGER AS $$
BEGIN
    NEW.transaction_hash := calculate_transaction_hash(
        NEW.parcel_id,
        NEW.device_id,
        NEW.before_state,
        NEW.after_state,
        NEW.timestamp
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_transaction_log_hash
BEFORE INSERT ON transaction_log
FOR EACH ROW
EXECUTE FUNCTION trigger_set_transaction_hash();

-- Trigger to clean up stale sessions
CREATE OR REPLACE FUNCTION trigger_cleanup_stale_sessions()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE user_presence
    SET status = 'away'
    WHERE last_update < CURRENT_TIMESTAMP - INTERVAL '5 minutes'
        AND status = 'active';
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- View: Active sessions dashboard
CREATE OR REPLACE VIEW v_active_sessions AS
SELECT
    ws.device_id,
    ws.user_id,
    u.username,
    ws.status,
    ws.connected_at,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - ws.connected_at)) as connected_seconds,
    ws.total_messages_sent,
    ws.total_messages_received,
    ws.last_heartbeat,
    CASE WHEN ws.last_heartbeat < CURRENT_TIMESTAMP - INTERVAL '30 seconds'
         THEN 'STALE' ELSE 'HEALTHY' END as health_status
FROM websocket_sessions ws
LEFT JOIN users u ON ws.user_id = u.id
WHERE ws.status = 'active'
ORDER BY ws.connected_at DESC;

-- View: Conflict statistics
CREATE OR REPLACE VIEW v_conflict_statistics AS
SELECT
    parcel_id,
    COUNT(*) as total_conflicts,
    COUNT(CASE WHEN strategy = 'last_write_wins' THEN 1 END) as lww_count,
    COUNT(CASE WHEN strategy = 'user_choice' THEN 1 END) as user_choice_count,
    MAX(timestamp) as latest_conflict,
    COUNT(DISTINCT device_1) as devices_involved
FROM conflict_log
GROUP BY parcel_id
ORDER BY total_conflicts DESC;

-- View: Transaction audit trail
CREATE OR REPLACE VIEW v_transaction_audit AS
SELECT
    parcel_id,
    operation,
    COUNT(*) as operation_count,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT device_id) as unique_devices,
    MIN(timestamp) as first_transaction,
    MAX(timestamp) as latest_transaction
FROM transaction_log
GROUP BY parcel_id, operation
ORDER BY latest_transaction DESC;

-- Grant permissions (adjust as needed for your roles)
-- GRANT SELECT ON v_active_sessions TO app_role;
-- GRANT SELECT ON v_conflict_statistics TO app_role;
-- GRANT SELECT ON v_transaction_audit TO app_role;

-- Migration complete
COMMENT ON TABLE transaction_log IS 'Immutable audit trail of all parcel edits with temporal replay capability';
COMMENT ON TABLE conflict_log IS 'Historical record of detected conflicts and their resolutions';
COMMENT ON TABLE websocket_sessions IS 'Active WebSocket client session tracking';
COMMENT ON FUNCTION get_parcel_transaction_timeline IS 'Returns chronological transaction timeline for temporal replay';
COMMENT ON FUNCTION replay_parcel_state IS 'Reconstructs parcel state at any historical timestamp';
