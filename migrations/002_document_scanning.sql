-- Migration: Add Document Scanning and Facial Recognition Tables
-- Version: 002
-- Date: 2026-05-14

-- Document scans table (stores uploaded documents and OCR results)
CREATE TABLE IF NOT EXISTS document_scans (
  scan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID,
  document_type VARCHAR(50) NOT NULL, -- 'cadastral_deed', 'id_card', 'passport', 'property_photo'
  raw_image_path VARCHAR(500),
  scanned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  extraction_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'success', 'partial', 'failed'

  -- OCR results
  ocr_confidence FLOAT DEFAULT 0.0, -- 0-1 confidence of OCR extraction
  extracted_text TEXT, -- Full OCR output
  extracted_fields JSONB, -- Parsed structured data

  -- User matching results
  match_confidence FLOAT DEFAULT 0.0, -- 0-1 confidence of user identification
  matched_user_id UUID, -- null if no match
  match_method VARCHAR(50), -- 'exact_email', 'fuzzy_name', 'property_location', 'facial_recognition', 'auto_created'

  -- For audit and compliance
  signature BYTEA, -- Hash of extracted data
  verified_at TIMESTAMP,
  verified_by VARCHAR(50), -- 'system_auto', 'user_manual', 'admin'

  FOREIGN KEY (user_id) REFERENCES user_identities(user_id),
  FOREIGN KEY (matched_user_id) REFERENCES user_identities(user_id)
);

CREATE INDEX idx_document_scans_user ON document_scans(user_id);
CREATE INDEX idx_document_scans_matched_user ON document_scans(matched_user_id);
CREATE INDEX idx_document_scans_status ON document_scans(extraction_status);
CREATE INDEX idx_document_scans_type ON document_scans(document_type);

-- Facial recognition embeddings table
CREATE TABLE IF NOT EXISTS face_embeddings (
  embedding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE,
  embedding FLOAT8[] NOT NULL, -- 512-dimensional vector (as array for compatibility)
  captured_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  photo_source VARCHAR(50), -- 'id_card', 'passport', 'selfie', 'document_scan'
  confidence FLOAT DEFAULT 0.0, -- Quality of face detection

  FOREIGN KEY (user_id) REFERENCES user_identities(user_id)
);

CREATE INDEX idx_face_embeddings_user ON face_embeddings(user_id);

-- Property ownership records (extracted from cadastral scans)
CREATE TABLE IF NOT EXISTS property_ownership (
  ownership_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  property_id VARCHAR(100), -- Cadastral parcel ID
  property_address VARCHAR(500),
  location POINT, -- (latitude, longitude)
  owner_name VARCHAR(200),
  ownership_verified BOOLEAN DEFAULT FALSE,
  verified_via_document_id UUID,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (user_id) REFERENCES user_identities(user_id),
  FOREIGN KEY (verified_via_document_id) REFERENCES document_scans(scan_id)
);

CREATE INDEX idx_property_ownership_user ON property_ownership(user_id);
CREATE INDEX idx_property_ownership_property_id ON property_ownership(property_id);
CREATE INDEX idx_property_ownership_location ON property_ownership USING GIST (location);

-- User full names (denormalized for fuzzy matching)
ALTER TABLE user_identities ADD COLUMN IF NOT EXISTS full_name VARCHAR(200);
ALTER TABLE user_identities ADD COLUMN IF NOT EXISTS date_of_birth DATE;
ALTER TABLE user_identities ADD COLUMN IF NOT EXISTS nationality VARCHAR(50);
ALTER TABLE user_identities ADD COLUMN IF NOT EXISTS verification_status VARCHAR(50) DEFAULT 'unverified'; -- 'unverified', 'email_verified', 'document_verified', 'facial_verified'

-- Credential audit (for tracking email/username changes)
ALTER TABLE user_credentials ADD COLUMN IF NOT EXISTS last_changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Commit log
INSERT INTO _migrations (migration_id, applied_at)
VALUES ('002_document_scanning', CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;
