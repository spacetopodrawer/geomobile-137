// cadastre-server is the main entry point for the geo-mobile137 cadastral server
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cadastre_ia/pkg/cadastre"
	"cadastre_ia/pkg/cache"
	"cadastre_ia/pkg/database"
	"cadastre_ia/pkg/events"
	"cadastre_ia/pkg/handlers"
	"cadastre_ia/pkg/service"
	"cadastre_ia/pkg/svg_codification"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8080, "HTTP server port")
	dbMode := flag.String("db", "mock", "Database mode: 'mock' or 'postgres'")
	dbConn := flag.String("db-conn", "postgres://user:password@localhost/cadastre_ia", "PostgreSQL connection string")
	logFile := flag.String("log", "", "Log file path (empty = stdout)")

	flag.Parse()

	// Setup logging
	var logOutput *os.File = os.Stdout
	if *logFile != "" {
		var err error
		logOutput, err = os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer logOutput.Close()
	}

	logger := log.New(logOutput, "[cadastre-server] ", log.LstdFlags|log.Lshortfile)

	logger.Println("="*70)
	logger.Println("🚀 GEO-MOBILE137 CADASTRAL SERVER")
	logger.Println("Phase 2.2 — CAD Converter Integration")
	logger.Println("="*70)
	logger.Printf("Starting server on port %d\n", *port)
	logger.Printf("Database mode: %s\n", *dbMode)

	ctx := context.Background()

	// Initialize components
	logger.Println("\n📦 Initializing components...")

	// 1. Database layer
	var db database.CadastreDB
	if *dbMode == "postgres" {
		logger.Println("  → Connecting to PostgreSQL...")
		pgdb, err := database.NewPostgresDB(*dbConn, logger)
		if err != nil {
			logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer pgdb.Close()
		db = pgdb
		logger.Println("    ✓ PostgreSQL connected")
	} else {
		logger.Println("  → Using mock database (testing mode)...")
		db = database.NewMockDB(logger)
		logger.Println("    ✓ Mock database ready")
	}

	// 2. Cache layer
	logger.Println("  → Setting up cache...")
	tileCache := cache.NewMemoryCache(logger)
	logger.Println("    ✓ Memory cache ready")

	// 3. Event bus
	logger.Println("  → Setting up event bus...")
	eventBus := events.NewEventBus(logger)
	logger.Println("    ✓ Event bus ready")

	// 4. SVG rendering
	logger.Println("  → Initializing SVG renderer...")
	renderer := svg_codification.NewSVGRenderer(logger)
	logger.Println("    ✓ SVG renderer ready")

	// 5. Tile generation
	logger.Println("  → Setting up tile generator...")
	tileGen := svg_codification.NewTileGenerator(tileCache, renderer, db, logger)
	logger.Println("    ✓ Tile generator ready")

	// 6. CAD converter
	logger.Println("  → Initializing CAD converter...")
	converter := service.NewCadConverter(service.FormatDWG, logger)
	logger.Println("    ✓ CAD converter ready")

	// 7. Cadastre codifier
	logger.Println("  → Initializing codifier...")
	codifier := cadastre.NewCodifier("lekie")
	logger.Println("    ✓ Codifier ready")

	// 8. Cadastre adapter
	logger.Println("  → Initializing cadastre adapter...")
	adapter := service.NewCadastreAdapter(converter, codifier, tileGen, db, eventBus, logger)
	logger.Println("    ✓ Cadastre adapter ready")

	logger.Println("\n✅ All components initialized successfully\n")

	// Initialize HTTP handlers
	logger.Println("📍 Registering API endpoints...")
	cadastreHandlers := handlers.NewCadastreHandlers(adapter, tileGen, converter, logger)

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, cadastreHandlers)

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"cadastre-server","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	logger.Println("  ✓ GET  /api/v1/cadastre/tiles/{z}/{x}/{y}")
	logger.Println("  ✓ POST /api/v1/cadastre/convert")
	logger.Println("  ✓ POST /api/v1/cadastre/decode")
	logger.Println("  ✓ POST /api/v1/cadastre/validate")
	logger.Println("  ✓ GET  /api/v1/cadastre/legend")
	logger.Println("  ✓ GET  /api/v1/cadastre/health")
	logger.Println("  ✓ GET  /health")

	// Demo: Load test data if in mock mode
	if *dbMode == "mock" {
		logger.Println("\n🧪 Demo: Loading test cadastral data...")
		loadDemoData(ctx, adapter, logger)
	}

	// Start HTTP server
	logger.Printf("\n🌐 Starting HTTP server on port %d...\n", *port)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Server error: %v", err)
	}
}

// loadDemoData loads demo cadastral data for testing
func loadDemoData(ctx context.Context, adapter *service.CadastreAdapter, logger *log.Logger) {
	// Create sample CAD data (Lékié region, Cameroon)
	cadData := &service.RawCADData{
		Format:     service.FormatDWG,
		SourcePath: "demo_lekie_2021.dwg",
		VertexData: []service.CADVertex{
			// Building 1: Lékié Central (3.8667, 3.5667)
			{X: 3.8667, Y: 3.5667, Z: 0},
			{X: 3.8670, Y: 3.5667, Z: 0},
			{X: 3.8670, Y: 3.5670, Z: 0},
			{X: 3.8667, Y: 3.5670, Z: 0},
		},
		Metadata: map[string]interface{}{
			"source": "demo",
			"date":   "2026-05-11",
		},
	}

	entitySet, err := adapter.ConvertAndStoreCadastralData(ctx, cadData, "lekie", "Lékié Central")
	if err != nil {
		logger.Printf("⚠️  Demo data loading failed: %v\n", err)
		return
	}

	logger.Printf("✓ Loaded %d demo entities\n", len(entitySet.Entities))

	for _, entity := range entitySet.Entities {
		logger.Printf("  - %s (code: %s)\n", entity.ID, entity.Code)
	}

	logger.Println("\n📍 Try these endpoints to test:")
	logger.Println("  curl http://localhost:8080/api/v1/cadastre/health")
	logger.Println("  curl http://localhost:8080/api/v1/cadastre/legend")
	logger.Println("  curl http://localhost:8080/api/v1/cadastre/tiles/12/2048/1024")
}
