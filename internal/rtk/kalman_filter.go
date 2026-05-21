package rtk

import (
	"log"
	"math"
)

// KalmanFilter implements a 7-state Kalman filter for sensor fusion
// State: [latitude, longitude, altitude, vx, vy, vz, heading]
type KalmanFilter struct {
	StateVector      [7]float64      // Estimated state
	Covariance       [7][7]float64   // Estimation error covariance
	ProcessNoise     [7][7]float64   // Q matrix
	MeasurementNoise [3][3]float64   // R matrix for position
	Dt               float64         // Time step
}

// NewKalmanFilter creates a new Kalman filter
func NewKalmanFilter() *KalmanFilter {
	kf := &KalmanFilter{
		Dt: 0.1, // 100ms default timestep
	}

	// Initialize state vector
	kf.StateVector = [7]float64{0, 0, 0, 0, 0, 0, 0}

	// Initialize covariance (initial uncertainty)
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			if i == j {
				kf.Covariance[i][j] = 10.0 // 10m² initial uncertainty
			}
		}
	}

	// Initialize process noise
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			if i == j {
				if i < 3 {
					kf.ProcessNoise[i][j] = 0.01 // Position noise
				} else if i < 6 {
					kf.ProcessNoise[i][j] = 0.1 // Velocity noise
				} else {
					kf.ProcessNoise[i][j] = 0.01 // Heading noise
				}
			}
		}
	}

	// Initialize measurement noise (GPS stddev)
	kf.MeasurementNoise[0][0] = 1.0 // Latitude stddev (meters)
	kf.MeasurementNoise[1][1] = 1.0 // Longitude stddev (meters)
	kf.MeasurementNoise[2][2] = 2.0 // Altitude stddev (meters)

	return kf
}

// Predict performs the prediction step of the Kalman filter
func (kf *KalmanFilter) Predict(dt float64) {
	kf.Dt = dt

	// State transition matrix F (constant velocity model)
	F := kf.getStateTransitionMatrix()

	// Predict state: x = F * x
	newState := kf.multiplyMatrixVector(F, kf.StateVector)
	copy(kf.StateVector[:], newState[:])

	// Predict covariance: P = F * P * F^T + Q
	FTranspose := kf.transposeMatrix(F)
	temp := kf.multiplyMatrices(F, kf.Covariance)
	newCovariance := kf.multiplyMatrices(temp, FTranspose)

	// Add process noise
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			newCovariance[i][j] += kf.ProcessNoise[i][j]
		}
	}

	kf.Covariance = newCovariance

	// Ensure covariance remains symmetric and positive-definite
	kf.enforceSymmetry()
	kf.ensurePositiveDefinite()
}

// Update performs the update step with a measurement
func (kf *KalmanFilter) Update(measurement [3]float64) {
	// Measurement model: z = H * x, where H is [1 0 0 0 0 0 0; 0 1 0 0 0 0 0; 0 0 1 0 0 0 0]
	H := [3][7]float64{
		{1, 0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0},
	}

	// Innovation (measurement residual): y = z - H * x
	innovation := kf.computeInnovation(H, measurement)

	// Innovation covariance: S = H * P * H^T + R
	S := kf.computeInnovationCovariance(H)

	// Kalman gain: K = P * H^T * S^(-1)
	K := kf.computeKalmanGain(H, S)

	// Update state: x = x + K * y
	for i := 0; i < 7; i++ {
		for j := 0; j < 3; j++ {
			kf.StateVector[i] += K[i][j] * innovation[j]
		}
	}

	// Update covariance: P = (I - K * H) * P
	I := kf.identityMatrix()
	KH := kf.multiplyMatrices43(K, H)
	IminusKH := kf.subtractMatrices(I, KH)
	newCovariance := kf.multiplyMatrices(IminusKH, kf.Covariance)

	kf.Covariance = newCovariance

	// Ensure covariance remains healthy
	kf.enforceSymmetry()
	kf.ensurePositiveDefinite()
}

// GetState returns the current estimated state
func (kf *KalmanFilter) GetState() [7]float64 {
	return kf.StateVector
}

// getStateTransitionMatrix returns the F matrix for constant velocity model
func (kf *KalmanFilter) getStateTransitionMatrix() [7][7]float64 {
	F := [7][7]float64{}

	// Position = Position + Velocity * dt
	F[0][0] = 1.0 // lat += vx * dt
	F[0][3] = kf.Dt
	F[1][1] = 1.0 // lon += vy * dt
	F[1][4] = kf.Dt
	F[2][2] = 1.0 // alt += vz * dt
	F[2][5] = kf.Dt

	// Velocity remains constant (no acceleration model)
	F[3][3] = 1.0 // vx
	F[4][4] = 1.0 // vy
	F[5][5] = 1.0 // vz

	// Heading remains constant
	F[6][6] = 1.0 // heading

	return F
}

// computeInnovation computes measurement residual
func (kf *KalmanFilter) computeInnovation(H [3][7]float64, measurement [3]float64) [3]float64 {
	var innovation [3]float64

	for i := 0; i < 3; i++ {
		predicted := 0.0
		for j := 0; j < 7; j++ {
			predicted += H[i][j] * kf.StateVector[j]
		}
		innovation[i] = measurement[i] - predicted
	}

	return innovation
}

