package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"cadastreia/pkg/model"
	syncsvc "cadastreia/pkg/sync"
)

// StorageManager manages SQLite database operations
type StorageManager struct {
	mu       sync.RWMutex
	db       *sql.DB
	filepath string
	config   StorageConfig
}

// StorageConfig contains storage configuration
type StorageConfig struct {
	Path              string
	MaxConnections    int
	AutoBackup        bool
	BackupInterval    time.Duration
	EnableWAL         bool
	PageSize          int
	CacheSizePages    int
	QueryTimeoutMS    int
}

// NewStorageManager creates a new storage manager
func NewStorageManager(config StorageConfig) *StorageManager {
	return &StorageManager{
		filepath: config.Path,
		config:   config,
	}
}

// Initialize initializes the database
func (sm *StorageManager) Initialize() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Open database connection
	db, err := sql.Open("sqlite3", sm.filepath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(sm.config.MaxConnections)
	db.SetMaxIdleConns(2)

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Configure WAL if enabled
	if sm.config.EnableWAL {
		if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
			return fmt.Errorf("failed to enable WAL: %w", err)
		}
	}

	// Set cache size
	if _, err := db.Exec(fmt.Sprintf("PRAGMA cache_size = %d", sm.config.CacheSizePages)); err != nil {
		return fmt.Errorf("failed to set cache size: %w", err)
	}

	// Initialize database schema if tables don't exist
	if err := sm.initializeSchema(db); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	sm.db = db
	return nil
}

