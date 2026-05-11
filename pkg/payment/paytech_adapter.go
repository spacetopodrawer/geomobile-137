package payment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PaytechAdapter implements PaymentGateway for Paytech (Senegal-based)
type PaytechAdapter struct {
	merchantID string
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *log.Logger
}

// NewPaytechAdapter creates a new Paytech adapter
func NewPaytechAdapter(config *PaymentConfig, logger *log.Logger) *PaytechAdapter {
	if logger == nil {
		logger = log.New(log.Writer(), "[PaytechAdapter] ", log.LstdFlags)
	}

	baseURL := config.PaytechBaseURL
	if baseURL == "" {
		baseURL = "https://api.paytech.sn"
	}

	return &PaytechAdapter{
		merchantID: config.PaytechMerchantID,
		apiKey:     config.PaytechAPIKey,
		baseURL:    baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		logger: logger,
	}
}

// InitializePayment creates a payment transaction with Paytech
func (pa *PaytechAdapter) InitializePayment(transaction *PaymentTransaction) (string, string, error) {
	if pa.merchantID == "" || pa.apiKey == "" {
		return "", "", fmt.Errorf("paytech credentials not configured")
	}

	// Prepare Paytech request using form encoding
	formData := url.Values{}
	formData.Set("merchant_id", pa.merchantID)
	formData.Set("type", "SELL")
	formData.Set("api_key", pa.apiKey)
	formData.Set("amount", fmt.Sprintf("%.2f", float64(transaction.Amount)/100.0))
	formData.Set("currency", transaction.Currency)
	formData.Set("order_id", transaction.TransactionID)
	formData.Set("customer_email", transaction.CustomerEmail)
	formData.Set("customer_phone", transaction.CustomerPhone)
	formData.Set("item_name", transaction.ItemName)
	formData.Set("item_description", transaction.Description)
	formData.Set("custom_field", transaction.UserID) // User ID for reconciliation
	formData.Set("return_url", transaction.RedirectURL)
	formData.Set("cancel_url", transaction.RedirectURL + "?cancelled=true")
	formData.Set("webhook_url", "") // Can be set from config

	// Make request to Paytech
	req, err := http.NewRequest("POST", pa.baseURL+"/api/v1/checkout/", strings.NewReader(formData.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		pa.logger.Printf("Error calling Paytech API: %v\n", err)
		return "", "", fmt.Errorf("payment gateway unreachable: %w", err)
	}
	defer resp.Body.Close()

	// Parse JSON response
	var ptResponse struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Token   string `json:"token"`
		Link    string `json:"link"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &ptResponse); err != nil {
		pa.logger.Printf("Error parsing Paytech response: %v\nBody: %s\n", err, string(body))
		return "", "", fmt.Errorf("invalid gateway response: %w", err)
	}

	if ptResponse.Status != 200 {
		return "", "", fmt.Errorf("paytech error: %s", ptResponse.Message)
	}

	// Paytech returns a token to redirect to their hosted payment page
	paymentLink := fmt.Sprintf("%s/pay/%s", pa.baseURL, ptResponse.Token)

	pa.logger.Printf("Payment initialized: %s -> %s (Token: %s)\n", transaction.TransactionID, paymentLink, ptResponse.Token)

	return paymentLink, ptResponse.Token, nil
}

// VerifyPayment checks payment status with Paytech
func (pa *PaytechAdapter) VerifyPayment(transactionID string) (*PaymentVerification, error) {
	if pa.merchantID == "" || pa.apiKey == "" {
		return nil, fmt.Errorf("paytech credentials not configured")
	}

	// Create signature for verification request
	signatureData := fmt.Sprintf("%s%s%s", pa.merchantID, transactionID, pa.apiKey)
	signature := sha256.Sum256([]byte(signatureData))
	signatureHex := hex.EncodeToString(signature[:])

	// Make request to verify transaction
	queryParams := url.Values{}
	queryParams.Set("merchant_id", pa.merchantID)
	queryParams.Set("order_id", transactionID)
	queryParams.Set("signature", signatureHex)

	req, err := http.NewRequest("GET", pa.baseURL+"/api/v1/transaction/check/?"+queryParams.Encode(), nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create verification request: %w", err)
	}

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		pa.logger.Printf("Error verifying payment: %v\n", err)
		return nil, fmt.Errorf("verification failed: %w", err)
	}
	defer resp.Body.Close()

	var ptResponse struct {
		Status    int    `json:"status"`
		Message   string `json:"message"`
		OrderID   string `json:"order_id"`
		Reference string `json:"reference"`
		Amount    float64 `json:"amount"`
		Currency  string `json:"currency"`
		PaymentMethod string `json:"payment_method"`
		TransactionStatus string `json:"transaction_status"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &ptResponse); err != nil {
		pa.logger.Printf("Error parsing verification response: %v\n", err)
		return nil, fmt.Errorf("invalid verification response: %w", err)
	}

	if ptResponse.Status != 200 {
		return nil, fmt.Errorf("verification failed: %s", ptResponse.Message)
	}

	// Convert Paytech status to our PaymentStatus
	status := convertPaytechStatus(ptResponse.TransactionStatus)

	verification := &PaymentVerification{
		TransactionID:    transactionID,
		Status:           status,
		Amount:           int(ptResponse.Amount * 100), // Convert back to smallest units
		Currency:         ptResponse.Currency,
		PaymentMethod:    convertPaytechPaymentMethod(ptResponse.PaymentMethod),
		PaymentReference: ptResponse.Reference,
	}

	if status == StatusCompleted {
		now := time.Now()
		verification.PaidAt = &now
	}

	pa.logger.Printf("Payment verified: %s -> %s\n", transactionID, status)

	return verification, nil
}

