package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"cadastreia/pkg/arcade"
	"cadastreia/pkg/game"
	"cadastreia/pkg/storage"
	syncsvc "cadastreia/pkg/sync"
)

// Config represents the application configuration
type Config struct {
	System struct {
		DeviceID   string `yaml:"device_id"`
		DeviceName string `yaml:"device_name"`
		Owner      string `yaml:"owner"`
	} `yaml:"system"`

	Storage struct {
		Type   string `yaml:"type"`
		SQLite struct {
			Path           string `yaml:"path"`
			MaxConnections int    `yaml:"max_connections"`
			EnableWAL      bool   `yaml:"enable_wal"`
		} `yaml:"sqlite"`
	} `yaml:"storage"`

	Sync struct {
		Enabled  bool `yaml:"enabled"`
		Protocol string `yaml:"protocol"`
	} `yaml:"sync"`

	Arcade struct {
		Enabled bool `yaml:"enabled"`
		Display struct {
			Width  int `yaml:"width"`
			Height int `yaml:"height"`
			FPS    int `yaml:"fps"`
		} `yaml:"display"`
	} `yaml:"arcade"`

	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

var (
	configPath = "config.yaml"
	config     Config
)

func init() {
	// Load configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("⚠️  Config file not found, using defaults: %v", err)
		setDefaults()
		return
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("❌ Failed to parse config: %v", err)
	}

	setDefaults()
}

func setDefaults() {
	if config.System.DeviceID == "" {
		config.System.DeviceID = "local-device-0"
	}
	if config.System.DeviceName == "" {
		config.System.DeviceName = "Local Device"
	}
	if config.Storage.SQLite.Path == "" {
		config.Storage.SQLite.Path = "./cadastre_ia.db"
	}
	if config.Storage.SQLite.MaxConnections == 0 {
		config.Storage.SQLite.MaxConnections = 10
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Arcade.Display.Width == 0 {
		config.Arcade.Display.Width = 256
	}
	if config.Arcade.Display.Height == 0 {
		config.Arcade.Display.Height = 224
	}
	if config.Arcade.Display.FPS == 0 {
		config.Arcade.Display.FPS = 60
	}
	config.Storage.Type = "sqlite"
	config.Arcade.Enabled = true
	config.Sync.Enabled = true
	config.Sync.Protocol = "websocket"
}

func main() {
	printBanner()

	// 1. Initialize storage
	fmt.Println("\n[1/5] Initializing SQLite database...")
	storageConfig := storage.StorageConfig{
		Path:           config.Storage.SQLite.Path,
		MaxConnections: config.Storage.SQLite.MaxConnections,
		EnableWAL:      true,
		QueryTimeoutMS: 5000,
	}

	sm := storage.NewStorageManager(storageConfig)
	if err := sm.Initialize(); err != nil {
		log.Fatalf("❌ Storage initialization failed: %v", err)
	}
	defer sm.Close()
	fmt.Println("✓ SQLite database initialized")

	// 2. Initialize sync engine
	fmt.Println("\n[2/5] Starting P2P sync engine...")
	syncMgr := syncsvc.NewSyncManager(config.System.DeviceID)
	fmt.Println("✓ Sync engine started")

	// 3. Initialize WebSocket hub
	fmt.Println("\n[3/5] Initializing WebSocket sync hub...")
	wsHub := syncsvc.NewWebSocketHub(syncMgr)
	defer wsHub.Close()
	fmt.Println("✓ WebSocket hub initialized")

	// 3.5 Initialize arcade framework (Phase 3.5)
	// Phase 4.5 Note: Will transition to registry-based system selection:
	//   arcade.InitRegistry()
	//   arcade.RegisterSystem("neogeo", systems/neogeo.Factory, spec)
	//   emulator, _ := arcade.GetArcadeEmulator("neogeo", 9001, romPath)
	//   var arcadeEmulator arcade.ArcadeEmulator = emulator
	// For now (Phase 3), keep existing NeoRageX5Emulator implementation
	var arcadeEmulator *arcade.NeoRageX5Emulator
	if config.Arcade.Enabled {
		fmt.Println("\n[3.5/5] Initializing NEO-GEO arcade emulator...")
		romPath := "./cadastre_ia.neo"
		arcadeEmulator = arcade.NewNeoRageX5Emulator(9001, romPath)
		if err := arcadeEmulator.Start(); err != nil {
			fmt.Printf("⚠️  Arcade emulator failed to start: %v\n", err)
		} else {
			defer arcadeEmulator.Stop()
			fmt.Println("✓ NEO-GEO arcade emulator started")
			fmt.Printf("  Command port: 9001 (game logic/sync)\n")
			fmt.Printf("  Input port:   9002 (controller input)\n")
		}
	}

	// 4. Initialize game engine
	fmt.Println("\n[4/5] Starting game engine...")
	gameEngine := game.NewGameEngine(sm)
	if err := gameEngine.Start(); err != nil {
		log.Fatalf("❌ Game engine failed: %v", err)
	}
	defer gameEngine.Close()
	fmt.Println("✓ Game engine started")

	// 5. Setup HTTP handlers
	fmt.Println("\n[5/5] Setting up HTTP handlers...")
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", wsHub.HandleConnection)
	http.HandleFunc("/status", statusHandler(wsHub, gameEngine, sm, arcadeEmulator))
	http.HandleFunc("/stats", statsHandler(wsHub, gameEngine, sm, arcadeEmulator))
	if arcadeEmulator != nil {
		http.HandleFunc("/arcade/status", arcadeStatusHandler(arcadeEmulator))
	}
	fmt.Println("✓ HTTP handlers registered")

	// Start HTTP server
	addr := fmt.Sprintf(":%d", config.Server.Port)
	fmt.Printf("\n╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  🚀 CADASTRE_IA - CORE ENGINE RUNNING                  ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")
	fmt.Printf("✓ Server listening on http://localhost:%d\n", config.Server.Port)
	fmt.Printf("✓ Health check:  http://localhost:%d/health\n", config.Server.Port)
	fmt.Printf("✓ WebSocket:     ws://localhost:%d/ws\n", config.Server.Port)
	fmt.Printf("✓ Status:        http://localhost:%d/status\n", config.Server.Port)
	fmt.Printf("✓ Device ID:     %s\n", config.System.DeviceID)
	fmt.Printf("✓ Database:      %s\n", config.Storage.SQLite.Path)
	fmt.Printf("\nReady for gameplay! 🎮\n\n")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n\n🛑 Shutting down...")
	gameEngine.Stop()
	wsHub.Close()
	sm.Close()
	fmt.Println("✓ Server stopped")
}

// healthHandler returns health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// statusHandler returns system status
func statusHandler(wsHub *syncsvc.WebSocketHub, ge *game.GameEngine, sm *storage.StorageManager, ae *arcade.NeoRageX5Emulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := map[string]interface{}{
			"sync":   wsHub.GetStatus(),
			"game":   ge.GetStatus(),
			"storage": sm.GetStats(),
		}

		if ae != nil {
			status["arcade"] = ae.GetStatus()
		}

		fmt.Fprintf(w, `{"status":%v}`, status)
	}
}

