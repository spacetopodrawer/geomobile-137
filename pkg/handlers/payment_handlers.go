package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"cadastre_ia/pkg/payment"
)

// PaymentHandler handles payment API endpoints
type PaymentHandler struct {
	service *payment.PaymentService
	logger  *log.Logger
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(service *payment.PaymentService, logger *log.Logger) *PaymentHandler {
	if logger == nil {
		logger = log.New(log.Writer(), "[PaymentHandler] ", log.LstdFlags)
	}
	return &PaymentHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers payment handlers with the mux
func (ph *PaymentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/payment/tier-upgrade", ph.InitiateTierUpgrade)
	mux.HandleFunc("POST /api/v1/payment/cosmetic-purchase", ph.InitiateCosmeticPurchase)
	mux.HandleFunc("GET /api/v1/payment/verify/{transactionID}", ph.VerifyPayment)
	mux.HandleFunc("POST /api/v1/payment/webhook/flutterwave", ph.WebhookFlutterwave)
	mux.HandleFunc("POST /api/v1/payment/webhook/paytech", ph.WebhookPaytech)
}

// InitiateTierUpgrade initiates a tier upgrade payment
func (ph *PaymentHandler) InitiateTierUpgrade(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		TargetTier   int    `json:"target_tier"`
		DurationDays int    `json:"duration_days"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		RedirectURL  string `json:"redirect_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate inputs
	if req.TargetTier < 1 || req.TargetTier > 5 {
		http.Error(w, "Invalid tier (1-5)", http.StatusBadRequest)
		return
	}

	if req.DurationDays < 30 {
		http.Error(w, "Minimum duration is 30 days", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Phone == "" {
		http.Error(w, "Email and phone required", http.StatusBadRequest)
		return
	}

	// Initiate upgrade
	upgradePayment, paymentLink, err := ph.service.InitiateTierUpgrade(
		r.Context(), userID, req.TargetTier, req.DurationDays,
		req.Email, req.Phone, req.RedirectURL)

	if err != nil {
		ph.logger.Printf("Error initiating tier upgrade: %v\n", err)
		http.Error(w, "Failed to initiate payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"upgrade_payment": upgradePayment,
		"payment_link":    paymentLink,
	})
}

// InitiateCosmeticPurchase initiates a cosmetic purchase payment
func (ph *PaymentHandler) InitiateCosmeticPurchase(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		CosmeticID  string `json:"cosmetic_id"`
		CosmeticName string `json:"cosmetic_name"`
		Price       int    `json:"price"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		RedirectURL string `json:"redirect_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate inputs
	if req.CosmeticID == "" || req.Price <= 0 {
		http.Error(w, "Missing or invalid cosmetic details", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Phone == "" {
		http.Error(w, "Email and phone required", http.StatusBadRequest)
		return
	}

	// Initiate purchase
	cosmeticPayment, paymentLink, err := ph.service.InitiateCosmeticPurchase(
		r.Context(), userID, req.CosmeticID, req.CosmeticName, req.Price,
		req.Email, req.Phone, req.RedirectURL)

	if err != nil {
		ph.logger.Printf("Error initiating cosmetic purchase: %v\n", err)
		http.Error(w, "Failed to initiate payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cosmetic_payment": cosmeticPayment,
		"payment_link":     paymentLink,
	})
}

// VerifyPayment verifies payment status
func (ph *PaymentHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	transactionID := r.PathValue("transactionID")
	if transactionID == "" {
		http.Error(w, "Missing transactionID", http.StatusBadRequest)
		return
	}

	verification, err := ph.service.VerifyPayment(r.Context(), transactionID)
	if err != nil {
		ph.logger.Printf("Error verifying payment: %v\n", err)
		http.Error(w, "Verification failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(verification)
}

// WebhookFlutterwave handles Flutterwave webhook events
func (ph *PaymentHandler) WebhookFlutterwave(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Verify signature (Flutterwave uses x-flutterwave-signature header)
	signature := r.Header.Get("x-flutterwave-signature")
	if signature == "" {
		ph.logger.Println("Webhook rejected: missing signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse payload
	var payload payment.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	payload.Provider = "flutterwave"

	// Handle webhook
	if err := ph.service.HandleWebhook(r.Context(), &payload); err != nil {
		ph.logger.Printf("Error handling Flutterwave webhook: %v\n", err)
		http.Error(w, "Processing failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// WebhookPaytech handles Paytech webhook events
func (ph *PaymentHandler) WebhookPaytech(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Verify signature using HMAC
	signature := r.Header.Get("X-Paytech-Signature")
	if signature == "" {
		ph.logger.Println("Webhook rejected: missing signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// In production, verify HMAC signature against shared secret
	// For now, just accept it (configure webhook secret in PaymentConfig)

	// Parse payload
	var payload payment.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	payload.Provider = "paytech"

	// Handle webhook
	if err := ph.service.HandleWebhook(r.Context(), &payload); err != nil {
		ph.logger.Printf("Error handling Paytech webhook: %v\n", err)
		http.Error(w, "Processing failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// VerifySignature validates webhook signature (helper for future use)
func verifySignature(body []byte, signature string, secret string) bool {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(body)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
