// Package repository provides quest data access
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Quest represents a quest template
type Quest struct {
	QuestID         string          `json:"quest_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Type            string          `json:"type"`
	Difficulty      string          `json:"difficulty"`
	Region          string          `json:"region"`
	AdminUnit       string          `json:"admin_unit"`
	MinTier         int             `json:"min_tier"`
	MinXP           int             `json:"min_xp"`
	TargetXP        int             `json:"target_xp"`
	Objectives      json.RawMessage `json:"objectives"`
	XPReward        int             `json:"xp_reward"`
	BadgeReward     string          `json:"badge_reward,omitempty"`
	IsRepeatable    bool            `json:"is_repeatable"`
	MaxCompletions  *int            `json:"max_completions,omitempty"`
	QuestLimit      int             `json:"quest_limit"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// QuestSession represents an active quest instance
type QuestSession struct {
	SessionID        string          `json:"session_id"`
	UserID           string          `json:"user_id"`
	QuestID          string          `json:"quest_id"`
	StartedAt        time.Time       `json:"started_at"`
	ElapsedSeconds   int             `json:"elapsed_seconds"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	AbandonedAt      *time.Time      `json:"abandoned_at,omitempty"`
	ObjectiveProgress json.RawMessage `json:"objective_progress"`
	PartialXP        int             `json:"partial_xp"`
	Status           string          `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// QuestRepository handles quest database operations
type QuestRepository struct {
	db     *sql.DB
	logger *log.Logger
}

// NewQuestRepository creates a new quest repository
func NewQuestRepository(db *sql.DB, logger *log.Logger) *QuestRepository {
	if logger == nil {
		logger = log.New(log.Writer(), "[QuestRepo] ", log.LstdFlags)
	}
	return &QuestRepository{db: db, logger: logger}
}

// GetQuest retrieves a quest by ID
func (r *QuestRepository) GetQuest(ctx context.Context, questID string) (*Quest, error) {
	query := `
		SELECT quest_id, title, description, type, difficulty, region, admin_unit,
		       min_tier, min_xp, target_xp, objectives, xp_reward, badge_reward,
		       is_repeatable, max_completions, quest_limit, enabled, created_at, updated_at
		FROM quests
		WHERE quest_id = $1 AND (deleted_at IS NULL OR deleted_at > CURRENT_TIMESTAMP)
	`

	var quest Quest
	err := r.db.QueryRowContext(ctx, query, questID).Scan(
		&quest.QuestID, &quest.Title, &quest.Description, &quest.Type, &quest.Difficulty,
		&quest.Region, &quest.AdminUnit, &quest.MinTier, &quest.MinXP, &quest.TargetXP,
		&quest.Objectives, &quest.XPReward, &quest.BadgeReward,
		&quest.IsRepeatable, &quest.MaxCompletions, &quest.QuestLimit, &quest.Enabled,
		&quest.CreatedAt, &quest.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("quest not found: %s", questID)
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &quest, nil
}

// GetAvailableQuests returns quests available for a user
func (r *QuestRepository) GetAvailableQuests(ctx context.Context, userID string, userTier int, userXP int, limit int) ([]Quest, error) {
	query := `
		SELECT quest_id, title, description, type, difficulty, region, admin_unit,
		       min_tier, min_xp, target_xp, objectives, xp_reward, badge_reward,
		       is_repeatable, max_completions, quest_limit, enabled, created_at, updated_at
		FROM quests
		WHERE enabled = TRUE
		  AND (deleted_at IS NULL OR deleted_at > CURRENT_TIMESTAMP)
		  AND min_tier <= $1
		  AND min_xp <= $2
		ORDER BY difficulty ASC, target_xp DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, userTier, userXP, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var quests []Quest
	for rows.Next() {
		var quest Quest
		if err := rows.Scan(
			&quest.QuestID, &quest.Title, &quest.Description, &quest.Type, &quest.Difficulty,
			&quest.Region, &quest.AdminUnit, &quest.MinTier, &quest.MinXP, &quest.TargetXP,
			&quest.Objectives, &quest.XPReward, &quest.BadgeReward,
			&quest.IsRepeatable, &quest.MaxCompletions, &quest.QuestLimit, &quest.Enabled,
			&quest.CreatedAt, &quest.UpdatedAt,
		); err != nil {
			r.logger.Printf("Error scanning quest: %v\n", err)
			continue
		}
		quests = append(quests, quest)
	}

	return quests, rows.Err()
}

// GetQuestsByRegion returns quests for a specific region
func (r *QuestRepository) GetQuestsByRegion(ctx context.Context, region string) ([]Quest, error) {
	query := `
		SELECT quest_id, title, description, type, difficulty, region, admin_unit,
		       min_tier, min_xp, target_xp, objectives, xp_reward, badge_reward,
		       is_repeatable, max_completions, quest_limit, enabled, created_at, updated_at
		FROM quests
		WHERE region = $1 AND enabled = TRUE AND (deleted_at IS NULL OR deleted_at > CURRENT_TIMESTAMP)
		ORDER BY difficulty ASC
	`

	rows, err := r.db.QueryContext(ctx, query, region)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var quests []Quest
	for rows.Next() {
		var quest Quest
		if err := rows.Scan(
			&quest.QuestID, &quest.Title, &quest.Description, &quest.Type, &quest.Difficulty,
			&quest.Region, &quest.AdminUnit, &quest.MinTier, &quest.MinXP, &quest.TargetXP,
			&quest.Objectives, &quest.XPReward, &quest.BadgeReward,
			&quest.IsRepeatable, &quest.MaxCompletions, &quest.QuestLimit, &quest.Enabled,
			&quest.CreatedAt, &quest.UpdatedAt,
		); err != nil {
			continue
		}
		quests = append(quests, quest)
	}

	return quests, rows.Err()
}

// StartSession creates a new quest session
func (r *QuestRepository) StartSession(ctx context.Context, sessionID string, userID string, questID string) (*QuestSession, error) {
	now := time.Now()
	session := &QuestSession{
		SessionID:        sessionID,
		UserID:           userID,
		QuestID:          questID,
		StartedAt:        now,
		ElapsedSeconds:   0,
		ObjectiveProgress: json.RawMessage("{}"),
		PartialXP:        0,
		Status:           "in_progress",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	query := `
		INSERT INTO quest_sessions (session_id, user_id, quest_id, started_at, status, objective_progress, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query, session.SessionID, session.UserID, session.QuestID,
		session.StartedAt, session.Status, session.ObjectiveProgress, session.CreatedAt, session.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	r.logger.Printf("Created quest session %s for user %s, quest %s\n", sessionID, userID, questID)
	return session, nil
}

// GetSession retrieves an active quest session
func (r *QuestRepository) GetSession(ctx context.Context, sessionID string) (*QuestSession, error) {
	query := `
		SELECT session_id, user_id, quest_id, started_at, elapsed_seconds, completed_at,
		       abandoned_at, objective_progress, partial_xp, status, created_at, updated_at
		FROM quest_sessions
		WHERE session_id = $1
	`

	var session QuestSession
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.SessionID, &session.UserID, &session.QuestID, &session.StartedAt,
		&session.ElapsedSeconds, &session.CompletedAt, &session.AbandonedAt,
		&session.ObjectiveProgress, &session.PartialXP, &session.Status,
		&session.CreatedAt, &session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &session, nil
}

// CompleteSession marks a quest session as completed
func (r *QuestRepository) CompleteSession(ctx context.Context, sessionID string, finalXP int) error {
	now := time.Now()
	query := `
		UPDATE quest_sessions
		SET status = $1, completed_at = $2, partial_xp = $3, updated_at = $2
		WHERE session_id = $4
	`

	_, err := r.db.ExecContext(ctx, query, "completed", now, finalXP, sessionID)
	if err != nil {
		return fmt.Errorf("failed to complete session: %w", err)
	}

	r.logger.Printf("Completed quest session %s with %d XP\n", sessionID, finalXP)
	return nil
}

// AbandonSession marks a quest session as abandoned
func (r *QuestRepository) AbandonSession(ctx context.Context, sessionID string) error {
	now := time.Now()
	query := `
		UPDATE quest_sessions
		SET status = $1, abandoned_at = $2, updated_at = $2
		WHERE session_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, "abandoned", now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to abandon session: %w", err)
	}

	r.logger.Printf("Abandoned quest session %s\n", sessionID)
	return nil
}

// GetUserActiveSessions returns all active sessions for a user
func (r *QuestRepository) GetUserActiveSessions(ctx context.Context, userID string) ([]QuestSession, error) {
	query := `
		SELECT session_id, user_id, quest_id, started_at, elapsed_seconds, completed_at,
		       abandoned_at, objective_progress, partial_xp, status, created_at, updated_at
		FROM quest_sessions
		WHERE user_id = $1 AND status = 'in_progress'
		ORDER BY started_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var sessions []QuestSession
	for rows.Next() {
		var session QuestSession
		if err := rows.Scan(
			&session.SessionID, &session.UserID, &session.QuestID, &session.StartedAt,
			&session.ElapsedSeconds, &session.CompletedAt, &session.AbandonedAt,
			&session.ObjectiveProgress, &session.PartialXP, &session.Status,
			&session.CreatedAt, &session.UpdatedAt,
		); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// GetQuestStats returns statistics about quest completion
func (r *QuestRepository) GetQuestStats(ctx context.Context, questID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_starts,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'abandoned' THEN 1 END) as abandoned,
			AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_time_seconds
		FROM quest_sessions
		WHERE quest_id = $1
	`

	var totalStarts, completed, abandoned int
	var avgTime sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, questID).Scan(&totalStarts, &completed, &abandoned, &avgTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	stats := map[string]interface{}{
		"total_starts":   totalStarts,
		"completed":      completed,
		"abandoned":      abandoned,
		"completion_rate": 0.0,
		"avg_time_seconds": 0.0,
	}

	if totalStarts > 0 {
		stats["completion_rate"] = float64(completed) / float64(totalStarts)
	}

	if avgTime.Valid {
		stats["avg_time_seconds"] = avgTime.Float64
	}

	return stats, nil
}
