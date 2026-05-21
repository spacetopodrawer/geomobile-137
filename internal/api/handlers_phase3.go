package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	ws "cadastre_ia/internal/websocket"
)

// Phase3Handlers holds Phase 3 dependencies and in-memory state.
type Phase3Handlers struct {
	Hub *ws.Hub

	mu            sync.RWMutex
	activity      []ActivityEntry
	locks         map[string]ParcelLock
	notifications []Notification
	webhooks      []Webhook
}

type ActivityEntry struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Target    string                 `json:"target,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type ParcelLock struct {
	ParcelID  string    `json:"parcel_id"`
	UserID    string    `json:"user_id"`
	AcquiredAt time.Time `json:"acquired_at"`
}

type Notification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id,omitempty"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body,omitempty"`
	Read      bool                   `json:"read"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Event     string    `json:"event"`
	CreatedAt time.Time `json:"created_at"`
}

// NewPhase3Handlers creates a new Phase3Handlers instance.
func NewPhase3Handlers(hub *ws.Hub) *Phase3Handlers {
	return &Phase3Handlers{
		Hub:           hub,
		activity:      make([]ActivityEntry, 0),
		locks:         make(map[string]ParcelLock),
		notifications: make([]Notification, 0),
		webhooks:      make([]Webhook, 0),
	}
}

// Phase 3A — Real-Time collaboration

func (p *Phase3Handlers) HubStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"clients":   p.Hub.GetConnectedClients(),
		"devices":   p.Hub.GetConnectedDevices(),
		"timestamp": time.Now().UTC(),
	})
}

func (p *Phase3Handlers) GetPresence(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"devices":   p.Hub.GetConnectedDevices(),
		"timestamp": time.Now().UTC(),
	})
}

func (p *Phase3Handlers) LogActivity(c *gin.Context) {
	var entry ActivityEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	p.mu.Lock()
	p.activity = append(p.activity, entry)
	p.mu.Unlock()
	c.JSON(http.StatusCreated, entry)
}

func (p *Phase3Handlers) GetActivityLog(c *gin.Context) {
	p.mu.RLock()
	out := make([]ActivityEntry, len(p.activity))
	copy(out, p.activity)
	p.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"entries": out, "total": len(out)})
}

func (p *Phase3Handlers) AcquireLock(c *gin.Context) {
	parcelID := c.Param("id")
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if existing, ok := p.locks[parcelID]; ok && existing.UserID != userID {
		c.JSON(http.StatusConflict, gin.H{"error": "locked", "owner": existing.UserID})
		return
	}
	lock := ParcelLock{ParcelID: parcelID, UserID: userID, AcquiredAt: time.Now().UTC()}
	p.locks[parcelID] = lock
	c.JSON(http.StatusOK, lock)
}

func (p *Phase3Handlers) ReleaseLock(c *gin.Context) {
	parcelID := c.Param("id")
	p.mu.Lock()
	delete(p.locks, parcelID)
	p.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"parcel_id": parcelID, "released": true})
}

// Phase 3B — Notifications & Webhooks

func (p *Phase3Handlers) ListNotifications(c *gin.Context) {
	p.mu.RLock()
	out := make([]Notification, len(p.notifications))
	copy(out, p.notifications)
	p.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"notifications": out, "total": len(out)})
}

func (p *Phase3Handlers) MarkNotificationRead(c *gin.Context) {
	id := c.Param("id")
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.notifications {
		if p.notifications[i].ID == id {
			p.notifications[i].Read = true
			c.JSON(http.StatusOK, p.notifications[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func (p *Phase3Handlers) BroadcastNotification(c *gin.Context) {
	var n Notification
	if err := c.ShouldBindJSON(&n); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	n.CreatedAt = time.Now().UTC()
	p.mu.Lock()
	p.notifications = append(p.notifications, n)
	p.mu.Unlock()

	p.Hub.Broadcast(&ws.Message{
		Type:      "notification",
		Timestamp: n.CreatedAt.UnixMilli(),
		MessageID: n.ID,
		Payload: map[string]interface{}{
			"title": n.Title,
			"body":  n.Body,
		},
	})
	c.JSON(http.StatusCreated, n)
}

func (p *Phase3Handlers) ListWebhooks(c *gin.Context) {
	p.mu.RLock()
	out := make([]Webhook, len(p.webhooks))
	copy(out, p.webhooks)
	p.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"webhooks": out, "total": len(out)})
}

func (p *Phase3Handlers) CreateWebhook(c *gin.Context) {
	var w Webhook
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w.ID = uuid.New().String()
	w.CreatedAt = time.Now().UTC()
	p.mu.Lock()
	p.webhooks = append(p.webhooks, w)
	p.mu.Unlock()
	c.JSON(http.StatusCreated, w)
}

func (p *Phase3Handlers) DeleteWebhook(c *gin.Context) {
	id := c.Param("id")
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.webhooks {
		if p.webhooks[i].ID == id {
			p.webhooks = append(p.webhooks[:i], p.webhooks[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}