// RefundPayment initiates a refund with Paytech
func (pa *PaytechAdapter) RefundPayment(transactionID string, amount int) error {
	if pa.merchantID == "" || pa.apiKey == "" {
		return fmt.Errorf("paytech credentials not configured")
	}

	// Create signature
	signatureData := fmt.Sprintf("%s%s%s", pa.merchantID, transactionID, pa.apiKey)
	signature := sha256.Sum256([]byte(signatureData))
	signatureHex := hex.EncodeToString(signature[:])

	refundRequest := map[string]interface{}{
		"merchant_id": pa.merchantID,
		"order_id":    transactionID,
		"amount":      float64(amount) / 100.0,
		"signature":   signatureHex,
	}

	payload, err := json.Marshal(refundRequest)
	if err != nil {
		return fmt.Errorf("failed to prepare refund request: %w", err)
	}

	// Make refund request
	req, err := http.NewRequest("POST", pa.baseURL+"/api/v1/transaction/refund/", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create refund request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		pa.logger.Printf("Error requesting refund: %v\n", err)
		return fmt.Errorf("refund request failed: %w", err)
	}
	defer resp.Body.Close()

	var ptResponse struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &ptResponse); err != nil {
		return fmt.Errorf("invalid refund response: %w", err)
	}

	if ptResponse.Status != 200 {
		return fmt.Errorf("refund failed: %s", ptResponse.Message)
	}

	pa.logger.Printf("Refund initiated: %s (amount: %d)\n", transactionID, amount)
	return nil
}

// GetTransactionStatus retrieves current payment status
func (pa *PaytechAdapter) GetTransactionStatus(transactionID string) (PaymentStatus, error) {
	verification, err := pa.VerifyPayment(transactionID)
	if err != nil {
		return StatusFailed, err
	}
	return verification.Status, nil
}

// Helper functions

func convertPaytechStatus(ptStatus string) PaymentStatus {
	switch strings.ToLower(ptStatus) {
	case "successful", "completed":
		return StatusCompleted
	case "pending", "initiated":
		return StatusPending
	case "failed":
		return StatusFailed
	case "cancelled":
		return StatusCancelled
	case "refunded":
		return StatusRefunded
	default:
		return StatusProcessing
	}
}

func convertPaytechPaymentMethod(method string) PaymentMethod {
	switch strings.ToLower(method) {
	case "card":
		return MethodVISA
	case "bank":
		return MethodBankTransfer
	case "mobile_money":
		return MethodOrangeMobileMoney
	default:
		return PaymentMethod(method)
	}
}
