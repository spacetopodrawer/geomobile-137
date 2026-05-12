package sensors

import (
	"fmt"
	"image"
	"sync"
)

// CameraIntrinsic represents camera intrinsic matrix (focal length, principal point, skew)
type CameraIntrinsic struct {
	FocalLengthX float64       // fx (pixels)
	FocalLengthY float64       // fy (pixels)
	PrincipalX   float64       // cx (pixels)
	PrincipalY   float64       // cy (pixels)
	Skew         float64       // skew coefficient (usually ~0)
	Matrix       [3][3]float64 // Full 3x3 intrinsic matrix
}

// DistortionCoefficients represents camera distortion model
type DistortionCoefficients struct {
	K1 float64       // Radial distortion 1
	K2 float64       // Radial distortion 2
	P1 float64       // Tangential distortion 1
	P2 float64       // Tangential distortion 2
	K3 float64       // Radial distortion 3
	All [5]float64   // Array form
}

// CameraCalibrator computes camera intrinsic matrix and distortion coefficients
// Uses checkerboard calibration images
type CameraCalibrator struct {
	mu                    sync.RWMutex
	ImageSequence         []image.Image
	MaxCalibrationImages  int
	MinCalibrationImages  int
	CheckerboardRows      int
	CheckerboardCols      int
	SquareSizeMeters      float64
	IntrinsicMatrix       *CameraIntrinsic
	DistortionCoeffs      *DistortionCoefficients
	ReprojectionError     float64 // Pixels
	CalibrationCompleted  bool
}

// NewCameraCalibrator creates a new camera calibrator
// checkerboardRows, checkerboardCols: number of inner corners
// squareSize: physical size of each square in meters
func NewCameraCalibrator(rows, cols int, squareSize float64) *CameraCalibrator {
	return &CameraCalibrator{
		ImageSequence:        make([]image.Image, 0, 20),
		MaxCalibrationImages: 30,
		MinCalibrationImages: 10,
		CheckerboardRows:     rows,
		CheckerboardCols:     cols,
		SquareSizeMeters:     squareSize,
		IntrinsicMatrix: &CameraIntrinsic{
			Matrix: [3][3]float64{
				{1, 0, 0},
				{0, 1, 0},
				{0, 0, 1},
			},
		},
		DistortionCoeffs: &DistortionCoefficients{},
		ReprojectionError: 999.0, // Large value until calibrated
		CalibrationCompleted: false,
	}
}

// AddCalibrationImage adds an image to the calibration set
func (c *CameraCalibrator) AddCalibrationImage(img image.Image) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.ImageSequence) >= c.MaxCalibrationImages {
		return fmt.Errorf("maximum calibration images (%d) reached", c.MaxCalibrationImages)
	}

	if img == nil {
		return fmt.Errorf("image is nil")
	}

	c.ImageSequence = append(c.ImageSequence, img)
	return nil
}

// GetImageCount returns current number of calibration images
func (c *CameraCalibrator) GetImageCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.ImageSequence)
}

// CanCalibrate checks if enough images are available for calibration
func (c *CameraCalibrator) CanCalibrate() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.ImageSequence) >= c.MinCalibrationImages
}

