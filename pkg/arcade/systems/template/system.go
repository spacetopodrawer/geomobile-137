package template

import (
	"fmt"
	"net"
	"sync"
	"time"

	"cadastreia/pkg/arcade"
)

// TemplateEmulator is a template showing how to implement ArcadeEmulator interface
type TemplateEmulator struct {
	mu               sync.RWMutex
	isRunning        bool
	basePort         int
	romPath          string
	connectedClients map[string]*Client
	broadcastChannel chan *arcade.GameFrame
	gameState        *arcade.GameState
	frameCount       uint64
	fps              int
	closeChan        chan bool
	system           *arcade.SystemInfo
}

// Client represents a connected remote emulator
type Client struct {
	deviceID     string
	conn         net.Conn
	lastFrame    *arcade.GameFrame
	frameBuffer  chan *arcade.GameFrame
	joinedAt     time.Time
	lastActivity time.Time
}

// NewTemplateEmulator creates a new template emulator instance
func NewTemplateEmulator(basePort int, romPath string) (*TemplateEmulator, error) {
	tm := &TemplateEmulator{
		basePort:         basePort,
		romPath:          romPath,
		connectedClients: make(map[string]*Client),
		broadcastChannel: make(chan *arcade.GameFrame, 100),
		gameState:        &arcade.GameState{},
		fps:              60,
		closeChan:        make(chan bool),
		system:           arcade.GetSystemInfo("neogeo"), // Template uses NEO-GEO specs
	}
	return tm, nil
}

// Factory function for registry
func Factory(basePort int, romPath string) (arcade.ArcadeEmulator, error) {
	return NewTemplateEmulator(basePort, romPath)
}

// Start begins the emulator and game loop
func (te *TemplateEmulator) Start() error {
	te.mu.Lock()
	if te.isRunning {
		te.mu.Unlock()
		return fmt.Errorf("emulator already running")
	}

	te.isRunning = true
	te.mu.Unlock()

	// Start game loop
	go te.gameLoop()

	// Start event listener (for accepting remote connections)
	go te.eventLoop()

	return nil
}

// Stop halts the emulator
func (te *TemplateEmulator) Stop() error {
	te.mu.Lock()
	if !te.isRunning {
		te.mu.Unlock()
		return fmt.Errorf("emulator not running")
	}

	te.isRunning = false
	te.mu.Unlock()

	close(te.closeChan)
	return nil
}

// IsRunning returns whether the emulator is active
func (te *TemplateEmulator) IsRunning() bool {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.isRunning
}

// GetFrame returns the current game frame
func (te *TemplateEmulator) GetFrame() *arcade.GameFrame {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if te.gameState == nil {
		return nil
	}

	return &arcade.GameFrame{
		FrameID:      te.frameCount,
		Timestamp:    time.Now(),
		PlayerState:  te.gameState.Player,
		Objects:      te.gameState.Objects,
		SourceDevice: te.system.SystemID,
	}
}

// GetStatus returns emulator statistics
func (te *TemplateEmulator) GetStatus() map[string]interface{} {
	te.mu.RLock()
	defer te.mu.RUnlock()

	return map[string]interface{}{
		"running":           te.isRunning,
		"frame_count":       te.frameCount,
		"fps":               te.fps,
		"connected_clients": len(te.connectedClients),
		"system":            te.system.SystemID,
	}
}

// GetSystem returns system information
func (te *TemplateEmulator) GetSystem() *arcade.SystemInfo {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.system
}

// ConnectRemoteEmulator connects to another emulator instance
func (te *TemplateEmulator) ConnectRemoteEmulator(deviceID, address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	client := &Client{
		deviceID:    deviceID,
		conn:        conn,
		frameBuffer: make(chan *arcade.GameFrame, 10),
		joinedAt:    time.Now(),
	}

	te.mu.Lock()
	te.connectedClients[deviceID] = client
	te.mu.Unlock()

	return nil
}

// GetConnectedClients returns list of connected device IDs
func (te *TemplateEmulator) GetConnectedClients() []string {
	te.mu.RLock()
	defer te.mu.RUnlock()

	clients := make([]string, 0, len(te.connectedClients))
	for deviceID := range te.connectedClients {
		clients = append(clients, deviceID)
	}
	return clients
}

// BroadcastFrame sends a frame to all connected clients
func (te *TemplateEmulator) BroadcastFrame(frame *arcade.GameFrame) error {
	select {
	case te.broadcastChannel <- frame:
		return nil
	default:
		return fmt.Errorf("broadcast channel full")
	}
}

// ApplyInput processes input from a game controller
func (te *TemplateEmulator) ApplyInput(input *arcade.GameInput) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}

	// TODO: Apply input to game logic
	// This would update player position, actions, etc.

	return nil
}

// GetInputBuffer returns the input channel
func (te *TemplateEmulator) GetInputBuffer() chan *arcade.GameInput {
	// TODO: Implement based on actual system
	return nil
}

// GetFrameCount returns total frames rendered
func (te *TemplateEmulator) GetFrameCount() uint64 {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.frameCount
}

// GetFPS returns frames per second
func (te *TemplateEmulator) GetFPS() int {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.fps
}

// gameLoop runs the main emulation loop at specified FPS
func (te *TemplateEmulator) gameLoop() {
	ticker := time.NewTicker(time.Duration(1000000/te.fps) * time.Microsecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			te.mu.Lock()
			te.frameCount++
			// TODO: Update game logic here
			te.mu.Unlock()

		case <-te.closeChan:
			return
		}
	}
}

// eventLoop accepts connections from remote emulator instances
func (te *TemplateEmulator) eventLoop() {
	// TODO: Implement based on actual system requirements
	// This would listen on ports specified in config
}