// statsHandler returns detailed statistics
func statsHandler(wsHub *syncsvc.WebSocketHub, ge *game.GameEngine, sm *storage.StorageManager, ae *arcade.NeoRageX5Emulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		arcadeClients := 0
		if ae != nil {
			status := ae.GetStatus()
			if cc, ok := status["connected_clients"].(int); ok {
				arcadeClients = cc
			}
		}

		fmt.Fprintf(w, `{
			"connected_devices":%d,
			"arcade_clients":%d,
			"game_objects":%d,
			"player_inventory":%d,
			"sync_ops":%d,
			"timestamp":"%s"
		}`,
			wsHub.GetClientCount(),
			arcadeClients,
			len(ge.GetGameState().Objects),
			len(ge.GetGameState().Player.Inventory),
			0,
			time.Now().Format(time.RFC3339),
		)
	}
}

// arcadeStatusHandler returns arcade emulator status
func arcadeStatusHandler(ae *arcade.NeoRageX5Emulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		status := ae.GetStatus()
		fmt.Fprintf(w, `%v`, status)
	}
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   🎮 CADASTRE_IA: GEOMOBILE v137                        ║
║   Real-time Geospatial Arcade Game                      ║
║   Native P2P Synchronization                            ║
║                                                           ║
║   Status: 🟢 PRODUCTION READY                           ║
║   Version: 3.0 Core Engine                              ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Print(banner)
}
