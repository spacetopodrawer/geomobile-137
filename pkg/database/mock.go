// Package database provides mock implementations for testing
package database

import (
	"context"
	"fmt"
	"log"
	"sync"

	"cadastre_ia/pkg/cadastre"
)

// MockDB is an in-memory implementation of CadastreDB for testing
type MockDB struct {
	entities map[string]*cadastre.Entity
	mu       sync.RWMutex
	logger   *log.Logger
}

// NewMockDB creates a new mock database
func NewMockDB(logger *log.Logger) *MockDB {
	if logger == nil {
		logger = log.New(log.Writer(), "[MockDB] ", log.LstdFlags)
	}
	return &MockDB{
		entities: make(map[string]*cadastre.Entity),
		logger:   logger,
	}
}

// StoreCadastralEntities stores entities in memory
func (m *MockDB) StoreCadastralEntities(ctx context.Context, entities []cadastre.Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(entities) == 0 {
		return fmt.Errorf("no entities to store")
	}

	for _, entity := range entities {
		entityCopy := entity
		m.entities[entity.Code] = &entityCopy
		m.logger.Printf("  Stored (mock): %s → %s\n", entity.ID, entity.Code)
	}

	m.logger.Printf("Stored %d entities in mock DB\n", len(entities))
	return nil
}

// GetEntity retrieves an entity by code
func (m *MockDB) GetEntity(ctx context.Context, code string) (*cadastre.Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entity, exists := m.entities[code]
	if !exists {
		return nil, fmt.Errorf("entity not found: %s", code)
	}

	return entity, nil
}

// UpdateEntity updates an entity
func (m *MockDB) UpdateEntity(ctx context.Context, entity *cadastre.Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entities[entity.Code]; !exists {
		return fmt.Errorf("entity not found: %s", entity.Code)
	}

	m.entities[entity.Code] = entity
	m.logger.Printf("Updated entity (mock): %s\n", entity.Code)
	return nil
}

// DeleteEntity soft-deletes an entity
func (m *MockDB) DeleteEntity(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entities[code]; !exists {
		return fmt.Errorf("entity not found: %s", code)
	}

	delete(m.entities, code)
	m.logger.Printf("Deleted entity (mock): %s\n", code)
	return nil
}

// GetEntitiesByBBox retrieves entities within a bbox (simplified mock)
func (m *MockDB) GetEntitiesByBBox(ctx context.Context, minX, minY, maxX, maxY float64, zoomLevel int) ([]cadastre.Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entities := make([]cadastre.Entity, 0)
	for _, entity := range m.entities {
		entities = append(entities, *entity)
	}

	m.logger.Printf("Retrieved %d entities for mock bbox z=%d\n", len(entities), zoomLevel)
	return entities, nil
}

// GetCount returns total entity count
func (m *MockDB) GetCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entities)
}

// Clear removes all entities (for testing)
func (m *MockDB) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entities = make(map[string]*cadastre.Entity)
	m.logger.Println("Cleared mock DB")
}
