package service

import (
	"cadastre_ia/internal/models"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OCRService handles optical character recognition and document analysis
type OCRService struct {
	tempDir     string
	ocrEndpoint string // For cloud OCR (Google Vision API, etc.)
}

// NewOCRService creates a new OCR service
func NewOCRService(tempDir, ocrEndpoint string) *OCRService {
	return &OCRService{
		tempDir:     tempDir,
		ocrEndpoint: ocrEndpoint,
	}
}

// ProcessDocumentImage takes a base64 image and extracts text via OCR
func (s *OCRService) ProcessDocumentImage(imageBase64 string, docType string) (*models.DocumentFields, float64, string, error) {
	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Save image temporarily
	imagePath := filepath.Join(s.tempDir, fmt.Sprintf("doc_%d.jpg", time.Now().UnixNano()))
	err = ioutil.WriteFile(imagePath, imageData, 0644)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to save image: %w", err)
	}
	defer os.Remove(imagePath)

	// Run OCR (would call Tesseract or Cloud Vision API here)
	// For now, use placeholder
	extractedText, ocrConfidence := s.runOCR(imagePath)

	if ocrConfidence < 0.4 {
		return nil, ocrConfidence, "", fmt.Errorf("OCR confidence too low: %.2f", ocrConfidence)
	}

	// Parse extracted text based on document type
	fields := s.parseDocumentFields(extractedText, docType, imageBase64)

	return fields, ocrConfidence, extractedText, nil
}

// runOCR performs optical character recognition on an image
// In production, this would call Tesseract or a cloud OCR API
func (s *OCRService) runOCR(imagePath string) (string, float64) {
	// PLACEHOLDER: In production, integrate with:
	// 1. Tesseract via github.com/otiai10/gosseract/v2
	// 2. Google Cloud Vision API
	// 3. AWS Textract
	// 4. Azure Computer Vision

	// For now, return simulated OCR
	simulatedText := `
	ACTE DE PROPRIETE
	Titulaire: Jean Dupont
	Adresse: 123 Rue de la Paix, Paris
	Date: 2026-01-15
	Numero de parcelle: 75056-0001-AB-0123
	Email: jean.dupont@example.com
	Telephone: +33612345678
	`

	// In real implementation, would return actual OCR results
	return simulatedText, 0.85 // Simulated 85% confidence
}

