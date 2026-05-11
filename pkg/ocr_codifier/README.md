# 🔍 Module pkg/ocr_codifier — OCR + Auto-Codification System

**Version:** 0.1.0 (Design + Types)  
**Status:** 📐 Architecture designed, implementation pending  
**Part of:** geo-mobile137 cadastral modernization pipeline

---

## 🎯 Mission

Transform **unstructured data** (scans, photos, CAD text) into **structured cadastral attributes** with:
- Automatic text extraction (Tesseract OCR)
- Intelligent parsing (regex + ML patterns)
- Geometry linking (match text to polygons)
- Attribute normalization (validate & standardize)
- Admin approval workflow (human verification)
- Confidence scoring (track data quality)

---

## 🔄 Processing Pipeline

```
Input Document
    ↓
[Step 1: OCR Extraction]
  └─ Tesseract OCR → raw text + confidence scores
    ↓
[Step 2: Text Parsing]
  └─ Extract attributes (owner, parcel_id, area, land_use, etc.)
    ↓
[Step 3: Geometry Linking]
  └─ Match text location to cadastral geometry (parcel/building)
    ↓
[Step 4: Codification]
  └─ Normalize attributes (fix formats, validate constraints)
    ↓
[Step 5: Admin Approval]
  └─ Human review + sign-off (confidence threshold check)
    ↓
[Output: Codified Attributes]
  └─ Ready for database storage
```

---

## 📋 Typical Workflow

### Scenario: Scanning Old Cadastral Plan

```
User uploads: "MAPPE REGION FUSION 2021_scan.pdf"
    ↓
OCR extracts text:
  "Parcel: DEP_LEKIE_001_002_003"
  "Owner: Jean NKOTO"
  "Area: 0.45 ha"
  "Land use: agriculture"
    ↓
Parser identifies attributes:
  {
    "parcel_id": "DEP_LEKIE_001_002_003" (confidence: 95%)
    "owner_name": "Jean NKOTO" (confidence: 88%)
    "area_hectares": 0.45 (confidence: 92%)
    "land_use": "agriculture" (confidence: 90%)
  }
    ↓
Geometry linker finds matching parcel:
  "Match score: 98% (centroid within 2m of text)"
    ↓
Codifier normalizes:
  "area_hectares": 0.45 → 4500 m² (for consistency)
  "land_use": "agriculture" → "LAND_USE_AGRICULTURE" (enum)
    ↓
Admin reviews:
  ✅ "Looks good, approve"
    ↓
Attributes codified & ready for DB
```

---

## 🔐 Sensitive Data Handling

**CRITICAL:** OCR module works on **SANITIZED CAD files ONLY**

### Input Constraints:
- ❌ **DO NOT** process original DWG with owner annotations
- ✅ **DO** process sanitized DWG (vertices + OCR-extracted geometry)
- ✅ **DO** process public documents (scans of filed plans)
- ✅ **DO** process field survey forms (mobile app)

### Output Constraints:
- ❌ **DO NOT** store raw owner names in public DB
- ✅ **DO** store codified owner IDs (linked to secure reference)
- ✅ **DO** audit log all extractions (for compliance)
- ✅ **DO** require admin approval before final storage

---

## 🧠 Attribute Extraction Examples

### Example 1: Parcel ID Detection

**Input text (from OCR):**
```
"Parcelle : DEP_LEKIE_001_002_003"
```

**Parsing rule:**
```regex
Parcelle\s*:\s*([A-Z_0-9]+)
```

**Output:**
```json
{
  "key": "parcel_id",
  "value": "DEP_LEKIE_001_002_003",
  "confidence": 95,
  "raw_text": "Parcelle : DEP_LEKIE_001_002_003"
}
```

### Example 2: Area Extraction (Multiple Formats)

**Input text variations:**
```
"Superficie : 4500 m²"
"Area: 0.45 hectares"
"Área: 1.11 acres"
```

**Parsing rules (multi-format):**
```regex
(superficie|area|área)\s*:\s*([\d.]+)\s*(m²|hectares?|acres?)
```

**Output (normalized to m²):**
```json
{
  "key": "area_m2",
  "value": 4500,
  "confidence": 92,
  "unit_converted_from": "hectares",
  "raw_text": "Area: 0.45 hectares"
}
```

### Example 3: Owner Name with French Characters

**Input text (OCR may have errors):**
```
"Proprietaire : J€an NKOTO"  // € wrongly recognized as 'e'
```

**Fuzzy matching + correction:**
```
Original OCR: "J€an NKOTO" (confidence: 78%)
Corrected:    "Jean NKOTO" (confidence: 92% after spell check)
```

