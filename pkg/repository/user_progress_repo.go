// Package repository provides data access layers for game system tables
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// UserProgress represents user game progression data
type UserProgress struct {
	UserID             string     `json:"user_id"`
	TierLevel          int        `json:"tier_level"`
	TierExpiresAt      *time.Time `json:"tier_expires_at,omitempty"`
	TotalXP            int        `json:"total_xp"`
	CurrentLevel       int        `json:"current_level"`
	CurrentLevelXP     int        `json:"current_level_xp"`
	CompletedQuests    int        `json:"completed_quests"`
	TotalBadges        int        `json:"total_badges"`
	TotalCosmetics     int        `json:"total_cosmetics"`
	LastActivity       *time.Time `json:"last_activity,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DataQualityScore   float64    `json:"data_quality_score"`
}

// UserProgressRepository handles user progression database operations
type UserProgressRepository struct {
	db     *sql.DB
	logger *log.Logger
}

// NewUserProgressRepository creates a new repository instance
func NewUserProgressRepository(db *sql.DB, logger *log.Logger) *UserProgressRepository {
	if logger == nil {
		logger = log.New(log.Writer(), "[UserProgressRepo] ", log.LstdFlags)
	}
	return &UserProgressRepository{db: db, logger: logger}
}

// GetOrCreate retrieves or creates user progression
func (r *UserProgressRepository) GetOrCreate(ctx context.Context, userID string) (*UserProgress, error) {
	query := `
		SELECT user_id, tier_level, tier_expires_at, total_xp, current_level, current_level_xp,
		       completed_quests, total_badges, total_cosmetics, last_activity, created_at, updated_at,
		       data_quality_score
		FROM user_progress
		WHERE user_id = $1
	`

	row := r.db.QueryRowContext(ctx, query, userID)
	var progress UserProgress

	err := row.Scan(
		&progress.UserID,
		&progress.TierLevel,
		&progress.TierExpiresAt,
		&progress.TotalXP,
		&progress.CurrentLevel,
		&progress.CurrentLevelXP,
		&progress.CompletedQuests,
		&progress.TotalBadges,
		&progress.TotalCosmetics,
		&progress.LastActivity,
		&progress.CreatedAt,
		&progress.UpdatedAt,
		&progress.DataQualityScore,
	)

	if err == sql.ErrNoRows {
		// Create new user progress
		r.logger.Printf("Creating new user progression for %s\n", userID)
		now := time.Now()
		progress = UserProgress{
			UserID:           userID,
			TierLevel:        0, // Free tier
			TotalXP:          0,
			CurrentLevel:     1,
			CurrentLevelXP:   0,
			CompletedQuests:  0,
			TotalBadges:      0,
			TotalCosmetics:   0,
			CreatedAt:        now,
			UpdatedAt:        now,
			DataQualityScore: 1.0,
		}

		insertQuery := `
			INSERT INTO user_progress (user_id, tier_level, total_xp, current_level, current_level_xp,
			                           completed_quests, total_badges, total_cosmetics, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		if err := r.db.QueryRowContext(ctx, insertQuery, progress.UserID, progress.TierLevel,
			progress.TotalXP, progress.CurrentLevel, progress.CurrentLevelXP,
			progress.CompletedQuests, progress.TotalBadges, progress.TotalCosmetics,
			progress.CreatedAt, progress.UpdatedAt).Scan(); err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to create user progress: %w", err)
		}

		return &progress, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &progress, nil
}

// AddXP adds experience points and handles leveling up
func (r *UserProgressRepository) AddXP(ctx context.Context, userID string, xpAmount int) (*UserProgress, error) {
	progress, err := r.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user progress: %w", err)
	}

	progress.TotalXP += xpAmount
	progress.CurrentLevelXP += xpAmount

	// Check for level up (1000 XP per level)
	const XPPerLevel = 1000
	for progress.CurrentLevelXP >= XPPerLevel {
		progress.CurrentLevel++
		progress.CurrentLevelXP -= XPPerLevel
	}

	// Update database
	query := `
		UPDATE user_progress
		SET total_xp = $1, current_level = $2, current_level_xp = $3, updated_at = $4
		WHERE user_id = $5
	`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, query, progress.TotalXP, progress.CurrentLevel,
		progress.CurrentLevelXP, now, userID); err != nil {
		return nil, fmt.Errorf("failed to update XP: %w", err)
	}

	progress.UpdatedAt = now
	r.logger.Printf("Added %d XP to %s (total: %d, level: %d)\n",
		xpAmount, userID, progress.TotalXP, progress.CurrentLevel)

	return progress, nil
}

