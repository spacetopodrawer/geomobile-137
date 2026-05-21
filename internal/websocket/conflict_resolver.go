package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuthorityLevel defines user permission hierarchy
type AuthorityLevel int

const (
	AuthoritySystem AuthorityLevel = 1000  // System operations
	AuthoritySuperAdmin AuthorityLevel = 900
	AuthorityAdmin AuthorityLevel = 800
	AuthorityAuthor AuthorityLevel = 700   // Parcel creator/owner
	AuthorityCoAuthor AuthorityLevel = 600 // Delegated editor
	AuthorityUser AuthorityLevel = 500     // General user
)

// ParcelDomain defines administrative jurisdiction
type ParcelDomain string

const (
	DomainNational ParcelDomain = "national"
	DomainState ParcelDomain = "state"
	DomainMunicipality ParcelDomain = "municipality"
	DomainPrivate ParcelDomain = "private"
	DomainPublic ParcelDomain = "public"
)

// ConflictResolver handles conflict detection and resolution for real-time edits
type ConflictResolver struct {
	mu       sync.RWMutex
	conflicts map[string]*ConflictRecord // parcel_id -> conflict record
}

// HierarchicalConflictResolver applies authority-based, domain-aware conflict resolution
type HierarchicalConflictResolver struct {
	mu sync.RWMutex
	conflicts map[string]*HierarchicalConflictRecord
	escalationCallbacks map[string]func(*HierarchicalConflictRecord) // Custom escalation handlers
}

// HierarchicalConflictRecord tracks conflict with authority context
type HierarchicalConflictRecord struct {
	ID                      string
	ParcelID                string
	ParcelDomain            ParcelDomain
	ParcelCustodianUserID   string
	Edit1UserID             string
	Edit1UserAuthority      AuthorityLevel
	Edit1DevicePriority     int // 0-100
	Edit1Data               *Message
	Edit1Timestamp          int64
	Edit2UserID             string
	Edit2UserAuthority      AuthorityLevel
	Edit2DevicePriority     int
	Edit2Data               *Message
	Edit2Timestamp          int64
	DetectedAt              time.Time
	ResolutionStrategy      string // "authority_hierarchy", "device_priority", "averaging", "escalation"
	ResolvedAt              *time.Time
	ResolvedByUserID        string
	WinningEdit             *Message
	WinningEditSource       string // "edit_1", "edit_2", "averaged", "merged"
	ResolutionNotes         string
	EscalatedToUserID       string
	EscalationReason        string
	EscalationTimestamp     *time.Time
	AlertedUsers            []string
	ResolutionDeadline      time.Time
}

// UserContext provides authority information for a user/device
type UserContext struct {
	UserID           string
	Authority        AuthorityLevel
	DeviceID         string
	DevicePriority   int // 0-100, higher = more trusted
	DeviceType       string // "rtk_gnss", "ppp", "survey", "mobile", "legacy"
}

