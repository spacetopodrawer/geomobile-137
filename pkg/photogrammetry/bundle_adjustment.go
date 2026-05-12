package photogrammetry

import (
	"fmt"
	"math"
	"sync"
)

// BundleAdjustmentResult contains optimization results
type BundleAdjustmentResult struct {
	InitialError float64 // Initial reprojection error
	FinalError   float64 // After optimization
	Iterations   int
	Converged    bool
	RefinedPoints []*Point3D
	RefinedPoses  []*CameraFrame
}

// BundleAdjustment performs non-linear refinement of camera poses and 3D points
// Uses Levenberg-Marquardt algorithm for cost minimization
type BundleAdjustment struct {
	mu sync.RWMutex

	// Optimization parameters
	MaxIterations      int     // Max L-M iterations
	DampingFactor      float64 // Initial lambda for L-M
	DampingDecrease    float64 // Reduce lambda on success
	DampingIncrease    float64 // Increase lambda on failure
	ConvergenceThreshold float64 // Stop when error delta < threshold

	// State
	LastResult *BundleAdjustmentResult
}

// NewBundleAdjustment creates a new bundle adjustment optimizer
func NewBundleAdjustment() *BundleAdjustment {
	return &BundleAdjustment{
		MaxIterations:       20,
		DampingFactor:       0.001,
		DampingDecrease:     10.0,
		DampingIncrease:     0.1,
		ConvergenceThreshold: 1e-6,
	}
}

// Optimize refines camera poses and 3D points to minimize reprojection error
func (ba *BundleAdjustment) Optimize(
	points3D []*Point3D,
	poses []*CameraFrame,
	matches []*FeatureMatch,
) (*BundleAdjustmentResult, error) {
	if len(points3D) == 0 || len(poses) == 0 || len(matches) == 0 {
		return nil, fmt.Errorf("invalid input: %d points, %d poses, %d matches",
			len(points3D), len(poses), len(matches))
	}

	ba.mu.Lock()
	defer ba.mu.Unlock()

	result := &BundleAdjustmentResult{
		RefinedPoints: make([]*Point3D, len(points3D)),
		RefinedPoses:  make([]*CameraFrame, len(poses)),
		Converged:     false,
	}

	// Copy input data
	copy(result.RefinedPoints, points3D)
	copy(result.RefinedPoses, poses)

	// Compute initial reprojection error
	result.InitialError = ba.computeReprojectionError(
		result.RefinedPoints, result.RefinedPoses, matches)

	// Levenberg-Marquardt iterations
	lambda := ba.DampingFactor

	for iter := 0; iter < ba.MaxIterations; iter++ {
		result.Iterations = iter + 1

		// Compute Jacobian and Hessian
		jacobian := ba.computeJacobian(result.RefinedPoints, result.RefinedPoses, matches)
		hessian := ba.computeHessian(jacobian)

		// Add damping: H = H + lambda*I
		hessianDamped := ba.addDamping(hessian, lambda)

		// Solve normal equations: (H + lambda*I)*delta = -gradient
		delta := ba.solveNormalEquations(hessianDamped, jacobian)

		// Update parameters
		updatedPoints := ba.updatePoints(result.RefinedPoints, delta, len(result.RefinedPoints))
		updatedPoses := ba.updatePoses(result.RefinedPoses, delta, len(result.RefinedPoses))

		// Compute new error
		newError := ba.computeReprojectionError(updatedPoints, updatedPoses, matches)

		// Check convergence
		errorDelta := math.Abs(result.FinalError - newError)
		if errorDelta < ba.ConvergenceThreshold {
			result.Converged = true
			result.FinalError = newError
			result.RefinedPoints = updatedPoints
			result.RefinedPoses = updatedPoses
			break
		}

		// L-M update rule
		if newError < result.FinalError {
			// Accept update, reduce damping
			result.FinalError = newError
			result.RefinedPoints = updatedPoints
			result.RefinedPoses = updatedPoses
			lambda /= ba.DampingDecrease
		} else {
			// Reject update, increase damping
			lambda *= ba.DampingIncrease
		}
	}

	// Final error if not set
	if result.FinalError == 0 {
		result.FinalError = ba.computeReprojectionError(
			result.RefinedPoints, result.RefinedPoses, matches)
	}

	ba.LastResult = result
	return result, nil
}

// computeReprojectionError calculates total squared reprojection error
func (ba *BundleAdjustment) computeReprojectionError(
	points3D []*Point3D,
	poses []*CameraFrame,
	matches []*FeatureMatch,
) float64 {
	var totalError float64

	for _, match := range matches {
		if match.SourceKeyPoint == nil || match.TargetKeyPoint == nil {
			continue
		}

		// Project first 3D point using first pose
		proj1 := ba.projectPoint(points3D[0], poses[0])
		error1 := ba.residual(proj1, [2]float64{match.SourceKeyPoint.X, match.SourceKeyPoint.Y})

		// Project using second pose
		if len(poses) > 1 && len(points3D) > 1 {
			proj2 := ba.projectPoint(points3D[1], poses[1])
			error2 := ba.residual(proj2, [2]float64{match.TargetKeyPoint.X, match.TargetKeyPoint.Y})
			totalError += error1*error1 + error2*error2
		} else {
			totalError += error1 * error1
		}
	}

	return totalError
}

