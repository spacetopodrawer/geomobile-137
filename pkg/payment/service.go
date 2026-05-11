package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cadastre_ia/pkg/repository"
)

// PaymentService manages payment processing and tier upgrades
type PaymentService struct {
	gateway         PaymentGateway
	db              *sql.DB
	progressRepo    *repository.UserProgressRepository
	questRepo       *repository.QuestRepository
	logger          *log.Logger
	webhookSecret   string
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	gateway PaymentGateway,
	db *sql.DB,
	progressRepo *repository.UserProgressRepository,
	questRepo *repository.QuestRepository,
	webhookSecret string,
	logger *log.Logger,
) *PaymentService {
	if logger == nil {
		logger = log.New(log.Writer(), "[PaymentService] ", log.LstdFlags)
	}

	return &PaymentService{
		gateway:       gateway,
		db:            db,
		progressRepo:  progressRepo,
		questRepo:     questRepo,
		logger:        logger,
		webhookSecret: webhookSecret,
	}
}

// InitiateTierUpgrade starts the payment flow for tier upgrade
func (ps *PaymentService) InitiateTierUpgrade(
	ctx context.Context,
	userID string,
	targetTier int,
	durationDays int,
	email string,
	phone string,
	redirectURL string,
) (*TierUpgradePayment, string, error) {
	// Validate tier
	if targetTier < 1 || targetTier > 5 {
		return nil, "", fmt.Errorf("invalid tier: %d", targetTier)
	}

	// Get tier pricing
	tierPrice := getTierPrice(targetTier, durationDays)
	if tierPrice == 0 {
		return nil, "", fmt.Errorf("tier pricing not available")
	}

	// Get current user progress
	userProgress, err := ps.progressRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user progress: %w", err)
	}

	// Create transaction record
	txID := fmt.Sprintf("tier_%s_%d_%d", userID, targetTier, time.Now().UnixNano())
	expiresAt := time.Now().AddDate(0, 0, durationDays)

	transaction := &PaymentTransaction{
		TransactionID: txID,
		UserID:        userID,
		Type:          PaymentTierUpgrade,
		Amount:        tierPrice,
		Currency:      "XAF",
		Description:   fmt.Sprintf("Upgrade to tier %d for %d days", targetTier, durationDays),
		CustomerEmail: email,
		CustomerPhone: phone,
		ItemID:        fmt.Sprintf("tier_%d", targetTier),
		ItemName:      getTierName(targetTier),
		ItemQuantity:  1,
		RedirectURL:   redirectURL,
		Status:        StatusInitiated,
		CreatedAt:     time.Now(),
	}

	transaction.Metadata = map[string]interface{}{
		"current_tier":   userProgress.TierLevel,
		"duration_days":  durationDays,
		"expires_at":     expiresAt,
	}

	// Call payment gateway to get payment link
	paymentLink, gatewayTxID, err := ps.gateway.InitializePayment(transaction)
	if err != nil {
		ps.logger.Printf("Error initializing payment: %v\n", err)
		return nil, "", fmt.Errorf("payment gateway error: %w", err)
	}

	// Store transaction in database
	err = ps.storePaymentTransaction(ctx, transaction, gatewayTxID)
	if err != nil {
		ps.logger.Printf("Error storing transaction: %v\n", err)
		return nil, "", fmt.Errorf("failed to store transaction: %w", err)
	}

	// Create upgrade payment record
	upgradePayment := &TierUpgradePayment{
		TransactionID: txID,
		UserID:        userID,
		CurrentTier:   userProgress.TierLevel,
		UpgradeTier:   targetTier,
		Amount:        tierPrice,
		DurationDays:  durationDays,
		ExpiresAt:     expiresAt,
		Status:        StatusInitiated,
		CreatedAt:     time.Now(),
	}

	ps.logger.Printf("Tier upgrade initiated: %s -> tier %d (price: %d XAF)\n", userID, targetTier, tierPrice)

	return upgradePayment, paymentLink, nil
}

