// Package handlers provides HTTP request handlers for cadastre APIs
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cadastre_ia/pkg/service"
	"cadastre_ia/pkg/svg_codification"
)

// CadastreHandlers handles cadastral API endpoints
type CadastreHandlers struct {
	adapter   *service.CadastreAdapter
	tileGen   *svg_codification.TileGenerator
	converter *service.CadConverter
	logger    *log.Logger
}

// NewCadastreHandlers creates a new cadastre handlers instance
func NewCadastreHandlers(
	adapter *service.CadastreAdapter,
	tileGen *svg_codification.TileGenerator,
	converter *service.CadConverter,
	logger *log.Logger,
) *CadastreHandlers {
	if logger == nil {
		logger = log.New(log.Writer(), "[CadastreHandlers] ", log.LstdFlags)
	}
	return &CadastreHandlers{
		adapter:   adapter,
		tileGen:   tileGen,
		converter: converter,
		logger:    logger,
	}
}

// GetTileHandler handles GET /api/v1/cadastre/tiles/{z}/{x}/{y}
// Returns SVG tile with cadastral entities and codification
func (h *CadastreHandlers) GetTileHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("GetTileHandler: %s\n", r.URL.Path)

	// Extract path parameters
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 7 {
		h.respondError(w, http.StatusBadRequest, "Invalid path format")
		return
	}

	z, x, y := pathParts[5], pathParts[6], pathParts[7]

	// Parse zoom level
	zInt, err := strconv.Atoi(z)
	if err != nil || zInt < 0 || zInt > 28 {
		h.respondError(w, http.StatusBadRequest, "Invalid zoom level")
		return
	}

	// Parse tile coordinates
	xInt, err := strconv.Atoi(x)
	if err != nil || xInt < 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid X coordinate")
		return
	}

	yInt, err := strconv.Atoi(y)
	if err != nil || yInt < 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid Y coordinate")
		return
	}

	// Fetch tile from cache or generate
	tile, err := h.tileGen.GetTile(r.Context(), zInt, xInt, yInt)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Tile generation failed: %v", err))
		return
	}

	// Return SVG with proper headers
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", fmt.Sprintf("\"%d_%d_%d\"", zInt, xInt, yInt))
	w.WriteHeader(http.StatusOK)
	w.Write(tile)

	h.logger.Printf("Served tile z=%d x=%d y=%d (%d bytes)\n", zInt, xInt, yInt, len(tile))
}

// ConvertCADHandler handles POST /api/v1/cadastre/convert
// Converts uploaded CAD data to cadastral entities
func (h *CadastreHandlers) ConvertCADHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("ConvertCADHandler: %s\n", r.URL.Path)

	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB max
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("Form parsing failed: %v", err))
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Get region and admin unit from form
	regionCode := r.FormValue("region")
	adminUnit := r.FormValue("admin_unit")

	if regionCode == "" || adminUnit == "" {
		h.respondError(w, http.StatusBadRequest, "region and admin_unit required")
		return
	}

	// Read file data
	fileData := make([]byte, fileHeader.Size)
	if _, err := file.Read(fileData); err != nil {
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("File read failed: %v", err))
		return
	}

	// Determine CAD format from file extension
	format := detectCADFormat(fileHeader.Filename)

	// Create raw CAD data
	cadData := &service.RawCADData{
		Format:     format,
		SourcePath: fileHeader.Filename,
		VertexData: []service.CADVertex{}, // Would be populated from actual CAD parsing
		Metadata: map[string]interface{}{
			"original_filename": fileHeader.Filename,
			"uploaded_at":       "2026-05-11T10:00:00Z",
		},
	}

	// Convert and store
	entitySet, err := h.adapter.ConvertAndStoreCadastralData(r.Context(), cadData, regionCode, adminUnit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Conversion failed: %v", err))
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"entities_created": len(entitySet.Entities),
		"entities":      entitySet.Entities,
	})

	h.logger.Printf("Converted CAD file: %s → %d entities\n", fileHeader.Filename, len(entitySet.Entities))
}

// DecodeCadastreCodeHandler handles POST /api/v1/cadastre/decode
// Decodes a cadastre code and returns entity details
func (h *CadastreHandlers) DecodeCadastreCodeHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("DecodeCadastreCodeHandler: %s\n", r.URL.Path)

	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("JSON decode failed: %v", err))
		return
	}

	if req.Code == "" {
		h.respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	// In production: decode from database using code lookup
	// For now: placeholder response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": req.Code,
		"type": "parcel",
		"message": "Entity lookup not yet implemented",
	})

	h.logger.Printf("Decoded code: %s\n", req.Code)
}

// ValidateRoundTripHandler handles POST /api/v1/cadastre/validate
// Tests round-trip integrity for an entity
func (h *CadastreHandlers) ValidateRoundTripHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("ValidateRoundTripHandler: %s\n", r.URL.Path)

	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("JSON decode failed: %v", err))
		return
	}

	// In production: fetch entity by code, run validation
	// For now: placeholder
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": req.Code,
		"status": "pending",
		"message": "Round-trip validation not yet implemented",
	})

	h.logger.Printf("Validation check: %s\n", req.Code)
}

// GetLegendHandler handles GET /api/v1/cadastre/legend
// Returns SVG legend for tile interpretation
func (h *CadastreHandlers) GetLegendHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("GetLegendHandler: %s\n", r.URL.Path)

	renderer := svg_codification.NewSVGRenderer(h.logger)
	legend := renderer.RenderLegend()

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(legend)
}

// HealthHandler handles GET /api/v1/cadastre/health
// Returns service health status
func (h *CadastreHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"service": "cadastre",
		"timestamp": "2026-05-11T10:00:00Z",
	})
}

// Helper functions

// detectCADFormat determines CAD format from filename
func detectCADFormat(filename string) service.CADFormat {
	if strings.HasSuffix(strings.ToLower(filename), ".dwg") {
		return service.FormatDWG
	}
	if strings.HasSuffix(strings.ToLower(filename), ".dxf") {
		return service.FormatDXF
	}
	if strings.HasSuffix(strings.ToLower(filename), ".qgis") {
		return service.FormatQGIS
	}
	if strings.HasSuffix(strings.ToLower(filename), ".shp") {
		return service.FormatShapefile
	}
	return service.FormatDWG // Default
}

// respondError writes an error response
func (h *CadastreHandlers) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"status": statusCode,
	})
}

// RegisterRoutes registers all cadastre handlers with the HTTP mux
func RegisterRoutes(mux *http.ServeMux, handlers *CadastreHandlers) {
	mux.HandleFunc("/api/v1/cadastre/tiles/", handlers.GetTileHandler)
	mux.HandleFunc("/api/v1/cadastre/convert", handlers.ConvertCADHandler)
	mux.HandleFunc("/api/v1/cadastre/decode", handlers.DecodeCadastreCodeHandler)
	mux.HandleFunc("/api/v1/cadastre/validate", handlers.ValidateRoundTripHandler)
	mux.HandleFunc("/api/v1/cadastre/legend", handlers.GetLegendHandler)
	mux.HandleFunc("/api/v1/cadastre/health", handlers.HealthHandler)
}