// projectPoint projects 3D point to 2D image plane
func (ba *BundleAdjustment) projectPoint(point *Point3D, pose *CameraFrame) [2]float64 {
	if point == nil || pose == nil {
		return [2]float64{0, 0}
	}

	// Transform to camera frame: p_cam = R*p_world + t
	x := pose.RotationMatrix[0][0]*point.X + pose.RotationMatrix[0][1]*point.Y +
		pose.RotationMatrix[0][2]*point.Z + pose.Translation[0]
	y := pose.RotationMatrix[1][0]*point.X + pose.RotationMatrix[1][1]*point.Y +
		pose.RotationMatrix[1][2]*point.Z + pose.Translation[1]
	z := pose.RotationMatrix[2][0]*point.X + pose.RotationMatrix[2][1]*point.Y +
		pose.RotationMatrix[2][2]*point.Z + pose.Translation[2]

	// Project to image: p_img = K*p_cam
	if z > 0 {
		u := pose.CameraMatrix[0][0]*x/z + pose.CameraMatrix[0][2]
		v := pose.CameraMatrix[1][1]*y/z + pose.CameraMatrix[1][2]
		return [2]float64{u, v}
	}

	return [2]float64{0, 0}
}

// residual computes difference between projected and measured point
func (ba *BundleAdjustment) residual(projected, measured [2]float64) float64 {
	dx := projected[0] - measured[0]
	dy := projected[1] - measured[1]
	return math.Sqrt(dx*dx + dy*dy)
}

// computeJacobian computes the Jacobian matrix (simplified)
func (ba *BundleAdjustment) computeJacobian(
	points3D []*Point3D,
	poses []*CameraFrame,
	matches []*FeatureMatch,
) [][]float64 {
	// Simplified: Return identity-like matrix
	// In production: Compute actual derivative of reprojection w.r.t. parameters
	size := len(points3D)*3 + len(poses)*6 // 3D points + pose parameters
	jacobian := make([][]float64, len(matches)*2)

	for i := range jacobian {
		jacobian[i] = make([]float64, size)
	}

	return jacobian
}

// computeHessian computes Hessian matrix (simplified)
func (ba *BundleAdjustment) computeHessian(jacobian [][]float64) [][]float64 {
	if len(jacobian) == 0 {
		return nil
	}

	size := len(jacobian[0])
	hessian := make([][]float64, size)

	for i := range hessian {
		hessian[i] = make([]float64, size)
		// Simplified: Diagonal Hessian approximation
		if i < size {
			for j := range jacobian {
				hessian[i][i] += jacobian[j][i] * jacobian[j][i]
			}
		}
	}

	return hessian
}

// addDamping adds Levenberg-Marquardt damping to diagonal
func (ba *BundleAdjustment) addDamping(hessian [][]float64, lambda float64) [][]float64 {
	damped := make([][]float64, len(hessian))

	for i := range hessian {
		damped[i] = make([]float64, len(hessian[i]))
		copy(damped[i], hessian[i])
		if i < len(hessian) {
			damped[i][i] += lambda
		}
	}

	return damped
}

// solveNormalEquations solves H*delta = -gradient (simplified)
func (ba *BundleAdjustment) solveNormalEquations(hessian, jacobian [][]float64) []float64 {
	if len(hessian) == 0 {
		return make([]float64, 0)
	}

	delta := make([]float64, len(hessian))
	// Simplified: Return zero vector
	// In production: Use Cholesky or QR decomposition

	return delta
}

// updatePoints updates 3D points with delta
func (ba *BundleAdjustment) updatePoints(
	points []*Point3D,
	delta []float64,
	numPoints int,
) []*Point3D {
	updated := make([]*Point3D, len(points))
	copy(updated, points)

	// Simplified: Return copy
	// In production: Apply actual delta to point coordinates

	return updated
}

// updatePoses updates camera poses with delta
func (ba *BundleAdjustment) updatePoses(
	poses []*CameraFrame,
	delta []float64,
	numPoses int,
) []*CameraFrame {
	updated := make([]*CameraFrame, len(poses))
	copy(updated, poses)

	// Simplified: Return copy
	// In production: Apply actual delta to rotation/translation

	return updated
}

// GetLastResult returns the last optimization result
func (ba *BundleAdjustment) GetLastResult() *BundleAdjustmentResult {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.LastResult
}
