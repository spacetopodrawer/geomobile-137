# Arcade Emulator Framework v0.3.0

## Overview

The **Arcade Emulator Framework** is an industrial-grade, multi-system arcade emulation platform built into the Cadastre_IA engine. It provides a unified, interface-based abstraction layer supporting 6 classic arcade systems with equal priority and feature parity.

**Core Principle**: All supported arcade systems are implemented with identical code quality, interface compliance, and runtime capabilities. No system has exclusive privileges or hierarchical advantages.

## Supported Systems (Phase 4.5A)

| System | Resolution | Color Depth | Input | ROM Format | Status |
|--------|-----------|------------|-------|-----------|---------|
| NEO-GEO | 320×224 | 4-bit (16 colors) | 8-dir + 4 buttons | .bin | ✓ Core |
| MAME | 320×240 | 8-bit (256 colors) | 8-dir + 6 buttons | .zip | ✓ Implemented |
| FBNeo | 320×224 | 8-bit (256 colors) | 8-dir + 6 buttons | .zip | ✓ Implemented |
| Atari 2600 | 160×192 | 4-bit | 8-dir + 1 button | .bin | ⧖ Planned |
| Commodore 64 | 320×200 | 4-bit | 8-dir + 1 button | .bin | ⧖ Planned |
| CPS1/CPS2 | 384×224 | 8-bit | 8-dir + 6 buttons | .zip | ⧖ Planned |

## Architecture

### Core Interfaces

The framework defines three fundamental Go interfaces in `pkg/arcade/emulator.go`:

#### ArcadeEmulator
```go
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

    // Statistics
    GetFrameCount() uint64
    GetFPS() int
}
```

Every arcade system implements this interface identically. No system-specific methods exist.

#### ROMCompiler
```go
type ROMCompiler interface {
    Compile(gameState *GameState, outputFile string) error
    GetFormat() string
    GetSpec() *SystemSpec
    GetROMInfo(romPath string) (*ROMInfo, error)
    CompileSprites(spriteData []byte, paletteID uint8) ([]byte, error)
    CompileAudio(audioData []byte) ([]byte, error)
}
```

Each system's compiler handles format-specific ROM generation:
- **NEO-GEO**: Binary .bin format with header
- **MAME**: ZIP archive (cpu.rom, graphics.rom, sound.rom)
- **FBNeo**: ZIP archive format

#### ControllerMapping
```go
type ControllerMapping interface {
    PushInput(rawInput uint32) error
    GetGameInput() *GameInput
    MapButton(physical string, logical string) error
    GetButtonMap() map[string]string
    GetSupportedDirections() int
    GetActionButtons() []string
    CalibrateAxis(axisName string, minValue, maxValue uint16) error
}
```

Input mapping is system-specific but follows identical interface contracts.

### SystemRegistry (Factory Pattern)

Located in `pkg/arcade/registry.go`, the `SystemRegistry` manages dynamic system loading and caching:

```go
type SystemRegistry struct {
    mu        sync.RWMutex
    systems   map[string]SystemFactory
    instances map[string]ArcadeEmulator
    specs     map[string]*SystemSpec
}
```

**Key Functions**:
- `RegisterSystem(systemID, factory, spec)` - Register a new arcade system
- `GetArcadeEmulator(systemID, basePort, romPath)` - Factory method with instance caching
- `ListAvailableSystems()` - Enumerate all registered systems
- `GetSystemSpec(systemID)` - Retrieve system specifications

### System Specifications

System metadata is defined in `pkg/arcade/system.go` with the `SystemSpec` structure:

```go
type SystemSpec struct {
    SystemID      string
    Name          string
    Resolution   [2]int
    ColorDepth    int
    RefreshRate   int
    Joystick      JoystickSpec
    AudioChip     string
    MaxSprites    int
    CPUArchitecture string
    ROMFormat     string
    Priority      int
}
```

All systems are defined with equal priority (no hierarchical ordering).

## System Implementation

Each system is located in `pkg/arcade/systems/{systemID}/` with standardized structure:

### Files Per System

1. **system.go** - Factory function and SystemInfo registration
2. **emulator.go** - ArcadeEmulator implementation with game loop
3. **compiler.go** - ROMCompiler implementation for format-specific compilation
4. **controller.go** - ControllerMapping for input handling
5. **protocol.go** - (Optional) Network protocol definitions

### Example: NEO-GEO System

**File**: `pkg/arcade/systems/neogeo/system.go`

```go
func Factory(basePort int, romPath string) (arcade.ArcadeEmulator, error) {
    return NewNEOGEOEmulator(basePort, romPath)
}

func Info() *arcade.SystemInfo {
    return &arcade.SystemInfo{
        SystemID: "neogeo",
        Name: "SNK NEO-GEO",
        Resolution: [2]int{320, 224},
        // ... additional specs
    }
}
```

## Game Loop & Synchronization

### Frame-Based Execution

Each emulator runs at **60 FPS** with frame-locked game loops:

```go
func (ne *NEOGEOEmulator) gameLoop() {
    ticker := time.NewTicker(time.Duration(1000000/60) * time.Microsecond)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            ne.mu.Lock()
            ne.frameCount++
            // Update game logic
            ne.mu.Unlock()
        case <-ne.closeChan:
            return
        }
    }
}
```

### Frame Structure

