-- Phase 3B.2: Hierarchical Conflict Resolution & GNSS Infrastructure
-- Migration 009: Add authority, device trust, and station management tables

-- User Authority Hierarchy Table
CREATE TABLE IF NOT EXISTS user_authority (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    authority_level VARCHAR(50) NOT NULL, -- system, super_admin, admin, author, co_author, user
    domain VARCHAR(50),                   -- national, state, municipality, private, public
    parcel_permission_type VARCHAR(50),   -- read_only, write, admin
    delegated_by_user_id UUID,
    delegation_scope JSONB,               -- {parcel_ids: [...], regions: [...], active: true}
    delegation_valid_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_authority_user_fk FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT user_authority_delegator_fk FOREIGN KEY (delegated_by_user_id) REFERENCES users(id)
);

CREATE INDEX idx_user_authority_user ON user_authority(user_id);
CREATE INDEX idx_user_authority_level ON user_authority(authority_level);
CREATE INDEX idx_user_authority_domain ON user_authority(domain);

-- Device Trust Profile Table
CREATE TABLE IF NOT EXISTS device_trust_profile (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id VARCHAR(255) NOT NULL,
    device_type VARCHAR(50) NOT NULL,    -- phone, tablet, survey_instrument, vehicle
    gnss_receiver_type VARCHAR(100),     -- rtk_gnss, ppp, standard, legacy
    accuracy_classification VARCHAR(50), -- cm_level, dm_level, m_level
    calibration_status VARCHAR(50),      -- calibrated, uncalibrated, expired
    last_calibration TIMESTAMP,
    confidence_score FLOAT DEFAULT 50.0, -- 0-100 (based on historical accuracy)
    owner_user_id UUID NOT NULL,
    primary_device BOOLEAN DEFAULT FALSE,
    priority_level INT DEFAULT 50,       -- 0-100, higher = more trusted
    calibration_error_mm FLOAT,          -- Last measured error in mm
    gps_constellations TEXT[],           -- [GPS, GLONASS, Galileo, BeiDou]
    fixed_solution_rate FLOAT,           -- % of time achieving RTK FIXED
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_verified TIMESTAMP,
    CONSTRAINT device_trust_profile_user_fk FOREIGN KEY (owner_user_id) REFERENCES users(id)
);

CREATE INDEX idx_device_trust_profile_device_id ON device_trust_profile(device_id);
CREATE INDEX idx_device_trust_profile_owner ON device_trust_profile(owner_user_id);
CREATE INDEX idx_device_trust_profile_priority ON device_trust_profile(priority_level);

-- GNSS Base Stations Registry
CREATE TABLE IF NOT EXISTS gnss_stations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    station_code VARCHAR(10) UNIQUE NOT NULL, -- E.g., YAOUNDE_01
    location GEOMETRY(POINT, 4326) NOT NULL,
    elevation_m FLOAT,
    receiver_types TEXT[],                    -- [u-blox, Septentrio, Novatel, ...]
    constellations TEXT[],                    -- [GPS, GLONASS, Galileo, BeiDou]
    rtcm_version VARCHAR(10),                 -- RTCM 2.3, 3.1, 3.2, 3.3
    ntrip_mount_point VARCHAR(255) UNIQUE,
    correction_types TEXT[],                  -- [RTK, PPP, SSR, ...]
    update_rate_hz FLOAT DEFAULT 1.0,
    accuracy_horizontal_cm FLOAT,
    accuracy_vertical_cm FLOAT,
    calibrated_at TIMESTAMP,
    last_verification TIMESTAMP,
    operational_status VARCHAR(50) DEFAULT 'operational', -- operational, maintenance, offline
    uptime_percent FLOAT DEFAULT 100.0,
    custodian_user_id UUID NOT NULL,
    owner_entity VARCHAR(255),                -- National, State, Private, Public, etc.
    region_code VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT gnss_stations_custodian_fk FOREIGN KEY (custodian_user_id) REFERENCES users(id)
);

CREATE INDEX idx_gnss_stations_region ON gnss_stations(region_code);
CREATE INDEX idx_gnss_stations_status ON gnss_stations(operational_status);
CREATE INDEX idx_gnss_stations_location ON gnss_stations USING GIST(location);

