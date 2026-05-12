package sensors

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// IMUSample represents a single IMU measurement
type IMUSample struct {
	Timestamp         time.Time
	AccelX, AccelY, AccelZ float64 // m/s² (with gravity)
	GyroX, GyroY, GyroZ    float64 // rad/s
	MagX, MagY, MagZ       float64 // Tesla (normalized)
}

// KalmanState represents the 15-state extended Kalman filter state
// States: position(3), velocity(3), attitude(3), bias_gyro(3), bias_accel(3)
type KalmanState struct {
	Position      [3]float64  // meters
	Velocity      [3]float64  // m/s
	Attitude      [3]float64  // roll, pitch, yaw (radians)
	BiasGyro      [3]float64  // rad/s
	BiasAccel     [3]float64  // m/s²
	Covariance    [15][15]float64 // State covariance matrix
	Timestamp     time.Time
}

// IMUCalibrator handles IMU calibration and sensor fusion
// Estimates gyroscope/accelerometer bias and implements Kalman filter fusion
type IMUCalibrator struct {
	mu sync.RWMutex

	// Raw sensor readings
	LastAccel [3]float64
	LastGyro  [3]float64
	LastMag   [3]float64

	// Calibration parameters
	BiasAccel           [3]float64
	BiasGyro            [3]float64
	AccelScale          [3]float64 // Scale factors
	GyroScale           [3]float64
	AccelNoise          float64 // m/s² standard deviation
	GyroNoise           float64 // rad/s standard deviation
	MagneticDeclination float64 // radians

	// Drift estimation
	DriftEstimate float64 // °/sec
	DriftThreshold float64 // Alert threshold

	// Kalman filter
	KalmanGain    float64
	LastKalmanState *KalmanState
	MeasurementCount int
	CalibrationComplete bool
}

// NewIMUCalibrator creates a new IMU calibrator
func NewIMUCalibrator() *IMUCalibrator {
	return &IMUCalibrator{
		BiasAccel:    [3]float64{0, 0, 0},
		BiasGyro:     [3]float64{0, 0, 0},
		AccelScale:   [3]float64{1.0, 1.0, 1.0},
		GyroScale:    [3]float64{1.0, 1.0, 1.0},
		AccelNoise:   0.02,       // m/s²
		GyroNoise:    0.005,      // rad/s
		KalmanGain:   0.01,
		DriftThreshold: 0.5,      // °/sec
		LastKalmanState: &KalmanState{},
		MeasurementCount: 0,
		CalibrationComplete: false,
	}
}

