// Package quest provides the Lékié Quest VR game engine and quest management system.
// Part of geo-mobile137 — Cadastre_IA + Arcade Game Integration
//
// Module: pkg/quest
// Purpose: Quest mechanics, progression, monetization (freemium 6-tier)
// Version: 0.1.0 (Alpha)
//
// Architecture:
//   Quest System (core) ├─ User Progress Tracking
//                       ├─ XP/Badge Management
//                       ├─ Leaderboard
//                       └─ Reward Pipeline
//
// Freemium Tiers:
//   Tier 0 (Free) .......... 3 quests, 50 XP max/week, cosmetics 1K XAF
//   Tier 1 (Casual) ........ 6 quests, 500 XP/week, cosmetics 2K XAF
//   Tier 2 (Player) ........ 10 quests, 2K XP/week, exclusive skins
//   Tier 3 (Expert) ........ 15 quests, 10K XP/week, beta features
//   Tier 4 (Pro) ........... Unlimited quests, 50K XP/week, API access
//   Tier 5 (PRO ATELIER) ... Premium features + Cadastre_IA integration

package quest

import (
	"time"
)

// TierLevel represents user subscription tier
type TierLevel int

const (
	TierFree TierLevel = iota
	TierCasual
	TierPlayer
	TierExpert
	TierPro
	TierAtelier
)

// QuestType categorizes different quest mechanics
type QuestType string

const (
	QuestTimeline      QuestType = "timeline"      // Reconstruct commune 2021-2025
	QuestPOIHunt       QuestType = "poi_hunt"      // Find specific amenities
	QuestParcelChallenge QuestType = "parcel"      // Quiz on cadastral attributes
	QuestBuildingDetective QuestType = "detective" // Count/identify buildings
	QuestTemporalGlitch QuestType = "glitch"       // Spot differences between years
	QuestOwnerQuiz     QuestType = "owner_quiz"    // Predict owner attributes
)

// QuestStatus tracks quest progression
type QuestStatus string

const (
	QuestStatusAvailable QuestStatus = "available"
	QuestStatusActive    QuestStatus = "active"
	QuestStatusComplete  QuestStatus = "complete"
	QuestStatusAbandoned QuestStatus = "abandoned"
	QuestStatusLocked    QuestStatus = "locked"     // Locked until user reaches tier/XP
)

// Difficulty scaling for different user levels
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyNormal Difficulty = "normal"
	DifficultySoft   Difficulty = "hard"
	DifficultyMaster Difficulty = "master"
)

// Quest defines a single quest instance
type Quest struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Type          QuestType         `json:"type"`
	Difficulty    Difficulty        `json:"difficulty"`
	Region        string            `json:"region"`           // "lekie", "douala", etc.
	AdminUnit     string            `json:"admin_unit"`       // Commune name
	MinTier       TierLevel         `json:"min_tier"`         // Minimum subscription tier
	MinXP         int               `json:"min_xp"`           // XP requirement to unlock
	TargetXP      int               `json:"target_xp"`        // XP earned on completion
	EstimatedTime int               `json:"estimated_time_min"` // In minutes
	Objectives    []Objective       `json:"objectives"`
	Rewards       Rewards           `json:"rewards"`
	Status        QuestStatus       `json:"status"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at"`       // Quest expires if not completed
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Objective is a sub-task within a quest
type Objective struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        string      `json:"type"`           // "click_entity", "find_poi", "answer_quiz", etc.
	Trigger     string      `json:"trigger"`        // Codification code or condition
	Progress    int         `json:"progress"`       // 0-100%
	Required    int         `json:"required"`       // How many items needed
	Completed   bool        `json:"completed"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// Rewards granted on quest completion
type Rewards struct {
	XP            int               `json:"xp"`
	Badges        []Badge           `json:"badges"`
	Cosmetics     []Cosmetic        `json:"cosmetics"`     // Skins, avatars, etc.
	UnlockFeatures []string         `json:"unlock_features"` // e.g., "offline_mode", "satlink_preview"
	MoneyValue    float64           `json:"money_value_fcfa"` // Value of reward in FCFA (for monetization tracking)
}

// Badge represents a user achievement
type Badge struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Icon         string    `json:"icon"`            // URL to badge image
	UnlockedAt   time.Time `json:"unlocked_at"`
	Rarity       string    `json:"rarity"`          // "common", "rare", "epic", "legendary"
}

// Cosmetic represents purchasable cosmetic item (IAP)
type Cosmetic struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`           // "skin", "avatar", "map_theme", "sound_pack"
	Price           int       `json:"price_xaf"`      // Price in XAF (1K-50K range for P1)
	Description     string    `json:"description"`
	ImageURL        string    `json:"image_url"`
	ApplicableTo    []string  `json:"applicable_to"`  // Which UI elements: "player_avatar", "map_terrain", etc.
	ReleasedAt      time.Time `json:"released_at"`
	LimitedEdition  bool      `json:"limited_edition"`
	AvailableUntil  *time.Time `json:"available_until,omitempty"` // For time-limited cosmetics
}

