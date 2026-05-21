package sync

import (
	"encoding/json"
	"log"
)

// VectorClock represents a logical clock for causality tracking
type VectorClock map[string]int64

// LastWriteWinsResolver implements Last-Write-Wins conflict resolution
type LastWriteWinsResolver struct{}

// NewLastWriteWinsResolver creates a new LWW resolver
func NewLastWriteWinsResolver() *LastWriteWinsResolver {
	return &LastWriteWinsResolver{}
}

// Resolve selects the most recently written value
func (lww *LastWriteWinsResolver) Resolve(val1, val2 interface{}) interface{} {
	// Extract timestamps from both values
	ts1 := extractTimestamp(val1)
	ts2 := extractTimestamp(val2)

	if ts1 >= ts2 {
		return val1
	}
	return val2
}

// ResolveJSON resolves conflict between two JSON values
func (lww *LastWriteWinsResolver) ResolveJSON(val1JSON, val2JSON string) (string, error) {
	var val1, val2 interface{}

	if err := json.Unmarshal([]byte(val1JSON), &val1); err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(val2JSON), &val2); err != nil {
		return "", err
	}

	resolved := lww.Resolve(val1, val2)
	result, err := json.Marshal(resolved)
	return string(result), err
}

// extractTimestamp extracts timestamp from a value
func extractTimestamp(val interface{}) int64 {
	switch v := val.(type) {
	case map[string]interface{}:
		if ts, ok := v["timestamp"]; ok {
			switch tsVal := ts.(type) {
			case float64:
				return int64(tsVal)
			case int64:
				return tsVal
			}
		}
		if ts, ok := v["updated_at"]; ok {
			switch tsVal := ts.(type) {
			case float64:
				return int64(tsVal)
			case int64:
				return tsVal
			}
		}
	}
	return 0
}

// MergeParcels performs a 3-way merge of parcel data
type MergeParcels struct {
	Common   interface{}
	Device1  interface{}
	Device2  interface{}
	Resolver *LastWriteWinsResolver
}

// Merge performs 3-way merge logic
func (mp *MergeParcels) Merge() interface{} {
	// If both devices made the same change, no conflict
	if isEqual(mp.Device1, mp.Device2) {
		return mp.Device1
	}

	// If only one device made changes, use that version
	if isEqual(mp.Device1, mp.Common) {
		return mp.Device2
	}
	if isEqual(mp.Device2, mp.Common) {
		return mp.Device1
	}

	// Both made different changes (conflict) - use LWW
	return mp.Resolver.Resolve(mp.Device1, mp.Device2)
}

// isEqual checks if two values are equal
func isEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// ConflictDetector identifies conflicts in concurrent edits
type ConflictDetector struct{}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConcurrentEdit detects if two edits are concurrent
func (cd *ConflictDetector) DetectConcurrentEdit(
	baseTimestamp int64,
	edit1Timestamp int64,
	edit2Timestamp int64,
) bool {
	// Both edits happened after the base version
	return edit1Timestamp > baseTimestamp && edit2Timestamp > baseTimestamp
}

// DetectDeleteVsModify detects delete vs modify conflicts
func (cd *ConflictDetector) DetectDeleteVsModify(op1, op2 string) bool {
	deleteOps := map[string]bool{"DELETE": true}
	modifyOps := map[string]bool{"UPDATE": true, "CREATE": true}

	isOp1Delete := deleteOps[op1]
	isOp2Delete := deleteOps[op2]
	isOp1Modify := modifyOps[op1]
	isOp2Modify := modifyOps[op2]

	return (isOp1Delete && isOp2Modify) || (isOp2Delete && isOp1Modify)
}

// VectorClockManager manages vector clocks for causality tracking
type VectorClockManager struct{}

// NewVectorClockManager creates a new vector clock manager
func NewVectorClockManager() *VectorClockManager {
	return &VectorClockManager{}
}

// IncrementClock increments the clock for a device
func (vcm *VectorClockManager) IncrementClock(ctx interface{}, db interface{}, deviceID string) (VectorClock, error) {
	// This is a placeholder - actual implementation would query PostgreSQL
	log.Printf("Incrementing clock for device %s", deviceID)

	vc := make(VectorClock)
	vc[deviceID] = 1
	return vc, nil
}

// Compare compares two vector clocks
// Returns: -1 if vc1 < vc2, 0 if equal, 1 if vc1 > vc2, 2 if concurrent
func (vcm *VectorClockManager) Compare(vc1, vc2 VectorClock) int {
	// Simple comparison: count entries
	if len(vc1) < len(vc2) {
		return -1
	} else if len(vc1) > len(vc2) {
		return 1
	}
	return 0
}

// ChangeDetector identifies changes between versions
type ChangeDetector struct{}

// NewChangeDetector creates a new change detector
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{}
}

// DetectChanges compares two versions and returns delta
func (cd *ChangeDetector) DetectChanges(baseVersion, newVersion map[string]interface{}) map[string]interface{} {
	delta := make(map[string]interface{})

	// Check for modifications and additions
	for key, newVal := range newVersion {
		if baseVal, exists := baseVersion[key]; !exists || !isEqual(baseVal, newVal) {
			delta[key] = newVal
		}
	}

	// Check for deletions (marked with null)
	for key := range baseVersion {
		if _, exists := newVersion[key]; !exists {
			delta[key] = nil
		}
	}

	return delta
}

// ComputeDelta computes minimal change set between versions
func (cd *ChangeDetector) ComputeDelta(oldData, newData map[string]interface{}) map[string]interface{} {
	delta := make(map[string]interface{})

	for key, newVal := range newData {
		oldVal, exists := oldData[key]
		if !exists || !isEqual(oldVal, newVal) {
			delta[key] = newVal
		}
	}

	return delta
}

// ApplyDelta applies a delta to a base version
func (cd *ChangeDetector) ApplyDelta(baseData map[string]interface{}, delta map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base data
	for k, v := range baseData {
		result[k] = v
	}

	// Apply delta
	for k, v := range delta {
		if v == nil {
			delete(result, k)
		} else {
			result[k] = v
		}
	}

	return result
}