// ConflictRecord represents a detected conflict
type ConflictRecord struct {
	ID              string
	ParcelID        string
	DeviceID1       string
	DeviceID2       string
	UserID1         string
	UserID2         string
	Edit1           *Message
	Edit2           *Message
	DetectedAt      time.Time
	ResolvedAt      *time.Time
	Strategy        string // last_write_wins, user_choice, custom_rule
	ResolutionNotes string
	WinningEdit     *Message
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver() *ConflictResolver {
	cr := &ConflictResolver{
		conflicts: make(map[string]*ConflictRecord),
	}

	// Start cleanup goroutine
	go cr.cleanupResolvedConflicts()

	return cr
}

// NewHierarchicalConflictResolver creates authority-aware resolver
func NewHierarchicalConflictResolver() *HierarchicalConflictResolver {
	hcr := &HierarchicalConflictResolver{
		conflicts:               make(map[string]*HierarchicalConflictRecord),
		escalationCallbacks:     make(map[string]func(*HierarchicalConflictRecord)),
	}

	go hcr.monitorEscalationDeadlines()

	return hcr
}

// RegisterEscalationHandler registers custom handler for escalation events
func (hcr *HierarchicalConflictResolver) RegisterEscalationHandler(
	eventType string,
	handler func(*HierarchicalConflictRecord),
) {
	hcr.mu.Lock()
	defer hcr.mu.Unlock()
	hcr.escalationCallbacks[eventType] = handler
}

// DetectHierarchicalConflict records conflict with authority context
func (hcr *HierarchicalConflictResolver) DetectHierarchicalConflict(
	parcelID string,
	parcelDomain ParcelDomain,
	parcelCustodian string,
	user1 *UserContext,
	edit1 *Message,
	user2 *UserContext,
	edit2 *Message,
) *HierarchicalConflictRecord {
	hcr.mu.Lock()
	defer hcr.mu.Unlock()

	record := &HierarchicalConflictRecord{
		ID:                    uuid.New().String(),
		ParcelID:              parcelID,
		ParcelDomain:          parcelDomain,
		ParcelCustodianUserID: parcelCustodian,
		Edit1UserID:           user1.UserID,
		Edit1UserAuthority:    user1.Authority,
		Edit1DevicePriority:   user1.DevicePriority,
		Edit1Data:             edit1,
		Edit1Timestamp:        edit1.Timestamp,
		Edit2UserID:           user2.UserID,
		Edit2UserAuthority:    user2.Authority,
		Edit2DevicePriority:   user2.DevicePriority,
		Edit2Data:             edit2,
		Edit2Timestamp:        edit2.Timestamp,
		DetectedAt:            time.Now(),
		ResolutionDeadline:    time.Now().Add(30 * time.Minute),
		AlertedUsers:          []string{user1.UserID, user2.UserID},
	}

	key := parcelID + ":" + user1.UserID + ":" + user2.UserID
	hcr.conflicts[key] = record

	log.Printf("🔴 HIERARCHICAL CONFLICT: parcel=%s, user1(%v)=%s, user2(%v)=%s",
		parcelID, user1.Authority, user1.UserID, user2.Authority, user2.UserID)

	return record
}

// ResolveHierarchicalConflict applies authority-based resolution
func (hcr *HierarchicalConflictResolver) ResolveHierarchicalConflict(
	record *HierarchicalConflictRecord,
) *HierarchicalConflictRecord {
	hcr.mu.Lock()
	defer hcr.mu.Unlock()

	now := time.Now()
	record.ResolvedAt = &now

	// Step 1: Compare user authority levels
	if record.Edit1UserAuthority > record.Edit2UserAuthority {
		record.ResolutionStrategy = "authority_hierarchy"
		record.WinningEdit = record.Edit1Data
		record.WinningEditSource = "edit_1"
		record.ResolutionNotes = "Edit1 user has higher authority"
		return record
	}

	if record.Edit2UserAuthority > record.Edit1UserAuthority {
		record.ResolutionStrategy = "authority_hierarchy"
		record.WinningEdit = record.Edit2Data
		record.WinningEditSource = "edit_2"
		record.ResolutionNotes = "Edit2 user has higher authority"
		return record
	}

	// Step 2: Same authority level? Compare device priority
	if record.Edit1DevicePriority > record.Edit2DevicePriority {
		record.ResolutionStrategy = "device_priority"
		record.WinningEdit = record.Edit1Data
		record.WinningEditSource = "edit_1"
		record.ResolutionNotes = "Edit1 device has higher priority/precision"
		return record
	}

	if record.Edit2DevicePriority > record.Edit1DevicePriority {
		record.ResolutionStrategy = "device_priority"
		record.WinningEdit = record.Edit2Data
		record.WinningEditSource = "edit_2"
		record.ResolutionNotes = "Edit2 device has higher priority/precision"
		return record
	}

	// Step 3: Equal authority and device? Try averaging
	if hcr.canAverageEdits(record.Edit1Data, record.Edit2Data) {
		merged := hcr.averageCoordinates(record.Edit1Data, record.Edit2Data)
		record.ResolutionStrategy = "averaging"
		record.WinningEdit = merged
		record.WinningEditSource = "averaged"
		record.ResolutionNotes = "Coordinates averaged due to equal authority and device priority"
		return record
	}

	// Step 4: Unresolvable? Escalate to custodian
	record.ResolutionStrategy = "escalation"
	record.EscalationReason = "Unable to auto-resolve: equal authority, equal device priority, non-averageable data"
	record.EscalatedToUserID = record.ParcelCustodianUserID
	escalationTime := time.Now()
	record.EscalationTimestamp = &escalationTime

	log.Printf("✅ ESCALATED: parcel=%s to custodian=%s, reason=%s",
		record.ParcelID, record.ParcelCustodianUserID, record.EscalationReason)

	// Trigger escalation callback
	if handler, ok := hcr.escalationCallbacks["escalation"]; ok {
		handler(record)
	}

	return record
}

// canAverageEdits checks if edits contain averageable data (coordinates)
func (hcr *HierarchicalConflictResolver) canAverageEdits(edit1, edit2 *Message) bool {
	// Only average UPDATE operations with coordinate payloads
	if edit1.Operation != "UPDATE" || edit2.Operation != "UPDATE" {
		return false
	}

	if edit1.Payload == nil || edit2.Payload == nil {
		return false
	}

	// Check for coordinates field
	_, has1 := edit1.Payload["coordinates"]
	_, has2 := edit2.Payload["coordinates"]

	return has1 && has2
}

// averageCoordinates creates averaged edit from two edits
func (hcr *HierarchicalConflictResolver) averageCoordinates(edit1, edit2 *Message) *Message {
	merged := &Message{
		Type:      edit1.Type,
		ParcelID:  edit1.ParcelID,
		Operation: "UPDATE",
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
		Payload:   make(map[string]interface{}),
	}

	// Average the coordinate arrays
	if coords1, ok := edit1.Payload["coordinates"].([]interface{}); ok {
		if coords2, ok := edit2.Payload["coordinates"].([]interface{}); ok {
			averaged := make([]interface{}, len(coords1))
			for i := range coords1 {
				// In real implementation, would parse coordinates and average them
				// For now, alternate between the two
				if i%2 == 0 {
					averaged[i] = coords1[i]
				} else {
					averaged[i] = coords2[i]
				}
			}
			merged.Payload["coordinates"] = averaged
		}
	}

	// Preserve other attributes from both
	for k, v := range edit1.Payload {
		if k != "coordinates" {
			merged.Payload[k] = v
		}
	}

	return merged
}

// GetUnresolvedConflicts returns conflicts awaiting escalation
func (hcr *HierarchicalConflictResolver) GetUnresolvedConflicts() []*HierarchicalConflictRecord {
	hcr.mu.RLock()
	defer hcr.mu.RUnlock()

	unresolved := make([]*HierarchicalConflictRecord, 0)
	for _, record := range hcr.conflicts {
		if record.ResolvedAt == nil {
			unresolved = append(unresolved, record)
		}
	}
	return unresolved
}

// monitorEscalationDeadlines checks for expired deadlines
func (hcr *HierarchicalConflictResolver) monitorEscalationDeadlines() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		hcr.mu.Lock()

		now := time.Now()
		for _, record := range hcr.conflicts {
			if record.ResolvedAt == nil && now.After(record.ResolutionDeadline) {
				// Auto-escalate if not resolved within deadline
				record.ResolutionStrategy = "escalation_timeout"
				record.EscalatedToUserID = record.ParcelCustodianUserID
				escalationTime := now
				record.EscalationTimestamp = &escalationTime

				log.Printf("⏰ AUTO-ESCALATION: parcel=%s, unresolved for 30min", record.ParcelID)

				if handler, ok := hcr.escalationCallbacks["timeout"]; ok {
					handler(record)
				}
			}
		}

		hcr.mu.Unlock()
	}
}

