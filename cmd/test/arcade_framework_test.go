package test

import (
	"testing"

	"cadastreia/pkg/arcade"
	"cadastreia/pkg/arcade/systems/template"
)

// TestArcadeEmulatorInterface verifies emulator interface compliance
func TestArcadeEmulatorInterface(t *testing.T) {
	emulator, err := template.Factory(9001, "")
	if err != nil {
		t.Fatalf("failed to create template emulator: %v", err)
	}

	// Verify interface compliance (compiler check)
	var _ arcade.ArcadeEmulator = emulator

	// Test lifecycle
	if emulator.IsRunning() {
		t.Error("emulator should not be running initially")
	}

	err = emulator.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !emulator.IsRunning() {
		t.Error("emulator should be running after Start()")
	}

	err = emulator.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if emulator.IsRunning() {
		t.Error("emulator should not be running after Stop()")
	}
}

// TestSystemRegistry verifies registry functions
func TestSystemRegistry(t *testing.T) {
	arcade.InitRegistry()

	// Register template system
	err := arcade.RegisterSystem("template", template.Factory, &arcade.SystemSpec{
		DisplayWidth:  320,
		DisplayHeight: 224,
		RefreshRate:   60,
		ColorDepth:    4,
		MaxColors:     16,
	})
	if err != nil {
		t.Fatalf("RegisterSystem failed: %v", err)
	}

	// Verify it's in the list
	systems := arcade.ListAvailableSystems()
	found := false
	for _, sys := range systems {
		if sys == "template" {
			found = true
			break
		}
	}
	if !found {
		t.Error("registered system not found in list")
	}

	// Attempt to register duplicate
	err = arcade.RegisterSystem("template", template.Factory, nil)
	if err == nil {
		t.Error("should not allow duplicate registration")
	}
}

// TestGetArcadeEmulator verifies factory creation
func TestGetArcadeEmulator(t *testing.T) {
	arcade.InitRegistry()

	arcade.RegisterSystem("template", template.Factory, &arcade.SystemSpec{
		DisplayWidth:  320,
		DisplayHeight: 224,
	})

	emulator, err := arcade.GetArcadeEmulator("template", 9001, "")
	if err != nil {
		t.Fatalf("GetArcadeEmulator failed: %v", err)
	}

	if emulator == nil {
		t.Error("emulator is nil")
	}

	// Verify it implements the interface
	var _ arcade.ArcadeEmulator = emulator
}

// TestGetSystemSpec verifies spec retrieval
func TestGetSystemSpec(t *testing.T) {
	spec := arcade.GetSystemSpec("neogeo")
	if spec == nil {
		t.Error("NEO-GEO spec is nil")
	}

	if spec.DisplayWidth != 320 || spec.DisplayHeight != 224 {
		t.Error("NEO-GEO spec incorrect resolution")
	}

	if spec.RefreshRate != 60 {
		t.Error("NEO-GEO spec refresh rate should be 60 Hz")
	}

	spec = arcade.GetSystemSpec("mame")
	if spec == nil {
		t.Error("MAME spec is nil")
	}

	if spec.DisplayHeight != 240 {
		t.Error("MAME spec height incorrect")
	}
}

// TestSystemInfo verifies system info retrieval
func TestSystemInfo(t *testing.T) {
	info := arcade.GetSystemInfo("neogeo")
	if info == nil {
		t.Error("NEO-GEO info is nil")
	}

	if info.Name != "NEO-GEO" {
		t.Errorf("expected 'NEO-GEO', got '%s'", info.Name)
	}

	if info.SystemID != "neogeo" {
		t.Errorf("expected 'neogeo', got '%s'", info.SystemID)
	}

	if info.Spec == nil {
		t.Error("info.Spec is nil")
	}

	// Verify all expected systems exist
	expectedSystems := []string{"neogeo", "mame", "fbneo", "atari2600", "commodore64"}
	for _, sysID := range expectedSystems {
		if arcade.GetSystemInfo(sysID) == nil {
			t.Errorf("system %q info is nil", sysID)
		}
	}
}

// TestEmulatorMethods verifies basic emulator methods
func TestEmulatorMethods(t *testing.T) {
	emulator, err := template.Factory(9001, "")
	if err != nil {
		t.Fatalf("failed to create emulator: %v", err)
	}

	// Test GetSystem
	sys := emulator.GetSystem()
	if sys == nil {
		t.Error("GetSystem returned nil")
	}

	// Test GetStatus before running
	status := emulator.GetStatus()
	if status == nil {
		t.Error("GetStatus returned nil")
	}

	if running, ok := status["running"].(bool); !ok || running {
		t.Error("status[running] should be false initially")
	}

	// Test Start and GetStatus
	emulator.Start()
	defer emulator.Stop()

	status = emulator.GetStatus()
	if running, ok := status["running"].(bool); !ok || !running {
		t.Error("status[running] should be true after Start()")
	}

	// Test GetFrameCount
	count := emulator.GetFrameCount()
	if count < 0 {
		t.Error("GetFrameCount returned negative value")
	}

	// Test GetFPS
	fps := emulator.GetFPS()
	if fps <= 0 {
		t.Error("GetFPS should be positive")
	}
}

// TestListAvailableSystems verifies system enumeration
func TestListAvailableSystems(t *testing.T) {
	systems := arcade.ListAvailableSystems()
	if systems == nil {
		t.Error("ListAvailableSystems returned nil")
	}

	// Should have at least the core systems defined
	if len(systems) < 5 {
		t.Logf("Expected at least 5 systems, got %d: %v", len(systems), systems)
	}
}

// TestGameFrameTypes verifies frame structure
func TestGameFrameTypes(t *testing.T) {
	frame := &arcade.GameFrame{
		FrameID:     1,
		PlayerState: &arcade.PlayerState{X: 100, Y: 100},
		Objects:     make(map[string]*arcade.ObjectState),
	}

	if frame.FrameID != 1 {
		t.Error("frame ID mismatch")
	}

	if frame.PlayerState == nil {
		t.Error("PlayerState is nil")
	}

	if frame.Objects == nil {
		t.Error("Objects map is nil")
	}

	// Add an object
	frame.Objects["test"] = &arcade.ObjectState{
		ID: "test",
		X:  50,
		Y:  50,
	}

	if _, ok := frame.Objects["test"]; !ok {
		t.Error("object not found in frame")
	}
}

// BenchmarkRegistryLookup measures registry performance
func BenchmarkRegistryLookup(b *testing.B) {
	arcade.InitRegistry()
	arcade.RegisterSystem("benchmark", template.Factory, &arcade.SystemSpec{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arcade.ListAvailableSystems()
	}
}

// BenchmarkSystemInfoRetrieval measures system info lookup
func BenchmarkSystemInfoRetrieval(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arcade.GetSystemInfo("neogeo")
	}
}