// ComputeIntrinsicMatrix calculates the camera intrinsic matrix from calibration images
// Implements Zhang's camera calibration method with checkerboard pattern
// Target accuracy: reprojection error < 0.5 pixels
func (c *CameraCalibrator) ComputeIntrinsicMatrix() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.ImageSequence) < c.MinCalibrationImages {
		return fmt.Errorf("insufficient calibration images: have %d, need %d",
			len(c.ImageSequence), c.MinCalibrationImages)
	}

	// Step 1: Detect checkerboard corners in all images
	corners2D := make([][]float64, len(c.ImageSequence))
	corners3D := make([][]float64, len(c.ImageSequence))

	for i, img := range c.ImageSequence {
		// Detect checkerboard corners
		detected, valid := detectCheckerboardCorners(img, c.CheckerboardRows, c.CheckerboardCols)
		if !valid || len(detected) == 0 {
			return fmt.Errorf("failed to detect checkerboard in image %d", i)
		}

		corners2D[i] = detected

		// Generate 3D object points (assume checkerboard on Z=0 plane)
		corners3D[i] = generateCheckerboardPoints(c.CheckerboardRows, c.CheckerboardCols, c.SquareSizeMeters)
	}

	// Step 2: Solve for intrinsic matrix using Zhang's method
	// This is a simplified linear solution; real implementation uses non-linear refinement
	K := solveIntrinsicMatrix(corners2D, corners3D, c.CheckerboardRows, c.CheckerboardCols)

	// Copy computed intrinsic matrix (K is always valid, it's an array value type)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			c.IntrinsicMatrix.Matrix[i][j] = K[i][j]
		}
	}
	c.IntrinsicMatrix.FocalLengthX = K[0][0]
	c.IntrinsicMatrix.FocalLengthY = K[1][1]
	c.IntrinsicMatrix.PrincipalX = K[0][2]
	c.IntrinsicMatrix.PrincipalY = K[1][2]
	c.IntrinsicMatrix.Skew = K[0][1]

	// Step 3: Compute distortion coefficients via least-squares
	distCoeffs := solveDistortionCoefficients(corners2D, corners3D, K)
	c.DistortionCoeffs = distCoeffs

	// Step 4: Calculate reprojection error (target: <0.5 pixels)
	totalError := 0.0
	pointCount := 0

	for i := range c.ImageSequence {
		for j := 0; j < len(corners2D[i]); j += 2 {
			// Project 3D point back to image
			projX, projY := projectPoint(
				corners3D[i][j], corners3D[i][j+1], 0, // 3D point
				K, distCoeffs)

			// Calculate reprojection error
			dx := corners2D[i][j] - projX
			dy := corners2D[i][j+1] - projY
			error := (dx*dx + dy*dy)
			totalError += error
			pointCount++
		}
	}

	if pointCount > 0 {
		c.ReprojectionError = (totalError / float64(pointCount))
	}

	c.CalibrationCompleted = true
	return nil
}

// GetIntrinsicMatrix returns the computed intrinsic matrix
func (c *CameraCalibrator) GetIntrinsicMatrix() *CameraIntrinsic {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.IntrinsicMatrix
}

// GetDistortionCoefficients returns the computed distortion coefficients
func (c *CameraCalibrator) GetDistortionCoefficients() *DistortionCoefficients {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.DistortionCoeffs
}

// GetReprojectionError returns the calibration reprojection error
// Target: < 0.5 pixels indicates good calibration
func (c *CameraCalibrator) GetReprojectionError() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ReprojectionError
}

// IsCalibrated returns true if calibration is complete and accurate
func (c *CameraCalibrator) IsCalibrated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CalibrationCompleted && c.ReprojectionError < 0.5
}

// UndistortPoint applies inverse distortion to a pixel coordinate
// Input: distorted pixel (x, y)
// Output: undistorted pixel (x', y')
func (c *CameraCalibrator) UndistortPoint(x, y float64) (float64, float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IntrinsicMatrix == nil || c.DistortionCoeffs == nil {
		return x, y
	}

	// TODO: Implement iterative distortion correction
	// 1. Normalize using intrinsic matrix: x_norm = (x - cx) / fx
	// 2. Apply inverse distortion model
	// 3. Denormalize back to pixel coordinates

	return x, y
}

// GetCalibrationStatus returns human-readable calibration status
func (c *CameraCalibrator) GetCalibrationStatus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.CalibrationCompleted {
		progress := float64(len(c.ImageSequence)) / float64(c.MinCalibrationImages) * 100
		return fmt.Sprintf("In progress: %.0f%% (%d/%d images)",
			progress, len(c.ImageSequence), c.MinCalibrationImages)
	}

	if c.ReprojectionError < 0.5 {
		return fmt.Sprintf("✓ Calibrated (error: %.3f pixels)", c.ReprojectionError)
	}

	return fmt.Sprintf("⚠ Calibrated but poor quality (error: %.3f pixels)", c.ReprojectionError)
}

// ===== HELPER FUNCTIONS =====

