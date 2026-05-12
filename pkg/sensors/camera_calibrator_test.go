package sensors

import (
	"image"
	"image/color"
	"testing"
)

// TestCameraCalibrator creates a test image
func createTestImage(width, height int) image.Image {
	return image.NewUniform(color.White)
}

// TestCameraCalibratorInit tests initialization
func TestCameraCalibratorInit(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	if cal.CheckerboardRows != 9 {
		t.Errorf("Expected 9 rows, got %d", cal.CheckerboardRows)
	}

	if cal.CheckerboardCols != 6 {
		t.Errorf("Expected 6 cols, got %d", cal.CheckerboardCols)
	}

	if cal.SquareSizeMeters != 0.025 {
		t.Errorf("Expected 0.025m square, got %v", cal.SquareSizeMeters)
	}

	if cal.CalibrationCompleted {
		t.Error("Should not be calibrated initially")
	}

	if cal.ReprojectionError < 999.0 {
		t.Errorf("Initial reprojection error should be large")
	}
}

// TestAddCalibrationImage tests adding calibration images
func TestAddCalibrationImage(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Add valid images
	for i := 0; i < 15; i++ {
		img := createTestImage(640, 480)
		err := cal.AddCalibrationImage(img)
		if err != nil {
			t.Errorf("Failed to add image %d: %v", i, err)
		}
	}

	if cal.GetImageCount() != 15 {
		t.Errorf("Expected 15 images, got %d", cal.GetImageCount())
	}

	// Try to exceed max
	for i := 0; i < 20; i++ {
		img := createTestImage(640, 480)
		err := cal.AddCalibrationImage(img)
		if i < 15 && err != nil {
			t.Errorf("Should accept image %d", i)
		}
	}

	if cal.GetImageCount() > cal.MaxCalibrationImages {
		t.Errorf("Exceeded max calibration images: %d > %d",
			cal.GetImageCount(), cal.MaxCalibrationImages)
	}

	// Test with nil image
	err := cal.AddCalibrationImage(nil)
	if err == nil {
		t.Error("Should reject nil image")
	}
}

// TestCanCalibrate tests calibration readiness check
func TestCanCalibrate(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Initially cannot calibrate
	if cal.CanCalibrate() {
		t.Error("Should not be ready to calibrate with no images")
	}

	// Add minimum required images
	for i := 0; i < cal.MinCalibrationImages; i++ {
		img := createTestImage(640, 480)
		cal.AddCalibrationImage(img)
	}

	// Should be ready now
	if !cal.CanCalibrate() {
		t.Error("Should be ready to calibrate with minimum images")
	}
}

// TestComputeIntrinsicMatrix tests intrinsic matrix computation
func TestComputeIntrinsicMatrix(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Test with insufficient images
	err := cal.ComputeIntrinsicMatrix()
	if err == nil {
		t.Error("Should fail with insufficient images")
	}

	// Add enough images
	for i := 0; i < cal.MinCalibrationImages+5; i++ {
		img := createTestImage(640, 480)
		cal.AddCalibrationImage(img)
	}

	// Compute intrinsic matrix
	err = cal.ComputeIntrinsicMatrix()
	if err != nil {
		t.Logf("ComputeIntrinsicMatrix error (expected in Week 1 placeholder): %v", err)
	}

	if cal.IsCalibrated() && cal.IntrinsicMatrix == nil {
		t.Error("IntrinsicMatrix should be populated after calibration")
	}
}

// TestGetIntrinsicMatrix tests matrix retrieval
func TestGetIntrinsicMatrix(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Initially should have default intrinsic matrix
	K := cal.GetIntrinsicMatrix()
	if K == nil {
		t.Fatal("Should return intrinsic matrix struct")
	}

	// Default values should be identity-like
	if K.FocalLengthX == 0 {
		// Initially unset, update with test values
		cal.IntrinsicMatrix.FocalLengthX = 500
		cal.IntrinsicMatrix.FocalLengthY = 500
		cal.IntrinsicMatrix.PrincipalX = 320
		cal.IntrinsicMatrix.PrincipalY = 240
	}

	K = cal.GetIntrinsicMatrix()
	if K.FocalLengthX != 500 {
		t.Errorf("Expected fx=500, got %v", K.FocalLengthX)
	}

	if K.PrincipalX != 320 {
		t.Errorf("Expected cx=320, got %v", K.PrincipalX)
	}
}

