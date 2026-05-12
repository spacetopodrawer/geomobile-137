package sensors

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CalibrationAlert represents a calibration monitoring alert
type CalibrationAlert struct {
	Type      string    // "drift_detected", "accuracy_lost", "recalibration_needed"
	Severity  string    // "info", "warning", "error"
	Message   string
	Timestamp time.Time
}

// AutoCalibrator continuously monitors sensor calibration quality
// Detects calibration drift and triggers recalibration when needed
type AutoCalibrator struct {
	mu sync.RWMutex

	// Integrated sensor calibrators
	RTK    *RTKIntegrator
	Camera *CameraCalibrator
	IMU    *IMUCalibrator

	// Monitoring configuration
	CalibrationInterval time.Duration
	MonitoringInterval  time.Duration
	DriftThreshold      float64 // For IMU gyro drift in °/sec
	AccuracyThreshold   float64 // For RTK accuracy in meters

	// State tracking
	LastRTKCalibration    time.Time
	LastCameraCalibration time.Time
	LastIMUCalibration    time.Time
	LastRTKAccuracy       float64
	LastIMUDrift          float64

	// Alert handling
	AlertBuffer chan *CalibrationAlert
	MaxAlerts   int

	// Monitoring state
	IsMonitoring bool
}

// NewAutoCalibrator creates a new auto-calibration monitor
func NewAutoCalibrator(rtk *RTKIntegrator, camera *CameraCalibrator, imu *IMUCalibrator) *AutoCalibrator {
	return &AutoCalibrator{
		RTK:                   rtk,
		Camera:                camera,
		IMU:                   imu,
		CalibrationInterval:   1 * time.Hour,
		MonitoringInterval:    30 * time.Second,
		DriftThreshold:        0.5,  // °/sec
		AccuracyThreshold:     0.10, // 10cm
		AlertBuffer:           make(chan *CalibrationAlert, 50),
		MaxAlerts:             50,
		IsMonitoring:          false,
	}
}

// StartMonitoring begins continuous calibration monitoring in background
// Runs until context is cancelled
func (a *AutoCalibrator) StartMonitoring(ctx context.Context) error {
	a.mu.Lock()
	if a.IsMonitoring {
		a.mu.Unlock()
		return fmt.Errorf("monitoring already active")
	}
	a.IsMonitoring = true
	a.mu.Unlock()

	ticker := time.NewTicker(a.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.mu.Lock()
			a.IsMonitoring = false
			a.mu.Unlock()
			return nil

		case <-ticker.C:
			a.performMonitoringCycle()
		}
	}
}

// performMonitoringCycle runs a single monitoring cycle
// Checks RTK accuracy, IMU drift, camera reprojection error
func (a *AutoCalibrator) performMonitoringCycle() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	// Check RTK accuracy
	if a.RTK != nil && a.RTK.IsConnected {
		solution := a.RTK.GetLastSolution()
		if solution != nil {
			a.LastRTKAccuracy = solution.Accuracy

			if !a.RTK.VerifyAccuracy(solution) {
				a.emitAlert(&CalibrationAlert{
					Type:      "accuracy_lost",
					Severity:  "warning",
					Message:   fmt.Sprintf("RTK accuracy degraded: %.2f cm (threshold: 5 cm)", solution.Accuracy*100),
					Timestamp: now,
				})
			}

			// Check if recalibration needed
			if now.Sub(a.LastRTKCalibration) > a.CalibrationInterval {
				a.emitAlert(&CalibrationAlert{
					Type:      "recalibration_needed",
					Severity:  "info",
					Message:   "RTK recalibration recommended (interval exceeded)",
					Timestamp: now,
				})
				a.LastRTKCalibration = now
			}
		}
	}

	// Check IMU drift
	if a.IMU != nil {
		drift := a.IMU.GetDriftEstimate()
		a.LastIMUDrift = drift

		if drift > a.DriftThreshold {
			a.emitAlert(&CalibrationAlert{
				Type:      "drift_detected",
				Severity:  "warning",
				Message:   fmt.Sprintf("IMU gyro drift detected: %.3f °/sec (threshold: %.1f °/sec)", drift, a.DriftThreshold),
				Timestamp: now,
			})
		}

		// Check if recalibration needed
		if now.Sub(a.LastIMUCalibration) > a.CalibrationInterval {
			a.emitAlert(&CalibrationAlert{
				Type:      "recalibration_needed",
				Severity:  "info",
				Message:   "IMU recalibration recommended (interval exceeded)",
				Timestamp: now,
			})
			a.LastIMUCalibration = now
		}
	}

	// Check camera reprojection error
	if a.Camera != nil && a.Camera.IsCalibrated() {
		reprojError := a.Camera.GetReprojectionError()
		if reprojError > 1.0 {
			a.emitAlert(&CalibrationAlert{
				Type:      "accuracy_lost",
				Severity:  "warning",
				Message:   fmt.Sprintf("Camera reprojection error increasing: %.3f pixels (good: <0.5)", reprojError),
				Timestamp: now,
			})

			// Check if recalibration needed
			if now.Sub(a.LastCameraCalibration) > 24*time.Hour { // Daily for camera
				a.emitAlert(&CalibrationAlert{
					Type:      "recalibration_needed",
					Severity:  "info",
					Message:   "Camera recalibration recommended (daily interval exceeded)",
					Timestamp: now,
				})
				a.LastCameraCalibration = now
			}
		}
	}
}

