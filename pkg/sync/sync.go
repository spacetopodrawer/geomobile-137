package sync

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SyncMessage represents a peer-to-peer sync message
type SyncMessage struct {
	Type      string          `json:"type"` // "operation", "request", "response", "conflict"
	MessageID string          `json:"message_id"`
	Timestamp int64           `json:"timestamp"` // Unix milliseconds
	DeviceID  string          `json:"device_id"`
	Version   int32           `json:"version"`

	// For operation messages
	Operation *Operation `json:"operation,omitempty"`

	// For conflict messages
	Conflict *ConflictInfo `json:"conflict,omitempty"`

	// Vector clock for causal ordering
	VectorClock map[string]int32 `json:"vector_clock"`
}

// Operation represents a single sync operation
type Operation struct {
	ID              string          `json:"id"`
	ObjectID        string          `json:"object_id"`
	Type            string          `json:"type"` // "create", "update", "delete"
	Timestamp       int64           `json:"timestamp"`
	DeviceID        string          `json:"device_id"`
	BeforeState     json.RawMessage `json:"before_state,omitempty"`
	AfterState      json.RawMessage `json:"after_state,omitempty"`
	VectorClock     map[string]int32 `json:"vector_clock"`
	ParentOperation *string         `json:"parent_operation,omitempty"` // For causal ordering
}

// ConflictInfo represents a detected conflict
type ConflictInfo struct {
	ObjectID        string     `json:"object_id"`
	OperationA      *Operation `json:"operation_a"`
	OperationB      *Operation `json:"operation_b"`
	ConflictType    string     `json:"conflict_type"` // "concurrent_edit", "concurrent_delete", etc.
	Resolution      string     `json:"resolution"`    // "pending", "ot", "merge", "user_choice"
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// VectorClock for causality tracking in distributed system
type VectorClock struct {
	mu    sync.RWMutex
	clock map[string]int32
}

// NewVectorClock creates a new vector clock
func NewVectorClock(deviceID string) *VectorClock {
	return &VectorClock{
		clock: map[string]int32{
			deviceID: 0,
		},
	}
}

// Increment increments the clock for a device
func (vc *VectorClock) Increment(deviceID string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.clock[deviceID]++
}

// Update updates the clock based on received clock
func (vc *VectorClock) Update(received map[string]int32) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	for device, time := range received {
		if currentTime, exists := vc.clock[device]; exists {
			if time > currentTime {
				vc.clock[device] = time
			}
		} else {
			vc.clock[device] = time
		}
	}

	// Increment our own clock
	for device := range vc.clock {
		vc.clock[device]++
	}
}

// Get returns a copy of the current clock
func (vc *VectorClock) Get() map[string]int32 {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	result := make(map[string]int32)
	for k, v := range vc.clock {
		result[k] = v
	}
	return result
}

// Happens-before relationship
// Returns: -1 if a < b, 0 if concurrent, 1 if a > b
func HappensBefore(clockA, clockB map[string]int32) int {
	lessOrEqual := true
	greaterOrEqual := true

	for device, timeB := range clockB {
		timeA := clockA[device]
		if timeA > timeB {
			lessOrEqual = false
		}
		if timeA < timeB {
			greaterOrEqual = false
		}
	}

	// Check devices only in A
	for device, timeA := range clockA {
		if _, exists := clockB[device]; !exists {
			if timeA > 0 {
				greaterOrEqual = false
			}
		}
	}

	if lessOrEqual && !greaterOrEqual {
		return -1 // a < b
	} else if greaterOrEqual && !lessOrEqual {
		return 1 // a > b
	}
	return 0 // concurrent
}

// OperationalTransform - Conflict resolution algorithm
type OperationalTransform struct {
	mu         sync.RWMutex
	operations map[string]*Operation // Indexed by operation ID
	history    []*Operation
}

// NewOperationalTransform creates a new OT engine
func NewOperationalTransform() *OperationalTransform {
	return &OperationalTransform{
		operations: make(map[string]*Operation),
		history:    make([]*Operation, 0),
	}
}

// ApplyOperation applies an operation with conflict detection
func (ot *OperationalTransform) ApplyOperation(op *Operation) (string, error) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// Check for conflicts with recent operations on same object
	for _, existing := range ot.history {
		if existing.ObjectID == op.ObjectID {
			// Check if operations are concurrent
			relation := HappensBefore(existing.VectorClock, op.VectorClock)
			if relation == 0 {
				// Concurrent operations detected - need transformation
				return "conflict_detected", nil
			}
		}
	}

	// No conflict - add to history
	ot.operations[op.ID] = op
	ot.history = append(ot.history, op)

	return "applied", nil
}

// Transform transforms operation B against operation A
// Used for conflict resolution when operations are concurrent
func (ot *OperationalTransform) Transform(opA, opB *Operation) *Operation {
	// For attribute updates, transform by taking the later operation
	if opA.Type == "update" && opB.Type == "update" && opA.ObjectID == opB.ObjectID {
		// Last-write-wins based on vector clock
		relation := HappensBefore(opA.VectorClock, opB.VectorClock)
		if relation < 0 {
			// opA happened before, keep opB as is
			return opB
		}
		// opB happened first or concurrent, take opA
		return opA
	}

	// For geometric/spatial operations, merge if possible
	// This would require domain-specific merging logic
	return opB
}

// GetHistory returns operation history (for sync)
func (ot *OperationalTransform) GetHistory(since int64) []*Operation {
	ot.mu.RLock()
	defer ot.mu.RUnlock()

	result := make([]*Operation, 0)
	for _, op := range ot.history {
		if op.Timestamp > since {
			result = append(result, op)
		}
	}
	return result
}

