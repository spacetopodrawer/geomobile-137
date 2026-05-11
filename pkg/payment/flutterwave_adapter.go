package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// FlutterwaveAdapter implements PaymentGateway for Flutterwave
type FlutterwaveAdapter struct {
	publicKey  string
	secretKey  string
	baseURL    string
	httpClient *http.Client
	logger     *log.Logger
}

// NewFlutterwaveAdapter creates a new Flutterwave adapter
func NewFlutterwaveAdapter(config *PaymentConfig, logger *log.Logger) *FlutterwaveAdapter {
	if logger == nil {
		logger = log.New(log.Writer(), "[FlutterwaveAdapter] ", log.LstdFlags)
	}

	baseURL := config.FlutterwaveBaseURL
	if baseURL == "" {
		baseURL = "https://api.flutterwave.com"
	}

	return &FlutterwaveAdapter{
		publicKey: config.FlutterwavePublicKey,
		secretKey: config.FlutterwaveSecretKey,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		logger: logger,
	}
}

// InitializePayment creates a payment transaction with Flutterwave
func (fa *FlutterwaveAdapter) InitializePayment(transaction *PaymentTransaction) (string, string, error) {
	if fa.secretKey == "" {
		return "", "", fmt.Errorf("flutterwave secret key not configured")
	}

	// Prepare Flutterwave request
	fwRequest := map[string]interface{}{
		"public_key": fa.publicKey,
		"tx_ref":     transaction.TransactionID,
		"amount":     float64(transaction.Amount) / 100.0, // Convert to currency (XAF is typically in smallest units)
		"currency":   transaction.Currency,
		"customer": map[string]interface{}{
			"email": transaction.CustomerEmail,
			"phone_number": transaction.CustomerPhone,
		},
		"customizations": map[string]interface{}{
			"title":       "Geomobile137 - " + transaction.ItemName,
			"description": transaction.Description,
			"logo":        "https://geomobile.app/logo.png",
		},
		"meta": map[string]interface{}{
			"user_id": transaction.UserID,
			"type":    string(transaction.Type),
			"item_id": transaction.ItemID,
		},
		"redirect_url": transaction.RedirectURL,
	}

	// Marshal to JSON
	payload, err := json.Marshal(fwRequest)
	if err != nil {
		fa.logger.Printf("Error marshaling Flutterwave request: %v\n", err)
		return "", "", fmt.Errorf("failed to prepare payment request: %w", err)
	}

	// Make request to Flutterwave
	req, err := http.NewRequest("POST", fa.baseURL+"/v3/payments", bytes.NewBuffer(payload))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fa.secretKey)

	resp, err := fa.httpClient.Do(req)
	if err != nil {
		fa.logger.Printf("Error calling Flutterwave API: %v\n", err)
		return "", "", fmt.Errorf("payment gateway unreachable: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var fwResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Link string `json:"link"`
			ID   int    `json:"id"`
		} `json:"data"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &fwResponse); err != nil {
		fa.logger.Printf("Error parsing Flutterwave response: %v\nBody: %s\n", err, string(body))
		return "", "", fmt.Errorf("invalid gateway response: %w", err)
	}

	if fwResponse.Status != "success" {
		return "", "", fmt.Errorf("flutterwave error: %s", fwResponse.Message)
	}

	if fwResponse.Data.Link == "" {
		return "", "", fmt.Errorf("no payment link returned from gateway")
	}

	// Store gateway transaction ID
	gatewayTxID := fmt.Sprintf("fw_%d", fwResponse.Data.ID)

	fa.logger.Printf("Payment initialized: %s -> %s (Gateway: %s)\n", transaction.TransactionID, fwResponse.Data.Link, gatewayTxID)

	return fwResponse.Data.Link, gatewayTxID, nil
}

// VerifyPayment checks payment status with Flutterwave
func (fa *FlutterwaveAdapter) VerifyPayment(transactionID string) (*PaymentVerification, error) {
	if fa.secretKey == "" {
		return nil, fmt.Errorf("flutterwave secret key not configured")
	}

	// Make request to Flutterwave verification endpoint
	req, err := http.NewRequest("GET", fa.baseURL+"/v3/transactions/verify_by_reference?tx_ref="+transactionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer " + fa.secretKey)

	resp, err := fa.httpClient.Do(req)
	if err != nil {
		fa.logger.Printf("Error verifying payment: %v\n", err)
		return nil, fmt.Errorf("verification failed: %w", err)
	}
	defer resp.Body.Close()

	var fwResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status         string `json:"status"`
			Amount         float64 `json:"amount"`
			Currency       string `json:"currency"`
			PaymentType    string `json:"payment_type"`
			ID             int    `json:"id"`
			TxRef          string `json:"tx_ref"`
			FlwRef         string `json:"flw_ref"`
		} `json:"data"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &fwResponse); err != nil {
		fa.logger.Printf("Error parsing verification response: %v\n", err)
		return nil, fmt.Errorf("invalid verification response: %w", err)
	}

	if fwResponse.Status != "success" {
		return nil, fmt.Errorf("verification failed: %s", fwResponse.Message)
	}

	// Convert Flutterwave status to our PaymentStatus
	status := convertFlutterwaveStatus(fwResponse.Data.Status)

	verification := &PaymentVerification{
		TransactionID:    transactionID,
		Status:           status,
		Amount:           int(fwResponse.Data.Amount * 100), // Convert back to smallest units
		Currency:         fwResponse.Data.Currency,
		PaymentMethod:    convertFlutterwavePaymentMethod(fwResponse.Data.PaymentType),
		PaymentReference: fwResponse.Data.FlwRef,
	}

	if status == StatusCompleted {
		now := time.Now()
		verification.PaidAt = &now
	}

	fa.logger.Printf("Payment verified: %s -> %s\n", transactionID, status)

	return verification, nil
}

