package photogrammetry

import (
	"testing"
)

// createMockImage creates mock image data for testing
func createMockImage(width, height int) [][3]uint8 {
	image := make([][3]uint8, width*height)
	for i := range image {
		image[i] = [3]uint8{uint8((i * 7) % 256), uint8((i * 13) % 256), uint8((i * 19) % 256)}
	}
	return image
}

// TestDenseReconstructionInit tests initialization
func TestDenseReconstructionInit(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	if dr.MinDepth != 0.1 {
		t.Errorf("Expected min depth 0.1, got %v", dr.MinDepth)
	}

	if dr.MaxDepth != 10.0 {
		t.Errorf("Expected max depth 10.0, got %v", dr.MaxDepth)
	}

	if dr.WindowSize != 5 {
		t.Errorf("Expected window size 5, got %d", dr.WindowSize)
	}

	if dr.PointCloudSize != 0 {
		t.Error("Should start with 0 points")
	}
}

// TestEstimateDepthMapInvalidInput tests error handling
func TestEstimateDepthMapInvalidInput(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	pose := createMockCameraFrame(0, false)
	_, err := dr.EstimateDepthMap(0, nil, nil, pose, pose, 0, 0)
	if err == nil {
		t.Error("Should return error for zero dimensions")
	}
}

// TestEstimateDepthMapValidInput tests depth map estimation
func TestEstimateDepthMapValidInput(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	width, height := 64, 64
	refImage := createMockImage(width, height)
	tgtImage := createMockImage(width, height)

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	depthMap, err := dr.EstimateDepthMap(0, refImage, tgtImage, pose1, pose2, width, height)
	if err != nil {
		t.Fatalf("EstimateDepthMap failed: %v", err)
	}

	if depthMap == nil {
		t.Fatal("Should return non-nil depth map")
	}

	if depthMap.Width != width || depthMap.Height != height {
		t.Errorf("Depth map size mismatch: %dx%d != %dx%d", 
			depthMap.Width, depthMap.Height, width, height)
	}

	if len(depthMap.Depth) != width*height {
		t.Errorf("Depth array size mismatch: %d != %d", len(depthMap.Depth), width*height)
	}
}

// TestDepthMapProperties tests depth map attribute validity
func TestDepthMapProperties(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	width, height := 32, 32
	refImage := createMockImage(width, height)
	tgtImage := createMockImage(width, height)

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	depthMap, _ := dr.EstimateDepthMap(0, refImage, tgtImage, pose1, pose2, width, height)

	for i, depth := range depthMap.Depth {
		// Depth should be within valid range
		if depth < dr.MinDepth-0.1 || depth > dr.MaxDepth+0.1 {
			if depthMap.Validity[i] {
				t.Logf("Valid depth out of range at index %d: %v", i, depth)
			}
		}

		// Confidence should be in [0, 1]
		conf := depthMap.Confidence[i]
		if conf < 0 || conf > 1 {
			t.Errorf("Confidence out of range at index %d: %v", i, conf)
		}
	}
}

// TestConvertDepthMapToPointCloud tests point cloud conversion
func TestConvertDepthMapToPointCloud(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	// Create simple depth map
	depthMap := &DepthMap{
		Width:  32,
		Height: 32,
		Depth:  make([]float64, 32*32),
		Confidence: make([]float64, 32*32),
		Validity: make([]bool, 32*32),
		FrameID: 0,
	}

	// Set some valid depths
	for i := 0; i < 100; i++ {
		depthMap.Depth[i] = 5.0
		depthMap.Confidence[i] = 0.8
		depthMap.Validity[i] = true
	}

	pose := createMockCameraFrame(0, false)
	points := dr.ConvertDepthMapToPointCloud(depthMap, pose)

	if points == nil {
		t.Fatal("Should return non-nil points slice")
	}

	// Should have generated some points
	if len(points) > 0 {
		for _, pt := range points {
			if pt.Depth <= 0 {
				t.Error("Point should have positive depth")
			}

			if pt.Uncertainty < 0 || pt.Uncertainty > 1 {
				t.Errorf("Point uncertainty out of range: %v", pt.Uncertainty)
			}
		}
	}
}

// TestMultipleDepthMaps tests sequential depth map estimation
func TestMultipleDepthMaps(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	width, height := 32, 32
	img1 := createMockImage(width, height)

	for i := 0; i < 5; i++ {
		img2 := createMockImage(width, height)
		pose1 := createMockCameraFrame(i, false)
		pose2 := createMockCameraFrame(i+1, true)

		_, err := dr.EstimateDepthMap(i, img1, img2, pose1, pose2, width, height)
		if err != nil {
			t.Fatalf("EstimateDepthMap %d failed: %v", i, err)
		}
	}

	if len(dr.GetDepthMaps()) != 5 {
		t.Errorf("Expected 5 depth maps, got %d", len(dr.GetDepthMaps()))
	}
}

// TestGetPointCloud tests point cloud retrieval
func TestGetPointCloud(t *testing.T) {
	dr := NewDenseReconstruction(0.1, 10.0)

	if dr.GetPointCloudSize() != 0 {
		t.Error("Should start with empty point cloud")
	}

	depthMap := &DepthMap{
		Width:  10,
		Height: 10,
		Depth:  make([]float64, 100),
		Confidence: make([]float64, 100),
		Validity: make([]bool, 100),
	}

	for i := 0; i < 50; i++ {
		depthMap.Depth[i] = 3.0
		depthMap.Confidence[i] = 0.9
		depthMap.Validity[i] = true
	}

	pose := createMockCameraFrame(0, false)
	dr.ConvertDepthMapToPointCloud(depthMap, pose)

	pointCloud := dr.GetPointCloud()
	if pointCloud == nil {
		t.Fatal("Should return non-nil point cloud")
	}

	if len(pointCloud) != dr.GetPointCloudSize() {
		t.Errorf("Point cloud size mismatch: %d != %d", len(pointCloud), dr.GetPointCloudSize())
	}
}

// BenchmarkEstimateDepthMap benchmarks depth map estimation
func BenchmarkEstimateDepthMap(b *testing.B) {
	dr := NewDenseReconstruction(0.1, 10.0)

	width, height := 64, 64
	refImage := createMockImage(width, height)
	tgtImage := createMockImage(width, height)

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dr.EstimateDepthMap(i, refImage, tgtImage, pose1, pose2, width, height)
	}
}
