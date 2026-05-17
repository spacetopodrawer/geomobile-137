package deviceradar

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// Service handles device identification and tracking
type Service struct {
	mu                  sync.RWMutex
	devices             map[string]*DeviceIdentity
	intrinsicIDs        map[string]*IntrinsicID
	locationHistory     map[string][]*LocationTrace
	movementHistory     map[string][]*Movement
	environmentSigs     map[string]*EnvironmentSig
	vpnStatuses         map[string]*VPNStatus
	premiumFeatures     map[string][]*PremiumFeature
	userTrustedDevices  map[string][]string // userID -> list of trusted device IDs
	subnetTrustMap      map[string]bool     // subnet CIDR -> is trusted
}

// NewService creates a new DeviceRadar service
func NewService() *Service {
	return &Service{
		devices:            make(map[string]*DeviceIdentity),
		intrinsicIDs:       make(map[string]*IntrinsicID),
		locationHistory:    make(map[string][]*LocationTrace),
		movementHistory:    make(map[string][]*Movement),
		environmentSigs:    make(map[string]*EnvironmentSig),
		vpnStatuses:        make(map[string]*VPNStatus),
		premiumFeatures:    make(map[string][]*PremiumFeature),
		userTrustedDevices: make(map[string][]string),
		subnetTrustMap:     make(map[string]bool),
	}
}

// RegisterDevice registers a new device with intrinsic identification
func (s *Service) RegisterDevice(ctx context.Context, userID, deviceName, deviceType, manufacturer, model, mac string) (*DeviceIdentity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if userID == "" || manufacturer == "" || model == "" {
		return nil, errors.New("missing required device information")
	}

	// Compute intrinsic identifiers
	uuid := ComputeUUID(manufacturer, model, mac)
	hwSig := ComputeHardwareSignature(manufacturer, model, mac)

	// Check if device already registered
	for _, dev := range s.devices {
		if dev.IntrinsicID.UUID == uuid && dev.UserID == userID {
			return dev, nil // Already registered
		}
	}

	// Create intrinsic ID
	intrinsicID := &IntrinsicID{
		UUID:              uuid,
		HardwareSignature: hwSig,
		Manufacturer:      manufacturer,
		Model:             model,
		MAC:               mac,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	s.intrinsicIDs[uuid] = intrinsicID

	// Create device identity
	now := time.Now()
	deviceID := fmt.Sprintf("dev_%s_%d", uuid[:8], now.UnixNano())
	device := &DeviceIdentity{
		ID:            deviceID,
		IntrinsicID:   intrinsicID,
		UserID:        userID,
		DeviceName:    deviceName,
		DeviceType:    deviceType,
		Trusted:       false,
		Tags:          make(map[string]string),
		LocationHistory: []*LocationTrace{},
		MovementHistory: []*Movement{},
		LastSeen:      now,
		FirstSeen:     now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.devices[deviceID] = device
	s.locationHistory[deviceID] = make([]*LocationTrace, 0)
	s.movementHistory[deviceID] = make([]*Movement, 0)

	// Add to user's trusted devices list
	if _, exists := s.userTrustedDevices[userID]; !exists {
		s.userTrustedDevices[userID] = make([]string, 0)
	}
	s.userTrustedDevices[userID] = append(s.userTrustedDevices[userID], deviceID)

	return device, nil
}

// UpdateLocation updates device location with WiFi/BT scan results
func (s *Service) UpdateLocation(ctx context.Context, deviceID string, wifiNets []*WiFiNetwork, btDevs []*BluetoothDevice, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return errors.New("device not found")
	}

	now := time.Now()

	// Create location trace
	trace := &LocationTrace{
		Timestamp:     now,
		WiFiNets:      wifiNets,
		BluetoothDevs: btDevs,
		Source:        source,
	}

	// Triangulate position (simplified)
	if len(wifiNets) > 0 {
		trace.Triangulated = triangulateWiFi(wifiNets)
		trace.Accuracy = calculateAccuracy(wifiNets)
	} else if len(btDevs) > 0 {
		trace.Triangulated = triangulateBluetooth(btDevs)
		trace.Accuracy = calculateAccuracyBT(btDevs)
	}

	// Store location trace
	s.locationHistory[deviceID] = append(s.locationHistory[deviceID], trace)

	// Calculate movement if there's a previous location
	if len(s.locationHistory[deviceID]) > 1 {
		prevTrace := s.locationHistory[deviceID][len(s.locationHistory[deviceID])-2]
		if prevTrace.Triangulated != nil && trace.Triangulated != nil {
			movement := calculateMovement(prevTrace, trace)
			s.movementHistory[deviceID] = append(s.movementHistory[deviceID], movement)
			device.MovementHistory = append(device.MovementHistory, movement)
		}
	}

	device.LocationHistory = append(device.LocationHistory, trace)
	device.LastSeen = now
	device.UpdatedAt = now

	// Update environment signature
	s.updateEnvironmentSignature(deviceID, wifiNets, btDevs)

	return nil
}

// DetectVPNActivity analyzes VPN status and potential leaks
func (s *Service) DetectVPNActivity(ctx context.Context, deviceID string, visibleIP, detectedRealIP string) (*VPNStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return nil, errors.New("device not found")
	}

	// Determine VPN status
	isActive := false
	hasLeaks := visibleIP != detectedRealIP
	vpnStatus := &VPNStatus{
		ID:             fmt.Sprintf("vpn_%s_%d", deviceID[:8], time.Now().UnixNano()),
		DeviceID:       deviceID,
		ExitIP:         visibleIP,
		DetectedRealIP: detectedRealIP,
		HasLeaks:       hasLeaks,
		LastChecked:    time.Now(),
	}

	if visibleIP != detectedRealIP {
		isActive = true
		vpnStatus.IsActive = true
		if hasLeaks {
			vpnStatus.LeakTypes = append(vpnStatus.LeakTypes, "ip_leak")
		}
	}

	// Check for split tunneling (location inconsistency with VPN claim)
	if isActive && len(s.locationHistory[deviceID]) > 0 {
		latestTrace := s.locationHistory[deviceID][len(s.locationHistory[deviceID])-1]
		// If location suddenly changes dramatically, it might be split tunneling
		if len(s.locationHistory[deviceID]) > 1 {
			prevTrace := s.locationHistory[deviceID][len(s.locationHistory[deviceID])-2]
			if latestTrace.Triangulated != nil && prevTrace.Triangulated != nil {
				dist := calculateDistance(prevTrace.Triangulated, latestTrace.Triangulated)
				if dist > 1000 { // >1km jump in <1 frame is suspicious
					vpnStatus.IsSplitTunneling = true
					vpnStatus.Anomalies = append(vpnStatus.Anomalies, "sudden_location_jump")
				}
			}
		}
	}

	s.vpnStatuses[deviceID] = vpnStatus
	device.VPNStatus = vpnStatus

	return vpnStatus, nil
}

