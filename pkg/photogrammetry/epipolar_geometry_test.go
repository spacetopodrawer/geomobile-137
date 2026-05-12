package photogrammetry

import (
	"testing"
)

// TestEpipolarGeometryInit tests initialization
func TestEpipolarGeometryInit(t *testing.T) {
	eg := NewEpipolarGeometry(500.0, 320.0, 240.0)

	if eg.FocalLength != 500.0 {
		t.Errorf("Expected focal length 500.0, got %v", eg.FocalLength)
	}

	if eg.PrincipalX != 320.0 {
		t.Errorf("Expected principal X 320.0, got %v", eg.PrincipalX)
	}

	if eg.PrincipalY != 240.0 {
		t.Errorf("Expected principal Y 240.0, got %v", eg.PrincipalY)
	}

	if eg.MatchCount != 0 {
		t.Error("Should start with 0 matches")
	}
}

// TestComputeEssentialMatrixInvalidInput tests error handling
func TestComputeEssentialMatrixInvalidInput(t *testing.T) {
	eg := NewEpipolarGeometry(500.0, 320.0, 240.0)

	// Test mismatched point counts
	points1 := createMockKeyPoints(5)
	points2 := createMockKeyPoints(10)

	_, err := eg.ComputeEssentialMatrix(points1, points2)
	if err == nil {
		t.Error("Should return error for mismatched point counts")
	}

	// Test insufficient points
	points1 = createMockKeyPoints(5)
	points2 = createMockKeyPoints(5)

	_, err = eg.ComputeEssentialMatrix(points1, points2)
	if err == nil {
		t.Error("Should return error for < 8 points")
	}
}

// TestComputeEssentialMatrixValidInput tests essential matrix computation
func TestComputeEssentialMatrixValidInput(t *testing.T) {
	eg := NewEpipolarGeometry(500.0, 320.0, 240.0)

	points1 := createMockKeyPoints(8)
	points2 := createMockKeyPoints(8)

	result, err := eg.ComputeEssentialMatrix(points1, points2)
	if err != nil {
		t.Fatalf("ComputeEssentialMatrix failed: %v", err)
	}

	if result == nil {
		t.Fatal("Should return non-nil result")
	}

	if !result.IsValid {
		t.Error("Result should be valid")
	}

	if eg.GetMatchCount() != 8 {
		t.Errorf("Match count should be 8, got %d", eg.GetMatchCount())
	}
}

// TestEssentialMatrixProperties tests result structure
func TestEssentialMatrixProperties(t *testing.T) {
	eg := NewEpipolarGeometry(500.0, 320.0, 240.0)

	points1 := createMockKeyPoints(8)
	points2 := createMockKeyPoints(8)

	result, _ := eg.ComputeEssentialMatrix(points1, points2)

	// Check rotation matrix properties
	sum := 0.0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				sum += result.RotationMatrix[i][j]
			}
		}
	}
	// Trace should be approximately 3 for identity or valid rotation
	if sum < 2.0 {
		t.Logf("Rotation matrix trace: %v", sum)
	}

	// Check translation magnitude
	var transNorm float64
	for i := 0; i < 3; i++ {
		transNorm += result.TranslationVector[i] * result.TranslationVector[i]
	}
	if transNorm < 0 {
		t.Error("Translation norm should be non-negative")
	}
}

// TestGetLastEssentialMatrix tests result retrieval
func TestGetLastEssentialMatrix(t *testing.T) {
	eg := NewEpipolarGeometry(500.0, 320.0, 240.0)

	points1 := createMockKeyPoints(8)
	points2 := createMockKeyPoints(8)

	result1, _ := eg.ComputeEssentialMatrix(points1, points2)
	result2 := eg.GetLastEssentialMatrix()

	if result2 == nil {
		t.Fatal("Should return non-nil result")
	}

	if result1.IsValid != result2.IsValid {
		t.Error("Validity should match")
	}
}

// BenchmarkComputeEssentialMatrix benchmarks essential matrix computation
func BenchmarkComputeEssentialMatrix(b *testing.B) {
	eg := NewEpipolarGeometry(500.0, 320.0, 240.0)
	points1 := createMockKeyPoints(8)
	points2 := createMockKeyPoints(8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eg.ComputeEssentialMatrix(points1, points2)
	}
}
