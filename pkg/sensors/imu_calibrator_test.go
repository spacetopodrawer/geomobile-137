package sensors

import (
	"math"
	"testing"
	"time"
)

// TestIMUCalibratorInit tests initialization
func TestIMUCalibratorInit(t *testing.T) {
	imu := NewIMUCalibrator()

	if imu.AccelNoise != 0.02 {
		t.Errorf("Expected accel noise 0.02, got %v", imu.AccelNoise)
	}

	if imu.GyroNoise != 0.005 {
		t.Errorf("Expected gyro noise 0.005, got %v", imu.GyroNoise)
	}

	if imu.KalmanGain != 0.01 {
		t.Errorf("Expected Kalman gain 0.01, got %v", imu.KalmanGain)
	}

	if imu.CalibrationComplete {
		t.Error("Should not be calibrated initially")
	}
}

// TestCalibrateBias tests bias estimation from static samples
func TestCalibrateBias(t *testing.T) {
	imu := NewIMUCalibrator()

	// Generate static samples (device at rest, level surface)
	samples := make([]IMUSample, 200)
	for i := 0; i < 200; i++ {
		samples[i] = IMUSample{
			Timestamp: time.Now().Add(time.Duration(i) * 5 * time.Millisecond),
			AccelX:    0.01,  // Small noise around zero
			AccelY:    -0.01,
			AccelZ:    9.81,  // Gravity
			GyroX:     0.001, // Small drift
			GyroY:     0.0,
			GyroZ:     -0.002,
			MagX:      0.5,
			MagY:      0.5,
			MagZ:      0.1,
		}
	}

	err := imu.CalibrateBias(samples, 1*time.Second)
	if err != nil {
		t.Logf("CalibrateBias error (checking gravity): %v", err)
	}

	// Check bias estimates
	biasGyro := imu.GetBiasGyro()
	if math.Abs(biasGyro[0]) > 0.01 {
		t.Errorf("Gyro bias X too large: %v", biasGyro[0])
	}

	biasAccel := imu.GetBiasAccel()
	// Z bias should be close to zero (gravity removed)
	if math.Abs(biasAccel[2]) > 0.5 {
		t.Errorf("Accel bias Z should be near zero, got %v", biasAccel[2])
	}
}

// TestCalibrateBiasInsufficientSamples tests error on insufficient data
func TestCalibrateBiasInsufficientSamples(t *testing.T) {
	imu := NewIMUCalibrator()

	samples := make([]IMUSample, 50) // Less than 100
	err := imu.CalibrateBias(samples, 100*time.Millisecond)
	if err == nil {
		t.Error("Should fail with insufficient samples")
	}
}

// TestEstimateDrift tests gyroscope drift calculation
func TestEstimateDrift(t *testing.T) {
	imu := NewIMUCalibrator()

	// Set known bias
	imu.BiasGyro = [3]float64{0.001, 0.002, -0.001}

	// Generate samples with constant drift
	samples := make([]IMUSample, 50)
	for i := 0; i < 50; i++ {
		samples[i] = IMUSample{
			Timestamp: time.Now().Add(time.Duration(i) * 20 * time.Millisecond),
			GyroX:     0.02,  // Drift: 0.02 rad/s
			GyroY:     -0.01,
			GyroZ:     0.005,
		}
	}

	drift := imu.EstimateDrift(samples)

	// Drift should be positive (rotation accumulates)
	if drift <= 0 {
		t.Errorf("Expected positive drift, got %v", drift)
	}

	// Should be in range of a few degrees per second
	if drift > 50 {
		t.Errorf("Drift seems too large: %v °/sec", drift)
	}

	if imu.GetDriftEstimate() != drift {
		t.Errorf("GetDriftEstimate should match EstimateDrift")
	}
}

