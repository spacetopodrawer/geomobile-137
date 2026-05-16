package photogrammetry

import (
	"testing"
	"time"
)

// TestRealTimeProcessorInit tests initialization
func TestRealTimeProcessorInit(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	if rtp == nil {
		t.Fatal("Processor should not be nil")
	}

	if rtp.TargetFPS != 10 {
		t.Errorf("Expected target FPS 10, got %d", rtp.TargetFPS)
	}

	if rtp.TargetFrameTimeMs != 100.0 {
		t.Errorf("Expected frame time 100ms, got %.2f", rtp.TargetFrameTimeMs)
	}
}

// TestProcessFrameRealTimeValidInput tests frame processing
func TestProcessFrameRealTimeValidInput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	metrics, err := rtp.ProcessFrameRealTime(0, image, 100, 100, frame)

	if err != nil {
		t.Logf("ProcessFrameRealTime error (non-critical): %v", err)
	}

	if metrics == nil {
		t.Fatal("Should return metrics")
	}

	if metrics.FrameID != 0 {
		t.Errorf("Expected frame ID 0, got %d", metrics.FrameID)
	}
}

// TestProcessFrameInvalidInput tests error handling
func TestProcessFrameRealTimeInvalidInput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	frame := createTestFrame(0, false)

	// Test nil image
	_, err := rtp.ProcessFrameRealTime(0, nil, 100, 100, frame)
	if err == nil {
		t.Error("Should return error for nil image")
	}

	// Test nil pose
	image := createSfMTestImage(100, 100)
	_, err = rtp.ProcessFrameRealTime(0, image, 100, 100, nil)
	if err == nil {
		t.Error("Should return error for nil pose")
	}
}

// TestMultipleFrameProcessing tests sequential frame processing
func TestMultipleFrameProcessing(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	for i := 0; i < 5; i++ {
		image := createSfMTestImage(80, 80)
		frame := createTestFrame(i, i > 0)

		metrics, err := rtp.ProcessFrameRealTime(i, image, 80, 80, frame)
		if err != nil {
			t.Logf("Frame %d error (non-critical): %v", i, err)
		}

		if metrics == nil {
			t.Fatalf("Frame %d should return metrics", i)
		}
	}

	if rtp.ProcessedFrames+rtp.SkippedFrames != 5 {
		t.Errorf("Expected 5 total frames, got %d",
			rtp.ProcessedFrames+rtp.SkippedFrames)
	}
}

// TestGetThroughput tests FPS calculation
func TestGetThroughput(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	for i := 0; i < 3; i++ {
		rtp.ProcessFrameRealTime(i, image, 100, 100, frame)
		time.Sleep(10 * time.Millisecond) // Simulate processing time
	}

	fps := rtp.GetThroughput()
	if fps < 0 {
		t.Errorf("FPS should be non-negative: %.2f", fps)
	}

	t.Logf("Measured throughput: %.2f FPS", fps)
}

// TestGetMetricsReport tests metrics report generation
func TestGetMetricsReport(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	rtp.ProcessFrameRealTime(0, image, 100, 100, frame)

	report := rtp.GetMetricsReport()
	if report == nil {
		t.Fatal("Should return metrics report")
	}

	if _, ok := report["target_fps"]; !ok {
		t.Error("Report should contain target_fps")
	}

	if _, ok := report["average_frame_time_ms"]; !ok {
		t.Error("Report should contain average_frame_time_ms")
	}
}

// TestSetTargetFPS tests FPS configuration
func TestSetTargetFPS(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	// Valid FPS
	err := rtp.SetTargetFPS(30)
	if err != nil {
		t.Fatalf("SetTargetFPS(30) failed: %v", err)
	}

	if rtp.TargetFPS != 30 {
		t.Errorf("Expected FPS 30, got %d", rtp.TargetFPS)
	}

	expectedTime := 1000.0 / 30.0
	if rtp.TargetFrameTimeMs != expectedTime {
		t.Errorf("Expected frame time %.2f, got %.2f", expectedTime, rtp.TargetFrameTimeMs)
	}

	// Invalid FPS
	err = rtp.SetTargetFPS(0)
	if err == nil {
		t.Error("SetTargetFPS(0) should return error")
	}

	err = rtp.SetTargetFPS(100)
	if err == nil {
		t.Error("SetTargetFPS(100) should return error")
	}
}

