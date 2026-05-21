// Package svg_codification handles SVG tile generation and caching for cadastral entities
package svg_codification

import (
	"context"
	"fmt"
	"log"
	"math"
)

// TileGenerator creates SVG tiles from cadastral entities with three-level caching
type TileGenerator struct {
	cache     TileCache
	renderer  *SVGRenderer
	db        EntityStore
	logger    *log.Logger
}

// TileCache defines the caching interface (Browser → Redis → PostgreSQL)
type TileCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}

// EntityStore defines the database interface for entity retrieval
type EntityStore interface {
	GetEntitiesByBBox(ctx context.Context, minX, minY, maxX, maxY float64, zoomLevel int) ([]interface{}, error)
}

// NewTileGenerator creates a new tile generator instance
func NewTileGenerator(cache TileCache, renderer *SVGRenderer, db EntityStore, logger *log.Logger) *TileGenerator {
	if logger == nil {
		logger = log.New(log.Writer(), "[TileGenerator] ", log.LstdFlags)
	}
	return &TileGenerator{
		cache:    cache,
		renderer: renderer,
		db:       db,
		logger:   logger,
	}
}

// GetTile retrieves or generates a tile for the given coordinates
// Uses Web Mercator projection for XYZ tile coordinates
func (tg *TileGenerator) GetTile(ctx context.Context, z, x, y int) ([]byte, error) {
	tg.logger.Printf("Fetching tile z=%d x=%d y=%d\n", z, x, y)

	// Generate cache key
	cacheKey := fmt.Sprintf("cadastre:tile:z%d:x%d:y%d", z, x, y)

	// Check cache first
	if cachedTile, err := tg.cache.Get(ctx, cacheKey); err == nil {
		tg.logger.Printf("Cache HIT for %s\n", cacheKey)
		return cachedTile, nil
	}
	tg.logger.Printf("Cache MISS for %s, regenerating\n", cacheKey)

	// Not in cache: generate new tile
	tile, err := tg.generateTile(ctx, z, x, y)
	if err != nil {
		return nil, fmt.Errorf("tile generation failed: %w", err)
	}

	// Store in cache with TTL based on zoom level
	// Higher zooms (more detail) = shorter TTL, lower zooms (less detail) = longer TTL
	ttl := calculateCacheTTL(z)
	if err := tg.cache.Set(ctx, cacheKey, tile, ttl); err != nil {
		tg.logger.Printf("Warning: cache store failed: %v\n", err)
		// Don't fail if cache store fails, tile is still generated
	}

	return tile, nil
}

// RegenerateTiles regenerates all affected tiles for a bounding box
// Called when entities are created, updated, or deleted
func (tg *TileGenerator) RegenerateTiles(ctx context.Context, bbox *BoundingBox) error {
	if bbox == nil {
		return fmt.Errorf("bounding box is nil")
	}

	tg.logger.Printf("Regenerating tiles for bbox: %.2f,%.2f to %.2f,%.2f\n",
		bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY)

	// Determine affected zoom levels (z5-z18 is typical)
	affectedZooms := determineAffectedZooms(bbox)
	tg.logger.Printf("Affected zoom levels: %v\n", affectedZooms)

	// For each zoom level, determine affected tiles and clear cache
	totalRegenerated := 0
	for _, z := range affectedZooms {
		tiles := getBoundingTiles(z, bbox)
		for _, tile := range tiles {
			cacheKey := fmt.Sprintf("cadastre:tile:z%d:x%d:y%d", z, tile.X, tile.Y)
			if err := tg.cache.Delete(ctx, cacheKey); err != nil {
				tg.logger.Printf("Warning: cache delete failed for %s: %v\n", cacheKey, err)
			}
			totalRegenerated++
		}
	}

	tg.logger.Printf("Invalidated %d tile cache entries\n", totalRegenerated)
	return nil
}

// generateTile creates an SVG tile from entities in the tile's geographic bounds
func (tg *TileGenerator) generateTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Convert XYZ tile coordinates to lat/lon bounding box
	bbox := tileToLatLon(z, x, y)

	// Fetch entities within this tile's bounds
	// Detail level depends on zoom: z0-5 (communes only), z6-10 (parcels), z11-18 (buildings+details)
	entities, err := tg.db.GetEntitiesByBBox(ctx, bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY, z)
	if err != nil {
		return nil, fmt.Errorf("entity fetch failed: %w", err)
	}
	tg.logger.Printf("Fetched %d entities for tile z=%d\n", len(entities), z)

	// Render SVG tile with appropriate detail level
	svg, err := tg.renderer.RenderTile(ctx, z, x, y, entities)
	if err != nil {
		return nil, fmt.Errorf("SVG rendering failed: %w", err)
	}

	return svg, nil
}

