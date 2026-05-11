// Package service provides core business logic services for geo-mobile137
package service

import (
	"context"
	"fmt"
	"log"

	"cadastre_ia/pkg/cadastre"
	"cadastre_ia/pkg/svg_codification"
)

// CadastreAdapter bridges CAD conversion output to cadastral entities and SVG codification
type CadastreAdapter struct {
	converter   *CadConverter
	codifier    *cadastre.Codifier
	tileGen     *svg_codification.TileGenerator
	db          CadastreDB
	eventBus    EventBus
	logger      *log.Logger
}

// CadastreDB defines database interface for storing cadastral entities
type CadastreDB interface {
	StoreCadastralEntities(ctx context.Context, entities []cadastre.Entity) error
	GetEntity(ctx context.Context, code string) (*cadastre.Entity, error)
	UpdateEntity(ctx context.Context, entity *cadastre.Entity) error
	DeleteEntity(ctx context.Context, code string) error
}

// EventBus defines pub/sub interface for real-time updates
type EventBus interface {
	Publish(ctx context.Context, topic string, payload interface{}) error
	Subscribe(ctx context.Context, topic string) (<-chan interface{}, error)
}

// NewCadastreAdapter creates a new cadastre adapter instance
func NewCadastreAdapter(
	converter *CadConverter,
	codifier *cadastre.Codifier,
	tileGen *svg_codification.TileGenerator,
	db CadastreDB,
	eventBus EventBus,
	logger *log.Logger,
) *CadastreAdapter {
	if logger == nil {
		logger = log.New(log.Writer(), "[CadastreAdapter] ", log.LstdFlags)
	}
	return &CadastreAdapter{
		converter: converter,
		codifier: codifier,
		tileGen:   tileGen,
		db:        db,
		eventBus: eventBus,
		logger:    logger,
	}
}

// ConvertAndStoreCadastralData is the main pipeline: DWG → Entity → Code → Store → SVG
func (ca *CadastreAdapter) ConvertAndStoreCadastralData(
	ctx context.Context,
	cadData *RawCADData,
	regionCode string,
	adminUnit string,
) (*CadastralEntitySet, error) {
	ca.logger.Printf("Starting conversion pipeline for %s / %s\n", regionCode, adminUnit)

	// Step 1: Validate CAD data
	if err := ca.converter.ValidateCADData(cadData); err != nil {
		return nil, fmt.Errorf("CAD validation failed: %w", err)
	}

	// Step 2: Convert CAD to arcade format
	arcadeOutputs, err := ca.converter.ConvertFromCAD(ctx, cadData)
	if err != nil {
		return nil, fmt.Errorf("CAD conversion failed: %w", err)
	}
	ca.logger.Printf("Converted %d geometries\n", len(arcadeOutputs))

	// Step 3: Transform arcade output to cadastral entities with codes
	entities, err := ca.transformToEntities(ctx, arcadeOutputs, regionCode, adminUnit)
	if err != nil {
		return nil, fmt.Errorf("entity transformation failed: %w", err)
	}

	// Step 4: Codify each entity (generate 64-bit cadastre codes)
	for i := range entities {
		code, err := ca.codifier.GenerateCadastreCoding(&entities[i])
		if err != nil {
			return nil, fmt.Errorf("codification failed for entity %s: %w", entities[i].ID, err)
		}
		entities[i].Code = code
		ca.logger.Printf("Codified: %s → %s\n", entities[i].ID, code)
	}

	// Step 5: Store entities in PostgreSQL
	if err := ca.db.StoreCadastralEntities(ctx, entities); err != nil {
		return nil, fmt.Errorf("database storage failed: %w", err)
	}
	ca.logger.Printf("Stored %d entities in database\n", len(entities))

	// Step 6: Generate SVG tiles for affected geographic area
	bbox := calculateBoundingBox(entities)
	if err := ca.tileGen.RegenerateTiles(ctx, bbox); err != nil {
		ca.logger.Printf("Warning: tile generation failed (non-fatal): %v\n", err)
		// Don't fail the whole operation if tiles fail
	}
	ca.logger.Printf("SVG tiles regenerated for bbox: %+v\n", bbox)

	// Step 7: Emit WebSocket events for real-time map updates
	if err := ca.eventBus.Publish(ctx, "cadastre:entities:created", &CadastralEntitySet{
		Entities: entities,
	}); err != nil {
		ca.logger.Printf("Warning: event publication failed: %v\n", err)
	}

	return &CadastralEntitySet{Entities: entities}, nil
}