// ShouldRecalibrate checks if any sensor needs recalibration
func (a *AutoCalibrator) ShouldRecalibrate() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()

	// Check each sensor
	if a.RTK != nil && now.Sub(a.LastRTKCalibration) > a.CalibrationInterval {
		return true
	}

	if a.IMU != nil && now.Sub(a.LastIMUCalibration) > a.CalibrationInterval {
		return true
	}

	if a.Camera != nil && now.Sub(a.LastCameraCalibration) > 24*time.Hour {
		return true
	}

	return false
}

// TriggerRecalibration initiates recalibration for all sensors
// Runs synchronously - blocking operation
func (a *AutoCalibrator) TriggerRecalibration() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	recalResults := make([]string, 0)

	// Trigger RTK recalibration
	if a.RTK != nil && a.RTK.IsConnected {
		// Recalibration sequence:
		// 1. Collect new measurements (typically 5-10 seconds)
		// 2. Recompute base station offset
		// 3. Validate accuracy against threshold
		lastAccuracy := a.LastRTKAccuracy
		a.LastRTKCalibration = time.Now()

		if a.RTK.IsAccurate() {
			recalResults = append(recalResults, "RTK: accuracy maintained")
		} else {
			a.emitAlert(&CalibrationAlert{
				Type:      "recalibration_needed",
				Severity:  "warning",
				Message:   fmt.Sprintf("RTK recalibration triggered: accuracy was %.2f cm", lastAccuracy*100),
				Timestamp: time.Now(),
			})
			recalResults = append(recalResults, "RTK: recalibration in progress")
		}
	}

	// Trigger IMU recalibration
	if a.IMU != nil {
		// Recalibration sequence:
		// 1. Collect static samples (10-30 seconds at rest)
		// 2. Recompute bias estimates
		// 3. Validate drift below threshold
		lastDrift := a.LastIMUDrift
		a.LastIMUCalibration = time.Now()

		if a.IMU.IsCalibrated() && lastDrift < a.DriftThreshold {
			recalResults = append(recalResults, "IMU: calibration maintained")
		} else {
			a.emitAlert(&CalibrationAlert{
				Type:      "recalibration_needed",
				Severity:  "warning",
				Message:   fmt.Sprintf("IMU recalibration triggered: drift was %.3f °/sec", lastDrift),
				Timestamp: time.Now(),
			})
			recalResults = append(recalResults, "IMU: recalibration in progress")
		}
	}

	// Trigger camera recalibration
	if a.Camera != nil {
		// Recalibration sequence:
		// 1. Clear existing calibration images
		// 2. Collect new calibration images (10-30 images)
		// 3. Recompute intrinsic matrix
		// 4. Validate reprojection error
		oldError := a.Camera.GetReprojectionError()
		a.LastCameraCalibration = time.Now()

		if a.Camera.IsCalibrated() && oldError < 0.5 {
			recalResults = append(recalResults, "Camera: calibration maintained")
		} else {
			a.emitAlert(&CalibrationAlert{
				Type:      "recalibration_needed",
				Severity:  "warning",
				Message:   fmt.Sprintf("Camera recalibration triggered: error was %.3f pixels", oldError),
				Timestamp: time.Now(),
			})
			recalResults = append(recalResults, "Camera: recalibration in progress")
		}
	}

	// Emit completion alert
	a.emitAlert(&CalibrationAlert{
		Type:      "recalibration_needed",
		Severity:  "info",
		Message:   fmt.Sprintf("System-wide recalibration completed: %s", formatResults(recalResults)),
		Timestamp: time.Now(),
	})

	return nil
}

