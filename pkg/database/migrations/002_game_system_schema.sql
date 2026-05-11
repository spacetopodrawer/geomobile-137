-- Migration: Create game system schema (user progression, quests, leaderboard)
-- Phase: 2.3 Game Systems Integration
-- Date: 2026-05-11

-- ============================================================================
-- USER PROGRESSION TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_progress (
    user_id VARCHAR(50) PRIMARY KEY,

    -- Subscription tier (0=Free, 1=Casual, 2=Player, 3=Expert, 4=Pro, 5=PRO ATELIER)
    tier_level INT DEFAULT 0,
    tier_expires_at TIMESTAMP,

    -- XP and leveling
    total_xp INT DEFAULT 0,
    current_level INT DEFAULT 1,
    current_level_xp INT DEFAULT 0,

    -- Completion tracking
    completed_quests INT DEFAULT 0,
    total_badges INT DEFAULT 0,
    total_cosmetics INT DEFAULT 0,

    -- Engagement
    last_activity TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Data quality
    data_quality_score FLOAT DEFAULT 1.0
);

CREATE INDEX idx_user_progress_tier ON user_progress(tier_level);
CREATE INDEX idx_user_progress_xp ON user_progress(total_xp DESC);
CREATE INDEX idx_user_progress_last_activity ON user_progress(last_activity DESC);

-- ============================================================================
-- QUESTS TABLE (Definitions/Templates)
-- ============================================================================

