// Package database provides database access and persistence
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"cadastre_ia/pkg/cadastre"
)

// PostgresDB implements CadastreDB interface using PostgreSQL
type PostgresDB struct {
	db     *sql.DB
	logger *log.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(connString string, logger *log.Logger) (*PostgresDB, error) {
	if logger == nil {
		logger = log.New(log.Writer(), "[PostgresDB] ", log.LstdFlags)
	}

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	logger.Println("Connected to PostgreSQL")
	return &PostgresDB{db: db, logger: logger}, nil
}

// StoreCadastralEntities stores multiple entities in the database
func (p *PostgresDB) StoreCadastralEntities(ctx context.Context, entities []cadastre.Entity) error {
	if len(entities) == 0 {
		return fmt.Errorf("no entities to store")
	}

	p.logger.Printf("Storing %d entities in database\n", len(entities))

	// Begin transaction
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, entity := range entities {
		// Serialize attributes to JSON
		attrsJSON, err := json.Marshal(entity.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes for %s: %w", entity.ID, err)
		}

		// Convert geometry to GeoJSON string for PostGIS
		geomJSON, err := json.Marshal(entity.Geometry)
		if err != nil {
			return fmt.Errorf("failed to marshal geometry for %s: %w", entity.ID, err)
		}

		// Insert entity
		query := `
			INSERT INTO cadastral_entities (id, code, type, region, admin_unit, geometry, attributes, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, ST_GeomFromGeoJSON($6), $7, $8, $9)
			ON CONFLICT (code) DO UPDATE SET
				attributes = EXCLUDED.attributes,
				updated_at = EXCLUDED.updated_at,
				version = version + 1
		`

		now := time.Now().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx, query,
			entity.ID,
			entity.Code,
			entity.Type,
			entity.Region,
			entity.AdminUnit,
			string(geomJSON),
			attrsJSON,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert entity %s: %w", entity.ID, err)
		}

		// Log to audit trail
		p.logger.Printf("  Stored: %s → %s\n", entity.ID, entity.Code)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.logger.Printf("Successfully stored %d entities\n", len(entities))
	return nil
}

// GetEntity retrieves a single entity by cadastre code
func (p *PostgresDB) GetEntity(ctx context.Context, code string) (*cadastre.Entity, error) {
	query := `
		SELECT id, code, type, region, admin_unit,
		       ST_AsGeoJSON(geometry) as geometry, attributes, created_at, updated_at
		FROM cadastral_entities
		WHERE code = $1 AND deleted_at IS NULL
	`

	row := p.db.QueryRowContext(ctx, query, code)

	var entity cadastre.Entity
	var geomStr, attrsStr string
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&entity.ID,
		&entity.Code,
		&entity.Type,
		&entity.Region,
		&entity.AdminUnit,
		&geomStr,
		&attrsStr,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity not found: %s", code)
		}
		return nil, fmt.Errorf("failed to query entity: %w", err)
	}

	// Unmarshal geometry
	if err := json.Unmarshal([]byte(geomStr), &entity.Geometry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geometry: %w", err)
	}

	// Unmarshal attributes
	if err := json.Unmarshal([]byte(attrsStr), &entity.Attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	entity.CreatedAt = createdAt.Format(time.RFC3339)
	entity.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &entity, nil
}

// UpdateEntity updates an existing entity
func (p *PostgresDB) UpdateEntity(ctx context.Context, entity *cadastre.Entity) error {
	if entity.Code == "" {
		return fmt.Errorf("entity code is required for update")
	}

	// Serialize attributes
	attrsJSON, err := json.Marshal(entity.Attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Serialize geometry
	geomJSON, err := json.Marshal(entity.Geometry)
	if err != nil {
		return fmt.Errorf("failed to marshal geometry: %w", err)
	}

	query := `
		UPDATE cadastral_entities
		SET geometry = ST_GeomFromGeoJSON($1),
		    attributes = $2,
		    updated_at = $3,
		    version = version + 1
		WHERE code = $4 AND deleted_at IS NULL
		RETURNING id
	`

	now := time.Now().Format(time.RFC3339)
	var returnedID string
	err = p.db.QueryRowContext(ctx, query, string(geomJSON), attrsJSON, now, entity.Code).Scan(&returnedID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entity not found: %s", entity.Code)
		}
		return fmt.Errorf("failed to update entity: %w", err)
	}

	p.logger.Printf("Updated entity: %s\n", entity.Code)
	return nil
}

