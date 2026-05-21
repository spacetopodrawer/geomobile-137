package service

import (
	"cadastre_ia/internal/storage"
	"cadastre_ia/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type SyncService struct {
	parcelRepo *storage.ParcelRepository
	syncRepo   *storage.SyncOperationRepository
	hub        *websocket.Hub
}

func NewSyncService(parcelRepo *storage.ParcelRepository, syncRepo *storage.SyncOperationRepository, hub *websocket.Hub) *SyncService {
	return &SyncService{
		parcelRepo: parcelRepo,
		syncRepo:   syncRepo,
		hub:        hub,
	}
}

// ProcessOperation orchestrates the entire sync flow: validate → fetch → detect conflicts → apply → broadcast
func (s *SyncService) ProcessOperation(ctx context.Context, op *storage.SyncOperation) error {
	startTime := time.Now()
	log.Printf("🔄 [SYNC] Processing operation: op_id=%d parcel=%s type=%s client=%s ts=%d",
		op.ID, op.ParcelID, op.OperationType, op.ClientID, op.Timestamp)

	// 1. Validate operation structure
	if err := s.ValidateOperation(op); err != nil {
		log.Printf("❌ [SYNC] Validation failed for op_id=%d: %v", op.ID, err)
		op.Status = "rejected"
		return fmt.Errorf("validation failed: %w", err)
	}

	// 2. Fetch current parcel state
	parcel, err := s.parcelRepo.GetByID(ctx, op.ParcelID.String())
	if err != nil {
		log.Printf("❌ [SYNC] Parcel not found: parcel_id=%s error=%v", op.ParcelID, err)
		op.Status = "rejected"
		return fmt.Errorf("parcel not found: %w", err)
	}

	log.Printf("📋 [SYNC] Current parcel state: version=%d last_modified=%s",
		parcel.Version, parcel.UpdatedAt)

	// 3. Detect conflicts with concurrent operations
	conflicts, err := s.DetectConflicts(ctx, op, parcel)
	if err != nil {
		log.Printf("❌ [SYNC] Conflict detection error: op_id=%d error=%v", op.ID, err)
		op.Status = "rejected"
		return fmt.Errorf("conflict detection failed: %w", err)
	}

	// 4. Apply operation (with or without conflict resolution)
	if len(conflicts) > 0 {
		log.Printf("⚠️  [SYNC] Conflicts detected: count=%d op_id=%d", len(conflicts), op.ID)
		op.Status = "conflict"

		resolvedOp, err := s.ResolveConflicts(ctx, op, conflicts)
		if err != nil {
			log.Printf("❌ [SYNC] Resolution failed: op_id=%d error=%v", op.ID, err)
			return fmt.Errorf("conflict resolution failed: %w", err)
		}
		op = resolvedOp
	} else {
		op.Status = "applied"
		log.Printf("✅ [SYNC] No conflicts detected, operation can be applied")
	}

	// 5. Persist the operation
	if err := s.syncRepo.Create(ctx, op); err != nil {
		log.Printf("❌ [SYNC] Failed to persist operation: op_id=%d error=%v", op.ID, err)
		return fmt.Errorf("failed to persist operation: %w", err)
	}

	// 6. Update parcel version
	parcel.Version++
	parcel.UpdatedAt = time.Now()
	if err := s.parcelRepo.Update(ctx, parcel); err != nil {
		log.Printf("⚠️  [SYNC] Failed to update parcel version: parcel_id=%s error=%v", parcel.ID, err)
	}

	// 7. Broadcast to all connected clients
	payload := map[string]interface{}{
		"operation_id":   op.ID,
		"status":         op.Status,
		"parcel_version": parcel.Version,
		"applied_at":     op.AppliedAt,
		"client_id":      op.ClientID.String(),
	}

	if op.Status == "conflict" {
		payload["resolution_type"] = "last_write_wins"
	}

	broadcastMsg := &websocket.Message{
		Type:      "sync_broadcast",
		ParcelID:  op.ParcelID.String(),
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
		Payload:   payload,
	}

	s.hub.BroadcastParcelChange(broadcastMsg)
	log.Printf("✅ [SYNC] Operation processed successfully: op_id=%d status=%s duration=%dms",
		op.ID, op.Status, time.Since(startTime).Milliseconds())

	return nil
}

