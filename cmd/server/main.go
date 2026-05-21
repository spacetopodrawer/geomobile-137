package main

import (
	"cadastre_ia/internal/api"
	"cadastre_ia/internal/config"
	"cadastre_ia/internal/storage"
	"cadastre_ia/internal/websocket"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Parse command-line flags
	port := flag.String("port", "8080", "HTTP server port")
	dbMode := flag.String("db", "mock", "Database mode: 'mock' or 'postgres'")
	flag.Parse()

	log.Printf("🚀 GEO-MOBILE137 CADASTRAL SERVER")
	log.Printf("📡 Port: %s | DB Mode: %s", *port, *dbMode)

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database based on mode
	var dbPool *pgxpool.Pool
	var err error
	var store *storage.Storage

	if *dbMode == "postgres" {
		log.Printf("📦 Connecting to PostgreSQL...")
		dbPool, err = initDatabase(cfg)
		if err != nil {
			log.Fatalf("❌ Failed to initialize database: %v", err)
		}
		defer dbPool.Close()
		log.Printf("✅ PostgreSQL connection established")
		store = storage.NewStorage(dbPool)
		defer store.Close()
	} else {
		log.Printf("📦 Using mock database (test mode)...")
		store = &storage.Storage{}
		log.Printf("✅ Mock database ready")
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Printf("✅ WebSocket hub started")

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.Default()

	// Setup routes
	api.SetupRoutes(router, store, wsHub, cfg)
	log.Printf("✅ Routes configured")

	// Health check endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("🚀 Server listening on http://0.0.0.0:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("⚠️  Shutdown signal received: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server shutdown error: %v", err)
	}

	log.Printf("✅ Server stopped gracefully")
}

// initDatabase creates a PostgreSQL connection pool
func initDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
