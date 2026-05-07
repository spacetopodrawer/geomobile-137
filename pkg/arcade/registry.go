package arcade

import (
	"fmt"
	"sync"
)

// SystemRegistry manages available arcade systems and factory functions
type SystemRegistry struct {
	mu        sync.RWMutex
	systems   map[string]SystemFactory
	instances map[string]ArcadeEmulator
	specs     map[string]*SystemSpec
}

var globalRegistry *SystemRegistry

// InitRegistry initializes the global system registry
func InitRegistry() {
	globalRegistry = &SystemRegistry{
		systems:   make(map[string]SystemFactory),
		instances: make(map[string]ArcadeEmulator),
		specs:     make(map[string]*SystemSpec),
	}
}

// RegisterSystem registers a system factory
func RegisterSystem(systemID string, factory SystemFactory, spec *SystemSpec) error {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.systems[systemID]; exists {
		return fmt.Errorf("system %q already registered", systemID)
	}

	globalRegistry.systems[systemID] = factory
	globalRegistry.specs[systemID] = spec
	return nil
}

// GetArcadeEmulator creates or returns a cached emulator instance
func GetArcadeEmulator(systemID string, basePort int, romPath string) (ArcadeEmulator, error) {
	globalRegistry.mu.RLock()
	factory, exists := globalRegistry.systems[systemID]
	globalRegistry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("system %q not registered", systemID)
	}

	instanceKey := fmt.Sprintf("%s:%d", systemID, basePort)

	globalRegistry.mu.RLock()
	if instance, cached := globalRegistry.instances[instanceKey]; cached {
		globalRegistry.mu.RUnlock()
		return instance, nil
	}
	globalRegistry.mu.RUnlock()

	emulator, err := factory(basePort, romPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create %q emulator: %w", systemID, err)
	}

	globalRegistry.mu.Lock()
	globalRegistry.instances[instanceKey] = emulator
	globalRegistry.mu.Unlock()

	return emulator, nil
}

// ListAvailableSystems returns all registered system IDs
func ListAvailableSystems() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	systems := make([]string, 0, len(globalRegistry.systems))
	for systemID := range globalRegistry.systems {
		systems = append(systems, systemID)
	}
	return systems
}

// GetSystemSpec returns specification for a system
func GetSystemSpec(systemID string) *SystemSpec {
	// First check registry (for registered systems)
	globalRegistry.mu.RLock()
	if spec, ok := globalRegistry.specs[systemID]; ok {
		globalRegistry.mu.RUnlock()
		return spec
	}
	globalRegistry.mu.RUnlock()

	// Fall back to built-in specs from system.go
	return getSystemSpec(systemID)
}

// DetectSystemFromROM attempts to detect system from ROM file magic bytes
func DetectSystemFromROM(romPath string) (string, error) {
	// Placeholder: Actual detection based on file extension and magic bytes
	// Will be implemented in each system's compiler
	return "", fmt.Errorf("ROM detection not yet implemented")
}

// GetRegistry returns the global registry instance
func GetRegistry() *SystemRegistry {
	if globalRegistry == nil {
		InitRegistry()
	}
	return globalRegistry
}

// ClearRegistry clears all registered systems and instances (for testing)
func ClearRegistry() {
	globalRegistry = nil
	InitRegistry()
}
