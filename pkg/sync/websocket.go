package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketHub manages WebSocket connections and broadcasts
type WebSocketHub struct {
	mu           sync.RWMutex
	clients      map[string]*WSClient
	broadcast    chan *SyncMessage
	register     chan *WSClient
	unregister   chan *WSClient
	syncManager  *SyncManager
	maxMessageSize int64
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	hub      *WebSocketHub
	conn     *websocket.Conn
	deviceID string
	send     chan *SyncMessage
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(syncMgr *SyncManager) *WebSocketHub {
	hub := &WebSocketHub{
		clients:        make(map[string]*WSClient),
		broadcast:      make(chan *SyncMessage, 256),
		register:       make(chan *WSClient, 256),
		unregister:     make(chan *WSClient, 256),
		syncManager:    syncMgr,
		maxMessageSize: 1048576, // 1MB
	}

	// Start hub event loop
	go hub.eventLoop()

	return hub
}

// eventLoop runs the hub's event loop
func (h *WebSocketHub) eventLoop() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.deviceID] = client
			h.mu.Unlock()
			log.Printf("✓ Client registered: %s (total: %d)", client.deviceID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.deviceID]; ok {
				delete(h.clients, client.deviceID)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("✓ Client unregistered: %s (total: %d)", client.deviceID, len(h.clients))

		case msg := <-h.broadcast:
			h.broadcastMessage(msg)
		}
	}
}

// HandleConnection handles a new WebSocket connection
func (h *WebSocketHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}

	deviceID := uuid.New().String()
	client := &WSClient{
		hub:      h,
		conn:     conn,
		deviceID: deviceID,
		send:     make(chan *SyncMessage, 256),
	}

	h.register <- client

	// Start client reader/writer goroutines
	go client.readPump()
	go client.writePump()
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.hub.maxMessageSize)
	// No deadline for reads - use zero time

	for {
		var msg SyncMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			return
		}

		// Update device ID from message if provided
		if msg.DeviceID != "" {
			c.deviceID = msg.DeviceID
		}

		// Process message through sync manager
		if err := c.hub.syncManager.ReceiveMessage(&msg); err != nil {
			log.Printf("❌ Sync error: %v", err)
		}

		// Broadcast to other clients
		c.hub.broadcast <- &msg
	}
}

// writePump writes messages to the WebSocket connection
func (c *WSClient) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("❌ JSON marshal error: %v", err)
			return
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			return
		}
	}
}

// broadcastMessage broadcasts a message to all connected clients
func (h *WebSocketHub) broadcastMessage(msg *SyncMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for deviceID, client := range h.clients {
		// Don't send to originating client
		if deviceID == msg.DeviceID {
			continue
		}

		select {
		case client.send <- msg:
		default:
			// Client's send channel is full, drop the message
			log.Printf("⚠️  Client send queue full: %s", deviceID)
		}
	}
}

// SendToDevice sends a message to a specific device
func (h *WebSocketHub) SendToDevice(deviceID string, msg *SyncMessage) error {
	h.mu.RLock()
	client, exists := h.clients[deviceID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device not connected: %s", deviceID)
	}

	select {
	case client.send <- msg:
		return nil
	default:
		return fmt.Errorf("client send queue full: %s", deviceID)
	}
}

// GetConnectedDevices returns list of connected device IDs
func (h *WebSocketHub) GetConnectedDevices() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	devices := make([]string, 0, len(h.clients))
	for deviceID := range h.clients {
		devices = append(devices, deviceID)
	}
	return devices
}

// GetClientCount returns number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetStatus returns hub status
func (h *WebSocketHub) GetStatus() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"connected_clients": len(h.clients),
		"devices":           h.GetConnectedDevices(),
		"queue_size":        len(h.broadcast),
	}
}

// Close closes all connections
func (h *WebSocketHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		close(client.send)
		client.conn.Close()
	}
}
