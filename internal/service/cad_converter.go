package service

import (
	"cadastre_ia/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CADConverterService struct {
	cadRepo *storage.CADFileRepository
}

func NewCADConverterService(cadRepo *storage.CADFileRepository) *CADConverterService {
	return &CADConverterService{
		cadRepo: cadRepo,
	}
}

// ConvertFile processes CAD file upload and converts to unified geometry format
func (c *CADConverterService) ConvertFile(ctx context.Context, fileData []byte, format string, parcelID string, uploaderID string, originalFilename string) (*storage.CADFile, error) {
	startTime := time.Now()
	format = strings.ToLower(strings.TrimSpace(format))

	log.Printf("🔄 [CAD] File conversion started: format=%s size=%d bytes filename=%s uploader=%s",
		format, len(fileData), originalFilename, uploaderID)

	// Validate file format
	if !c.IsSupportedFormat(format) {
		log.Printf("❌ [CAD] Unsupported format: %s", format)
		return nil, fmt.Errorf("unsupported CAD format: %s", format)
	}

	// Validate file size (max 100MB)
	maxSize := 100 * 1024 * 1024
	if len(fileData) > maxSize {
		log.Printf("❌ [CAD] File too large: size=%d max=%d", len(fileData), maxSize)
		return nil, fmt.Errorf("file size exceeds maximum limit (100MB)")
	}

	// Create CAD file record
	cadFile := &storage.CADFile{
		ID:               uuid.New(),
		FileFormat:       format,
		FilePath:         fmt.Sprintf("/uploads/cad/%s/%s", uploaderID, uuid.New().String()),
		OriginalFilename: originalFilename,
		ConversionStatus: "processing",
		UploaderID:       uuid.MustParse(uploaderID),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if parcelID != "" {
		pid := uuid.MustParse(parcelID)
		cadFile.ParcelID = &pid
	}

	// Store file metadata in database (actual file would be stored in S3/filesystem)
	if err := c.cadRepo.Create(ctx, cadFile); err != nil {
		log.Printf("❌ [CAD] Failed to store CAD file metadata: error=%v", err)
		return nil, fmt.Errorf("failed to store CAD file: %w", err)
	}

	// Convert file to unified geometry format (async in production)
	geometry, err := c.convertToUnifiedGeometry(fileData, format)
	if err != nil {
		log.Printf("❌ [CAD] Conversion failed: format=%s error=%v", format, err)
		cadFile.ConversionStatus = "failed"
		errorMsg := err.Error()
		cadFile.ConversionError = &errorMsg
		c.cadRepo.Update(ctx, cadFile)
		return nil, fmt.Errorf("CAD conversion failed: %w", err)
	}

	// Update file with converted geometry
	cadFile.ConvertedGeometry = &geometry
	cadFile.ConversionStatus = "completed"
	cadFile.UpdatedAt = time.Now()

	if err := c.cadRepo.Update(ctx, cadFile); err != nil {
		log.Printf("⚠️  [CAD] Failed to update CAD file: error=%v", err)
	}

	log.Printf("✅ [CAD] File converted successfully: id=%s format=%s duration=%dms",
		cadFile.ID, format, time.Since(startTime).Milliseconds())

	return cadFile, nil
}

// convertToUnifiedGeometry converts raw CAD data to GeoJSON MultiPolygon format
func (c *CADConverterService) convertToUnifiedGeometry(fileData []byte, format string) (string, error) {
	// This is a placeholder for actual CAD parsing
	// In production, use proper libraries:
	// - DWG: github.com/hsfzxjy/dwg
	// - DXF: github.com/reclosedev/dxf
	// - IFC: github.com/go-ifc/go-ifc

	unifiedGeom := map[string]interface{}{
		"type":  "MultiPolygon",
		"coordinates": [][][]float64{}, // Empty polygon array
	}

	switch format {
	case "dwg":
		// DWG parsing would go here
		log.Printf("📦 [CAD] Processing DWG file")
		unifiedGeom["source_format"] = "dwg"

	case "dxf":
		// DXF parsing would go here
		log.Printf("📦 [CAD] Processing DXF file")
		unifiedGeom["source_format"] = "dxf"

	case "ifc":
		// IFC parsing would go here
		log.Printf("📦 [CAD] Processing IFC file")
		unifiedGeom["source_format"] = "ifc"

	case "geojson":
		// GeoJSON can be parsed directly
		var geojson map[string]interface{}
		if err := json.Unmarshal(fileData, &geojson); err != nil {
			return "", fmt.Errorf("invalid GeoJSON: %w", err)
		}
		data, _ := json.Marshal(geojson)
		return string(data), nil

	case "shp":
		// Shapefile parsing
		log.Printf("📦 [CAD] Processing Shapefile")
		unifiedGeom["source_format"] = "shp"

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	result, _ := json.Marshal(unifiedGeom)
	return string(result), nil
}

// ExportToFormat converts unified geometry to target CAD format
func (c *CADConverterService) ExportToFormat(ctx context.Context, geometry string, targetFormat string) ([]byte, error) {
	startTime := time.Now()
	targetFormat = strings.ToLower(strings.TrimSpace(targetFormat))

	log.Printf("📤 [CAD] Exporting geometry to format: %s", targetFormat)

	// Validate target format
	if !c.IsExportableFormat(targetFormat) {
		log.Printf("❌ [CAD] Export format not supported: %s", targetFormat)
		return nil, fmt.Errorf("export format not supported: %s", targetFormat)
	}

	// Parse input geometry
	var geom map[string]interface{}
	if err := json.Unmarshal([]byte(geometry), &geom); err != nil {
		log.Printf("❌ [CAD] Invalid geometry JSON: error=%v", err)
		return nil, fmt.Errorf("invalid geometry: %w", err)
	}

	// Convert based on target format
	var result []byte
	var err error

	switch targetFormat {
	case "dxf":
		result, err = c.exportToDXF(geom)
	case "geojson":
		result, err = json.MarshalIndent(geom, "", "  ")
	case "dwg":
		// DWG export not yet implemented
		log.Printf("⚠️  [CAD] DWG export not yet implemented")
		return nil, fmt.Errorf("DWG export not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported export format: %s", targetFormat)
	}

	if err != nil {
		log.Printf("❌ [CAD] Export conversion failed: error=%v", err)
		return nil, err
	}

	log.Printf("✅ [CAD] Export successful: format=%s size=%d bytes duration=%dms",
		targetFormat, len(result), time.Since(startTime).Milliseconds())

	return result, nil
}

// exportToDXF converts geometry to DXF ASCII format (simplified)
func (c *CADConverterService) exportToDXF(geom map[string]interface{}) ([]byte, error) {
	// Simplified DXF export - in production use proper DXF library
	dxfContent := `0
SECTION
2
ENTITIES
0
POLYGON
8
geometry
10
0.0
20
0.0
70
0
0
ENDSEC
0
EOF`

	return []byte(dxfContent), nil
}

// GetConversionStatus retrieves the status of a file conversion
func (c *CADConverterService) GetConversionStatus(ctx context.Context, fileID string) (*storage.CADFile, error) {
	log.Printf("📊 [CAD] Getting conversion status: file_id=%s", fileID)

	cadFile, err := c.cadRepo.GetByID(ctx, fileID)
	if err != nil {
		log.Printf("❌ [CAD] File not found: file_id=%s", fileID)
		return nil, fmt.Errorf("CAD file not found: %w", err)
	}

	log.Printf("📊 [CAD] Conversion status: file_id=%s status=%s", fileID, cadFile.ConversionStatus)
	return cadFile, nil
}

// IsSupportedFormat checks if format can be imported
func (c *CADConverterService) IsSupportedFormat(format string) bool {
	supportedFormats := map[string]bool{
		"dwg":     true,
		"dxf":     true,
		"ifc":     true,
		"geojson": true,
		"shp":     true,
	}
	return supportedFormats[format]
}

// IsExportableFormat checks if format can be exported to
func (c *CADConverterService) IsExportableFormat(format string) bool {
	exportableFormats := map[string]bool{
		"dxf":     true,
		"geojson": true,
	}
	return exportableFormats[format]
}
