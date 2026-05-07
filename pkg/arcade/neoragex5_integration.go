package arcade

import (
	"fmt"
	"net"
	"sync"
	"time"

	syncsvc "cadastreia/pkg/sync"
)

// NeoRageX5Emulator represents the NeoRageX5 emulator runtime
type NeoRageX5Emulator struct {
	mu                 sync.RWMutex
	isRunning          bool
	processID          int
	basePort           int
	romPath            string
	connectedClients   map[string]*EmulatorClient
	broadcastChannel   chan *GameFrame
	commandListener    net.Listener
	inputListener      net.Listener
	protocol           *NeoRageX5Protocol
	gameState          *GameState
	syncManager        *syncsvc.SyncManager
	frameCount         uint64
	fps                int
	running            bool
	closeChan          chan bool
}

// EmulatorClient represents a connected emulator instance
type EmulatorClient struct {
	deviceID     string
	conn         net.Conn
	lastFrame    *GameFrame
	frameBuffer  chan *GameFrame
	inputMapper  *ControlMapper
	joinedAt     time.Time
	lastActivity time.Time
}

// GameFrame represents a complete game state frame for emulator
type GameFrame struct {
	FrameID      uint64
	Timestamp    time.Time
	PlayerState  *PlayerState
	Objects      map[string]*ObjectState
	InputData    []byte
	SyncID       string
	SourceDevice string
	CheckSum     uint32
}

// PlayerState represents player state in emulator frame
type PlayerState struct {
	X             float32
	Y             float32
	Direction     float32 // 0-360 degrees
	Health        uint8
	Score         uint32
	CurrentAction string
	SpriteID      uint16
}

// ObjectState represents an object in the game world
type ObjectState struct {
	ID        string
	Type      string
	X         float32
	Y         float32
	SpriteID  uint16
	Rotation  uint8
	PaletteID uint8
	Visible   bool
}

// GameState represents the complete game state
type GameState struct {
	mu       sync.RWMutex
	Player   *PlayerState
	Objects  map[string]*ObjectState
	Camera   CameraState
	GameTime uint64
	Paused   bool
	GameOver bool
}

// CameraState represents camera positioning
type CameraState struct {
	X    float32
	Y    float32
	Zoom float32
}

// NewNeoRageX5Emulator creates a new emulator instance
func NewNeoRageX5Emulator(basePort int, romPath string) *NeoRageX5Emulator {
	return &NeoRageX5Emulator{
		basePort:         basePort,
		romPath:          romPath,
		connectedClients: make(map[string]*EmulatorClient),
		broadcastChannel: make(chan *GameFrame, 100),
		protocol:         NewNeoRageX5Protocol(),
		gameState: &GameState{
			Player: &PlayerState{
				X:             160,
				Y:             112,
				Direction:     0,
				Health:        100,
				Score:         0,
				CurrentAction: "idle",
				SpriteID:      0,
			},
			Objects: make(map[string]*ObjectState),
			Camera: CameraState{
				X:    160,
				Y:    112,
				Zoom: 1.0,
			},
		},
		syncManager: syncsvc.NewSyncManager("emulator-main"),
		fps:         60,
		closeChan:   make(chan bool),
	}
}

// Start initializes and starts the emulator
func (e *NeoRageX5Emulator) Start() error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return fmt.Errorf("emulator already running")
	}
	e.isRunning = true
	e.running = true
	e.mu.Unlock()

	// Start command listener (for game logic and sync)
	cmdListener, err := net.Listen("tcp", fmt.Sprintf(":%d", e.basePort))
	if err != nil {
		return fmt.Errorf("failed to start command listener: %w", err)
	}
	e.commandListener = cmdListener

	// Start input listener (for controller data)
	inputListener, err := net.Listen("tcp", fmt.Sprintf(":%d", e.basePort+1))
	if err != nil {
		cmdListener.Close()
		return fmt.Errorf("failed to start input listener: %w", err)
	}
	e.inputListener = inputListener

	// Start goroutines
	go e.acceptConnections()
	go e.gameLoop()
	go e.broadcastState()

	fmt.Printf("✓ NeoRageX5 Emulator started on port %d (commands), %d (input)\n", e.basePort, e.basePort+1)
	fmt.Printf("  ROM: %s\n", e.romPath)

	return nil
}

// Stop shuts down the emulator
func (e *NeoRageX5Emulator) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return fmt.Errorf("emulator not running")
	}

	e.running = false
	close(e.closeChan)

	// Close all client connections
	for deviceID, client := range e.connectedClients {
		if client.conn != nil {
			client.conn.Close()
		}
		fmt.Printf("Disconnected client: %s\n", deviceID)
	}

	// Close listeners
	if e.commandListener != nil {
		e.commandListener.Close()
	}
	if e.inputListener != nil {
		e.inputListener.Close()
	}

	e.isRunning = false
	fmt.Println("✓ NeoRageX5 Emulator stopped")

	return nil
}

