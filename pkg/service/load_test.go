// Package service provides load testing for cadastral pipeline
package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"cadastre_ia/pkg/cadastre"
	"cadastre_ia/pkg/cache"
	"cadastre_ia/pkg/database"
	"cadastre_ia/pkg/events"
	"cadastre_ia/pkg/svg_codification"
)

// BenchmarkCadastreConversion benchmarks the conversion performance
func BenchmarkCadastreConversion(b *testing.B) {
	logger := log.New(log.Writer(), "[Bench] ", log.LstdFlags)
	ctx := context.Background()

	// Setup
	converter := NewCadConverter(FormatDWG, logger)
	codifier := cadastre.NewCodifier("lekie")
	tileCache := cache.NewMemoryCache(logger)
	db := database.NewMockDB(logger)
	eventBus := events.NewEventBus(logger)
	renderer := svg_codification.NewSVGRenderer(logger)
	tileGen := svg_codification.NewTileGenerator(tileCache, renderer, db, logger)
	adapter := NewCadastreAdapter(converter, codifier, tileGen, db, eventBus, logger)

	// Test data
	cadData := &RawCADData{
		Format:     FormatDWG,
		SourcePath: "bench_lekie.dwg",
		VertexData: generateTestVertices(100),
		Metadata:   map[string]interface{}{"source": "benchmark"},
	}

	// Reset timer after setup
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := adapter.ConvertAndStoreCadastralData(ctx, cadData, "lekie", "Lékié Central")
		if err != nil {
			b.Fatalf("Conversion failed: %v", err)
		}
	}
}

// TestLoadConversion tests conversion with multiple concurrent goroutines
func TestLoadConversion(t *testing.T) {
	logger := log.New(log.Writer(), "[Load] ", log.LstdFlags)
	ctx := context.Background()

	// Setup
	converter := NewCadConverter(FormatDWG, logger)
	codifier := cadastre.NewCodifier("lekie")
	tileCache := cache.NewMemoryCache(logger)
	db := database.NewMockDB(logger)
	eventBus := events.NewEventBus(logger)
	renderer := svg_codification.NewSVGRenderer(logger)
	tileGen := svg_codification.NewTileGenerator(tileCache, renderer, db, logger)
	adapter := NewCadastreAdapter(converter, codifier, tileGen, db, eventBus, logger)

	t.Run("Sequential Loading", func(t *testing.T) {
		sequentialLoadTest(t, adapter, logger, ctx, 10)
	})

	t.Run("Concurrent Loading", func(t *testing.T) {
		concurrentLoadTest(t, adapter, logger, ctx, 10, 4)
	})

	t.Run("High Volume Sequential", func(t *testing.T) {
		sequentialLoadTest(t, adapter, logger, ctx, 100)
	})
}

// TestTileCachePerformance tests cache hit/miss ratios
func TestTileCachePerformance(t *testing.T) {
	logger := log.New(log.Writer(), "[Cache] ", log.LstdFlags)
	ctx := context.Background()

	tileCache := cache.NewMemoryCache(logger)

	logger.Println("=== CACHE PERFORMANCE TEST ===\n")

	// Scenario 1: Cache miss (first access)
	logger.Println("Scenario 1: Cache Miss")
	start := time.Now()
	_, err := tileCache.Get(ctx, "z12_x2048_y1024")
	duration := time.Since(start)
	logger.Printf("  Expected error: %v (duration: %v)\n", err, duration)
	if err == nil {
		t.Fatalf("Expected cache miss")
	}

	// Scenario 2: Cache set
	logger.Println("\nScenario 2: Cache Set")
	testData := []byte("<svg>test tile data</svg>")
	start = time.Now()
	tileCache.Set(ctx, "z12_x2048_y1024", testData, 3600)
	duration = time.Since(start)
	logger.Printf("  ✓ Data cached (duration: %v)\n", duration)

	// Scenario 3: Cache hit
	logger.Println("\nScenario 3: Cache Hit")
	start = time.Now()
	cached, err := tileCache.Get(ctx, "z12_x2048_y1024")
	duration = time.Since(start)
	if err != nil {
		t.Fatalf("Cache hit failed: %v", err)
	}
	logger.Printf("  ✓ Cache hit (duration: %v)\n", duration)

	if string(cached) != string(testData) {
		t.Fatalf("Cache data mismatch")
	}

	// Scenario 4: High-frequency access (simulate tile server load)
	logger.Println("\nScenario 4: High-Frequency Access (1000 requests)")
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, _ = tileCache.Get(ctx, "z12_x2048_y1024")
	}
	totalDuration := time.Since(start)
	avgDuration := totalDuration / 1000
	logger.Printf("  Total: %v, Avg per request: %v\n", totalDuration, avgDuration)

	if avgDuration > 1*time.Millisecond {
		logger.Printf("  ⚠️  Warning: Cache access time exceeds target (<1ms)\n")
	} else {
		logger.Printf("  ✓ Cache access time within target (<1ms)\n")
	}
}

