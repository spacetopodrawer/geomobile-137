package game

import (
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	"cadastreia/pkg/convert"
	"cadastreia/pkg/model"
	"cadastreia/pkg/storage"
)

// GameState represents the current game state
type GameState struct {
	Player       PlayerState
	Objects      map[string]*model.VectorObject
	SelectedID   *string
	CameraX      float32
	CameraY      float32
	Zoom         float32
	FPS          int
	ElapsedTime  time.Duration
}

// PlayerState represents player information
type PlayerState struct {
	X              float32
	Y              float32
	Direction      float32 // 0-360 degrees
	Health         int32
	Inventory      []string // Object IDs
	LastInputTime  time.Time
	CurrentAction  string   // "idle", "scanning", "moving"
}

// Input represents player input
type Input struct {
	Type      string // "movement", "button", "scan"
	Direction string // "up", "down", "left", "right"
	Button    string // "a", "b", "c", "d"
}

// GameEngine manages game logic and rendering
type GameEngine struct {
	mu         sync.RWMutex
	state      *GameState
	converter  *convert.VectorToArcadeConverter
	storage    *storage.StorageManager
	running    bool
	targetFPS  int
	deltaTime  time.Duration
	lastFrame  time.Time
	inputChan  chan Input
	renderChan chan *image.Paletted
}

// NewGameEngine creates a new game engine
func NewGameEngine(sm *storage.StorageManager) *GameEngine {
	ge := &GameEngine{
		state: &GameState{
			Objects: make(map[string]*model.VectorObject),
			Player: PlayerState{
				X:            0,
				Y:            0,
				Health:       100,
				Inventory:    make([]string, 0),
				CurrentAction: "idle",
			},
			CameraX: 0,
			CameraY: 0,
			Zoom:    1.0,
			FPS:     60,
		},
		converter:  convert.NewVectorToArcadeConverter(32, 32),
		storage:    sm,
		targetFPS:  60,
		inputChan:  make(chan Input, 100),
		renderChan: make(chan *image.Paletted, 1),
	}

	return ge
}

// Start starts the game engine
func (ge *GameEngine) Start() error {
	ge.mu.Lock()
	ge.running = true
	ge.lastFrame = time.Now()
	ge.mu.Unlock()

	// Load initial objects from storage
	objects, err := ge.storage.ListObjects(100, 0)
	if err != nil {
		return fmt.Errorf("failed to load objects: %w", err)
	}

	ge.mu.Lock()
	for _, obj := range objects {
		ge.state.Objects[obj.ID.String()] = obj
	}
	ge.mu.Unlock()

	log.Printf("✓ Game engine started (loaded %d objects)", len(objects))

	// Start game loop
	go ge.gameLoop()

	return nil
}

// Stop stops the game engine
func (ge *GameEngine) Stop() {
	ge.mu.Lock()
	ge.running = false
	ge.mu.Unlock()
	log.Printf("✓ Game engine stopped")
}

// gameLoop runs the main game loop
func (ge *GameEngine) gameLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(ge.targetFPS))
	defer ticker.Stop()

	for range ticker.C {
		ge.mu.RLock()
		if !ge.running {
			ge.mu.RUnlock()
			break
		}
		ge.mu.RUnlock()

		// Calculate delta time
		now := time.Now()
		ge.mu.Lock()
		ge.deltaTime = now.Sub(ge.lastFrame)
		ge.lastFrame = now
		ge.mu.Unlock()

		// Update game state
		ge.updateFrame()

		// Render frame
		img := ge.renderFrame()

		// Send rendered frame
		select {
		case ge.renderChan <- img:
		default:
			// Drop frame if render channel is full
		}
	}
}

// updateFrame updates game state
func (ge *GameEngine) updateFrame() {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	// Process input
	select {
	case input := <-ge.inputChan:
		ge.processInput(input)
	default:
	}

	// Update player position based on current action
	switch ge.state.Player.CurrentAction {
	case "moving":
		// Movement logic here
	case "scanning":
		// Scanning animation
	}

	// Update FPS counter
	if ge.deltaTime > 0 {
		fps := time.Second / ge.deltaTime
		ge.state.FPS = int(fps)
	}
}

// processInput handles player input
func (ge *GameEngine) processInput(input Input) {
	switch input.Type {
	case "movement":
		ge.handleMovement(input.Direction)

	case "button":
		ge.handleButton(input.Button)

	case "scan":
		ge.handleScan()
	}

	ge.state.Player.LastInputTime = time.Now()
}