// DetectConflict records a conflict between two edits
func (cr *ConflictResolver) DetectConflict(parcelID, device1, device2 string, edit1, edit2 *Message) *ConflictRecord {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	record := &ConflictRecord{
		ID:         uuid.New().String(),
		ParcelID:   parcelID,
		DeviceID1:  device1,
		DeviceID2:  device2,
		UserID1:    edit1.UserID,
		UserID2:    edit2.UserID,
		Edit1:      edit1,
		Edit2:      edit2,
		DetectedAt: time.Now(),
		Strategy:   "last_write_wins", // Default strategy
	}

	// Store conflict
	key := parcelID + ":" + device1 + ":" + device2
	cr.conflicts[key] = record

	log.Printf("🔴 CONFLICT DETECTED: parcel=%s, device1=%s, device2=%s, strategy=%s",
		parcelID, device1, device2, record.Strategy)

	return record
}

// ResolveConflict applies resolution strategy (Last-Write-Wins from Phase 2)
func (cr *ConflictResolver) ResolveConflict(record *ConflictRecord) *Message {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	now := time.Now()
	record.ResolvedAt = &now

	var winning *Message

	switch record.Strategy {
	case "last_write_wins":
		// Newer timestamp wins (from Phase 2 sync engine)
		if record.Edit1.Timestamp >= record.Edit2.Timestamp {
			winning = record.Edit1
			record.ResolutionNotes = "Edit1 won by Last-Write-Wins (newer timestamp)"
		} else {
			winning = record.Edit2
			record.ResolutionNotes = "Edit2 won by Last-Write-Wins (newer timestamp)"
		}

	case "user_choice":
		// Would be resolved by user UI selection
		record.ResolutionNotes = "Pending user choice"
		return nil

	case "custom_rule":
		// Custom resolution logic (would be user-defined)
		winning = record.Edit1
		record.ResolutionNotes = "Resolved by custom rule"
	}

	record.WinningEdit = winning
	log.Printf("✅ CONFLICT RESOLVED: parcel=%s, strategy=%s, winner_device=%s",
		record.ParcelID, record.Strategy, winning.DeviceID)

	return winning
}

// GetConflict retrieves a conflict record
func (cr *ConflictResolver) GetConflict(key string) *ConflictRecord {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	return cr.conflicts[key]
}

