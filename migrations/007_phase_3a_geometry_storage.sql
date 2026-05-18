-- Phase 3A: Unified Geometry Storage (glTF-based)
-- Implements efficient multi-platform asset management

-- Cache for converted geometries (glTF documents)
CREATE TABLE IF NOT EXISTS cached_geometries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Source information
    source_format VARCHAR(50) NOT NULL,        -- 'geojson', 'geotiff', 'shapefile', 'pointcloud'
    source_path TEXT NOT NULL,                 -- Original file path/URL
    source_hash VARCHAR(64),                   -- SHA256 of source for caching

    -- glTF binary (primary artifact)
    gltf_binary BYTEA NOT NULL,                -- Complete glTF 2.0 document (compressed)
    gltf_size_bytes INT,                       -- Size of glTF before compression

    -- Metadata and properties
    gltf_metadata JSONB,                       -- Custom properties: parcel_id, owner, area, etc.
    feature_count INT,                         -- Number of features/parcels

    -- LOD levels (pre-generated)
    lod_high BYTEA,                            -- Full detail for VR/desktop
    lod_medium BYTEA,                          -- Mid detail for web
    lod_low BYTEA,                             -- Low detail for mobile

    -- Geometry statistics
    vertex_count INT,
    triangle_count INT,
    file_size_bytes INT,

    -- Compression metrics
    compression_ratio FLOAT,                   -- Compressed / Original size
    draco_compression_enabled BOOLEAN DEFAULT true,
    quantization_bits INT DEFAULT 16,          -- Vertex quantization precision

    -- Geospatial information
    bounds_minx DOUBLE PRECISION,
    bounds_miny DOUBLE PRECISION,
    bounds_minz DOUBLE PRECISION,
    bounds_maxx DOUBLE PRECISION,
    bounds_maxy DOUBLE PRECISION,
    bounds_maxz DOUBLE PRECISION,
    bounds_geom GEOMETRY(Envelope, 4326),     -- WGS84 bounding box

    -- CRS configuration
    cesium_georeference JSONB,                 -- CRS metadata for Cesium

    -- Platform-specific variants
    ue5_asset_id UUID,                         -- Reference to UE5 asset
    web_asset_id UUID,                         -- Reference to web asset
    mobile_asset_id UUID,                      -- Reference to mobile asset

    -- Lifecycle
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,                      -- Cache expiration

    -- Indices for performance
    CONSTRAINT bounds_valid CHECK (bounds_minx <= bounds_maxx AND bounds_miny <= bounds_maxy),
    INDEX idx_source_format (source_format),
    INDEX idx_source_hash (source_hash),
    INDEX idx_bounds (bounds_geom),
    INDEX idx_created_at (created_at),
    INDEX idx_expires_at (expires_at)
);

-- Platform-specific asset variants
CREATE TABLE IF NOT EXISTS platform_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Link to source geometry
    geometry_id UUID NOT NULL REFERENCES cached_geometries(id) ON DELETE CASCADE,

    -- Platform information
    platform VARCHAR(50) NOT NULL,             -- 'ue5', 'web', 'mobile', 'webxr'
    format VARCHAR(50),                        -- 'uasset', 'glb', 'glb_mobile'

    -- Asset data
    asset_binary BYTEA NOT NULL,               -- Platform-specific binary format
    asset_size_bytes INT,

    -- Optimization details
    lod_level INT,                             -- 0=full, 1=medium, 2=low
    vertex_count INT,
    triangle_count INT,

    -- Performance metrics
    compression_ratio FLOAT,
    load_time_ms INT,                          -- Time to deserialize on target platform

    -- Material optimization
    material_count INT,
    supports_pbr BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_geometry_id (geometry_id),
    INDEX idx_platform (platform)
);

-- Geometry import jobs and history
CREATE TABLE IF NOT EXISTS geometry_imports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Job information
    source_file TEXT NOT NULL,
    source_format VARCHAR(50),

    -- Processing status
    status VARCHAR(50) DEFAULT 'pending',      -- pending, processing, completed, failed
    error_message TEXT,

    -- Output
    geometry_id UUID REFERENCES cached_geometries(id) ON DELETE SET NULL,

    -- Statistics
    feature_count INT,
    processing_time_ms INT,
    file_size_input_bytes INT,
    file_size_output_bytes INT,

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    INDEX idx_status (status),
    INDEX idx_geometry_id (geometry_id),
    INDEX idx_created_at (created_at)
);

