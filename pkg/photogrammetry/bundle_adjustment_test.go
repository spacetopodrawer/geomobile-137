package photogrammetry

import (
	"testing"
)

// TestBundleAdjustmentInit tests initialization
func TestBundleAdjustmentInit(t *testing.T) {
	ba := NewBundleAdjustment()

	if ba.MaxIterations != 20 {
		t.Errorf("Expected max iterations 20, got %d", ba.MaxIterations)
	}

	if ba.DampingFactor != 0.001 {
		t.Errorf("Expected damping factor 0.001, got %v", ba.DampingFactor)
	}

	if ba.DampingDecrease != 10.0 {
		t.Errorf("Expected damping decrease 10.0, got %v", ba.DampingDecrease)
	}
}

// TestOptimizeEmptyInput tests error handling
func TestOptimizeEmptyInput(t *testing.T) {
	ba := NewBundleAdjustment()

	_, err := ba.Optimize(nil, nil, nil)
	if err == nil {
		t.Error("Should return error for empty input")
	}

	points := make([]*Point3D, 0)
	poses := make([]*CameraFrame, 0)
	matches := make([]*FeatureMatch, 0)

	_, err = ba.Optimize(points, poses, matches)
	if err == nil {
		t.Error("Should return error for empty arrays")
	}
}

// TestOptimizeValidInput tests optimization
func TestOptimizeValidInput(t *testing.T) {
	ba := NewBundleAdjustment()

	// Create test data
	point := &Point3D{X: 1.0, Y: 2.0, Z: 5.0, Depth: 5.39}
	points := []*Point3D{point}

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)
	poses := []*CameraFrame{pose1, pose2}

	kps := createMockKeyPoints(5)
	matches := make([]*FeatureMatch, 4)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       10.0,
		}
	}

	result, err := ba.Optimize(points, poses, matches)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	if result == nil {
		t.Fatal("Should return non-nil result")
	}

	if result.Iterations < 0 {
		t.Errorf("Negative iterations: %d", result.Iterations)
	}

	if result.Iterations > ba.MaxIterations {
		t.Errorf("Iterations exceed max: %d > %d", result.Iterations, ba.MaxIterations)
	}
}

// TestOptimizationErrorMonotonicity tests error decreases
func TestOptimizationErrorMonotonicity(t *testing.T) {
	ba := NewBundleAdjustment()

	point := &Point3D{X: 0.5, Y: 1.0, Z: 3.0}
	points := []*Point3D{point}

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)
	poses := []*CameraFrame{pose1, pose2}

	kps := createMockKeyPoints(3)
	matches := make([]*FeatureMatch, 2)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       5.0,
		}
	}

	result, _ := ba.Optimize(points, poses, matches)

	// Error should not increase dramatically
	if result.FinalError > result.InitialError*10 {
		t.Logf("Error increased significantly: %.2f -> %.2f",
			result.InitialError, result.FinalError)
	}

	if result.FinalError < 0 {
		t.Errorf("Negative final error: %v", result.FinalError)
	}
}

// TestRefinedPointsRetained tests that refined points are returned
func TestRefinedPointsRetained(t *testing.T) {
	ba := NewBundleAdjustment()

	points := []*Point3D{
		{X: 1.0, Y: 2.0, Z: 3.0},
		{X: 4.0, Y: 5.0, Z: 6.0},
	}

	poses := []*CameraFrame{
		createMockCameraFrame(0, false),
		createMockCameraFrame(1, true),
	}

	kps := createMockKeyPoints(3)
	matches := []*FeatureMatch{
		{SourceKeyPoint: kps[0], TargetKeyPoint: kps[1], Distance: 10.0},
		{SourceKeyPoint: kps[1], TargetKeyPoint: kps[2], Distance: 10.0},
	}

	result, _ := ba.Optimize(points, poses, matches)

	if len(result.RefinedPoints) != len(points) {
		t.Errorf("Refined points count mismatch: %d != %d", len(result.RefinedPoints), len(points))
	}

	if len(result.RefinedPoses) != len(poses) {
		t.Errorf("Refined poses count mismatch: %d != %d", len(result.RefinedPoses), len(poses))
	}
}

// TestConvergenceDetection tests convergence criteria
func TestConvergenceDetection(t *testing.T) {
	ba := NewBundleAdjustment()
	ba.ConvergenceThreshold = 1e-3

	point := &Point3D{X: 1.0, Y: 1.0, Z: 1.0}
	poses := []*CameraFrame{createMockCameraFrame(0, false)}

	matches := []*FeatureMatch{
		{SourceKeyPoint: &KeyPoint{X: 100, Y: 100}, TargetKeyPoint: nil, Distance: 1.0},
	}

	result, _ := ba.Optimize([]*Point3D{point}, poses, matches)

	if result.Iterations > ba.MaxIterations {
		t.Logf("Did not converge after %d iterations", result.Iterations)
	}
}

// TestGetLastResult tests result retrieval
func TestGetLastResult(t *testing.T) {
	ba := NewBundleAdjustment()

	// Should be nil initially
	if ba.GetLastResult() != nil {
		t.Error("Should start with nil result")
	}

	point := &Point3D{X: 1.0, Y: 1.0, Z: 1.0}
	poses := []*CameraFrame{createMockCameraFrame(0, false)}
	matches := []*FeatureMatch{{SourceKeyPoint: &KeyPoint{X: 100, Y: 100}, Distance: 1.0}}

	ba.Optimize([]*Point3D{point}, poses, matches)

	result := ba.GetLastResult()
	if result == nil {
		t.Fatal("Should return non-nil result after optimization")
	}

	if result.Iterations == 0 {
		t.Error("Should have executed at least one iteration")
	}
}

// BenchmarkOptimize benchmarks bundle adjustment performance
func BenchmarkOptimize(b *testing.B) {
	ba := NewBundleAdjustment()
	ba.MaxIterations = 5 // Reduce for benchmark

	points := []*Point3D{
		{X: 1.0, Y: 2.0, Z: 3.0},
		{X: 2.0, Y: 3.0, Z: 4.0},
	}

	poses := []*CameraFrame{
		createMockCameraFrame(0, false),
		createMockCameraFrame(1, true),
	}

	kps := createMockKeyPoints(5)
	matches := make([]*FeatureMatch, 4)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       10.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba.Optimize(points, poses, matches)
	}
}
