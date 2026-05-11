-- Migration: Create cadastral_entities table with spatial indices
-- Phase: 2.2 Day 2 Implementation
-- Date: 2026-05-11

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS uuid-ossp;

-- Main cadastral entities table
CREATE TABLE IF NOT EXISTS cadastral_entities (
    id VARCHAR(50) PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL,
    region VARCHAR(50) NOT NULL,
    admin_unit VARCHAR(100) NOT NULL,

    -- Geometry stored as GeoJSON for flexibility
    geometry GEOMETRY(Geometry, 4326) NOT NULL,

    -- Flexible attribute storage (JSON)
    attributes JSONB DEFAULT '{}',

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    updated_by VARCHAR(100),

    -- Data quality tracking
    data_quality_score FLOAT DEFAULT 1.0,
    confidence_level FLOAT DEFAULT 1.0,

    -- Soft delete support
    deleted_at TIMESTAMP,

    -- Version control
    version INT DEFAULT 1
);

-- Indices for common queries
CREATE INDEX idx_cadastral_entities_code
    ON cadastral_entities(code);

CREATE INDEX idx_cadastral_entities_type
    ON cadastral_entities(type);

CREATE INDEX idx_cadastral_entities_region_admin
    ON cadastral_entities(region, admin_unit);

CREATE INDEX idx_cadastral_entities_created_at
    ON cadastral_entities(created_at DESC);

CREATE INDEX idx_cadastral_entities_geometry
    ON cadastral_entities USING GIST(geometry);

-- Spatial index for fast geographic queries
CREATE INDEX idx_cadastral_entities_bbox
    ON cadastral_entities USING GIST(ST_Envelope(geometry));

-- JSON attribute indices for common lookups
CREATE INDEX idx_cadastral_entities_owner
    ON cadastral_entities USING GIN(attributes);

-- Audit log table for tracking changes
CREATE TABLE IF NOT EXISTS cadastral_audit_log (
    id SERIAL PRIMARY KEY,
    entity_id VARCHAR(50) NOT NULL,
    entity_code VARCHAR(100),
    action VARCHAR(50) NOT NULL,  -- 'created', 'updated', 'deleted'
    old_values JSONB,
    new_values JSONB,
    changed_by VARCHAR(100),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (entity_id) REFERENCES cadastral_entities(id) ON DELETE CASCADE
);

CREATE INDEX idx_cadastral_audit_entity_id
    ON cadastral_audit_log(entity_id);

CREATE INDEX idx_cadastral_audit_action
    ON cadastral_audit_log(action);

CREATE INDEX idx_cadastral_audit_changed_at
    ON cadastral_audit_log(changed_at DESC);

-- Materialized view for tile generation (performance optimization)
CREATE MATERIALIZED VIEW IF NOT EXISTS cadastral_entities_by_tile AS
SELECT
    id,
    code,
    type,
    region,
    admin_unit,
    geometry,
    attributes,
    -- Web Mercator tile coordinate helpers
    x,
    y,
    z
FROM cadastral_entities
WHERE deleted_at IS NULL
-- View will be refreshed after batch inserts
;

-- Performance comment
COMMENT ON TABLE cadastral_entities IS
    'Core cadastral entities (buildings, parcels, POIs). Geometry in WGS84 (EPSG:4326).';

COMMENT ON TABLE cadastral_audit_log IS
    'Audit trail for cadastral entity changes. Used for compliance tracking.';