// SyncState represents the state of synchronization with a peer
type SyncState struct {
	DeviceID      string
	LastSync      time.Time
	VectorClock   map[string]int32
	PendingOps    []*Operation
	UnresolvedConflicts []*ConflictInfo
}

// SyncManager manages peer-to-peer synchronization
type SyncManager struct {
	mu              sync.RWMutex
	deviceID        string
	vectorClock     *VectorClock
	ot              *OperationalTransform
	peerStates      map[string]*SyncState
	operationQueue  chan *Operation
	syncHandlers    []SyncHandler
	conflictHandler ConflictHandler
}

// SyncHandler is called when sync occurs
type SyncHandler func(msg *SyncMessage) error

// ConflictHandler handles conflict resolution
type ConflictHandler func(conflict *ConflictInfo) (string, error) // Returns resolution method

// NewSyncManager creates a new sync manager
func NewSyncManager(deviceID string) *SyncManager {
	return &SyncManager{
		deviceID:        deviceID,
		vectorClock:     NewVectorClock(deviceID),
		ot:              NewOperationalTransform(),
		peerStates:      make(map[string]*SyncState),
		operationQueue:  make(chan *Operation, 1000),
		syncHandlers:    make([]SyncHandler, 0),
	}
}

// RegisterSyncHandler registers a handler for sync events
func (sm *SyncManager) RegisterSyncHandler(handler SyncHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.syncHandlers = append(sm.syncHandlers, handler)
}

// SetConflictHandler sets the conflict resolution handler
func (sm *SyncManager) SetConflictHandler(handler ConflictHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.conflictHandler = handler
}

// CreateOperation creates and enqueues a new operation
func (sm *SyncManager) CreateOperation(objID string, opType string, before, after json.RawMessage) *Operation {
	sm.vectorClock.Increment(sm.deviceID)

	op := &Operation{
		ID:          uuid.New().String(),
		ObjectID:    objID,
		Type:        opType,
		Timestamp:   time.Now().UnixMilli(),
		DeviceID:    sm.deviceID,
		BeforeState: before,
		AfterState:  after,
		VectorClock: sm.vectorClock.Get(),
	}

	// Enqueue for processing
	sm.operationQueue <- op

	return op
}

// ProcessOperations processes queued operations
func (sm *SyncManager) ProcessOperations() {
	for op := range sm.operationQueue {
		status, err := sm.ot.ApplyOperation(op)

		if status == "conflict_detected" {
			// Handle conflict
			sm.handleConflict(op)
		} else if err == nil {
			// Broadcast to peers
			sm.broadcastOperation(op)
		}
	}
}

// ReceiveMessage processes an incoming sync message
func (sm *SyncManager) ReceiveMessage(msg *SyncMessage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update vector clock from message
	sm.vectorClock.Update(msg.VectorClock)

	// Update peer state
	if _, exists := sm.peerStates[msg.DeviceID]; !exists {
		sm.peerStates[msg.DeviceID] = &SyncState{
			DeviceID:    msg.DeviceID,
			LastSync:    time.Now(),
			VectorClock: msg.VectorClock,
		}
	} else {
		sm.peerStates[msg.DeviceID].LastSync = time.Now()
		sm.peerStates[msg.DeviceID].VectorClock = msg.VectorClock
	}

	// Process based on message type
	switch msg.Type {
	case "operation":
		sm.ot.ApplyOperation(msg.Operation)
		return nil

	case "conflict":
		if sm.conflictHandler != nil {
			resolution, _ := sm.conflictHandler(msg.Conflict)
			msg.Conflict.Resolution = resolution
		}
		return nil
	}

	return nil
}

// broadcastOperation broadcasts operation to all handlers
func (sm *SyncManager) broadcastOperation(op *Operation) {
	msg := &SyncMessage{
		Type:        "operation",
		MessageID:   uuid.New().String(),
		Timestamp:   time.Now().UnixMilli(),
		DeviceID:    sm.deviceID,
		Version:     1,
		Operation:   op,
		VectorClock: sm.vectorClock.Get(),
	}

	for _, handler := range sm.syncHandlers {
		go handler(msg)
	}
}

// handleConflict handles a detected conflict
func (sm *SyncManager) handleConflict(op *Operation) {
	// Find conflicting operation
	for _, existing := range sm.ot.history {
		if existing.ObjectID == op.ObjectID {
			relation := HappensBefore(existing.VectorClock, op.VectorClock)
			if relation == 0 {
				conflict := &ConflictInfo{
					ObjectID:   op.ObjectID,
					OperationA: existing,
					OperationB: op,
					ConflictType: "concurrent_edit",
					Resolution: "pending",
				}

				// Use handler if available
				if sm.conflictHandler != nil {
					resolution, _ := sm.conflictHandler(conflict)
					conflict.Resolution = resolution
				}

				// Broadcast conflict to peers
				msg := &SyncMessage{
					Type:        "conflict",
					MessageID:   uuid.New().String(),
					Timestamp:   time.Now().UnixMilli(),
					DeviceID:    sm.deviceID,
					Conflict:    conflict,
					VectorClock: sm.vectorClock.Get(),
				}

				for _, handler := range sm.syncHandlers {
					go handler(msg)
				}

				return
			}
		}
	}
}

// GetSyncStatus returns current sync status
func (sm *SyncManager) GetSyncStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"device_id":     sm.deviceID,
		"vector_clock":  sm.vectorClock.Get(),
		"peer_count":    len(sm.peerStates),
		"queue_size":    len(sm.operationQueue),
		"history_size":  len(sm.ot.history),
	}
}