// IncrementQuestCompletion increments completed quest count
func (r *UserProgressRepository) IncrementQuestCompletion(ctx context.Context, userID string) error {
	query := `
		UPDATE user_progress
		SET completed_quests = completed_quests + 1, updated_at = $1
		WHERE user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to increment quest count: %w", err)
	}

	r.logger.Printf("Incremented quest count for %s\n", userID)
	return nil
}

// UpdateTier updates user subscription tier
func (r *UserProgressRepository) UpdateTier(ctx context.Context, userID string, newTier int, durationDays int) error {
	expiresAt := time.Now().AddDate(0, 0, durationDays)

	query := `
		UPDATE user_progress
		SET tier_level = $1, tier_expires_at = $2, updated_at = $3
		WHERE user_id = $4
	`

	_, err := r.db.ExecContext(ctx, query, newTier, expiresAt, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update tier: %w", err)
	}

	tierNames := []string{"Free", "Casual", "Player", "Expert", "Pro", "PRO ATELIER"}
	tierName := "Unknown"
	if newTier >= 0 && newTier < len(tierNames) {
		tierName = tierNames[newTier]
	}

	r.logger.Printf("Updated %s to tier %d (%s), expires: %s\n",
		userID, newTier, tierName, expiresAt.Format("2006-01-02"))

	return nil
}

// UpdateLastActivity updates last activity timestamp
func (r *UserProgressRepository) UpdateLastActivity(ctx context.Context, userID string) error {
	query := `UPDATE user_progress SET last_activity = $1, updated_at = $1 WHERE user_id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

// GetLeaderboardPosition returns user's global leaderboard rank
func (r *UserProgressRepository) GetLeaderboardPosition(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*) + 1
		FROM user_progress
		WHERE total_xp > (SELECT total_xp FROM user_progress WHERE user_id = $1)
	`

	var rank int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&rank)
	if err != nil {
		return 0, fmt.Errorf("failed to get rank: %w", err)
	}

	return rank, nil
}

// GetTopUsers returns top N users by XP
func (r *UserProgressRepository) GetTopUsers(ctx context.Context, limit int) ([]UserProgress, error) {
	query := `
		SELECT user_id, tier_level, tier_expires_at, total_xp, current_level, current_level_xp,
		       completed_quests, total_badges, total_cosmetics, last_activity, created_at, updated_at,
		       data_quality_score
		FROM user_progress
		ORDER BY total_xp DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}
	defer rows.Close()

	var users []UserProgress
	for rows.Next() {
		var user UserProgress
		if err := rows.Scan(
			&user.UserID,
			&user.TierLevel,
			&user.TierExpiresAt,
			&user.TotalXP,
			&user.CurrentLevel,
			&user.CurrentLevelXP,
			&user.CompletedQuests,
			&user.TotalBadges,
			&user.TotalCosmetics,
			&user.LastActivity,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.DataQualityScore,
		); err != nil {
			r.logger.Printf("Error scanning user: %v\n", err)
			continue
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// GetStatistics returns aggregate statistics
func (r *UserProgressRepository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_users,
			COUNT(CASE WHEN tier_level > 0 THEN 1 END) as premium_users,
			AVG(total_xp) as avg_xp,
			MAX(total_xp) as max_xp,
			AVG(completed_quests) as avg_quests_completed
		FROM user_progress
	`

	var totalUsers, premiumUsers int
	var avgXP, maxXP, avgQuests sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query).Scan(&totalUsers, &premiumUsers, &avgXP, &maxXP, &avgQuests)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	stats := map[string]interface{}{
		"total_users":             totalUsers,
		"premium_users":           premiumUsers,
		"average_xp":              avgXP.Float64,
		"max_xp":                  maxXP.Float64,
		"average_quests_completed": avgQuests.Float64,
	}

	return stats, nil
}