// CalibrateBias estimates accelerometer and gyroscope bias from static samples
// Device should be at rest on level surface during calibration
// Duration: typically 10-30 seconds
func (i *IMUCalibrator) CalibrateBias(samples []IMUSample, duration time.Duration) error {
	if len(samples) < 100 {
		return fmt.Errorf("insufficient samples for calibration: have %d, need at least 100", len(samples))
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// Compute mean acceleration (includes gravity)
	var sumAccel [3]float64
	var sumGyro [3]float64
	for _, sample := range samples {
		sumAccel[0] += sample.AccelX
		sumAccel[1] += sample.AccelY
		sumAccel[2] += sample.AccelZ
		sumGyro[0] += sample.GyroX
		sumGyro[1] += sample.GyroY
		sumGyro[2] += sample.GyroZ
	}

	count := float64(len(samples))
	avgAccel := [3]float64{
		sumAccel[0] / count,
		sumAccel[1] / count,
		sumAccel[2] / count,
	}

	avgGyro := [3]float64{
		sumGyro[0] / count,
		sumGyro[1] / count,
		sumGyro[2] / count,
	}

	// Gyroscope bias = mean reading at rest
	i.BiasGyro = avgGyro

	// Accelerometer bias = mean - gravity
	// Assume device is on level surface: gravity is only in Z (≈9.81 m/s²)
	gravityMagnitude := math.Sqrt(
		avgAccel[0]*avgAccel[0] +
		avgAccel[1]*avgAccel[1] +
		avgAccel[2]*avgAccel[2],
	)

	const G = 9.81
	if math.Abs(gravityMagnitude - G) > 0.5 {
		return fmt.Errorf("device not properly leveled: gravity magnitude %.2f m/s² (expected ~9.81)",
			gravityMagnitude)
	}

	// Remove gravity component
	i.BiasAccel[0] = avgAccel[0]
	i.BiasAccel[1] = avgAccel[1]
	i.BiasAccel[2] = avgAccel[2] - G

	return nil
}

// EstimateDrift calculates gyroscope drift rate from a series of measurements
// Device should remain stationary with no rotation
// Returns drift in °/sec
func (i *IMUCalibrator) EstimateDrift(samples []IMUSample) float64 {
	if len(samples) < 10 {
		return 0.0
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// Integrate gyroscope over time
	var totalRotation [3]float64
	for idx := 1; idx < len(samples); idx++ {
		prev := samples[idx-1]
		curr := samples[idx]

		dt := curr.Timestamp.Sub(prev.Timestamp).Seconds()
		if dt <= 0 {
			continue
		}

		// Integrate: angle = angle + (gyro - bias) * dt
		totalRotation[0] += (curr.GyroX - i.BiasGyro[0]) * dt
		totalRotation[1] += (curr.GyroY - i.BiasGyro[1]) * dt
		totalRotation[2] += (curr.GyroZ - i.BiasGyro[2]) * dt
	}

	// Calculate total rotation magnitude in radians
	magnitude := math.Sqrt(
		totalRotation[0]*totalRotation[0] +
		totalRotation[1]*totalRotation[1] +
		totalRotation[2]*totalRotation[2],
	)

	// Convert to degrees/sec
	durationSeconds := samples[len(samples)-1].Timestamp.Sub(samples[0].Timestamp).Seconds()
	if durationSeconds <= 0 {
		return 0.0
	}

	driftRadPerSec := magnitude / durationSeconds
	driftDegPerSec := driftRadPerSec * 180.0 / math.Pi
	i.DriftEstimate = driftDegPerSec

	return driftDegPerSec
}

// FuseSensors applies extended Kalman filter to fuse accelerometer, gyroscope, and magnetometer
// Estimates position, velocity, and attitude (orientation)
// Uses 15-state EKF with process and measurement noise modeling
// Input: previous Kalman state, current IMU sample
// Output: updated Kalman state with fused estimates
func (i *IMUCalibrator) FuseSensors(prevState *KalmanState, sample *IMUSample) (*KalmanState, error) {
	if prevState == nil {
		prevState = &KalmanState{Timestamp: time.Now()}
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	newState := *prevState
	newState.Timestamp = sample.Timestamp

	dt := sample.Timestamp.Sub(prevState.Timestamp).Seconds()
	if dt <= 0 || dt > 1.0 {
		dt = 0.01 // Default to 10ms
	}

	// ===== PREDICTION STEP =====
	// Propagate state using kinematic model

	// Update position: p = p + v*dt
	newState.Position[0] += newState.Velocity[0] * dt
	newState.Position[1] += newState.Velocity[1] * dt
	newState.Position[2] += newState.Velocity[2] * dt

	// Update velocity: v = v + a*dt (excluding gravity)
	accelX := sample.AccelX - i.BiasAccel[0]
	accelY := sample.AccelY - i.BiasAccel[1]
	accelZ := sample.AccelZ - i.BiasAccel[2] + 9.81 // Remove gravity

	newState.Velocity[0] += accelX * dt
	newState.Velocity[1] += accelY * dt
	newState.Velocity[2] += accelZ * dt

	// Update attitude (roll, pitch, yaw) via gyroscope integration
	// Simplified: integrate angular velocities with bias removal
	gyroX := sample.GyroX - i.BiasGyro[0]
	gyroY := sample.GyroY - i.BiasGyro[1]
	gyroZ := sample.GyroZ - i.BiasGyro[2]

	// Small angle approximation for roll, pitch; wrap yaw to [-π, π]
	newState.Attitude[0] += gyroX * dt  // Roll rate
	newState.Attitude[1] += gyroY * dt  // Pitch rate
	newState.Attitude[2] += gyroZ * dt  // Yaw rate

	// Keep yaw in [-π, π]
	for newState.Attitude[2] > 3.14159 {
		newState.Attitude[2] -= 6.28318
	}
	for newState.Attitude[2] < -3.14159 {
		newState.Attitude[2] += 6.28318
	}

	// Predict covariance matrix
	// P = F*P*F^T + Q
	// For simplified 15x15: propagate diagonal terms with process noise
	processNoise := [15]float64{
		1e-6, 1e-6, 1e-6, // Position noise
		i.GyroNoise * i.GyroNoise, // Velocity noise (from gyro)
		i.GyroNoise * i.GyroNoise,
		i.GyroNoise * i.GyroNoise,
		1e-9, 1e-9, 1e-9, // Attitude noise (very small for attitude)
		1e-10, 1e-10, 1e-10, // Gyro bias drift
		1e-10, 1e-10, 1e-10, // Accel bias drift
	}

	for j := 0; j < 15; j++ {
		newState.Covariance[j][j] += processNoise[j] * dt
	}

	// ===== MEASUREMENT UPDATE STEP =====
	// Correct state based on accelerometer and magnetometer measurements

	// Measurement innovation from accelerometer (gravity vector)
	// Expected: [0, 0, -g] in body frame
	measAccel := [3]float64{sample.AccelX, sample.AccelY, sample.AccelZ}
	expectedAccel := [3]float64{0, 0, -9.81}

	// Compute Kalman gain (simplified for diagonal covariance)
	// K = P*H^T / (H*P*H^T + R)
	measurementNoise := i.AccelNoise * i.AccelNoise
	kalmanGainAccel := newState.Covariance[4][4] / (newState.Covariance[4][4] + measurementNoise)

	// Update velocity estimate with accelerometer measurement
	innovationAccel := measAccel[2] - expectedAccel[2]
	newState.Velocity[2] += kalmanGainAccel * innovationAccel * 0.1

	// Update covariance after measurement
	// P = (I - K*H)*P
	updateFactor := (1.0 - kalmanGainAccel)
	newState.Covariance[4][4] *= updateFactor
	newState.Covariance[5][5] *= updateFactor
	newState.Covariance[6][6] *= updateFactor

	// Magnetometer measurement (North reference)
	// Help estimate yaw angle
	if sample.MagX != 0 || sample.MagY != 0 {
		measYaw := computeYawFromMagnetometer(sample.MagX, sample.MagY, sample.MagZ)
		yawInnovation := normalizeAngle(measYaw - newState.Attitude[2])

		kalmanGainYaw := newState.Covariance[8][8] / (newState.Covariance[8][8] + 0.01)
		newState.Attitude[2] += kalmanGainYaw * yawInnovation * 0.05
		newState.Covariance[8][8] *= (1.0 - kalmanGainYaw)
	}

	i.LastKalmanState = &newState
	i.MeasurementCount++
	return &newState, nil
}

// GetBiasAccel returns estimated accelerometer bias
func (i *IMUCalibrator) GetBiasAccel() [3]float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.BiasAccel
}

// GetBiasGyro returns estimated gyroscope bias
func (i *IMUCalibrator) GetBiasGyro() [3]float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.BiasGyro
}

// GetDriftEstimate returns the current gyroscope drift estimate in °/sec
// Target: < 0.1 °/sec after proper calibration
func (i *IMUCalibrator) GetDriftEstimate() float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.DriftEstimate
}