// handleMovement handles joystick movement
func (ge *GameEngine) handleMovement(direction string) {
	speed := float32(2.0)

	switch direction {
	case "up":
		ge.state.Player.Y -= speed
		ge.state.Player.Direction = 0
		ge.state.Player.CurrentAction = "moving"

	case "down":
		ge.state.Player.Y += speed
		ge.state.Player.Direction = 180
		ge.state.Player.CurrentAction = "moving"

	case "left":
		ge.state.Player.X -= speed
		ge.state.Player.Direction = 270
		ge.state.Player.CurrentAction = "moving"

	case "right":
		ge.state.Player.X += speed
		ge.state.Player.Direction = 90
		ge.state.Player.CurrentAction = "moving"

	default:
		ge.state.Player.CurrentAction = "idle"
	}

	// Update camera to follow player
	ge.state.CameraX = ge.state.Player.X
	ge.state.CameraY = ge.state.Player.Y
}

// handleButton handles button presses
func (ge *GameEngine) handleButton(button string) {
	switch button {
	case "a":
		ge.handleScan()

	case "b":
		if ge.state.Zoom < 2.0 {
			ge.state.Zoom += 0.1
		}

	case "c":
		log.Printf("📦 Inventory: %d items", len(ge.state.Player.Inventory))

	case "d":
		log.Printf("🔄 Syncing...")
	}
}

// handleScan handles scanning action
func (ge *GameEngine) handleScan() {
	ge.state.Player.CurrentAction = "scanning"

	// Find nearby objects
	for id, obj := range ge.state.Objects {
		dx := obj.Geometry != nil // Check if within range
		if dx && len(ge.state.Player.Inventory) < 10 {
			ge.state.Player.Inventory = append(ge.state.Player.Inventory, id)
			log.Printf("✓ Scanned: %s", obj.Name)
			break
		}
	}

	ge.state.Player.CurrentAction = "idle"
}

// renderFrame renders the current frame
func (ge *GameEngine) renderFrame() *image.Paletted {
	// Create image buffer (256×224 arcade resolution)
	img := image.NewPaletted(
		image.Rect(0, 0, 256, 224),
		convert.DefaultArcadePalette[:],
	)

	// Fill background
	for y := 0; y < 224; y++ {
		for x := 0; x < 256; x++ {
			img.SetColorIndex(x, y, 0) // Black background
		}
	}

	// Render player (center of screen)
	playerScreenX := 128
	playerScreenY := 112
	playerColor := uint8(3) // Yellow

	// Draw player as simple square
	for y := playerScreenY - 2; y < playerScreenY+2; y++ {
		for x := playerScreenX - 2; x < playerScreenX+2; x++ {
			if x >= 0 && x < 256 && y >= 0 && y < 224 {
				img.SetColorIndex(x, y, playerColor)
			}
		}
	}

	// Render nearby objects
	objColor := uint8(2) // Green
	for _, obj := range ge.state.Objects {
		// Simple visualization: render objects as dots
		if obj.Geometry != nil {
			x := 64 + int(ge.state.Player.X)%128
			y := 32 + int(ge.state.Player.Y)%160

			if x >= 0 && x < 256 && y >= 0 && y < 224 {
				img.SetColorIndex(x, y, objColor)
			}
		}
	}

	// Render UI text (simplified - just render borders)
	uiBorderColor := uint8(7) // White
	for x := 0; x < 256; x++ {
		img.SetColorIndex(x, 0, uiBorderColor)
		img.SetColorIndex(x, 223, uiBorderColor)
	}
	for y := 0; y < 224; y++ {
		img.SetColorIndex(0, y, uiBorderColor)
		img.SetColorIndex(255, y, uiBorderColor)
	}

	return img
}

// HandleInput handles external input
func (ge *GameEngine) HandleInput(input Input) {
	select {
	case ge.inputChan <- input:
	default:
		log.Printf("⚠️  Input queue full")
	}
}

// GetGameState returns current game state
func (ge *GameEngine) GetGameState() *GameState {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	// Return copy to avoid race conditions
	stateCopy := *ge.state
	return &stateCopy
}

// GetStatus returns game engine status
func (ge *GameEngine) GetStatus() map[string]interface{} {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	return map[string]interface{}{
		"running":      ge.running,
		"fps":          ge.state.FPS,
		"player_x":     ge.state.Player.X,
		"player_y":     ge.state.Player.Y,
		"objects":      len(ge.state.Objects),
		"inventory":    len(ge.state.Player.Inventory),
		"action":       ge.state.Player.CurrentAction,
		"health":       ge.state.Player.Health,
	}
}

// Close closes the game engine
func (ge *GameEngine) Close() {
	ge.Stop()
	close(ge.inputChan)
}
