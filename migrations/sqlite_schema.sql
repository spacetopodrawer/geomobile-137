-- Cadastre_IA Core - SQLite Schema
-- Embedded database for autonomous operation

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- ============================================================================
-- VECTOR OBJECTS TABLE (Core Data)
-- ============================================================================

CREATE TABLE IF NOT EXISTS vector_objects (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK(type IN (
        'parcel', 'building', 'tree', 'landmark', 'route', 'sensor',
        'structure', 'vegetation', 'water', 'custom'
    )),
    name TEXT NOT NULL,
    description TEXT,

    -- Geometry (stored as JSON)
    geometry TEXT,  -- GeoJSON as TEXT
    coordinate_frame TEXT DEFAULT 'WGS84',
    accuracy REAL,

    -- Sensor data (stored as JSON)
    sensor_data TEXT,  -- JSON blob containing all sensor data
    extracted_at TIMESTAMP,

    -- Properties
    properties TEXT,  -- JSONB alternative: JSON
    owner TEXT,
    classification TEXT,

    -- Rendering
    render_style TEXT,  -- JSON

    -- Versioning
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_modified_by TEXT,

    -- Sync metadata
    sync_id TEXT UNIQUE,
    last_sync_at TIMESTAMP,
    is_deleted INTEGER DEFAULT 0,
    deleted_at TIMESTAMP,

    -- Indexes
    created_index_version INTEGER DEFAULT 0  -- For optimistic locking
);

