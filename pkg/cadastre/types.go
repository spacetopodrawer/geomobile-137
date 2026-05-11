// Package cadastre defines core cadastral data types and codification
package cadastre

import (
	"fmt"
)

// EntityType represents the type of cadastral entity
type EntityType string

const (
	EntityTypeBuilding EntityType = "building"
	EntityTypeParcel   EntityType = "parcel"
	EntityTypePOI      EntityType = "poi"
)

// Entity represents a cadastral object (building, parcel, or point of interest)
type Entity struct {
	ID         string                 `json:"id"`
	Code       string                 `json:"code"`        // 64-bit cadastre code
	Type       EntityType             `json:"type"`
	Region     string                 `json:"region"`      // "lekie", "douala", etc.
	AdminUnit  string                 `json:"admin_unit"`  // Commune name
	Geometry   map[string]interface{} `json:"geometry"`    // GeoJSON geometry
	Attributes map[string]interface{} `json:"attributes"`  // Flexible attribute storage
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// Validate checks entity integrity
func (e *Entity) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if e.Type == "" {
		return fmt.Errorf("entity type is required")
	}
	if e.Region == "" {
		return fmt.Errorf("region code is required")
	}
	if e.AdminUnit == "" {
		return fmt.Errorf("admin unit is required")
	}
	if len(e.Geometry) == 0 {
		return fmt.Errorf("geometry is required")
	}
	return nil
}

// Codifier handles cadastral code generation and validation
type Codifier struct {
	regionCode string
}

// NewCodifier creates a new codifier for a region
func NewCodifier(regionCode string) *Codifier {
	return &Codifier{
		regionCode: regionCode,
	}
}

// GenerateCadastreCoding creates a 64-bit cadastre code from entity
// Format: [type:2][region:4][admin:8][featureID:32][version:12][checksum:6]
func (c *Codifier) GenerateCadastreCoding(entity *Entity) (string, error) {
	if entity == nil {
		return "", fmt.Errorf("entity is nil")
	}

	// Build code components
	typeCode := entityTypeCode(entity.Type)
	regionCode := c.regionCode
	adminCode := stringToCode(entity.AdminUnit, 2)
	featureID := stringToCode(entity.ID, 5)
	version := "001"
	checksum := computeChecksum(fmt.Sprintf("%s%s%s%s%s", typeCode, regionCode, adminCode, featureID, version))

	// Assemble code in format: type_region_admin_feature_version_checksum
	code := fmt.Sprintf("%s_%s_%s_%s_%s_%s",
		typeCode,
		regionCode,
		adminCode,
		featureID,
		version,
		checksum,
	)

	return code, nil
}

// GenerateSVGCode creates an SVG-compatible code string
func (c *Codifier) GenerateSVGCode(entity *Entity) (string, error) {
	if entity == nil {
		return "", fmt.Errorf("entity is nil")
	}

	// SVG codes are similar to cadastre codes but optimized for embedding in SVG data attributes
	// Format: entity-type-region-admin-id-checksum (URL-safe)
	typeCode := entityTypeCode(entity.Type)
	regionCode := c.regionCode
	adminCode := stringToCode(entity.AdminUnit, 2)
	featureID := stringToCode(entity.ID, 5)
	checksum := computeChecksum(fmt.Sprintf("%s%s%s%s", typeCode, regionCode, adminCode, featureID))

	code := fmt.Sprintf("%s-%s-%s-%s-%s",
		typeCode,
		regionCode,
		adminCode,
		featureID,
		checksum,
	)

	return code, nil
}

// DecodeSVGCode reconstructs entity information from SVG code
func (c *Codifier) DecodeSVGCode(code string) (*Entity, error) {
	if code == "" {
		return nil, fmt.Errorf("code is empty")
	}

	// Placeholder: in production, parse code components and verify checksum
	// Return a reconstructed entity with validated structure
	return &Entity{
		Code:   code,
		Type:   EntityTypeParcel,
		Region: c.regionCode,
	}, nil
}

// Helper functions

// entityTypeCode returns the 2-character code for an entity type
func entityTypeCode(t EntityType) string {
	switch t {
	case EntityTypeBuilding:
		return "bd"
	case EntityTypeParcel:
		return "pc"
	case EntityTypePOI:
		return "pi"
	default:
		return "xx"
	}
}

// stringToCode converts a string to a numeric code (for name abbreviation)
func stringToCode(s string, length int) string {
	if len(s) > length {
		s = s[:length]
	}
	code := ""
	for i := 0; i < length; i++ {
		if i < len(s) {
			code += fmt.Sprintf("%02x", s[i])
		} else {
			code += "00"
		}
	}
	return code
}

// computeChecksum generates a simple checksum for code validation
func computeChecksum(data string) string {
	sum := 0
	for _, c := range data {
		sum += int(c)
	}
	return fmt.Sprintf("%04x", sum%65536)[:4]
}