// computeInnovationCovariance computes S = H * P * H^T + R
func (kf *KalmanFilter) computeInnovationCovariance(H [3][7]float64) [3][3]float64 {
	// H * P
	HP := [3][7]float64{}
	for i := 0; i < 3; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				HP[i][j] += H[i][k] * kf.Covariance[k][j]
			}
		}
	}

	// H * P * H^T + R
	var S [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			S[i][j] = kf.MeasurementNoise[i][j]
			for k := 0; k < 7; k++ {
				S[i][j] += HP[i][k] * H[j][k]
			}
		}
	}

	return S
}

// computeKalmanGain computes K = P * H^T * S^(-1)
func (kf *KalmanFilter) computeKalmanGain(H [3][7]float64, S [3][3]float64) [7][3]float64 {
	// Invert S
	Sinv := kf.invertMatrix3x3(S)

	// P * H^T
	PHT := [7][3]float64{}
	for i := 0; i < 7; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 7; k++ {
				PHT[i][j] += kf.Covariance[i][k] * H[j][k]
			}
		}
	}

	// K = P * H^T * S^(-1)
	var K [7][3]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				K[i][j] += PHT[i][k] * Sinv[k][j]
			}
		}
	}

	return K
}

// invertMatrix3x3 inverts a 3x3 matrix
func (kf *KalmanFilter) invertMatrix3x3(m [3][3]float64) [3][3]float64 {
	var inv [3][3]float64

	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])

	if math.Abs(det) < 1e-10 {
		log.Printf("Warning: Matrix is singular, det=%v", det)
		// Return identity to avoid NaN
		for i := 0; i < 3; i++ {
			inv[i][i] = 1.0
		}
		return inv
	}

	inv[0][0] = (m[1][1]*m[2][2] - m[1][2]*m[2][1]) / det
	inv[0][1] = (m[0][2]*m[2][1] - m[0][1]*m[2][2]) / det
	inv[0][2] = (m[0][1]*m[1][2] - m[0][2]*m[1][1]) / det
	inv[1][0] = (m[1][2]*m[2][0] - m[1][0]*m[2][2]) / det
	inv[1][1] = (m[0][0]*m[2][2] - m[0][2]*m[2][0]) / det
	inv[1][2] = (m[0][2]*m[1][0] - m[0][0]*m[1][2]) / det
	inv[2][0] = (m[1][0]*m[2][1] - m[1][1]*m[2][0]) / det
	inv[2][1] = (m[0][1]*m[2][0] - m[0][0]*m[2][1]) / det
	inv[2][2] = (m[0][0]*m[1][1] - m[0][1]*m[1][0]) / det

	return inv
}

// Helper matrix operations
func (kf *KalmanFilter) multiplyMatrixVector(m [7][7]float64, v [7]float64) [7]float64 {
	var result [7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			result[i] += m[i][j] * v[j]
		}
	}
	return result
}

func (kf *KalmanFilter) multiplyMatrices(a, b [7][7]float64) [7][7]float64 {
	var result [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}

func (kf *KalmanFilter) multiplyMatrices43(a [7][3]float64, b [3][7]float64) [7][7]float64 {
	var result [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 3; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}

func (kf *KalmanFilter) transposeMatrix(m [7][7]float64) [7][7]float64 {
	var result [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			result[j][i] = m[i][j]
		}
	}
	return result
}

func (kf *KalmanFilter) subtractMatrices(a, b [7][7]float64) [7][7]float64 {
	var result [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			result[i][j] = a[i][j] - b[i][j]
		}
	}
	return result
}

func (kf *KalmanFilter) identityMatrix() [7][7]float64 {
	var I [7][7]float64
	for i := 0; i < 7; i++ {
		I[i][i] = 1.0
	}
	return I
}

func (kf *KalmanFilter) enforceSymmetry() {
	for i := 0; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			avg := (kf.Covariance[i][j] + kf.Covariance[j][i]) / 2.0
			kf.Covariance[i][j] = avg
			kf.Covariance[j][i] = avg
		}
	}
}

func (kf *KalmanFilter) ensurePositiveDefinite() {
	// Simple fix: add small value to diagonal
	for i := 0; i < 7; i++ {
		if kf.Covariance[i][i] < 1e-6 {
			kf.Covariance[i][i] = 1e-6
		}
	}
}

// NTRIPClient represents an NTRIP correction client
type NTRIPClient struct {
	url      string
	username string
	password string
	stream   []byte
}

// NewNTRIPClient creates a new NTRIP client
func NewNTRIPClient() *NTRIPClient {
	return &NTRIPClient{}
}

// GetCorrection retrieves RTK corrections from NTRIP caster
func (nc *NTRIPClient) GetCorrection(config RTKConfig) ([]byte, error) {
	// Simplified implementation - in reality would connect to NTRIP caster
	// and stream RTCM corrections over HTTP
	// For now, return empty bytes (will be expanded)
	return []byte{}, nil
}
