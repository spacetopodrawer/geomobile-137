package websocket

import (
	"log"
	"sync"
	"time"
)

// PresenceManager manages user presence and viewport tracking
type PresenceManager struct {
	hub       *Hub
	presences map[string]*ClientPresence // device_id -> presence
	mu        sync.RWMutex
}

// ClientPresence represents a client's current presence state
type ClientPresence struct {
	DeviceID   string
	UserID     string
	Viewport   *Viewport
	Status     string    // active, idle, away
	LastUpdate time.Time
	ParcelIDs  map[string]bool // Parcels this client is viewing
}

// NewPresenceManager creates a new presence manager
func NewPresenceManager(hub *Hub) *PresenceManager {
	pm := &PresenceManager{
		hub:       hub,
		presences: make(map[string]*ClientPresence),
	}

	// Start cleanup goroutine
	go pm.cleanupStalePresences()

	return pm
}

// UpdatePresence updates or creates a presence record
func (pm *PresenceManager) UpdatePresence(deviceID, userID string, viewport *Viewport) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if presence, ok := pm.presences[deviceID]; ok {
		presence.LastUpdate = time.Now()
		presence.Viewport = viewport
		presence.Status = "active"
	} else {
		pm.presences[deviceID] = &ClientPresence{
			DeviceID:   deviceID,
			UserID:     userID,
			Viewport:   viewport,
			Status:     "active",
			LastUpdate: time.Now(),
			ParcelIDs:  make(map[string]bool),
		}
	}
}

// AddParcelToViewport marks that a client is viewing a parcel
func (pm *PresenceManager) AddParcelToViewport(deviceID, parcelID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if presence, ok := pm.presences[deviceID]; ok {
		presence.ParcelIDs[parcelID] = true
	}
}

// RemoveParcelFromViewport marks that a client is no longer viewing a parcel
func (pm *PresenceManager) RemoveParcelFromViewport(deviceID, parcelID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if presence, ok := pm.presences[deviceID]; ok {
		delete(presence.ParcelIDs, parcelID)
	}
}

// GetPresence retrieves presence information for a device
func (pm *PresenceManager) GetPresence(deviceID string) *ClientPresence {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if presence, ok := pm.presences[deviceID]; ok {
		return presence
	}
	return nil
}

// GetAllPresences returns all active presences
func (pm *PresenceManager) GetAllPresences() []*ClientPresence {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	presences := make([]*ClientPresence, 0, len(pm.presences))
	for _, presence := range pm.presences {
		if presence.Status != "away" {
			presences = append(presences, presence)
		}
	}
	return presences
}

// GetPresencesByParcel returns all clients viewing a specific parcel
func (pm *PresenceManager) GetPresencesByParcel(parcelID string) []*ClientPresence {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	presences := make([]*ClientPresence, 0)
	for _, presence := range pm.presences {
		if presence.ParcelIDs[parcelID] && presence.Status == "active" {
			presences = append(presences, presence)
		}
	}
	return presences
}

// GetPresencesInViewport returns all clients with viewports overlapping a region
func (pm *PresenceManager) GetPresencesInViewport(minX, maxX, minY, maxY float64) []*ClientPresence {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	presences := make([]*ClientPresence, 0)
	for _, presence := range pm.presences {
		if presence.Status == "active" && pm.viewportsOverlap(presence.Viewport, minX, maxX, minY, maxY) {
			presences = append(presences, presence)
		}
	}
	return presences
}

// viewportsOverlap checks if two viewports intersect
func (pm *PresenceManager) viewportsOverlap(vp *Viewport, minX, maxX, minY, maxY float64) bool {
	return vp.MinX < maxX && vp.MaxX > minX && vp.MinY < maxY && vp.MaxY > minY
}

// RemovePresence removes a presence record
func (pm *PresenceManager) RemovePresence(deviceID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.presences, deviceID)
	log.Printf("✓ Presence removed: device=%s", deviceID)
}

// MarkAway marks a client as away (idle)
func (pm *PresenceManager) MarkAway(deviceID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if presence, ok := pm.presences[deviceID]; ok {
		presence.Status = "away"
		presence.LastUpdate = time.Now()
	}
}

// MarkActive marks a client as active
func (pm *PresenceManager) MarkActive(deviceID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if presence, ok := pm.presences[deviceID]; ok {
		presence.Status = "active"
		presence.LastUpdate = time.Now()
	}
}

// cleanupStalePresences periodically removes stale presence records
func (pm *PresenceManager) cleanupStalePresences() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pm.mu.Lock()

		now := time.Now()
		staleThreshold := 30 * time.Minute

		for deviceID, presence := range pm.presences {
			if now.Sub(presence.LastUpdate) > staleThreshold {
				delete(pm.presences, deviceID)
				log.Printf("⚠ Stale presence cleaned up: device=%s", deviceID)
			}
		}

		pm.mu.Unlock()
	}
}

// BroadcastPresenceUpdate sends presence information to all clients
func (pm *PresenceManager) BroadcastPresenceUpdate(presenceUpdates []*ClientPresence) {
	// Convert to wire format
	payload := make([]map[string]interface{}, len(presenceUpdates))
	for i, presence := range presenceUpdates {
		payload[i] = map[string]interface{}{
			"device_id": presence.DeviceID,
			"user_id":   presence.UserID,
			"status":    presence.Status,
			"viewport": map[string]float64{
				"min_x": presence.Viewport.MinX,
				"max_x": presence.Viewport.MaxX,
				"min_y": presence.Viewport.MinY,
				"max_y": presence.Viewport.MaxY,
			},
			"timestamp": presence.LastUpdate.UnixMilli(),
		}
	}

	msg := &Message{
		Type:      "presence_update",
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"presences": payload,
		},
	}

	select {
	case pm.hub.broadcast <- msg:
	default:
		log.Printf("⚠ Broadcast channel full for presence update")
	}
}

// GetPresenceStatistics returns statistics about presence
func (pm *PresenceManager) GetPresenceStatistics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	activeCount := 0
	idleCount := 0
	awayCount := 0

	for _, presence := range pm.presences {
		switch presence.Status {
		case "active":
			activeCount++
		case "idle":
			idleCount++
		case "away":
			awayCount++
		}
	}

	return map[string]interface{}{
		"active_count": activeCount,
		"idle_count":   idleCount,
		"away_count":   awayCount,
		"total":        len(pm.presences),
		"timestamp":    time.Now().UnixMilli(),
	}
}

// SyncPresenceWithHub ensures presence tracking is synchronized with hub clients
func (pm *PresenceManager) SyncPresenceWithHub() {
	pm.hub.mu.RLock()
	defer pm.hub.mu.RUnlock()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Track devices in hub
	hubDevices := make(map[string]bool)
	for client := range pm.hub.clients {
		hubDevices[client.deviceID] = true
	}

	// Remove presences not in hub
	for deviceID := range pm.presences {
		if !hubDevices[deviceID] {
			delete(pm.presences, deviceID)
		}
	}

	log.Printf("✓ Presence synced with hub: %d clients", len(hubDevices))
}
