package arcade

// ArcadeEmulator is the generic interface for any arcade system emulator
type ArcadeEmulator interface {
	// Lifecycle
	Start() error
	Stop() error
	IsRunning() bool

	// Game loop & state
	GetFrame() *GameFrame
	GetStatus() map[string]interface{}
	GetSystem() *SystemInfo

	// Network
	ConnectRemoteEmulator(deviceID, address string) error
	GetConnectedClients() []string
	BroadcastFrame(frame *GameFrame) error

	// Input
	ApplyInput(input *GameInput) error
	GetInputBuffer() chan *GameInput

	// Sync integration (placeholder for external sync service)
	// GetSyncManager() *SyncManager
	// ApplySyncOperation(op *Operation) error

	// Statistics
	GetFrameCount() uint64
	GetFPS() int
}

// ROMCompiler is the generic interface for compiling game state to ROM format
type ROMCompiler interface {
	// Compilation
	Compile(gameState *GameState, outputFile string) error
	GetFormat() string        // "neogeo", "mame", "fbneo", etc.
	GetSpec() *SystemSpec
	GetROMInfo(romPath string) (*ROMInfo, error)

	// Asset handling
	CompileSprites(spriteData []byte, paletteID uint8) ([]byte, error)
	CompileAudio(audioData []byte) ([]byte, error)
}

// ControllerMapping is the generic interface for input mapping
type ControllerMapping interface {
	// Input processing
	PushInput(rawInput uint32) error
	GetGameInput() *GameInput
	MapButton(physical string, logical string) error

	// Query
	GetButtonMap() map[string]string
	GetSupportedDirections() int // 4 (4-dir), 8 (8-dir), etc.
	GetActionButtons() []string

	// Calibration
	CalibrateAxis(axisName string, minValue, maxValue uint16) error
}

// SystemFactory is a factory function for creating system instances
type SystemFactory func(basePort int, romPath string) (ArcadeEmulator, error)

// ROMInfo contains metadata about a compiled ROM
type ROMInfo struct {
	Format      string
	Size        int64
	CRC32       uint32
	Title       string
	ReleaseDate string
	Resolution  [2]int
	ColorDepth  int
	AudioChip   string
}
