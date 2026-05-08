package neogeo

import (
	"fmt"
	"net"
	"sync"
	"time"

	"cadastreia/pkg/arcade"
	syncsvc "cadastreia/pkg/sync"
)

// NEOGEOEmulator represents the NEO-GEO arcade system emulator
// Implements arcade.ArcadeEmulator interface
type NEOGEOEmulator struct {
	mu                 sync.RWMutex
	isRunning          bool
	basePort           int
	romPath            string
	connectedClients   map[string]*EmulatorClient
	broadcastChannel   chan *arcade.GameFrame
	commandListener    net.Listener
	inputListener      net.Listener
	protocol           *NeoRageX5Protocol
	gameState          *arcade.GameState
	syncManager        *syncsvc.SyncManager
	frameCount         uint64
	fps                int
	closeChan          chan bool
	inputBuffer        chan *arcade.GameInput
	system             *arcade.SystemInfo
}

// EmulatorClient represents a connected emulator instance
type EmulatorClient struct {
	deviceID     string
	conn         net.Conn
	lastFrame    *arcade.GameFrame
	frameBuffer  chan *arcade.GameFrame
	inputMapper  *arcade.ControlMapper
	joinedAt     time.Time
	lastActivity time.Time
}

// NewNEOGEOEmulator creates a new NEO-GEO emulator instance
func NewNEOGEOEmulator(basePort int, romPath string) (*NEOGEOEmulator, error) {
	return &NEOGEOEmulator{
		basePort:         basePort,
		romPath:          romPath,
		connectedClients: make(map[string]*EmulatorClient),
		broadcastChannel: make(chan *arcade.GameFrame, 100),
		protocol:         NewNeoRageX5Protocol(),
		gameState: &arcade.GameState{
			Player: &arcade.PlayerState{
				X:             160,
				Y:             112,
				Direction:     0,
				Health:        100,
				Score:         0,
				CurrentAction: "idle",
				SpriteID:      0,
			},
			Objects: make(map[string]*arcade.ObjectState),
			Camera: arcade.CameraState{
				X:    160,
				Y:    112,
				Zoom: 1.0,
			},
		},
		syncManager: syncsvc.NewSyncManager("neogeo-emulator"),
		fps:         60,
		closeChan:   make(chan bool),
		inputBuffer: make(chan *arcade.GameInput, 100),
		system:      arcade.GetSystemInfo("neogeo"),
	}, nil
}

// Factory function for arcade system registry
func Factory(basePort int, romPath string) (arcade.ArcadeEmulator, error) {
	return NewNEOGEOEmulator(basePort, romPath)
}

// Start initializes and starts the emulator
func (e *NEOGEOEmulator) Start() error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return fmt.Errorf("NEO-GEO emulator already running")
	}
	e.isRunning = true
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

	fmt.Printf("✓ NEO-GEO Emulator started on port %d (commands), %d (input)\n", e.basePort, e.basePort+1)
	fmt.Printf("  ROM: %s\n", e.romPath)

	return nil
}

// Stop shuts down the emulator
func (e *NEOGEOEmulator) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return fmt.Errorf("NEO-GEO emulator not running")
	}

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
	fmt.Println("✓ NEO-GEO Emulator stopped")

	return nil
}

// IsRunning returns whether the emulator is active
func (e *NEOGEOEmulator) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// GetFrame returns the current game frame
func (e *NEOGEOEmulator) GetFrame() *arcade.GameFrame {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.gameState == nil {
		return nil
	}

	return &arcade.GameFrame{
		FrameID:      e.frameCount,
		Timestamp:    time.Now(),
		PlayerState:  e.gameState.Player,
		Objects:      e.gameState.Objects,
		SourceDevice: "neogeo",
	}
}

// GetStatus returns emulator statistics
func (e *NEOGEOEmulator) GetStatus() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"running":           e.isRunning,
		"frame_count":       e.frameCount,
		"fps":               e.fps,
		"connected_clients": len(e.connectedClients),
		"system":            "neogeo",
	}
}

// GetSystem returns system information
func (e *NEOGEOEmulator) GetSystem() *arcade.SystemInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.system
}