-- Compression test results and benchmarks
CREATE TABLE IF NOT EXISTS compression_benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Geometry reference
    geometry_id UUID NOT NULL REFERENCES cached_geometries(id) ON DELETE CASCADE,

    -- Compression parameters
    method VARCHAR(50),                        -- 'draco', 'gzip', 'custom'
    compression_level INT,                     -- 0-10: higher = more compression
    quantization_bits INT,

    -- Results
    original_size_bytes INT,
    compressed_size_bytes INT,
    compression_ratio FLOAT,
    savings_percent FLOAT,

    -- Performance
    compression_time_ms INT,
    decompression_time_ms INT,

    -- Quality assessment
    error_threshold FLOAT,                     -- Max error introduced

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_geometry_id (geometry_id),
    INDEX idx_method (method)
);

-- Geometry validation and audit
CREATE TABLE IF NOT EXISTS geometry_validation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    geometry_id UUID NOT NULL REFERENCES cached_geometries(id) ON DELETE CASCADE,

    -- Validation checks
    check_name VARCHAR(100),
    check_status VARCHAR(50),                  -- 'pass', 'fail', 'warning'
    message TEXT,

    -- Validation details
    vertices_valid BOOLEAN,
    indices_valid BOOLEAN,
    bounds_valid BOOLEAN,
    metadata_valid BOOLEAN,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_geometry_id (geometry_id),
    INDEX idx_check_status (check_status)
);

-- Functions and triggers

-- Function: Calculate geometry statistics
CREATE OR REPLACE FUNCTION calculate_geometry_stats(
    vertex_count INT,
    triangle_count INT,
    file_size INT
) RETURNS TABLE(vertices INT, triangles INT, bytes INT, tri_per_vertex FLOAT) AS $$
BEGIN
    RETURN QUERY SELECT
        vertex_count,
        triangle_count,
        file_size,
        CASE WHEN vertex_count > 0 THEN triangle_count::FLOAT / vertex_count ELSE 0 END;
END;
$$ LANGUAGE plpgsql;

-- Function: Estimate compression savings
CREATE OR REPLACE FUNCTION estimate_compression_savings(
    original_size INT,
    compression_level INT
) RETURNS TABLE(estimated_size INT, savings_percent FLOAT, ratio FLOAT) AS $$
DECLARE
    estimated_ratio FLOAT;
BEGIN
    -- Empirical: ratio = 0.5 at level 5, improving with higher levels
    estimated_ratio := 0.5 - (compression_level::FLOAT / 100);
    estimated_ratio := GREATEST(estimated_ratio, 0.05); -- Min 5% of original

    RETURN QUERY SELECT
        ROUND(original_size * estimated_ratio)::INT,
        (1 - estimated_ratio) * 100,
        estimated_ratio;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Update timestamp on cached_geometries
CREATE OR REPLACE FUNCTION update_geometry_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_geometry_updated
BEFORE UPDATE ON cached_geometries
FOR EACH ROW
EXECUTE FUNCTION update_geometry_timestamp();

-- Trigger: Clean up expired cached geometries
CREATE OR REPLACE FUNCTION cleanup_expired_geometries()
RETURNS void AS $$
BEGIN
    DELETE FROM cached_geometries
    WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- View: Geometry cache statistics
CREATE OR REPLACE VIEW geometry_cache_stats AS
SELECT
    COUNT(*) as total_cached_geometries,
    SUM(file_size_bytes) as total_cache_size_bytes,
    AVG(compression_ratio) as avg_compression_ratio,
    AVG(vertex_count) as avg_vertices,
    AVG(triangle_count) as avg_triangles,
    COUNT(DISTINCT source_format) as unique_formats
FROM cached_geometries
WHERE expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP;

-- View: Platform asset distribution
CREATE OR REPLACE VIEW platform_asset_distribution AS
SELECT
    platform,
    COUNT(*) as asset_count,
    SUM(asset_size_bytes) as total_size_bytes,
    AVG(compression_ratio) as avg_compression_ratio,
    MIN(load_time_ms) as min_load_time_ms,
    MAX(load_time_ms) as max_load_time_ms,
    AVG(load_time_ms) as avg_load_time_ms
FROM platform_assets
GROUP BY platform;

-- Indexes for optimization
CREATE INDEX idx_geometry_source ON cached_geometries(source_format, source_hash);
CREATE INDEX idx_platform_optimization ON platform_assets(platform, lod_level, compression_ratio);
CREATE INDEX idx_import_status ON geometry_imports(status, created_at DESC);

-- Grant permissions
GRANT SELECT ON cached_geometries TO PUBLIC;
GRANT SELECT ON platform_assets TO PUBLIC;
GRANT SELECT ON geometry_cache_stats TO PUBLIC;
GRANT SELECT ON platform_asset_distribution TO PUBLIC;

-- Logging
INSERT INTO geometry_imports (source_file, source_format, status)
VALUES ('Migration 007: Phase 3A Geometry Storage', 'schema', 'completed')
ON CONFLICT DO NOTHING;