// RefundPayment initiates a refund with Flutterwave
func (fa *FlutterwaveAdapter) RefundPayment(transactionID string, amount int) error {
	if fa.secretKey == "" {
		return fmt.Errorf("flutterwave secret key not configured")
	}

	// For Flutterwave, we need the transaction ID from their system
	// This is simplified - in production, we'd look up the gateway transaction ID

	refundRequest := map[string]interface{}{
		"amount": float64(amount) / 100.0,
	}

	payload, err := json.Marshal(refundRequest)
	if err != nil {
		return fmt.Errorf("failed to prepare refund request: %w", err)
	}

	// Make refund request
	req, err := http.NewRequest("POST", fa.baseURL+"/v3/transactions/"+transactionID+"/refund", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create refund request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fa.secretKey)

	resp, err := fa.httpClient.Do(req)
	if err != nil {
		fa.logger.Printf("Error requesting refund: %v\n", err)
		return fmt.Errorf("refund request failed: %w", err)
	}
	defer resp.Body.Close()

	var fwResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &fwResponse); err != nil {
		return fmt.Errorf("invalid refund response: %w", err)
	}

	if fwResponse.Status != "success" {
		return fmt.Errorf("refund failed: %s", fwResponse.Message)
	}

	fa.logger.Printf("Refund initiated: %s (amount: %d)\n", transactionID, amount)
	return nil
}

// GetTransactionStatus retrieves current payment status
func (fa *FlutterwaveAdapter) GetTransactionStatus(transactionID string) (PaymentStatus, error) {
	verification, err := fa.VerifyPayment(transactionID)
	if err != nil {
		return StatusFailed, err
	}
	return verification.Status, nil
}

// Helper functions

func convertFlutterwaveStatus(fwStatus string) PaymentStatus {
	switch fwStatus {
	case "successful":
		return StatusCompleted
	case "pending":
		return StatusPending
	case "failed":
		return StatusFailed
	case "cancelled":
		return StatusCancelled
	default:
		return StatusProcessing
	}
}

func convertFlutterwavePaymentMethod(method string) PaymentMethod {
	switch method {
	case "mtn":
		return MethodMTNMobileMoney
	case "orange":
		return MethodOrangeMobileMoney
	case "vodafone":
		return MethodVodacomMobileMoney
	case "card":
		return MethodVISA // Could be either VISA or Mastercard
	case "bank_transfer":
		return MethodBankTransfer
	default:
		return PaymentMethod(method)
	}
}
