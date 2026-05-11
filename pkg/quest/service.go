package quest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"cadastre_ia/pkg/repository"
)

// QuestService handles quest lifecycle, progression, and monetization
type QuestService struct {
	questRepo    *repository.QuestRepository
	progressRepo *repository.UserProgressRepository
	db           *sql.DB
	cache        CacheProvider // Redis or in-memory cache
	logger       *log.Logger
}

// CacheProvider interface for flexibility (Redis/Memcached/in-memory)
type CacheProvider interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewQuestService creates a new quest service instance
func NewQuestService(db *sql.DB, cache CacheProvider, logger *log.Logger) *QuestService {
	if logger == nil {
		logger = log.New(log.Writer(), "[QuestService] ", log.LstdFlags)
	}
	return &QuestService{
		questRepo:    repository.NewQuestRepository(db, logger),
		progressRepo: repository.NewUserProgressRepository(db, logger),
		db:           db,
		cache:        cache,
		logger:       logger,
	}
}

// === QUEST RETRIEVAL ===

// GetAvailableQuests returns all quests user can play based on their tier/XP
func (qs *QuestService) GetAvailableQuests(
	ctx context.Context,
	userID string,
	progress *UserProgress,
) ([]Quest, error) {
	const questLimit = 20 // Default limit for available quests

	// Use repository to get available quests
	repoQuests, err := qs.questRepo.GetAvailableQuests(ctx, userID, int(progress.Tier), progress.TotalXP, questLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to query available quests: %w", err)
	}

	// Convert repository quests to service quests
	quests := make([]Quest, 0, len(repoQuests))
	for _, rq := range repoQuests {
		q := Quest{
			ID:          rq.QuestID,
			Title:       rq.Title,
			Description: rq.Description,
			Type:        QuestType(rq.Type),
			Difficulty:  Difficulty(rq.Difficulty),
			Region:      rq.Region,
			AdminUnit:   rq.AdminUnit,
			MinTier:     TierLevel(rq.MinTier),
			MinXP:       rq.MinXP,
			TargetXP:    rq.TargetXP,
			Status:      QuestStatusAvailable,
			CreatedAt:   rq.CreatedAt,
			UpdatedAt:   rq.UpdatedAt,
		}

		// Parse objectives from JSON
		if err := json.Unmarshal(rq.Objectives, &q.Objectives); err != nil {
			qs.logger.Printf("Warning: Failed to parse objectives for quest %s: %v\n", rq.QuestID, err)
		}

		quests = append(quests, q)
	}

	return quests, nil
}

// StartQuest initializes a new quest session for user
func (qs *QuestService) StartQuest(
	ctx context.Context,
	userID string,
	questID string,
) (*QuestSession, error) {
	// Validate user tier/XP requirements
	progress, err := qs.GetUserProgress(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Fetch quest
	quest, err := qs.GetQuestByID(ctx, questID)
	if err != nil {
		return nil, err
	}

	// Check tier and XP requirements
	if progress.Tier < quest.MinTier {
		return nil, errors.New("user tier insufficient for this quest")
	}
	if progress.TotalXP < quest.MinXP {
		return nil, errors.New("user XP too low for this quest")
	}

	// Check weekly quota
	weeklyQuests, err := qs.countWeeklyQuests(ctx, userID)
	if err != nil {
		return nil, err
	}

	tierConfig := getTierConfig(progress.Tier)
	if weeklyQuests >= tierConfig.MaxQuestsPerWeek {
		return nil, fmt.Errorf("user exceeded weekly quest limit (%d)", tierConfig.MaxQuestsPerWeek)
	}

	// Create session using repository
	sessionID := generateID("qs")
	repoSession, err := qs.questRepo.StartSession(ctx, sessionID, userID, questID)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Convert repository session to service session
	session := &QuestSession{
		ID:                 repoSession.SessionID,
		UserID:             repoSession.UserID,
		QuestID:            repoSession.QuestID,
		StartedAt:          repoSession.StartedAt,
		LastActivityAt:     repoSession.StartedAt,
		ElapsedSeconds:     0,
		ObjectiveProgress:  make(map[string]int),
		PartialXP:          0,
		Abandoned:          false,
	}

	// Initialize objective progress (all at 0%)
	for _, obj := range quest.Objectives {
		session.ObjectiveProgress[obj.ID] = 0
	}

	// Log event
	_ = qs.logQuestEvent(ctx, userID, "quest_start", questID, nil)

	return session, nil
}

// AbandonQuest marks a quest session as abandoned
func (qs *QuestService) AbandonQuest(
	ctx context.Context,
	sessionID string,
) error {
	// Load session to get user ID for logging
	session, err := qs.GetQuestSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Use repository to abandon session
	err = qs.questRepo.AbandonSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to abandon session: %w", err)
	}

	// Log event
	_ = qs.logQuestEvent(ctx, session.UserID, "quest_abandon", session.QuestID, nil)

	return nil
}