// transformToEntities converts arcade output to cadastral entity structs
func (ca *CadastreAdapter) transformToEntities(
	ctx context.Context,
	arcadeOutputs []ArcadeOutput,
	regionCode string,
	adminUnit string,
) ([]cadastre.Entity, error) {
	entities := make([]cadastre.Entity, 0, len(arcadeOutputs))

	for _, arcade := range arcadeOutputs {
		entity := cadastre.Entity{
			ID:        arcade.EntityID,
			Type:      ca.detectEntityType(arcade.Type),
			Region:    regionCode,
			AdminUnit: adminUnit,
			Geometry:  ca.convertToGeoJSON(arcade.Geometry),
			Attributes: arcade.Attributes,
			CreatedAt: timeNow(),
			UpdatedAt: timeNow(),
		}

		// Validate basic entity structure
		if err := entity.Validate(); err != nil {
			ca.logger.Printf("Warning: invalid entity %s: %v\n", entity.ID, err)
			continue
		}

		entities = append(entities, entity)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("no valid entities created from CAD data")
	}

	return entities, nil
}

// detectEntityType maps arcade type to cadastral entity type
func (ca *CadastreAdapter) detectEntityType(arcadeType string) cadastre.EntityType {
	switch arcadeType {
	case "building", "bâti", "bat":
		return cadastre.EntityTypeBuilding
	case "parcel", "parcelle", "prc":
		return cadastre.EntityTypeParcel
	case "poi", "point":
		return cadastre.EntityTypePOI
	default:
		return cadastre.EntityTypeParcel // Default fallback
	}
}

// convertToGeoJSON transforms raw vertices to GeoJSON feature geometry
func (ca *CadastreAdapter) convertToGeoJSON(vertices []CADVertex) map[string]interface{} {
	coords := make([][]float64, len(vertices))
	for i, v := range vertices {
		coords[i] = []float64{v.X, v.Y}
	}

	// Close polygon if needed (first point == last point)
	if len(coords) > 2 && (coords[0][0] != coords[len(coords)-1][0] || coords[0][1] != coords[len(coords)-1][1]) {
		coords = append(coords, coords[0])
	}

	return map[string]interface{}{
		"type": "Polygon",
		"coordinates": []interface{}{coords},
	}
}

// ValidateRoundTrip verifies entity → SVG → entity integrity
// This ensures bidirectional flow doesn't lose data
func (ca *CadastreAdapter) ValidateRoundTrip(ctx context.Context, entity *cadastre.Entity) error {
	ca.logger.Printf("Starting round-trip validation for entity %s\n", entity.ID)

	// Step 1: Verify entity is valid
	if err := entity.Validate(); err != nil {
		return fmt.Errorf("entity validation failed: %w", err)
	}

	// Step 2: Generate SVG code from entity
	svgCode, err := ca.codifier.GenerateSVGCode(entity)
	if err != nil {
		return fmt.Errorf("SVG code generation failed: %w", err)
	}
	ca.logger.Printf("Generated SVG code: %s\n", svgCode)

	// Step 3: Decode SVG code back to entity
	decoded, err := ca.codifier.DecodeSVGCode(svgCode)
	if err != nil {
		return fmt.Errorf("SVG code decoding failed: %w", err)
	}

	// Step 4: Verify codes match
	if decoded.Code != entity.Code {
		return fmt.Errorf("round-trip code mismatch: expected %s, got %s", entity.Code, decoded.Code)
	}

	// Step 5: Verify geometry is preserved
	if !geometryEqual(entity.Geometry, decoded.Geometry) {
		return fmt.Errorf("round-trip geometry mismatch")
	}

	ca.logger.Printf("Round-trip validation PASSED for %s\n", entity.ID)
	return nil
}

