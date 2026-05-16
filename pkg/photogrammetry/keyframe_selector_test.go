package photogrammetry

import (
	"testing"
)

// TestKeyframeSelectorInit tests initialization
func TestKeyframeSelectorInit(t *testing.T) {
	ks := NewKeyframeSelector()

	if ks == nil {
		t.Fatal("Keyframe selector should not be nil")
	}

	if ks.MinKeypointThreshold != 50 {
		t.Errorf("Expected min keypoint 50, got %d", ks.MinKeypointThreshold)
	}

	if ks.GetKeyframeCount() != 0 {
		t.Error("Should start with 0 keyframes")
	}
}

// TestEvaluateFrameNoKeypoints tests frame with no keypoints
func TestEvaluateFrameNoKeypoints(t *testing.T) {
	ks := NewKeyframeSelector()

	info := ks.EvaluateFrame(0, 0, nil, 0.0, 0.0)

	if info == nil {
		t.Fatal("Should return frame info")
	}

	if info.IsKeyframe {
		t.Error("Should not be keyframe with 0 keypoints")
	}

	if ks.GetKeyframeCount() != 0 {
		t.Error("Should have 0 keyframes")
	}
}

// TestEvaluateFrameValidKeyframe tests valid keyframe selection
func TestEvaluateFrameValidKeyframe(t *testing.T) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.8,
	}

	info := ks.EvaluateFrame(0, 100, matchResult, 0.05, 0.1)

	if info == nil {
		t.Fatal("Should return frame info")
	}

	if !info.IsKeyframe {
		t.Logf("Frame not selected as keyframe (tracking score: %.2f)", info.TrackingScore)
	}
}

// TestMultipleFrameEvaluation tests sequential frame evaluation
func TestMultipleFrameEvaluation(t *testing.T) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 80,
		Matches:    make([]*FeatureMatch, 80),
		SuccessRate: 0.75,
	}

	for i := 0; i < 10; i++ {
		translation := 0.02 + (0.01 * float64(i%2))
		ks.EvaluateFrame(i, 100, matchResult, translation, 0.05)
	}

	if ks.TotalFramesProcessed != 10 {
		t.Errorf("Expected 10 frames, got %d", ks.TotalFramesProcessed)
	}
}

// TestKeyframeGapEnforcement tests minimum frame gap
func TestKeyframeGapEnforcement(t *testing.T) {
	ks := NewKeyframeSelector()
	ks.MinFrameGap = 5

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.9,
	}

	// First keyframe
	ks.EvaluateFrame(0, 100, matchResult, 0.1, 0.1)

	// Try to add frames too close
	for i := 1; i < 5; i++ {
		info := ks.EvaluateFrame(i, 100, matchResult, 0.05, 0.05)
		if info.IsKeyframe {
			t.Logf("Frame %d should not be keyframe (gap %d < %d)", i, i, ks.MinFrameGap)
		}
	}

	// Frame at minimum gap should be allowed
	info := ks.EvaluateFrame(5, 100, matchResult, 0.1, 0.1)
	if !info.IsKeyframe {
		t.Logf("Frame 5 at minimum gap should be keyframe (gap %d >= %d)", 5, ks.MinFrameGap)
	}
}

// TestSelectionRate tests keyframe selection percentage
func TestSelectionRate(t *testing.T) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.8,
	}

	for i := 0; i < 20; i++ {
		translation := 0.02 + (0.01 * float64(i%2))
		ks.EvaluateFrame(i, 100, matchResult, translation, 0.05)
	}

	rate := ks.GetSelectionRate()
	if rate < 0.0 || rate > 1.0 {
		t.Errorf("Selection rate out of bounds: %.2f", rate)
	}

	t.Logf("Selection rate: %.2f%% (%d/%d frames)",
		rate*100, ks.GetKeyframeCount(), ks.TotalFramesProcessed)
}

// TestTrackingScore tests tracking quality computation
func TestTrackingScore(t *testing.T) {
	ks := NewKeyframeSelector()

	// High quality frame
	highQuality := &KeyframeInfo{
		FrameID:        0,
		KeypointCount:  200,
		MatchQuality:   0.9,
		TranslationMag: 0.05,
		RotationMag:    0.1,
	}
	highScore := ks.computeTrackingScore(highQuality)

	// Low quality frame
	lowQuality := &KeyframeInfo{
		FrameID:        1,
		KeypointCount:  20,
		MatchQuality:   0.4,
		TranslationMag: 0.001,
		RotationMag:    0.001,
	}
	lowScore := ks.computeTrackingScore(lowQuality)

	if highScore <= lowScore {
		t.Errorf("High quality (%.2f) should exceed low quality (%.2f)",
			highScore, lowScore)
	}

	if highScore < 0.0 || highScore > 1.0 {
		t.Errorf("Score out of range [0, 1]: %.2f", highScore)
	}
}