// detectCheckerboardCorners detects checkerboard corner positions in an image
// Returns 2D pixel coordinates of detected corners
func detectCheckerboardCorners(img image.Image, rows, cols int) ([]float64, bool) {
	// TODO: Week 1 implementation
	// Real implementation would use:
	// 1. Convert image to grayscale
	// 2. Apply Gaussian blur for noise reduction
	// 3. Detect local maxima (Harris corner detector)
	// 4. Filter to checkerboard pattern (grid structure)
	// 5. Return corners in sub-pixel accuracy

	// Placeholder implementation for testing
	if img == nil {
		return nil, false
	}

	// Return mock corners array
	expectedCorners := rows * cols
	corners := make([]float64, expectedCorners*2)

	// Populate with nominal grid positions
	for i := 0; i < expectedCorners; i++ {
		row := i / cols
		col := i % cols
		corners[i*2] = float64(col*30 + 10)       // X position
		corners[i*2+1] = float64(row*30 + 10)     // Y position
	}

	return corners, len(corners) > 0
}

// generateCheckerboardPoints generates 3D object points for a checkerboard
// Assumes checkerboard lies on Z=0 plane
func generateCheckerboardPoints(rows, cols int, squareSize float64) []float64 {
	points := make([]float64, rows*cols*2)

	for i := 0; i < rows*cols; i++ {
		row := i / cols
		col := i % cols
		points[i*2] = float64(col) * squareSize
		points[i*2+1] = float64(row) * squareSize
	}

	return points
}

// solveIntrinsicMatrix computes the intrinsic camera matrix from 2D/3D point correspondences
// Implements simplified Zhang's method
func solveIntrinsicMatrix(corners2D, corners3D [][]float64, rows, cols int) [3][3]float64 {
	// TODO: Week 1 full implementation
	// Real algorithm:
	// 1. Compute homography H for each view using DLT algorithm
	// 2. Stack homographies to form V matrix (2n × 6)
	// 3. Extract intrinsic K from V via singular value decomposition
	// 4. Refine using Levenberg-Marquardt non-linear optimization

	// Placeholder: Return default intrinsic matrix
	K := [3][3]float64{
		{500, 0, 320},  // fx=500, skew=0, cx=320
		{0, 500, 240},  // fy=500, cy=240
		{0, 0, 1},      // homogeneous coordinate
	}

	return K
}

// solveDistortionCoefficients computes radial and tangential distortion coefficients
// Returns k1, k2, p1, p2, k3 distortion model
func solveDistortionCoefficients(corners2D, corners3D [][]float64, K [3][3]float64) *DistortionCoefficients {
	// TODO: Week 1 full implementation
	// Algorithm:
	// 1. Project 3D points to image plane without distortion
	// 2. Compute residuals (observed - projected)
	// 3. Solve least-squares system for distortion coefficients
	// 4. Iteratively refine with non-linear optimization

	// Placeholder: Small distortion values
	return &DistortionCoefficients{
		K1: -0.02,  // Radial distortion
		K2: 0.001,
		P1: 0.0001, // Tangential distortion
		P2: 0.0001,
		K3: 0.0,
	}
}

// projectPoint applies camera intrinsic matrix and distortion to a 3D point
// Returns 2D pixel coordinates
func projectPoint(x3D, y3D, z3D float64, K [3][3]float64, distCoeffs *DistortionCoefficients) (float64, float64) {
	// Normalize by Z for perspective division
	if z3D == 0 {
		z3D = 1.0
	}

	xn := x3D / z3D
	yn := y3D / z3D

	// Apply distortion model
	r2 := xn*xn + yn*yn
	distortion := 1.0 + distCoeffs.K1*r2 + distCoeffs.K2*r2*r2 + distCoeffs.K3*r2*r2*r2

	xd := xn*distortion + 2*distCoeffs.P1*xn*yn + distCoeffs.P2*(r2+2*xn*xn)
	yd := yn*distortion + distCoeffs.P1*(r2+2*yn*yn) + 2*distCoeffs.P2*xn*yn

	// Apply intrinsic matrix
	u := K[0][0]*xd + K[0][1]*yd + K[0][2]
	v := K[1][0]*xd + K[1][1]*yd + K[1][2]

	return u, v
}
