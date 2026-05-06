-- Cadastre_IA - Initial Schema

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS uuid-ossp;

-- Users table
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email VARCHAR(255) UNIQUE NOT NULL,
  name VARCHAR(255),
  password_hash VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Parcels table (with geospatial support)
CREATE TABLE IF NOT EXISTS parcels (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_number VARCHAR(100) UNIQUE NOT NULL,
  owner_id UUID REFERENCES users(id),
  geometry GEOMETRY(MultiPolygon, 4326),
  metadata JSONB,
  version INT DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create spatial index
CREATE INDEX IF NOT EXISTS idx_parcels_geometry ON parcels USING GIST(geometry);

-- Sync operations table (for Operational Transform)
CREATE TABLE IF NOT EXISTS sync_operations (
  id BIGSERIAL PRIMARY KEY,
  parcel_id UUID REFERENCES parcels(id),
  client_id UUID,
  operation_type VARCHAR(50),
  operation_data JSONB,
  timestamp BIGINT,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_parcel ON sync_operations(parcel_id);
CREATE INDEX IF NOT EXISTS idx_sync_timestamp ON sync_operations(timestamp);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID REFERENCES users(id),
  token_hash VARCHAR(255),
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Chat messages table
CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID REFERENCES parcels(id),
  user_id UUID REFERENCES users(id),
  content TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_parcel ON messages(parcel_id);
CREATE INDEX IF NOT EXISTS idx_messages_user ON messages(user_id);

-- Files table
CREATE TABLE IF NOT EXISTS files (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID REFERENCES parcels(id),
  file_key VARCHAR(500),
  file_name VARCHAR(255),
  file_size BIGINT,
  checksum VARCHAR(64),
  uploader_id UUID REFERENCES users(id),
  version INT DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_files_parcel ON files(parcel_id);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  action VARCHAR(50),
  entity_type VARCHAR(50),
  entity_id UUID,
  user_id UUID,
  changes JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);

-- Insert test data
INSERT INTO users (email, name) VALUES
  ('user1@cadastreia.local', 'User One'),
  ('user2@cadastreia.local', 'User Two'),
  ('admin@cadastreia.local', 'Administrator')
ON CONFLICT (email) DO NOTHING;

INSERT INTO parcels (parcel_number, owner_id, metadata)
SELECT 'PARCEL-001', id, '{"area": 1000, "location": "Test Area 1"}'::jsonb
FROM users WHERE email = 'user1@cadastreia.local'
ON CONFLICT (parcel_number) DO NOTHING;

INSERT INTO parcels (parcel_number, owner_id, metadata)
SELECT 'PARCEL-002', id, '{"area": 2000, "location": "Test Area 2"}'::jsonb
FROM users WHERE email = 'user2@cadastreia.local'
ON CONFLICT (parcel_number) DO NOTHING;