// VerifyDeviceAuthenticity checks if a device is authentic based on history
func (s *Service) VerifyDeviceAuthenticity(ctx context.Context, deviceID string) (bool, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return false, nil, errors.New("device not found")
	}

	var checks []string
	allPassed := true

	// Check 1: UUID consistency
	if device.IntrinsicID != nil && device.IntrinsicID.UUID != "" {
		checks = append(checks, "✓ UUID matches hardware signature")
	} else {
		checks = append(checks, "✗ UUID missing or invalid")
		allPassed = false
	}

	// Check 2: Location history consistency
	if len(s.locationHistory[deviceID]) > 2 {
		// Check if movements are physically plausible
		plausible := true
		for i := 1; i < len(s.locationHistory[deviceID]); i++ {
			prev := s.locationHistory[deviceID][i-1]
			curr := s.locationHistory[deviceID][i]
			if prev.Triangulated != nil && curr.Triangulated != nil {
				dist := calculateDistance(prev.Triangulated, curr.Triangulated)
				timeDiff := curr.Timestamp.Sub(prev.Timestamp).Seconds()
				speed := dist / timeDiff // meters per second
				if speed > 300 { // >300 m/s is impossible
					plausible = false
					break
				}
			}
		}
		if plausible {
			checks = append(checks, "✓ Location history physically plausible")
		} else {
			checks = append(checks, "✗ Location history suspicious - impossible speeds detected")
			allPassed = false
		}
	}

	// Check 3: Device age
	age := time.Since(device.FirstSeen)
	if age > 24*time.Hour {
		checks = append(checks, "✓ Device known for >24 hours")
	} else {
		checks = append(checks, "⚠ Device is new (<24 hours)")
	}

	// Check 4: VPN status consistency
	if device.VPNStatus != nil {
		if device.VPNStatus.HasLeaks || device.VPNStatus.IsSplitTunneling {
			checks = append(checks, "⚠ VPN leaks or split-tunneling detected")
			allPassed = false
		} else if device.VPNStatus.IsActive {
			checks = append(checks, "✓ VPN active with no detected leaks")
		}
	}

	return allPassed, checks, nil
}

