// Package payment provides payment gateway integration (Flutterwave, Paytech)
// Part of geo-mobile137 — Freemium Subscription & Cosmetics IAP
//
// Module: pkg/payment
// Purpose: Payment processing, tier upgrades, cosmetic purchases
// Currencies: XAF (CFA Franc) for Cameroon market
//
// Supported Payment Methods:
//   - Flutterwave (30+ payment methods)
//   - Paytech (SENEGAL - expandable to CM)
//   - Mobile Money (MTN, Orange, Vodafone)
//   - Card payments (Mastercard, VISA)
//   - Bank transfers

package payment

import (
	"time"
)

// PaymentGateway defines the interface for payment processing
type PaymentGateway interface {
	// InitializePayment creates a payment transaction
	InitializePayment(transaction *PaymentTransaction) (paymentLink string, transactionID string, err error)

	// VerifyPayment checks payment status with gateway
	VerifyPayment(transactionID string) (*PaymentVerification, error)

	// RefundPayment initiates a refund
	RefundPayment(transactionID string, amount int) error

	// GetTransactionStatus retrieves current status
	GetTransactionStatus(transactionID string) (PaymentStatus, error)
}

// PaymentStatus represents payment state
type PaymentStatus string

const (
	StatusPending     PaymentStatus = "pending"
	StatusInitiated   PaymentStatus = "initiated"
	StatusProcessing  PaymentStatus = "processing"
	StatusCompleted   PaymentStatus = "completed"
	StatusFailed      PaymentStatus = "failed"
	StatusCancelled   PaymentStatus = "cancelled"
	StatusRefunded    PaymentStatus = "refunded"
	StatusExpired     PaymentStatus = "expired"
)

// PaymentType distinguishes transaction purpose
type PaymentType string

const (
	PaymentTierUpgrade  PaymentType = "tier_upgrade"
	PaymentCosmeticPack PaymentType = "cosmetic_purchase"
	PaymentBundle       PaymentType = "bundle_purchase"
)

// PaymentTransaction represents a single payment request
type PaymentTransaction struct {
	TransactionID   string      `json:"transaction_id"`
	UserID          string      `json:"user_id"`
	Type            PaymentType `json:"type"`
	Amount          int         `json:"amount"`           // Amount in XAF (e.g., 5000 = 5,000 XAF)
	Currency        string      `json:"currency"`         // "XAF"
	Description     string      `json:"description"`      // "Tier upgrade to Player" or "Purchase: Shadow Skin"
	CustomerEmail   string      `json:"customer_email"`
	CustomerPhone   string      `json:"customer_phone"`
	ItemID          string      `json:"item_id"`          // Tier ID (1-5) or Cosmetic ID
	ItemName        string      `json:"item_name"`        // "Player Tier" or cosmetic name
	ItemQuantity    int         `json:"item_quantity"`    // Usually 1
	Metadata        map[string]interface{} `json:"metadata"`
	RedirectURL     string      `json:"redirect_url"`     // Client redirect after payment
	Status          PaymentStatus `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	InitiatedAt     *time.Time  `json:"initiated_at,omitempty"`
	CompletedAt     *time.Time  `json:"completed_at,omitempty"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`
}

// PaymentVerification returned by gateway after verification
type PaymentVerification struct {
	TransactionID    string        `json:"transaction_id"`
	Status           PaymentStatus `json:"status"`
	Amount           int           `json:"amount"`
	Currency         string        `json:"currency"`
	PaidAt           *time.Time    `json:"paid_at,omitempty"`
	PaymentMethod    string        `json:"payment_method"`     // "mtn_mobile_money", "card", etc.
	PaymentReference string        `json:"payment_reference"`  // Gateway-provided reference
	FailureReason    string        `json:"failure_reason,omitempty"`
	GatewayResponse  map[string]interface{} `json:"gateway_response,omitempty"`
}

// WebhookPayload received from payment gateway
type WebhookPayload struct {
	Provider        string                 `json:"provider"`          // "flutterwave" or "paytech"
	EventType       string                 `json:"event_type"`        // "charge.completed", "transfer.reversed"
	TransactionID   string                 `json:"transaction_id"`
	Amount          int                    `json:"amount"`
	Status          string                 `json:"status"`
	Timestamp       time.Time              `json:"timestamp"`
	PaymentMethod   string                 `json:"payment_method"`
	RawPayload      map[string]interface{} `json:"raw_payload"` // Full gateway response
}

// RefundRequest specifies refund details
type RefundRequest struct {
	TransactionID  string `json:"transaction_id"`
	Amount         int    `json:"amount"`              // Can be partial (leave empty for full)
	Reason         string `json:"reason"`              // "user_request", "duplicate", "fraud"
	AdminUserID    string `json:"admin_user_id"`       // Who authorized the refund
	AdminNotes     string `json:"admin_notes"`
}

// RefundResponse returned after refund initiation
type RefundResponse struct {
	RefundID       string        `json:"refund_id"`
	TransactionID  string        `json:"transaction_id"`
	Amount         int           `json:"amount"`
	Status         PaymentStatus `json:"status"`          // Usually "refunded" or "processing"
	ProcessedAt    time.Time     `json:"processed_at"`
	GatewayRefundID string       `json:"gateway_refund_id"` // External refund reference
}

// PaymentMethod supported payment types
type PaymentMethod string

const (
	// Mobile Money
	MethodMTNMobileMoney    PaymentMethod = "mtn_mobile_money"
	MethodOrangeMobileMoney PaymentMethod = "orange_mobile_money"
	MethodVodacomMobileMoney PaymentMethod = "vodacom_mobile_money"

	// Cards
	MethodVISA       PaymentMethod = "visa"
	MethodMastercard PaymentMethod = "mastercard"

	// Bank Transfers
	MethodBankTransfer PaymentMethod = "bank_transfer"

	// Wallets
	MethodFlutterwaveWallet PaymentMethod = "flutterwave_wallet"
)

// PaymentConfig contains gateway configuration
type PaymentConfig struct {
	// Flutterwave
	FlutterwavePublicKey  string
	FlutterwaveSecretKey  string
	FlutterwaveBaseURL    string // https://api.flutterwave.com

	// Paytech
	PaytechMerchantID     string
	PaytechAPIKey         string
	PaytechBaseURL        string // https://api.paytech.sn

	// Webhook
	WebhookSecret         string
	WebhookURL            string // https://api.geomobile.app/webhooks/payment

	// Retry Policy
	MaxRetries            int
	RetryDelaySeconds     int

	// Timeouts
	TimeoutSeconds        int

	// Default currency
	Currency              string // "XAF"
}

// PaymentEvent logged for audit trail
type PaymentEvent struct {
	EventID       string                 `json:"event_id"`
	TransactionID string                 `json:"transaction_id"`
	UserID        string                 `json:"user_id"`
	EventType     string                 `json:"event_type"`      // "payment_initiated", "payment_verified", "refund_issued"
	Status        PaymentStatus          `json:"status"`
	Amount        int                    `json:"amount"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
}