-- Spatial indexes (using ROWID for SQLite)
CREATE INDEX IF NOT EXISTS idx_vector_type ON vector_objects(type);
CREATE INDEX IF NOT EXISTS idx_vector_owner ON vector_objects(owner);
CREATE INDEX IF NOT EXISTS idx_vector_created ON vector_objects(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vector_modified ON vector_objects(modified_at DESC);
CREATE INDEX IF NOT EXISTS idx_vector_sync_id ON vector_objects(sync_id);
CREATE INDEX IF NOT EXISTS idx_vector_deleted ON vector_objects(is_deleted);
CREATE INDEX IF NOT EXISTS idx_vector_classification ON vector_objects(classification);

-- ============================================================================
-- TAGS TABLE (Many-to-many for object tags)
-- ============================================================================

CREATE TABLE IF NOT EXISTS object_tags (
    object_id TEXT NOT NULL REFERENCES vector_objects(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (object_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_tags_object ON object_tags(object_id);
CREATE INDEX IF NOT EXISTS idx_tags_name ON object_tags(tag);

-- ============================================================================
-- SYNC OPERATIONS LOG (Event Sourcing)
-- ============================================================================

CREATE TABLE IF NOT EXISTS sync_operations (
    id TEXT PRIMARY KEY,
    object_id TEXT NOT NULL REFERENCES vector_objects(id) ON DELETE CASCADE,
    operation_type TEXT NOT NULL CHECK(operation_type IN ('create', 'update', 'delete')),
    timestamp INTEGER NOT NULL,  -- Unix milliseconds
    device_id TEXT NOT NULL,

    -- Before/after states (JSON)
    before_state TEXT,
    after_state TEXT,

    -- Vector clock for CRDT
    vector_clock TEXT,  -- JSON: {"device-1": 5, "device-2": 3}

    -- Conflict resolution
    conflict_resolved_at TIMESTAMP,
    conflict_resolution_method TEXT,  -- 'ot', 'merge', 'user_choice', 'last_write_wins'

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    applied_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_object ON sync_operations(object_id);
CREATE INDEX IF NOT EXISTS idx_sync_timestamp ON sync_operations(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sync_device ON sync_operations(device_id);
CREATE INDEX IF NOT EXISTS idx_sync_applied ON sync_operations(applied_at);

-- ============================================================================
-- CONFLICTS TABLE (Concurrent edit tracking)
-- ============================================================================

CREATE TABLE IF NOT EXISTS conflicts (
    id TEXT PRIMARY KEY,
    object_id TEXT NOT NULL REFERENCES vector_objects(id) ON DELETE CASCADE,
    operation_a_id TEXT NOT NULL REFERENCES sync_operations(id),
    operation_b_id TEXT NOT NULL REFERENCES sync_operations(id),

    -- Conflict details (JSON)
    conflict_details TEXT,

    -- Resolution
    resolution_method TEXT,  -- 'ot', 'merge', 'user_choice', 'pending'
    resolved_at TIMESTAMP,
    resolved_by TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_conflicts_object ON conflicts(object_id);
CREATE INDEX IF NOT EXISTS idx_conflicts_resolution ON conflicts(resolution_method);

-- ============================================================================
-- DEVICE REGISTRY (Multi-device tracking)
-- ============================================================================

CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    device_type TEXT,  -- 'arcade', 'mobile', 'web', 'desktop'
    os TEXT,
    last_seen TIMESTAMP,
    is_online INTEGER DEFAULT 0,
    vector_clock TEXT,  -- Current vector clock state for this device
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_devices_online ON devices(is_online);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen DESC);

-- ============================================================================
-- SESSION TABLE (Multi-device sessions)
-- ============================================================================

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL REFERENCES devices(id),
    user_id TEXT,
    token TEXT UNIQUE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_device ON sessions(device_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- ============================================================================
-- AUDIT LOG TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id TEXT,
    user_id TEXT,
    device_id TEXT,

    -- Change details (JSON)
    changes TEXT,
    result TEXT,  -- 'success', 'failure'
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_device ON audit_logs(device_id);

-- ============================================================================
-- METADATA TABLE (System configuration)
-- ============================================================================

CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Initialize system metadata
INSERT OR IGNORE INTO metadata VALUES ('db_version', '1.0', CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO metadata VALUES ('schema_created', datetime('now'), CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO metadata VALUES ('last_sync_check', datetime('now'), CURRENT_TIMESTAMP);

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- Active objects (non-deleted)
CREATE VIEW IF NOT EXISTS v_active_objects AS
SELECT * FROM vector_objects
WHERE is_deleted = 0;

-- Objects with all their tags
CREATE VIEW IF NOT EXISTS v_objects_with_tags AS
SELECT
    vo.id,
    vo.name,
    vo.type,
    GROUP_CONCAT(ot.tag, ',') as tags
FROM vector_objects vo
LEFT JOIN object_tags ot ON vo.id = ot.object_id
WHERE vo.is_deleted = 0
GROUP BY vo.id;

-- Recent changes
CREATE VIEW IF NOT EXISTS v_recent_changes AS
SELECT
    so.id,
    so.object_id,
    so.operation_type,
    so.device_id,
    so.timestamp,
    vo.name,
    vo.type
FROM sync_operations so
LEFT JOIN vector_objects vo ON so.object_id = vo.id
ORDER BY so.timestamp DESC
LIMIT 100;

-- Unresolved conflicts
CREATE VIEW IF NOT EXISTS v_pending_conflicts AS
SELECT *
FROM conflicts
WHERE resolution_method = 'pending'
ORDER BY created_at DESC;

-- Device sync status
CREATE VIEW IF NOT EXISTS v_device_status AS
SELECT
    d.id,
    d.name,
    d.device_type,
    d.is_online,
    d.last_seen,
    COUNT(so.id) as operations_count,
    MAX(so.timestamp) as last_operation
FROM devices d
LEFT JOIN sync_operations so ON d.id = so.device_id
GROUP BY d.id;

-- ============================================================================
-- TRIGGERS FOR AUTOMATIC TIMESTAMPS
-- ============================================================================

-- Update modified_at when object changes
CREATE TRIGGER IF NOT EXISTS trig_update_modified
AFTER UPDATE ON vector_objects
FOR EACH ROW
BEGIN
    UPDATE vector_objects SET modified_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

-- Log changes to audit table
CREATE TRIGGER IF NOT EXISTS trig_audit_insert
AFTER INSERT ON vector_objects
FOR EACH ROW
BEGIN
    INSERT INTO audit_logs (action, entity_type, entity_id, changes)
    VALUES ('CREATE', 'vector_object', NEW.id, json_object('name', NEW.name, 'type', NEW.type));
END;

CREATE TRIGGER IF NOT EXISTS trig_audit_update
AFTER UPDATE ON vector_objects
FOR EACH ROW
BEGIN
    INSERT INTO audit_logs (action, entity_type, entity_id, changes)
    VALUES ('UPDATE', 'vector_object', NEW.id,
            json_object('modified_by', NEW.last_modified_by, 'version', NEW.version));
END;

CREATE TRIGGER IF NOT EXISTS trig_audit_delete
AFTER UPDATE ON vector_objects
WHEN NEW.is_deleted = 1 AND OLD.is_deleted = 0
FOR EACH ROW
BEGIN
    INSERT INTO audit_logs (action, entity_type, entity_id)
    VALUES ('DELETE', 'vector_object', NEW.id);
END;

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Create default device entry for this instance
INSERT OR IGNORE INTO devices (id, name, device_type)
VALUES ('local-device-0', 'Local Device', 'desktop');

-- ============================================================================
-- PRAGMAS FOR PERFORMANCE
-- ============================================================================

-- Journal mode for better concurrent access
PRAGMA journal_mode = WAL;

-- Synchronous mode for data safety
PRAGMA synchronous = NORMAL;

-- Cache size for performance
PRAGMA cache_size = 10000;

-- Query optimization
PRAGMA optimize;

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