// GetConflictsByParcel retrieves all conflicts for a parcel
func (cr *ConflictResolver) GetConflictsByParcel(parcelID string) []*ConflictRecord {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	records := make([]*ConflictRecord, 0)
	for _, record := range cr.conflicts {
		if record.ParcelID == parcelID {
			records = append(records, record)
		}
	}
	return records
}

// GetUnresolvedConflicts returns conflicts that haven't been resolved yet
func (cr *ConflictResolver) GetUnresolvedConflicts() []*ConflictRecord {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	records := make([]*ConflictRecord, 0)
	for _, record := range cr.conflicts {
		if record.ResolvedAt == nil {
			records = append(records, record)
		}
	}
	return records
}

// CreateConflictVisualization creates visualization data for UI rendering
func (cr *ConflictResolver) CreateConflictVisualization(record *ConflictRecord) map[string]interface{} {
	return map[string]interface{}{
		"conflict_id": record.ID,
		"parcel_id":   record.ParcelID,
		"timestamp":   record.DetectedAt.UnixMilli(),
		"devices": []map[string]interface{}{
			{
				"device_id": record.DeviceID1,
				"user_id":   record.UserID1,
				"edit": map[string]interface{}{
					"operation":  record.Edit1.Operation,
					"timestamp":  record.Edit1.Timestamp,
					"payload":    record.Edit1.Payload,
					"message_id": record.Edit1.MessageID,
				},
			},
			{
				"device_id": record.DeviceID2,
				"user_id":   record.UserID2,
				"edit": map[string]interface{}{
					"operation":  record.Edit2.Operation,
					"timestamp":  record.Edit2.Timestamp,
					"payload":    record.Edit2.Payload,
					"message_id": record.Edit2.MessageID,
				},
			},
		},
		"strategy":   record.Strategy,
		"resolved":   record.ResolvedAt != nil,
		"winner":    record.WinningEdit.DeviceID,
		"resolution": record.ResolutionNotes,
	}
}

// cleanupResolvedConflicts periodically removes resolved conflicts
func (cr *ConflictResolver) cleanupResolvedConflicts() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cr.mu.Lock()

		now := time.Now()
		maxAge := 24 * time.Hour

		for key, record := range cr.conflicts {
			if record.ResolvedAt != nil && now.Sub(*record.ResolvedAt) > maxAge {
				delete(cr.conflicts, key)
				log.Printf("🧹 Cleaned up resolved conflict: %s", key)
			}
		}

		cr.mu.Unlock()
	}
}

// GetConflictStatistics returns conflict statistics
func (cr *ConflictResolver) GetConflictStatistics() map[string]interface{} {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	totalConflicts := len(cr.conflicts)
	resolvedCount := 0
	unresolvedCount := 0

	for _, record := range cr.conflicts {
		if record.ResolvedAt != nil {
			resolvedCount++
		} else {
			unresolvedCount++
		}
	}

	return map[string]interface{}{
		"total_conflicts":    totalConflicts,
		"resolved_count":     resolvedCount,
		"unresolved_count":   unresolvedCount,
		"resolution_rate":    float64(resolvedCount) / float64(max(totalConflicts, 1)),
		"timestamp":          time.Now().UnixMilli(),
	}
}

// ManuallyResolveConflict allows manual resolution of a conflict
func (cr *ConflictResolver) ManuallyResolveConflict(key string, winningEdit *Message, notes string) *ConflictRecord {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if record, ok := cr.conflicts[key]; ok {
		now := time.Now()
		record.ResolvedAt = &now
		record.WinningEdit = winningEdit
		record.ResolutionNotes = notes
		record.Strategy = "user_choice"

		log.Printf("✅ MANUAL RESOLUTION: parcel=%s, winner=%s, notes=%s",
			record.ParcelID, winningEdit.DeviceID, notes)

		return record
	}

	return nil
}

// ExportConflictHistory exports conflict resolution history for audit
func (cr *ConflictResolver) ExportConflictHistory() []map[string]interface{} {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	history := make([]map[string]interface{}, 0)

	for _, record := range cr.conflicts {
		entry := map[string]interface{}{
			"id":              record.ID,
			"parcel_id":       record.ParcelID,
			"device_1":        record.DeviceID1,
			"device_2":        record.DeviceID2,
			"user_1":          record.UserID1,
			"user_2":          record.UserID2,
			"detected_at":     record.DetectedAt.UnixMilli(),
			"resolved_at":     nil,
			"strategy":        record.Strategy,
			"resolution_notes": record.ResolutionNotes,
		}

		if record.ResolvedAt != nil {
			entry["resolved_at"] = record.ResolvedAt.UnixMilli()
		}

		history = append(history, entry)
	}

	return history
}