// acceptConnections handles incoming client connections
func (e *NeoRageX5Emulator) acceptConnections() {
	go e.acceptCommandConnections()
	go e.acceptInputConnections()
}

// acceptCommandConnections accepts game logic/sync connections
func (e *NeoRageX5Emulator) acceptCommandConnections() {
	for e.running {
		conn, err := e.commandListener.Accept()
		if err != nil {
			if !e.running {
				return
			}
			fmt.Printf("Error accepting command connection: %v\n", err)
			continue
		}

		go e.handleCommandConnection(conn)
	}
}

// acceptInputConnections accepts controller input connections
func (e *NeoRageX5Emulator) acceptInputConnections() {
	for e.running {
		conn, err := e.inputListener.Accept()
		if err != nil {
			if !e.running {
				return
			}
			fmt.Printf("Error accepting input connection: %v\n", err)
			continue
		}

		go e.handleInputConnection(conn)
	}
}

// handleCommandConnection processes game state updates from clients
func (e *NeoRageX5Emulator) handleCommandConnection(conn net.Conn) {
	defer conn.Close()

	// Read device handshake
	handshake := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(handshake)
	if err != nil {
		fmt.Printf("Handshake read error: %v\n", err)
		return
	}

	deviceID := string(handshake[:n])

	e.mu.Lock()
	client := &EmulatorClient{
		deviceID:    deviceID,
		conn:        conn,
		frameBuffer: make(chan *GameFrame, 10),
		joinedAt:    time.Now(),
	}
	e.connectedClients[deviceID] = client
	e.mu.Unlock()

	fmt.Printf("✓ Command client connected: %s\n", deviceID)

	// Send current game state
	e.sendGameState(conn, client)

	// Listen for updates
	for e.running {
		frameBuf := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(frameBuf)
		if err != nil {
			break
		}

		// Parse received frame
		if n > 0 {
			e.processClientUpdate(frameBuf[:n], deviceID)
		}

		client.lastActivity = time.Now()
	}

	e.mu.Lock()
	delete(e.connectedClients, deviceID)
	e.mu.Unlock()

	fmt.Printf("✗ Command client disconnected: %s\n", deviceID)
}

// handleInputConnection processes controller input from clients
func (e *NeoRageX5Emulator) handleInputConnection(conn net.Conn) {
	defer conn.Close()

	// Read device ID
	deviceID := make([]byte, 32)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(deviceID)
	if err != nil {
		fmt.Printf("Input device ID read error: %v\n", err)
		return
	}

	devID := string(deviceID[:n])

	// Get or create input mapper for this device
	mapper := NewControlMapper("neoragex5")

	fmt.Printf("✓ Input client connected: %s\n", devID)

	// Process input frames
	frameBuf := make([]byte, 4)
	for e.running {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := conn.Read(frameBuf)
		if err != nil {
			break
		}

		if n == 4 {
			// Push raw input to mapper
			buttons := JoystickButton(frameBuf[0]) |
				(JoystickButton(frameBuf[1]) << 8) |
				(JoystickButton(frameBuf[2]) << 16)

			mapper.PushInput(buttons)

			// Get processed game input
			gameInput := mapper.GetGameInput()
			if gameInput != nil {
				// Apply to game state
				e.applyPlayerInput(gameInput)
			}
		}
	}

	fmt.Printf("✗ Input client disconnected: %s\n", devID)
}

// processClientUpdate handles game state updates from other emulator instances
func (e *NeoRageX5Emulator) processClientUpdate(data []byte, sourceDeviceID string) {
	// This would parse network protocol updates
	// For now, we accept but don't process

	// Create sync operation for replication
	_ = e.syncManager.CreateOperation(
		"game-state",
		"update",
		nil,
		data,
	)
}

// applyPlayerInput applies controller input to game state
func (e *NeoRageX5Emulator) applyPlayerInput(input *GameInput) {
	e.gameState.mu.Lock()
	defer e.gameState.mu.Unlock()

	// Update player position
	e.gameState.Player.X += input.Movement[0]
	e.gameState.Player.Y += input.Movement[1]

	// Clamp to game boundaries (NEO-GEO 320x224)
	if e.gameState.Player.X < 0 {
		e.gameState.Player.X = 0
	}
	if e.gameState.Player.X > 320 {
		e.gameState.Player.X = 320
	}
	if e.gameState.Player.Y < 0 {
		e.gameState.Player.Y = 0
	}
	if e.gameState.Player.Y > 224 {
		e.gameState.Player.Y = 224
	}

	// Update action
	if input.Action != "idle" {
		e.gameState.Player.CurrentAction = input.Action
	} else {
		e.gameState.Player.CurrentAction = "idle"
	}

	// Update direction
	switch input.Direction {
	case "up":
		e.gameState.Player.Direction = 0
	case "down":
		e.gameState.Player.Direction = 180
	case "left":
		e.gameState.Player.Direction = 270
	case "right":
		e.gameState.Player.Direction = 90
	}
}