CREATE TABLE IF NOT EXISTS quests (
    quest_id VARCHAR(50) PRIMARY KEY,

    -- Quest metadata
    title VARCHAR(255) NOT NULL,
    description TEXT,

    -- Quest type (timeline, poi_hunt, parcel_challenge, building_detective, temporal_glitch, owner_quiz)
    type VARCHAR(50) NOT NULL,

    -- Difficulty
    difficulty VARCHAR(20) NOT NULL, -- "easy", "normal", "hard", "master"

    -- Geographic scope
    region VARCHAR(50) NOT NULL, -- "lekie", "douala", etc.
    admin_unit VARCHAR(100),

    -- Progression gating
    min_tier INT DEFAULT 0,
    min_xp INT DEFAULT 0,
    target_xp INT NOT NULL,

    -- Objectives (stored as JSON array)
    -- Format: [{id, title, trigger, required_count, progress}]
    objectives JSONB NOT NULL,

    -- Rewards
    xp_reward INT NOT NULL,
    badge_reward VARCHAR(255),
    cosmetic_rewards JSONB, -- [{id, name, type}]
    unlock_features JSONB, -- List of features to unlock
    money_value INT DEFAULT 0, -- Value in XAF

    -- Constraints
    is_repeatable BOOLEAN DEFAULT FALSE,
    max_completions INT,
    quest_limit INT DEFAULT 1, -- Max per week by tier

    -- Content
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    enabled BOOLEAN DEFAULT TRUE,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_quests_type ON quests(type);
CREATE INDEX idx_quests_region ON quests(region, admin_unit);
CREATE INDEX idx_quests_difficulty ON quests(difficulty);
CREATE INDEX idx_quests_min_tier ON quests(min_tier);
CREATE INDEX idx_quests_enabled ON quests(enabled);

-- ============================================================================
-- QUEST SESSIONS TABLE (User Quest Instances)
-- ============================================================================

CREATE TABLE IF NOT EXISTS quest_sessions (
    session_id VARCHAR(50) PRIMARY KEY,

    -- Relationships
    user_id VARCHAR(50) NOT NULL,
    quest_id VARCHAR(50) NOT NULL,

    -- Timing
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    elapsed_seconds INT DEFAULT 0,
    completed_at TIMESTAMP,
    abandoned_at TIMESTAMP,

    -- Progress
    objective_progress JSONB DEFAULT '{}', -- {objective_id: {progress, completed}}
    partial_xp INT DEFAULT 0,

    -- Status
    status VARCHAR(20) DEFAULT 'in_progress', -- "in_progress", "completed", "abandoned"

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE,
    FOREIGN KEY (quest_id) REFERENCES quests(quest_id) ON DELETE CASCADE
);

CREATE INDEX idx_quest_sessions_user_id ON quest_sessions(user_id);
CREATE INDEX idx_quest_sessions_quest_id ON quest_sessions(quest_id);
CREATE INDEX idx_quest_sessions_status ON quest_sessions(status);
CREATE INDEX idx_quest_sessions_started_at ON quest_sessions(started_at DESC);
CREATE UNIQUE INDEX idx_quest_sessions_active ON quest_sessions(user_id, quest_id)
    WHERE status = 'in_progress';

-- ============================================================================
-- LEADERBOARD TABLES
-- ============================================================================

-- Global leaderboard (all users)
CREATE TABLE IF NOT EXISTS leaderboard_global (
    rank SERIAL PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    total_xp INT NOT NULL,
    tier_level INT NOT NULL,
    completed_quests INT NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_leaderboard_global_user ON leaderboard_global(user_id);
CREATE INDEX idx_leaderboard_global_xp ON leaderboard_global(total_xp DESC);

-- Regional leaderboard
CREATE TABLE IF NOT EXISTS leaderboard_regional (
    rank SERIAL PRIMARY KEY,
    region VARCHAR(50) NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    total_xp INT NOT NULL,
    completed_quests INT NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_leaderboard_regional_user ON leaderboard_regional(region, user_id);
CREATE INDEX idx_leaderboard_regional_xp ON leaderboard_regional(region, total_xp DESC);

-- Weekly leaderboard
CREATE TABLE IF NOT EXISTS leaderboard_weekly (
    rank SERIAL PRIMARY KEY,
    week_start DATE NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    weekly_xp INT NOT NULL,
    quests_completed INT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_leaderboard_weekly_user ON leaderboard_weekly(week_start, user_id);
CREATE INDEX idx_leaderboard_weekly_xp ON leaderboard_weekly(week_start, weekly_xp DESC);

-- ============================================================================
-- BADGES & COSMETICS TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS badges (
    badge_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon_url VARCHAR(255),
    rarity VARCHAR(20), -- "common", "uncommon", "rare", "epic", "legendary"
    unlock_condition VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS user_badges (
    user_id VARCHAR(50) NOT NULL,
    badge_id VARCHAR(50) NOT NULL,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (user_id, badge_id),
    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE,
    FOREIGN KEY (badge_id) REFERENCES badges(badge_id)
);

CREATE INDEX idx_user_badges_user ON user_badges(user_id);

CREATE TABLE IF NOT EXISTS cosmetics (
    cosmetic_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50), -- "avatar", "emote", "border", "title", "effect"
    price_xaf INT,
    applicable_to VARCHAR(255), -- Comma-separated: "avatar", "quest", "leaderboard"
    limited_edition BOOLEAN DEFAULT FALSE,
    available_until TIMESTAMP,
    icon_url VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS user_cosmetics (
    user_id VARCHAR(50) NOT NULL,
    cosmetic_id VARCHAR(50) NOT NULL,
    purchased_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    equipped BOOLEAN DEFAULT FALSE,

    PRIMARY KEY (user_id, cosmetic_id),
    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE,
    FOREIGN KEY (cosmetic_id) REFERENCES cosmetics(cosmetic_id)
);

CREATE INDEX idx_user_cosmetics_user ON user_cosmetics(user_id);
CREATE INDEX idx_user_cosmetics_equipped ON user_cosmetics(user_id) WHERE equipped = TRUE;

-- ============================================================================
-- OCR EVENT TRACKING TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS ocr_events (
    event_id SERIAL PRIMARY KEY,

    -- Source
    source_file VARCHAR(255) NOT NULL,
    source_type VARCHAR(50), -- "dwg", "scan", "photo", "pdf", "field_form"

    -- Entity linking
    entity_id VARCHAR(50),
    cadastre_code VARCHAR(100),

    -- OCR content
    extracted_text TEXT,
    ocr_confidence FLOAT,

    -- Parsing
    parsed_attributes JSONB DEFAULT '{}',
    parsing_confidence FLOAT,

    -- Geometry matching
    geometry_match_score FLOAT,
    match_method VARCHAR(50), -- "centroid", "bbox_overlap", "manual"

    -- Codification
    codified_attributes JSONB DEFAULT '{}',
    data_quality_score FLOAT,

    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- "pending", "processed", "approved", "rejected"
    admin_notes TEXT,

    -- User tracking
    processed_by VARCHAR(50),
    approved_by VARCHAR(50),

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    approved_at TIMESTAMP,

    FOREIGN KEY (entity_id) REFERENCES cadastral_entities(id) ON DELETE SET NULL
);

CREATE INDEX idx_ocr_events_status ON ocr_events(status);
CREATE INDEX idx_ocr_events_entity_id ON ocr_events(entity_id);
CREATE INDEX idx_ocr_events_created_at ON ocr_events(created_at DESC);
CREATE INDEX idx_ocr_events_cadastre_code ON ocr_events(cadastre_code);

-- ============================================================================
-- QUEST COMPLETION HISTORY (Audit Trail)
-- ============================================================================

CREATE TABLE IF NOT EXISTS quest_completion_history (
    completion_id SERIAL PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    quest_id VARCHAR(50) NOT NULL,
    completion_time INT, -- seconds to complete
    xp_earned INT,
    badges_earned JSONB DEFAULT '[]',
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_progress(user_id) ON DELETE CASCADE,
    FOREIGN KEY (quest_id) REFERENCES quests(quest_id) ON DELETE CASCADE
);

CREATE INDEX idx_quest_completion_user ON quest_completion_history(user_id);
CREATE INDEX idx_quest_completion_quest ON quest_completion_history(quest_id);
CREATE INDEX idx_quest_completion_completed_at ON quest_completion_history(completed_at DESC);

-- ============================================================================
-- MATERIALIZED VIEW FOR LEADERBOARD PERFORMANCE
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS leaderboard_snapshot AS
SELECT
    ROW_NUMBER() OVER (ORDER BY total_xp DESC) as global_rank,
    user_id,
    total_xp,
    tier_level,
    completed_quests,
    last_activity
FROM user_progress
WHERE tier_expires_at IS NULL OR tier_expires_at > CURRENT_TIMESTAMP
ORDER BY total_xp DESC;

CREATE UNIQUE INDEX idx_leaderboard_snapshot_user ON leaderboard_snapshot(user_id);

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE user_progress IS 'Core user progression data (XP, tier, badges)';
COMMENT ON TABLE quests IS 'Quest template/definition data';
COMMENT ON TABLE quest_sessions IS 'Active and completed quest instances';
COMMENT ON TABLE leaderboard_global IS 'Global rankings snapshot';
COMMENT ON TABLE badges IS 'Badge definitions and rarity';
COMMENT ON TABLE cosmetics IS 'In-app cosmetic items for purchase/unlock';
COMMENT ON TABLE ocr_events IS 'OCR processing audit trail and status';
COMMENT ON TABLE quest_completion_history IS 'Historical record of quest completions';

COMMENT ON COLUMN user_progress.tier_level IS '0=Free, 1=Casual, 2=Player, 3=Expert, 4=Pro, 5=PRO ATELIER';
COMMENT ON COLUMN quests.type IS 'timeline, poi_hunt, parcel_challenge, building_detective, temporal_glitch, owner_quiz';
COMMENT ON COLUMN quest_sessions.status IS 'in_progress, completed, abandoned';
COMMENT ON COLUMN ocr_events.status IS 'pending, processed, approved, rejected';
