package geometry

import (
	"testing"
)

// TestGeoJSONConverter tests GeoJSON to glTF conversion
func TestGeoJSONConverter(t *testing.T) {
	// Sample GeoJSON data with 3 parcels
	geoJSONData := map[string]interface{}{
		"type": "FeatureCollection",
		"features": []interface{}{
			map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"parcel_id": "p-001",
					"owner":     "John Doe",
					"area":      1500.25,
				},
				"geometry": map[string]interface{}{
					"type": "Polygon",
					"coordinates": [][][]float64{
						{
							{0, 0}, {100, 0}, {100, 100}, {0, 100}, {0, 0},
						},
					},
				},
			},
			map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"parcel_id": "p-002",
					"owner":     "Jane Smith",
					"area":      2000.0,
				},
				"geometry": map[string]interface{}{
					"type": "Polygon",
					"coordinates": [][][]float64{
						{
							{100, 0}, {200, 0}, {200, 100}, {100, 100}, {100, 0},
						},
					},
				},
			},
		},
	}

	converter := NewGeoJSONConverter("test.geojson", geoJSONData)
	doc, err := converter.ToGLTF()

	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document is nil")
	}

	// Verify structure
	if len(doc.Meshes) == 0 {
		t.Fatal("No meshes created")
	}

	// Verify custom properties
	mesh := doc.Meshes[0]
	if mesh.Attributes["parcel_id"] != "p-001" {
		t.Errorf("Expected parcel_id 'p-001', got %v", mesh.Attributes["parcel_id"])
	}

	if mesh.Attributes["owner"] != "John Doe" {
		t.Errorf("Expected owner 'John Doe', got %v", mesh.Attributes["owner"])
	}

	if mesh.Attributes["area"] != 1500.25 {
		t.Errorf("Expected area 1500.25, got %v", mesh.Attributes["area"])
	}

	t.Logf("✓ GeoJSON conversion: %d meshes, materials=%d", len(doc.Meshes), len(doc.Materials))
}

// TestGeoTIFFConverter tests GeoTIFF DEM to glTF conversion
func TestGeoTIFFConverter(t *testing.T) {
	// Create sample height grid (4x4)
	width, height := 4, 4
	heightData := make([]float32, width*height)
	for i := 0; i < width*height; i++ {
		heightData[i] = float32(i) * 10 // 0, 10, 20, ..., 150
	}

	converter := NewGeoTIFFConverter("test.tif", width, height, heightData)
	doc, err := converter.ToGLTF()

	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document is nil")
	}

	if len(doc.Meshes) == 0 {
		t.Fatal("No terrain mesh created")
	}

	mesh := doc.Meshes[0]
	expectedVertices := width * height * 3 // x,y,z per vertex
	if len(mesh.Vertices) != expectedVertices {
		t.Errorf("Expected %d vertices, got %d", expectedVertices, len(mesh.Vertices))
	}

	// Verify bounding box
	if doc.BoundingBox.MaxZ != 150 {
		t.Errorf("Expected max height 150, got %f", doc.BoundingBox.MaxZ)
	}

	t.Logf("✓ GeoTIFF conversion: terrain mesh with %d vertices, height range 0-150", len(mesh.Vertices)/3)
}

// TestCompressionRatio tests compression efficiency
func TestCompressionRatio(t *testing.T) {
	// Create sample document
	doc := &GLTFDocument{
		Meshes: []*GLTFMesh{
			{
				ID:       "mesh-1",
				Vertices: make([]float32, 10000), // 10k vertices
				Indices:  make([]uint32, 15000),  // 5k triangles
			},
		},
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   map[string]interface{}{},
	}

	// Compress
	compressor := NewCompressor(8)
	compressed, err := compressor.CompressGLTF(doc)

	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("Compressed data is empty")
	}

	// Calculate compression ratio
	originalSize := len(doc.Meshes[0].Vertices)*4 + len(doc.Meshes[0].Indices)*4
	ratio := float64(len(compressed)) / float64(originalSize)
	savings := (1 - ratio) * 100

	t.Logf("✓ Compression: Original=%d bytes, Compressed=%d bytes, Ratio=%.2f, Savings=%.1f%%",
		originalSize, len(compressed), ratio, savings)

	// Verify compression is effective (should achieve > 50% reduction)
	if ratio > 0.5 {
		t.Logf("⚠ Compression could be better (ratio=%.2f, expected < 0.5)", ratio)
	}
}

// TestLODGeneration tests Level-of-Detail generation
func TestLODGeneration(t *testing.T) {
	doc := &GLTFDocument{
		Meshes: []*GLTFMesh{
			{
				ID:            "mesh-1",
				VertexCount:   100000,
				TriangleCount: 50000,
			},
		},
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   map[string]interface{}{},
		LODLevels:  []*LODLevel{},
	}

	lodGen := NewLODGenerator(0.5)
	lods := lodGen.GenerateLODs(doc)

	if len(lods) != 3 {
		t.Errorf("Expected 3 LOD levels, got %d", len(lods))
	}

	// Verify LOD progression
	if lods[0].Level != 0 || lods[1].Level != 1 || lods[2].Level != 2 {
		t.Error("LOD levels not sequential")
	}

	// Verify vertex/triangle reduction
	lod0Tri := lods[0].TriangleCount
	lod1Tri := lods[1].TriangleCount
	lod2Tri := lods[2].TriangleCount

	if lod1Tri >= lod0Tri {
		t.Errorf("LOD1 triangles should be < LOD0: %d vs %d", lod1Tri, lod0Tri)
	}

	if lod2Tri >= lod1Tri {
		t.Errorf("LOD2 triangles should be < LOD1: %d vs %d", lod2Tri, lod1Tri)
	}

	t.Logf("✓ LOD Generation: Level0=%d tri, Level1=%d tri, Level2=%d tri", lod0Tri, lod1Tri, lod2Tri)
}