// TestFuseSensorsBasic tests Kalman filter fusion
func TestFuseSensorsBasic(t *testing.T) {
	imu := NewIMUCalibrator()
	imu.BiasAccel = [3]float64{0, 0, 0}
	imu.BiasGyro = [3]float64{0, 0, 0}

	// Initial state
	prevState := &KalmanState{
		Position:   [3]float64{0, 0, 100},
		Velocity:   [3]float64{0, 0, 0},
		Attitude:   [3]float64{0, 0, 0},
		BiasGyro:   [3]float64{0, 0, 0},
		BiasAccel:  [3]float64{0, 0, 0},
		Timestamp:  time.Now(),
	}

	// Initialize covariance
	for i := 0; i < 15; i++ {
		prevState.Covariance[i][i] = 1.0
	}

	// Sample with gravity
	sample := IMUSample{
		Timestamp: prevState.Timestamp.Add(10 * time.Millisecond),
		AccelX:    0.0,
		AccelY:    0.0,
		AccelZ:    9.81,
		GyroX:     0.0,
		GyroY:     0.0,
		GyroZ:     0.0,
		MagX:      1.0,
		MagY:      0.0,
		MagZ:      0.0,
	}

	newState, err := imu.FuseSensors(prevState, &sample)
	if err != nil {
		t.Logf("FuseSensors error: %v", err)
	}

	if newState == nil {
		t.Fatal("Should return valid state")
	}

	// State should be updated
	if newState.Timestamp != sample.Timestamp {
		t.Errorf("Timestamp not updated")
	}

	// Measurement count should increase
	if imu.GetMeasurementCount() != 1 {
		t.Errorf("Expected 1 measurement, got %d", imu.GetMeasurementCount())
	}
}

// TestFuseSensorsSequential tests multiple fusion steps
func TestFuseSensorsSequential(t *testing.T) {
	imu := NewIMUCalibrator()
	imu.BiasAccel = [3]float64{0, 0, 0}
	imu.BiasGyro = [3]float64{0, 0, 0}

	state := &KalmanState{
		Position:   [3]float64{0, 0, 100},
		Velocity:   [3]float64{0, 0, 0},
		Attitude:   [3]float64{0, 0, 0},
		BiasGyro:   [3]float64{0, 0, 0},
		BiasAccel:  [3]float64{0, 0, 0},
		Timestamp:  time.Now(),
	}

	// Initialize covariance
	for i := 0; i < 15; i++ {
		state.Covariance[i][i] = 1.0
	}

	// Fuse multiple measurements
	for i := 0; i < 10; i++ {
		sample := IMUSample{
			Timestamp: state.Timestamp.Add(10 * time.Millisecond),
			AccelX:    0.0,
			AccelY:    0.0,
			AccelZ:    9.81,
			GyroX:     0.05 * math.Sin(float64(i)*0.1),
			GyroY:     0.0,
			GyroZ:     0.0,
			MagX:      1.0,
			MagY:      0.0,
			MagZ:      0.0,
		}

		newState, _ := imu.FuseSensors(state, &sample)
		state = newState
	}

	// Should have processed 10 measurements
	if imu.GetMeasurementCount() != 10 {
		t.Errorf("Expected 10 measurements, got %d", imu.GetMeasurementCount())
	}

	// Attitude should have changed due to gyro rotation
	if state.Attitude[0] == 0 {
		t.Error("Roll should have changed from gyro input")
	}
}

// TestGetBiasAccel tests accelerometer bias getter
func TestGetBiasAccel(t *testing.T) {
	imu := NewIMUCalibrator()

	imu.BiasAccel = [3]float64{0.1, -0.2, 0.05}
	bias := imu.GetBiasAccel()

	if bias[0] != 0.1 || bias[1] != -0.2 || bias[2] != 0.05 {
		t.Errorf("Bias mismatch: %v", bias)
	}
}

// TestGetBiasGyro tests gyroscope bias getter
func TestGetBiasGyro(t *testing.T) {
	imu := NewIMUCalibrator()

	imu.BiasGyro = [3]float64{0.001, 0.002, -0.001}
	bias := imu.GetBiasGyro()

	if bias[0] != 0.001 || bias[1] != 0.002 || bias[2] != -0.001 {
		t.Errorf("Bias mismatch: %v", bias)
	}
}

// TestIMUIsCalibrated tests calibration status
func TestIMUIsCalibrated(t *testing.T) {
	imu := NewIMUCalibrator()

	// Initially not calibrated
	if imu.IsCalibrated() {
		t.Error("Should not be calibrated initially")
	}

	// Set calibration complete with good drift
	imu.CalibrationComplete = true
	imu.DriftEstimate = 0.05

	if !imu.IsCalibrated() {
		t.Error("Should be calibrated with good drift estimate")
	}

	// Set poor drift
	imu.DriftEstimate = 0.2

	if imu.IsCalibrated() {
		t.Error("Should not be calibrated with poor drift")
	}
}

