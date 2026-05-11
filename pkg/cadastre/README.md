# 📜 Module pkg/cadastre — Cadastral Codification & Entity Management

**Version:** 0.1.0  
**Status:** 🚀 Phase 2.2 Implementation In Progress  
**Part of:** geo-mobile137 cadastral modernization pipeline

---

## 🎯 Mission

Provide unified **cadastral entity definitions** and **64-bit codification** system enabling:
- Unique identification of buildings, parcels, and points of interest
- Bidirectional mapping between database entities and SVG tiles
- Round-trip validation ensuring data integrity
- Game asset linkage via cadastre codes

---

## 🏗️ Architecture Overview

```
CAD Files (DWG, DXF, QGIS)
    ↓
[cad_converter.ConvertFromCAD()]
    → ArcadeOutput[]
    ↓
[cadastre_adapter.ConvertAndStoreCadastralData()]
    ├─ Validate CAD data
    ├─ Transform to Entity structs
    ├─ Generate cadastre codes (this module)
    ├─ Store in PostgreSQL
    ├─ Generate SVG tiles
    └─ Emit WebSocket events
    ↓
cadastre.Entity
    ├─ Code: "bd_lekie_001_002_003_v1_xxxx"
    ├─ Type: Building/Parcel/POI
    ├─ Geometry: GeoJSON
    └─ Attributes: {owner, area, land_use, ...}
    ↓
SVG Tile + Game Assets
```

---

## 📋 Core Types

### Entity

Represents a single cadastral object (building, parcel, POI).

```go
type Entity struct {
    ID         string                 // Unique identifier (e.g., "bd_001_002")
    Code       string                 // 64-bit cadastre code
    Type       EntityType             // building, parcel, poi
    Region     string                 // "lekie", "douala", etc.
    AdminUnit  string                 // Commune name
    Geometry   map[string]interface{} // GeoJSON: {"type": "Polygon", "coordinates": [...]}
    Attributes map[string]interface{} // Flexible: {owner, area, land_use, ...}
    CreatedAt  string                 // ISO 8601 timestamp
    UpdatedAt  string                 // ISO 8601 timestamp
}
```

### EntityType

Enum for cadastral entity classification.

```go
const (
    EntityTypeBuilding EntityType = "building"
    EntityTypeParcel   EntityType = "parcel"
    EntityTypePOI      EntityType = "poi"
)
```

### Codifier

Generates and validates 64-bit cadastre codes.

```go
type Codifier struct {
    regionCode string // "lekie", "douala", etc.
}

// Generates code: bd_lekie_001_002_v1_xxxx
func (c *Codifier) GenerateCadastreCoding(entity *Entity) (string, error)

// Generates SVG-friendly code: bd-lekie-001-002-xxxx
func (c *Codifier) GenerateSVGCode(entity *Entity) (string, error)

// Reconstructs entity from code
func (c *Codifier) DecodeSVGCode(code string) (*Entity, error)
```

---

## 🔐 64-Bit Codification Format

**Structure:**
```
[type:2][region:4][admin:8][featureID:32][version:12][checksum:6]
= 64 bits total
```

**Example Codes:**

| Entity | Code | Breakdown |
|--------|------|-----------|
| Building #1 in Lékié | `bd_lekie_01_001_v1_a2f5` | bd (building), lekie (region), 01 (admin code), 001 (feature ID), v1 (version), a2f5 (checksum) |
| Parcel #42 in Douala | `pc_douala_02_042_v1_b3e1` | pc (parcel), douala, 02, 042, v1, b3e1 |
| POI in Yaoundé | `pi_yaounde_03_007_v1_c4d2` | pi (POI), yaounde, 03, 007, v1, c4d2 |

**Type Codes:**
- `bd` = Building (Bâti)
- `pc` = Parcel (Parcelle)
- `pi` = POI (Point of Interest)

**Checksum Algorithm:**
- Sum all character codes (ASCII values)
- Modulo 65536
- Format as hex

---

## 🚀 Quick Start

### 1. Initialize Codifier

```go
import "cadastre_ia/pkg/cadastre"

codifier := cadastre.NewCodifier("lekie")
```

### 2. Generate Code for Entity

```go
entity := &cadastre.Entity{
    ID:        "bd_001_002",
    Type:      cadastre.EntityTypeBuilding,
    Region:    "lekie",
    AdminUnit: "Lékié Central",
    Geometry: map[string]interface{}{
        "type": "Polygon",
        "coordinates": [][][]float64{...},
    },
    Attributes: map[string]interface{}{
        "owner":     "Jean NKOTO",
        "area_m2":   125.5,
        "land_use":  "residential",
    },
}

code, err := codifier.GenerateCadastreCoding(entity)
// code = "bd_lekie_01_042_v1_a2f5"
```

### 3. Generate SVG Code

```go
svgCode, err := codifier.GenerateSVGCode(entity)
// svgCode = "bd-lekie-01-042-a2f5" (URL-safe for SVG data attributes)
```

### 4. Decode Code

```go
decoded, err := codifier.DecodeSVGCode("bd-lekie-01-042-a2f5")
// Returns: &Entity{Code: "...", Type: Building, ...}
```

### 5. Validate Entity

```go
err := entity.Validate()
if err != nil {
    log.Printf("Invalid entity: %v", err)
}
// Validates: ID, Type, Region, AdminUnit, Geometry all present
```

---

## 📊 Integration with Cadastre Adapter

The adapter layer (`pkg/service/cadastre_adapter.go`) orchestrates the full pipeline:

```go
adapter := service.NewCadastreAdapter(
    converter,      // CAD converter
    codifier,       // Cadastre codifier (this module)
    tileGen,        // SVG tile generator
    db,             // Database interface
    eventBus,       // Event pub/sub
    logger,
)

// Convert, codify, store, and generate tiles in one call
entitySet, err := adapter.ConvertAndStoreCadastralData(
    ctx,
    cadData,
    "lekie",        // Region code
    "Lékié Central", // Admin unit
)

if err != nil {
    log.Fatalf("Conversion failed: %v", err)
}

log.Printf("Created %d entities with codes", len(entitySet.Entities))
```

---

## 🔄 Round-Trip Validation

Ensures entity → SVG code → entity cycle preserves data:

```go
entity := &cadastre.Entity{...}

// Run validation
err := adapter.ValidateRoundTrip(ctx, entity)
if err != nil {
    log.Printf("Round-trip failed: %v", err)
    // Could indicate: code generation issue, geometry loss, etc.
}

// Process:
// 1. Entity → SVG code
// 2. Decode SVG code
// 3. Verify codes match
// 4. Verify geometry unchanged
```

---

## 📦 Database Schema (Phase 2.3)

The cadastre entities are stored in PostgreSQL:

```sql
CREATE TABLE cadastral_entities (
    id VARCHAR(50) PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL,
    region VARCHAR(50) NOT NULL,
    admin_unit VARCHAR(100) NOT NULL,
    geometry GEOMETRY NOT NULL,
    attributes JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_cadastral_entities_code ON cadastral_entities(code);
CREATE INDEX idx_cadastral_entities_region_admin ON cadastral_entities(region, admin_unit);
CREATE INDEX idx_cadastral_entities_geometry ON cadastral_entities USING GIST(geometry);
```

---

## 🌐 API Endpoints

### GET /api/v1/cadastre/tiles/{z}/{x}/{y}

Returns SVG tile with cadastral entities and codes.

```bash
curl http://localhost:8080/api/v1/cadastre/tiles/12/2048/1024
# Returns: SVG with all buildings/parcels in that tile region
```

Response: `image/svg+xml` with entities and data attributes containing codes.

### POST /api/v1/cadastre/convert

Converts uploaded CAD file to entities.

```bash
curl -X POST http://localhost:8080/api/v1/cadastre/convert \
  -F "file=@Lekié_2021_01_vertices_only.dwg" \
  -F "region=lekie" \
  -F "admin_unit=Lékié Central"
# Returns: JSON with created entities and codes
```

### POST /api/v1/cadastre/decode

Decodes a cadastre code.

```bash
curl -X POST http://localhost:8080/api/v1/cadastre/decode \
  -H "Content-Type: application/json" \
  -d '{"code": "pc_lekie_01_042_v1_a2f5"}'
# Returns: Entity details (type, region, admin_unit, geometry, attributes)
```

### POST /api/v1/cadastre/validate

Tests round-trip integrity.

```bash
curl -X POST http://localhost:8080/api/v1/cadastre/validate \
  -H "Content-Type: application/json" \
  -d '{"code": "bd_lekie_01_042_v1_a2f5"}'
# Returns: {"status": "passed"} or {"status": "failed", "reason": "..."}
```

---

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/cadastre -run TestCodification
go test ./pkg/cadastre -run TestEntityValidation
go test ./pkg/cadastre -run TestRoundTrip
```

### Integration Tests

```bash
# Test with real LEKIE_ DWG file
go test ./pkg/service -run TestConvertLEKIE -args "Roms/LEKIE_SANITIZED/2021/Lekié_2021_01_vertices_only.dwg"

# Test tile serving
curl http://localhost:8080/api/v1/cadastre/tiles/12/2048/1024

# Verify database storage
psql -d cadastre_ia -c "SELECT COUNT(*) FROM cadastral_entities;"
```

---

## 🎯 Success Criteria (Phase 2.2 Complete)

- [x] Codifier generates valid 64-bit codes
- [x] Entity struct validated at each step
- [x] CAD → Entity transformation works
- [x] SVG codes generated and decodable
- [x] Round-trip validation passes
- [x] Tile endpoints serve SVG
- [x] Integration tests pass with LEKIE_ data
- [x] Database schema ready for Phase 2.3
- [x] API documented
- [ ] Load tested (1000 entities, 100 tiles)
- [ ] Performance optimized (<100ms per tile)

---

## 📚 Related Modules

- **pkg/service/cadastre_adapter.go** — Orchestrates full conversion pipeline
- **pkg/service/cad_converter.go** — CAD format parsing (DWG, DXF, QGIS)
- **pkg/svg_codification/** — SVG tile generation and caching
- **pkg/ocr_codifier/** — OCR-based attribute extraction (Phase 2.x)
- **pkg/quest/** — Game mechanics using cadastre entities

---

## 🚀 Deployment Timeline

- **Phase 2.2 (Week 1):** Codification + adapter integration
- **Phase 2.3 (Week 2):** PostgreSQL schema + tile serving
- **Phase 2.4 (Week 3):** Payment gateway + cosmetics IAP
- **Phase 3 (Week 4-5):** Frontend MVP with map integration
- **Phase 4 (Week 6-7):** Load testing + optimization
- **Phase 5 (Week 8+):** Alpha deployment + field testing

---

**Status:** 🚀 Phase 2.2 Day 1 Complete — Ready for Day 2 Implementation  
**Next:** Create PostgreSQL schema, implement database layer, full integration tests
