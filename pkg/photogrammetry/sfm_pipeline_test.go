package photogrammetry

import (
	"testing"
)

// createMockFrame creates a mock camera frame with test data
func createTestFrame(id int, withTranslation bool) *CameraFrame {
	frame := &CameraFrame{
		ImageID:    id,
		IsKeyFrame: true,
	}

	// Identity rotation
	frame.RotationMatrix[0][0] = 1.0
	frame.RotationMatrix[1][1] = 1.0
	frame.RotationMatrix[2][2] = 1.0

	// Optional translation
	if withTranslation {
		frame.Translation[0] = 0.1 * float64(id)
	}

	// Camera intrinsics
	frame.CameraMatrix[0][0] = 500.0
	frame.CameraMatrix[0][2] = 320.0
	frame.CameraMatrix[1][1] = 500.0
	frame.CameraMatrix[1][2] = 240.0
	frame.CameraMatrix[2][2] = 1.0

	return frame
}

// createSfMTestImage creates a simple test image for SfM pipeline
func createSfMTestImage(width, height int) [][3]uint8 {
	image := make([][3]uint8, width*height)
	for i := 0; i < width*height; i++ {
		val := uint8((i * 7) % 256)
		image[i] = [3]uint8{val, val, val}
	}
	return image
}

// TestSfMPipelineInit tests pipeline initialization
func TestSfMPipelineInit(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	if pipeline == nil {
		t.Fatal("Pipeline should not be nil")
	}

	if pipeline.Config == nil {
		t.Fatal("Config should not be nil")
	}

	if pipeline.ProcessedFrames != 0 {
		t.Error("Should start with 0 frames")
	}

	if len(pipeline.ReconstructedPoints) != 0 {
		t.Error("Should start with 0 points")
	}
}

// TestSfMPipelineCustomConfig tests custom configuration
func TestSfMPipelineCustomConfig(t *testing.T) {
	config := DefaultConfig()
	config.TargetFPS = 15
	config.LoopMinFrameGap = 30

	pipeline := NewSfMPipelineWithConfig(500.0, 320.0, 240.0, config)

	if pipeline.Config.TargetFPS != 15 {
		t.Errorf("Expected FPS 15, got %d", pipeline.Config.TargetFPS)
	}

	if pipeline.Config.LoopMinFrameGap != 30 {
		t.Errorf("Expected loop gap 30, got %d", pipeline.Config.LoopMinFrameGap)
	}
}

// TestProcessFrameInvalidInput tests error handling
func TestProcessFrameInvalidInput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	frame := createTestFrame(0, false)

	// Test nil image
	err := pipeline.ProcessFrame(0, nil, 640, 480, frame)
	if err == nil {
		t.Error("Should return error for nil image")
	}

	// Test nil pose
	image := createSfMTestImage(640, 480)
	err = pipeline.ProcessFrame(0, image, 640, 480, nil)
	if err == nil {
		t.Error("Should return error for nil pose")
	}
}

// TestProcessFrameValidInput tests frame processing
func TestProcessFrameValidInput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	err := pipeline.ProcessFrame(0, image, 100, 100, frame)
	if err != nil {
		t.Logf("ProcessFrame returned error (may be due to feature detection): %v", err)
	}

	if pipeline.ProcessedFrames != 1 {
		t.Errorf("Expected 1 processed frame, got %d", pipeline.ProcessedFrames)
	}
}

// TestProcessMultipleFrames tests sequential frame processing
func TestProcessMultipleFrames(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	width, height := 100, 100

	for i := 0; i < 5; i++ {
		image := createSfMTestImage(width, height)
		frame := createTestFrame(i, i > 0)

		err := pipeline.ProcessFrame(i, image, width, height, frame)
		if err != nil {
			t.Logf("Frame %d error (non-critical): %v", i, err)
		}
	}

	if pipeline.ProcessedFrames != 5 {
		t.Errorf("Expected 5 frames, got %d", pipeline.ProcessedFrames)
	}
}

// TestLoopClosureDetectionInPipeline tests loop closure detection
func TestLoopClosureDetectionInPipeline(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	width, height := 80, 80

	// Add initial frame
	image1 := createSfMTestImage(width, height)
	frame1 := createTestFrame(0, false)
	pipeline.ProcessFrame(0, image1, width, height, frame1)

	// Add intermediate frames
	for i := 1; i < 25; i++ {
		image := createSfMTestImage(width, height)
		frame := createTestFrame(i, true)
		pipeline.ProcessFrame(i, image, width, height, frame)
	}

	// Add frame that might close a loop
	image2 := createSfMTestImage(width, height)
	frame2 := createTestFrame(50, true)
	pipeline.ProcessFrame(50, image2, width, height, frame2)

	loops := pipeline.GetLoops()
	if loops == nil {
		t.Fatal("Should return non-nil loops")
	}

	t.Logf("Detected %d loop closures", len(loops))
}

