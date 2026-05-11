-- Migration: Payment Gateway Schema (Flutterwave, Paytech)
-- Phase: 2.4 Payment Gateway Integration
-- Date: 2026-05-11

-- ============================================================================
-- PAYMENT TRANSACTIONS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS payment_transactions (
    transaction_id VARCHAR(100) PRIMARY KEY,
    gateway_transaction_id VARCHAR(100),          -- Flutterwave/Paytech transaction ID
    user_id VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,                    -- "tier_upgrade", "cosmetic_purchase"
    amount INT NOT NULL,                          -- Amount in smallest currency unit (XAF)
    currency VARCHAR(3) DEFAULT 'XAF',
    description TEXT,
    customer_email VARCHAR(100),
    customer_phone VARCHAR(20),
    item_id VARCHAR(100),                         -- Tier ID or Cosmetic ID
    item_name VARCHAR(255),
    item_quantity INT DEFAULT 1,

    -- Gateway information
    payment_method VARCHAR(50),                   -- "mtn", "card", "bank_transfer", etc.
    payment_reference VARCHAR(255),               -- Gateway-provided reference

    -- Status tracking
    status VARCHAR(20) DEFAULT 'initiated',       -- "initiated", "processing", "completed", "failed", "refunded"
    failure_reason TEXT,

    -- Metadata (JSON)
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    initiated_at TIMESTAMP,
    completed_at TIMESTAMP,
    expires_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_payment_transactions_user ON payment_transactions(user_id);
CREATE INDEX idx_payment_transactions_status ON payment_transactions(status);
CREATE INDEX idx_payment_transactions_type ON payment_transactions(type);
CREATE INDEX idx_payment_transactions_created ON payment_transactions(created_at DESC);
CREATE INDEX idx_payment_transactions_gateway_id ON payment_transactions(gateway_transaction_id);

-- ============================================================================
-- PAYMENT EVENTS TABLE (Audit Trail)
-- ============================================================================

CREATE TABLE IF NOT EXISTS payment_events (
    event_id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(50),
    event_type VARCHAR(50) NOT NULL,              -- "payment_initiated", "payment_verified", "fulfillment_complete", "refund_issued"
    status VARCHAR(20),                           -- Payment status at event time
    amount INT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (transaction_id) REFERENCES payment_transactions(transaction_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE SET NULL
);

CREATE INDEX idx_payment_events_transaction ON payment_events(transaction_id);
CREATE INDEX idx_payment_events_user ON payment_events(user_id);
CREATE INDEX idx_payment_events_type ON payment_events(event_type);
CREATE INDEX idx_payment_events_created ON payment_events(created_at DESC);

-- ============================================================================
-- REFUNDS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS payment_refunds (
    refund_id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    amount INT NOT NULL,                          -- Refund amount in smallest units
    reason VARCHAR(100),                          -- "user_request", "duplicate", "fraud", "system_error"
    status VARCHAR(20) DEFAULT 'pending',         -- "pending", "processing", "completed", "failed"
    gateway_refund_id VARCHAR(255),               -- External refund reference

    -- Admin info
    requested_by VARCHAR(50),                     -- Admin user ID or system
    approved_by VARCHAR(50),
    notes TEXT,

    -- Timestamps
    requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP,
    completed_at TIMESTAMP,

    FOREIGN KEY (transaction_id) REFERENCES payment_transactions(transaction_id),
    FOREIGN KEY (user_id) REFERENCES user_progress(user_id),
    FOREIGN KEY (requested_by) REFERENCES user_progress(user_id),
    FOREIGN KEY (approved_by) REFERENCES user_progress(user_id)
);

CREATE INDEX idx_payment_refunds_transaction ON payment_refunds(transaction_id);
CREATE INDEX idx_payment_refunds_user ON payment_refunds(user_id);
CREATE INDEX idx_payment_refunds_status ON payment_refunds(status);

-- ============================================================================
-- TIER UPGRADE PAYMENTS (Denormalized for analytics)
-- ============================================================================

CREATE TABLE IF NOT EXISTS tier_upgrade_payments (
    transaction_id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    current_tier INT,
    upgrade_tier INT NOT NULL,
    amount INT NOT NULL,
    duration_days INT,
    expires_at TIMESTAMP,
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,

    FOREIGN KEY (transaction_id) REFERENCES payment_transactions(transaction_id),
    FOREIGN KEY (user_id) REFERENCES user_progress(user_id)
);

CREATE INDEX idx_tier_upgrade_user ON tier_upgrade_payments(user_id);
CREATE INDEX idx_tier_upgrade_tier ON tier_upgrade_payments(upgrade_tier);

-- ============================================================================
-- COSMETIC PURCHASE PAYMENTS (Denormalized for analytics)
-- ============================================================================

CREATE TABLE IF NOT EXISTS cosmetic_purchase_payments (
    transaction_id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    cosmetic_id VARCHAR(100) NOT NULL,
    cosmetic_name VARCHAR(255),
    original_price INT NOT NULL,
    discount_amount INT DEFAULT 0,
    final_price INT NOT NULL,
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,

    FOREIGN KEY (transaction_id) REFERENCES payment_transactions(transaction_id),
    FOREIGN KEY (user_id) REFERENCES user_progress(user_id),
    FOREIGN KEY (cosmetic_id) REFERENCES cosmetics(cosmetic_id)
);

CREATE INDEX idx_cosmetic_purchase_user ON cosmetic_purchase_payments(user_id);
CREATE INDEX idx_cosmetic_purchase_cosmetic ON cosmetic_purchase_payments(cosmetic_id);

-- ============================================================================
-- PAYMENT STATISTICS VIEW
-- ============================================================================

CREATE VIEW IF NOT EXISTS payment_statistics AS
SELECT
    COUNT(*) as total_transactions,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_transactions,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_transactions,
    SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END) as total_revenue,
    AVG(CASE WHEN status = 'completed' THEN amount ELSE NULL END) as avg_transaction_value,
    COUNT(DISTINCT user_id) as unique_payers
FROM payment_transactions;

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE payment_transactions IS 'Core payment transaction records from Flutterwave/Paytech';
COMMENT ON TABLE payment_events IS 'Audit trail of payment lifecycle events';
COMMENT ON TABLE payment_refunds IS 'Refund requests and processing history';
COMMENT ON TABLE tier_upgrade_payments IS 'Denormalized tier upgrade payment data for analytics';
COMMENT ON TABLE cosmetic_purchase_payments IS 'Denormalized cosmetic purchase data for analytics';

COMMENT ON COLUMN payment_transactions.status IS 'initiated, processing, completed, failed, cancelled, refunded';
COMMENT ON COLUMN payment_transactions.type IS 'tier_upgrade, cosmetic_purchase, bundle_purchase';
COMMENT ON COLUMN payment_refunds.status IS 'pending, processing, completed, failed';