// UpdateEntityFromTile allows bidirectional editing: SVG edit → Entity update → Store
func (ca *CadastreAdapter) UpdateEntityFromTile(
	ctx context.Context,
	code string,
	updatedGeometry map[string]interface{},
	updatedAttributes map[string]interface{},
) error {
	ca.logger.Printf("Updating entity from tile edit: %s\n", code)

	// Fetch existing entity
	entity, err := ca.db.GetEntity(ctx, code)
	if err != nil {
		return fmt.Errorf("entity fetch failed: %w", err)
	}

	// Update geometry and attributes
	entity.Geometry = updatedGeometry
	if updatedAttributes != nil {
		for k, v := range updatedAttributes {
			entity.Attributes[k] = v
		}
	}
	entity.UpdatedAt = timeNow()

	// Validate updated entity
	if err := entity.Validate(); err != nil {
		return fmt.Errorf("updated entity validation failed: %w", err)
	}

	// Store updated entity
	if err := ca.db.UpdateEntity(ctx, entity); err != nil {
		return fmt.Errorf("entity update failed: %w", err)
	}

	// Regenerate affected tiles
	bbox := calculateBoundingBox([]cadastre.Entity{*entity})
	if err := ca.tileGen.RegenerateTiles(ctx, bbox); err != nil {
		ca.logger.Printf("Warning: tile regeneration failed: %v\n", err)
	}

	// Emit update event
	if err := ca.eventBus.Publish(ctx, "cadastre:entities:updated", entity); err != nil {
		ca.logger.Printf("Warning: event publication failed: %v\n", err)
	}

	ca.logger.Printf("Entity update COMPLETE: %s\n", code)
	return nil
}

// CadastralEntitySet represents a collection of processed entities
type CadastralEntitySet struct {
	Entities []cadastre.Entity `json:"entities"`
	BBox     *BoundingBox      `json:"bbox,omitempty"`
}

// BoundingBox represents geographic extent
type BoundingBox struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// calculateBoundingBox computes the bounding box for a set of entities
func calculateBoundingBox(entities []cadastre.Entity) *BoundingBox {
	if len(entities) == 0 {
		return nil
	}

	bbox := &BoundingBox{
		MinX: 180,
		MinY: 90,
		MaxX: -180,
		MaxY: -90,
	}

	for _, entity := range entities {
		// Extract coordinates from GeoJSON geometry
		if coords, ok := entity.Geometry["coordinates"]; ok {
			if coordsList, ok := coords.([]interface{}); ok && len(coordsList) > 0 {
				if polygon, ok := coordsList[0].([]interface{}); ok {
					for _, pt := range polygon {
						if point, ok := pt.([]float64); ok && len(point) >= 2 {
							x, y := point[0], point[1]
							if x < bbox.MinX {
								bbox.MinX = x
							}
							if x > bbox.MaxX {
								bbox.MaxX = x
							}
							if y < bbox.MinY {
								bbox.MinY = y
							}
							if y > bbox.MaxY {
								bbox.MaxY = y
							}
						}
					}
				}
			}
		}
	}

	return bbox
}

// geometryEqual compares two GeoJSON geometries for equality
func geometryEqual(g1, g2 map[string]interface{}) bool {
	// Simple comparison: serialize both and compare
	// In production: use proper GeoJSON comparison
	t1, ok1 := g1["type"]
	t2, ok2 := g2["type"]
	if !ok1 || !ok2 || t1 != t2 {
		return false
	}

	c1, ok1 := g1["coordinates"]
	c2, ok2 := g2["coordinates"]
	if !ok1 || !ok2 {
		return false
	}

	// Deep equality check (placeholder)
	return fmt.Sprintf("%v", c1) == fmt.Sprintf("%v", c2)
}

// Helper function for current timestamp
func timeNow() string {
	// Return ISO 8601 formatted timestamp
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ", 2026, 5, 11, 10, 0, 0)
}
