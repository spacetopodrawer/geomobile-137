-- PHASE 2A: Multi-Device Synchronization Schema
-- Created: 2026-05-17
-- Purpose: Enable hierarchical sync, conflict resolution, offline support

-- Extend devices table with sync capabilities
ALTER TABLE devices ADD COLUMN IF NOT EXISTS (
    sync_enabled BOOLEAN DEFAULT true,
    last_sync_timestamp BIGINT,
    sync_group_id UUID,
    device_fingerprint VARCHAR(255) UNIQUE,
    vector_clock JSONB DEFAULT '{"clock": 0}'::jsonb
);

-- Device synchronization state
CREATE TABLE IF NOT EXISTS device_sync_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL UNIQUE,
    last_sync_timestamp BIGINT DEFAULT 0,
    vector_clock JSONB DEFAULT '{"clock": 0}'::jsonb,
    pending_changes_count INTEGER DEFAULT 0,
    sync_status VARCHAR(50) DEFAULT 'idle',
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Offline synchronization queue (for changes made while offline)
CREATE TABLE IF NOT EXISTS sync_queue (
    id SERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    parcel_id UUID NOT NULL,
    operation VARCHAR(50) NOT NULL, -- 'CREATE', 'UPDATE', 'DELETE'
    payload JSONB NOT NULL,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    synced_at BIGINT,
    sync_attempt_count INTEGER DEFAULT 0,
    last_error VARCHAR(500),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (parcel_id) REFERENCES parcels(id) ON DELETE CASCADE
);

-- Conflict resolution log
CREATE TABLE IF NOT EXISTS conflict_log (
    id SERIAL PRIMARY KEY,
    parcel_id UUID NOT NULL,
    device_1_id UUID NOT NULL,
    device_2_id UUID NOT NULL,
    conflict_type VARCHAR(100) NOT NULL, -- 'concurrent_update', 'concurrent_delete', etc.
    device_1_value JSONB,
    device_2_value JSONB,
    resolved_value JSONB,
    resolution_strategy VARCHAR(50) NOT NULL, -- 'LWW', 'manual', 'merge'
    resolved_timestamp BIGINT,
    resolved_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (parcel_id) REFERENCES parcels(id) ON DELETE CASCADE,
    FOREIGN KEY (device_1_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (device_2_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Device pairing/approval workflow
CREATE TABLE IF NOT EXISTS device_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID UNIQUE,
    device_fingerprint VARCHAR(255) NOT NULL,
    device_name VARCHAR(255),
    requested_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    approved_at BIGINT,
    approved_by_user_id UUID,
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    rejection_reason VARCHAR(500),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL,
    FOREIGN KEY (approved_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Sync statistics for monitoring
CREATE TABLE IF NOT EXISTS sync_statistics (
    id SERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    period_start BIGINT,
    period_end BIGINT,
    total_changes_synced INTEGER DEFAULT 0,
    conflicts_resolved INTEGER DEFAULT 0,
    sync_failures INTEGER DEFAULT 0,
    avg_sync_latency_ms FLOAT DEFAULT 0,
    max_sync_latency_ms INTEGER DEFAULT 0,
    offline_duration_seconds INTEGER DEFAULT 0,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_device_sync_state_device_id ON device_sync_state(device_id);
CREATE INDEX IF NOT EXISTS idx_sync_queue_device_id ON sync_queue(device_id);
CREATE INDEX IF NOT EXISTS idx_sync_queue_synced_at ON sync_queue(synced_at);
CREATE INDEX IF NOT EXISTS idx_conflict_log_parcel_id ON conflict_log(parcel_id);
CREATE INDEX IF NOT EXISTS idx_conflict_log_resolved_at ON conflict_log(resolved_at);
CREATE INDEX IF NOT EXISTS idx_device_approvals_status ON device_approvals(status);

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON device_sync_state TO cadastre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sync_queue TO cadastre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON conflict_log TO cadastre_app;
GRANT SELECT, INSERT, UPDATE ON device_approvals TO cadastre_app;
GRANT SELECT, INSERT ON sync_statistics TO cadastre_app;

-- Function to update vector clock
CREATE OR REPLACE FUNCTION increment_vector_clock(device_id UUID)
RETURNS JSONB AS $$
DECLARE
    current_clock JSONB;
    clock_value INTEGER;
BEGIN
    SELECT vector_clock INTO current_clock FROM device_sync_state WHERE device_id = $1;
    IF current_clock IS NULL THEN
        current_clock := '{"clock": 0}'::jsonb;
    END IF;

    clock_value := (current_clock->'clock')::INTEGER + 1;
    current_clock := jsonb_set(current_clock, '{clock}', to_jsonb(clock_value));

    UPDATE device_sync_state SET vector_clock = current_clock WHERE device_id = $1;
    RETURN current_clock;
END;
$$ LANGUAGE plpgsql;

-- Function to detect conflicts
CREATE OR REPLACE FUNCTION detect_sync_conflict(
    p_parcel_id UUID,
    p_device_1_id UUID,
    p_device_2_id UUID,
    p_device_1_version JSONB,
    p_device_2_version JSONB
)
RETURNS BOOLEAN AS $$
BEGIN
    -- Conflict exists if both devices modified the parcel with different timestamps
    IF (p_device_1_version->>'updated_at')::BIGINT > (p_device_2_version->>'base_updated_at')::BIGINT
    AND (p_device_2_version->>'updated_at')::BIGINT > (p_device_1_version->>'base_updated_at')::BIGINT THEN
        RETURN TRUE;
    END IF;
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- Function to apply LWW (Last-Write-Wins) resolution
CREATE OR REPLACE FUNCTION resolve_lww(
    p_parcel_id UUID,
    p_device_1_value JSONB,
    p_device_2_value JSONB
)
RETURNS JSONB AS $$
DECLARE
    timestamp_1 BIGINT;
    timestamp_2 BIGINT;
BEGIN
    timestamp_1 := (p_device_1_value->>'timestamp')::BIGINT;
    timestamp_2 := (p_device_2_value->>'timestamp')::BIGINT;

    IF timestamp_1 > timestamp_2 THEN
        RETURN p_device_1_value;
    ELSE
        RETURN p_device_2_value;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update sync statistics
CREATE OR REPLACE FUNCTION update_sync_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.synced_at IS NOT NULL AND OLD.synced_at IS NULL THEN
        UPDATE sync_statistics
        SET total_changes_synced = total_changes_synced + 1
        WHERE device_id = NEW.device_id
        AND period_start <= EXTRACT(EPOCH FROM NOW())::BIGINT
        AND period_end >= EXTRACT(EPOCH FROM NOW())::BIGINT;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_queue_stats
AFTER UPDATE ON sync_queue
FOR EACH ROW
EXECUTE FUNCTION update_sync_stats();

-- Initialize sync records for existing devices
INSERT INTO device_sync_state (device_id)
SELECT id FROM devices
ON CONFLICT (device_id) DO NOTHING;

INSERT INTO device_approvals (device_id, device_fingerprint, status, approved_at)
SELECT id, 'EXISTING_' || id::text, 'approved', EXTRACT(EPOCH FROM NOW())::BIGINT FROM devices
ON CONFLICT (device_fingerprint) DO NOTHING;

COMMIT;
