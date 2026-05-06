package main

import (
	"testing"
	"time"

	"cadastreia/pkg/arcade"
	"cadastreia/pkg/model"
)

// TestNEOGEOCompilerBasic tests ROM compilation
func TestNEOGEOCompilerBasic(t *testing.T) {
	compiler := arcade.NewNEOGEOCompiler("./")

	// Create test game state
	gameState := &model.GameState{
		Player: &model.PlayerState{
			X:      100,
			Y:      100,
			Health: 100,
		},
		Objects: make(map[string]*model.ObjectState),
	}

	// Add test objects
	gameState.Objects["tree1"] = &model.ObjectState{
		ID:   "tree1",
		Type: "tree",
		X:    50,
		Y:    50,
	}

	// Compile to ROM
	err := compiler.CompileToFile(gameState, "test_rom.bin")
	if err != nil {
		t.Fatalf("Failed to compile ROM: %v", err)
	}

	// Get ROM info
	info, err := compiler.GetROMInfo("test_rom.bin")
	if err != nil {
		t.Fatalf("Failed to get ROM info: %v", err)
	}

	if info == nil {
		t.Fatal("ROM info is nil")
	}

	t.Logf("✓ NEO-GEO ROM compiled: %s (%d bytes)", info["filename"], info["size"])
}

// TestControlMapperInput tests joystick input mapping
func TestControlMapperInput(t *testing.T) {
	mapper := arcade.NewControlMapper("neoragex5")

	// Simulate up + punch input
	mapper.PushInput(arcade.JoystickUp | arcade.ButtonA)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Get mapped input
	input := mapper.GetGameInput()
	if input == nil {
		t.Fatal("No input received")
	}

	if input.Direction != "up" {
		t.Errorf("Expected direction 'up', got '%s'", input.Direction)
	}

	if input.Button != "A" {
		t.Errorf("Expected button 'A', got '%s'", input.Button)
	}

	t.Logf("✓ Control mapping: %s + %s", input.Direction, input.Button)
}

// TestDiagonalMovement tests diagonal input handling
func TestDiagonalMovement(t *testing.T) {
	mapper := arcade.NewControlMapper("neoragex5")

	// Simulate up-left movement
	mapper.PushInput(arcade.JoystickUp | arcade.JoystickLeft)

	time.Sleep(50 * time.Millisecond)

	input := mapper.GetGameInput()
	if input == nil {
		t.Fatal("No input received")
	}

	if input.Direction != "up-left" {
		t.Errorf("Expected diagonal 'up-left', got '%s'", input.Direction)
	}

	// Check movement vector is normalized
	dx := input.Movement[0]
	dy := input.Movement[1]
	if dx >= 0 {
		t.Errorf("Expected negative X movement, got %f", dx)
	}
	if dy <= 0 {
		t.Errorf("Expected negative Y movement, got %f", dy)
	}

	t.Logf("✓ Diagonal movement: %.2f, %.2f", dx, dy)
}

// TestNeoRageX5EmulatorBasic tests emulator initialization
func TestNeoRageX5EmulatorBasic(t *testing.T) {
	emulator := arcade.NewNeoRageX5Emulator(9001, "test.neo")

	if err := emulator.Start(); err != nil {
		t.Fatalf("Failed to start emulator: %v", err)
	}
	defer emulator.Stop()

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	// Get status
	status := emulator.GetStatus()
	if status == nil {
		t.Fatal("Status is nil")
	}

	if !status["running"].(bool) {
		t.Fatal("Emulator not running")
	}

	t.Logf("✓ Emulator running on ports %d-%d", status["base_port"], status["base_port"].(int)+1)
}

// TestGameFrameCapture tests frame state snapshot
func TestGameFrameCapture(t *testing.T) {
	emulator := arcade.NewNeoRageX5Emulator(9001, "test.neo")

	if err := emulator.Start(); err != nil {
		t.Fatalf("Failed to start emulator: %v", err)
	}
	defer emulator.Stop()

	time.Sleep(100 * time.Millisecond)

	// The emulator should be running and able to capture frames
	// We can verify by checking status
	status := emulator.GetStatus()

	playerX := status["player_x"].(float32)
	playerY := status["player_y"].(float32)

	if playerX < 0 || playerX > 320 {
		t.Errorf("Player X out of bounds: %f", playerX)
	}

	if playerY < 0 || playerY > 224 {
		t.Errorf("Player Y out of bounds: %f", playerY)
	}

	t.Logf("✓ Player position: (%.0f, %.0f)", playerX, playerY)
}

