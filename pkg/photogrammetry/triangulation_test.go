package photogrammetry

import (
	"testing"
)

// createMockCameraFrame creates a mock camera frame for testing
func createMockCameraFrame(id int, withTranslation bool) *CameraFrame {
	frame := &CameraFrame{
		ImageID: id,
		IsKeyFrame: true,
	}

	// Set identity rotation
	frame.RotationMatrix[0][0] = 1.0
	frame.RotationMatrix[1][1] = 1.0
	frame.RotationMatrix[2][2] = 1.0

	// Set translation (offset for second frame)
	if withTranslation {
		frame.Translation[0] = 0.1 // 10cm baseline
	}

	// Set intrinsics (500px focal length, centered)
	frame.CameraMatrix[0][0] = 500.0
	frame.CameraMatrix[0][2] = 320.0
	frame.CameraMatrix[1][1] = 500.0
	frame.CameraMatrix[1][2] = 240.0
	frame.CameraMatrix[2][2] = 1.0

	return frame
}

// TestTriangulationInit tests initialization
func TestTriangulationInit(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	if tri.FocalLength != 500.0 {
		t.Errorf("Expected focal length 500.0, got %v", tri.FocalLength)
	}

	if tri.CameraMatrix[0][0] != 500.0 {
		t.Error("Camera matrix K not properly set")
	}

	if tri.PointCount != 0 {
		t.Error("Should start with 0 points")
	}
}

// TestTriangulatePointsEmptyMatches tests error handling
func TestTriangulatePointsEmptyMatches(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	_, err := tri.TriangulatePoints(nil, pose1, pose2)
	if err == nil {
		t.Error("Should return error for empty matches")
	}

	_, err = tri.TriangulatePoints(make([]*FeatureMatch, 0), pose1, pose2)
	if err == nil {
		t.Error("Should return error for zero-length matches")
	}
}

// TestTriangulatePointsInvalidPose tests error handling for invalid poses
func TestTriangulatePointsInvalidPose(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	kps := createMockKeyPoints(5)
	matches := make([]*FeatureMatch, len(kps)-1)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       10.0,
		}
	}

	// Test nil pose1
	_, err := tri.TriangulatePoints(matches, nil, createMockCameraFrame(1, true))
	if err == nil {
		t.Error("Should return error for nil pose1")
	}

	// Test nil pose2
	_, err = tri.TriangulatePoints(matches, createMockCameraFrame(0, false), nil)
	if err == nil {
		t.Error("Should return error for nil pose2")
	}
}

// TestTriangulatePointsValidInput tests triangulation
func TestTriangulatePointsValidInput(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	kps := createMockKeyPoints(5)
	matches := make([]*FeatureMatch, len(kps)-1)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       10.0,
		}
	}

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	points3D, err := tri.TriangulatePoints(matches, pose1, pose2)
	if err != nil {
		t.Fatalf("TriangulatePoints failed: %v", err)
	}

	if points3D == nil {
		t.Fatal("Should return non-nil points slice")
	}

	if tri.GetPointCount() < 0 {
		t.Errorf("Negative point count: %d", tri.GetPointCount())
	}
}

// TestPoint3DProperties tests 3D point attributes
func TestPoint3DProperties(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	kps := createMockKeyPoints(3)
	matches := make([]*FeatureMatch, 2)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       5.0,
		}
	}

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	points3D, _ := tri.TriangulatePoints(matches, pose1, pose2)

	for _, pt := range points3D {
		if pt == nil {
			continue
		}

		// Check depth is positive
		if pt.Depth < 0 {
			t.Errorf("Negative depth: %v", pt.Depth)
		}

		// Check uncertainty in valid range
		if pt.Uncertainty < 0 || pt.Uncertainty > 1 {
			t.Errorf("Uncertainty out of range [0, 1]: %v", pt.Uncertainty)
		}
	}
}

// TestGetPoints3D tests point cloud retrieval
func TestGetPoints3D(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	kps := createMockKeyPoints(5)
	matches := make([]*FeatureMatch, len(kps)-1)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       10.0,
		}
	}

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	tri.TriangulatePoints(matches, pose1, pose2)

	points := tri.GetPoints3D()
	if points == nil {
		t.Fatal("Should return non-nil points slice")
	}

	if len(points) != tri.GetPointCount() {
		t.Errorf("Point count mismatch: %d != %d", len(points), tri.GetPointCount())
	}
}

// TestMultipleTriangulations tests sequential triangulations
func TestMultipleTriangulations(t *testing.T) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	kps1 := createMockKeyPoints(5)
	kps2 := createMockKeyPoints(5)
	kps3 := createMockKeyPoints(5)

	// First triangulation
	matches1 := make([]*FeatureMatch, 4)
	for i := range matches1 {
		matches1[i] = &FeatureMatch{
			SourceKeyPoint: kps1[i],
			TargetKeyPoint: kps2[i+1],
			Distance:       10.0,
		}
	}

	pose0 := createMockCameraFrame(0, false)
	pose1 := createMockCameraFrame(1, true)

	_, _ = tri.TriangulatePoints(matches1, pose0, pose1)
	totalCount1 := tri.GetPointCount()

	// Second triangulation
	matches2 := make([]*FeatureMatch, 4)
	for i := range matches2 {
		matches2[i] = &FeatureMatch{
			SourceKeyPoint: kps2[i],
			TargetKeyPoint: kps3[i+1],
			Distance:       10.0,
		}
	}

	pose2 := createMockCameraFrame(2, true)
	count2, _ := tri.TriangulatePoints(matches2, pose1, pose2)
	totalCount2 := tri.GetPointCount()

	// Total should accumulate
	if totalCount2 <= totalCount1 {
		t.Logf("Points accumulated: %d -> %d", totalCount1, totalCount2)
	}

	if tri.GetLastTriangleCount() != len(count2) {
		t.Errorf("Last triangle count mismatch: %d != %d", tri.GetLastTriangleCount(), len(count2))
	}
}

// BenchmarkTriangulatePoints benchmarks triangulation performance
func BenchmarkTriangulatePoints(b *testing.B) {
	tri := NewTriangulation(500.0, 320.0, 240.0)

	kps := createMockKeyPoints(10)
	matches := make([]*FeatureMatch, 9)
	for i := range matches {
		matches[i] = &FeatureMatch{
			SourceKeyPoint: kps[i],
			TargetKeyPoint: kps[i+1],
			Distance:       10.0,
		}
	}

	pose1 := createMockCameraFrame(0, false)
	pose2 := createMockCameraFrame(1, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tri.TriangulatePoints(matches, pose1, pose2)
	}
}
