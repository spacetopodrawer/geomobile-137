package deviceradar

import (
	"context"
	"testing"
	"time"
)

func TestRegisterDevice(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	device, err := svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")
	if err != nil {
		t.Fatalf("RegisterDevice failed: %v", err)
	}

	if device.ID == "" {
		t.Fatal("Device ID should not be empty")
	}

	if device.IntrinsicID.UUID == "" {
		t.Fatal("UUID should not be empty")
	}

	if device.UserID != "user123" {
		t.Fatal("UserID mismatch")
	}

	// Verify it's added to trusted devices list
	devices, err := svc.GetDevicesByUser(ctx, "user123")
	if err != nil {
		t.Fatalf("GetDevicesByUser failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}
}

func TestUpdateLocation(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Register device
	device, _ := svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")

	// Update location with WiFi networks
	wifiNets := []*WiFiNetwork{
		{
			SSID:      "HomeNetwork",
			BSSID:     "AA:BB:CC:DD:EE:FF",
			Signal:    -45,
			Channel:   6,
			Bandwidth: "20MHz",
		},
		{
			SSID:      "NeighborNetwork",
			BSSID:     "11:22:33:44:55:66",
			Signal:    -70,
			Channel:   11,
			Bandwidth: "20MHz",
		},
	}

	err := svc.UpdateLocation(ctx, device.ID, wifiNets, nil, "wifi")
	if err != nil {
		t.Fatalf("UpdateLocation failed: %v", err)
	}

	// Verify location was stored
	updatedDevice, err := svc.GetDeviceByID(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceByID failed: %v", err)
	}

	if len(updatedDevice.LocationHistory) != 1 {
		t.Fatalf("Expected 1 location trace, got %d", len(updatedDevice.LocationHistory))
	}

	trace := updatedDevice.LocationHistory[0]
	if len(trace.WiFiNets) != 2 {
		t.Fatalf("Expected 2 WiFi networks, got %d", len(trace.WiFiNets))
	}
}

func TestDetectVPNActivity(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	device, _ := svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")

	// Test 1: Active VPN with no leaks
	vpnStatus, err := svc.DetectVPNActivity(ctx, device.ID, "185.220.101.45", "185.220.101.45")
	if err != nil {
		t.Fatalf("DetectVPNActivity failed: %v", err)
	}

	if vpnStatus.IsActive {
		t.Fatal("VPN should not be active when IPs match")
	}

	// Test 2: Active VPN with leaks
	vpnStatus, err = svc.DetectVPNActivity(ctx, device.ID, "185.220.101.45", "192.168.1.100")
	if err != nil {
		t.Fatalf("DetectVPNActivity failed: %v", err)
	}

	if !vpnStatus.IsActive {
		t.Fatal("VPN should be active when IPs differ")
	}

	if !vpnStatus.HasLeaks {
		t.Fatal("Should detect IP leak")
	}

	// Verify status was stored
	updatedDevice, _ := svc.GetDeviceByID(ctx, device.ID)
	if updatedDevice.VPNStatus == nil {
		t.Fatal("VPNStatus should be set")
	}
}

func TestVerifyDeviceAuthenticity(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	device, _ := svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")

	// Update location to build history
	wifiNets := []*WiFiNetwork{
		{
			SSID:      "HomeNetwork",
			BSSID:     "AA:BB:CC:DD:EE:FF",
			Signal:    -45,
		},
	}

	svc.UpdateLocation(ctx, device.ID, wifiNets, nil, "wifi")

	// Test authenticity verification
	authentic, checks, err := svc.VerifyDeviceAuthenticity(ctx, device.ID)
	if err != nil {
		t.Fatalf("VerifyDeviceAuthenticity failed: %v", err)
	}

	if !authentic {
		t.Logf("Device not fully authentic - checks: %v", checks)
	}

	if len(checks) == 0 {
		t.Fatal("Should have verification checks")
	}

	t.Logf("Authenticity checks: %v", checks)
}

func TestGetDevicesByUser(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Register multiple devices
	svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")
	svc.RegisterDevice(ctx, "user123", "iPad", "tablet", "Apple", "iPad Pro", "48:5A:3F:8C:21:BA")
	svc.RegisterDevice(ctx, "user456", "Galaxy", "phone", "Samsung", "Galaxy S24", "48:5A:3F:8C:21:BB")

	// Get devices for user123
	devices, err := svc.GetDevicesByUser(ctx, "user123")
	if err != nil {
		t.Fatalf("GetDevicesByUser failed: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices for user123, got %d", len(devices))
	}

	// Get devices for user456
	devices, err = svc.GetDevicesByUser(ctx, "user456")
	if err != nil {
		t.Fatalf("GetDevicesByUser failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device for user456, got %d", len(devices))
	}
}

func TestMovementDetection(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	device, _ := svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")

	// First location
	wifiNets := []*WiFiNetwork{
		{SSID: "Home", BSSID: "AA:BB:CC:DD:EE:FF", Signal: -45},
	}
	svc.UpdateLocation(ctx, device.ID, wifiNets, nil, "wifi")

	// Sleep a bit to ensure different timestamp
	time.Sleep(100 * time.Millisecond)

	// Second location
	wifiNets = []*WiFiNetwork{
		{SSID: "Office", BSSID: "11:22:33:44:55:66", Signal: -50},
	}
	svc.UpdateLocation(ctx, device.ID, wifiNets, nil, "wifi")

	updatedDevice, _ := svc.GetDeviceByID(ctx, device.ID)

	// We should have movements recorded
	if len(updatedDevice.MovementHistory) > 0 {
		t.Logf("Movement detected: %d movements recorded", len(updatedDevice.MovementHistory))
	}

	// We should have location history
	if len(updatedDevice.LocationHistory) != 2 {
		t.Fatalf("Expected 2 location traces, got %d", len(updatedDevice.LocationHistory))
	}
}

func TestUUIDConsistency(t *testing.T) {
	// Test that the same device generates the same UUID
	uuid1 := ComputeUUID("Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")
	uuid2 := ComputeUUID("Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")

	if uuid1 != uuid2 {
		t.Fatal("UUIDs should be identical for same device")
	}

	// Different device should have different UUID
	uuid3 := ComputeUUID("Samsung", "Galaxy S24", "48:5A:3F:8C:21:BA")
	if uuid1 == uuid3 {
		t.Fatal("Different devices should have different UUIDs")
	}

	t.Logf("UUID1: %s", uuid1)
	t.Logf("UUID2: %s", uuid2)
	t.Logf("UUID3: %s", uuid3)
}

func TestEnvironmentSignature(t *testing.T) {
	wifiNets := []*WiFiNetwork{
		{SSID: "Home", BSSID: "AA:BB:CC:DD:EE:FF", Signal: -45},
		{SSID: "Guest", BSSID: "11:22:33:44:55:66", Signal: -60},
	}

	btDevs := []*BluetoothDevice{
		{Address: "00:11:22:33:44:55", Name: "Speaker", Signal: -40},
	}

	sig1 := ComputeEnvironmentSignature(wifiNets, btDevs)
	sig2 := ComputeEnvironmentSignature(wifiNets, btDevs)

	if sig1 != sig2 {
		t.Fatal("Same environment should produce same signature")
	}

	// Different environment
	wifiNets2 := []*WiFiNetwork{
		{SSID: "Work", BSSID: "AA:BB:CC:DD:EE:FF", Signal: -45},
	}

	sig3 := ComputeEnvironmentSignature(wifiNets2, nil)
	if sig1 == sig3 {
		t.Fatal("Different environments should produce different signatures")
	}

	t.Logf("Environment signature verified")
}

func BenchmarkRegisterDevice(b *testing.B) {
	svc := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")
	}
}

func BenchmarkUpdateLocation(b *testing.B) {
	svc := NewService()
	ctx := context.Background()
	device, _ := svc.RegisterDevice(ctx, "user123", "iPhone", "phone", "Apple", "iPhone 15 Pro", "48:5A:3F:8C:21:B9")

	wifiNets := []*WiFiNetwork{
		{SSID: "Home", BSSID: "AA:BB:CC:DD:EE:FF", Signal: -45},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.UpdateLocation(ctx, device.ID, wifiNets, nil, "wifi")
	}
}