// ConnectRemoteEmulator connects to another emulator instance
func (e *NEOGEOEmulator) ConnectRemoteEmulator(deviceID, address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	client := &EmulatorClient{
		deviceID:    deviceID,
		conn:        conn,
		frameBuffer: make(chan *arcade.GameFrame, 10),
		joinedAt:    time.Now(),
	}

	e.mu.Lock()
	e.connectedClients[deviceID] = client
	e.mu.Unlock()

	return nil
}

// GetConnectedClients returns list of connected device IDs
func (e *NEOGEOEmulator) GetConnectedClients() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	clients := make([]string, 0, len(e.connectedClients))
	for deviceID := range e.connectedClients {
		clients = append(clients, deviceID)
	}
	return clients
}

// BroadcastFrame sends a frame to all connected clients
func (e *NEOGEOEmulator) BroadcastFrame(frame *arcade.GameFrame) error {
	select {
	case e.broadcastChannel <- frame:
		return nil
	default:
		return fmt.Errorf("broadcast channel full")
	}
}

// ApplyInput processes input from a game controller
func (e *NEOGEOEmulator) ApplyInput(input *arcade.GameInput) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}

	select {
	case e.inputBuffer <- input:
		return nil
	default:
		return fmt.Errorf("input buffer full")
	}
}

// GetInputBuffer returns the input channel
func (e *NEOGEOEmulator) GetInputBuffer() chan *arcade.GameInput {
	return e.inputBuffer
}

// GetFrameCount returns total frames rendered
func (e *NEOGEOEmulator) GetFrameCount() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.frameCount
}

// GetFPS returns frames per second
func (e *NEOGEOEmulator) GetFPS() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.fps
}

// gameLoop runs the main emulation loop at 60 FPS
func (e *NEOGEOEmulator) gameLoop() {
	ticker := time.NewTicker(time.Duration(1000000/e.fps) * time.Microsecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.mu.Lock()
			e.frameCount++

			// Process input if available
			select {
			case input := <-e.inputBuffer:
				if input != nil {
					// Update player position based on input
					if input.Movement[0] != 0 || input.Movement[1] != 0 {
						e.gameState.Player.X += input.Movement[0]
						e.gameState.Player.Y += input.Movement[1]
					}
					e.gameState.Player.CurrentAction = input.Action
				}
			default:
			}

			e.mu.Unlock()

		case <-e.closeChan:
			return
		}
	}
}

// acceptConnections handles incoming client connections
func (e *NEOGEOEmulator) acceptConnections() {
	go e.acceptCommandConnections()
	go e.acceptInputConnections()
}

// acceptCommandConnections accepts connections on the game logic port
func (e *NEOGEOEmulator) acceptCommandConnections() {
	if e.commandListener == nil {
		return
	}

	for {
		select {
		case <-e.closeChan:
			return
		default:
		}

		conn, err := e.commandListener.Accept()
		if err != nil {
			continue
		}

		deviceID := fmt.Sprintf("client-%d", len(e.connectedClients))
		client := &EmulatorClient{
			deviceID:    deviceID,
			conn:        conn,
			frameBuffer: make(chan *arcade.GameFrame, 10),
			joinedAt:    time.Now(),
		}

		e.mu.Lock()
		e.connectedClients[deviceID] = client
		e.mu.Unlock()

		fmt.Printf("✓ Client connected: %s\n", deviceID)
	}
}

// acceptInputConnections accepts connections on the input port
func (e *NEOGEOEmulator) acceptInputConnections() {
	if e.inputListener == nil {
		return
	}

	for {
		select {
		case <-e.closeChan:
			return
		default:
		}

		conn, err := e.inputListener.Accept()
		if err != nil {
			continue
		}

		go e.handleInputConnection(conn)
	}
}

// handleInputConnection processes input from a connected controller
func (e *NEOGEOEmulator) handleInputConnection(conn net.Conn) {
	defer conn.Close()

	// TODO: Read input frames from connection and process
}

// broadcastState sends game state to all connected clients
func (e *NEOGEOEmulator) broadcastState() {
	for {
		select {
		case frame := <-e.broadcastChannel:
			e.mu.RLock()
			clients := make(map[string]*EmulatorClient)
			for k, v := range e.connectedClients {
				clients[k] = v
			}
			e.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.frameBuffer <- frame:
				default:
					// Client buffer full, skip
				}
			}

		case <-e.closeChan:
			return
		}
	}
}
