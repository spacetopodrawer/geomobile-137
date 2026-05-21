package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"cadastre_ia/pkg/quest"
)

// QuestHandler handles quest API endpoints
type QuestHandler struct {
	service *quest.QuestService
	logger  *log.Logger
}

// NewQuestHandler creates a new quest handler
func NewQuestHandler(service *quest.QuestService, logger *log.Logger) *QuestHandler {
	if logger == nil {
		logger = log.New(log.Writer(), "[QuestHandler] ", log.LstdFlags)
	}
	return &QuestHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers quest handlers with the mux
func (qh *QuestHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/quest/available", qh.GetAvailableQuests)
	mux.HandleFunc("POST /api/v1/quest/start", qh.StartQuest)
	mux.HandleFunc("POST /api/v1/quest/abandon", qh.AbandonQuest)
	mux.HandleFunc("POST /api/v1/quest/objective-complete", qh.CompleteObjective)
	mux.HandleFunc("POST /api/v1/quest/complete", qh.CompleteQuest)
	mux.HandleFunc("GET /api/v1/quest/session/{sessionID}", qh.GetSession)
	mux.HandleFunc("GET /api/v1/user/progress", qh.GetUserProgress)
	mux.HandleFunc("POST /api/v1/user/tier-upgrade", qh.UpgradeTier)
	mux.HandleFunc("GET /api/v1/leaderboard", qh.GetLeaderboard)
	mux.HandleFunc("POST /api/v1/cosmetic/purchase", qh.PurchaseCosmetic)
}

// GetAvailableQuests returns available quests for the user
func (qh *QuestHandler) GetAvailableQuests(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	// Get user progress
	progress, err := qh.service.GetUserProgress(r.Context(), userID)
	if err != nil {
		qh.logger.Printf("Error getting user progress: %v\n", err)
		http.Error(w, "Failed to get user progress", http.StatusInternalServerError)
		return
	}

	// Get available quests
	quests, err := qh.service.GetAvailableQuests(r.Context(), userID, progress)
	if err != nil {
		qh.logger.Printf("Error getting available quests: %v\n", err)
		http.Error(w, "Failed to get available quests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"quests": quests,
		"count":  len(quests),
	})
}

// StartQuest starts a new quest session
func (qh *QuestHandler) StartQuest(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		QuestID string `json:"quest_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, err := qh.service.StartQuest(r.Context(), userID, req.QuestID)
	if err != nil {
		qh.logger.Printf("Error starting quest: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// AbandonQuest abandons a quest session
func (qh *QuestHandler) AbandonQuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := qh.service.AbandonQuest(r.Context(), req.SessionID); err != nil {
		qh.logger.Printf("Error abandoning quest: %v\n", err)
		http.Error(w, "Failed to abandon quest", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CompleteObjective marks an objective as complete
func (qh *QuestHandler) CompleteObjective(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID   string `json:"session_id"`
		ObjectiveID string `json:"objective_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := qh.service.CompleteObjective(r.Context(), req.SessionID, req.ObjectiveID); err != nil {
		qh.logger.Printf("Error completing objective: %v\n", err)
		http.Error(w, "Failed to complete objective", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CompleteQuest completes a quest session
func (qh *QuestHandler) CompleteQuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := qh.service.CompleteQuest(r.Context(), req.SessionID); err != nil {
		qh.logger.Printf("Error completing quest: %v\n", err)
		http.Error(w, "Failed to complete quest", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSession retrieves a quest session
func (qh *QuestHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	if sessionID == "" {
		http.Error(w, "Missing sessionID parameter", http.StatusBadRequest)
		return
	}

	session, err := qh.service.GetQuestSession(r.Context(), sessionID)
	if err != nil {
		qh.logger.Printf("Error getting session: %v\n", err)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// GetUserProgress retrieves user progression data
func (qh *QuestHandler) GetUserProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	progress, err := qh.service.GetUserProgress(r.Context(), userID)
	if err != nil {
		qh.logger.Printf("Error getting user progress: %v\n", err)
		http.Error(w, "Failed to get user progress", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

// UpgradeTier upgrades user subscription tier (called after payment)
func (qh *QuestHandler) UpgradeTier(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		NewTier      int `json:"new_tier"`
		DurationDays int `json:"duration_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := qh.service.UpgradeTier(r.Context(), userID, quest.TierLevel(req.NewTier), req.DurationDays); err != nil {
		qh.logger.Printf("Error upgrading tier: %v\n", err)
		http.Error(w, "Failed to upgrade tier", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetLeaderboard returns the leaderboard (global, regional, or weekly)
func (qh *QuestHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")  // "" for global
	scope := r.URL.Query().Get("scope")    // "global" or "weekly"
	limitStr := r.URL.Query().Get("limit") // default 100

	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
		if limit > 1000 {
			limit = 1000 // Cap at 1000
		}
	}

	entries, err := qh.service.GetLeaderboard(r.Context(), region, scope, limit)
	if err != nil {
		qh.logger.Printf("Error getting leaderboard: %v\n", err)
		http.Error(w, "Failed to get leaderboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

// PurchaseCosmetic handles cosmetic purchase
func (qh *QuestHandler) PurchaseCosmetic(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		CosmeticID string `json:"cosmetic_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := qh.service.PurchaseCosmetic(r.Context(), userID, req.CosmeticID); err != nil {
		qh.logger.Printf("Error purchasing cosmetic: %v\n", err)
		http.Error(w, "Failed to purchase cosmetic", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
