// Package ocr_codifier provides OCR extraction and automatic attribute codification
// from CAD files, scans, and field documents to structured cadastral data.
//
// Workflow:
//   1. Extract text from image/PDF/CAD (Tesseract OCR)
//   2. Parse text to identify attributes (owner, parcel ID, area, etc.)
//   3. Link text to geometry (nearest polygon, centroid matching)
//   4. Codify attributes (normalize values, validate constraints)
//   5. Store in database with confidence score
//   6. Admin approval workflow (human validation required)

package ocr_codifier

import (
	"time"
)

// ==================== INPUT SOURCES ====================

// DocumentType represents source document category
type DocumentType string

const (
	DocumentTypeCADFile    DocumentType = "cad_file"        // DWG/DXF file
	DocumentTypeScan       DocumentType = "scan"            // Scanned paper document
	DocumentTypePhoto      DocumentType = "photo"           // Mobile photo of document/sign
	DocumentTypePDF        DocumentType = "pdf"             // PDF document
	DocumentTypeFieldForm  DocumentType = "field_form"      // Mobile app field survey
)

// OCRSource represents a document being processed
type OCRSource struct {
	ID              string        `json:"id"`
	Type            DocumentType  `json:"type"`
	SourcePath      string        `json:"source_path"`       // File path or URL
	UploadedAt      time.Time     `json:"uploaded_at"`
	UploadedBy      string        `json:"uploaded_by"`       // User ID
	RegionCode      string        `json:"region_code"`       // "lekie", "douala", etc.
	AdminUnit       string        `json:"admin_unit"`        // Commune name
	DocumentDate    *time.Time    `json:"document_date,omitempty"` // Date on document
	Status          string        `json:"status"`            // "pending", "processing", "complete", "error"
	ErrorMessage    string        `json:"error_message,omitempty"`
	ProcessedAt     *time.Time    `json:"processed_at,omitempty"`
}

// ==================== OCR EXTRACTION ====================

// ExtractedText represents raw OCR output
type ExtractedText struct {
	ID            string            `json:"id"`
	SourceID      string            `json:"source_id"`    // Link to OCRSource
	RawText       string            `json:"raw_text"`     // Full OCR output
	Confidence    float64           `json:"confidence"`   // 0-100% (Tesseract score)
	Lines         []TextLine        `json:"lines"`        // Per-line breakdown
	DetectedLang  string            `json:"detected_lang"` // "fr", "en", "multi"
	ExtractedAt   time.Time         `json:"extracted_at"`
}

// TextLine represents a single line of extracted text
type TextLine struct {
	LineNum   int     `json:"line_num"`
	Text      string  `json:"text"`
	BBox      BBox    `json:"bbox"`          // Bounding box on original document
	Confidence float64 `json:"confidence"`   // Per-line confidence
}

// BBox represents bounding box in image coordinates
type BBox struct {
	X0 float64 `json:"x0"`  // Top-left X
	Y0 float64 `json:"y0"`  // Top-left Y
	X1 float64 `json:"x1"`  // Bottom-right X
	Y1 float64 `json:"y1"`  // Bottom-right Y
}

// ==================== PARSING & ATTRIBUTE EXTRACTION ====================

// ParsedAttributes represents extracted cadastral attributes from text
type ParsedAttributes struct {
	ID              string                    `json:"id"`
	ExtractedTextID string                    `json:"extracted_text_id"`
	Attributes      map[string]AttributeValue `json:"attributes"`
	ParsingRules    []string                  `json:"parsing_rules"`   // Which regex patterns matched
	Confidence      float64                   `json:"confidence"`      // Average confidence
	MatchedLines    []int                     `json:"matched_lines"`   // Line numbers that contributed
	ParsedAt        time.Time                 `json:"parsed_at"`
}

// AttributeValue represents a single extracted attribute
type AttributeValue struct {
	Key        string      `json:"key"`       // "owner_name", "parcel_id", "area_m2", etc.
	Value      interface{} `json:"value"`     // Actual value (string, float, etc.)
	RawText    string      `json:"raw_text"`  // Original text matched
	Confidence float64     `json:"confidence"` // 0-100%
	SourceLine int         `json:"source_line"` // Which text line
	ValidationStatus string `json:"validation_status"` // "pending", "valid", "invalid"
}

// ==================== GEOMETRY LINKING ====================

// GeometryMatch represents linking extracted text to cadastral geometry
type GeometryMatch struct {
	ID              string    `json:"id"`
	ParsedAttrID    string    `json:"parsed_attr_id"`
	GeometryID      string    `json:"geometry_id"`        // Existing parcel/building ID
	GeometryType    string    `json:"geometry_type"`      // "parcel", "building", "poi"
	MatchMethod     string    `json:"match_method"`       // "centroid", "bbox_overlap", "manual"
	MatchScore      float64   `json:"match_score"`        // 0-100% (how confident is the match)
	BBoxDistance    float64   `json:"bbox_distance"`      // Distance from text bbox to geometry
	HumanValidated  bool      `json:"human_validated"`    // Requires admin sign-off
	ValidatedBy     string    `json:"validated_by,omitempty"` // Admin user ID
	ValidatedAt     *time.Time `json:"validated_at,omitempty"`
}

