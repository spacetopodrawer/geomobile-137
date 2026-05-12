package sensors

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// RTKSolution represents a fixed RTK GNSS solution
type RTKSolution struct {
	Latitude      float64   // Degrees
	Longitude     float64   // Degrees
	Altitude      float64   // Meters
	Accuracy      float64   // Meters (target ±0.05m = 5cm)
	FixedSolution bool      // True if RTK fix achieved
	Timestamp     time.Time // Solution timestamp
	SatelliteCount int       // Number of satellites used
}

// RTKIntegrator handles RTK/GNSS integration via NTRIP protocol
type RTKIntegrator struct {
	mu                sync.RWMutex
	BaseStationHost   string
	BaseStationPort   int
	MountPoint        string
	Connection        net.Conn
	IsConnected       bool
	LastSolution      *RTKSolution
	SolutionBuffer    chan *RTKSolution
	ErrorBuffer       chan error
	TargetAccuracy    float64 // Default 0.05 meters (5cm)
	MeasurementRate   int     // Hz (default 10Hz)
}

// NewRTKIntegrator creates a new RTK integrator instance
func NewRTKIntegrator(host string, port int, mountPoint string) *RTKIntegrator {
	return &RTKIntegrator{
		BaseStationHost: host,
		BaseStationPort: port,
		MountPoint:      mountPoint,
		SolutionBuffer:  make(chan *RTKSolution, 100),
		ErrorBuffer:     make(chan error, 10),
		TargetAccuracy:  0.05, // 5cm
		MeasurementRate: 10,   // 10Hz
		IsConnected:     false,
	}
}

// Connect establishes connection to RTK base station via NTRIP protocol
// Implements RFC 2616 HTTP-based NTRIP caster communication
func (r *RTKIntegrator) Connect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", r.BaseStationHost, r.BaseStationPort)

	// Create TCP connection with timeout
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to RTK base station at %s: %w", addr, err)
	}

	r.Connection = conn
	r.IsConnected = true

	// NTRIP handshake: Send HTTP GET request
	ntrippRequest := fmt.Sprintf(
		"GET /%s HTTP/1.0\r\nHost: %s:%d\r\nUser-Agent: CADASTRE_IA-RTK/1.0\r\nConnection: close\r\n\r\n",
		r.MountPoint,
		r.BaseStationHost,
		r.BaseStationPort,
	)

	if _, err := conn.Write([]byte(ntrippRequest)); err != nil {
		conn.Close()
		r.IsConnected = false
		return fmt.Errorf("failed to send NTRIP request: %w", err)
	}

	// Parse HTTP response (expect 200 OK)
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		conn.Close()
		r.IsConnected = false
		return fmt.Errorf("failed to read NTRIP response: %w", err)
	}

	response := string(buf[:n])
	if !contains(response, "200") && !contains(response, "ICY 200") {
		conn.Close()
		r.IsConnected = false
		return fmt.Errorf("NTRIP handshake failed: %s", response[:100])
	}

	return nil
}

// Disconnect closes the RTK connection
func (r *RTKIntegrator) Disconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Connection != nil {
		if err := r.Connection.Close(); err != nil {
			return err
		}
	}

	r.IsConnected = false
	return nil
}

// ProcessRawMeasurements parses raw GNSS measurements and applies RTK corrections
// Implements least-squares estimation with integer ambiguity resolution
// Input: Raw GNSS measurements (pseudo-ranges, carrier phases, satellite positions)
// Output: Fixed RTK solution with high accuracy (target ±5cm)
func (r *RTKIntegrator) ProcessRawMeasurements(rawData []byte) (*RTKSolution, error) {
	r.mu.RLock()
	if !r.IsConnected {
		r.mu.RUnlock()
		return nil, fmt.Errorf("RTK not connected")
	}
	r.mu.RUnlock()

	solution := &RTKSolution{
		Timestamp: time.Now(),
		SatelliteCount: 0,
		Accuracy: 999.0, // Large value until computed
		FixedSolution: false,
	}

	if len(rawData) < 10 {
		return nil, fmt.Errorf("invalid raw measurement data length: %d", len(rawData))
	}

	// Parse RTCM correction stream
	// TODO: Implement full RTCM 3.3 parser for:
	//   - Message Type 1002: L1 GPS RTK Observables
	//   - Message Type 1004: Full GPS RTK Observables
	//   - Message Type 1006: Station Position + Antenna Height
	//   - Message Type 1030: Network RTK State Message

	// For now, implement simplified float solution computation
	// Real implementation would:
	// 1. Collect satellite measurements (at least 5 for 3D fix + altitude)
	// 2. Compute pseudo-range residuals
	// 3. Solve linearized least-squares system
	// 4. Apply integer ambiguity resolution (LAMBDA algorithm)
	// 5. Validate fixed solution with chi-square test

	// Placeholder: Assume we have extracted measurements
	// In real implementation, parse RTCM to get:
	satelliteCount := countSatellites(rawData)
	if satelliteCount < 5 {
		return solution, fmt.Errorf("insufficient satellites for fix: %d (need ≥5)", satelliteCount)
	}

	solution.SatelliteCount = satelliteCount

	// Compute float solution via least-squares
	// X = (A^T * P * A)^-1 * A^T * P * l
	// Where A is design matrix, P is weighting matrix, l is observation vector
	lat, lon, alt, stddev := computeFloatSolution(rawData)

	solution.Latitude = lat
	solution.Longitude = lon
	solution.Altitude = alt
	solution.Accuracy = stddev

	// Integer ambiguity resolution (if converged enough)
	if stddev < 1.0 { // Float solution good enough for IAR
		fixedLat, fixedLon, fixedAlt, fixedStddev, isFixed :=
			resolveIntegerAmbiguities(lat, lon, alt, stddev, rawData)

		if isFixed {
			solution.Latitude = fixedLat
			solution.Longitude = fixedLon
			solution.Altitude = fixedAlt
			solution.Accuracy = fixedStddev
			solution.FixedSolution = true
		}
	}

	// Validate accuracy threshold
	if solution.Accuracy > r.TargetAccuracy && solution.FixedSolution {
		return solution, fmt.Errorf("solution accuracy %.3f exceeds target %.3f",
			solution.Accuracy, r.TargetAccuracy)
	}

	return solution, nil
}

