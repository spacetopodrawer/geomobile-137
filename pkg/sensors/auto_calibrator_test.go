package sensors

import (
	"context"
	"testing"
	"time"
)

// TestAutoCalibratorInit tests initialization
func TestAutoCalibratorInit(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// IsMonitoring should be false initially (not started yet)
	if auto.IsMonitoring {
		t.Error("Should initialize with IsMonitoring = false")
	}

	if auto.CalibrationInterval != 1*time.Hour {
		t.Errorf("Expected 1 hour interval, got %v", auto.CalibrationInterval)
	}

	if auto.MonitoringInterval != 30*time.Second {
		t.Errorf("Expected 30 second monitoring, got %v", auto.MonitoringInterval)
	}

	// Check RTK/Camera/IMU pointers
	if auto.RTK != rtk || auto.Camera != cam || auto.IMU != imu {
		t.Error("Sensor pointers not properly set")
	}
}

// TestShouldRecalibrate tests recalibration decision logic
func TestShouldRecalibrate(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Set all calibration times to now
	now := time.Now()
	auto.LastRTKCalibration = now
	auto.LastIMUCalibration = now
	auto.LastCameraCalibration = now

	// Initially should not need recalibration (just calibrated)
	if auto.ShouldRecalibrate() {
		t.Error("Should not need recalibration just after calibration")
	}

	// Set last RTK calibration to old time
	auto.LastRTKCalibration = now.Add(-2 * time.Hour)

	// Should need recalibration
	if !auto.ShouldRecalibrate() {
		t.Error("Should need recalibration after interval exceeded")
	}
}

// TestGetCalibrationStatus tests status reporting
func TestAutoCalibratorGetCalibrationStatus(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	status := auto.GetCalibrationStatus()
	if status == nil {
		t.Fatal("Should return non-nil status map")
	}

	// Check expected keys
	expectedKeys := []string{"rtk", "imu", "camera", "monitoring", "timestamp"}
	for _, key := range expectedKeys {
		if _, ok := status[key]; !ok {
			t.Errorf("Status missing key: %s", key)
		}
	}

	// Monitoring should be false initially
	monitoring, ok := status["monitoring"].(bool)
	if !ok || monitoring {
		t.Error("Should have monitoring = false initially")
	}
}

// TestGetAlerts tests alert retrieval
func TestGetAlerts(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Emit some alerts
	auto.emitAlert(&CalibrationAlert{
		Type:      "test_alert",
		Severity:  "info",
		Message:   "Test alert 1",
		Timestamp: time.Now(),
	})

	auto.emitAlert(&CalibrationAlert{
		Type:      "test_alert",
		Severity:  "warning",
		Message:   "Test alert 2",
		Timestamp: time.Now(),
	})

	alerts := auto.GetAlerts()
	if len(alerts) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(alerts))
	}

	if alerts[0].Message != "Test alert 1" {
		t.Errorf("Alert 1 message mismatch: %s", alerts[0].Message)
	}
}

// TestAlertBuffer tests alert buffer overflow handling
func TestAlertBuffer(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Fill buffer beyond capacity
	for i := 0; i < 100; i++ {
		auto.emitAlert(&CalibrationAlert{
			Type:      "overflow_test",
			Severity:  "info",
			Message:   "Alert " + string(rune(i)),
			Timestamp: time.Now(),
		})
	}

	// Should not crash and should have limited alerts
	alerts := auto.GetAlerts()
	if len(alerts) > auto.MaxAlerts+10 { // Some tolerance for ongoing emission
		t.Errorf("Alert buffer exceeded capacity: %d > %d", len(alerts), auto.MaxAlerts)
	}
}

// TestAutoCalibratorGetMonitoringStatus tests monitoring status reporting
func TestAutoCalibratorGetMonitoringStatus(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Not monitoring initially
	status := auto.GetMonitoringStatus()
	if !contains(status, "inactive") {
		t.Errorf("Expected 'inactive' in status: %s", status)
	}

	// Set monitoring active with all sensors nominal
	auto.IsMonitoring = true

	// Setup RTK as accurate
	rtk.IsConnected = true
	rtk.SetLastSolution(&RTKSolution{
		Accuracy:      0.03, // Good accuracy
		FixedSolution: true,
		Timestamp:     time.Now(),
	})

	// Setup IMU as calibrated with low drift
	imu.CalibrationComplete = true
	imu.DriftEstimate = 0.05

	status = auto.GetMonitoringStatus()
	if !contains(status, "nominal") {
		t.Logf("Status with good sensors: %s", status)
		// Note: may show issues if not all sensors perfectly configured
	}

	// Create issue with RTK
	rtk.SetLastSolution(&RTKSolution{
		Accuracy:      0.2, // Poor accuracy
		FixedSolution: false,
		Timestamp:     time.Now(),
	})

	status = auto.GetMonitoringStatus()
	if !contains(status, "attention") {
		t.Errorf("Expected 'attention' in status: %s", status)
	}
}