// BoundingBox represents geographic extent
type BoundingBox struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// TileCoord represents a single tile coordinate
type TileCoord struct {
	Z int
	X int
	Y int
}

// Helper functions

// calculateCacheTTL returns cache TTL in seconds based on zoom level
func calculateCacheTTL(z int) int {
	switch {
	case z <= 5:
		return 86400 * 7 // 7 days for low-detail commune tiles
	case z <= 10:
		return 86400 * 3 // 3 days for medium-detail parcel tiles
	case z <= 15:
		return 3600 * 12 // 12 hours for building-level detail
	default:
		return 3600 // 1 hour for high-detail tiles
	}
}

// determineAffectedZooms returns the zoom levels affected by a bbox change
// Typical range: z5 (regions) through z18 (street level)
func determineAffectedZooms(bbox *BoundingBox) []int {
	// For simplicity, regenerate z5-z16
	// In production: calculate minimal affected range
	zooms := make([]int, 0)
	for z := 5; z <= 16; z++ {
		zooms = append(zooms, z)
	}
	return zooms
}

// getBoundingTiles returns all tiles at zoom z that overlap the bbox
func getBoundingTiles(z int, bbox *BoundingBox) []TileCoord {
	tiles := make([]TileCoord, 0)

	// Convert bbox to tile coordinates
	minTile := latLonToTile(z, bbox.MinX, bbox.MaxY) // Top-left (min lat)
	maxTile := latLonToTile(z, bbox.MaxX, bbox.MinY) // Bottom-right (max lat)

	for x := minTile.X; x <= maxTile.X; x++ {
		for y := minTile.Y; y <= maxTile.Y; y++ {
			tiles = append(tiles, TileCoord{Z: z, X: x, Y: y})
		}
	}

	return tiles
}

// tileToLatLon converts XYZ tile coordinates to geographic bounding box
func tileToLatLon(z, x, y int) *BoundingBox {
	n := float64(int64(1) << uint(z))
	lonMin := float64(x)/n*360.0 - 180.0
	lonMax := float64(x+1)/n*360.0 - 180.0

	// Mercator projection for latitude
	latRad := 2.0 * 3.141592653589793 * (float64(y) / n)
	latMin := 2*3.141592653589793 - latRad
	latMin = 180.0 / 3.141592653589793 * (2.0*atan(exp(latMin)) - 3.141592653589793/2.0)

	latRad = 2.0 * 3.141592653589793 * (float64(y+1) / n)
	latMax := 2*3.141592653589793 - latRad
	latMax = 180.0 / 3.141592653589793 * (2.0*atan(exp(latMax)) - 3.141592653589793/2.0)

	return &BoundingBox{
		MinX: lonMin,
		MaxX: lonMax,
		MinY: latMin,
		MaxY: latMax,
	}
}

// latLonToTile converts geographic coordinates to XYZ tile coordinates
func latLonToTile(z int, lon, lat float64) TileCoord {
	n := float64(int64(1) << uint(z))

	// Convert to radians
	latRad := lat * 3.141592653589793 / 180.0

	x := int(n * (lon + 180.0) / 360.0)
	y := int(n * (1.0 - (math.Log(math.Tan(latRad)+1.0/math.Cos(latRad)) / 3.141592653589793) / 2.0) / 2.0)

	return TileCoord{Z: z, X: x, Y: y}
}

// Trigonometric helpers
func atan(x float64) float64 {
	if x < -1 {
		return -1.5707963267948966 + atan(1.0/x) + 3.141592653589793
	}
	if x > 1 {
		return 1.5707963267948966 - atan(1.0/x)
	}
	// Approximation for atan
	return x / (1.0 + 0.28*x*x)
}

func tan(x float64) float64 {
	// Approximation for tan
	return x / (1.0 - 0.333333*x*x)
}

func cos(x float64) float64 {
	// Approximation for cos
	return 1.0 - (x*x)/2.0 + (x*x*x*x)/24.0
}

func logApprox(x float64) float64 {
	// Simple approximation (use math.Log in production)
	if x <= 0 {
		return 0
	}
	result := 0.0
	for x > 2.0 {
		x /= 2.0
		result += 0.693147
	}
	return result
}

func exp(x float64) float64 {
	// Simple approximation (use math.Exp in production)
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
	}
	return result
}