// InitiateCosmeticPurchase starts the payment flow for cosmetic purchase
func (ps *PaymentService) InitiateCosmeticPurchase(
	ctx context.Context,
	userID string,
	cosmeticID string,
	cosmeticName string,
	originalPrice int,
	email string,
	phone string,
	redirectURL string,
) (*CosmeticPurchasePayment, string, error) {
	// Get user progress for tier discount
	userProgress, err := ps.progressRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user progress: %w", err)
	}

	// Calculate discounted price
	discountRate := getTierCosmeticDiscount(userProgress.TierLevel)
	discountAmount := int(float64(originalPrice) * discountRate)
	finalPrice := originalPrice - discountAmount

	// Create transaction record
	txID := fmt.Sprintf("cos_%s_%s_%d", userID, cosmeticID, time.Now().UnixNano())

	transaction := &PaymentTransaction{
		TransactionID: txID,
		UserID:        userID,
		Type:          PaymentCosmeticPack,
		Amount:        finalPrice,
		Currency:      "XAF",
		Description:   fmt.Sprintf("Purchase cosmetic: %s", cosmeticName),
		CustomerEmail: email,
		CustomerPhone: phone,
		ItemID:        cosmeticID,
		ItemName:      cosmeticName,
		ItemQuantity:  1,
		RedirectURL:   redirectURL,
		Status:        StatusInitiated,
		CreatedAt:     time.Now(),
	}

	transaction.Metadata = map[string]interface{}{
		"original_price": originalPrice,
		"discount_rate":  discountRate,
		"user_tier":      userProgress.TierLevel,
	}

	// Call payment gateway
	paymentLink, gatewayTxID, err := ps.gateway.InitializePayment(transaction)
	if err != nil {
		ps.logger.Printf("Error initializing cosmetic payment: %v\n", err)
		return nil, "", fmt.Errorf("payment gateway error: %w", err)
	}

	// Store transaction in database
	err = ps.storePaymentTransaction(ctx, transaction, gatewayTxID)
	if err != nil {
		ps.logger.Printf("Error storing cosmetic transaction: %v\n", err)
		return nil, "", fmt.Errorf("failed to store transaction: %w", err)
	}

	// Create cosmetic purchase record
	cosmeticPayment := &CosmeticPurchasePayment{
		TransactionID:  txID,
		UserID:         userID,
		CosmeticID:     cosmeticID,
		CosmeticName:   cosmeticName,
		OriginalPrice:  originalPrice,
		DiscountAmount: discountAmount,
		FinalPrice:     finalPrice,
		Status:         StatusInitiated,
		CreatedAt:      time.Now(),
	}

	ps.logger.Printf("Cosmetic purchase initiated: %s -> %s (price: %d XAF, discount: %d XAF)\n",
		userID, cosmeticID, finalPrice, discountAmount)

	return cosmeticPayment, paymentLink, nil
}

// VerifyPayment checks payment status and fulfills orders
func (ps *PaymentService) VerifyPayment(ctx context.Context, transactionID string) (*PaymentVerification, error) {
	// Get verification from gateway
	verification, err := ps.gateway.VerifyPayment(transactionID)
	if err != nil {
		ps.logger.Printf("Error verifying payment: %v\n", err)
		return nil, err
	}

	// Update transaction status in database
	err = ps.updateTransactionStatus(ctx, transactionID, verification.Status)
	if err != nil {
		ps.logger.Printf("Error updating transaction status: %v\n", err)
	}

	// If payment is successful, fulfill the order
	if verification.Status == StatusCompleted {
		// Get transaction details
		tx, err := ps.getPaymentTransaction(ctx, transactionID)
		if err != nil {
			ps.logger.Printf("Error retrieving transaction: %v\n", err)
			return verification, nil // Still return verification, just log error
		}

		// Fulfill based on type
		switch tx.Type {
		case PaymentTierUpgrade:
			err = ps.fulfillTierUpgrade(ctx, tx)
		case PaymentCosmeticPack:
			err = ps.fulfillCosmeticPurchase(ctx, tx)
		}

		if err != nil {
			ps.logger.Printf("Error fulfilling order: %v\n", err)
			// Log event but don't fail verification
			ps.logPaymentEvent(ctx, transactionID, tx.UserID, "fulfillment_error", StatusCompleted, tx.Amount)
		} else {
			ps.logPaymentEvent(ctx, transactionID, tx.UserID, "payment_fulfilled", StatusCompleted, tx.Amount)
		}
	}

	return verification, nil
}

// HandleWebhook processes payment confirmations from gateway
func (ps *PaymentService) HandleWebhook(ctx context.Context, payload *WebhookPayload) error {
	ps.logger.Printf("Webhook received: %s - %s (%s)\n", payload.Provider, payload.EventType, payload.TransactionID)

	// Only process successful charges
	if payload.EventType != "charge.completed" && payload.Status != "successful" {
		ps.logger.Printf("Ignoring webhook event: %s\n", payload.EventType)
		return nil
	}

	// Verify payment
	_, err := ps.VerifyPayment(ctx, payload.TransactionID)
	if err != nil {
		ps.logger.Printf("Error verifying webhook payment: %v\n", err)
		return err
	}

	// Log webhook event
	ps.logPaymentEvent(ctx, payload.TransactionID, "", "webhook_processed", StatusCompleted, payload.Amount)

	return nil
}

// Helper methods