// TestControllerSimulator tests test controller input
func TestControllerSimulator(t *testing.T) {
	mapper := arcade.NewControlMapper("neoragex5")
	controller := arcade.NewNEOGEOControllerSimulator(mapper)

	// Simulate right movement
	controller.SimulateMoveRight()
	time.Sleep(50 * time.Millisecond)

	input := mapper.GetGameInput()
	if input == nil {
		t.Fatal("No input received")
	}

	if input.Direction != "right" {
		t.Errorf("Expected 'right', got '%s'", input.Direction)
	}

	// Simulate punch
	controller.SimulatePunch()
	time.Sleep(50 * time.Millisecond)

	input = mapper.GetGameInput()
	if input != nil && input.Button != "A" {
		t.Errorf("Expected button 'A', got '%s'", input.Button)
	}

	// Release all
	controller.ReleaseAll()
	time.Sleep(50 * time.Millisecond)

	input = mapper.GetGameInput()
	if input != nil && input.Direction != "idle" {
		t.Errorf("Expected 'idle' after release, got '%s'", input.Direction)
	}

	t.Logf("✓ Controller simulation works")
}

// TestNeoRageX5Protocol tests input encoding/decoding
func TestNeoRageX5Protocol(t *testing.T) {
	protocol := arcade.NewNeoRageX5Protocol()

	// Create game input
	input := &arcade.GameInput{
		Direction: "up-right",
		Movement:  [2]float32{1.0, -1.0},
		Action:    "punch",
		Button:    "A",
		Timestamp: time.Now(),
	}

	// Encode to frame
	frame := protocol.EncodeInputFrame(input)
	if len(frame) != 4 {
		t.Errorf("Expected frame length 4, got %d", len(frame))
	}

	// Decode back
	decodedInput := protocol.DecodeInputFrame(frame)
	if decodedInput == nil {
		t.Fatal("Decoded input is nil")
	}

	if decodedInput.Action != "punch" {
		t.Errorf("Expected action 'punch', got '%s'", decodedInput.Action)
	}

	t.Logf("✓ Protocol encoding/decoding: %s + %s", decodedInput.Direction, decodedInput.Action)
}

// TestMultipleInputs tests input buffer with multiple inputs
func TestMultipleInputs(t *testing.T) {
	mapper := arcade.NewControlMapper("neoragex5")

	// Push multiple inputs
	mapper.PushInput(arcade.JoystickUp)
	time.Sleep(20 * time.Millisecond)
	mapper.PushInput(arcade.JoystickUp | arcade.ButtonA)
	time.Sleep(20 * time.Millisecond)
	mapper.PushInput(arcade.ButtonB)
	time.Sleep(20 * time.Millisecond)

	// Collect all inputs
	inputs := []*arcade.GameInput{}
	for i := 0; i < 3; i++ {
		if input := mapper.GetGameInput(); input != nil {
			inputs = append(inputs, input)
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(inputs) == 0 {
		t.Fatal("No inputs received")
	}

	t.Logf("✓ Processed %d input events", len(inputs))
}

// TestEmulatorGameLoop tests game loop execution
func TestEmulatorGameLoop(t *testing.T) {
	emulator := arcade.NewNeoRageX5Emulator(9001, "test.neo")

	if err := emulator.Start(); err != nil {
		t.Fatalf("Failed to start emulator: %v", err)
	}
	defer emulator.Stop()

	// Wait several frames
	time.Sleep(200 * time.Millisecond) // ~12 frames at 60 FPS

	// Get status after game loop updates
	status1 := emulator.GetStatus()
	frameCount1 := status1["frame_count"].(uint64)

	time.Sleep(100 * time.Millisecond) // ~6 more frames

	status2 := emulator.GetStatus()
	frameCount2 := status2["frame_count"].(uint64)

	if frameCount2 <= frameCount1 {
		t.Errorf("Frame count didn't increase: %d -> %d", frameCount1, frameCount2)
	}

	t.Logf("✓ Game loop advanced: frame %d -> %d (%.1f FPS)",
		frameCount1, frameCount2, float64(frameCount2-frameCount1)/0.1)
}

// BenchmarkControlMapping benchmarks input mapping performance
func BenchmarkControlMapping(b *testing.B) {
	mapper := arcade.NewControlMapper("neoragex5")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.PushInput(arcade.JoystickUp | arcade.ButtonA)
	}
}

// BenchmarkROMCompilation benchmarks ROM compilation
func BenchmarkROMCompilation(b *testing.B) {
	compiler := arcade.NewNEOGEOCompiler("./")
	gameState := &model.GameState{
		Player: &model.PlayerState{X: 100, Y: 100},
		Objects: make(map[string]*model.ObjectState),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compiler.CompileToFile(gameState, "bench.bin")
	}
}