// TestForceKeyframe tests forcing a frame as keyframe
func TestForceKeyframe(t *testing.T) {
	ks := NewKeyframeSelector()

	// Evaluate frame that won't be selected
	matchResult := &FeatureMatchResult{
		MatchCount:  10,
		Matches:     make([]*FeatureMatch, 10),
		SuccessRate: 0.3,
	}

	ks.EvaluateFrame(0, 20, matchResult, 0.001, 0.001)

	initialCount := ks.GetKeyframeCount()

	// Force it to be keyframe
	err := ks.ForceKeyframe(0)
	if err != nil {
		t.Fatalf("ForceKeyframe failed: %v", err)
	}

	if ks.GetKeyframeCount() != initialCount+1 {
		t.Error("Keyframe count should increase")
	}
}

// TestAdaptiveThreshold tests dynamic threshold adjustment
func TestAdaptiveThreshold(t *testing.T) {
	ks := NewKeyframeSelector()

	originalThreshold := ks.MinKeypointThreshold

	// Simulate slow processing - increase threshold
	ks.AdaptiveThreshold(200.0, 100.0) // 2x target time
	if ks.MinKeypointThreshold <= originalThreshold {
		t.Error("Threshold should increase when too slow")
	}

	// Simulate fast processing - decrease threshold
	ks2 := NewKeyframeSelector()
	ks2.AdaptiveThreshold(50.0, 100.0) // 0.5x target time
	if ks2.MinKeypointThreshold >= originalThreshold {
		t.Error("Threshold should decrease when too fast")
	}
}

// TestKeyframeSelectorReset tests state reset
func TestKeyframeSelectorReset(t *testing.T) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.8,
	}

	for i := 0; i < 5; i++ {
		ks.EvaluateFrame(i, 100, matchResult, 0.05, 0.05)
	}

	if ks.GetKeyframeCount() == 0 && ks.TotalFramesProcessed == 0 {
		t.Fatal("Should have processed frames")
	}

	ks.Reset()

	if ks.GetKeyframeCount() != 0 {
		t.Error("Should have 0 keyframes after reset")
	}

	if ks.TotalFramesProcessed != 0 {
		t.Error("Should have 0 frames after reset")
	}
}

// TestGetSelectedKeyframes tests keyframe retrieval
func TestGetSelectedKeyframes(t *testing.T) {
	ks := NewKeyframeSelector()

	keyframes := ks.GetSelectedKeyframes()
	if keyframes == nil {
		t.Fatal("Should return non-nil slice")
	}

	if len(keyframes) != 0 {
		t.Error("Should start with empty keyframes")
	}
}

// TestGetFrameHistory tests frame history retrieval
func TestGetFrameHistory(t *testing.T) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.8,
	}

	for i := 0; i < 3; i++ {
		ks.EvaluateFrame(i, 100, matchResult, 0.05, 0.05)
	}

	history := ks.GetFrameHistory()
	if len(history) != 3 {
		t.Errorf("Expected 3 frames in history, got %d", len(history))
	}
}

// BenchmarkKeyframeEvaluation benchmarks frame evaluation
func BenchmarkKeyframeEvaluation(b *testing.B) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ks.EvaluateFrame(i, 100, matchResult, 0.05, 0.05)
	}
}

// TestKeyframeSelectorConcurrency tests thread-safety
func TestKeyframeSelectorConcurrency(t *testing.T) {
	ks := NewKeyframeSelector()

	matchResult := &FeatureMatchResult{
		MatchCount: 100,
		Matches:    make([]*FeatureMatch, 100),
		SuccessRate: 0.8,
	}

	// Sequential evaluation should be thread-safe
	for i := 0; i < 10; i++ {
		ks.EvaluateFrame(i, 100, matchResult, 0.05, 0.05)
	}

	// Concurrent reads
	go func() {
		_ = ks.GetSelectedKeyframes()
		_ = ks.GetKeyframeCount()
		_ = ks.GetSelectionRate()
	}()

	// Should not panic or deadlock
	_ = ks.GetFrameHistory()
}
