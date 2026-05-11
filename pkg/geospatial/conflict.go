package geospatial

import (
	"log"
	"time"
)

type ConflictStrategy int

const (
	OperationalTransform ConflictStrategy = iota
	LastWriteWins
	ManualResolution
)

type ConflictResolver struct {
	strategy ConflictStrategy
}

func NewConflictResolver(strategy ConflictStrategy) *ConflictResolver {
	return &ConflictResolver{strategy: strategy}
}

func (cr *ConflictResolver) Resolve(op1, op2 interface{}, ts1, ts2 int64) interface{} {
	log.Printf("🔄 Resolving conflict: strategy=%v ts1=%d ts2=%d", cr.strategy, ts1, ts2)

	switch cr.strategy {
	case LastWriteWins:
		if ts2 > ts1 {
			log.Printf("✅ Resolved via Last-Write-Wins: op2 wins (ts2=%d > ts1=%d)", ts2, ts1)
			return op2
		}
		log.Printf("✅ Resolved via Last-Write-Wins: op1 wins (ts1=%d >= ts2=%d)", ts1, ts2)
		return op1

	case OperationalTransform:
		log.Printf("⚠️  Operational Transform not yet implemented, using Last-Write-Wins")
		if ts2 > ts1 {
			return op2
		}
		return op1

	case ManualResolution:
		log.Printf("⚠️  Manual resolution required")
		return nil

	default:
		return op1
	}
}

func CanMerge(op1, op2 map[string]interface{}) bool {
	keys1 := make(map[string]bool)
	for k := range op1 {
		keys1[k] = true
	}

	for k := range op2 {
		if keys1[k] {
			return false
		}
	}
	return true
}
