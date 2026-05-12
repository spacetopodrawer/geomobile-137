package photogrammetry

import (
	"testing"
)

// createMockKeyPoints creates mock keypoints for testing
func createMockKeyPoints(count int) []*KeyPoint {
	kps := make([]*KeyPoint, count)
	for i := 0; i < count; i++ {
		kps[i] = &KeyPoint{
			X:      float64(i * 10),
			Y:      float64(i * 10),
			Scale:  1.0,
			Response: 100.0 - float64(i),
		}
		// Set descriptor values
		for j := 0; j < 128; j++ {
			kps[i].Descriptor[j] = float64(i*10+j) / 255.0
		}
	}
	return kps
}

// TestFeatureMatcherInit tests initialization
func TestFeatureMatcherInit(t *testing.T) {
	fm := NewFeatureMatcher()

	if fm.DistanceThreshold != 100.0 {
		t.Errorf("Expected distance threshold 100.0, got %v", fm.DistanceThreshold)
	}

	if fm.RatioThreshold != 0.75 {
		t.Errorf("Expected ratio threshold 0.75, got %v", fm.RatioThreshold)
	}

	if fm.MinMatchCount != 8 {
		t.Errorf("Expected min matches 8, got %d", fm.MinMatchCount)
	}
}

// TestMatchFeaturesEmptyInput tests error handling for empty inputs
func TestMatchFeaturesEmptyInput(t *testing.T) {
	fm := NewFeatureMatcher()

	_, err := fm.MatchFeatures(nil, nil)
	if err == nil {
		t.Error("Should return error for nil inputs")
	}

	kps := createMockKeyPoints(5)
	_, err = fm.MatchFeatures(kps, nil)
	if err == nil {
		t.Error("Should return error for nil target keypoints")
	}

	_, err = fm.MatchFeatures(nil, kps)
	if err == nil {
		t.Error("Should return error for nil source keypoints")
	}
}

// TestMatchFeaturesValidInput tests basic matching
func TestMatchFeaturesValidInput(t *testing.T) {
	fm := NewFeatureMatcher()

	source := createMockKeyPoints(10)
	target := createMockKeyPoints(10)

	result, err := fm.MatchFeatures(source, target)
	if err != nil {
		t.Fatalf("MatchFeatures failed: %v", err)
	}

	if result == nil {
		t.Fatal("Should return non-nil result")
	}

	if result.MatchCount < 0 {
		t.Errorf("Negative match count: %d", result.MatchCount)
	}

	if result.MatchCount > len(source) {
		t.Errorf("Match count exceeds source keypoints: %d > %d", result.MatchCount, len(source))
	}
}

// TestMatchCountStats tests match statistics
func TestMatchCountStats(t *testing.T) {
	fm := NewFeatureMatcher()

	source := createMockKeyPoints(20)
	target := createMockKeyPoints(20)

	result, _ := fm.MatchFeatures(source, target)

	if result.MatchCount != len(result.Matches) {
		t.Errorf("MatchCount mismatch: %d != %d", result.MatchCount, len(result.Matches))
	}

	if result.InlierCount < 0 {
		t.Errorf("Negative inlier count: %d", result.InlierCount)
	}

	if result.OutlierCount < 0 {
		t.Errorf("Negative outlier count: %d", result.OutlierCount)
	}

	if result.AverageError < 0 {
		t.Errorf("Negative average error: %v", result.AverageError)
	}

	if result.SuccessRate < 0 || result.SuccessRate > 1 {
		t.Errorf("Success rate out of range [0, 1]: %v", result.SuccessRate)
	}
}

// TestMatchingConsistency tests repeated matching
func TestMatchingConsistency(t *testing.T) {
	fm := NewFeatureMatcher()

	source := createMockKeyPoints(15)
	target := createMockKeyPoints(15)

	result1, _ := fm.MatchFeatures(source, target)
	result2, _ := fm.MatchFeatures(source, target)

	if result1.MatchCount != result2.MatchCount {
		t.Errorf("Inconsistent match counts: %d != %d", result1.MatchCount, result2.MatchCount)
	}

	if result1.AverageError != result2.AverageError {
		t.Errorf("Inconsistent average errors: %v != %v", result1.AverageError, result2.AverageError)
	}
}

// TestGetLastMatches tests match result retrieval
func TestGetLastMatches(t *testing.T) {
	fm := NewFeatureMatcher()

	source := createMockKeyPoints(10)
	target := createMockKeyPoints(10)

	result, _ := fm.MatchFeatures(source, target)
	lastResult := fm.GetLastMatches()

	if lastResult == nil {
		t.Fatal("Should return non-nil last matches")
	}

	if lastResult.MatchCount != result.MatchCount {
		t.Errorf("Last matches mismatch: %d != %d", lastResult.MatchCount, result.MatchCount)
	}
}

// TestMatchWithDifferentSizes tests matching with different keypoint counts
func TestMatchWithDifferentSizes(t *testing.T) {
	fm := NewFeatureMatcher()

	source := createMockKeyPoints(5)
	target := createMockKeyPoints(20)

	result, err := fm.MatchFeatures(source, target)
	if err != nil {
		t.Fatalf("MatchFeatures failed: %v", err)
	}

	if result.MatchCount > len(source) {
		t.Errorf("Match count exceeds source: %d > %d", result.MatchCount, len(source))
	}

	// Success rate should be based on minimum keypoints
	expectedMax := float64(len(source)) / float64(len(source))
	if result.SuccessRate > expectedMax+0.01 {
		t.Errorf("Success rate exceeds expected: %v > %v", result.SuccessRate, expectedMax)
	}
}

// TestDescriptorDistance tests distance computation
func TestDescriptorDistance(t *testing.T) {
	fm := NewFeatureMatcher()

	// Identical descriptors should have zero distance
	var d1, d2 [128]float64
	dist := fm.descriptorDistance(d1, d2)
	if dist != 0.0 {
		t.Errorf("Identical descriptors should have zero distance, got %v", dist)
	}

	// Different descriptors should have positive distance
	for i := 0; i < 128; i++ {
		d1[i] = 0.5
		d2[i] = 0.0
	}
	dist = fm.descriptorDistance(d1, d2)
	if dist <= 0 {
		t.Errorf("Different descriptors should have positive distance, got %v", dist)
	}
}

// BenchmarkMatchFeatures benchmarks matching performance
func BenchmarkMatchFeatures(b *testing.B) {
	fm := NewFeatureMatcher()
	source := createMockKeyPoints(100)
	target := createMockKeyPoints(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fm.MatchFeatures(source, target)
	}
}