// UserProgress tracks individual user's quest and overall progress
type UserProgress struct {
	UserID          string            `json:"user_id"`
	Tier            TierLevel         `json:"tier"`
	TierExpiresAt   *time.Time        `json:"tier_expires_at,omitempty"` // For trial subscriptions
	TotalXP         int               `json:"total_xp"`
	CurrentLevelXP  int               `json:"current_level_xp"`  // XP for current level (0-1000)
	Level           int               `json:"level"`             // 1-999
	AvailableXPToday int              `json:"available_xp_today"` // Remaining XP quota
	LastXPReset     time.Time         `json:"last_xp_reset"`
	CompletedQuests []string          `json:"completed_quests"`
	ActiveQuests    []string          `json:"active_quests"`
	OwnedCosmetics  []string          `json:"owned_cosmetics"`   // Cosmetic IDs
	Badges          []Badge           `json:"badges"`
	Leaderboard     LeaderboardRank   `json:"leaderboard"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// LeaderboardRank tracks user position in regional leaderboards
type LeaderboardRank struct {
	GlobalRank      int       `json:"global_rank"`
	RegionalRank    int       `json:"regional_rank"`    // e.g., "Lekie top 100"
	CommuneRank     int       `json:"commune_rank"`     // e.g., "Monatele top 10"
	WeeklyRank      int       `json:"weekly_rank"`      // Resets every Monday
	TotalPoints     int       `json:"total_points"`     // XP + bonus multipliers
	LastUpdated     time.Time `json:"last_updated"`
}

// QuestSession tracks active gameplay session
type QuestSession struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	QuestID         string    `json:"quest_id"`
	StartedAt       time.Time `json:"started_at"`
	LastActivityAt  time.Time `json:"last_activity_at"`
	ElapsedSeconds  int       `json:"elapsed_seconds"`
	ObjectiveProgress map[string]int `json:"objective_progress"` // ObjectiveID -> progress %
	PartialXP       int       `json:"partial_xp"`       // Earned so far (before completion)
	Abandoned       bool      `json:"abandoned"`
	AbandonedAt     *time.Time `json:"abandoned_at,omitempty"`
}

// SubscriptionTier defines pricing and features per tier
type SubscriptionTier struct {
	Level              TierLevel    `json:"level"`
	Name               string       `json:"name"`
	MaxQuestsPerWeek   int          `json:"max_quests_per_week"`
	MaxXPPerWeek       int          `json:"max_xp_per_week"`
	CosmeticDiscount   float64      `json:"cosmetic_discount"`    // % discount (0.1 = 10%)
	ExclusiveFeatures  []string     `json:"exclusive_features"`   // e.g., "offline_mode", "advanced_analytics"
	PriceMonthly       int          `json:"price_monthly_fcfa"`   // 0 for Free
	IncludedCosmetics  []string     `json:"included_cosmetics"`   // Free cosmetics with tier
	MaxUsers           *int         `json:"max_users,omitempty"`  // Cap on tier (nil = unlimited)
}

// QuestEvent logs user interactions for analytics
type QuestEvent struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Type        string                 `json:"type"`               // "quest_start", "objective_complete", "quest_complete", "cosmetic_purchase"
	QuestID     string                 `json:"quest_id,omitempty"`
	ObjectiveID string                 `json:"objective_id,omitempty"`
	Data        map[string]interface{} `json:"data"`               // Additional context
	CreatedAt   time.Time              `json:"created_at"`
}

// QuestMetrics for game analytics and balancing
type QuestMetrics struct {
	QuestID                 string  `json:"quest_id"`
	TotalStarts             int     `json:"total_starts"`
	CompletionRate          float64 `json:"completion_rate"`      // 0-100%
	AverageTimeMinutes      float64 `json:"average_time_minutes"`
	AbandonmentRate         float64 `json:"abandonment_rate"`     // 0-100%
	AverageXPEarned         float64 `json:"average_xp_earned"`
	MostDifficultObjective  string  `json:"most_difficult_objective"` // Objective ID
	DifficultyRating        float64 `json:"difficulty_rating"`    // User-reported, 1-5 stars
	LastUpdated             time.Time `json:"last_updated"`
}
