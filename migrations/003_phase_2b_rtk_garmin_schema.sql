-- PHASE 2B: RTK Positioning & Garmin Integration Schema
-- Created: 2026-05-17
-- Purpose: Support sub-decimeter positioning and Garmin sensor fusion

-- RTK configuration and state
CREATE TABLE IF NOT EXISTS rtk_corrections (
    id SERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    timestamp BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    rtk_state VARCHAR(50) NOT NULL DEFAULT 'DISABLED', -- DISABLED, INITIALIZATION, FLOAT, FIXED, ERROR
    -- Reference station position
    base_latitude DOUBLE PRECISION,
    base_longitude DOUBLE PRECISION,
    base_height DOUBLE PRECISION,
    -- Corrected position
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    height DOUBLE PRECISION,
    -- Accuracy (stddev in meters)
    latitude_stddev REAL,
    longitude_stddev REAL,
    height_stddev REAL,
    -- Quality indicators
    num_satellites INTEGER,
    pdop REAL,
    age_of_differential INTEGER, -- milliseconds
    ratio REAL, -- Ambiguity ratio (>3.0 = confident fix)
    -- RTCM correction stream
    rtcm_message_id VARCHAR(50),
    correction_source VARCHAR(100),
    -- Metadata
    is_fixed BOOLEAN DEFAULT FALSE,
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- RTK session management
CREATE TABLE IF NOT EXISTS rtk_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL UNIQUE,
    rtk_enabled BOOLEAN DEFAULT FALSE,
    ntrip_url VARCHAR(500),
    ntrip_username VARCHAR(100),
    ntrip_password VARCHAR(100),
    ntrip_mount_point VARCHAR(100),
    -- RTK status
    rtk_status VARCHAR(50) DEFAULT 'DISABLED',
    initialization_time BIGINT,
    float_time BIGINT,
    fixed_time BIGINT,
    last_fix_time BIGINT,
    consecutive_fix_count INTEGER DEFAULT 0,
    consecutive_gap_count INTEGER DEFAULT 0,
    -- Reference station
    reference_station_id VARCHAR(100),
    reference_station_distance DOUBLE PRECISION, -- meters
    -- Performance metrics
    avg_correction_latency_ms FLOAT DEFAULT 0,
    max_correction_latency_ms INTEGER DEFAULT 0,
    correction_health_percentage FLOAT DEFAULT 100,
    -- Last correction timestamp
    last_correction_time BIGINT,
    last_position_update BIGINT,
    -- Metadata
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Garmin device pairing and metadata
CREATE TABLE IF NOT EXISTS garmin_pairing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    garmin_serial_number VARCHAR(100) NOT NULL,
    garmin_device_name VARCHAR(255),
    garmin_model VARCHAR(100),
    connection_method VARCHAR(50) NOT NULL, -- 'USB', 'WiFi', 'Bluetooth', 'SIMULATOR'
    paired_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_seen_at BIGINT,
    is_active BOOLEAN DEFAULT FALSE,
    is_connected BOOLEAN DEFAULT FALSE,
    firmware_version VARCHAR(50),
    battery_level REAL,
    battery_status VARCHAR(50),
    -- Connection details
    usb_port VARCHAR(50),
    wifi_ssid VARCHAR(255),
    bluetooth_address VARCHAR(50),
    -- Metadata
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    UNIQUE(device_id, garmin_serial_number)
);

-- Garmin sensor data stream
CREATE TABLE IF NOT EXISTS garmin_sensors (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    garmin_id UUID NOT NULL,
    sensor_type VARCHAR(100) NOT NULL, -- 'GPS', 'BAROMETER', 'COMPASS', 'ACCELEROMETER', 'GYROSCOPE', 'CAMERA'
    timestamp BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    sample_rate_hz REAL,
    -- Raw sensor data (sensor-specific)
    raw_data JSONB NOT NULL,
    -- Processed data
    processed_data JSONB,
    -- Quality indicators
    data_quality VARCHAR(50), -- 'GOOD', 'DEGRADED', 'INVALID'
    is_valid BOOLEAN DEFAULT TRUE,
    -- Metadata
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (garmin_id) REFERENCES garmin_pairing(id) ON DELETE CASCADE
);