// CompleteObjective marks an objective as complete and awards partial XP
func (qs *QuestService) CompleteObjective(
	ctx context.Context,
	sessionID string,
	objectiveID string,
) error {
	// Load session
	session, err := qs.GetQuestSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Load quest and find objective
	quest, err := qs.GetQuestByID(ctx, session.QuestID)
	if err != nil {
		return err
	}

	var objective *Objective
	for i := range quest.Objectives {
		if quest.Objectives[i].ID == objectiveID {
			objective = &quest.Objectives[i]
			break
		}
	}

	if objective == nil {
		return errors.New("objective not found in quest")
	}

	// Update progress
	session.ObjectiveProgress[objectiveID] = 100
	session.LastActivityAt = time.Now()

	// Award partial XP (proportional to quest difficulty)
	xpPortion := quest.TargetXP / len(quest.Objectives)
	session.PartialXP += xpPortion

	// Store updated session
	err = qs.storeQuestSession(ctx, session)
	if err != nil {
		return err
	}

	// Log event
	_ = qs.logQuestEvent(ctx, session.UserID, "objective_complete", session.QuestID, map[string]interface{}{
		"objective_id": objectiveID,
		"xp_earned":    xpPortion,
	})

	// Check if all objectives complete
	allComplete := true
	for _, obj := range quest.Objectives {
		progress, exists := session.ObjectiveProgress[obj.ID]
		if !exists || progress < 100 {
			allComplete = false
			break
		}
	}

	if allComplete {
		return qs.CompleteQuest(ctx, sessionID)
	}

	return nil
}

// CompleteQuest awards full XP, badges, and updates leaderboard
func (qs *QuestService) CompleteQuest(
	ctx context.Context,
	sessionID string,
) error {
	// Load session
	session, err := qs.GetQuestSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Load quest for rewards
	quest, err := qs.GetQuestByID(ctx, session.QuestID)
	if err != nil {
		return err
	}

	// Update user progress using repository
	progress, err := qs.GetUserProgress(ctx, session.UserID)
	if err != nil {
		return err
	}

	// Award XP
	xpToAward := quest.TargetXP

	// Add XP using repository method
	updatedProgress, err := qs.progressRepo.AddXP(ctx, session.UserID, xpToAward)
	if err != nil {
		return fmt.Errorf("failed to add XP: %w", err)
	}

	// Increment quest completion
	err = qs.progressRepo.IncrementQuestCompletion(ctx, session.UserID)
	if err != nil {
		return fmt.Errorf("failed to increment quest count: %w", err)
	}

	// Update leaderboard (async)
	progress.TotalXP = updatedProgress.TotalXP
	progress.Level = updatedProgress.CurrentLevel
	progress.CurrentLevelXP = updatedProgress.CurrentLevelXP
	qs.updateLeaderboard(ctx, session.UserID, progress)

	// Complete session using repository
	err = qs.questRepo.CompleteSession(ctx, sessionID, xpToAward)
	if err != nil {
		return fmt.Errorf("failed to complete session: %w", err)
	}

	// Invalidate cache
	_ = qs.cache.Delete(ctx, fmt.Sprintf("user_progress:%s", session.UserID))

	// Log event
	qs.logQuestEvent(ctx, session.UserID, "quest_complete", session.QuestID, map[string]interface{}{
		"xp_awarded": xpToAward,
		"level":      updatedProgress.CurrentLevel,
	})

	return nil
}

