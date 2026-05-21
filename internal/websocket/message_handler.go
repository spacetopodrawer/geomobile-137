package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MessageHandler handles WebSocket message routing and business logic
type MessageHandler struct {
	hub *Hub
	mu  sync.RWMutex
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(hub *Hub) *MessageHandler {
	return &MessageHandler{
		hub: hub,
	}
}

// HandleParcelUpdate processes parcel update messages
// This is where conflict detection and resolution happens
func (mh *MessageHandler) HandleParcelUpdate(msg *Message) error {
	if msg.ParcelID == "" {
		log.Printf("⚠ Parcel update missing parcel_id: device=%s", msg.DeviceID)
		return nil
	}

	// Validate operation type
	if msg.Operation != "CREATE" && msg.Operation != "UPDATE" && msg.Operation != "DELETE" {
		log.Printf("⚠ Invalid operation: %s", msg.Operation)
		return nil
	}

	// Set timestamp if not already set
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}

	// Check for conflicts (would integrate with Phase 2 sync engine)
	if msg.Operation == "UPDATE" {
		conflicts := mh.DetectConflict(msg.ParcelID, msg.DeviceID)
		if conflicts {
			msg.Conflict = true
			log.Printf("⚠️ Conflict detected on parcel=%s from device=%s", msg.ParcelID, msg.DeviceID)
		}
	}

	// Broadcast to subscribers
	mh.hub.BroadcastParcelChange(msg)

	// Log transaction (would write to database in production)
	mh.logTransaction(msg)

	return nil
}

// DetectConflict checks if multiple devices are editing the same parcel
func (mh *MessageHandler) DetectConflict(parcelID string, currentDevice string) bool {
	mh.hub.mu.RLock()
	defer mh.hub.mu.RUnlock()

	// Check if other devices are subscribed to this parcel
	if subscribers, ok := mh.hub.parcelSubs[parcelID]; ok {
		deviceCount := make(map[string]bool)
		for client := range subscribers {
			deviceCount[client.deviceID] = true
		}

		// Conflict if more than one device is viewing/editing this parcel
		if len(deviceCount) > 1 {
			return true
		}
	}

	return false
}

// GetConflictingDevices returns list of devices currently editing a parcel
func (mh *MessageHandler) GetConflictingDevices(parcelID string) []string {
	mh.hub.mu.RLock()
	defer mh.hub.mu.RUnlock()

	devices := []string{}
	if subscribers, ok := mh.hub.parcelSubs[parcelID]; ok {
		seen := make(map[string]bool)
		for client := range subscribers {
			if !seen[client.deviceID] {
				devices = append(devices, client.deviceID)
				seen[client.deviceID] = true
			}
		}
	}
	return devices
}

// HandlePresenceUpdate processes presence/viewport updates
func (mh *MessageHandler) HandlePresenceUpdate(msg *Message) error {
	if msg.Payload == nil {
		return nil
	}

	// Extract viewport from payload
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

	presence := &PresenceUpdate{
		DeviceID:  msg.DeviceID,
		UserID:    msg.UserID,
		Viewport:  viewport,
		Timestamp: time.Now().UnixMilli(),
		Status:    "active",
	}

	// Process through hub
	select {
	case mh.hub.presenceUpdate <- presence:
	default:
		log.Printf("⚠ Presence channel full for device=%s", msg.DeviceID)
	}

	return nil
}

// SubscribeClient adds a client to a parcel's subscription list
func (mh *MessageHandler) SubscribeClient(client *Client, parcelID string) {
	mh.hub.SubscribeToParcel(client, parcelID)

	// Send confirmation
	confirmMsg := &Message{
		Type:      "subscribed",
		ParcelID:  parcelID,
		DeviceID:  client.deviceID,
		UserID:    client.userID,
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
		Payload: map[string]interface{}{
			"subscriber_count": mh.hub.GetSubscriberCount(parcelID),
		},
	}

	select {
	case client.send <- confirmMsg:
	case <-client.done:
	}
}

// UnsubscribeClient removes a client from a parcel's subscription list
func (mh *MessageHandler) UnsubscribeClient(client *Client, parcelID string) {
	mh.hub.UnsubscribeFromParcel(client, parcelID)

	// Send confirmation
	confirmMsg := &Message{
		Type:      "unsubscribed",
		ParcelID:  parcelID,
		DeviceID:  client.deviceID,
		UserID:    client.userID,
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
	}

	select {
	case client.send <- confirmMsg:
	case <-client.done:
	}
}

// logTransaction would write transaction to database
// In production, this would integrate with the PostgreSQL transaction_log table
func (mh *MessageHandler) logTransaction(msg *Message) {
	// TODO: Write to transaction_log table with:
	// - parcel_id
	// - device_id
	// - user_id
	// - operation
	// - before_state (if UPDATE)
	// - after_state
	// - timestamp
	// - conflict flag
	log.Printf("📝 Transaction logged: parcel=%s, op=%s, device=%s, conflict=%v",
		msg.ParcelID, msg.Operation, msg.DeviceID, msg.Conflict)
}

// GetActiveParcels returns list of parcels with active subscribers
func (mh *MessageHandler) GetActiveParcels() []string {
	mh.hub.mu.RLock()
	defer mh.hub.mu.RUnlock()

	parcels := []string{}
	for parcelID := range mh.hub.parcelSubs {
		parcels = append(parcels, parcelID)
	}
	return parcels
}

// GetClientCount returns total number of connected clients
func (mh *MessageHandler) GetClientCount() int {
	return mh.hub.GetConnectedClients()
}

// BroadcastSystemMessage sends a system-level message to all clients
func (mh *MessageHandler) BroadcastSystemMessage(msgType string, payload map[string]interface{}) {
	msg := &Message{
		Type:      msgType,
		DeviceID:  "system",
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
		Payload:   payload,
	}

	select {
	case mh.hub.broadcast <- msg:
	default:
		log.Printf("⚠ System broadcast channel full")
	}
}

// GetHubStats returns statistics about the hub state
func (mh *MessageHandler) GetHubStats() map[string]interface{} {
	mh.hub.mu.RLock()
	defer mh.hub.mu.RUnlock()

	activeParcels := len(mh.hub.parcelSubs)
	connectedClients := len(mh.hub.clients)

	totalSubscriptions := 0
	for _, subscribers := range mh.hub.parcelSubs {
		totalSubscriptions += len(subscribers)
	}

	return map[string]interface{}{
		"connected_clients":     connectedClients,
		"active_parcels":        activeParcels,
		"total_subscriptions":   totalSubscriptions,
		"avg_subscriptions":     float64(totalSubscriptions) / float64(max(activeParcels, 1)),
		"timestamp":             time.Now().UnixMilli(),
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ResolveConflict handles conflict resolution using Phase 2 Last-Write-Wins strategy
// In production, this would integrate with the sync.ConflictResolver from Phase 2
func (mh *MessageHandler) ResolveConflict(parcelID string, edit1, edit2 *Message) *Message {
	// Last-Write-Wins: newer timestamp wins
	if edit1.Timestamp >= edit2.Timestamp {
		return edit1
	}
	return edit2
}

// ValidateMessage checks message integrity and required fields
func (mh *MessageHandler) ValidateMessage(msg *Message) error {
	if msg.MessageID == "" {
		msg.MessageID = uuid.New().String()
	}

	if msg.DeviceID == "" {
		return nil // Will be set by client
	}

	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}

	return nil
}