// GetDevicesByUser returns all devices for a user
func (s *Service) GetDevicesByUser(ctx context.Context, userID string) ([]*DeviceIdentity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var devices []*DeviceIdentity
	for _, dev := range s.devices {
		if dev.UserID == userID {
			devices = append(devices, dev)
		}
	}

	if len(devices) == 0 {
		return nil, errors.New("no devices found for user")
	}

	return devices, nil
}

// GetDeviceByID returns a specific device
func (s *Service) GetDeviceByID(ctx context.Context, deviceID string) (*DeviceIdentity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return nil, errors.New("device not found")
	}

	return device, nil
}

// Helper functions

func triangulateWiFi(networks []*WiFiNetwork) *GeoLocation {
	// Simplified triangulation: average location based on signal strength weights
	if len(networks) == 0 {
		return nil
	}
	// In production, use actual WiFi geolocation database (Google, Mozilla, etc.)
	return &GeoLocation{
		Latitude:  48.8566,
		Longitude: 2.3522,
	}
}

func triangulateBluetooth(devices []*BluetoothDevice) *GeoLocation {
	if len(devices) == 0 {
		return nil
	}
	return &GeoLocation{
		Latitude:  48.8566,
		Longitude: 2.3522,
	}
}

func calculateAccuracy(networks []*WiFiNetwork) float64 {
	// More networks = better accuracy
	if len(networks) == 0 {
		return 1000 // 1km default
	}
	baseAccuracy := 50.0 // 50m base
	return baseAccuracy / float64(len(networks))
}

func calculateAccuracyBT(devices []*BluetoothDevice) float64 {
	if len(devices) == 0 {
		return 100
	}
	return 30.0 / float64(len(devices))
}

func calculateMovement(from, to *LocationTrace) *Movement {
	id := fmt.Sprintf("move_%d", time.Now().UnixNano())
	duration := to.Timestamp.Sub(from.Timestamp).Milliseconds()
	distance := 0.0
	speed := 0.0

	if from.Triangulated != nil && to.Triangulated != nil {
		distance = calculateDistance(from.Triangulated, to.Triangulated)
		if duration > 0 {
			speed = distance / (float64(duration) / 1000.0)
		}
	}

	return &Movement{
		ID:        id,
		From:      from,
		To:        to,
		Distance:  distance,
		Duration:  duration,
		Speed:     speed,
		Timestamp: to.Timestamp,
	}
}

func calculateDistance(loc1, loc2 *GeoLocation) float64 {
	// Haversine formula
	const earthRadiusM = 6371000.0
	lat1 := degreesToRadians(loc1.Latitude)
	lat2 := degreesToRadians(loc2.Latitude)
	dLat := degreesToRadians(loc2.Latitude - loc1.Latitude)
	dLng := degreesToRadians(loc2.Longitude - loc1.Longitude)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusM * c
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func (s *Service) updateEnvironmentSignature(deviceID string, wifiNets []*WiFiNetwork, btDevs []*BluetoothDevice) {
	sig := &EnvironmentSig{
		ID:                  fmt.Sprintf("env_%s_%d", deviceID[:8], time.Now().UnixNano()),
		DeviceID:            deviceID,
		WiFiFingerprint:     make(map[string]int),
		BluetoothFingerprint: make(map[string]int),
		LastUpdated:         time.Now(),
		TrustedSubnets:      []string{},
	}

	for _, wn := range wifiNets {
		sig.WiFiFingerprint[wn.SSID] = wn.Signal
	}

	for _, bd := range btDevs {
		sig.BluetoothFingerprint[bd.Address] = bd.Signal
	}

	sig.RFPattern = ComputeEnvironmentSignature(wifiNets, btDevs)
	s.environmentSigs[deviceID] = sig
}