// === USER PROGRESS ===

// GetUserProgress retrieves user's overall progress (tier, XP, badges, etc.)
func (qs *QuestService) GetUserProgress(
	ctx context.Context,
	userID string,
) (*UserProgress, error) {
	// Try cache first
	cached, _ := qs.cache.Get(ctx, fmt.Sprintf("user_progress:%s", userID))
	if cached != nil {
		if up, ok := cached.(*UserProgress); ok {
			return up, nil
		}
	}

	// Get from repository
	repoProg, err := qs.progressRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user progress: %w", err)
	}

	// Convert repository UserProgress to service UserProgress
	progress := &UserProgress{
		UserID:         repoProg.UserID,
		Tier:           TierLevel(repoProg.TierLevel),
		TierExpiresAt:  repoProg.TierExpiresAt,
		TotalXP:        repoProg.TotalXP,
		CurrentLevelXP: repoProg.CurrentLevelXP,
		Level:          repoProg.CurrentLevel,
		CreatedAt:      repoProg.CreatedAt,
		UpdatedAt:      repoProg.UpdatedAt,
	}

	// Calculate available XP today based on tier
	tierConfig := getTierConfig(progress.Tier)
	progress.AvailableXPToday = tierConfig.MaxXPPerWeek

	// Set last XP reset to now (can be enhanced with actual tracking)
	progress.LastXPReset = time.Now()

	// Initialize empty slices
	progress.CompletedQuests = []string{}
	progress.ActiveQuests = []string{}
	progress.OwnedCosmetics = []string{}
	progress.Badges = []Badge{}

	// Cache for 5 minutes
	_ = qs.cache.Set(ctx, fmt.Sprintf("user_progress:%s", userID), progress, 5*time.Minute)

	return progress, nil
}

// UpgradeTier upgrades user to a higher subscription tier (called on payment success)
func (qs *QuestService) UpgradeTier(
	ctx context.Context,
	userID string,
	newTier TierLevel,
	durationDays int,
) error {
	// Use repository to update tier
	err := qs.progressRepo.UpdateTier(ctx, userID, int(newTier), durationDays)
	if err != nil {
		return fmt.Errorf("failed to update tier: %w", err)
	}

	// Invalidate cache
	_ = qs.cache.Delete(ctx, fmt.Sprintf("user_progress:%s", userID))

	expiresAt := time.Now().AddDate(0, 0, durationDays)

	// Log event
	_ = qs.logQuestEvent(ctx, userID, "tier_upgrade", "", map[string]interface{}{
		"new_tier":    newTier,
		"expires_at":  expiresAt,
	})

	return nil
}

// === COSMETICS & IAP ===

// PurchaseCosmetic handles cosmetic purchase (called after payment)
func (qs *QuestService) PurchaseCosmetic(
	ctx context.Context,
	userID string,
	cosmeticID string,
) error {
	// Verify cosmetic exists
	cosmetic, err := qs.GetCosmeticByID(ctx, cosmeticID)
	if err != nil {
		return fmt.Errorf("cosmetic not found: %w", err)
	}

	// Get user progress to apply tier-based discount
	progress, err := qs.GetUserProgress(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user progress: %w", err)
	}

	tierConfig := getTierConfig(progress.Tier)
	finalPrice := cosmetic.Price - int(float64(cosmetic.Price)*tierConfig.CosmeticDiscount)

	// Store purchase in database
	query := `
		INSERT INTO user_cosmetics (user_id, cosmetic_id, purchased_at)
		VALUES (?, ?, ?)
	`

	_, err = qs.db.ExecContext(ctx, query, userID, cosmeticID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record cosmetic purchase: %w", err)
	}

	// Invalidate cache
	_ = qs.cache.Delete(ctx, fmt.Sprintf("user_progress:%s", userID))

	// Log event
	_ = qs.logQuestEvent(ctx, userID, "cosmetic_purchase", "", map[string]interface{}{
		"cosmetic_id": cosmeticID,
		"price_paid":  finalPrice,
	})

	qs.logger.Printf("User %s purchased cosmetic %s for %d XAF\n", userID, cosmeticID, finalPrice)

	return nil
}