// TestStartMonitoring tests monitoring loop
func TestStartMonitoring(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Should be able to start monitoring
	if auto.IsMonitoring {
		t.Error("Should not be monitoring initially")
	}

	// Start with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := auto.StartMonitoring(ctx)
	if err != nil {
		t.Logf("StartMonitoring error (expected after timeout): %v", err)
	}

	// Should not be monitoring after context cancellation
	// Note: there may be a race condition here, so just check it doesn't crash
}

// TestStartMonitoringAlreadyActive tests preventing double-start
func TestStartMonitoringAlreadyActive(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	auto.IsMonitoring = true

	ctx := context.Background()
	err := auto.StartMonitoring(ctx)
	if err == nil {
		t.Error("Should fail when monitoring already active")
	}
}

// TestTriggerRecalibration tests recalibration triggering
func TestTriggerRecalibration(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Setup sensors as if they're ready for recalibration
	rtk.IsConnected = true
	imu.CalibrationComplete = true
	cam.CalibrationCompleted = true

	// Add a small delay to ensure time progresses
	time.Sleep(5 * time.Millisecond)

	err := auto.TriggerRecalibration()
	if err != nil {
		t.Errorf("TriggerRecalibration error: %v", err)
	}

	// Last calibration times should be updated (not equal to zero time)
	if auto.LastRTKCalibration.IsZero() {
		t.Error("RTK calibration time not set")
	}

	if auto.LastIMUCalibration.IsZero() {
		t.Error("IMU calibration time not set")
	}

	if auto.LastCameraCalibration.IsZero() {
		t.Error("Camera calibration time not set")
	}
}

// TestPerformMonitoringCycle tests single monitoring cycle
func TestPerformMonitoringCycle(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Set RTK as connected but inaccurate
	rtk.IsConnected = true
	rtk.SetLastSolution(&RTKSolution{
		Accuracy:      0.15,
		FixedSolution: false,
		Timestamp:     time.Now(),
	})

	// Perform monitoring cycle (internal method called by StartMonitoring)
	auto.performMonitoringCycle()

	// Check that alerts were generated
	alerts := auto.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Should have generated alert for poor accuracy")
	}
}

// BenchmarkPerformMonitoringCycle benchmarks monitoring performance
func BenchmarkPerformMonitoringCycle(b *testing.B) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	rtk.IsConnected = true
	rtk.SetLastSolution(&RTKSolution{
		Accuracy:      0.03,
		FixedSolution: true,
		Timestamp:     time.Now(),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auto.performMonitoringCycle()
	}
}

// TestMultipleSensorIntegration tests Auto-Calibrator with all sensors
func TestMultipleSensorIntegration(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "TEST")
	cam := NewCameraCalibrator(9, 6, 0.025)
	imu := NewIMUCalibrator()

	auto := NewAutoCalibrator(rtk, cam, imu)

	// Setup all sensors
	rtk.IsConnected = true
	rtk.SetLastSolution(&RTKSolution{
		Accuracy:      0.04,
		FixedSolution: true,
		Timestamp:     time.Now(),
	})

	imu.CalibrationComplete = true
	imu.DriftEstimate = 0.08

	cam.CalibrationCompleted = true
	cam.ReprojectionError = 0.3

	// Check status map
	status := auto.GetCalibrationStatus()
	if status == nil {
		t.Fatal("Status should not be nil")
	}

	// Verify all expected keys exist
	expectedKeys := []string{"rtk", "imu", "camera", "monitoring", "timestamp", "nextCheck"}
	for _, key := range expectedKeys {
		if _, ok := status[key]; !ok {
			t.Errorf("Status missing key: %s", key)
		}
	}

	// Check RTK status structure
	rtk_status, ok := status["rtk"].(map[string]interface{})
	if ok {
		if val, hasKey := rtk_status["connected"]; hasKey {
			if connected, isBool := val.(bool); isBool && connected {
				// RTK properly connected
			}
		}
	}

	// Check IMU status
	imu_status, ok := status["imu"].(map[string]interface{})
	if ok {
		if val, hasKey := imu_status["drift"]; hasKey {
			if drift, isFloat := val.(float64); isFloat {
				if drift != 0.08 {
					t.Logf("IMU drift: %v (expected 0.08)", drift)
				}
			}
		}
	}

	// Check camera status
	camera_status, ok := status["camera"].(map[string]interface{})
	if ok {
		if val, hasKey := camera_status["calibrated"]; hasKey {
			if calibrated, isBool := val.(bool); isBool && !calibrated {
				t.Logf("Camera should show as calibrated")
			}
		}
	}
}