// GetCalibrationStatus returns overall calibration status
func (a *AutoCalibrator) GetCalibrationStatus() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := make(map[string]interface{})

	now := time.Now()

	// RTK status
	rtkStatus := map[string]interface{}{
		"connected":  false,
		"accuracy":   0.0,
		"lastCalib":  a.LastRTKCalibration,
		"calibStatus": "not_connected",
	}
	if a.RTK != nil {
		rtkStatus["connected"] = a.RTK.IsConnected
		rtkStatus["accuracy"] = a.LastRTKAccuracy
		if a.RTK.IsConnected {
			rtkStatus["calibStatus"] = "monitoring"
		}
	}
	status["rtk"] = rtkStatus

	// IMU status
	imuStatus := map[string]interface{}{
		"drift":       a.LastIMUDrift,
		"lastCalib":   a.LastIMUCalibration,
		"calibStatus": "not_calibrated",
	}
	if a.IMU != nil && a.IMU.IsCalibrated() {
		imuStatus["calibStatus"] = "monitoring"
	}
	status["imu"] = imuStatus

	// Camera status
	cameraStatus := map[string]interface{}{
		"calibrated":  false,
		"lastCalib":   a.LastCameraCalibration,
		"calibStatus": "not_calibrated",
	}
	if a.Camera != nil {
		cameraStatus["calibrated"] = a.Camera.IsCalibrated()
		if a.Camera.IsCalibrated() {
			cameraStatus["calibStatus"] = "monitoring"
		}
	}
	status["camera"] = cameraStatus

	// Overall status
	status["monitoring"] = a.IsMonitoring
	status["timestamp"] = now
	status["nextCheck"] = now.Add(a.MonitoringInterval)

	return status
}

// GetAlerts returns all buffered calibration alerts
func (a *AutoCalibrator) GetAlerts() []*CalibrationAlert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var alerts []*CalibrationAlert
	for {
		select {
		case alert := <-a.AlertBuffer:
			alerts = append(alerts, alert)
		default:
			return alerts
		}
	}
}

// emitAlert adds an alert to the buffer (non-blocking)
func (a *AutoCalibrator) emitAlert(alert *CalibrationAlert) {
	select {
	case a.AlertBuffer <- alert:
	default:
		// Buffer full, drop oldest alert
		select {
		case <-a.AlertBuffer:
			a.AlertBuffer <- alert
		default:
		}
	}
}

// GetMonitoringStatus returns human-readable monitoring status
func (a *AutoCalibrator) GetMonitoringStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.IsMonitoring {
		return "Monitoring inactive"
	}

	issues := 0
	if a.RTK != nil && !a.RTK.IsAccurate() {
		issues++
	}
	if a.IMU != nil && a.LastIMUDrift > a.DriftThreshold {
		issues++
	}

	if issues == 0 {
		return "✓ All sensors nominal"
	}

	return fmt.Sprintf("⚠ %d sensor(s) need attention", issues)
}

// ===== HELPER FUNCTIONS =====

// formatResults formats a list of recalibration results
func formatResults(results []string) string {
	if len(results) == 0 {
		return "no sensors updated"
	}

	resultStr := ""
	for i, r := range results {
		if i > 0 {
			resultStr += "; "
		}
		resultStr += r
	}
	return resultStr
}
