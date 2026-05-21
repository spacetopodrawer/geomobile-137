package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// DocumentScan represents a scanned document (property deed, ID, etc.)
type DocumentScan struct {
	ScanID            uuid.UUID          `json:"scan_id" db:"scan_id"`
	UserID            *uuid.UUID         `json:"user_id" db:"user_id"`
	DocumentType      string             `json:"document_type" db:"document_type"` // 'cadastral_deed', 'id_card', 'passport', 'property_photo'
	RawImagePath      string             `json:"raw_image_path" db:"raw_image_path"`
	ScannedAt         time.Time          `json:"scanned_at" db:"scanned_at"`
	ExtractionStatus  string             `json:"extraction_status" db:"extraction_status"` // 'pending', 'processing', 'success', 'partial', 'failed'
	OCRConfidence     float64            `json:"ocr_confidence" db:"ocr_confidence"`       // 0-1
	ExtractedText     string             `json:"extracted_text" db:"extracted_text"`
	ExtractedFields   DocumentFields     `json:"extracted_fields" db:"extracted_fields"`
	MatchConfidence   float64            `json:"match_confidence" db:"match_confidence"`   // 0-1
	MatchedUserID     *uuid.UUID         `json:"matched_user_id" db:"matched_user_id"`
	MatchMethod       string             `json:"match_method" db:"match_method"` // 'exact_email', 'fuzzy_name', 'property_location', 'facial_recognition', 'auto_created'
	Signature         []byte             `json:"signature" db:"signature"`
	VerifiedAt        *time.Time         `json:"verified_at" db:"verified_at"`
	VerifiedBy        string             `json:"verified_by" db:"verified_by"`
}

// DocumentFields represents parsed fields from a scanned document
type DocumentFields struct {
	// From property/cadastral documents
	OwnerName        string  `json:"owner_name"`
	OwnerEmail       *string `json:"owner_email"`
	OwnerPhone       *string `json:"owner_phone"`
	PropertyAddress  string  `json:"property_address"`
	PropertyID       string  `json:"property_id"`       // Cadastral parcel ID
	PropertyLocation [2]float64 `json:"property_location"` // [lat, lon]

	// From ID documents
	FullName       string     `json:"full_name"`
	IDNumber       string     `json:"id_number"`
	DateOfBirth    *time.Time `json:"date_of_birth"`
	Nationality    string     `json:"nationality"`
	IssuingCountry string     `json:"issuing_country"`

	// Photo data
	HasPhoto      bool   `json:"has_photo"`
	PhotoBase64   string `json:"photo_base64"`   // For facial recognition
	PhotoMIMEType string `json:"photo_mime_type"` // 'image/jpeg', 'image/png'

	// Document metadata
	DocumentDate      *time.Time `json:"document_date"`
	ExpiryDate        *time.Time `json:"expiry_date"`
	IssuingAuthority  string     `json:"issuing_authority"`

	// Confidence scores for each field
	FieldConfidences map[string]float64 `json:"field_confidences"`
}

// Scan implements sql.Scanner for JSONB
func (df *DocumentFields) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &df)
}

// Value implements driver.Valuer for JSONB
func (df DocumentFields) Value() (driver.Value, error) {
	return json.Marshal(df)
}

// FaceEmbedding represents a facial recognition embedding
type FaceEmbedding struct {
	EmbeddingID uuid.UUID  `json:"embedding_id" db:"embedding_id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Embedding   []float64  `json:"embedding" db:"embedding"`     // 512-dimensional vector
	CapturedAt  time.Time  `json:"captured_at" db:"captured_at"`
	PhotoSource string     `json:"photo_source" db:"photo_source"` // 'id_card', 'passport', 'selfie', 'document_scan'
	Confidence  float64    `json:"confidence" db:"confidence"`      // Quality of face detection
}

// PropertyOwnership represents verified property ownership
type PropertyOwnership struct {
	OwnershipID              uuid.UUID  `json:"ownership_id" db:"ownership_id"`
	UserID                   uuid.UUID  `json:"user_id" db:"user_id"`
	PropertyID               string     `json:"property_id" db:"property_id"`       // Cadastral ID
	PropertyAddress          string     `json:"property_address" db:"property_address"`
	Location                 [2]float64 `json:"location" db:"location"`            // [lat, lon]
	OwnerName                string     `json:"owner_name" db:"owner_name"`
	OwnershipVerified        bool       `json:"ownership_verified" db:"ownership_verified"`
	VerifiedViaDocumentID    *uuid.UUID `json:"verified_via_document_id" db:"verified_via_document_id"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
}

// DocumentScanRequest is the request to upload and scan a document
type DocumentScanRequest struct {
	DocumentType string `json:"document_type" binding:"required"` // 'cadastral_deed', 'id_card', 'passport', 'property_photo'
	ImageBase64  string `json:"image_base64" binding:"required"`  // Base64 encoded image
}

// DocumentScanResponse is the response after scanning
type DocumentScanResponse struct {
	ScanID              uuid.UUID       `json:"scan_id"`
	ExtractionStatus    string          `json:"extraction_status"`
	OCRConfidence       float64         `json:"ocr_confidence"`
	ExtractedFields     DocumentFields  `json:"extracted_fields"`
	MatchConfidence     float64         `json:"match_confidence"`
	MatchedUserID       *uuid.UUID      `json:"matched_user_id"`
	MatchMethod         string          `json:"match_method"`
	AutoCreatedUserID   *uuid.UUID      `json:"auto_created_user_id"` // If user was auto-created
	AccessToken         *string         `json:"access_token"`         // If auto-identified
	Message             string          `json:"message"`
}

// FacialRecognitionRequest for facial login
type FacialRecognitionRequest struct {
	PhotoBase64 string `json:"photo_base64" binding:"required"` // Base64 encoded photo
}

// FacialRecognitionResponse for facial login result
type FacialRecognitionResponse struct {
	Matched       bool       `json:"matched"`
	MatchedUserID *uuid.UUID `json:"matched_user_id"`
	Confidence    float64    `json:"confidence"`
	AccessToken   *string    `json:"access_token"` // If match successful
	Message       string     `json:"message"`
}

// IdentityVerificationResult combines all verification signals
type IdentityVerificationResult struct {
	UserID                uuid.UUID  `json:"user_id"`
	CredentialScore       float64    `json:"credential_score"`   // 0-1
	DeviceScore           float64    `json:"device_score"`       // 0-1
	DocumentScore         float64    `json:"document_score"`     // 0-1
	FacialScore           float64    `json:"facial_score"`       // 0-1
	OverallConfidence     float64    `json:"overall_confidence"` // 0-1
	RiskLevel             string     `json:"risk_level"`         // 'low', 'medium', 'high', 'critical'
	RequiredVerification  string     `json:"required_verification"` // '', 'email', '2fa', 'security_question'
	VerificationSources   []string   `json:"verification_sources"`
}
