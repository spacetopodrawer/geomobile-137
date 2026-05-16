package photogrammetry

import (
	"testing"
)

// TestArcadeProcessorInit tests initialization
func TestArcadeProcessorInit(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(640, 480, pipeline, rtp)

	if ap == nil {
		t.Fatal("Arcade processor should not be nil")
	}

	if ap.TargetWidth != 640 {
		t.Errorf("Expected width 640, got %d", ap.TargetWidth)
	}

	if ap.TargetHeight != 480 {
		t.Errorf("Expected height 480, got %d", ap.TargetHeight)
	}

	if ap.ProcessedFrames != 0 {
		t.Error("Should start with 0 frames")
	}
}

// TestClassifyArcadeScene tests scene type detection
func TestClassifyArcadeScene(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(640, 480, pipeline, rtp)

	// Create high-variance image (2D sprite)
	image2D := make([][3]uint8, 640*480)
	for i := 0; i < len(image2D); i++ {
		if i%2 == 0 {
			image2D[i] = [3]uint8{255, 0, 0} // Red
		} else {
			image2D[i] = [3]uint8{0, 255, 0} // Green
		}
	}

	sceneType := ap.classifyArcadeScene(image2D)
	if sceneType == "Unknown" {
		t.Logf("Scene type: %s (high variance image)", sceneType)
	}

	// Create low-variance image (3D or UI)
	image3D := make([][3]uint8, 640*480)
	for i := 0; i < len(image3D); i++ {
		val := uint8((i / 100) % 256)
		image3D[i] = [3]uint8{val, val, val}
	}

	sceneType = ap.classifyArcadeScene(image3D)
	t.Logf("Scene type: %s (low variance image)", sceneType)
}

// TestDetectTextureRegions tests region detection
func TestDetectTextureRegions(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(400, 300, pipeline, rtp)

	image := make([][3]uint8, 400*300)
	for i := 0; i < len(image); i++ {
		image[i] = [3]uint8{128, 128, 128}
	}

	regions := ap.detectTextureRegions(image)

	if regions == nil {
		t.Fatal("Should return non-nil regions")
	}

	t.Logf("Detected %d regions", len(regions))
}

// TestProcessArcadeFrameValidInput tests frame processing
func TestProcessArcadeFrameValidInput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)

	info, err := ap.ProcessArcadeFrame(0, image, pose)

	if err != nil {
		t.Logf("ProcessArcadeFrame error (non-critical): %v", err)
	}

	if info == nil {
		t.Fatal("Should return frame info")
	}

	if info.FrameID != 0 {
		t.Errorf("Expected frame ID 0, got %d", info.FrameID)
	}

	if info.SceneType == "Unknown" {
		t.Logf("Scene type: %s", info.SceneType)
	}
}

// TestProcessArcadeFrameInvalidInput tests error handling
func TestProcessArcadeFrameInvalidInput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	pose := createTestFrame(0, false)

	// Test nil image
	_, err := ap.ProcessArcadeFrame(0, nil, pose)
	if err == nil {
		t.Error("Should return error for nil image")
	}

	// Test nil pose
	image := createSfMTestImage(100, 100)
	_, err = ap.ProcessArcadeFrame(0, image, nil)
	if err == nil {
		t.Error("Should return error for nil pose")
	}
}

// TestMultipleArcadeFrames tests sequential frame processing
func TestMultipleArcadeFrames(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(80, 80, pipeline, rtp)

	for i := 0; i < 5; i++ {
		image := createSfMTestImage(80, 80)
		pose := createTestFrame(i, i > 0)

		info, err := ap.ProcessArcadeFrame(i, image, pose)
		if err != nil {
			t.Logf("Frame %d error (non-critical): %v", i, err)
		}

		if info == nil {
			t.Fatalf("Frame %d should return info", i)
		}
	}

	if ap.ProcessedFrames != 5 {
		t.Errorf("Expected 5 frames, got %d", ap.ProcessedFrames)
	}
}

// TestGetSuccessRate tests success rate calculation
func TestGetSuccessRate(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)

	ap.ProcessArcadeFrame(0, image, pose)

	rate := ap.GetSuccessRate()
	if rate < 0.0 || rate > 1.0 {
		t.Errorf("Success rate out of range: %.2f", rate)
	}

	t.Logf("Success rate: %.2f%%", rate*100)
}

