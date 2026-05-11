// Package svg_codification handles SVG rendering for cadastral tiles
package svg_codification

import (
	"bytes"
	"context"
	"fmt"
	"log"
)

// SVGRenderer converts cadastral entities to SVG tiles
type SVGRenderer struct {
	logger *log.Logger
}

// NewSVGRenderer creates a new SVG renderer instance
func NewSVGRenderer(logger *log.Logger) *SVGRenderer {
	if logger == nil {
		logger = log.New(log.Writer(), "[SVGRenderer] ", log.LstdFlags)
	}
	return &SVGRenderer{
		logger: logger,
	}
}

// RenderTile generates an SVG tile for the given coordinates and entities
func (sr *SVGRenderer) RenderTile(ctx context.Context, z, x, y int, entities []interface{}) ([]byte, error) {
	sr.logger.Printf("Rendering SVG tile z=%d x=%d y=%d with %d entities\n", z, x, y, len(entities))

	// Initialize SVG buffer
	var buf bytes.Buffer

	// SVG header with viewport based on tile coordinates
	sr.writeSVGHeader(&buf, z, x, y)

	// Render each entity based on its type and detail level
	for i, entity := range entities {
		sr.renderEntity(&buf, z, x, y, entity, i)
	}

	// SVG footer
	sr.writeSVGFooter(&buf)

	return buf.Bytes(), nil
}