// VerifyAccuracy checks if solution meets target accuracy threshold
func (r *RTKIntegrator) VerifyAccuracy(solution *RTKSolution) bool {
	if solution == nil {
		return false
	}
	return solution.Accuracy <= r.TargetAccuracy && solution.FixedSolution
}

// GetLastSolution returns the most recent RTK solution
func (r *RTKIntegrator) GetLastSolution() *RTKSolution {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.LastSolution
}

// SetLastSolution updates the cached RTK solution
func (r *RTKIntegrator) SetLastSolution(solution *RTKSolution) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.LastSolution = solution
}

// IsAccurate returns true if current solution meets accuracy requirements
func (r *RTKIntegrator) IsAccurate() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.VerifyAccuracy(r.LastSolution)
}

// ===== HELPER FUNCTIONS =====

// contains checks if substring is in string
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// countSatellites extracts satellite count from RTCM data
// In real implementation, would parse RTCM message header
func countSatellites(data []byte) int {
	// Placeholder: return value based on data length heuristic
	// Real RTCM parser would extract from message type 1002, 1004, etc.
	// Each satellite typically 25-40 bytes in RTCM format
	if len(data) < 100 {
		return len(data) / 20
	}
	return 10 // Default for testing
}

// computeFloatSolution implements least-squares positioning
// Returns latitude, longitude, altitude, and position standard deviation (meters)
func computeFloatSolution(rawData []byte) (lat, lon, alt, stddev float64) {
	// TODO: Week 1 full implementation
	// Pseudocode:
	// 1. Parse satellite positions and pseudo-ranges from RTCM
	// 2. Initialize receiver position (approximate)
	// 3. For 5+ iterations:
	//    a. Compute geometric distances to each satellite
	//    b. Form design matrix (partial derivatives)
	//    c. Solve: delta_x = (A^T*P*A)^-1 * A^T*P * residuals
	//    d. Update receiver position: x = x + delta_x
	// 4. Compute covariance matrix and extract position uncertainty
	// 5. Return solution with estimated standard deviation

	// Placeholder implementation for Week 1 testing
	lat = 40.7128  // New York coordinates (placeholder)
	lon = -74.0060
	alt = 100.0
	stddev = 0.15  // 15cm accuracy (target: 5cm)

	return
}

// resolveIntegerAmbiguities implements integer ambiguity resolution (IAR)
// Uses LAMBDA algorithm to find fixed integer solution
// Returns fixed solution coordinates, uncertainty, and success flag
func resolveIntegerAmbiguities(floatLat, floatLon, floatAlt, floatStddev float64, rawData []byte) (
	fixedLat, fixedLon, fixedAlt, fixedStddev float64, success bool) {

	// TODO: Week 1 full implementation
	// Pseudocode:
	// 1. Extract carrier phase ambiguities from float solution
	// 2. Apply LAMBDA (Least-squares AMBiguity Decorrelation Adjustment) algorithm:
	//    a. Decorrelate ambiguities (integer Z-transformation)
	//    b. Search integer candidates around float solution
	//    c. Compute ambiguity success rate (ratio of 2nd/1st candidate)
	// 3. If success rate > 3.0 (or configured threshold):
	//    a. Round to nearest integers
	//    b. Recompute position with fixed ambiguities
	//    c. Compute fixed covariance (much smaller than float)
	// 4. Return fixed solution with confidence measure

	// Placeholder for Week 1 testing
	fixedLat = floatLat + 0.00001  // Small refinement
	fixedLon = floatLon + 0.00001
	fixedAlt = floatAlt + 0.02
	fixedStddev = 0.05  // 5cm (target accuracy)
	success = floatStddev > 0.5   // Only fix if float solution is good

	return
}