// TestGetMetricsHistory tests metrics history retrieval
func TestGetMetricsHistory(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	history := rtp.GetMetricsHistory()
	if history == nil {
		t.Fatal("Should return non-nil history")
	}

	if len(history) != 0 {
		t.Error("Should start with empty history")
	}

	// Add some frames
	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	for i := 0; i < 3; i++ {
		rtp.ProcessFrameRealTime(i, image, 100, 100, frame)
	}

	history = rtp.GetMetricsHistory()
	if len(history) < 1 {
		t.Error("History should contain processed frames")
	}
}

// TestResetMetrics tests metrics reset
func TestResetMetrics(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	image := createSfMTestImage(100, 100)
	frame := createTestFrame(0, false)

	rtp.ProcessFrameRealTime(0, image, 100, 100, frame)

	if rtp.ProcessedFrames == 0 && rtp.SkippedFrames == 0 {
		t.Fatal("Should have processed or skipped frames")
	}

	rtp.ResetMetrics()

	if rtp.ProcessedFrames != 0 {
		t.Error("Should have 0 processed frames after reset")
	}

	if rtp.SkippedFrames != 0 {
		t.Error("Should have 0 skipped frames after reset")
	}

	if len(rtp.GetMetricsHistory()) != 0 {
		t.Error("History should be empty after reset")
	}
}

// TestSetDynamicFrameSkip tests dynamic frame skipping toggle
func TestSetDynamicFrameSkip(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	if !rtp.DynamicFrameSkip {
		t.Error("Should start with dynamic frame skip enabled")
	}

	rtp.SetDynamicFrameSkip(false)
	if rtp.DynamicFrameSkip {
		t.Error("Dynamic frame skip should be disabled")
	}

	rtp.SetDynamicFrameSkip(true)
	if !rtp.DynamicFrameSkip {
		t.Error("Dynamic frame skip should be enabled")
	}
}

// TestGetKeyframeSelector tests keyframe selector access
func TestGetKeyframeSelector(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	ks := rtp.GetKeyframeSelector()
	if ks == nil {
		t.Fatal("Should return non-nil keyframe selector")
	}

	if ks.GetKeyframeCount() != 0 {
		t.Error("Should start with 0 keyframes")
	}
}

// TestGetPipeline tests pipeline access
func TestGetPipeline(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	retrieved := rtp.GetPipeline()
	if retrieved == nil {
		t.Fatal("Should return non-nil pipeline")
	}

	if retrieved != pipeline {
		t.Error("Should return same pipeline instance")
	}
}

// TestIntegrationRealTime tests complete real-time processing flow
func TestIntegrationRealTime(t *testing.T) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	// Set target to 10 Hz
	rtp.SetTargetFPS(10)

	numFrames := 10
	for i := 0; i < numFrames; i++ {
		image := createSfMTestImage(100, 100)
		frame := createTestFrame(i, i > 0)

		metrics, err := rtp.ProcessFrameRealTime(i, image, 100, 100, frame)
		if err != nil {
			t.Logf("Frame %d error (non-critical): %v", i, err)
		}

		if metrics == nil {
			t.Fatalf("Frame %d should return metrics", i)
		}
	}

	report := rtp.GetMetricsReport()
	t.Logf("Real-time processing report: %v", report)

	if rtp.ProcessedFrames+rtp.SkippedFrames != numFrames {
		t.Errorf("Expected %d frames, got %d",
			numFrames, rtp.ProcessedFrames+rtp.SkippedFrames)
	}
}

// BenchmarkRealTimeProcessing benchmarks real-time processing
func BenchmarkRealTimeProcessing(b *testing.B) {
	pipeline := NewSfMPipeline(500.0, 320.0, 240.0)
	rtp := NewRealTimeProcessor(pipeline)

	image := createSfMTestImage(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := createTestFrame(i, i > 0)
		rtp.ProcessFrameRealTime(i, image, 100, 100, frame)
	}
}