// DeleteEntity soft-deletes an entity
func (p *PostgresDB) DeleteEntity(ctx context.Context, code string) error {
	query := `
		UPDATE cadastral_entities
		SET deleted_at = $1, updated_at = $1
		WHERE code = $2 AND deleted_at IS NULL
	`

	now := time.Now().Format(time.RFC3339)
	result, err := p.db.ExecContext(ctx, query, now, code)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity not found: %s", code)
	}

	p.logger.Printf("Deleted entity: %s\n", code)
	return nil
}

// GetEntitiesByBBox retrieves entities within a geographic bounding box
func (p *PostgresDB) GetEntitiesByBBox(ctx context.Context, minX, minY, maxX, maxY float64, zoomLevel int) ([]interface{}, error) {
	query := `
		SELECT id, code, type, region, admin_unit,
		       ST_AsGeoJSON(geometry) as geometry, attributes, created_at, updated_at
		FROM cadastral_entities
		WHERE ST_Intersects(geometry, ST_MakeEnvelope($1, $2, $3, $4, 4326))
		  AND deleted_at IS NULL
		ORDER BY ST_Area(geometry) DESC
		LIMIT 1000
	`

	rows, err := p.db.QueryContext(ctx, query, minX, minY, maxX, maxY)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer rows.Close()

	entities := make([]interface{}, 0)

	for rows.Next() {
		var entity cadastre.Entity
		var geomStr, attrsStr string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&entity.ID,
			&entity.Code,
			&entity.Type,
			&entity.Region,
			&entity.AdminUnit,
			&geomStr,
			&attrsStr,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			p.logger.Printf("Error scanning row: %v\n", err)
			continue
		}

		// Unmarshal geometry
		if err := json.Unmarshal([]byte(geomStr), &entity.Geometry); err != nil {
			p.logger.Printf("Error unmarshaling geometry: %v\n", err)
			continue
		}

		// Unmarshal attributes
		if err := json.Unmarshal([]byte(attrsStr), &entity.Attributes); err != nil {
			p.logger.Printf("Error unmarshaling attributes: %v\n", err)
			continue
		}

		entity.CreatedAt = createdAt.Format(time.RFC3339)
		entity.UpdatedAt = updatedAt.Format(time.RFC3339)

		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	p.logger.Printf("Retrieved %d entities for bbox z=%d\n", len(entities), zoomLevel)
	return entities, nil
}

// GetEntitiesByRegion retrieves all entities in a region
func (p *PostgresDB) GetEntitiesByRegion(ctx context.Context, region string) ([]cadastre.Entity, error) {
	query := `
		SELECT id, code, type, region, admin_unit,
		       ST_AsGeoJSON(geometry) as geometry, attributes, created_at, updated_at
		FROM cadastral_entities
		WHERE region = $1 AND deleted_at IS NULL
		ORDER BY admin_unit, code
	`

	rows, err := p.db.QueryContext(ctx, query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer rows.Close()

	entities := make([]cadastre.Entity, 0)

	for rows.Next() {
		var entity cadastre.Entity
		var geomStr, attrsStr string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&entity.ID,
			&entity.Code,
			&entity.Type,
			&entity.Region,
			&entity.AdminUnit,
			&geomStr,
			&attrsStr,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(geomStr), &entity.Geometry); err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(attrsStr), &entity.Attributes); err != nil {
			continue
		}

		entity.CreatedAt = createdAt.Format(time.RFC3339)
		entity.UpdatedAt = updatedAt.Format(time.RFC3339)

		entities = append(entities, entity)
	}

	return entities, rows.Err()
}

// GetStatistics returns statistics about cadastral entities
func (p *PostgresDB) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_entities,
			COUNT(CASE WHEN type = 'building' THEN 1 END) as buildings,
			COUNT(CASE WHEN type = 'parcel' THEN 1 END) as parcels,
			COUNT(CASE WHEN type = 'poi' THEN 1 END) as pois,
			COUNT(DISTINCT region) as regions,
			COUNT(DISTINCT admin_unit) as admin_units,
			ST_AsText(ST_Extent(geometry)) as spatial_extent
		FROM cadastral_entities
		WHERE deleted_at IS NULL
	`

	row := p.db.QueryRowContext(ctx, query)

	var stats struct {
		Total       int64
		Buildings   int64
		Parcels     int64
		POIs        int64
		Regions     int64
		AdminUnits  int64
		SpatialExt  string
	}

	err := row.Scan(&stats.Total, &stats.Buildings, &stats.Parcels, &stats.POIs,
		&stats.Regions, &stats.AdminUnits, &stats.SpatialExt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return map[string]interface{}{
		"total_entities":  stats.Total,
		"buildings":       stats.Buildings,
		"parcels":         stats.Parcels,
		"pois":            stats.POIs,
		"regions":         stats.Regions,
		"admin_units":     stats.AdminUnits,
		"spatial_extent":  stats.SpatialExt,
	}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}