// BenchmarkFuseSensors benchmarks Kalman filter performance
func BenchmarkFuseSensors(b *testing.B) {
	imu := NewIMUCalibrator()

	state := &KalmanState{
		Position:   [3]float64{0, 0, 100},
		Velocity:   [3]float64{1, 2, 0},
		Attitude:   [3]float64{0, 0, 0},
		BiasGyro:   [3]float64{0, 0, 0},
		BiasAccel:  [3]float64{0, 0, 0},
		Timestamp:  time.Now(),
	}

	for i := 0; i < 15; i++ {
		state.Covariance[i][i] = 1.0
	}

	sample := IMUSample{
		Timestamp: state.Timestamp.Add(10 * time.Millisecond),
		AccelX:    0.1,
		AccelY:    0.0,
		AccelZ:    9.81,
		GyroX:     0.01,
		GyroY:     0.0,
		GyroZ:     0.0,
		MagX:      1.0,
		MagY:      0.0,
		MagZ:      0.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		imu.FuseSensors(state, &sample)
	}
}

// TestGetMeasurementCount tests measurement counter
func TestGetMeasurementCount(t *testing.T) {
	imu := NewIMUCalibrator()

	if imu.GetMeasurementCount() != 0 {
		t.Error("Should start with 0 measurements")
	}
}

// TestIMUGetCalibrationStatus tests status reporting
func TestIMUGetCalibrationStatus(t *testing.T) {
	imu := NewIMUCalibrator()

	status := imu.GetCalibrationStatus()
	if status == "" {
		t.Error("Should return non-empty status")
	}

	if !contains(status, "not started") && !contains(status, "Calibration") {
		t.Errorf("Status should mention calibration state: %s", status)
	}

	// Set calibrated
	imu.CalibrationComplete = true
	imu.DriftEstimate = 0.05
	imu.MeasurementCount = 100

	status = imu.GetCalibrationStatus()
	if !contains(status, "Calibrated") {
		t.Errorf("Status should indicate calibration: %s", status)
	}

	if !contains(status, "0.050") && !contains(status, "100") {
		t.Errorf("Status should show drift and sample count: %s", status)
	}
}

// TestNormalizeAngle tests angle normalization helper
func TestNormalizeAngle(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{1.57, 1.57},         // π/2
		{3.14159, 3.14159},    // π
		{6.28318, 0},          // 2π → 0
		{-3.14159, -3.14159},  // -π
	}

	for _, tt := range tests {
		result := normalizeAngle(tt.input)
		diff := math.Abs(result - tt.expected)
		// Allow for some floating point error
		if diff > 0.01 {
			t.Errorf("normalizeAngle(%v) = %v, expected ~%v (diff: %v)", tt.input, result, tt.expected, diff)
		}
	}

	// Test that result is in [-π, π] range
	rangeTests := []float64{0, 1.57, 3.14159, 6.28318, 9.42478, -3.14159, -6.28318}
	for _, angle := range rangeTests {
		result := normalizeAngle(angle)
		if result > 3.14159 || result < -3.14159 {
			t.Errorf("normalizeAngle(%v) = %v is outside [-π, π] range", angle, result)
		}
	}
}

// TestComputeYaw tests yaw computation from magnetometer
func TestComputeYaw(t *testing.T) {
	// North: magX=1, magY=0
	yaw := computeYawFromMagnetometer(1.0, 0.0, 0.0)
	if math.Abs(yaw) > 0.1 {
		t.Errorf("North should have yaw ~0, got %v rad", yaw)
	}

	// East: magX=0, magY=1
	yaw = computeYawFromMagnetometer(0.0, 1.0, 0.0)
	expectedEast := math.Pi / 2
	if math.Abs(yaw-expectedEast) > 0.1 {
		t.Errorf("East should have yaw ~π/2, got %v rad", yaw)
	}

	// Zero magnitude
	yaw = computeYawFromMagnetometer(0.0, 0.0, 0.0)
	if yaw != 0 {
		t.Errorf("Zero magnitude should return 0, got %v", yaw)
	}
}