// TestGetMetrics tests metrics collection
func TestGetMetrics(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	pipeline.ProcessFrame(0, image, 100, 100, frame)

	metrics := pipeline.GetMetrics()
	if metrics == nil {
		t.Fatal("Should return non-nil metrics")
	}

	processedFrames, ok := metrics["processed_frames"]
	if !ok {
		t.Error("Metrics should contain processed_frames")
	}

	if processedFrames != 1 {
		t.Errorf("Expected 1 processed frame in metrics, got %v", processedFrames)
	}

	fps, ok := metrics["fps"]
	if !ok {
		t.Error("Metrics should contain fps")
	}

	if fps == "" {
		t.Error("FPS should not be empty")
	}
}

// TestGetReconstruction tests reconstruction retrieval
func TestGetReconstruction(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	points := pipeline.GetReconstruction()
	if points == nil {
		t.Fatal("Should return non-nil points slice")
	}

	if len(points) != 0 {
		t.Error("Should start with empty reconstruction")
	}
}

// TestReset tests pipeline reset
func TestReset(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	pipeline.ProcessFrame(0, image, 100, 100, frame)

	if pipeline.ProcessedFrames != 1 {
		t.Error("Should have processed 1 frame")
	}

	// Reset pipeline
	pipeline.Reset()

	if pipeline.ProcessedFrames != 0 {
		t.Error("Should have 0 frames after reset")
	}

	if len(pipeline.ImageSequence) != 0 {
		t.Error("Should have empty image sequence after reset")
	}

	if len(pipeline.ReconstructedPoints) != 0 {
		t.Error("Should have empty reconstruction after reset")
	}
}

// TestFrameCount tests frame counting
func TestFrameCount(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	for i := 0; i < 3; i++ {
		image := createSfMTestImage(80, 80)
		frame := createTestFrame(i, false)
		pipeline.ProcessFrame(i, image, 80, 80, frame)
	}

	count := pipeline.GetFrameCount()
	if count != 3 {
		t.Errorf("Expected frame count 3, got %d", count)
	}
}

// TestConcurrentProcessing tests thread-safe operation
func TestConcurrentProcessing(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	// Sequential processing is thread-safe
	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	// Call GetMetrics while processing
	go pipeline.ProcessFrame(0, image, 100, 100, frame)

	// Should not panic or deadlock
	metrics := pipeline.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should be available during processing")
	}
}

// TestIntegrationPipeline tests complete pipeline flow
func TestIntegrationPipeline(t *testing.T) {
	config := DefaultConfig()
	pipeline := NewSfMPipelineWithConfig(500.0, 320.0, 240.0, config)

	width, height := 96, 96

	// Process sequence of frames
	numFrames := 10
	for i := 0; i < numFrames; i++ {
		image := createSfMTestImage(width, height)
		frame := createTestFrame(i, i > 0)

		err := pipeline.ProcessFrame(i, image, width, height, frame)
		if err != nil {
			t.Logf("Frame %d processing (may skip features): %v", i, err)
		}
	}

	// Verify final state
	if pipeline.GetFrameCount() != numFrames {
		t.Errorf("Expected %d frames, got %d", numFrames, pipeline.GetFrameCount())
	}

	metrics := pipeline.GetMetrics()
	processedFrames := metrics["processed_frames"].(int)
	if processedFrames != numFrames {
		t.Errorf("Metrics show %d frames, expected %d", processedFrames, numFrames)
	}

	// Verify reconstruction exists (may be empty if no features detected)
	reconstruction := pipeline.GetReconstruction()
	if reconstruction == nil {
		t.Fatal("Reconstruction should not be nil")
	}

	t.Logf("Pipeline integration test: %d frames processed, %d points reconstructed",
		pipeline.GetFrameCount(), len(reconstruction))
}

// BenchmarkSfMPipeline benchmarks pipeline performance
func BenchmarkSfMPipeline(b *testing.B) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)

	width, height := 100, 100
	image := createSfMTestImage(width, height)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := createTestFrame(i, i > 0)
		pipeline.ProcessFrame(i, image, width, height, frame)
	}

	b.ReportMetric(float64(pipeline.ProcessedFrames), "frames")
	b.ReportMetric(float64(len(pipeline.ReconstructedPoints)), "points")
}