// parseDocumentFields extracts structured data from OCR text based on document type
func (s *OCRService) parseDocumentFields(text, docType, photoBase64 string) *models.DocumentFields {
	fields := &models.DocumentFields{
		FieldConfidences: make(map[string]float64),
		PhotoBase64:      photoBase64,
		HasPhoto:         photoBase64 != "",
	}

	// Extract common fields
	fields.OwnerName = s.extractField(text, `(?i)(?:titulaire|owner|proprietaire|nom):\s*([A-Z][A-Za-z\s]+)`)
	fields.FullName = fields.OwnerName

	fields.OwnerEmail = s.extractOptionalField(text, `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	fields.OwnerPhone = s.extractOptionalField(text, `(?:\+33|0)[0-9\s\-\.]{8,}`)

	// Document-specific extraction
	switch docType {
	case "cadastral_deed", "property_deed":
		fields.PropertyAddress = s.extractField(text, `(?i)(?:adresse|address):\s*([^\n]+)`)
		fields.PropertyID = s.extractField(text, `(?i)(?:parcelle|parcel|numero):\s*([A-Z0-9\-]+)`)
		fields.DocumentDate = s.extractDate(text, `(?i)(?:date|dated):\s*(\d{1,2}[/-]\d{1,2}[/-]\d{4})`)

		fields.FieldConfidences["owner_name"] = 0.9
		fields.FieldConfidences["property_id"] = 0.85
		fields.FieldConfidences["property_address"] = 0.8

	case "id_card", "passport":
		fields.FullName = s.extractField(text, `(?i)(?:nom|name):\s*([A-Z][A-Za-z\s]+)`)
		fields.IDNumber = s.extractField(text, `(?i)(?:numero|number|id):\s*([A-Z0-9]+)`)
		fields.DateOfBirth = s.extractDate(text, `(?i)(?:nee?|birth|dob):\s*(\d{1,2}[/-]\d{1,2}[/-]\d{4})`)
		fields.Nationality = s.extractField(text, `(?i)nationalite:\s*([A-Z]{2})`)

		fields.FieldConfidences["full_name"] = 0.95
		fields.FieldConfidences["id_number"] = 0.9
		fields.FieldConfidences["date_of_birth"] = 0.85

	case "property_photo":
		// For property photos, try to extract visible text only
		fields.PropertyAddress = s.extractField(text, `(?i)(?:address|rue|avenue):\s*([^\n]+)`)
		fields.FieldConfidences["property_address"] = 0.7
	}

	return fields
}

// extractField extracts a single field using regex
func (s *OCRService) extractField(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractOptionalField extracts an optional field (returns pointer)
func (s *OCRService) extractOptionalField(text, pattern string) *string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		return &value
	}
	return nil
}

// extractDate parses a date string in various formats
func (s *OCRService) extractDate(text, pattern string) *time.Time {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		dateStr := matches[1]
		// Try various date formats
		formats := []string{
			"02/01/2006", "02-01-2006",
			"2006/01/02", "2006-01-02",
			"01/02/2006", "01-02-2006",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				return &t
			}
		}
	}
	return nil
}

// FuzzyStringMatch calculates Levenshtein distance for name matching
func (s *OCRService) FuzzyStringMatch(str1, str2 string) float64 {
	str1 = strings.ToLower(str1)
	str2 = strings.ToLower(str2)

	distance := levenshteinDistance(str1, str2)
	maxLen := math.Max(float64(len(str1)), float64(len(str2)))

	if maxLen == 0 {
		return 1.0
	}

	similarity := 1.0 - (float64(distance) / maxLen)
	return similarity
}

// levenshteinDistance calculates edit distance between two strings
func levenshteinDistance(str1, str2 string) int {
	lenStr1 := len(str1)
	lenStr2 := len(str2)

	matrix := make([][]int, lenStr1+1)
	for i := range matrix {
		matrix[i] = make([]int, lenStr2+1)
		matrix[i][0] = i
	}
	for j := 0; j <= lenStr2; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= lenStr1; i++ {
		for j := 1; j <= lenStr2; j++ {
			cost := 0
			if str1[i-1] != str2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[lenStr1][lenStr2]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// IdentifyUserFromDocument attempts to identify or create a user based on document fields
type UserIdentificationResult struct {
	UserID              *string  `json:"user_id"`
	MatchMethod         string   `json:"match_method"`
	MatchConfidence     float64  `json:"match_confidence"`
	AutoCreated         bool     `json:"auto_created"`
	VerificationStatus  string   `json:"verification_status"`
	RequiresConfirmation bool    `json:"requires_confirmation"`
	Message             string   `json:"message"`
}

// GetMatchConfidence calculates combined confidence from multiple signals
func (s *OCRService) CalculateMatchConfidence(
	exactNameMatch, fuzzyNameMatch, emailMatch, locationMatch, propertyIDMatch float64,
) float64 {
	// Weighted combination of different match types
	weights := map[string]float64{
		"exact_name":      0.25,
		"fuzzy_name":      0.15,
		"email":           0.20,
		"location":        0.20,
		"property_id":     0.20,
	}

	confidence := exactNameMatch*weights["exact_name"] +
		fuzzyNameMatch*weights["fuzzy_name"] +
		emailMatch*weights["email"] +
		locationMatch*weights["location"] +
		propertyIDMatch*weights["property_id"]

	return confidence
}

// ExtractFaceFromDocument extracts face from document image for facial recognition
func (s *OCRService) ExtractFaceFromDocument(imageBase64 string) (string, float64, error) {
	// PLACEHOLDER: In production, use face detection library
	// like github.com/go-face/face or TensorFlow.js on frontend

	// For now, return the image as-is with high confidence that it contains a face
	// In real implementation, would use face detection to verify

	return imageBase64, 0.9, nil
}