**Output:**
```json
{
  "key": "owner_name",
  "value": "Jean NKOTO",
  "confidence": 85,  // Average of OCR + correction
  "ocr_confidence": 78,
  "correction_applied": "spell_check",
  "raw_text": "Proprietaire : J€an NKOTO"
}
```

---

## 🔗 Geometry Linking Algorithms

### Method 1: Centroid Matching (Default)

```
For each extracted text (with bounding box):
  1. Find all parcels within 50m radius
  2. Calculate distance from text centroid to parcel centroid
  3. If distance < 10m → strong match (score: 95%+)
  4. If distance < 50m → weak match (score: 60-80%)
  5. If distance > 50m → no match
  
Result: Single best match or "ambiguous" if tie
```

### Method 2: Bounding Box Overlap

```
For each text bbox:
  1. Find parcels that bbox overlaps
  2. Calculate intersection area / text bbox area
  3. If overlap > 70% → strong match (95%+)
  4. If overlap > 30% → weak match (60-80%)
  5. If overlap < 30% → no match
```

### Method 3: Manual Validation (Fallback)

If algorithms can't decide (ambiguous matches):
- Flag for admin review
- Show multiple candidate matches
- Admin selects correct one
- System learns from correction

---

## ✅ Validation Rules (Built-in Constraints)

### Parcel ID Format

```yaml
attribute_key: parcel_id
data_type: string
regex: ^[A-Z_]{2,10}_[A-Z_]{2,10}_\d{3}_\d{3}(_\d{3})?$
# Matches: DEP_LEKIE_001_002_003, REG_DOUALA_045_678
error_message: "Parcel ID must match cadastral format"
```

### Area (m²)

```yaml
attribute_key: area_m2
data_type: float
min_value: 1.0          # Minimum 1 m²
max_value: 1000000.0    # Maximum 1 km²
error_message: "Area must be between 1 m² and 1 million m²"
```

### Owner Type

```yaml
attribute_key: owner_type
data_type: string
allowed_values:
  - "private"
  - "public"
  - "cooperative"
  - "government"
error_message: "Owner type must be one of: private, public, cooperative, government"
```

---

## 📊 Quality Metrics

### Confidence Scoring (0-100%)

```
Codified attribute quality =
  (OCR_confidence + parsing_confidence + geometry_match_score) / 3
  
Example:
  OCR: 90% (clear text)
  Parsing: 95% (regex matched perfectly)
  Geometry: 98% (text centroid 1m from parcel centroid)
  ────────────────
  Final: 94% (HIGH QUALITY)
```

### Approval Thresholds

```
Quality > 90%  → Auto-approve (no human review needed)
Quality 70-90% → Admin review (1-2 min per attribute)
Quality < 70%  → Manual entry (too uncertain, skip OCR, user types)
```

---

## 🔄 Bidirectional Flow (Future)

### Forward (Terrain → Documents → Database)

```
Field surveyors collect RTK data + photos
    ↓
OCR extracts visible text from photos
    ↓
Auto-codify attributes
    ↓
Link to RTK geometry
    ↓
Admin approves
    ↓
Database updated (auto-generate cadastral plans)
```

### Reverse (Database → Documents)

```
Admin edits parcel in database (change owner, area, land_use)
    ↓
System auto-generates new cadastral document
    ↓
SVG plan updated
    ↓
PDF extracted (for filing with authorities)
```

---

## 🛠️ Dependencies

- **Tesseract OCR:** Text extraction (open-source)
- **GDAL:** CAD/shapefile handling
- **PostgreSQL:** Attribute storage
- **Python NLP:** Advanced pattern matching (spaCy, future)

---

## 📈 Expected Performance (P1)

| Metric | Target | Notes |
|--------|--------|-------|
| OCR accuracy | 85-95% | Depends on document quality |
| Auto-match rate | 60-80% | Geometry linking success |
| Admin approval rate | 90%+ | After human review |
| Processing time | 5-10 sec/doc | Single-threaded |
| Throughput | 300-500 docs/day | Single server |

---

## 🚀 Deployment Timeline

- **P1 (M0-3):** Proof-of-concept (1 commune, manual validation)
- **P2 (M3-6):** Beta (3-5 communes, mixed auto/manual)
- **P3 (M6-12):** Production (CEMAC-wide, 90%+ automation)
- **P4 (M12+):** Scale (Africa-wide, ML improvements)

---

**Status:** 📐 Architecture complete, implementation underway  
**Next:** Implement service.go (OCR orchestration) + handlers.go (admin UI)