-- Mountpoint Streams (NTrip)
CREATE TABLE IF NOT EXISTS mountpoint_streams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gnss_station_id UUID NOT NULL,
    stream_type VARCHAR(50) NOT NULL,       -- ntrip, rtcm_raw, custom_protocol
    mount_point_name VARCHAR(255) NOT NULL,
    protocol_version VARCHAR(50),           -- RTCM3.1, CMR, SPARTN, etc.
    data_format VARCHAR(50),
    update_interval_ms INT DEFAULT 1000,
    active BOOLEAN DEFAULT TRUE,
    subscriber_count INT DEFAULT 0,
    last_data_received TIMESTAMP,
    quality_score FLOAT DEFAULT 100.0,      -- 0-100 based on reception quality
    redundancy_type VARCHAR(50),            -- primary, backup, load_balance
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT mountpoint_streams_station_fk FOREIGN KEY (gnss_station_id)
        REFERENCES gnss_stations(id) ON DELETE CASCADE,
    CONSTRAINT mountpoint_streams_unique UNIQUE (gnss_station_id, mount_point_name)
);

-- Hierarchical Conflict Records (Extended)
CREATE TABLE IF NOT EXISTS hierarchical_conflict_record (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id VARCHAR(255) NOT NULL,
    parcel_domain VARCHAR(50),
    parcel_custodian_user_id UUID,

    -- Edit 1
    edit_1_user_id UUID NOT NULL,
    edit_1_user_authority VARCHAR(50),      -- system, super_admin, admin, author, ...
    edit_1_device_id VARCHAR(255),
    edit_1_device_priority INT,
    edit_1_data JSONB,
    edit_1_timestamp TIMESTAMP,

    -- Edit 2
    edit_2_user_id UUID NOT NULL,
    edit_2_user_authority VARCHAR(50),
    edit_2_device_id VARCHAR(255),
    edit_2_device_priority INT,
    edit_2_data JSONB,
    edit_2_timestamp TIMESTAMP,

    -- Detection & Resolution
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolution_strategy VARCHAR(50),        -- authority_hierarchy, device_priority, averaging, escalation, escalation_timeout
    resolved_at TIMESTAMP,
    resolved_by_user_id UUID,
    winning_edit JSONB,
    winning_edit_source VARCHAR(50),        -- edit_1, edit_2, merged, averaged
    resolution_notes TEXT,

    -- Escalation
    escalated_to_user_id UUID,
    escalation_reason TEXT,
    escalation_timestamp TIMESTAMP,
    resolution_deadline TIMESTAMP,
    alerted_users TEXT[],

    CONSTRAINT hierarchical_conflict_record_parcel_fk FOREIGN KEY (parcel_id)
        REFERENCES parcels(parcel_id),
    CONSTRAINT hierarchical_conflict_record_edit1_user_fk FOREIGN KEY (edit_1_user_id)
        REFERENCES users(id),
    CONSTRAINT hierarchical_conflict_record_edit2_user_fk FOREIGN KEY (edit_2_user_id)
        REFERENCES users(id),
    CONSTRAINT hierarchical_conflict_record_custodian_fk FOREIGN KEY (parcel_custodian_user_id)
        REFERENCES users(id),
    CONSTRAINT hierarchical_conflict_record_resolver_fk FOREIGN KEY (resolved_by_user_id)
        REFERENCES users(id),
    CONSTRAINT hierarchical_conflict_record_escalator_fk FOREIGN KEY (escalated_to_user_id)
        REFERENCES users(id)
);

CREATE INDEX idx_hierarchical_conflict_parcel ON hierarchical_conflict_record(parcel_id);
CREATE INDEX idx_hierarchical_conflict_unresolved ON hierarchical_conflict_record(resolved_at)
    WHERE resolved_at IS NULL;
CREATE INDEX idx_hierarchical_conflict_escalated ON hierarchical_conflict_record(escalated_to_user_id)
    WHERE escalated_to_user_id IS NOT NULL AND resolved_at IS NULL;

-- Authority Hierarchy Reference (Lookup Table)
CREATE TABLE IF NOT EXISTS authority_hierarchy (
    level_id INT PRIMARY KEY,
    level_name VARCHAR(50) UNIQUE NOT NULL,
    numeric_priority INT UNIQUE NOT NULL, -- 1000=System, 900=SuperAdmin, 800=Admin, 700=Author, 600=CoAuthor, 500=User
    description TEXT,
    can_override_lower BOOLEAN DEFAULT FALSE
);

INSERT INTO authority_hierarchy (level_id, level_name, numeric_priority, description, can_override_lower) VALUES
(1, 'System', 1000, 'System-level operations', TRUE),
(2, 'Super-Admin', 900, 'Override any decision', TRUE),
(3, 'Admin', 800, 'Regional/national coordinator', TRUE),
(4, 'Author', 700, 'Parcel creator/owner', FALSE),
(5, 'Co-Author', 600, 'Delegated editor', FALSE),
(6, 'User', 500, 'General user/viewer', FALSE);

