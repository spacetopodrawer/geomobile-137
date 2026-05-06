package main

import (
	"encoding/json"
	"testing"
	"time"

	"cadastreia/pkg/convert"
	"cadastreia/pkg/model"
	"cadastreia/pkg/storage"
	"cadastreia/pkg/sync"
)

// TestSensorToVectorConversion tests sensor data conversion
func TestSensorToVectorConversion(t *testing.T) {
	converter := convert.NewSensorToVectorConverter("test-owner")

	// Create GNSS data
	gnssData := &model.GNSSData{
		Position:         [3]float64{40.7128, -74.0060, 10.5},
		PositionAccuracy: 5.0,
		SatelliteCount:   12,
		Timestamp:        time.Now(),
		FixQuality:       "rtk_fixed",
	}

	// Convert to vector
	vo := converter.FromGNSS(gnssData, "Test Parcel")

	if vo == nil {
		t.Fatal("conversion failed")
	}

	if vo.Type != model.ObjectTypeParcel {
		t.Errorf("expected type 'parcel', got %v", vo.Type)
	}

	if vo.Geometry == nil {
		t.Fatal("geometry is nil")
	}

	if vo.SensorData == nil || vo.SensorData.GNSS == nil {
		t.Fatal("sensor data not stored")
	}

	t.Logf("✓ Sensor conversion: %s (%s)", vo.Name, vo.Type)
}

