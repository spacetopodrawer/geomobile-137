-- Enable PostGIS for geospatial support
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Workspaces table
CREATE TABLE workspaces (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(255) NOT NULL,
  owner_id UUID NOT NULL REFERENCES users(id),
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP
);

CREATE INDEX idx_workspaces_owner_id ON workspaces(owner_id);
CREATE INDEX idx_workspaces_deleted_at ON workspaces(deleted_at);

-- Workspace members
CREATE TABLE workspace_members (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(50) NOT NULL DEFAULT 'member', -- 'owner', 'editor', 'viewer'
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_workspace_id ON workspace_members(workspace_id);
CREATE INDEX idx_workspace_members_user_id ON workspace_members(user_id);

-- Parcels table (geospatial)
CREATE TABLE parcels (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  parcel_number VARCHAR(255) NOT NULL,
  geometry GEOMETRY(MultiPolygon, 4326) NOT NULL,
  owner_id UUID REFERENCES users(id),
  metadata JSONB DEFAULT '{}',
  version INT DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP,
  UNIQUE(workspace_id, parcel_number)
);

CREATE INDEX idx_parcels_workspace_id ON parcels(workspace_id);
CREATE INDEX idx_parcels_owner_id ON parcels(owner_id);
CREATE INDEX idx_parcels_geometry ON parcels USING GIST(geometry);
CREATE INDEX idx_parcels_deleted_at ON parcels(deleted_at);
CREATE INDEX idx_parcels_version ON parcels(version);

-- Sync operations log (Operational Transform)
CREATE TABLE sync_operations (
  id BIGSERIAL PRIMARY KEY,
  parcel_id UUID NOT NULL REFERENCES parcels(id) ON DELETE CASCADE,
  client_id UUID NOT NULL,
  operation_type VARCHAR(50) NOT NULL, -- 'update_geometry', 'update_metadata', 'delete'
  operation_data JSONB NOT NULL,
  timestamp BIGINT NOT NULL, -- microseconds for ordering
  applied_at TIMESTAMP,
  status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'applied', 'conflict'
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_operations_parcel_id ON sync_operations(parcel_id);
CREATE INDEX idx_sync_operations_client_id ON sync_operations(client_id);
CREATE INDEX idx_sync_operations_timestamp ON sync_operations(parcel_id, timestamp);
CREATE INDEX idx_sync_operations_status ON sync_operations(status);

-- Conflicts table
CREATE TABLE conflicts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID NOT NULL REFERENCES parcels(id) ON DELETE CASCADE,
  client1_id UUID,
  client2_id UUID,
  operation1 JSONB,
  operation2 JSONB,
  resolution VARCHAR(50), -- 'merge', 'client1_wins', 'client2_wins', 'manual'
  resolved_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conflicts_parcel_id ON conflicts(parcel_id);

-- CAD Files table
CREATE TABLE cad_files (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID NOT NULL REFERENCES parcels(id) ON DELETE CASCADE,
  uploader_id UUID NOT NULL REFERENCES users(id),
  file_format VARCHAR(50) NOT NULL, -- 'dwg', 'dxf', 'ifc', 'geojson', 'shp'
  file_path VARCHAR(1000) NOT NULL,
  original_filename VARCHAR(500) NOT NULL,
  file_size BIGINT,
  converted_geometry GEOMETRY(MultiPolygon, 4326),
  conversion_status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'converting', 'completed', 'failed'
  conversion_error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cad_files_parcel_id ON cad_files(parcel_id);
CREATE INDEX idx_cad_files_uploader_id ON cad_files(uploader_id);
CREATE INDEX idx_cad_files_status ON cad_files(conversion_status);

-- File versions table
CREATE TABLE file_versions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID NOT NULL REFERENCES parcels(id) ON DELETE CASCADE,
  uploader_id UUID NOT NULL REFERENCES users(id),
  file_name VARCHAR(500) NOT NULL,
  file_size BIGINT NOT NULL,
  file_path VARCHAR(1000) NOT NULL,
  checksum VARCHAR(64) NOT NULL, -- MD5 hash
  version_number INT DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP
);

CREATE INDEX idx_file_versions_parcel_id ON file_versions(parcel_id);
CREATE INDEX idx_file_versions_checksum ON file_versions(checksum);
CREATE INDEX idx_file_versions_deleted_at ON file_versions(deleted_at);

-- Messages table
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id),
  content TEXT NOT NULL,
  thread_id UUID, -- For threaded messages
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  edited_at TIMESTAMP,
  deleted_at TIMESTAMP
);

CREATE INDEX idx_messages_workspace_id ON messages(workspace_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_thread_id ON messages(thread_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

-- 3D Assets library
CREATE TABLE assets_3d (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  pack_id UUID,
  name VARCHAR(255) NOT NULL,
  category VARCHAR(100),
  model_url VARCHAR(1000) NOT NULL,
  model_format VARCHAR(50), -- 'fbx', 'gltf', 'obj', 'usdz'
  scale FLOAT DEFAULT 1.0,
  preview_icon VARCHAR(1000),
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assets_3d_pack_id ON assets_3d(pack_id);
CREATE INDEX idx_assets_3d_category ON assets_3d(category);

-- Asset placements on parcels
CREATE TABLE asset_placements (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID NOT NULL REFERENCES parcels(id) ON DELETE CASCADE,
  asset_id UUID NOT NULL REFERENCES assets_3d(id),
  position GEOMETRY(Point, 4326) NOT NULL, -- GeoJSON Point
  rotation FLOAT DEFAULT 0, -- degrees 0-360
  scale FLOAT DEFAULT 1.0, -- 0.1 to 10x
  placed_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_asset_placements_parcel_id ON asset_placements(parcel_id);
CREATE INDEX idx_asset_placements_asset_id ON asset_placements(asset_id);
CREATE INDEX idx_asset_placements_position ON asset_placements USING GIST(position);

-- Animation sequences
CREATE TABLE animation_sequences (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  parcel_id UUID REFERENCES parcels(id) ON DELETE SET NULL,
  asset_placement_id UUID REFERENCES asset_placements(id) ON DELETE SET NULL,
  type VARCHAR(50) NOT NULL, -- 'keyframe', 'timeseries', 'property'
  name VARCHAR(255) NOT NULL,
  duration FLOAT NOT NULL, -- seconds
  keyframes JSONB, -- Array of keyframe data
  timeseries_data JSONB, -- Time series data points
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_animation_sequences_parcel_id ON animation_sequences(parcel_id);
CREATE INDEX idx_animation_sequences_asset_placement_id ON animation_sequences(asset_placement_id);

-- Packages (feature packages, symbol libraries, etc.)
CREATE TABLE packages (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(255) NOT NULL,
  type VARCHAR(50) NOT NULL, -- 'cad', 'assets', 'symbols', 'animation'
  version VARCHAR(20) NOT NULL, -- semantic versioning X.Y.Z
  size BIGINT NOT NULL, -- bytes
  description TEXT,
  content_path VARCHAR(1000) NOT NULL,
  dependencies TEXT[], -- Array of package IDs
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_packages_type ON packages(type);
CREATE INDEX idx_packages_version ON packages(version);

-- User sessions
CREATE TABLE user_sessions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  workspace_id UUID REFERENCES workspaces(id),
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  last_activity TIMESTAMP NOT NULL DEFAULT NOW(),
  client_info JSONB, -- browser, OS, device type
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_workspace_id ON user_sessions(workspace_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
