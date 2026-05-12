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
// Uses the Zhang's camera calibration method with checkerboard pattern
// Target accuracy: reprojection error < 0.5 pixels
func (c *CameraCalibrator) ComputeIntrinsicMatrix() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.ImageSequence) < c.MinCalibrationImages {
		return fmt.Errorf("insufficient calibration images: have %d, need %d",
			len(c.ImageSequence), c.MinCalibrationImages)
	}

	// TODO: Implement checkerboard corner detection in all images
	// TODO: Solve for intrinsic matrix using Zhang's method
	//   1. Detect checkerboard corners in each image
	//   2. Compute homography matrix for each view
	//   3. Extract intrinsic parameters from homographies
	//   4. Refine using non-linear optimization (Levenberg-Marquardt)
	// TODO: Compute distortion coefficients
	// TODO: Calculate reprojection error

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
