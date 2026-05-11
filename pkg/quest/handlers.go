package quest

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// QuestHandlers wraps HTTP handlers for quest endpoints
type QuestHandlers struct {
	service *QuestService
}

// NewQuestHandlers creates a new handler instance
func NewQuestHandlers(service *QuestService) *QuestHandlers {
	return &QuestHandlers{service: service}
}

// === QUEST ENDPOINTS ===

// GetAvailableQuestsHandler returns quests user can play
// GET /api/v1/quest/available
func (h *QuestHandlers) GetAvailableQuestsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID") // From auth middleware

	// Get user progress
	progress, err := h.service.GetUserProgress(ctx, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch user progress")
		return
	}

	// Get available quests
	quests, err := h.service.GetAvailableQuests(ctx, userID, progress)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch quests")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"quests":   quests,
		"user_tier": progress.Tier,
		"total_xp": progress.TotalXP,
	})
}

// StartQuestHandler initializes a quest session
// POST /api/v1/quest/start
func (h *QuestHandlers) StartQuestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")

	var req struct {
		QuestID string `json:"quest_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	session, err := h.service.StartQuest(ctx, userID, req.QuestID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, session)
}

// CompleteObjectiveHandler marks an objective as complete
// POST /api/v1/quest/objective/complete
func (h *QuestHandlers) CompleteObjectiveHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		SessionID   string `json:"session_id"`
		ObjectiveID string `json:"objective_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.service.CompleteObjective(ctx, req.SessionID, req.ObjectiveID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "objective_completed"})
}

// GetQuestStatusHandler retrieves current session progress
// GET /api/v1/quest/session/{sessionId}
func (h *QuestHandlers) GetQuestStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := r.PathValue("sessionId") // Go 1.22+ router parameter

	session, err := h.service.GetQuestSession(ctx, sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Session not found")
		return
	}

	respondJSON(w, http.StatusOK, session)
}

// === USER PROGRESS ENDPOINTS ===

// GetUserProgressHandler returns user's overall progress
// GET /api/v1/quest/progress
func (h *QuestHandlers) GetUserProgressHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")

	progress, err := h.service.GetUserProgress(ctx, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch progress")
		return
	}

	respondJSON(w, http.StatusOK, progress)
}

// === COSMETICS & IAP ENDPOINTS ===

// PurchaseCosmeticHandler handles cosmetic purchase (after payment)
// POST /api/v1/quest/cosmetic/purchase
func (h *QuestHandlers) PurchaseCosmeticHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")

	var req struct {
		CosmeticID string `json:"cosmetic_id"`
		PaymentID  string `json:"payment_id"` // Payment proof from payment gateway
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Verify payment with payment gateway
	// if !verifyPayment(req.PaymentID) {
	//     respondError(w, http.StatusUnauthorized, "Payment verification failed")
	//     return
	// }

	err := h.service.PurchaseCosmetic(ctx, userID, req.CosmeticID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "cosmetic_purchased"})
}

// === SUBSCRIPTION ENDPOINTS ===

// UpgradeTierHandler upgrades user subscription (after payment)
// POST /api/v1/quest/tier/upgrade
func (h *QuestHandlers) UpgradeTierHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")

	var req struct {
		NewTier        int    `json:"new_tier"` // TierLevel enum
		DurationMonths int    `json:"duration_months"`
		PaymentID      string `json:"payment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Verify payment
	err := h.service.UpgradeTier(ctx, userID, TierLevel(req.NewTier), req.DurationMonths)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "tier_upgraded",
		"tier":   req.NewTier,
	})
}

// === LEADERBOARD ENDPOINTS ===

// GetLeaderboardHandler returns leaderboard rankings
// GET /api/v1/quest/leaderboard?region=lekie&scope=global&limit=100
func (h *QuestHandlers) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	region := r.URL.Query().Get("region")  // "" = global
	scope := r.URL.Query().Get("scope")    // "global" or "weekly"
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if scope == "" {
		scope = "global"
	}

	entries, err := h.service.GetLeaderboard(ctx, region, scope, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch leaderboard")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"region":  region,
		"scope":   scope,
		"count":   len(entries),
	})
}

// === UTILITY HANDLERS ===

// respondJSON sends JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondError sends error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// === ROUTES REGISTRATION ===

// RegisterRoutes mounts quest endpoints on the provided router
// Note: Using Go 1.22+ ServeMux or chi/gorilla/echo style router
func RegisterRoutes(mux *http.ServeMux, handlers *QuestHandlers) {
	// Quest endpoints
	mux.HandleFunc("GET /api/v1/quest/available", handlers.GetAvailableQuestsHandler)
	mux.HandleFunc("POST /api/v1/quest/start", handlers.StartQuestHandler)
	mux.HandleFunc("POST /api/v1/quest/objective/complete", handlers.CompleteObjectiveHandler)
	mux.HandleFunc("GET /api/v1/quest/session/{sessionId}", handlers.GetQuestStatusHandler)

	// User progress
	mux.HandleFunc("GET /api/v1/quest/progress", handlers.GetUserProgressHandler)

	// Cosmetics & IAP
	mux.HandleFunc("POST /api/v1/quest/cosmetic/purchase", handlers.PurchaseCosmeticHandler)

	// Subscription
	mux.HandleFunc("POST /api/v1/quest/tier/upgrade", handlers.UpgradeTierHandler)

	// Leaderboard
	mux.HandleFunc("GET /api/v1/quest/leaderboard", handlers.GetLeaderboardHandler)
}