// writeSVGHeader writes the SVG XML header with proper viewport settings
func (sr *SVGRenderer) writeSVGHeader(buf *bytes.Buffer, z, x, y int) {
	width := 256.0  // Standard tile size
	height := 256.0

	// SVG declaration
	fmt.Fprintf(buf, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="256" height="256" viewBox="0 0 256 256" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <style>
      .parcel { fill: #e8f4f8; stroke: #2c5aa0; stroke-width: 1; opacity: 0.8; }
      .building { fill: #d4a574; stroke: #8b6a47; stroke-width: 1.5; opacity: 0.9; }
      .poi { fill: #ff6b6b; stroke: #c92a2a; stroke-width: 2; opacity: 0.95; }
      .label { font-size: 8px; font-family: sans-serif; fill: #333; }
      .cadastre-code { font-size: 6px; font-family: monospace; fill: #666; opacity: 0.7; }
    </style>
  </defs>
  <!-- Tile background -->
  <rect width="256" height="256" fill="#f0f0f0"/>
  <!-- Grid reference (optional, debug mode) -->
  <g class="grid" stroke="#ddd" stroke-width="0.5" opacity="0.3">
    <line x1="0" y1="128" x2="256" y2="128"/>
    <line x1="128" y1="0" x2="128" y2="256"/>
  </g>
  <!-- Entities -->
`)
}

// renderEntity renders a single cadastral entity in the SVG
func (sr *SVGRenderer) renderEntity(buf *bytes.Buffer, z, x, y int, entity interface{}, index int) {
	// Extract entity data (type-agnostic handling)
	// In production: proper type assertion with Entity struct

	// Render placeholder geometry
	// For z0-5: simplified commune outlines
	// For z6-10: parcel polygons
	// For z11-18: detailed buildings with addresses

	switch z {
	case 0, 1, 2, 3, 4, 5:
		// Commune level: larger polygons
		sr.renderCommunePolygon(buf, index)
	case 6, 7, 8, 9, 10:
		// Parcel level: mid-size polygons
		sr.renderParcelPolygon(buf, index)
	default:
		// Building level: detailed geometry
		sr.renderBuildingGeometry(buf, index)
	}
}

// renderCommunePolygon renders a simplified polygon for commune-level detail
func (sr *SVGRenderer) renderCommunePolygon(buf *bytes.Buffer, index int) {
	// Sample polygon covering ~30% of tile
	baseX := 50.0 + float64(index%3)*60
	baseY := 50.0 + float64(index%3)*60

	fmt.Fprintf(buf, `  <polygon class="parcel" points="%f,%f %f,%f %f,%f %f,%f" data-cadastre-id="comm_%d"/>
`, baseX, baseY, baseX+50, baseY, baseX+50, baseY+50, baseX, baseY+50, index)

	fmt.Fprintf(buf, `  <text class="label" x="%f" y="%f">Commune %d</text>
`, baseX+10, baseY+30, index)
}

// renderParcelPolygon renders a parcel polygon with cadastre code annotation
func (sr *SVGRenderer) renderParcelPolygon(buf *bytes.Buffer, index int) {
	baseX := 30.0 + float64(index%5)*40
	baseY := 30.0 + float64(index/5)*40

	code := fmt.Sprintf("PC_L_%03d_%03d", index/10, index%10)

	fmt.Fprintf(buf, `  <polygon class="parcel" points="%f,%f %f,%f %f,%f %f,%f" data-cadastre-code="%s"/>
`, baseX, baseY, baseX+35, baseY, baseX+35, baseY+35, baseX, baseY+35, code)

	fmt.Fprintf(buf, `  <text class="cadastre-code" x="%f" y="%f">%s</text>
`, baseX+2, baseY+15, code)
}

// renderBuildingGeometry renders detailed building outlines with addresses
func (sr *SVGRenderer) renderBuildingGeometry(buf *bytes.Buffer, index int) {
	baseX := 20.0 + float64(index%8)*30
	baseY := 20.0 + float64(index/8)*30

	code := fmt.Sprintf("BD_L_%04d_%04d", index/50, index%50)

	fmt.Fprintf(buf, `  <rect class="building" x="%f" y="%f" width="25" height="25" data-cadastre-code="%s"/>
`, baseX, baseY, code)

	fmt.Fprintf(buf, `  <text class="cadastre-code" x="%f" y="%f">%s</text>
`, baseX+1, baseY+10, code)
}

// writeSVGFooter writes the closing SVG tags
func (sr *SVGRenderer) writeSVGFooter(buf *bytes.Buffer) {
	fmt.Fprintf(buf, `  <!-- Metadata -->
  <metadata>
    <timestamp>2026-05-11T10:00:00Z</timestamp>
    <projection>Web Mercator</projection>
    <license>CC-BY-SA 4.0</license>
  </metadata>
</svg>
`)
}

// RenderStaticMap generates a full-page SVG map for a region (not tiled)
func (sr *SVGRenderer) RenderStaticMap(ctx context.Context, regionCode string, adminUnit string, entities []interface{}) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, `<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <style>
      .title { font-size: 24px; font-weight: bold; fill: #000; }
      .region-name { font-size: 16px; fill: #333; }
    </style>
  </defs>
  <rect width="1024" height="1024" fill="#f5f5f5"/>
  <text class="title" x="50" y="50">Cadastral Map</text>
  <text class="region-name" x="50" y="80">%s / %s</text>
  <!-- Entities will be rendered here -->
</svg>
`, regionCode, adminUnit)

	return buf.Bytes(), nil
}

// RenderLegend generates an SVG legend for tile interpretation
func (sr *SVGRenderer) RenderLegend() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, `<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 300 400" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <style>
      .legend-title { font-size: 16px; font-weight: bold; fill: #000; }
      .legend-item { font-size: 12px; fill: #333; }
      .legend-color { stroke-width: 1; }
    </style>
  </defs>
  <rect width="300" height="400" fill="white" stroke="#ccc" stroke-width="1"/>

  <!-- Title -->
  <text class="legend-title" x="20" y="30">Cadastral Features</text>

  <!-- Parcel legend -->
  <rect class="legend-color parcel" x="20" y="50" width="20" height="20"/>
  <text class="legend-item" x="50" y="65">Parcel (Parcelle)</text>

  <!-- Building legend -->
  <rect class="legend-color building" x="20" y="85" width="20" height="20"/>
  <text class="legend-item" x="50" y="100">Building (Bâti)</text>

  <!-- POI legend -->
  <circle class="legend-color poi" cx="30" cy="140" r="10"/>
  <text class="legend-item" x="50" y="145">Point of Interest</text>

  <!-- Color meanings -->
  <text class="legend-item" x="20" y="190" style="font-weight: bold;">Color Meanings:</text>
  <text class="legend-item" x="20" y="210">• Light blue = public parcel</text>
  <text class="legend-item" x="20" y="230">• Brown = residential building</text>
  <text class="legend-item" x="20" y="250">• Red = administrative facility</text>

  <!-- Zoom indicators -->
  <text class="legend-item" x="20" y="290" style="font-weight: bold;">Zoom Levels:</text>
  <text class="legend-item" x="20" y="310">z0-5: Communes</text>
  <text class="legend-item" x="20" y="330">z6-10: Parcels</text>
  <text class="legend-item" x="20" y="350">z11-18: Buildings</text>
</svg>
`)

	return buf.Bytes()
}