// ==================== CODIFICATION & NORMALIZATION ====================

// CodifiedAttribute represents final, standardized attribute ready for DB
type CodifiedAttribute struct {
	ID              string                 `json:"id"`
	GeometryMatchID string                 `json:"geometry_match_id"`
	CadastrCode     string                 `json:"cadastre_code"`         // From codification module
	Attributes      map[string]interface{} `json:"attributes"`            // Normalized values
	SourceParseID   string                 `json:"source_parse_id"`       // Audit trail
	Normalization   map[string]NormRule    `json:"normalization_applied"` // Which rules were applied
	DataQuality     float64                `json:"data_quality"`          // 0-100%
	AdminApproved   bool                   `json:"admin_approved"`
	ApprovedBy      string                 `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time             `json:"approved_at,omitempty"`
	CodifiedAt      time.Time              `json:"codified_at"`
}

// NormRule represents a normalization rule that was applied
type NormRule struct {
	RuleID    string      `json:"rule_id"`
	RuleName  string      `json:"rule_name"`      // "trim_whitespace", "validate_owner_id", etc.
	InputValue interface{} `json:"input_value"`
	OutputValue interface{} `json:"output_value"`
	Applied   bool        `json:"applied"`
	Error     string      `json:"error,omitempty"`
}

// ==================== VALIDATION RULES ====================

// ValidationRule defines constraints for attribute values
type ValidationRule struct {
	AttributeKey  string      `json:"attribute_key"`
	DataType      string      `json:"data_type"`          // "string", "float", "int", "date"
	Required      bool        `json:"required"`
	MinLength     *int        `json:"min_length,omitempty"`
	MaxLength     *int        `json:"max_length,omitempty"`
	Regex         string      `json:"regex,omitempty"`     // Validation pattern
	AllowedValues []string    `json:"allowed_values,omitempty"` // Enum
	MinValue      *float64    `json:"min_value,omitempty"`
	MaxValue      *float64    `json:"max_value,omitempty"`
	ErrorMessage  string      `json:"error_message"`
}

// ==================== ADMIN WORKFLOW ====================

// ApprovalTask represents a manual approval task for admin
type ApprovalTask struct {
	ID                string       `json:"id"`
	Type              string       `json:"type"`            // "geometry_match", "attribute_validation", "data_quality_check"
	CodifiedAttrID    string       `json:"codified_attr_id"`
	Status            string       `json:"status"`          // "pending", "approved", "rejected"
	CreatedAt         time.Time    `json:"created_at"`
	AssignedTo        string       `json:"assigned_to"`     // Admin user ID
	Priority          string       `json:"priority"`        // "low", "medium", "high"
	ReasonForReview   string       `json:"reason_for_review"` // Why human review needed
	AdminNotes        string       `json:"admin_notes,omitempty"`
	ApprovedAt        *time.Time   `json:"approved_at,omitempty"`
	RejectionReason   string       `json:"rejection_reason,omitempty"`
	SuggestionForUser string       `json:"suggestion_for_user,omitempty"`
}

// BulkImportJob represents batch processing of multiple documents
type BulkImportJob struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`              // "LEKIE_2025_batch_001"
	DocumentCount  int       `json:"document_count"`    // Total documents in batch
	ProcessedCount  int       `json:"processed_count"`
	SuccessCount    int       `json:"success_count"`
	ErrorCount      int       `json:"error_count"`
	CreatedAt       time.Time `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	Status          string    `json:"status"`           // "pending", "processing", "complete", "error"
	ProgressPercent float64   `json:"progress_percent"`
}

// ==================== AUDIT & ANALYTICS ====================

// ProcessingStats tracks OCR module performance
type ProcessingStats struct {
	TimestampDate     time.Time `json:"timestamp_date"`
	DocumentsProcessed int      `json:"documents_processed"`
	AvgOCRConfidence  float64   `json:"avg_ocr_confidence"`
	AttributesExtracted int     `json:"attributes_extracted"`
	MatchSuccess      float64   `json:"match_success"`        // % of texts matched to geometry
	AdminApprovalRate float64   `json:"admin_approval_rate"`  // % approved after review
	AvgTimePerDocument float64   `json:"avg_time_per_document"` // Seconds
	ErrorRate         float64   `json:"error_rate"`           // % documents with errors
}

// AuditLog tracks all changes for compliance
type AuditLog struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`      // "extracted", "parsed", "matched", "codified", "approved"
	ResourceID  string    `json:"resource_id"` // Document/attribute ID
	OldValue    string    `json:"old_value,omitempty"`
	NewValue    string    `json:"new_value,omitempty"`
	Reason      string    `json:"reason,omitempty"` // Why the change
}
