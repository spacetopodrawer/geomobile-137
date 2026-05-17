package deviceradar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// IntrinsicID represents the hardware-level unique identifier
type IntrinsicID struct {
	UUID              string            `json:"uuid"`              // MAC + Manufacturer + Model hash
	HardwareSignature string            `json:"hardware_signature"` // Full hardware fingerprint
	Manufacturer      string            `json:"manufacturer"`
	Model             string            `json:"model"`
	MAC               string            `json:"mac"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// DeviceIdentity represents a device with intrinsic identification
type DeviceIdentity struct {
	ID                    string            `json:"id"`
	IntrinsicID           *IntrinsicID      `json:"intrinsic_id"`
	UserID                string            `json:"user_id"`
	DeviceName            string            `json:"device_name"`
	DeviceType            string            `json:"device_type"` // "phone", "tablet", "laptop", etc.
	Trusted               bool              `json:"trusted"`
	Tags                  map[string]string `json:"tags"`
	LocationHistory       []*LocationTrace  `json:"location_history"`
	MovementHistory       []*Movement       `json:"movement_history"`
	EnvironmentSignature  *EnvironmentSig   `json:"environment_signature"`
	VPNStatus             *VPNStatus        `json:"vpn_status"`
	LastSeen              time.Time         `json:"last_seen"`
	FirstSeen             time.Time         `json:"first_seen"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// LocationTrace represents WiFi/BT location at a point in time
type LocationTrace struct {
	Timestamp     time.Time           `json:"timestamp"`
	WiFiNets      []*WiFiNetwork      `json:"wifi_nets"`
	BluetoothDevs []*BluetoothDevice  `json:"bluetooth_devs"`
	Triangulated  *GeoLocation        `json:"triangulated"`
	Accuracy      float64             `json:"accuracy"` // meters
	Source        string              `json:"source"`   // "wifi", "bt", "gps", etc.
}

// WiFiNetwork represents a detected WiFi network
type WiFiNetwork struct {
	SSID      string  `json:"ssid"`
	BSSID     string  `json:"bssid"`
	Signal    int     `json:"signal"` // dBm (-100 to 0)
	Channel   int     `json:"channel"`
	Bandwidth string  `json:"bandwidth"`
	Security  string  `json:"security"`
	Distance  float64 `json:"distance"` // estimated meters
}

// BluetoothDevice represents a detected Bluetooth device
type BluetoothDevice struct {
	Address   string  `json:"address"`
	Name      string  `json:"name"`
	Signal    int     `json:"signal"` // dBm (-100 to 0)
	Distance  float64 `json:"distance"` // estimated meters
	Type      string  `json:"type"`     // "audio", "wearable", "phone", etc.
}

// GeoLocation represents a geographic location
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// Movement represents physical movement between locations
type Movement struct {
	ID        string       `json:"id"`
	From      *LocationTrace `json:"from"`
	To        *LocationTrace `json:"to"`
	Distance  float64      `json:"distance"` // meters
	Duration  int64        `json:"duration"` // milliseconds
	Speed     float64      `json:"speed"`    // m/s
	Timestamp time.Time    `json:"timestamp"`
}

// EnvironmentSig represents RF/physical environment characteristics
type EnvironmentSig struct {
	ID                    string                 `json:"id"`
	DeviceID              string                 `json:"device_id"`
	WiFiFingerprint       map[string]int         `json:"wifi_fingerprint"` // SSID -> signal strength
	BluetoothFingerprint  map[string]int         `json:"bluetooth_fingerprint"` // address -> signal strength
	RFPattern             string                 `json:"rf_pattern"` // hash of RF environment
	LastUpdated           time.Time              `json:"last_updated"`
	HomeEnvironment       bool                   `json:"home_environment"` // is this the home network?
	TrustedSubnets        []string               `json:"trusted_subnets"`
}

// VPNStatus represents VPN connection status and analysis
type VPNStatus struct {
	ID                  string    `json:"id"`
	DeviceID            string    `json:"device_id"`
	IsActive            bool      `json:"is_active"`
	ExitIP              string    `json:"exit_ip"`
	ExitCountry         string    `json:"exit_country"`
	ExitProvider        string    `json:"exit_provider"`
	DetectedRealIP      string    `json:"detected_real_ip"`
	HasLeaks            bool      `json:"has_leaks"`
	LeakTypes           []string  `json:"leak_types"` // "dns", "webrtc", "ipv6", etc.
	IsSplitTunneling    bool      `json:"is_split_tunneling"`
	SuspiciousActivity  bool      `json:"suspicious_activity"`
	LastChecked         time.Time `json:"last_checked"`
	Anomalies           []string  `json:"anomalies"`
}

// PremiumFeature represents premium features available for user
type PremiumFeature struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	FeatureName string    `json:"feature_name"`
	Enabled     bool      `json:"enabled"`
	ExpiresAt   time.Time `json:"expires_at"`
	Tier        string    `json:"tier"` // "basic", "pro", "enterprise"
}

// Helper functions

// ComputeUUID generates a unique UUID from device characteristics
func ComputeUUID(manufacturer, model, mac string) string {
	data := fmt.Sprintf("%s:%s:%s", manufacturer, model, mac)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// ComputeHardwareSignature generates a full hardware signature
func ComputeHardwareSignature(manufacturer, model, mac string, additionalData ...string) string {
	data := fmt.Sprintf("%s:%s:%s", manufacturer, model, mac)
	for _, extra := range additionalData {
		data += ":" + extra
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeEnvironmentSignature generates RF pattern hash
func ComputeEnvironmentSignature(wifiNets []*WiFiNetwork, btDevs []*BluetoothDevice) string {
	data := ""
	for _, wn := range wifiNets {
		data += fmt.Sprintf("%s:%d;", wn.BSSID, wn.Signal)
	}
	for _, bd := range btDevs {
		data += fmt.Sprintf("%s:%d;", bd.Address, bd.Signal)
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
