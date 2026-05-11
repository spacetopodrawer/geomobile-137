// Package cache provides caching implementations
package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// MemoryCache is an in-memory cache implementation with TTL support
type MemoryCache struct {
	data   map[string]*cacheEntry
	mu     sync.RWMutex
	logger *log.Logger
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(logger *log.Logger) *MemoryCache {
	if logger == nil {
		logger = log.New(log.Writer(), "[MemoryCache] ", log.LstdFlags)
	}

	cache := &MemoryCache{
		data:   make(map[string]*cacheEntry),
		logger: logger,
	}

	// Start cleanup goroutine to remove expired entries
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a value from cache
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		m.mu.RUnlock()
		m.Delete(ctx, key) // Remove expired entry
		m.mu.RLock()
		return nil, fmt.Errorf("key expired: %s", key)
	}

	m.logger.Printf("Cache HIT: %s\n", key)
	return entry.value, nil
}

// Set stores a value in cache with TTL (in seconds)
func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ttlSeconds <= 0 {
		ttlSeconds = 3600 // Default 1 hour
	}

	m.data[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	m.logger.Printf("Cache SET: %s (TTL: %ds)\n", key, ttlSeconds)
	return nil
}

// Delete removes a value from cache
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	m.logger.Printf("Cache DELETE: %s\n", key)
	return nil
}

// Clear removes all entries from cache
func (m *MemoryCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]*cacheEntry)
	m.logger.Println("Cache CLEAR: All entries removed")
	return nil
}

// cleanupExpired periodically removes expired entries
func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		count := 0

		for key, entry := range m.data {
			if now.After(entry.expiresAt) {
				delete(m.data, key)
				count++
			}
		}

		if count > 0 {
			m.logger.Printf("Cleanup: Removed %d expired entries\n", count)
		}
		m.mu.Unlock()
	}
}

// GetStats returns cache statistics
func (m *MemoryCache) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_entries": len(m.data),
	}
}