-- Functions for hierarchical resolution

-- Function to get user authority level
CREATE OR REPLACE FUNCTION get_user_authority_level(p_user_id UUID)
RETURNS VARCHAR AS $$
BEGIN
    RETURN (SELECT authority_level FROM user_authority
            WHERE user_id = p_user_id
            ORDER BY authority_level DESC LIMIT 1);
END;
$$ LANGUAGE plpgsql;

-- Function to resolve conflict based on hierarchy
CREATE OR REPLACE FUNCTION resolve_conflict_hierarchically(
    p_parcel_id VARCHAR,
    p_user1_id UUID,
    p_user1_authority VARCHAR,
    p_user1_device_priority INT,
    p_user2_id UUID,
    p_user2_authority VARCHAR,
    p_user2_device_priority INT
)
RETURNS VARCHAR AS $$
BEGIN
    -- Step 1: Compare authority levels
    IF (SELECT numeric_priority FROM authority_hierarchy WHERE level_name = p_user1_authority) >
       (SELECT numeric_priority FROM authority_hierarchy WHERE level_name = p_user2_authority) THEN
        RETURN p_user1_id || ' wins by authority (' || p_user1_authority || ')';
    END IF;

    IF (SELECT numeric_priority FROM authority_hierarchy WHERE level_name = p_user2_authority) >
       (SELECT numeric_priority FROM authority_hierarchy WHERE level_name = p_user1_authority) THEN
        RETURN p_user2_id || ' wins by authority (' || p_user2_authority || ')';
    END IF;

    -- Step 2: Same authority? Compare device priority
    IF p_user1_device_priority > p_user2_device_priority THEN
        RETURN p_user1_id || ' wins by device priority (' || p_user1_device_priority || ')';
    END IF;

    IF p_user2_device_priority > p_user1_device_priority THEN
        RETURN p_user2_id || ' wins by device priority (' || p_user2_device_priority || ')';
    END IF;

    -- Step 3: Unresolvable
    RETURN 'ESCALATE to custodian';
END;
$$ LANGUAGE plpgsql;

-- View for active GNSS stations dashboard
CREATE OR REPLACE VIEW v_gnss_stations_status AS
SELECT
    station_code,
    elevation_m,
    operational_status,
    uptime_percent,
    accuracy_horizontal_cm,
    accuracy_vertical_cm,
    (SELECT COUNT(*) FROM mountpoint_streams WHERE gnss_station_id = gs.id) as stream_count,
    COALESCE((SELECT SUM(subscriber_count) FROM mountpoint_streams WHERE gnss_station_id = gs.id), 0) as total_subscribers,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - last_verification)) / 3600 as hours_since_verification
FROM gnss_stations gs
ORDER BY uptime_percent DESC;

-- View for unresolved conflicts needing escalation
CREATE OR REPLACE VIEW v_escalation_queue AS
SELECT
    hcr.id,
    hcr.parcel_id,
    hcr.detected_at,
    hcr.resolution_deadline,
    EXTRACT(EPOCH FROM (hcr.resolution_deadline - CURRENT_TIMESTAMP)) / 60 as minutes_until_deadline,
    u1.username as user1_name,
    hcr.edit_1_user_authority as user1_authority,
    u2.username as user2_name,
    hcr.edit_2_user_authority as user2_authority,
    u3.username as escalated_to,
    hcr.escalation_reason
FROM hierarchical_conflict_record hcr
LEFT JOIN users u1 ON hcr.edit_1_user_id = u1.id
LEFT JOIN users u2 ON hcr.edit_2_user_id = u2.id
LEFT JOIN users u3 ON hcr.escalated_to_user_id = u3.id
WHERE hcr.resolved_at IS NULL
ORDER BY hcr.resolution_deadline ASC;

-- Grant permissions
-- GRANT SELECT ON v_gnss_stations_status TO app_role;
-- GRANT SELECT ON v_escalation_queue TO app_role;

-- Migration complete
COMMENT ON TABLE user_authority IS 'Hierarchical user permissions and role delegation';
COMMENT ON TABLE device_trust_profile IS 'Device precision and trust scoring for conflict resolution';
COMMENT ON TABLE gnss_stations IS 'Reference GNSS base stations and corrections infrastructure';
COMMENT ON TABLE mountpoint_streams IS 'NTrip/RTCM streams for real-time corrections';
COMMENT ON TABLE hierarchical_conflict_record IS 'Conflict detection with authority-aware resolution';
COMMENT ON FUNCTION resolve_conflict_hierarchically IS 'Determines winner based on authority hierarchy and device priority';