// initializeSchema creates tables if they don't exist
func (sm *StorageManager) initializeSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS vector_objects (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		geometry TEXT,
		coordinate_frame TEXT DEFAULT 'WGS84',
		accuracy REAL,
		sensor_data TEXT,
		extracted_at TIMESTAMP,
		properties TEXT,
		owner TEXT,
		classification TEXT,
		render_style TEXT,
		version INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		modified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_modified_by TEXT,
		sync_id TEXT UNIQUE,
		last_sync_at TIMESTAMP,
		is_deleted INTEGER DEFAULT 0,
		deleted_at TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_vector_type ON vector_objects(type);
	CREATE INDEX IF NOT EXISTS idx_vector_owner ON vector_objects(owner);
	CREATE INDEX IF NOT EXISTS idx_vector_sync_id ON vector_objects(sync_id);

	CREATE TABLE IF NOT EXISTS sync_operations (
		id TEXT PRIMARY KEY,
		object_id TEXT,
		operation_type TEXT,
		timestamp INTEGER,
		device_id TEXT,
		before_state TEXT,
		after_state TEXT,
		vector_clock TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_sync_object ON sync_operations(object_id);
	CREATE INDEX IF NOT EXISTS idx_sync_device ON sync_operations(device_id);
	`

	_, err := db.Exec(schema)
	return err
}

// SaveObject saves a vector object to the database
func (sm *StorageManager) SaveObject(vo *model.VectorObject) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.db == nil {
		return fmt.Errorf("database not initialized")
	}

	geomJSON, _ := json.Marshal(vo.Geometry)
	sensorJSON, _ := json.Marshal(vo.SensorData)
	propsJSON, _ := json.Marshal(vo.Properties)
	renderJSON, _ := json.Marshal(vo.RenderStyle)

	query := `
		INSERT OR REPLACE INTO vector_objects
		(id, type, name, description, geometry, coordinate_frame, accuracy,
		 sensor_data, extracted_at, properties, owner, classification, render_style,
		 version, created_at, modified_at, last_modified_by, sync_id, last_sync_at,
		 is_deleted, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := sm.db.Exec(query,
		vo.ID.String(), vo.Type, vo.Name, vo.Description,
		string(geomJSON), vo.CoordinateFrame, vo.Accuracy,
		string(sensorJSON), vo.ExtractedAt, string(propsJSON),
		vo.Owner, vo.Classification, string(renderJSON),
		vo.Version, vo.CreatedAt, vo.ModifiedAt, vo.LastModifiedBy,
		vo.SyncID, vo.LastSyncAt, vo.IsDeleted, vo.DeletedAt,
	)

	return err
}

// LoadObject loads a vector object from the database
func (sm *StorageManager) LoadObject(id string) (*model.VectorObject, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, type, name, description, geometry, coordinate_frame, accuracy,
		       sensor_data, extracted_at, properties, owner, classification, render_style,
		       version, created_at, modified_at, last_modified_by, sync_id, last_sync_at,
		       is_deleted, deleted_at
		FROM vector_objects WHERE id = ?
	`

	var (
		vo model.VectorObject
		geomJSON, sensorJSON, propsJSON, renderJSON string
	)

	err := sm.db.QueryRow(query, id).Scan(
		&vo.ID, &vo.Type, &vo.Name, &vo.Description,
		&geomJSON, &vo.CoordinateFrame, &vo.Accuracy,
		&sensorJSON, &vo.ExtractedAt, &propsJSON,
		&vo.Owner, &vo.Classification, &renderJSON,
		&vo.Version, &vo.CreatedAt, &vo.ModifiedAt, &vo.LastModifiedBy,
		&vo.SyncID, &vo.LastSyncAt, &vo.IsDeleted, &vo.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON blobs
	json.Unmarshal([]byte(geomJSON), &vo.Geometry)
	json.Unmarshal([]byte(sensorJSON), &vo.SensorData)
	json.Unmarshal([]byte(propsJSON), &vo.Properties)
	json.Unmarshal([]byte(renderJSON), &vo.RenderStyle)

	return &vo, nil
}

// ListObjects lists vector objects with optional filtering
func (sm *StorageManager) ListObjects(limit int, offset int) ([]*model.VectorObject, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, type, name, description, geometry, coordinate_frame, accuracy,
		       sensor_data, extracted_at, properties, owner, classification, render_style,
		       version, created_at, modified_at, last_modified_by, sync_id, last_sync_at,
		       is_deleted, deleted_at
		FROM vector_objects WHERE is_deleted = 0
		ORDER BY modified_at DESC LIMIT ? OFFSET ?
	`

	rows, err := sm.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := make([]*model.VectorObject, 0)
	for rows.Next() {
		var (
			vo model.VectorObject
			geomJSON, sensorJSON, propsJSON, renderJSON string
		)

		err := rows.Scan(
			&vo.ID, &vo.Type, &vo.Name, &vo.Description,
			&geomJSON, &vo.CoordinateFrame, &vo.Accuracy,
			&sensorJSON, &vo.ExtractedAt, &propsJSON,
			&vo.Owner, &vo.Classification, &renderJSON,
			&vo.Version, &vo.CreatedAt, &vo.ModifiedAt, &vo.LastModifiedBy,
			&vo.SyncID, &vo.LastSyncAt, &vo.IsDeleted, &vo.DeletedAt,
		)

		if err != nil {
			continue
		}

		json.Unmarshal([]byte(geomJSON), &vo.Geometry)
		json.Unmarshal([]byte(sensorJSON), &vo.SensorData)
		json.Unmarshal([]byte(propsJSON), &vo.Properties)
		json.Unmarshal([]byte(renderJSON), &vo.RenderStyle)

		objects = append(objects, &vo)
	}

	return objects, nil
}

// DeleteObject soft-deletes an object
func (sm *StorageManager) DeleteObject(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	query := `UPDATE vector_objects SET is_deleted = 1, deleted_at = ? WHERE id = ?`
	_, err := sm.db.Exec(query, now, id)
	return err
}

// SaveOperation saves a sync operation to the log
func (sm *StorageManager) SaveOperation(op *syncsvc.Operation) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.db == nil {
		return fmt.Errorf("database not initialized")
	}

	vcJSON, _ := json.Marshal(op.VectorClock)
	beforeJSON, _ := json.Marshal(op.BeforeState)
	afterJSON, _ := json.Marshal(op.AfterState)

	query := `
		INSERT INTO sync_operations
		(id, object_id, operation_type, timestamp, device_id,
		 before_state, after_state, vector_clock, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := sm.db.Exec(query,
		op.ID, op.ObjectID, op.Type, op.Timestamp, op.DeviceID,
		beforeJSON, afterJSON, vcJSON, time.Now(),
	)

	return err
}

// GetOperationsSince gets operations since a timestamp
func (sm *StorageManager) GetOperationsSince(timestamp int64) ([]*syncsvc.Operation, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, object_id, operation_type, timestamp, device_id,
		       before_state, after_state, vector_clock
		FROM sync_operations WHERE timestamp > ? ORDER BY timestamp ASC
	`

	rows, err := sm.db.Query(query, timestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ops := make([]*syncsvc.Operation, 0)
	for rows.Next() {
		var (
			op syncsvc.Operation
			beforeJSON, afterJSON, vcJSON string
		)

		err := rows.Scan(
			&op.ID, &op.ObjectID, &op.Type, &op.Timestamp, &op.DeviceID,
			&beforeJSON, &afterJSON, &vcJSON,
		)

		if err != nil {
			continue
		}

		json.Unmarshal([]byte(beforeJSON), &op.BeforeState)
		json.Unmarshal([]byte(afterJSON), &op.AfterState)
		json.Unmarshal([]byte(vcJSON), &op.VectorClock)

		ops = append(ops, &op)
	}

	return ops, nil
}

// GetStats returns database statistics
func (sm *StorageManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]interface{})

	if sm.db == nil {
		return stats
	}

	var objCount, opCount int64
	sm.db.QueryRow("SELECT COUNT(*) FROM vector_objects WHERE is_deleted = 0").Scan(&objCount)
	sm.db.QueryRow("SELECT COUNT(*) FROM sync_operations").Scan(&opCount)

	stats["objects_count"] = objCount
	stats["operations_count"] = opCount
	stats["filepath"] = sm.filepath
	stats["timestamp"] = time.Now()

	return stats
}

// Close closes the database connection
func (sm *StorageManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}
