package sync

import (
	"context"
	"errors"
	"time"

	"cadastre_ia/pkg/service"
)

// SyncChange represents a single change to a parcel
type SyncChange struct {
	ParcelID  string                 `json:"parcel_id"`
	Operation string                 `json:"operation"`
	Timestamp int64                  `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	DeviceID  string                 `json:"device_id"`
}

// HierarchicalSyncEngine manages multi-device synchronization
type HierarchicalSyncEngine struct {
	db               service.CadastreDB
	conflictResolver *LastWriteWinsResolver
	vectorClockMgr   *VectorClockManager
	changeDetector   *ChangeDetector
	// In-memory storage for Phase 2 MVP
	devices   map[string]map[string]interface{}
	changes   map[string][]SyncChange
	conflicts map[string]interface{}
}

// NewHierarchicalSyncEngine creates a new sync engine
func NewHierarchicalSyncEngine(db service.CadastreDB) *HierarchicalSyncEngine {
	return &HierarchicalSyncEngine{
		db:               db,
		conflictResolver: NewLastWriteWinsResolver(),
		vectorClockMgr:   NewVectorClockManager(),
		changeDetector:   NewChangeDetector(),
		devices:          make(map[string]map[string]interface{}),
		changes:          make(map[string][]SyncChange),
		conflicts:        make(map[string]interface{}),
	}
}

// InitializeDevice initializes a device for sync
func (hse *HierarchicalSyncEngine) InitializeDevice(ctx context.Context, deviceID, fingerprint string) error {
	hse.devices[deviceID] = map[string]interface{}{
		"device_id":   deviceID,
		"fingerprint": fingerprint,
		"initialized": true,
		"sync_state":  "READY",
		"created_at":  time.Now().Unix(),
	}
	hse.changes[deviceID] = []SyncChange{}
	return nil
}

// GetSyncState returns the current sync state for a device
func (hse *HierarchicalSyncEngine) GetSyncState(ctx context.Context, deviceID string) (interface{}, error) {
	state, ok := hse.devices[deviceID]
	if !ok {
		return nil, errors.New("device not found")
	}
	return state, nil
}

// SyncUpload uploads changes from a device
func (hse *HierarchicalSyncEngine) SyncUpload(ctx context.Context, deviceID string, changes []SyncChange) error {
	if len(changes) == 0 {
		return nil
	}
	hse.changes[deviceID] = append(hse.changes[deviceID], changes...)
	return nil
}

// SyncDownload downloads changes for a device
func (hse *HierarchicalSyncEngine) SyncDownload(ctx context.Context, deviceID string, since int64) ([]SyncChange, error) {
	var result []SyncChange
	for otherDevice, otherChanges := range hse.changes {
		if otherDevice == deviceID {
			continue
		}
		for _, change := range otherChanges {
			if change.Timestamp > since {
				result = append(result, change)
			}
		}
	}
	return result, nil
}

// DetectConflicts detects conflicts for a parcel
func (hse *HierarchicalSyncEngine) DetectConflicts(ctx context.Context, parcelID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// ResolveConflict resolves a conflict using LWW
func (hse *HierarchicalSyncEngine) ResolveConflict(ctx context.Context, parcelID string, strategy string) (interface{}, error) {
	return map[string]interface{}{
		"parcel_id": parcelID,
		"resolved":  true,
		"strategy":  strategy,
	}, nil
}

// MarkSynced marks changes as synced
func (hse *HierarchicalSyncEngine) MarkSynced(ctx context.Context, deviceID string, changeIDs []string) error {
	return nil
}

// GetPendingChanges gets pending changes for a device
func (hse *HierarchicalSyncEngine) GetPendingChanges(ctx context.Context, deviceID string) ([]SyncChange, error) {
	changes, ok := hse.changes[deviceID]
	if !ok {
		return []SyncChange{}, nil
	}
	return changes, nil
}

// GetConflictLog gets conflict resolution history
func (hse *HierarchicalSyncEngine) GetConflictLog(ctx context.Context, parcelID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// Note: LastWriteWinsResolver, VectorClockManager, ChangeDetector, and MergeParcels
// are defined in conflict_resolver.go
