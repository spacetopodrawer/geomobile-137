-- DeviceRadar Module - Database Schema
-- Phase 1 Implementation
-- Created: 2026-05-17

-- Enable PostGIS for geospatial queries
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: intrinsic_ids
-- Stores hardware-based device identifiers
CREATE TABLE IF NOT EXISTS intrinsic_ids (
    id SERIAL PRIMARY KEY,
    uuid VARCHAR(64) UNIQUE NOT NULL,
    hardware_signature VARCHAR(256) NOT NULL,
    manufacturer VARCHAR(100),
    model VARCHAR(100),
    mac_address VARCHAR(17) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_device_signature UNIQUE (manufacturer, model, mac_address)
);

CREATE INDEX idx_intrinsic_uuid ON intrinsic_ids(uuid);
CREATE INDEX idx_intrinsic_mac ON intrinsic_ids(mac_address);

-- Table: device_identities
-- Core device records with ownership information
CREATE TABLE IF NOT EXISTS device_identities (
    id VARCHAR(128) PRIMARY KEY,
    intrinsic_id_uuid VARCHAR(64) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    device_name VARCHAR(255) NOT NULL,
    device_type VARCHAR(50), -- "phone", "tablet", "laptop", "desktop"
    is_trusted BOOLEAN DEFAULT FALSE,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (intrinsic_id_uuid) REFERENCES intrinsic_ids(uuid) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_device_user ON device_identities(user_id);
CREATE INDEX idx_device_intrinsic ON device_identities(intrinsic_id_uuid);
CREATE INDEX idx_device_last_seen ON device_identities(last_seen DESC);
CREATE INDEX idx_device_type ON device_identities(device_type);

-- Table: device_tags
-- User-assigned tags for device identification (e.g., "bedroom-iphone", "work-laptop")
CREATE TABLE IF NOT EXISTS device_tags (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    tag_key VARCHAR(100) NOT NULL,
    tag_value VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    CONSTRAINT unique_device_tag UNIQUE (device_id, tag_key)
);

CREATE INDEX idx_device_tag_device ON device_tags(device_id);
CREATE INDEX idx_device_tag_key ON device_tags(tag_key);

-- Table: location_traces
-- WiFi/BT location scan results with geospatial data
CREATE TABLE IF NOT EXISTS location_traces (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    altitude DECIMAL(10, 2),
    accuracy_meters DECIMAL(10, 2),
    location_point GEOGRAPHY(POINT, 4326),
    source VARCHAR(50), -- "wifi", "bt", "gps", "hybrid"
    wifi_count INTEGER DEFAULT 0,
    bt_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE
);

CREATE INDEX idx_location_device ON location_traces(device_id);
CREATE INDEX idx_location_timestamp ON location_traces(device_id, timestamp DESC);
CREATE INDEX idx_location_point ON location_traces USING GIST(location_point);
CREATE INDEX idx_location_date ON location_traces(DATE(timestamp));

-- Table: wifi_networks
-- WiFi networks detected during location scans
CREATE TABLE IF NOT EXISTS wifi_networks (
    id BIGSERIAL PRIMARY KEY,
    location_trace_id BIGINT NOT NULL,
    ssid VARCHAR(255),
    bssid VARCHAR(17),
    signal_strength INTEGER, -- dBm (-100 to 0)
    channel INTEGER,
    bandwidth VARCHAR(20),
    security VARCHAR(50),
    estimated_distance DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (location_trace_id) REFERENCES location_traces(id) ON DELETE CASCADE
);

CREATE INDEX idx_wifi_location ON wifi_networks(location_trace_id);
CREATE INDEX idx_wifi_bssid ON wifi_networks(bssid);
CREATE INDEX idx_wifi_ssid ON wifi_networks(ssid);

-- Table: bluetooth_devices
-- Bluetooth devices detected during location scans
CREATE TABLE IF NOT EXISTS bluetooth_devices (
    id BIGSERIAL PRIMARY KEY,
    location_trace_id BIGINT NOT NULL,
    address VARCHAR(17),
    name VARCHAR(255),
    signal_strength INTEGER, -- dBm (-100 to 0)
    device_type VARCHAR(50), -- "audio", "wearable", "phone", "accessory"
    estimated_distance DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (location_trace_id) REFERENCES location_traces(id) ON DELETE CASCADE
);

CREATE INDEX idx_bt_location ON bluetooth_devices(location_trace_id);
CREATE INDEX idx_bt_address ON bluetooth_devices(address);

-- Table: movement_history
-- Calculated movements between location traces
CREATE TABLE IF NOT EXISTS movement_history (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    from_location_id BIGINT,
    to_location_id BIGINT NOT NULL,
    distance_meters DECIMAL(10, 2),
    duration_ms BIGINT,
    speed_ms DECIMAL(10, 4),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE,
    FOREIGN KEY (from_location_id) REFERENCES location_traces(id) ON DELETE SET NULL,
    FOREIGN KEY (to_location_id) REFERENCES location_traces(id) ON DELETE CASCADE
);

CREATE INDEX idx_movement_device ON movement_history(device_id);
CREATE INDEX idx_movement_timestamp ON movement_history(device_id, timestamp DESC);
CREATE INDEX idx_movement_distance ON movement_history(distance_meters);

-- Table: environment_signatures
-- RF environment fingerprints for location verification
CREATE TABLE IF NOT EXISTS environment_signatures (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    wifi_fingerprint JSONB, -- { "SSID": signal_strength, ... }
    bt_fingerprint JSONB, -- { "address": signal_strength, ... }
    rf_pattern VARCHAR(256), -- SHA256 hash of environment
    is_home_environment BOOLEAN DEFAULT FALSE,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE
);

CREATE INDEX idx_env_device ON environment_signatures(device_id);
CREATE INDEX idx_env_pattern ON environment_signatures(rf_pattern);

-- Table: vpn_statuses
-- VPN activity and leak detection results
CREATE TABLE IF NOT EXISTS vpn_statuses (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    is_active BOOLEAN DEFAULT FALSE,
    visible_ip INET,
    detected_real_ip INET,
    has_leaks BOOLEAN DEFAULT FALSE,
    leak_types TEXT[], -- ARRAY of leak types: "dns", "webrtc", "ipv6", etc.
    is_split_tunneling BOOLEAN DEFAULT FALSE,
    suspicious_activity BOOLEAN DEFAULT FALSE,
    exit_country VARCHAR(100),
    exit_provider VARCHAR(255),
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE
);

CREATE INDEX idx_vpn_device ON vpn_statuses(device_id);
CREATE INDEX idx_vpn_timestamp ON vpn_statuses(device_id, last_checked DESC);
CREATE INDEX idx_vpn_active ON vpn_statuses(is_active);
CREATE INDEX idx_vpn_leaks ON vpn_statuses(has_leaks);

-- Table: premium_features
-- Premium feature allocations per user
CREATE TABLE IF NOT EXISTS premium_features (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    feature_name VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN DEFAULT FALSE,
    tier VARCHAR(50), -- "free", "basic", "pro", "enterprise"
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT unique_user_feature UNIQUE (user_id, feature_name)
);

CREATE INDEX idx_premium_user ON premium_features(user_id);
CREATE INDEX idx_premium_feature ON premium_features(feature_name);
CREATE INDEX idx_premium_tier ON premium_features(tier);
CREATE INDEX idx_premium_expires ON premium_features(expires_at);

-- Table: device_authenticity_reports
-- Historical authenticity verification records
CREATE TABLE IF NOT EXISTS device_authenticity_reports (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    is_authentic BOOLEAN,
    confidence_score DECIMAL(5, 4), -- 0.00 to 1.00
    checks_passed TEXT[], -- Array of passed check names
    checks_failed TEXT[], -- Array of failed check names
    risk_level VARCHAR(50), -- "low", "medium", "high"
    verified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE
);

CREATE INDEX idx_auth_device ON device_authenticity_reports(device_id);
CREATE INDEX idx_auth_timestamp ON device_authenticity_reports(device_id, verified_at DESC);
CREATE INDEX idx_auth_risk ON device_authenticity_reports(risk_level);

-- Table: suspicious_activities
-- Flagged anomalies and suspicious behavior
CREATE TABLE IF NOT EXISTS suspicious_activities (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    activity_type VARCHAR(100), -- "impossible_speed", "location_jump", "vpn_leak", etc.
    severity VARCHAR(50), -- "low", "medium", "high", "critical"
    description TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device_identities(id) ON DELETE CASCADE
);

CREATE INDEX idx_suspicious_device ON suspicious_activities(device_id);
CREATE INDEX idx_suspicious_type ON suspicious_activities(activity_type);
CREATE INDEX idx_suspicious_severity ON suspicious_activities(severity);
CREATE INDEX idx_suspicious_resolved ON suspicious_activities(resolved);

-- View: active_devices
-- Devices currently online or recently active
CREATE OR REPLACE VIEW active_devices AS
SELECT
    di.id,
    di.user_id,
    di.device_name,
    di.device_type,
    di.last_seen,
    AGE(NOW(), di.last_seen) as time_since_seen,
    CASE
        WHEN di.last_seen > NOW() - INTERVAL '5 minutes' THEN 'online'
        WHEN di.last_seen > NOW() - INTERVAL '1 day' THEN 'recently_active'
        ELSE 'offline'
    END as status,
    COUNT(DISTINCT lt.id) as location_count,
    MAX(lt.timestamp) as last_location
FROM device_identities di
LEFT JOIN location_traces lt ON di.id = lt.device_id
GROUP BY di.id, di.user_id, di.device_name, di.device_type, di.last_seen;

-- View: device_location_summary
-- Latest location info for each device
CREATE OR REPLACE VIEW device_location_summary AS
SELECT
    di.id,
    di.user_id,
    di.device_name,
    lt.id as location_id,
    lt.timestamp,
    lt.latitude,
    lt.longitude,
    lt.accuracy_meters,
    lt.source,
    ROW_NUMBER() OVER (PARTITION BY di.id ORDER BY lt.timestamp DESC) as rn
FROM device_identities di
LEFT JOIN location_traces lt ON di.id = lt.device_id
WHERE ROW_NUMBER() OVER (PARTITION BY di.id ORDER BY lt.timestamp DESC) = 1;

-- Stored Procedure: cleanup_old_data
-- Archive/delete location data older than retention period
CREATE OR REPLACE PROCEDURE cleanup_old_data(retention_days INTEGER DEFAULT 90)
LANGUAGE plpgsql
AS $$
DECLARE
    deleted_traces BIGINT;
    deleted_movements BIGINT;
BEGIN
    -- Delete old location traces (and cascade deletes)
    DELETE FROM location_traces
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS deleted_traces = ROW_COUNT;

    -- Delete old movements
    DELETE FROM movement_history
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS deleted_movements = ROW_COUNT;

    RAISE NOTICE 'Cleanup complete: % traces, % movements deleted', deleted_traces, deleted_movements;
END;
$$;

-- Stored Procedure: calculate_movements
-- Calculate movement records from location traces
CREATE OR REPLACE PROCEDURE calculate_movements(target_device_id VARCHAR(128))
LANGUAGE plpgsql
AS $$
DECLARE
    prev_location RECORD;
    curr_location RECORD;
    distance DECIMAL;
    duration BIGINT;
    speed DECIMAL;
BEGIN
    FOR prev_location, curr_location IN
        SELECT
            lt1.id as from_id, lt2.id as to_id,
            lt1.location_point as from_point, lt2.location_point as to_point,
            lt1.timestamp as from_time, lt2.timestamp as to_time
        FROM location_traces lt1
        JOIN location_traces lt2 ON lt1.device_id = lt2.device_id
            AND lt2.timestamp > lt1.timestamp
        WHERE lt1.device_id = target_device_id
            AND lt2.timestamp = (
                SELECT MIN(timestamp) FROM location_traces
                WHERE device_id = target_device_id
                AND timestamp > lt1.timestamp
            )
    LOOP
        -- Calculate distance in meters
        distance := ST_DistanceSphere(prev_location.from_point, prev_location.to_point);

        -- Calculate duration in milliseconds
        duration := EXTRACT(EPOCH FROM (prev_location.to_time - prev_location.from_time)) * 1000;

        -- Calculate speed (m/s)
        IF duration > 0 THEN
            speed := distance / (duration / 1000.0);
        ELSE
            speed := 0;
        END IF;

        -- Insert movement record if it doesn't exist
        INSERT INTO movement_history (device_id, from_location_id, to_location_id, distance_meters, duration_ms, speed_ms)
        SELECT target_device_id, prev_location.from_id, prev_location.to_id, distance, duration, speed
        WHERE NOT EXISTS (
            SELECT 1 FROM movement_history
            WHERE device_id = target_device_id
            AND from_location_id = prev_location.from_id
            AND to_location_id = prev_location.to_id
        );
    END LOOP;
END;
$$;

-- Grant permissions (adjust as needed)
GRANT SELECT, INSERT, UPDATE, DELETE ON intrinsic_ids TO geomobile_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON device_identities TO geomobile_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON location_traces TO geomobile_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON movement_history TO geomobile_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON vpn_statuses TO geomobile_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON premium_features TO geomobile_app;

-- Migration metadata
COMMENT ON TABLE intrinsic_ids IS 'Hardware-based device identifiers - UUID generation and signature storage';
COMMENT ON TABLE device_identities IS 'Core device records with user ownership and trust status';
COMMENT ON TABLE location_traces IS 'WiFi/BT location scan results with geospatial data';
COMMENT ON TABLE movement_history IS 'Calculated physical movements between locations';
COMMENT ON TABLE vpn_statuses IS 'VPN activity and leak detection results';
COMMENT ON TABLE premium_features IS 'Premium feature allocations and licensing';