GameFrame contains:
- `FrameID`: Monotonic frame counter
- `Timestamp`: Wall-clock time of frame generation
- `PlayerState`: Current player position and action
- `Objects`: All game objects in scene
- `SourceDevice`: Device ID generating frame (for P2P sync)

### Network Broadcasting

Emulators can broadcast frames to connected peers:

```go
func (ne *NEOGEOEmulator) BroadcastFrame(frame *arcade.GameFrame) error {
    select {
    case ne.broadcastChannel <- frame:
        return nil
    default:
        return fmt.Errorf("broadcast channel full")
    }
}
```

Broadcasting runs in a separate goroutine and respects backpressure.

## Configuration

The `config.yaml` file controls arcade framework behavior:

```yaml
arcade:
  enabled: true
  
  systems:
    neogeo:
      enabled: true
      priority: 1
      privileged: true  # Primary system (hardcoded in main.go)
      port: 9001
    
    mame:
      enabled: true
      priority: 2
      port: 9003
    
    fbneo:
      enabled: true
      priority: 2
      port: 9005

  display:
    width: 320
    height: 224
    fps: 60
```

## HTTP Endpoints

The server exposes arcade-specific endpoints:

- **GET /health** - Server health status
- **GET /status** - System status (sync, game, storage, arcade)
- **GET /stats** - Detailed statistics (connected devices, arcade clients, game objects)
- **GET /arcade/status** - Arcade emulator status (frames, FPS, connected clients)
- **WS /ws** - WebSocket for P2P sync

## Testing

Comprehensive test suite in `cmd/test/arcade_framework_test.go` validates:

- Interface compliance across all systems
- Registry factory pattern behavior
- System specification retrieval
- Emulator lifecycle management
- Frame generation and broadcast

**Run tests**:
```bash
go test ./cmd/test -v
```

**Run benchmarks**:
```bash
go test ./cmd/test -bench=. -benchmem
```

## Project Naming Convention

All arcade framework components follow the project's branding structure:

- **Project Family**: `geo-mobile`
- **Engine**: `Cadastre_IA v0.3.0`
- **Framework**: `Arcade Emulator Framework v0.3.0`
- **Release Tag**: `v0.3.0-arcade-framework-week234-137`

## Compilation Requirements

- **Go Version**: 1.21+
- **CGO**: Enabled (SQLite3 storage backend)
- **Platform**: Windows 11 Pro, macOS, Linux

**Build**:
```bash
go build -o cadastreia-server ./cmd/server
```

## Performance Targets

- **Frame Rate**: 60 FPS per system (locked)
- **Input Latency**: <16ms (one frame @ 60 FPS)
- **Memory per Emulator**: ~50-100MB (system-dependent)
- **Network Sync**: <100ms (P2P WebSocket)

## Future Extensions (Phase 4.5B)

### Multi-ROM Compiler
Compile a single game state to multiple ROM formats simultaneously:
```
GameState → {NEO-GEO .bin, MAME .zip, FBNeo .zip}
```

### Cross-Platform Synchronization
Share game progression, inventory, and scores across different arcade systems within same game session.

### Universal Save Format
Standardized game state serialization supporting migration between systems.

## File Structure

```
C:\geomobile137-solo\
├── pkg/arcade/
│   ├── emulator.go              (Core interfaces)
│   ├── system.go                (SystemInfo, SystemSpec)
│   ├── registry.go              (SystemRegistry, factory)
│   ├── neogeo_compiler.go       (Legacy NEO-GEO compiler)
│   ├── control_mapping.go       (Input structures)
│   ├── systems/
│   │   ├── neogeo/              (NEO-GEO implementation)
│   │   │   ├── system.go
│   │   │   ├── emulator.go
│   │   │   ├── compiler.go
│   │   │   ├── controller.go
│   │   │   └── protocol.go
│   │   ├── mame/                (MAME implementation)
│   │   │   ├── system.go
│   │   │   ├── emulator.go
│   │   │   ├── compiler.go
│   │   │   └── controller.go
│   │   ├── fbneo/               (FBNeo implementation)
│   │   │   ├── system.go
│   │   │   ├── emulator.go
│   │   │   ├── compiler.go
│   │   │   └── controller.go
│   │   └── template/            (Implementation template)
│   │       └── system.go
│   └── ...
├── cmd/
│   ├── server/
│   │   └── main.go              (Server entry point with arcade init)
│   └── test/
│       └── arcade_framework_test.go
├── config.yaml                  (Arcade framework config)
└── ARCADE_FRAMEWORK.md          (This file)
```

## Version History

### v0.3.0 (Current)
- **Release Date**: May 2026
- **Focus**: Arcade Emulator Framework generalization
- **Systems**: NEO-GEO (core), MAME, FBNeo
- **Status**: Production ready for 3 systems

### v0.2.0
- Phase 3: NEO-GEO specific implementation (NeoRageX5)

### v0.1.0
- Phase 1-2: Core engine, game state, P2P sync

## Contributing

When adding new arcade systems:

1. Create `pkg/arcade/systems/{systemID}/` package
2. Implement all 3 interfaces: ArcadeEmulator, ROMCompiler, ControllerMapping
3. Register in `RegisterSystem()` call during init
4. Add system tests to `arcade_framework_test.go`
5. Document system specs in `pkg/arcade/system.go`
6. Update this file with system details

## License

See LICENSE file in repository root.
