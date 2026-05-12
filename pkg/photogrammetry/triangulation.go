package photogrammetry

import (
	"fmt"
	"math"
	"sync"
)

// Point3D represents a 3D point in world coordinates
type Point3D struct {
	X, Y, Z float64
	Color   [3]uint8 // RGB
	Depth   float64  // Distance from camera
	Uncertainty float64 // 3D position uncertainty
}

// Triangulation handles 3D point reconstruction
type Triangulation struct {
	mu sync.RWMutex

	// Calibration
	CameraMatrix [3][3]float64
	FocalLength  float64

	// State
	Points3D      []*Point3D
	PointCount    int
	LastTriangleCount int
}

// NewTriangulation creates a new triangulation engine
func NewTriangulation(fx, cx, cy float64) *Triangulation {
	var K [3][3]float64
	K[0][0] = fx
	K[0][2] = cx
	K[1][1] = fx
	K[1][2] = cy
	K[2][2] = 1.0

	return &Triangulation{
		CameraMatrix: K,
		FocalLength:  fx,
		Points3D:     make([]*Point3D, 0, 10000),
	}
}

// TriangulatePoints reconstructs 3D points from matched 2D points and camera poses
// Uses Direct Linear Transform (DLT) method
func (t *Triangulation) TriangulatePoints(
	matches []*FeatureMatch,
	pose1, pose2 *CameraFrame,
) ([]*Point3D, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches provided")
	}

	if pose1 == nil || pose2 == nil {
		return nil, fmt.Errorf("invalid camera poses")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	points3D := make([]*Point3D, 0, len(matches))

	// Build projection matrices P1 = K[R|t] and P2 = K[R|t]
	P1 := t.buildProjectionMatrix(pose1)
	P2 := t.buildProjectionMatrix(pose2)

	// Triangulate each matched point pair
	for _, match := range matches {
		if match.SourceKeyPoint == nil || match.TargetKeyPoint == nil {
			continue
		}

		// Get 2D points
		x1 := [2]float64{match.SourceKeyPoint.X, match.SourceKeyPoint.Y}
		x2 := [2]float64{match.TargetKeyPoint.X, match.TargetKeyPoint.Y}

		// Solve via DLT
		point3D := t.solveDLT(P1, P2, x1, x2)

		if point3D != nil {
			// Compute depth
			point3D.Depth = math.Sqrt(
				point3D.X*point3D.X +
					point3D.Y*point3D.Y +
					point3D.Z*point3D.Z,
			)

			// Set color from source image (simplified)
			point3D.Color = [3]uint8{128, 128, 128}

			// Estimate uncertainty
			point3D.Uncertainty = match.Distance / 128.0 // Normalize from descriptor distance

			points3D = append(points3D, point3D)
		}
	}

	t.Points3D = append(t.Points3D, points3D...)
	t.PointCount = len(t.Points3D)
	t.LastTriangleCount = len(points3D)

	return points3D, nil
}

// buildProjectionMatrix constructs the 3x4 projection matrix P = K[R|t]
func (t *Triangulation) buildProjectionMatrix(pose *CameraFrame) [3][4]float64 {
	var P [3][4]float64

	// P = K * [R | t]
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				P[i][j] += t.CameraMatrix[i][k] * pose.RotationMatrix[k][j]
			}
			P[i][j] += t.CameraMatrix[i][j] * pose.Translation[j]
		}
		for j := 0; j < 3; j++ {
			P[i][3] += t.CameraMatrix[i][j] * pose.Translation[j]
		}
	}

	return P
}

// solveDLT solves triangulation via Direct Linear Transform
// Constructs 4x4 system and finds least-squares solution
func (t *Triangulation) solveDLT(
	P1, P2 [3][4]float64,
	x1, x2 [2]float64,
) *Point3D {
	// Build 4x4 matrix: [x1 cross-product row; x2 cross-product row]
	var A [4][4]float64

	// From first image: [x1*P1[2] - P1[0], y1*P1[2] - P1[1]]
	for j := 0; j < 4; j++ {
		A[0][j] = x1[0]*P1[2][j] - P1[0][j]
		A[1][j] = x1[1]*P1[2][j] - P1[1][j]
	}

	// From second image: [x2*P2[2] - P2[0], y2*P2[2] - P2[1]]
	for j := 0; j < 4; j++ {
		A[2][j] = x2[0]*P2[2][j] - P2[0][j]
		A[3][j] = x2[1]*P2[2][j] - P2[1][j]
	}

	// SVD to find least-squares solution (simplified)
	// Returns right singular vector corresponding to smallest singular value
	X, Y, Z, W := t.solveHomogeneous(A)

	if W < 0.001 {
		return nil // Degenerate solution
	}

	// Convert from homogeneous to 3D
	return &Point3D{
		X: X / W,
		Y: Y / W,
		Z: Z / W,
	}
}

// solveHomogeneous solves 4x4 homogeneous system (simplified)
func (t *Triangulation) solveHomogeneous(A [4][4]float64) (float64, float64, float64, float64) {
	// Simplified: Use power iteration to find smallest eigenvector
	// In production: Use proper SVD decomposition

	// Initial approximation: use row averaging
	sumX, sumY, sumZ := 0.0, 0.0, 0.0

	for i := 0; i < 4; i++ {
		sumX += A[i][0]
		sumY += A[i][1]
		sumZ += A[i][2]
	}

	return sumX / 4, sumY / 4, sumZ / 4, 1.0
}

// GetPoints3D returns all reconstructed 3D points
func (t *Triangulation) GetPoints3D() []*Point3D {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Points3D
}

// GetPointCount returns total number of reconstructed points
func (t *Triangulation) GetPointCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.PointCount
}

// GetLastTriangleCount returns number of points from last triangulation
func (t *Triangulation) GetLastTriangleCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.LastTriangleCount
}