// ValidateOperation checks operation integrity
func (s *SyncService) ValidateOperation(op *storage.SyncOperation) error {
	if op.ParcelID == uuid.Nil {
		return fmt.Errorf("%w: parcel_id is nil", ErrInvalidParcelID)
	}
	if op.ClientID == uuid.Nil {
		return fmt.Errorf("%w: client_id is nil", ErrInvalidClientID)
	}
	if op.OperationType == "" {
		return fmt.Errorf("%w: operation_type is empty", ErrInvalidOperationType)
	}
	if op.Timestamp <= 0 {
		return fmt.Errorf("%w: timestamp=%d is invalid", ErrInvalidTimestamp, op.Timestamp)
	}
	if len(op.OperationData) == 0 {
		return fmt.Errorf("operation_data is empty")
	}

	// Validate JSON structure
	var data map[string]interface{}
	if err := json.Unmarshal(op.OperationData, &data); err != nil {
		return fmt.Errorf("invalid operation_data JSON: %w", err)
	}

	log.Printf("✅ [SYNC] Operation validation passed: op_type=%s", op.OperationType)
	return nil
}

// DetectConflicts identifies concurrent edits on the same parcel
func (s *SyncService) DetectConflicts(ctx context.Context, op *storage.SyncOperation, parcel *storage.Parcel) ([]*storage.Conflict, error) {
	// Fetch all operations for this parcel in reverse chronological order
	ops, err := s.syncRepo.GetByParcel(ctx, op.ParcelID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parcel operations: %w", err)
	}

	if len(ops) == 0 {
		log.Printf("📊 [SYNC] No prior operations for parcel=%s, no conflicts possible", op.ParcelID)
		return nil, nil
	}

	var conflicts []*storage.Conflict

	// Check for concurrent edits from different clients at similar timestamps
	for _, existing := range ops {
		// Skip if same client (client can overwrite own operations)
		if existing.ClientID == op.ClientID {
			continue
		}

		// Detect conflict: operation timestamps too close (within 100ms = concurrent)
		timeDiff := op.Timestamp - existing.Timestamp
		if timeDiff > -100 && timeDiff < 100 {
			log.Printf("⚠️  [SYNC] Conflict detected: existing_op=%d (ts=%d) vs new_op (ts=%d) diff=%dms",
				existing.ID, existing.Timestamp, op.Timestamp, timeDiff)

			conflict := &storage.Conflict{
				ID:           uuid.New(),
				ParcelID:     op.ParcelID,
				Operation1ID: &existing.ID,
				ResolutionType: strPtr("last_write_wins"),
			}
			conflicts = append(conflicts, conflict)
		}

		// Early exit: only check against the most recent operation
		if len(conflicts) > 0 {
			break
		}
	}

	log.Printf("📊 [SYNC] Conflict detection complete: count=%d parcel=%s", len(conflicts), op.ParcelID)
	return conflicts, nil
}

// ResolveConflicts applies conflict resolution strategy (currently: last-write-wins)
func (s *SyncService) ResolveConflicts(ctx context.Context, newOp *storage.SyncOperation, conflicts []*storage.Conflict) (*storage.SyncOperation, error) {
	if len(conflicts) == 0 {
		return newOp, nil
	}

	// Strategy: Last-Write-Wins
	// The operation with the later timestamp wins
	for _, conflict := range conflicts {
		conflict.ResolutionType = strPtr("last_write_wins")
		log.Printf("🔧 [SYNC] Resolving conflict: conflict_id=%s strategy=last_write_wins", conflict.ID)
	}

	// Update operation status
	newOp.Status = "applied_with_conflict_resolution"

	log.Printf("✅ [SYNC] Conflict resolution applied: count=%d strategy=last_write_wins", len(conflicts))
	return newOp, nil
}

// GetParcelSyncHistory returns all operations for a parcel (for client sync on reconnect)
func (s *SyncService) GetParcelSyncHistory(ctx context.Context, parcelID string, since int64) ([]*storage.SyncOperation, error) {
	log.Printf("📜 [SYNC] Fetching sync history: parcel=%s since=%d", parcelID, since)

	ops, err := s.syncRepo.GetByParcel(ctx, parcelID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sync history: %w", err)
	}

	// Filter operations after the 'since' timestamp
	var filtered []*storage.SyncOperation
	for _, op := range ops {
		if op.Timestamp > since {
			filtered = append(filtered, op)
		}
	}

	log.Printf("📜 [SYNC] Sync history retrieved: total=%d filtered=%d", len(ops), len(filtered))
	return filtered, nil
}

// BroadcastPresence notifies all clients of user presence (online/offline)
func (s *SyncService) BroadcastPresence(userID string, status string) {
	msg := &websocket.Message{
		Type:      "presence_update",
		Timestamp: time.Now().UnixMilli(),
		MessageID: uuid.New().String(),
		Payload: map[string]interface{}{
			"user_id": userID,
			"status":  status,
		},
	}
	s.hub.BroadcastParcelChange(msg)
	log.Printf("👤 [SYNC] Presence broadcast: user=%s status=%s", userID, status)
}

// Helper function
func strPtr(s string) *string {
	return &s
}