// gameLoop runs the main game loop
func (e *NeoRageX5Emulator) gameLoop() {
	frameTime := time.Duration(1000000000 / int64(e.fps)) // Nanoseconds per frame
	ticker := time.NewTicker(frameTime)
	defer ticker.Stop()

	for {
		select {
		case <-e.closeChan:
			return
		case <-ticker.C:
			e.updateGameState()
			e.frameCount++
		}
	}
}

// updateGameState updates game logic each frame
func (e *NeoRageX5Emulator) updateGameState() {
	e.gameState.mu.Lock()
	defer e.gameState.mu.Unlock()

	// Update game time
	e.gameState.GameTime++

	// Update player sprite based on action
	switch e.gameState.Player.CurrentAction {
	case "idle":
		e.gameState.Player.SpriteID = 0
	case "moving":
		// Cycle through movement sprites
		e.gameState.Player.SpriteID = uint16((e.gameState.GameTime / 5) % 4)
	case "punch":
		e.gameState.Player.SpriteID = 10
		e.gameState.Player.Score += 10
	case "kick":
		e.gameState.Player.SpriteID = 11
		e.gameState.Player.Score += 15
	case "special":
		e.gameState.Player.SpriteID = 12
		e.gameState.Player.Score += 50
	}
}

// broadcastState sends game state to all connected clients
func (e *NeoRageX5Emulator) broadcastState() {
	ticker := time.NewTicker(time.Duration(1000000000 / int64(e.fps)) * time.Nanosecond)
	defer ticker.Stop()

	for {
		select {
		case <-e.closeChan:
			return
		case <-ticker.C:
			frame := e.captureFrame()
			e.mu.RLock()
			for deviceID, client := range e.connectedClients {
				select {
				case client.frameBuffer <- frame:
				default:
					fmt.Printf("Frame buffer full for %s, dropping\n", deviceID)
				}
			}
			e.mu.RUnlock()
		}
	}
}

// captureFrame creates a game frame snapshot
func (e *NeoRageX5Emulator) captureFrame() *GameFrame {
	e.gameState.mu.RLock()
	defer e.gameState.mu.RUnlock()

	// Create deep copy of game state
	frame := &GameFrame{
		FrameID:      e.frameCount,
		Timestamp:    time.Now(),
		PlayerState:  &PlayerState{}, // Copy values
		Objects:      make(map[string]*ObjectState),
		SourceDevice: "emulator-main",
	}

	// Copy player state
	*frame.PlayerState = *e.gameState.Player

	// Copy objects
	for id, obj := range e.gameState.Objects {
		objCopy := *obj
		frame.Objects[id] = &objCopy
	}

	return frame
}

// sendGameState sends complete game state to a new client
func (e *NeoRageX5Emulator) sendGameState(conn net.Conn, client *EmulatorClient) {
	frame := e.captureFrame()

	// Serialize and send frame
	// This is simplified - real implementation would use binary protocol
	data := fmt.Sprintf("FRAME:%d:PLAYER:%.0f:%.0f:%d\n",
		frame.FrameID,
		frame.PlayerState.X,
		frame.PlayerState.Y,
		frame.PlayerState.Score)

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	conn.Write([]byte(data))
}

// GetStatus returns emulator runtime status
func (e *NeoRageX5Emulator) GetStatus() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"running":        e.isRunning,
		"rom_path":       e.romPath,
		"connected_clients": len(e.connectedClients),
		"fps":            e.fps,
		"frame_count":    e.frameCount,
		"base_port":      e.basePort,
		"player_x":       e.gameState.Player.X,
		"player_y":       e.gameState.Player.Y,
		"player_score":   e.gameState.Player.Score,
		"player_action":  e.gameState.Player.CurrentAction,
	}
}

// ConnectRemoteEmulator connects to another emulator for P2P sync
func (e *NeoRageX5Emulator) ConnectRemoteEmulator(remoteAddr string) error {
	// Connect to remote emulator's command port
	conn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to remote emulator: %w", err)
	}

	// Send handshake
	deviceID := fmt.Sprintf("emulator-%d", time.Now().UnixNano()%10000)
	conn.Write([]byte(deviceID))

	fmt.Printf("✓ Connected to remote emulator: %s\n", remoteAddr)

	// Start receiving state updates from remote
	go func() {
		defer conn.Close()
		buf := make([]byte, 512)
		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				e.processClientUpdate(buf[:n], remoteAddr)
			}
		}
	}()

	return nil
}