-- Fused trajectory (output of Kalman filter)
CREATE TABLE IF NOT EXISTS fused_trajectories (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    timestamp BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    -- Fused position (best estimate)
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    altitude DOUBLE PRECISION,
    -- Velocity (m/s)
    velocity_north REAL,
    velocity_east REAL,
    velocity_up REAL,
    -- Heading and rates
    heading REAL,
    pitch REAL,
    roll REAL,
    -- Covariance matrix (for uncertainty bounds)
    covariance_pos JSONB, -- 3x3 position covariance
    covariance_vel JSONB, -- 3x3 velocity covariance
    covariance_att JSONB, -- 3x3 attitude covariance
    -- Source information
    source_sensors TEXT[], -- ['GPS', 'RTK', 'IMU', 'BAROMETER', 'COMPASS']
    gps_quality VARCHAR(50),
    num_satellites INTEGER,
    -- Metadata
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Kalman filter state (for session persistence)
CREATE TABLE IF NOT EXISTS kalman_filter_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL UNIQUE,
    -- State vector [lat, lon, alt, vx, vy, vz, heading]
    state_vector JSONB NOT NULL DEFAULT '{"lat": 0, "lon": 0, "alt": 0, "vx": 0, "vy": 0, "vz": 0, "heading": 0}'::jsonb,
    -- Covariance matrix (7x7, stored as JSON array)
    covariance_matrix JSONB NOT NULL,
    -- Process noise
    process_noise JSONB NOT NULL,
    -- Last update
    last_update_timestamp BIGINT,
    -- Filter statistics
    innovation_magnitude REAL,
    likelihood REAL,
    -- Metadata
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- RTK correction performance log
CREATE TABLE IF NOT EXISTS rtk_performance_log (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    timestamp BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    rtk_state VARCHAR(50),
    correction_latency_ms INTEGER,
    position_accuracy_cm REAL,
    num_satellites INTEGER,
    pdop REAL,
    consecutive_fixed_seconds INTEGER,
    consecutive_gap_seconds INTEGER,
    -- Metadata
    created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_rtk_corrections_device_id ON rtk_corrections(device_id);
CREATE INDEX IF NOT EXISTS idx_rtk_corrections_timestamp ON rtk_corrections(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_rtk_corrections_is_fixed ON rtk_corrections(is_fixed);
CREATE INDEX IF NOT EXISTS idx_rtk_state_device_id ON rtk_state(device_id);
CREATE INDEX IF NOT EXISTS idx_garmin_pairing_device_id ON garmin_pairing(device_id);
CREATE INDEX IF NOT EXISTS idx_garmin_pairing_active ON garmin_pairing(is_active);
CREATE INDEX IF NOT EXISTS idx_garmin_sensors_device_id ON garmin_sensors(device_id);
CREATE INDEX IF NOT EXISTS idx_garmin_sensors_timestamp ON garmin_sensors(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_garmin_sensors_sensor_type ON garmin_sensors(sensor_type);
CREATE INDEX IF NOT EXISTS idx_fused_trajectories_device_id ON fused_trajectories(device_id);
CREATE INDEX IF NOT EXISTS idx_fused_trajectories_timestamp ON fused_trajectories(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_kalman_filter_state_device_id ON kalman_filter_state(device_id);
CREATE INDEX IF NOT EXISTS idx_rtk_performance_log_device_id ON rtk_performance_log(device_id);
CREATE INDEX IF NOT EXISTS idx_rtk_performance_log_timestamp ON rtk_performance_log(timestamp DESC);

-- Geospatial index (if PostGIS installed)
CREATE INDEX IF NOT EXISTS idx_fused_trajectories_location
    ON fused_trajectories USING GIST(
        ST_Point(longitude, latitude)
    );

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON rtk_corrections TO cadastre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON rtk_state TO cadastre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON garmin_pairing TO cadastre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON garmin_sensors TO cadastre_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON fused_trajectories TO cadastre_app;
GRANT SELECT, INSERT, UPDATE ON kalman_filter_state TO cadastre_app;
GRANT SELECT, INSERT ON rtk_performance_log TO cadastre_app;

-- View for real-time RTK status
CREATE OR REPLACE VIEW rtk_status_view AS
SELECT
    rs.device_id,
    d.device_name,
    rs.rtk_enabled,
    rs.rtk_status,
    rs.ntrip_url,
    rc.num_satellites,
    rc.pdop,
    rc.latitude_stddev,
    rc.longitude_stddev,
    rc.is_fixed,
    EXTRACT(EPOCH FROM NOW())::BIGINT - rs.last_correction_time AS seconds_since_last_correction,
    rs.consecutive_fix_count,
    ROUND((rs.correction_health_percentage)::numeric, 2) AS correction_health_pct
FROM rtk_state rs
LEFT JOIN devices d ON rs.device_id = d.id
LEFT JOIN rtk_corrections rc ON rs.device_id = rc.device_id
    AND rc.id = (SELECT id FROM rtk_corrections WHERE device_id = rs.device_id ORDER BY timestamp DESC LIMIT 1);

-- View for Garmin integration status
CREATE OR REPLACE VIEW garmin_status_view AS
SELECT
    gp.device_id,
    d.device_name,
    gp.garmin_device_name,
    gp.garmin_model,
    gp.connection_method,
    gp.is_connected,
    gp.battery_level,
    gp.battery_status,
    COUNT(DISTINCT gs.sensor_type) AS active_sensor_types,
    MAX(gs.timestamp) AS last_sensor_reading
FROM garmin_pairing gp
LEFT JOIN devices d ON gp.device_id = d.id
LEFT JOIN garmin_sensors gs ON gp.id = gs.garmin_id
GROUP BY gp.id, gp.device_id, d.device_name, gp.garmin_device_name,
         gp.garmin_model, gp.connection_method, gp.is_connected,
         gp.battery_level, gp.battery_status;

-- Function to update RTK state
CREATE OR REPLACE FUNCTION update_rtk_status(
    p_device_id UUID,
    p_new_status VARCHAR(50),
    p_num_satellites INTEGER,
    p_is_fixed BOOLEAN
)
RETURNS VOID AS $$
BEGIN
    UPDATE rtk_state
    SET
        rtk_status = p_new_status,
        last_correction_time = EXTRACT(EPOCH FROM NOW())::BIGINT,
        updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
        consecutive_fix_count = CASE WHEN p_is_fixed THEN consecutive_fix_count + 1 ELSE 0 END,
        consecutive_gap_count = CASE WHEN NOT p_is_fixed THEN consecutive_gap_count + 1 ELSE 0 END
    WHERE device_id = p_device_id;
END;
$$ LANGUAGE plpgsql;

-- Function to insert Kalman filter state
CREATE OR REPLACE FUNCTION save_kalman_state(
    p_device_id UUID,
    p_state_vector JSONB,
    p_covariance_matrix JSONB
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO kalman_filter_state (device_id, state_vector, covariance_matrix, process_noise)
    VALUES (
        p_device_id,
        p_state_vector,
        p_covariance_matrix,
        '{"diagonal": [0.01, 0.01, 0.01, 0.1, 0.1, 0.1, 0.1]}'::jsonb
    )
    ON CONFLICT (device_id) DO UPDATE SET
        state_vector = p_state_vector,
        covariance_matrix = p_covariance_matrix,
        updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;
END;
$$ LANGUAGE plpgsql;

COMMIT;