// TestDatabaseConcurrency tests concurrent database operations
func TestDatabaseConcurrency(t *testing.T) {
	logger := log.New(log.Writer(), "[DB] ", log.LstdFlags)
	ctx := context.Background()

	db := database.NewMockDB(logger)

	logger.Println("=== DATABASE CONCURRENCY TEST ===\n")

	// Create test entities
	var entities []cadastre.Entity
	for i := 0; i < 50; i++ {
		entity := cadastre.Entity{
			ID:        fmt.Sprintf("test_%d", i),
			Code:      fmt.Sprintf("bd_lekie_01_%03d_v1_xxxx", i),
			Type:      cadastre.EntityTypeBuilding,
			Region:    "lekie",
			AdminUnit: "Lékié Central",
			Geometry: map[string]interface{}{
				"type": "Polygon",
				"coordinates": [][][]float64{{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}},
			},
			Attributes: map[string]interface{}{"index": i},
		}
		entities = append(entities, entity)
	}

	logger.Printf("Storing %d entities...\n", len(entities))
	start := time.Now()
	if err := db.StoreCadastralEntities(ctx, entities); err != nil {
		t.Fatalf("Storage failed: %v", err)
	}
	duration := time.Since(start)
	logger.Printf("✓ Stored in %v (%.1f entities/sec)\n\n", duration, float64(len(entities))/duration.Seconds())

	// Concurrent reads
	logger.Println("Concurrent read test (10 goroutines, 100 reads each)...")
	start = time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount, failCount int

	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(groupID int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				entityIdx := (groupID*100 + i) % len(entities)
				_, err := db.GetEntity(ctx, entities[entityIdx].Code)
				mu.Lock()
				if err == nil {
					successCount++
				} else {
					failCount++
				}
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
	totalDuration := time.Since(start)
	logger.Printf("✓ Completed %d reads in %v\n", successCount+failCount, totalDuration)
	logger.Printf("  - Success: %d, Failed: %d\n", successCount, failCount)
	logger.Printf("  - Throughput: %.0f reads/sec\n\n", float64(successCount+failCount)/totalDuration.Seconds())

	if failCount > 0 {
		t.Logf("Warning: %d reads failed", failCount)
	}

	// Verify storage
	finalCount := db.GetCount()
	logger.Printf("Final entity count in DB: %d\n", finalCount)
	if finalCount != len(entities) {
		t.Fatalf("Entity count mismatch: expected %d, got %d", len(entities), finalCount)
	}
}

// TestTileGenerationPerformance measures tile generation speed
func TestTileGenerationPerformance(t *testing.T) {
	logger := log.New(log.Writer(), "[Tiles] ", log.LstdFlags)

	logger.Println("=== TILE GENERATION PERFORMANCE TEST ===\n")

	renderer := svg_codification.NewSVGRenderer(logger)

	// Create test entities
	testEntities := []interface{}{
		"entity_1", "entity_2", "entity_3", "entity_4", "entity_5",
	}

	// Test different zoom levels
	zoomLevels := []int{5, 10, 12, 15, 18}

	for _, z := range zoomLevels {
		start := time.Now()
		svg, err := renderer.RenderTile(context.Background(), z, 2048, 1024, testEntities)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Tile rendering failed at z=%d: %v", z, err)
		}

		logger.Printf("Zoom %2d: %v (%.1f KB)\n", z, duration, float64(len(svg))/1024)
	}

	logger.Println("\nTarget: <100ms per tile")
}

// Helper functions

func generateTestVertices(count int) []CADVertex {
	vertices := make([]CADVertex, count)
	for i := 0; i < count; i++ {
		// Generate coordinates in Lékié region (WGS84)
		vertices[i] = CADVertex{
			X: 3.8667 + float64(i%10)*0.001,
			Y: 3.5667 + float64(i/10)*0.001,
			Z: 0,
		}
	}
	return vertices
}

func sequentialLoadTest(t *testing.T, adapter *CadastreAdapter, logger *log.Logger, ctx context.Context, fileCount int) {
	logger.Printf("Sequential Load Test: %d files\n", fileCount)

	start := time.Now()
	var totalEntities int

	for i := 0; i < fileCount; i++ {
		cadData := &RawCADData{
			Format:     FormatDWG,
			SourcePath: fmt.Sprintf("sequential_test_%d.dwg", i),
			VertexData: generateTestVertices(50),
			Metadata:   map[string]interface{}{"file": i},
		}

		entitySet, err := adapter.ConvertAndStoreCadastralData(ctx, cadData, "lekie", "Lékié Central")
		if err != nil {
			t.Fatalf("Sequential conversion failed at file %d: %v", i, err)
		}

		totalEntities += len(entitySet.Entities)
		logger.Printf("  File %d: %d entities\n", i, len(entitySet.Entities))
	}

	totalDuration := time.Since(start)
	logger.Printf("✓ Sequential: %d files → %d entities in %v\n", fileCount, totalEntities, totalDuration)
	logger.Printf("  Throughput: %.1f files/sec, %.1f entities/sec\n\n",
		float64(fileCount)/totalDuration.Seconds(),
		float64(totalEntities)/totalDuration.Seconds())
}

func concurrentLoadTest(t *testing.T, adapter *CadastreAdapter, logger *log.Logger, ctx context.Context, fileCount int, concurrency int) {
	logger.Printf("Concurrent Load Test: %d files, %d goroutines\n", fileCount, concurrency)

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalEntities int
	var errorCount int

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < fileCount; i++ {
		wg.Add(1)
		go func(fileID int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			cadData := &RawCADData{
				Format:     FormatDWG,
				SourcePath: fmt.Sprintf("concurrent_test_%d.dwg", fileID),
				VertexData: generateTestVertices(50),
				Metadata:   map[string]interface{}{"file": fileID},
			}

			entitySet, err := adapter.ConvertAndStoreCadastralData(ctx, cadData, "lekie", "Lékié Central")

			mu.Lock()
			if err != nil {
				errorCount++
			} else {
				totalEntities += len(entitySet.Entities)
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	logger.Printf("✓ Concurrent: %d files → %d entities in %v\n", fileCount, totalEntities, totalDuration)
	logger.Printf("  Throughput: %.1f files/sec, %.1f entities/sec\n",
		float64(fileCount)/totalDuration.Seconds(),
		float64(totalEntities)/totalDuration.Seconds())
	if errorCount > 0 {
		logger.Printf("  ⚠️  Errors: %d\n", errorCount)
	}
	logger.Println()
}
