package api

import (
	"cadastre_ia/internal/config"
	"cadastre_ia/internal/models"
	"cadastre_ia/internal/service"
	"cadastre_ia/internal/storage"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DocumentHandlers manages document scanning and identification endpoints
type DocumentHandlers struct {
	store     *storage.Storage
	ocrSvc    *service.OCRService
	faceSvc   *service.FacialRecognitionService
	identSvc  *service.UserIdentificationService
	cfg       *config.Config
}

// NewDocumentHandlers creates new document handlers
func NewDocumentHandlers(store *storage.Storage, cfg *config.Config) *DocumentHandlers {
	ocrSvc := service.NewOCRService("/tmp", "")
	faceSvc := service.NewFacialRecognitionService()
	identSvc := service.NewUserIdentificationService(store, ocrSvc, faceSvc)

	return &DocumentHandlers{
		store:    store,
		ocrSvc:   ocrSvc,
		faceSvc:  faceSvc,
		identSvc: identSvc,
		cfg:      cfg,
	}
}

// ScanDocument handles document upload and OCR processing
// POST /api/v1/documents/scan
func (h *DocumentHandlers) ScanDocument(store *storage.Storage, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.DocumentScanRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
				"details": err.Error(),
			})
			return
		}

		// Create scan record
		scan := &models.DocumentScan{
			ScanID:           uuid.New(),
			DocumentType:     req.DocumentType,
			ScannedAt:        time.Now(),
			ExtractionStatus: "processing",
		}

		// Process image with OCR
		fields, ocrConfidence, extractedText, err := h.ocrSvc.ProcessDocumentImage(
			req.ImageBase64,
			req.DocumentType,
		)

		scan.OCRConfidence = ocrConfidence
		scan.ExtractedText = extractedText
		scan.ExtractedFields = *fields

		if err != nil {
			scan.ExtractionStatus = "failed"
			c.JSON(http.StatusOK, models.DocumentScanResponse{
				ScanID:           scan.ScanID,
				ExtractionStatus: "failed",
				Message:          fmt.Sprintf("OCR failed: %v", err),
			})
			return
		}

		scan.ExtractionStatus = "success"

		// Try to identify user from document
		matchedUserID, matchMethod, matchConfidence, _ := h.identSvc.IdentifyUserFromDocument(fields)

		scan.MatchedUserID = matchedUserID
		scan.MatchMethod = matchMethod
		scan.MatchConfidence = matchConfidence

		// Try facial recognition if photo available
		if fields.HasPhoto && matchConfidence < 0.75 {
			faceID, faceConfidence, err := h.identSvc.IdentifyUserByFacialRecognition(fields.PhotoBase64)
			if err == nil && faceID != nil && faceConfidence > 0.80 {
				scan.MatchedUserID = faceID
				scan.MatchMethod = "facial_recognition"
				scan.MatchConfidence = faceConfidence
			}
		}

		response := models.DocumentScanResponse{
			ScanID:           scan.ScanID,
			ExtractionStatus: scan.ExtractionStatus,
			OCRConfidence:    ocrConfidence,
			ExtractedFields:  *fields,
			MatchConfidence:  scan.MatchConfidence,
			MatchedUserID:    scan.MatchedUserID,
			MatchMethod:      scan.MatchMethod,
			Message:          fmt.Sprintf("Document scanned successfully. Match confidence: %.1f%%", matchConfidence*100),
		}

		// If high confidence match, auto-login or create account
		if matchConfidence > 0.85 && scan.MatchedUserID != nil {
			// User identified with high confidence
			token, err := generateJWTToken(scan.MatchedUserID.String(), cfg.JWTSecret)
			if err == nil {
				response.AccessToken = &token
				response.Message = fmt.Sprintf("Welcome back! Auto-identified as %s", fields.OwnerName)
			}
		} else if matchConfidence > 0.70 {
			// Medium confidence - ask for user confirmation
			response.Message = fmt.Sprintf("Document recognized. Please confirm your identity. (%.1f%% match)", matchConfidence*100)
		} else {
			// Low confidence - create new account
			newUserID, err := h.identSvc.AutoCreateUserFromDocument(fields, scan.ScanID)
			if err == nil && newUserID != nil {
				scan.MatchedUserID = newUserID
				scan.MatchMethod = "auto_created"
				response.AutoCreatedUserID = newUserID
				response.MatchedUserID = newUserID

				// Generate token for new user
				token, err := generateJWTToken(newUserID.String(), cfg.JWTSecret)
				if err == nil {
					response.AccessToken = &token
					response.Message = fmt.Sprintf("Account created for %s. Welcome!", fields.OwnerName)
				}
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// ConfirmDocumentIdentification allows user to confirm/correct identification
// POST /api/v1/documents/:scan_id/verify
func (h *DocumentHandlers) ConfirmDocumentIdentification(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		scanID := c.Param("scan_id")

		var req struct {
			UserID       string `json:"user_id" binding:"required"`
			Confirmation bool   `json:"confirmation" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Update scan record with confirmed identification
		// In production, would update database
		// store.UpdateDocumentScanVerification(scanID, req.UserID, true)

		c.JSON(http.StatusOK, gin.H{
			"scan_id": scanID,
			"status":  "verified",
			"message": "Document identification confirmed",
		})
	}
}

// GetDocumentHistory returns all documents scanned by current user
// GET /api/v1/documents/my-scans
func (h *DocumentHandlers) GetDocumentHistory(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Query all scans for this user
		// scans := store.GetUserDocumentScans(userID.(string))

		scans := []models.DocumentScan{} // Placeholder

		c.JSON(http.StatusOK, gin.H{
			"scans": scans,
			"count": len(scans),
		})
	}
}

// RegisterFacePhoto stores a face embedding for facial recognition login
// POST /api/v1/identity/register-face
func (h *DocumentHandlers) RegisterFacePhoto(store *storage.Storage, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req struct {
			PhotoBase64 string `json:"photo_base64" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Validate face quality
		isValid, quality, message := h.faceSvc.ValidateFaceQuality(req.PhotoBase64)
		if !isValid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "face quality insufficient",
				"quality": quality,
				"message": message,
			})
			return
		}

		// Extract face embedding
		embedding, confidence, err := h.faceSvc.ExtractFaceEmbedding(req.PhotoBase64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "face extraction failed",
			})
			return
		}

		// Store embedding
		userIDParsed, _ := uuid.Parse(userID.(string))
		faceEmbedding, err := h.faceSvc.StoreFaceEmbedding(
			userID.(string),
			embedding,
			"selfie",
			confidence,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to store face embedding",
			})
			return
		}

		faceEmbedding.UserID = userIDParsed // Set user ID

		c.JSON(http.StatusOK, gin.H{
			"status":       "face_registered",
			"confidence":   confidence,
			"message":      "Face photo registered for facial recognition login",
		})
	}
}