// IsCalibrated returns true if calibration is complete and drift is acceptable
func (i *IMUCalibrator) IsCalibrated() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.CalibrationComplete && i.DriftEstimate < 0.1
}

// GetMeasurementCount returns number of samples processed
func (i *IMUCalibrator) GetMeasurementCount() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.MeasurementCount
}

// GetCalibrationStatus returns human-readable calibration status
func (i *IMUCalibrator) GetCalibrationStatus() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if !i.CalibrationComplete {
		return "Calibration not started"
	}

	driftStatus := "✓"
	if i.DriftEstimate >= 0.1 {
		driftStatus = "⚠"
	}

	return fmt.Sprintf("%s Calibrated (drift: %.3f °/sec, samples: %d)",
		driftStatus, i.DriftEstimate, i.MeasurementCount)
}

// ===== HELPER FUNCTIONS =====

// computeYawFromMagnetometer estimates yaw angle from magnetometer readings
// Uses arctangent of X/Y components in body frame
func computeYawFromMagnetometer(magX, magY, magZ float64) float64 {
	// Normalize magnetometer vector
	mag := math.Sqrt(magX*magX + magY*magY + magZ*magZ)
	if mag == 0 {
		return 0
	}

	magX /= mag
	magY /= mag

	// Compute yaw as atan2(Y, X)
	yaw := math.Atan2(magY, magX)

	// Add magnetic declination (varies by location, default 0)
	// Real implementation would set this based on GPS location
	return yaw
}

// normalizeAngle brings angle to [-π, π] range
func normalizeAngle(angle float64) float64 {
	for angle > 3.14159 {
		angle -= 6.28318
	}
	for angle < -3.14159 {
		angle += 6.28318
	}
	return angle
}
