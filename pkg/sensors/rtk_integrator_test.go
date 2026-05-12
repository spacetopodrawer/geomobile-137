package sensors

import (
	"context"
	"testing"
	"time"
)

// TestRTKIntegratorConnection tests NTRIP connection establishment
func TestRTKIntegratorConnection(t *testing.T) {
	rtk := NewRTKIntegrator("localhost", 2101, "MOUNT_POINT")

	// Should not be connected initially
	if rtk.IsConnected {
		t.Error("RTK should not be connected initially")
	}

	// Verify structure is valid
	if rtk.TargetAccuracy != 0.05 {
		t.Errorf("Expected target accuracy 0.05m, got %v", rtk.TargetAccuracy)
	}

	if rtk.MeasurementRate != 10 {
		t.Errorf("Expected 10Hz measurement rate, got %d Hz", rtk.MeasurementRate)
	}
}

// TestRTKSolutionAccuracy tests accuracy verification
func TestRTKSolutionAccuracy(t *testing.T) {
	rtk := NewRTKIntegrator("test.server", 2101, "TEST")

	// Test with accurate solution
	goodSolution := &RTKSolution{
		Latitude:      40.7128,
		Longitude:     -74.0060,
		Altitude:      10.0,
		Accuracy:      0.03,  // 3cm (within 5cm target)
		FixedSolution: true,
		Timestamp:     time.Now(),
	}

	if !rtk.VerifyAccuracy(goodSolution) {
		t.Error("Should accept solution with 3cm accuracy (target: 5cm)")
	}

	// Test with poor accuracy
	poorSolution := &RTKSolution{
		Latitude:      40.7128,
		Longitude:     -74.0060,
		Accuracy:      0.15,  // 15cm (exceeds target)
		FixedSolution: false, // Not fixed
		Timestamp:     time.Now(),
	}

	if rtk.VerifyAccuracy(poorSolution) {
		t.Error("Should reject solution with 15cm accuracy (target: 5cm)")
	}

	// Test with nil solution
	if rtk.VerifyAccuracy(nil) {
		t.Error("Should reject nil solution")
	}
}

// TestRTKSolutionCaching tests LastSolution caching
func TestRTKSolutionCaching(t *testing.T) {
	rtk := NewRTKIntegrator("test.server", 2101, "TEST")

	solution := &RTKSolution{
		Latitude:  40.7128,
		Longitude: -74.0060,
		Accuracy:  0.03,
		Timestamp: time.Now(),
	}

	rtk.SetLastSolution(solution)

	retrieved := rtk.GetLastSolution()
	if retrieved == nil {
		t.Fatal("GetLastSolution returned nil")
	}

	if retrieved.Latitude != solution.Latitude {
		t.Errorf("Expected latitude %v, got %v", solution.Latitude, retrieved.Latitude)
	}
}

// TestRTKProcessRawMeasurements tests measurement processing
func TestRTKProcessRawMeasurements(t *testing.T) {
	rtk := NewRTKIntegrator("test.server", 2101, "TEST")

	// Test with not connected
	rawData := make([]byte, 100)
	_, err := rtk.ProcessRawMeasurements(rawData)
	if err == nil {
		t.Error("Should fail when not connected")
	}

	// Test with insufficient data
	rtk.IsConnected = true
	shortData := make([]byte, 5)
	_, err = rtk.ProcessRawMeasurements(shortData)
	if err == nil {
		t.Error("Should fail with insufficient data")
	}

	// Test with valid data length
	validData := make([]byte, 200)
	solution, err := rtk.ProcessRawMeasurements(validData)

	if solution != nil && solution.Timestamp.IsZero() {
		t.Error("Solution should have valid timestamp")
	}
}

// TestRTKIsAccurate tests accuracy status
func TestRTKIsAccurate(t *testing.T) {
	rtk := NewRTKIntegrator("test.server", 2101, "TEST")

	// Initially not accurate
	if rtk.IsAccurate() {
		t.Error("Should not be accurate initially")
	}

	// Set good solution
	rtk.SetLastSolution(&RTKSolution{
		Latitude:      40.7128,
		Longitude:     -74.0060,
		Accuracy:      0.04,  // 4cm
		FixedSolution: true,
		Timestamp:     time.Now(),
	})

	if !rtk.IsAccurate() {
		t.Error("Should report accurate with 4cm fixed solution")
	}

	// Set poor solution
	rtk.SetLastSolution(&RTKSolution{
		Accuracy:      0.2,   // 20cm
		FixedSolution: false,
		Timestamp:     time.Now(),
	})

	if rtk.IsAccurate() {
		t.Error("Should report not accurate with 20cm float solution")
	}
}

// BenchmarkRTKSolutionVerification benchmarks accuracy checking
func BenchmarkRTKSolutionVerification(b *testing.B) {
	rtk := NewRTKIntegrator("test", 2101, "MOUNT")

	solution := &RTKSolution{
		Latitude:      40.7128,
		Longitude:     -74.0060,
		Accuracy:      0.05,
		FixedSolution: true,
		Timestamp:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rtk.VerifyAccuracy(solution)
	}
}

// TestRTKContextTimeout tests context cancellation
func TestRTKContextTimeout(t *testing.T) {
	rtk := NewRTKIntegrator("unreachable.invalid", 2101, "TEST")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should fail quickly with timeout
	err := rtk.Connect(ctx)
	if err == nil {
		t.Error("Should fail to connect to unreachable host")
	}
}

// TestRTKMultipleSolutions tests handling of sequential solutions
func TestRTKMultipleSolutions(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "MOUNT")

	solutions := []*RTKSolution{
		{
			Latitude:      40.7128,
			Longitude:     -74.0060,
			Accuracy:      0.10,
			FixedSolution: false,
			Timestamp:     time.Now(),
		},
		{
			Latitude:      40.7129,
			Longitude:     -74.0059,
			Accuracy:      0.04,
			FixedSolution: true,
			Timestamp:     time.Now().Add(1 * time.Second),
		},
		{
			Latitude:      40.7129,
			Longitude:     -74.0059,
			Accuracy:      0.03,
			FixedSolution: true,
			Timestamp:     time.Now().Add(2 * time.Second),
		},
	}

	for i, sol := range solutions {
		rtk.SetLastSolution(sol)
		last := rtk.GetLastSolution()

		if last.Accuracy != sol.Accuracy {
			t.Errorf("Solution %d: Expected accuracy %v, got %v", i, sol.Accuracy, last.Accuracy)
		}
	}
}

// TestRTKTargetAccuracyConfig tests accuracy threshold configuration
func TestRTKTargetAccuracyConfig(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "MOUNT")

	// Default should be 5cm
	if rtk.TargetAccuracy != 0.05 {
		t.Errorf("Expected default target 0.05m, got %v", rtk.TargetAccuracy)
	}

	// Should be configurable
	rtk.TargetAccuracy = 0.10 // 10cm

	solution := &RTKSolution{
		Accuracy:      0.09,
		FixedSolution: true,
		Timestamp:     time.Now(),
	}

	if !rtk.VerifyAccuracy(solution) {
		t.Error("Should accept 9cm accuracy with 10cm target")
	}
}

// TestRTKDisconnect tests proper connection cleanup
func TestRTKDisconnect(t *testing.T) {
	rtk := NewRTKIntegrator("test", 2101, "MOUNT")

	rtk.IsConnected = true

	err := rtk.Disconnect()
	if err != nil {
		t.Logf("Disconnect error (expected for mock): %v", err)
	}

	if rtk.IsConnected {
		t.Error("Should be disconnected after Disconnect()")
	}
}