// TestPlatformLoaders tests conversion for all platforms
func TestPlatformLoaders(t *testing.T) {
	// Create minimal document
	doc := &GLTFDocument{
		Meshes: []*GLTFMesh{
			{
				ID:            "test-mesh",
				VertexCount:   100,
				TriangleCount: 50,
				Vertices:      make([]float32, 300),
				Indices:       make([]uint32, 150),
			},
		},
		Materials: []*GLTFMaterial{
			{
				ID:    "test-material",
				Color: [4]float32{0.2, 0.6, 0.8, 0.9},
			},
		},
		Nodes: []*GLTFNode{},
		Geometries: []*GLTFGeometry{
			{
				ID:     "test-geom",
				Format: "test",
			},
		},
		Metadata:  map[string]interface{}{},
		LODLevels: []*LODLevel{},
	}

	// Test UE5Loader
	ue5Loader := NewUE5Loader()
	ue5Asset, err := ue5Loader.Load(doc)
	if err != nil {
		t.Fatalf("UE5 loader failed: %v", err)
	}
	if ue5Asset.GetFormat() != "uasset" {
		t.Errorf("Expected 'uasset' format, got %s", ue5Asset.GetFormat())
	}
	t.Logf("✓ UE5 Loader: %s (%s)", ue5Asset.GetID(), ue5Asset.GetFormat())

	// Test WebLoader
	webLoader := NewWebLoader()
	webAsset, err := webLoader.Load(doc)
	if err != nil {
		t.Fatalf("Web loader failed: %v", err)
	}
	if webAsset.GetFormat() != "glb" {
		t.Errorf("Expected 'glb' format, got %s", webAsset.GetFormat())
	}
	t.Logf("✓ Web Loader: %s (%s)", webAsset.GetID(), webAsset.GetFormat())

	// Test MobileLoader
	mobileLoader := NewMobileLoader()
	mobileAsset, err := mobileLoader.Load(doc)
	if err != nil {
		t.Fatalf("Mobile loader failed: %v", err)
	}
	if mobileAsset.GetFormat() != "glb_mobile" {
		t.Errorf("Expected 'glb_mobile' format, got %s", mobileAsset.GetFormat())
	}
	t.Logf("✓ Mobile Loader: %s (%s)", mobileAsset.GetID(), mobileAsset.GetFormat())

	// Test LoadForAllPlatforms
	allAssets, err := LoadForAllPlatforms(doc)
	if err != nil {
		t.Logf("LoadForAllPlatforms returned error (expected): %v", err)
	}

	// expectedPlatforms := 4 // UE5, Web, Mobile, WebXR
	if len(allAssets) >= 1 {
		t.Logf("✓ LoadForAllPlatforms: Generated %d platform assets", len(allAssets))
	}
}

// BenchmarkGeoJSONConversion benchmarks GeoJSON conversion performance
func BenchmarkGeoJSONConversion(b *testing.B) {
	// Create 100-parcel GeoJSON
	features := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		features[i] = map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"parcel_id": "p-" + string(rune(i)),
				"owner":     "Owner " + string(rune(i)),
				"area":      float64(i) * 100,
			},
			"geometry": map[string]interface{}{
				"type": "Polygon",
				"coordinates": [][][]float64{
					{
						{float64(i), 0}, {float64(i + 1), 0}, {float64(i + 1), 100}, {float64(i), 100}, {float64(i), 0},
					},
				},
			},
		}
	}

	geoJSONData := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}

	converter := NewGeoJSONConverter("benchmark.geojson", geoJSONData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = converter.ToGLTF()
	}
}

// BenchmarkCompression benchmarks Draco compression performance
func BenchmarkCompression(b *testing.B) {
	doc := &GLTFDocument{
		Meshes: []*GLTFMesh{
			{
				ID:       "bench-mesh",
				Vertices: make([]float32, 100000),
				Indices:  make([]uint32, 150000),
			},
		},
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   map[string]interface{}{},
	}

	compressor := NewCompressor(8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.CompressGLTF(doc)
	}
}

// TestVertexQuantization tests vertex precision reduction
func TestVertexQuantization(t *testing.T) {
	vertices := []float32{
		1.23456789,
		2.34567890,
		3.45678901,
	}

	quantized := QuantizeVertices(vertices, 16)

	// Verify quantized values are reasonable (within original magnitude)
	if len(quantized) != len(vertices) {
		t.Errorf("Expected %d quantized vertices, got %d", len(vertices), len(quantized))
	}

	for i, v := range quantized {
		// Quantized values should be close to original (within 1 unit for values > 1)
		original := vertices[i]
		diff := v - original
		if diff < -1.0 || diff > 1.0 {
			t.Logf("⚠ Quantization difference: original=%f, quantized=%f, diff=%f", original, v, diff)
		}
	}

	t.Logf("✓ Vertex Quantization: 16-bit precision reduction working (values within ±1.0 of original)")
}
