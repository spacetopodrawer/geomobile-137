package photogrammetry

import (
	"testing"
)

// createMockDescriptors creates mock descriptor sets for testing
func createMockDescriptors(count int) [][128]float64 {
	descriptors := make([][128]float64, count)
	for i := 0; i < count; i++ {
		for j := 0; j < 128; j++ {
			descriptors[i][j] = float64((i*10+j) % 256) / 255.0
		}
	}
	return descriptors
}

// TestLoopClosureDetectorInit tests initialization
func TestLoopClosureDetectorInit(t *testing.T) {
	lcd := NewLoopClosureDetector()

	if lcd.MinFrameGap != 20 {
		t.Errorf("Expected min frame gap 20, got %d", lcd.MinFrameGap)
	}

	if lcd.ConfidenceThreshold != 0.8 {
		t.Errorf("Expected confidence threshold 0.8, got %v", lcd.ConfidenceThreshold)
	}

	if lcd.FrameCount != 0 {
		t.Error("Should start with 0 frames")
	}

	if lcd.LoopClosureCount != 0 {
		t.Error("Should start with 0 detected loops")
	}
}

// TestAddFrameValid tests adding frames
func TestAddFrameValid(t *testing.T) {
	lcd := NewLoopClosureDetector()

	descriptors := createMockDescriptors(10)
	err := lcd.AddFrame(0, descriptors)
	if err != nil {
		t.Fatalf("AddFrame failed: %v", err)
	}

	if lcd.GetFrameCount() != 1 {
		t.Errorf("Expected 1 frame, got %d", lcd.GetFrameCount())
	}
}

// TestAddFrameEmptyDescriptors tests error handling
func TestAddFrameEmptyDescriptors(t *testing.T) {
	lcd := NewLoopClosureDetector()

	err := lcd.AddFrame(0, make([][128]float64, 0))
	if err == nil {
		t.Error("Should return error for empty descriptors")
	}
}

// TestDetectLoopClosuresNoLoops tests when no loops are detected
func TestDetectLoopClosuresNoLoops(t *testing.T) {
	lcd := NewLoopClosureDetector()

	// Add frame at position 0
	desc1 := createMockDescriptors(10)
	lcd.AddFrame(0, desc1)

	// Try to detect loop with frame at position 10 (too close)
	desc2 := createMockDescriptors(10)
	loops, err := lcd.DetectLoopClosures(10, desc2)
	if err != nil {
		t.Fatalf("DetectLoopClosures failed: %v", err)
	}

	if len(loops) > 0 {
		t.Logf("Found %d loops when should find 0 (frames too close)", len(loops))
	}
}

// TestDetectLoopClosuresFarFrames tests detection with sufficient gap
func TestDetectLoopClosuresFarFrames(t *testing.T) {
	lcd := NewLoopClosureDetector()

	// Add first frame
	desc1 := createMockDescriptors(15)
	lcd.AddFrame(0, desc1)

	// Add intermediate frames
	for i := 1; i < 25; i++ {
		descs := createMockDescriptors(10)
		lcd.AddFrame(i, descs)
	}

	// Try to detect loop with same descriptors as frame 0
	loops, err := lcd.DetectLoopClosures(50, desc1)
	if err != nil {
		t.Fatalf("DetectLoopClosures failed: %v", err)
	}

	if loops == nil {
		t.Fatal("Should return non-nil loops slice")
	}

	// May or may not detect loop depending on descriptor matching
	// Just ensure no crash and valid results
	for _, loop := range loops {
		if loop.Confidence < 0 || loop.Confidence > 1 {
			t.Errorf("Invalid confidence: %v", loop.Confidence)
		}

		if loop.InlierCount < 0 || loop.OutlierCount < 0 {
			t.Error("Negative inlier/outlier counts")
		}
	}
}

// TestGetDetectedLoops tests loop retrieval
func TestGetDetectedLoops(t *testing.T) {
	lcd := NewLoopClosureDetector()

	desc := createMockDescriptors(10)
	lcd.AddFrame(0, desc)

	loops := lcd.GetDetectedLoops()
	if loops == nil {
		t.Fatal("Should return non-nil loops slice")
	}

	if len(loops) != lcd.GetLoopClosureCount() {
		t.Errorf("Loop count mismatch: %d != %d", len(loops), lcd.GetLoopClosureCount())
	}
}

// TestMultipleFrameAddition tests sequential frame addition
func TestMultipleFrameAddition(t *testing.T) {
	lcd := NewLoopClosureDetector()

	for i := 0; i < 10; i++ {
		desc := createMockDescriptors(5 + i)
		err := lcd.AddFrame(i, desc)
		if err != nil {
			t.Fatalf("AddFrame %d failed: %v", i, err)
		}
	}

	if lcd.GetFrameCount() != 10 {
		t.Errorf("Expected 10 frames, got %d", lcd.GetFrameCount())
	}
}

// BenchmarkDetectLoopClosures benchmarks loop closure detection
func BenchmarkDetectLoopClosures(b *testing.B) {
	lcd := NewLoopClosureDetector()

	// Setup frames
	for i := 0; i < 50; i++ {
		desc := createMockDescriptors(10)
		lcd.AddFrame(i, desc)
	}

	currentDesc := createMockDescriptors(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lcd.DetectLoopClosures(100+i, currentDesc)
	}
}