// === LEADERBOARD ===

// GetLeaderboard returns top players (global, regional, or weekly)
func (qs *QuestService) GetLeaderboard(
	ctx context.Context,
	region string, // "" = global, "lekie" = regional, "monatele" = commune
	scope string,   // "global", "weekly"
	limit int,
) ([]LeaderboardEntry, error) {
	entries := []LeaderboardEntry{}

	// Query varies by scope
	var query string
	if scope == "weekly" {
		query = `
			SELECT user_id, total_points, global_rank, weekly_rank
			FROM leaderboard_weekly
			WHERE region = ? OR region = ''
			ORDER BY total_points DESC
			LIMIT ?
		`
	} else {
		query = `
			SELECT user_id, total_points, global_rank, regional_rank
			FROM leaderboard_global
			WHERE region = ? OR region = ''
			ORDER BY total_points DESC
			LIMIT ?
		`
	}

	rows, err := qs.db.QueryContext(ctx, query, region, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(&entry.UserID, &entry.TotalPoints, &entry.GlobalRank, &entry.RegionalRank)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// === HELPERS ===

// getTierConfig returns feature config for a subscription tier
func getTierConfig(tier TierLevel) SubscriptionTier {
	configs := map[TierLevel]SubscriptionTier{
		TierFree: {
			Level:             TierFree,
			Name:              "Free",
			MaxQuestsPerWeek:  3,
			MaxXPPerWeek:      50,
			CosmeticDiscount:  0.0,
			ExclusiveFeatures: []string{},
			PriceMonthly:      0,
		},
		TierCasual: {
			Level:             TierCasual,
			Name:              "Casual",
			MaxQuestsPerWeek:  6,
			MaxXPPerWeek:      500,
			CosmeticDiscount:  0.1,
			ExclusiveFeatures: []string{"extended_quest_time"},
			PriceMonthly:      2000, // ~5 USD
		},
		TierPlayer: {
			Level:             TierPlayer,
			Name:              "Player",
			MaxQuestsPerWeek:  10,
			MaxXPPerWeek:      2000,
			CosmeticDiscount:  0.2,
			ExclusiveFeatures: []string{"extended_quest_time", "exclusive_skins"},
			PriceMonthly:      5000, // ~13 USD
		},
		TierExpert: {
			Level:             TierExpert,
			Name:              "Expert",
			MaxQuestsPerWeek:  15,
			MaxXPPerWeek:      10000,
			CosmeticDiscount:  0.3,
			ExclusiveFeatures: []string{"extended_quest_time", "exclusive_skins", "beta_features"},
			PriceMonthly:      15000, // ~40 USD
		},
		TierPro: {
			Level:             TierPro,
			Name:              "Pro",
			MaxQuestsPerWeek:  999, // Unlimited
			MaxXPPerWeek:      50000,
			CosmeticDiscount:  0.5,
			ExclusiveFeatures: []string{"all_features", "api_access", "custom_quests"},
			PriceMonthly:      50000, // ~130 USD
		},
		TierAtelier: {
			Level:             TierAtelier,
			Name:              "PRO ATELIER",
			MaxQuestsPerWeek:  999,
			MaxXPPerWeek:      999999,
			CosmeticDiscount:  1.0, // Free cosmetics
			ExclusiveFeatures: []string{"cadastre_pro_integration", "advanced_analytics", "dedicated_support"},
			PriceMonthly:      150000, // ~400 USD
		},
	}
	return configs[tier]
}

// needsXPReset checks if weekly XP quota should be reset
func needsXPReset(lastReset time.Time) bool {
	daysSince := time.Since(lastReset).Hours() / 24
	return daysSince >= 7
}

// generateID creates a unique ID with prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// === HELPER METHODS (Database operations using repositories) ===

func (qs *QuestService) GetQuestByID(ctx context.Context, questID string) (*Quest, error) {
	repoQuest, err := qs.questRepo.GetQuest(ctx, questID)
	if err != nil {
		return nil, err
	}

	// Convert repository Quest to service Quest
	quest := &Quest{
		ID:            repoQuest.QuestID,
		Title:         repoQuest.Title,
		Description:   repoQuest.Description,
		Type:          QuestType(repoQuest.Type),
		Difficulty:    Difficulty(repoQuest.Difficulty),
		Region:        repoQuest.Region,
		AdminUnit:     repoQuest.AdminUnit,
		MinTier:       TierLevel(repoQuest.MinTier),
		MinXP:         repoQuest.MinXP,
		TargetXP:      repoQuest.TargetXP,
		Status:        QuestStatusAvailable,
		CreatedAt:     repoQuest.CreatedAt,
		UpdatedAt:     repoQuest.UpdatedAt,
	}

	// Parse objectives from JSON
	if err := json.Unmarshal(repoQuest.Objectives, &quest.Objectives); err != nil {
		qs.logger.Printf("Warning: Failed to parse objectives for quest %s: %v\n", questID, err)
		quest.Objectives = []Objective{}
	}

	return quest, nil
}

func (qs *QuestService) GetQuestSession(ctx context.Context, sessionID string) (*QuestSession, error) {
	repoSession, err := qs.questRepo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Convert repository QuestSession to service QuestSession
	session := &QuestSession{
		ID:             repoSession.SessionID,
		UserID:         repoSession.UserID,
		QuestID:        repoSession.QuestID,
		StartedAt:      repoSession.StartedAt,
		LastActivityAt: repoSession.UpdatedAt,
		ElapsedSeconds: repoSession.ElapsedSeconds,
		PartialXP:      repoSession.PartialXP,
		Abandoned:      repoSession.Status == "abandoned",
		AbandonedAt:    repoSession.AbandonedAt,
	}

	// Parse objective progress from JSON
	if err := json.Unmarshal(repoSession.ObjectiveProgress, &session.ObjectiveProgress); err != nil {
		qs.logger.Printf("Warning: Failed to parse objective progress for session %s: %v\n", sessionID, err)
		session.ObjectiveProgress = make(map[string]int)
	}

	return session, nil
}

func (qs *QuestService) storeQuestSession(ctx context.Context, session *QuestSession) error {
	// Convert objective progress to JSON
	progressJSON, err := json.Marshal(session.ObjectiveProgress)
	if err != nil {
		return fmt.Errorf("failed to marshal objective progress: %w", err)
	}

	// Update session using repository
	repoSession := &repository.QuestSession{
		SessionID:         session.ID,
		UserID:            session.UserID,
		QuestID:           session.QuestID,
		StartedAt:         session.StartedAt,
		ElapsedSeconds:    session.ElapsedSeconds,
		ObjectiveProgress: progressJSON,
		PartialXP:         session.PartialXP,
		Status:            "in_progress",
		UpdatedAt:         time.Now(),
	}

	if session.Abandoned {
		repoSession.Status = "abandoned"
		repoSession.AbandonedAt = session.AbandonedAt
	}

	query := `
		INSERT INTO quest_sessions (session_id, user_id, quest_id, started_at, objective_progress, partial_xp, status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			objective_progress = excluded.objective_progress,
			partial_xp = excluded.partial_xp,
			status = excluded.status,
			updated_at = excluded.updated_at
	`

	_, err = qs.db.ExecContext(ctx, query,
		repoSession.SessionID, repoSession.UserID, repoSession.QuestID,
		repoSession.StartedAt, repoSession.ObjectiveProgress, repoSession.PartialXP,
		repoSession.Status, repoSession.UpdatedAt)

	return err
}

func (qs *QuestService) storeUserProgress(ctx context.Context, progress *UserProgress) error {
	// Update progress using repository
	userProgress, err := qs.progressRepo.GetOrCreate(ctx, progress.UserID)
	if err != nil {
		return err
	}

	// Update fields
	userProgress.TierLevel = int(progress.Tier)
	userProgress.TotalXP = progress.TotalXP
	userProgress.CurrentLevel = progress.Level
	userProgress.CurrentLevelXP = progress.CurrentLevelXP
	userProgress.CompletedQuests = len(progress.CompletedQuests)
	userProgress.UpdatedAt = time.Now()

	// Use the repository to update (requires adding update method)
	// For now, we'll use direct DB update
	query := `
		UPDATE user_progress
		SET tier_level = ?, total_xp = ?, current_level = ?, current_level_xp = ?, completed_quests = ?, updated_at = ?
		WHERE user_id = ?
	`

	_, err = qs.db.ExecContext(ctx, query,
		userProgress.TierLevel, userProgress.TotalXP, userProgress.CurrentLevel,
		userProgress.CurrentLevelXP, userProgress.CompletedQuests, userProgress.UpdatedAt,
		progress.UserID)

	// Invalidate cache
	_ = qs.cache.Delete(ctx, fmt.Sprintf("user_progress:%s", progress.UserID))

	return err
}

func (qs *QuestService) countWeeklyQuests(ctx context.Context, userID string) (int, error) {
	sessions, err := qs.questRepo.GetUserActiveSessions(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Count sessions started in the last 7 days
	weekAgo := time.Now().AddDate(0, 0, -7)
	count := 0
	for _, session := range sessions {
		if session.StartedAt.After(weekAgo) {
			count++
		}
	}

	return count, nil
}

func (qs *QuestService) logQuestEvent(ctx context.Context, userID, eventType, questID string, data map[string]interface{}) error {
	// Log to analytics (can be async)
	query := `
		INSERT INTO quest_events (user_id, type, quest_id, data, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	_, err = qs.db.ExecContext(ctx, query, userID, eventType, questID, dataJSON, time.Now())
	return err
}

func (qs *QuestService) GetCosmeticByID(ctx context.Context, cosmeticID string) (*Cosmetic, error) {
	query := `
		SELECT cosmetic_id, name, type, price_xaf, description, icon_url, applicable_to, limited_edition, available_until
		FROM cosmetics
		WHERE cosmetic_id = ?
	`

	var cosmetic Cosmetic
	var applicableTo string
	err := qs.db.QueryRowContext(ctx, query, cosmeticID).Scan(
		&cosmetic.ID, &cosmetic.Name, &cosmetic.Type, &cosmetic.Price, &cosmetic.Description,
		&cosmetic.ImageURL, &applicableTo, &cosmetic.LimitedEdition, &cosmetic.AvailableUntil)

	if err != nil {
		return nil, fmt.Errorf("cosmetic not found: %w", err)
	}

	// Parse applicable_to from string
	if applicableTo != "" {
		cosmetic.ApplicableTo = []string{applicableTo}
	}

	return &cosmetic, nil
}

func (qs *QuestService) updateLeaderboard(ctx context.Context, userID string, progress *UserProgress) {
	// Run async leaderboard update
	go func() {
		rank, err := qs.progressRepo.GetLeaderboardPosition(ctx, userID)
		if err != nil {
			qs.logger.Printf("Error updating leaderboard for %s: %v\n", userID, err)
			return
		}

		query := `
			INSERT INTO leaderboard_global (user_id, total_xp, tier_level, completed_quests, last_updated)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
				total_xp = excluded.total_xp,
				tier_level = excluded.tier_level,
				completed_quests = excluded.completed_quests,
				last_updated = excluded.last_updated
		`

		_, err = qs.db.ExecContext(ctx, query,
			userID, progress.TotalXP, progress.Tier, len(progress.CompletedQuests), time.Now())

		if err != nil {
			qs.logger.Printf("Error storing leaderboard entry for %s: %v\n", userID, err)
		}
	}()
}

// Support types
type LeaderboardEntry struct {
	UserID       string
	TotalPoints  int
	GlobalRank   int
	RegionalRank int
}