// FacialLogin authenticates user by facial recognition
// POST /api/v1/identity/login-facial
func (h *DocumentHandlers) FacialLogin(store *storage.Storage, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.FacialRecognitionRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Validate face quality
		isValid, _, _ := h.faceSvc.ValidateFaceQuality(req.PhotoBase64)
		if !isValid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "face quality insufficient",
			})
			return
		}

		// Try facial recognition
		matchedUserID, similarity, err := h.identSvc.IdentifyUserByFacialRecognition(req.PhotoBase64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "facial recognition failed",
			})
			return
		}

		response := models.FacialRecognitionResponse{}

		if matchedUserID != nil && similarity > 0.80 {
			// Generate token
			token, err := generateJWTToken(matchedUserID.String(), cfg.JWTSecret)
			if err == nil {
				response.Matched = true
				response.MatchedUserID = matchedUserID
				response.Confidence = similarity
				response.AccessToken = &token
				response.Message = "Facial recognition successful"

				c.JSON(http.StatusOK, response)
				return
			}
		}

		response.Matched = false
		response.Confidence = similarity
		response.Message = "Face not recognized"

		c.JSON(http.StatusUnauthorized, response)
	}
}

// GetDocumentSummary returns summary of user's identified properties
// GET /api/v1/documents/properties
func (h *DocumentHandlers) GetDocumentSummary(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Query all properties for user
		// properties := store.GetUserProperties(userID.(string))

		properties := []models.PropertyOwnership{} // Placeholder

		c.JSON(http.StatusOK, gin.H{
			"properties": properties,
			"count":      len(properties),
		})
	}
}

// generateJWTToken is a helper to generate JWT token
func generateJWTToken(userID string, secret string) (string, error) {
	// In production, would use proper JWT library
	// For now, return placeholder
	token := fmt.Sprintf("jwt_token_for_%s", userID)
	return token, nil
}
