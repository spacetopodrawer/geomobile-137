package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production: validate against allowed origins
		return true
	},
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// Hub maintains active client connections and broadcasts messages
type Hub struct {
	clients       map[*Client]bool              // Set of active clients
	register      chan *Client                  // Register client
	unregister    chan *Client                  // Unregister client
	broadcast     chan *Message                 // Broadcast to all
	parcelSubs    map[string]map[*Client]bool   // Parcel subscriptions: parcel_id -> [clients]
	presenceUpdate chan *PresenceUpdate        // Presence updates
	mu            sync.RWMutex                  // Protects maps
	closed        bool                          // Hub shutdown flag
}

// Client represents a connected WebSocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn      // WebSocket connection
	deviceID string                // Unique device identifier
	userID   string                // User UUID
	send     chan *Message         // Outbound message queue
	done     chan struct{}          // Shutdown signal
	viewport *Viewport             // Current viewing bounds
	mu       sync.RWMutex          // Protects client state
}

// Message represents a WebSocket message
type Message struct {
	Type        string                 `json:"type"`         // parcel_update, conflict, presence, etc.
	ParcelID    string                 `json:"parcel_id,omitempty"`
	DeviceID    string                 `json:"device_id"`
	UserID      string                 `json:"user_id,omitempty"`
	Operation   string                 `json:"operation,omitempty"` // CREATE, UPDATE, DELETE
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
	Conflict    bool                   `json:"conflict,omitempty"`
	MessageID   string                 `json:"message_id"` // UUID for idempotency
	Priority    int                    `json:"priority,omitempty"` // 0-10, higher = more urgent
}

// Viewport represents the client's viewing bounds
type Viewport struct {
	MinX float64 `json:"min_x"`
	MaxX float64 `json:"max_x"`
	MinY float64 `json:"min_y"`
	MaxY float64 `json:"max_y"`
}

// PresenceUpdate tracks user presence and viewport
type PresenceUpdate struct {
	DeviceID   string    `json:"device_id"`
	UserID     string    `json:"user_id"`
	Viewport   *Viewport `json:"viewport"`
	Timestamp  int64     `json:"timestamp"`
	Status     string    `json:"status"` // active, idle, away
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		register:       make(chan *Client, 256),
		unregister:     make(chan *Client, 256),
		broadcast:      make(chan *Message, 1024),
		parcelSubs:     make(map[string]map[*Client]bool),
		presenceUpdate: make(chan *PresenceUpdate, 256),
	}
}

// RegisterConnection creates and registers a new client from a WebSocket connection
func (h *Hub) RegisterConnection(conn *websocket.Conn, deviceID, userID string) *Client {
	client := &Client{
		hub:      h,
		conn:     conn,
		deviceID: deviceID,
		userID:   userID,
		send:     make(chan *Message, 256),
		done:     make(chan struct{}),
	}

	// Register with hub (asynchronously handled by Run loop)
	h.register <- client

	return client
}

// Run starts the hub's event loop
// This should be run in a goroutine
func (h *Hub) Run() {
	log.Println("🚀 WebSocket Hub started (Phase 3B Real-Time)")
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case msg := <-h.broadcast:
			h.handleBroadcast(msg)

		case presence := <-h.presenceUpdate:
			h.handlePresenceUpdate(presence)

		case <-heartbeatTicker.C:
			h.sendHeartbeat()
		}
	}
}

// handleRegister adds a new client to the hub
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		close(client.send)
		return
	}

	h.clients[client] = true
	log.Printf("✓ Client registered: device=%s, user=%s (total: %d)",
		client.deviceID, client.userID, len(h.clients))

	// Send welcome message
	msg := &Message{
		Type:      "welcome",
		DeviceID:  client.deviceID,
		UserID:    client.userID,
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
	}
	select {
	case client.send <- msg:
	case <-client.done:
	}
}

// handleUnregister removes a client from the hub
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	close(client.send)

	// Unsubscribe from all parcels
	for parcelID := range h.parcelSubs {
		delete(h.parcelSubs[parcelID], client)
		if len(h.parcelSubs[parcelID]) == 0 {
			delete(h.parcelSubs, parcelID)
		}
	}

	log.Printf("✗ Client unregistered: device=%s, user=%s (total: %d)",
		client.deviceID, client.userID, len(h.clients))
}

// handleBroadcast sends a message to appropriate clients
func (h *Hub) handleBroadcast(msg *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if msg.Type == "parcel_changed" || msg.Type == "conflict_detected" {
		// Send only to clients subscribed to this parcel
		if subscribers, ok := h.parcelSubs[msg.ParcelID]; ok {
			for client := range subscribers {
				select {
				case client.send <- msg:
				case <-client.done:
				}
			}
		}
	} else if msg.Type == "presence" {
		// Broadcast presence to all clients
		for client := range h.clients {
			select {
			case client.send <- msg:
			case <-client.done:
			}
		}
	} else {
		// Default: broadcast to all connected clients
		for client := range h.clients {
			select {
			case client.send <- msg:
			case <-client.done:
			}
		}
	}
}

// handlePresenceUpdate updates client viewport and broadcasts to others
func (h *Hub) handlePresenceUpdate(presence *PresenceUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find the client and update viewport
	for client := range h.clients {
		if client.deviceID == presence.DeviceID && client.userID == presence.UserID {
			client.mu.Lock()
			client.viewport = presence.Viewport
			client.mu.Unlock()
			break
		}
	}

	// Broadcast to all clients
	msg := &Message{
		Type:      "presence",
		DeviceID:  presence.DeviceID,
		UserID:    presence.UserID,
		Timestamp: presence.Timestamp,
		MessageID: uuid.New().String(),
		Payload: map[string]interface{}{
			"viewport": presence.Viewport,
			"status":   presence.Status,
		},
	}

	for client := range h.clients {
		select {
		case client.send <- msg:
		case <-client.done:
		}
	}
}