func (ps *PaymentService) fulfillTierUpgrade(ctx context.Context, tx *PaymentTransaction) error {
	// Extract metadata
	targetTier := 0
	durationDays := 0
	expiresAt := time.Time{}

	if meta, ok := tx.Metadata["duration_days"]; ok {
		if days, ok := meta.(int); ok {
			durationDays = days
		}
	}

	if meta, ok := tx.Metadata["expires_at"]; ok {
		if expiry, ok := meta.(time.Time); ok {
			expiresAt = expiry
		}
	}

	// Parse tier from ItemID
	_, _ = fmt.Sscanf(tx.ItemID, "tier_%d", &targetTier)

	// Update user tier
	err := ps.progressRepo.UpdateTier(ctx, tx.UserID, targetTier, durationDays)
	if err != nil {
		return fmt.Errorf("failed to update tier: %w", err)
	}

	ps.logger.Printf("Tier upgraded: %s -> tier %d (expires: %s)\n", tx.UserID, targetTier, expiresAt.Format("2006-01-02"))

	return nil
}

func (ps *PaymentService) fulfillCosmeticPurchase(ctx context.Context, tx *PaymentTransaction) error {
	// Add cosmetic to user's owned cosmetics
	query := `
		INSERT INTO user_cosmetics (user_id, cosmetic_id, purchased_at)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`

	_, err := ps.db.ExecContext(ctx, query, tx.UserID, tx.ItemID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add cosmetic: %w", err)
	}

	ps.logger.Printf("Cosmetic purchased: %s -> %s\n", tx.UserID, tx.ItemID)

	return nil
}

func (ps *PaymentService) storePaymentTransaction(ctx context.Context, tx *PaymentTransaction, gatewayTxID string) error {
	query := `
		INSERT INTO payment_transactions
		(transaction_id, gateway_transaction_id, user_id, type, amount, currency, description,
		 item_id, item_name, status, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	metadataJSON, _ := json.Marshal(tx.Metadata)

	_, err := ps.db.ExecContext(ctx, query,
		tx.TransactionID, gatewayTxID, tx.UserID, string(tx.Type), tx.Amount, tx.Currency,
		tx.Description, tx.ItemID, tx.ItemName, string(tx.Status), metadataJSON, tx.CreatedAt)

	return err
}

func (ps *PaymentService) updateTransactionStatus(ctx context.Context, transactionID string, status PaymentStatus) error {
	query := `
		UPDATE payment_transactions
		SET status = ?, updated_at = ?
		WHERE transaction_id = ?
	`

	_, err := ps.db.ExecContext(ctx, query, string(status), time.Now(), transactionID)
	return err
}

func (ps *PaymentService) getPaymentTransaction(ctx context.Context, transactionID string) (*PaymentTransaction, error) {
	query := `
		SELECT transaction_id, user_id, type, amount, item_id, metadata, created_at
		FROM payment_transactions
		WHERE transaction_id = ?
	`

	var tx PaymentTransaction
	var txType string
	var metadataJSON []byte

	err := ps.db.QueryRowContext(ctx, query, transactionID).Scan(
		&tx.TransactionID, &tx.UserID, &txType, &tx.Amount, &tx.ItemID, &metadataJSON, &tx.CreatedAt)

	if err != nil {
		return nil, err
	}

	tx.Type = PaymentType(txType)
	json.Unmarshal(metadataJSON, &tx.Metadata)

	return &tx, nil
}

func (ps *PaymentService) logPaymentEvent(ctx context.Context, transactionID, userID, eventType string, status PaymentStatus, amount int) error {
	query := `
		INSERT INTO payment_events (transaction_id, user_id, event_type, status, amount, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := ps.db.ExecContext(ctx, query, transactionID, userID, eventType, string(status), amount, time.Now())
	return err
}

// Pricing helper functions

func getTierPrice(tier, durationDays int) int {
	// Base monthly prices in XAF
	monthlyPrices := map[int]int{
		1: 2000,   // Casual
		2: 5000,   // Player
		3: 15000,  // Expert
		4: 50000,  // Pro
		5: 150000, // PRO ATELIER
	}

	monthlyPrice, ok := monthlyPrices[tier]
	if !ok {
		return 0
	}

	// Calculate based on duration (discount for longer periods)
	months := durationDays / 30
	if durationDays%30 > 0 {
		months++ // Round up
	}

	basePrice := monthlyPrice * months

	// Apply bulk discounts
	if months >= 12 {
		basePrice = int(float64(basePrice) * 0.8) // 20% discount for 1+ year
	} else if months >= 6 {
		basePrice = int(float64(basePrice) * 0.85) // 15% discount for 6 months
	} else if months >= 3 {
		basePrice = int(float64(basePrice) * 0.9) // 10% discount for 3 months
	}

	return basePrice
}

func getTierName(tier int) string {
	names := map[int]string{
		1: "Casual",
		2: "Player",
		3: "Expert",
		4: "Pro",
		5: "PRO ATELIER",
	}
	return names[tier]
}

func getTierCosmeticDiscount(tier int) float64 {
	discounts := map[int]float64{
		0: 0.0,    // Free
		1: 0.10,   // 10%
		2: 0.20,   // 20%
		3: 0.30,   // 30%
		4: 0.50,   // 50%
		5: 1.0,    // 100% (free)
	}
	return discounts[tier]
}