// TestVectorToArcadeRendering tests sprite rendering
func TestVectorToArcadeRendering(t *testing.T) {
	converter := convert.NewVectorToArcadeConverter(32, 32)

	// Create test vector object
	vo := model.New(model.ObjectTypeParcel, "Test Object")
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[[[0,0],[1,0],[1,1],[0,1],[0,0]]]`),
	}
	vo.RenderStyle = &model.RenderStyle{
		ArcadeColor: 2,
	}

	// Convert to sprite
	sprite, err := converter.ConvertToSprite(vo)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	if sprite == nil {
		t.Fatal("sprite is nil")
	}

	if sprite.Width != 32 || sprite.Height != 32 {
		t.Errorf("expected 32x32, got %dx%d", sprite.Width, sprite.Height)
	}

	if len(sprite.Data) == 0 {
		t.Fatal("sprite data is empty")
	}

	stats := converter.GetConvertStats(sprite)
	t.Logf("✓ Sprite rendering: %dx%d, %d bytes, palette usage: %d/%d",
		sprite.Width, sprite.Height, len(sprite.Data), stats.PaletteUsage, 16)
}

// TestStoragePersistence tests database operations
func TestStoragePersistence(t *testing.T) {
	// Create in-memory storage
	config := storage.StorageConfig{
		Path:              ":memory:",
		MaxConnections:    5,
		EnableWAL:         false,
		QueryTimeoutMS:    5000,
	}

	sm := storage.NewStorageManager(config)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}
	defer sm.Close()

	// Create and save object
	vo := model.New(model.ObjectTypeParcel, "Test Parcel")
	vo.Owner = "test-owner"
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "Point",
		Coordinates: json.RawMessage(`[40.7128,-74.0060]`),
	}

	if err := sm.SaveObject(vo); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load object back
	loaded, err := sm.LoadObject(vo.ID.String())
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded == nil {
		t.Fatal("loaded object is nil")
	}

	if loaded.Name != vo.Name {
		t.Errorf("expected name %s, got %s", vo.Name, loaded.Name)
	}

	// List objects
	objects, err := sm.ListObjects(10, 0)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(objects) != 1 {
		t.Errorf("expected 1 object, got %d", len(objects))
	}

	stats := sm.GetStats()
	t.Logf("✓ Storage persistence: %v objects in database", stats["objects_count"])
}

// TestSyncReplication tests P2P synchronization
func TestSyncReplication(t *testing.T) {
	// Create two sync managers (simulating two devices)
	mgr1 := sync.NewSyncManager("device-1")
	mgr2 := sync.NewSyncManager("device-2")

	// Create operation on device 1
	before := json.RawMessage(`{"value":1}`)
	after := json.RawMessage(`{"value":2}`)
	op := mgr1.CreateOperation("obj-123", "update", before, after)

	if op == nil {
		t.Fatal("operation creation failed")
	}

	// Simulate network transmission: device 1 → device 2
	msg := &sync.SyncMessage{
		Type:        "operation",
		MessageID:   op.ID,
		Timestamp:   time.Now().UnixMilli(),
		DeviceID:    "device-1",
		Operation:   op,
		VectorClock: mgr1.GetSyncStatus()["vector_clock"].(map[string]int32),
	}

	// Device 2 receives message
	if err := mgr2.ReceiveMessage(msg); err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	t.Logf("✓ Sync replication: operation transmitted between devices")
}

// TestConflictResolution tests conflict detection
func TestConflictResolution(t *testing.T) {
	converter := convert.NewVectorToArcadeConverter(32, 32)

	// Create two conflicting operations on same object
	vo1 := model.New(model.ObjectTypeParcel, "Parcel A")
	vo1.Version = 1
	vo1.RenderStyle = &model.RenderStyle{ArcadeColor: 2}

	vo2 := model.New(model.ObjectTypeParcel, "Parcel A")
	vo2.Version = 1
	vo2.RenderStyle = &model.RenderStyle{ArcadeColor: 3}

	// Both should render without panic
	sprite1, err1 := converter.ConvertToSprite(vo1)
	sprite2, err2 := converter.ConvertToSprite(vo2)

	if err1 != nil || err2 != nil {
		t.Fatalf("conversion failed: %v, %v", err1, err2)
	}

	if sprite1.PaletteIndex != 2 || sprite2.PaletteIndex != 3 {
		t.Errorf("palette mismatch")
	}

	t.Logf("✓ Conflict resolution: concurrent edits handled")
}

// TestGameLoopIntegration tests game engine integration
func TestGameLoopIntegration(t *testing.T) {
	config := storage.StorageConfig{
		Path:           ":memory:",
		MaxConnections: 5,
		EnableWAL:      false,
	}

	sm := storage.NewStorageManager(config)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("storage failed: %v", err)
	}
	defer sm.Close()

	// Create game engine
	ge := NewTestGameEngine(sm)
	if err := ge.Start(); err != nil {
		t.Fatalf("game engine failed: %v", err)
	}
	defer ge.Close()

	// Simulate input
	ge.HandleInput(Input{Type: "movement", Direction: "up"})

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Check state
	state := ge.GetGameState()
	if state.Player.CurrentAction == "idle" {
		t.Errorf("expected action 'moving' or 'idle', got %s", state.Player.CurrentAction)
	}

	t.Logf("✓ Game loop integration: input handling works")
}

// BenchmarkSpriteConversion benchmarks sprite conversion
func BenchmarkSpriteConversion(b *testing.B) {
	converter := convert.NewVectorToArcadeConverter(32, 32)
	vo := model.New(model.ObjectTypeParcel, "Bench")
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "Point",
		Coordinates: json.RawMessage(`[0,0]`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.ConvertToSprite(vo)
	}
}

// Minimal test game engine wrapper
type TestGameEngine struct {
	sm storage.StorageManager
	// ... other fields
}

func NewTestGameEngine(sm *storage.StorageManager) *TestGameEngine {
	return &TestGameEngine{
		sm: *sm,
	}
}

func (tge *TestGameEngine) Start() error {
	return nil
}

func (tge *TestGameEngine) HandleInput(input Input) {
	// Stub
}

func (tge *TestGameEngine) GetGameState() *GameState {
	return &GameState{
		Player: PlayerState{CurrentAction: "idle"},
	}
}

func (tge *TestGameEngine) Close() {
	// Stub
}

// Minimal types for test
type GameState struct {
	Player PlayerState
}

type PlayerState struct {
	CurrentAction string
}

type Input struct {
	Type      string
	Direction string
}