// sendHeartbeat sends periodic keep-alive messages
func (h *Hub) sendHeartbeat() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	heartbeat := &Message{
		Type:      "heartbeat",
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
	}

	for client := range h.clients {
		select {
		case client.send <- heartbeat:
		case <-client.done:
		}
	}
}

// SubscribeToParcel adds a client to a parcel's subscription list
func (h *Hub) SubscribeToParcel(client *Client, parcelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.parcelSubs[parcelID]; !ok {
		h.parcelSubs[parcelID] = make(map[*Client]bool)
	}
	h.parcelSubs[parcelID][client] = true

	log.Printf("✓ Client subscribed to parcel: device=%s, parcel=%s", client.deviceID, parcelID)
}

// UnsubscribeFromParcel removes a client from a parcel's subscription list
func (h *Hub) UnsubscribeFromParcel(client *Client, parcelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, ok := h.parcelSubs[parcelID]; ok {
		delete(subscribers, client)
		if len(subscribers) == 0 {
			delete(h.parcelSubs, parcelID)
		}
	}

	log.Printf("✓ Client unsubscribed from parcel: device=%s, parcel=%s", client.deviceID, parcelID)
}

// BroadcastParcelChange sends a parcel update to all subscribers
func (h *Hub) BroadcastParcelChange(msg *Message) {
	msg.MessageID = uuid.New().String()
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("⚠ Broadcast channel full, dropping message: %s", msg.MessageID)
	}
}

// BroadcastConflict sends a conflict detection message
func (h *Hub) BroadcastConflict(parcelID string, devices []string, strategy string) {
	msg := &Message{
		Type:      "conflict_detected",
		ParcelID:  parcelID,
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
		Conflict:  true,
		Payload: map[string]interface{}{
			"devices":  devices,
			"strategy": strategy,
		},
	}
	h.BroadcastParcelChange(msg)
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetSubscriberCount returns the number of subscribers to a parcel
func (h *Hub) GetSubscriberCount(parcelID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if subscribers, ok := h.parcelSubs[parcelID]; ok {
		return len(subscribers)
	}
	return 0
}

// Close gracefully shuts down the hub
func (h *Hub) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}

	h.closed = true

	// Close all client connections
	for client := range h.clients {
		close(client.done)
		client.conn.Close()
	}

	h.clients = make(map[*Client]bool)
	h.parcelSubs = make(map[string]map[*Client]bool)

	log.Printf("✓ WebSocket Hub closed")
	return nil
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, deviceID, userID string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		deviceID: deviceID,
		userID:   userID,
		send:     make(chan *Message, 256),
		done:     make(chan struct{}),
		viewport: &Viewport{},
	}
}

// ReadMessages reads incoming messages from the client's WebSocket connection
func (c *Client) ReadMessages() {
	defer func() {
		c.hub.unregister <- c
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		msg := &Message{}
		err := c.conn.ReadJSON(msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		// Set metadata
		if msg.MessageID == "" {
			msg.MessageID = uuid.New().String()
		}
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().UnixMilli()
		}
		msg.DeviceID = c.deviceID
		msg.UserID = c.userID

		// Route message based on type
		switch msg.Type {
		case "parcel_update":
			c.hub.broadcast <- msg
		case "subscribe":
			if msg.ParcelID != "" {
				c.hub.SubscribeToParcel(c, msg.ParcelID)
			}
		case "unsubscribe":
			if msg.ParcelID != "" {
				c.hub.UnsubscribeFromParcel(c, msg.ParcelID)
			}
		case "presence":
			presence := &PresenceUpdate{
				DeviceID:  c.deviceID,
				UserID:    c.userID,
				Viewport:  c.getViewport(msg),
				Timestamp: msg.Timestamp,
				Status:    "active",
			}
			c.hub.presenceUpdate <- presence
		default:
			log.Printf("⚠ Unknown message type: %s", msg.Type)
		}
	}
}

// WriteMessages writes outgoing messages to the client's WebSocket connection
func (c *Client) WriteMessages() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(msg)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// getViewport extracts viewport from message payload
func (c *Client) getViewport(msg *Message) *Viewport {
	if msg.Payload == nil {
		return &Viewport{}
	}

	viewport := &Viewport{}
	if minX, ok := msg.Payload["min_x"].(float64); ok {
		viewport.MinX = minX
	}
	if maxX, ok := msg.Payload["max_x"].(float64); ok {
		viewport.MaxX = maxX
	}
	if minY, ok := msg.Payload["min_y"].(float64); ok {
		viewport.MinY = minY
	}
	if maxY, ok := msg.Payload["max_y"].(float64); ok {
		viewport.MaxY = maxY
	}
	return viewport
}

// UpdateViewport updates the client's current viewport
func (c *Client) UpdateViewport(vp *Viewport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.viewport = vp
}

// GetViewport returns the client's current viewport
func (c *Client) GetViewport() *Viewport {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.viewport == nil {
		return &Viewport{}
	}
	return c.viewport
}

// Broadcast sends a message to ALL connected clients (Phase 3 REST support)
func (h *Hub) Broadcast(m *Message) {
    h.broadcast <- m
}

// GetConnectedDevices returns the list of connected device IDs (Phase 3 REST support)
func (h *Hub) GetConnectedDevices() []string {
    h.mu.RLock()
    defer h.mu.RUnlock()
    devices := make([]string, 0, len(h.clients))
    for client := range h.clients {
        devices = append(devices, client.deviceID)
    }
    return devices
}