// TestGetFrameInfo tests frame info retrieval
func TestGetFrameInfo(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)

	ap.ProcessArcadeFrame(0, image, pose)

	info := ap.GetFrameInfo(0)
	if info == nil {
		t.Error("Should return frame info for existing frame")
	}

	info = ap.GetFrameInfo(999)
	if info != nil {
		t.Error("Should return nil for non-existing frame")
	}
}

// TestGetSceneStatistics tests statistics report
func TestGetSceneStatistics(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)

	ap.ProcessArcadeFrame(0, image, pose)

	stats := ap.GetSceneStatistics()
	if stats == nil {
		t.Fatal("Should return statistics")
	}

	if _, ok := stats["total_frames"]; !ok {
		t.Error("Statistics should contain total_frames")
	}

	if _, ok := stats["success_rate"]; !ok {
		t.Error("Statistics should contain success_rate")
	}
}

// TestFinalizeReconstruction tests scene finalization
func TestFinalizeReconstruction(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)

	ap.ProcessArcadeFrame(0, image, pose)

	scene := ap.FinalizeReconstruction()

	if scene == nil {
		t.Fatal("Should return reconstructed scene")
	}

	if scene.TotalFrames != 1 {
		t.Errorf("Expected 1 frame, got %d", scene.TotalFrames)
	}
}

// TestGetReconstructedScene tests scene retrieval
func TestGetReconstructedScene(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	// Initially nil
	scene := ap.GetReconstructedScene()
	if scene != nil {
		t.Error("Should return nil before finalization")
	}

	// After finalization
	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)
	ap.ProcessArcadeFrame(0, image, pose)
	ap.FinalizeReconstruction()

	scene = ap.GetReconstructedScene()
	if scene == nil {
		t.Error("Should return scene after finalization")
	}
}

// TestReset tests state reset
func TestArcadeReset(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)
	pose := createTestFrame(0, false)

	ap.ProcessArcadeFrame(0, image, pose)

	if ap.ProcessedFrames != 1 {
		t.Error("Should have processed 1 frame")
	}

	ap.Reset()

	if ap.ProcessedFrames != 0 {
		t.Error("Should have 0 frames after reset")
	}

	if ap.AverageError != 0.0 {
		t.Error("Should have 0 error after reset")
	}
}

// TestComputeColorVariance tests color variance computation
func TestComputeColorVariance(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	// High variance image
	imageHigh := make([][3]uint8, 100)
	for i := 0; i < len(imageHigh); i++ {
		if i%2 == 0 {
			imageHigh[i] = [3]uint8{255, 0, 0}
		} else {
			imageHigh[i] = [3]uint8{0, 255, 0}
		}
	}

	varianceHigh := ap.computeColorVariance(imageHigh)

	// Low variance image
	imageLow := make([][3]uint8, 100)
	for i := 0; i < len(imageLow); i++ {
		imageLow[i] = [3]uint8{128, 128, 128}
	}

	varianceLow := ap.computeColorVariance(imageLow)

	if varianceHigh <= varianceLow {
		t.Errorf("High variance (%.3f) should exceed low variance (%.3f)",
			varianceHigh, varianceLow)
	}

	t.Logf("High variance: %.3f, Low variance: %.3f", varianceHigh, varianceLow)
}

// TestIntegrationArcadeProcessor tests complete arcade processing flow
func TestIntegrationArcadeProcessor(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	// Process multiple frames
	numFrames := 5
	for i := 0; i < numFrames; i++ {
		image := createSfMTestImage(100, 100)
		pose := createTestFrame(i, i > 0)

		info, err := ap.ProcessArcadeFrame(i, image, pose)
		if err != nil {
			t.Logf("Frame %d error (non-critical): %v", i, err)
		}

		if info == nil {
			t.Fatalf("Frame %d should return info", i)
		}
	}

	// Get statistics
	stats := ap.GetSceneStatistics()
	t.Logf("Arcade processing statistics: %v", stats)

	// Finalize
	scene := ap.FinalizeReconstruction()
	if scene == nil {
		t.Fatal("Should have finalized scene")
	}

	if scene.TotalFrames != numFrames {
		t.Errorf("Expected %d frames, got %d", numFrames, scene.TotalFrames)
	}
}

// BenchmarkArcadeProcessing benchmarks arcade frame processing
func BenchmarkArcadeProcessing(b *testing.B) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)
	ap := NewArcadeProcessor(100, 100, pipeline, rtp)

	image := createSfMTestImage(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pose := createTestFrame(i, i > 0)
		ap.ProcessArcadeFrame(i, image, pose)
	}
}