// TierUpgradePayment specific to tier purchases
type TierUpgradePayment struct {
	TransactionID  string    `json:"transaction_id"`
	UserID         string    `json:"user_id"`
	CurrentTier    int       `json:"current_tier"`
	UpgradeTier    int       `json:"upgrade_tier"`
	Amount         int       `json:"amount"`
	DurationDays   int       `json:"duration_days"`
	ExpiresAt      time.Time `json:"expires_at"`
	Status         PaymentStatus `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// CosmeticPurchasePayment specific to cosmetic purchases
type CosmeticPurchasePayment struct {
	TransactionID  string    `json:"transaction_id"`
	UserID         string    `json:"user_id"`
	CosmeticID     string    `json:"cosmetic_id"`
	CosmeticName   string    `json:"cosmetic_name"`
	OriginalPrice  int       `json:"original_price"`
	DiscountAmount int       `json:"discount_amount"`  // Based on tier
	FinalPrice     int       `json:"final_price"`
	Status         PaymentStatus `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// TierPricing defines tier pricing
type TierPricing struct {
	TierID        int       `json:"tier_id"`
	TierName      string    `json:"tier_name"`
	PriceMonthly  int       `json:"price_monthly"`    // Monthly price in XAF
	PriceThreeMonth int    `json:"price_three_month"` // Bulk discount
	PriceSixMonth int      `json:"price_six_month"`   // Larger bulk discount
	PriceYearly   int      `json:"price_yearly"`      // Annual discount
	Features      []string `json:"features"`
	Discount      float64  `json:"discount_percent"`  // Cosmetic discount %
}

// PaymentStats for analytics
type PaymentStats struct {
	TotalTransactions     int       `json:"total_transactions"`
	TotalRevenue          int       `json:"total_revenue"`
	CompletedTransactions int       `json:"completed_transactions"`
	FailedTransactions    int       `json:"failed_transactions"`
	AvgTransactionValue   float64   `json:"avg_transaction_value"`
	ConversionRate        float64   `json:"conversion_rate"`     // % of sessions that pay
	TopPaymentMethod      PaymentMethod `json:"top_payment_method"`
	ChurnRate             float64   `json:"churn_rate"`          // % who don't renew tier
}