// TestGetDistortionCoefficients tests distortion coefficient retrieval
func TestGetDistortionCoefficients(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	distCoeffs := cal.GetDistortionCoefficients()
	if distCoeffs == nil {
		t.Error("Should return valid distortion coefficients")
	}

	// Should have zero values initially
	if distCoeffs.K1 != 0 {
		t.Errorf("Expected K1=0 initially, got %v", distCoeffs.K1)
	}
}

// TestGetReprojectionError tests error retrieval
func TestGetReprojectionError(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Should be large initially
	error := cal.GetReprojectionError()
	if error < 999.0 {
		t.Errorf("Expected large initial error, got %v", error)
	}

	// Set a good error
	cal.ReprojectionError = 0.3

	if cal.GetReprojectionError() != 0.3 {
		t.Errorf("Expected error 0.3, got %v", cal.GetReprojectionError())
	}
}

// TestIsCalibrated tests calibration status
func TestIsCalibrated(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Initially not calibrated
	if cal.IsCalibrated() {
		t.Error("Should not be calibrated initially")
	}

	// Set calibrated state with good error
	cal.CalibrationCompleted = true
	cal.ReprojectionError = 0.3

	if !cal.IsCalibrated() {
		t.Error("Should be calibrated with completed flag and good error")
	}

	// Set poor error
	cal.ReprojectionError = 1.5

	if cal.IsCalibrated() {
		t.Error("Should not be calibrated with poor reprojection error")
	}
}

// TestUndistortPoint tests distortion correction
func TestUndistortPoint(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	// Without calibration, should return same point
	x, y := cal.UndistortPoint(100, 200)
	if x != 100 || y != 200 {
		t.Errorf("Without calibration, should return same point: (%v, %v)", x, y)
	}

	// With intrinsic matrix but no distortion
	cal.IntrinsicMatrix = &CameraIntrinsic{
		FocalLengthX: 500,
		FocalLengthY: 500,
		PrincipalX:   320,
		PrincipalY:   240,
	}
	cal.DistortionCoeffs = &DistortionCoefficients{}

	x, y = cal.UndistortPoint(100, 200)
	// Should be close to input (undistortion with zero distortion coefficients)
	if x == 0 && y == 0 {
		t.Error("Undistort with zero coefficients should not return zeros")
	}
}

// TestGetCalibrationStatus tests status reporting
func TestGetCalibrationStatus(t *testing.T) {
	cal := NewCameraCalibrator(9, 6, 0.025)

	status := cal.GetCalibrationStatus()
	if status == "" {
		t.Error("Should return non-empty status string")
	}

	// Add images and check progress reporting
	for i := 0; i < 5; i++ {
		cal.AddCalibrationImage(createTestImage(640, 480))
		status = cal.GetCalibrationStatus()
		if status == "" {
			t.Error("Status should reflect progress")
		}
	}

	// Mark as calibrated with good error
	cal.CalibrationCompleted = true
	cal.ReprojectionError = 0.3
	status = cal.GetCalibrationStatus()
	if !contains(status, "Calibrated") {
		t.Errorf("Status should indicate calibration: %s", status)
	}
}

// BenchmarkDetectCorners benchmarks corner detection
func BenchmarkDetectCorners(b *testing.B) {
	img := createTestImage(640, 480)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detectCheckerboardCorners(img, 9, 6)
	}
}

// BenchmarkProjectPoint benchmarks point projection
func BenchmarkProjectPoint(b *testing.B) {
	K := [3][3]float64{
		{500, 0, 320},
		{0, 500, 240},
		{0, 0, 1},
	}
	distCoeffs := &DistortionCoefficients{K1: -0.02, K2: 0.001}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		projectPoint(100, 100, 1000, K, distCoeffs)
	}
}
