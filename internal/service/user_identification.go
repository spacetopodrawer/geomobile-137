package service

import (
	"cadastre_ia/internal/models"
	"cadastre_ia/internal/storage"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UserIdentificationService identifies or creates users based on document scans
type UserIdentificationService struct {
	store   *storage.Storage
	ocrSvc  *OCRService
	faceSvc *FacialRecognitionService
}

// NewUserIdentificationService creates a new user identification service
func NewUserIdentificationService(store *storage.Storage, ocrSvc *OCRService, faceSvc *FacialRecognitionService) *UserIdentificationService {
	return &UserIdentificationService{
		store:   store,
		ocrSvc:  ocrSvc,
		faceSvc: faceSvc,
	}
}

// IdentifyUserFromDocument identifies a user from document fields
// Returns: (userID, matchMethod, confidence, error)
func (u *UserIdentificationService) IdentifyUserFromDocument(fields *models.DocumentFields) (*uuid.UUID, string, float64, error) {
	if fields.OwnerName == "" {
		return nil, "", 0, fmt.Errorf("no owner name extracted")
	}

	// Strategy 1: Exact email match (highest confidence)
	if fields.OwnerEmail != nil && *fields.OwnerEmail != "" {
		// PLACEHOLDER: In production, query user repository by email
		// user := u.store.FindUserByEmail(*fields.OwnerEmail)
		// if user != nil {
		//	return &user.ID, "exact_email", 0.95, nil
		// }
	}

	// Strategy 2: Fuzzy name match
	fuzzyMatch, fuzzyConfidence := u.fuzzyNameMatch(fields.OwnerName)
	if fuzzyMatch != nil && fuzzyConfidence > 0.85 {
		return &fuzzyMatch.ID, "fuzzy_name_match", fuzzyConfidence, nil
	}

	// Strategy 3: Property location match (for cadastral deeds)
	if len(fields.PropertyLocation) == 2 && fields.PropertyLocation[0] != 0 && fields.PropertyLocation[1] != 0 {
		locationMatch := u.propertyLocationMatch(fields.PropertyLocation)
		if locationMatch != nil {
			return &locationMatch.ID, "property_location", 0.80, nil
		}
	}

	// Strategy 4: Cadastral property ID match
	if fields.PropertyID != "" {
		propertyMatch := u.propertyIDMatch(fields.PropertyID)
		if propertyMatch != nil {
			return &propertyMatch.ID, "property_ownership", 0.90, nil
		}
	}

	// No match found
	return nil, "", 0, nil
}

// IdentifyUserByFacialRecognition identifies a user by facial recognition
func (u *UserIdentificationService) IdentifyUserByFacialRecognition(imageBase64 string) (*uuid.UUID, float64, error) {
	// Extract face embedding
	embedding, _, err := u.faceSvc.ExtractFaceEmbedding(imageBase64)
	if err != nil {
		return nil, 0, fmt.Errorf("face extraction failed: %w", err)
	}

	// Find best matching face in database
	matches, err := u.faceSvc.FindFacialMatch(embedding, 0.75)
	if err != nil {
		return nil, 0, fmt.Errorf("facial search failed: %w", err)
	}

	if len(matches) > 0 && matches[0].Similarity > 0.80 {
		userID := uuid.MustParse(matches[0].UserID)
		return &userID, matches[0].Similarity, nil
	}

	return nil, 0, nil
}

// AutoCreateUserFromDocument creates a new user from document data
// Used when document confidence is high but no matching user found
func (u *UserIdentificationService) AutoCreateUserFromDocument(fields *models.DocumentFields, documentID uuid.UUID) (*uuid.UUID, error) {
	if fields.OwnerName == "" {
		return nil, fmt.Errorf("cannot create user without name")
	}

	// Create new user account
	userID := uuid.New()

	// Store in memory (would use database in production)
	// user := &storage.User{
	//   UserID: userID,
	//   Email: *fields.OwnerEmail,
	//   FullName: fields.OwnerName,
	//   PhoneNumber: *fields.OwnerPhone,
	//   CreatedAt: time.Now(),
	//   VerificationStatus: "document_verified",
	// }
	// u.store.InsertUser(user)

	// Create property ownership record
	if fields.PropertyID != "" {
		_ = &models.PropertyOwnership{
			OwnershipID:           uuid.New(),
			UserID:                userID,
			PropertyID:            fields.PropertyID,
			PropertyAddress:       fields.PropertyAddress,
			OwnerName:             fields.OwnerName,
			OwnershipVerified:     true,
			VerifiedViaDocumentID: &documentID,
		}
		// u.store.InsertPropertyOwnership(ownership)
	}

	return &userID, nil
}

// fuzzyNameMatch finds users with similar names
func (u *UserIdentificationService) fuzzyNameMatch(targetName string) (*storage.User, float64) {
	// PLACEHOLDER: In production, would query all users and fuzzy match
	// For now, return nil
	_ = targetName
	return nil, 0.0
}

// propertyLocationMatch finds users owning properties near a location
func (u *UserIdentificationService) propertyLocationMatch(location [2]float64) *storage.User {
	// PLACEHOLDER: Query database for properties near location
	// SELECT user_id FROM property_ownership
	// WHERE ST_DWithin(location, point($lat, $lon), 500)
	_ = location
	return nil
}

// propertyIDMatch finds the owner of a specific cadastral property
func (u *UserIdentificationService) propertyIDMatch(propertyID string) *storage.User {
	// PLACEHOLDER: Query database for property owner
	// SELECT u.* FROM users u
	// JOIN property_ownership p ON u.user_id = p.user_id
	// WHERE p.property_id = ?
	_ = propertyID
	return nil
}

// CalculateOverallVerificationScore combines multiple verification signals
func (u *UserIdentificationService) CalculateOverallVerificationScore(
	userID *uuid.UUID,
	credentialScore, deviceScore, documentScore, facialScore float64,
) *models.IdentityVerificationResult {
	result := &models.IdentityVerificationResult{
		CredentialScore:   credentialScore,
		DeviceScore:       deviceScore,
		DocumentScore:     documentScore,
		FacialScore:       facialScore,
		VerificationSources: []string{},
	}

	if userID != nil {
		result.UserID = *userID
	}

	// Add sources that were used
	if credentialScore > 0 {
		result.VerificationSources = append(result.VerificationSources, "credentials")
	}
	if deviceScore > 0 {
		result.VerificationSources = append(result.VerificationSources, "device")
	}
	if documentScore > 0 {
		result.VerificationSources = append(result.VerificationSources, "document")
	}
	if facialScore > 0 {
		result.VerificationSources = append(result.VerificationSources, "facial")
	}

	// Weighted combination
	weights := map[string]float64{
		"credential": 0.30,
		"device":     0.15,
		"document":   0.35,
		"facial":     0.20,
	}

	result.OverallConfidence = credentialScore*weights["credential"] +
		deviceScore*weights["device"] +
		documentScore*weights["document"] +
		facialScore*weights["facial"]

	// Determine risk level
	if result.OverallConfidence > 0.95 {
		result.RiskLevel = "low"
		result.RequiredVerification = ""
	} else if result.OverallConfidence > 0.80 {
		result.RiskLevel = "medium"
		result.RequiredVerification = "email"
	} else if result.OverallConfidence > 0.60 {
		result.RiskLevel = "high"
		result.RequiredVerification = "2fa"
	} else {
		result.RiskLevel = "critical"
		result.RequiredVerification = "manual_review"
	}

	return result
}

// ExtractAndNormalizeEmail normalizes an email address for matching
func (u *UserIdentificationService) ExtractAndNormalizeEmail(email string) string {
	email = strings.ToLower(email)
	email = strings.TrimSpace(email)
	return email
}

// ExtractAndNormalizeName normalizes a name for fuzzy matching
func (u *UserIdentificationService) ExtractAndNormalizeName(name string) string {
	// Remove accents, extra spaces, convert to uppercase
	name = strings.ToUpper(name)
	name = strings.TrimSpace(name)
	name = strings.Join(strings.Fields(name), " ") // Normalize spaces
	return name
}
