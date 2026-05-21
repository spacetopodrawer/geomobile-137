package api

import (
	"net/http"

	"geomobile/pkg/deviceradar"
	"github.com/gin-gonic/gin"
)

// DeviceRadarAPI represents the DeviceRadar API handlers
type DeviceRadarAPI struct {
	service *deviceradar.Service
}

// NewDeviceRadarAPI creates a new DeviceRadar API instance
func NewDeviceRadarAPI(service *deviceradar.Service) *DeviceRadarAPI {
	return &DeviceRadarAPI{
		service: service,
	}
}

// RegisterDeviceRequest represents a device registration request
type RegisterDeviceRequest struct {
	DeviceName   string `json:"device_name" binding:"required"`
	DeviceType   string `json:"device_type" binding:"required"` // "phone", "tablet", "laptop"
	Manufacturer string `json:"manufacturer" binding:"required"`
	Model        string `json:"model" binding:"required"`
	MAC          string `json:"mac" binding:"required"`
}

// UpdateLocationRequest represents a location update request
type UpdateLocationRequest struct {
	WiFiNetworks    []*WifiNetworkRequest    `json:"wifi_networks"`
	BluetoothDevices []*BluetoothDeviceRequest `json:"bluetooth_devices"`
	Source          string                  `json:"source"` // "wifi", "bt", "gps"
}

// WifiNetworkRequest represents a WiFi network discovery
type WifiNetworkRequest struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Bandwidth string `json:"bandwidth"`
	Security  string `json:"security"`
}

// BluetoothDeviceRequest represents a Bluetooth device discovery
type BluetoothDeviceRequest struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Signal  int    `json:"signal"` // dBm
	Type    string `json:"type"`   // "audio", "wearable", "phone"
}

// VPNCheckRequest represents a VPN status check request
type VPNCheckRequest struct {
	VisibleIP      string `json:"visible_ip" binding:"required"`
	DetectedRealIP string `json:"detected_real_ip" binding:"required"`
}

// RegisterDevice godoc
// @Summary Register a new device with intrinsic identification
// @Description Registers a device and generates unique hardware-based identifiers
// @Tags deviceradar
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param request body RegisterDeviceRequest true "Device registration data"
// @Success 201 {object} deviceradar.DeviceIdentity
// @Failure 400 {object} map[string]string
// @Router /api/v1/deviceradar/register [post]
func (api *DeviceRadarAPI) RegisterDevice(c *gin.Context) {
	userID := c.GetString("user_id") // From auth middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := api.service.RegisterDevice(
		c.Request.Context(),
		userID,
		req.DeviceName,
		req.DeviceType,
		req.Manufacturer,
		req.Model,
		req.MAC,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, device)
}

// UpdateLocation godoc
// @Summary Update device location with WiFi/Bluetooth scans
// @Description Updates device location and detects movement patterns
// @Tags deviceradar
// @Accept json
// @Produce json
// @Param deviceID path string true "Device ID"
// @Param request body UpdateLocationRequest true "Location data"
// @Success 200 {object} deviceradar.LocationTrace
// @Failure 400 {object} map[string]string
// @Router /api/v1/deviceradar/devices/{deviceID}/location [post]
func (api *DeviceRadarAPI) UpdateLocation(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing deviceID"})
		return
	}

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert request WiFi networks
	wifiNets := make([]*deviceradar.WiFiNetwork, len(req.WiFiNetworks))
	for i, w := range req.WiFiNetworks {
		wifiNets[i] = &deviceradar.WiFiNetwork{
			SSID:      w.SSID,
			BSSID:     w.BSSID,
			Signal:    w.Signal,
			Channel:   w.Channel,
			Bandwidth: w.Bandwidth,
			Security:  w.Security,
		}
	}

	// Convert request Bluetooth devices
	btDevs := make([]*deviceradar.BluetoothDevice, len(req.BluetoothDevices))
	for i, b := range req.BluetoothDevices {
		btDevs[i] = &deviceradar.BluetoothDevice{
			Address: b.Address,
			Name:    b.Name,
			Signal:  b.Signal,
			Type:    b.Type,
		}
	}

	err := api.service.UpdateLocation(c.Request.Context(), deviceID, wifiNets, btDevs, req.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, _ := api.service.GetDeviceByID(c.Request.Context(), deviceID)
	if device != nil && len(device.LocationHistory) > 0 {
		c.JSON(http.StatusOK, device.LocationHistory[len(device.LocationHistory)-1])
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "location updated"})
	}
}

// DetectVPNActivity godoc
// @Summary Detect VPN activity and potential leaks
// @Description Analyzes VPN status, leaks, and split-tunneling
// @Tags deviceradar
// @Accept json
// @Produce json
// @Param deviceID path string true "Device ID"
// @Param request body VPNCheckRequest true "VPN status data"
// @Success 200 {object} deviceradar.VPNStatus
// @Failure 400 {object} map[string]string
// @Router /api/v1/deviceradar/devices/{deviceID}/vpn-check [post]
func (api *DeviceRadarAPI) DetectVPNActivity(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing deviceID"})
		return
	}

	var req VPNCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vpnStatus, err := api.service.DetectVPNActivity(c.Request.Context(), deviceID, req.VisibleIP, req.DetectedRealIP)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, vpnStatus)
}

// VerifyDeviceAuthenticity godoc
// @Summary Verify if a device is authentic based on history
// @Description Performs cryptographic and behavioral verification
// @Tags deviceradar
// @Accept json
// @Produce json
// @Param deviceID path string true "Device ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/deviceradar/devices/{deviceID}/verify [get]
func (api *DeviceRadarAPI) VerifyDeviceAuthenticity(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing deviceID"})
		return
	}

	authentic, checks, err := api.service.VerifyDeviceAuthenticity(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authentic": authentic,
		"checks":    checks,
	})
}

// GetDevice godoc
// @Summary Get device details
// @Description Retrieve device information including location and movement history
// @Tags deviceradar
// @Accept json
// @Produce json
// @Param deviceID path string true "Device ID"
// @Success 200 {object} deviceradar.DeviceIdentity
// @Failure 404 {object} map[string]string
// @Router /api/v1/deviceradar/devices/{deviceID} [get]
func (api *DeviceRadarAPI) GetDevice(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing deviceID"})
		return
	}

	device, err := api.service.GetDeviceByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// GetUserDevices godoc
// @Summary Get all devices for a user
// @Description List all registered devices for the authenticated user
// @Tags deviceradar
// @Accept json
// @Produce json
// @Success 200 {object} []deviceradar.DeviceIdentity
// @Failure 400 {object} map[string]string
// @Router /api/v1/deviceradar/devices [get]
func (api *DeviceRadarAPI) GetUserDevices(c *gin.Context) {
	userID := c.GetString("user_id") // From auth middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	devices, err := api.service.GetDevicesByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
}

// TagDevice adds or updates tags on a device (Premium feature)
// @Summary Add tags to a device
// @Description Add identifying tags to help distinguish devices
// @Tags deviceradar
// @Accept json
// @Produce json
// @Param deviceID path string true "Device ID"
// @Param tags body map[string]string true "Tags to add"
// @Success 200 {object} deviceradar.DeviceIdentity
// @Failure 400 {object} map[string]string
// @Router /api/v1/deviceradar/devices/{deviceID}/tags [post]
func (api *DeviceRadarAPI) TagDevice(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing deviceID"})
		return
	}

	var tags map[string]string
	if err := c.ShouldBindJSON(&tags); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := api.service.GetDeviceByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Merge new tags
	for k, v := range tags {
		device.Tags[k] = v
	}

	c.JSON(http.StatusOK, device)
}

// HealthCheck returns DeviceRadar health status
func (api *DeviceRadarAPI) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"service": "deviceradar",
		"version": "1.0.0",
		"features": []string{
			"device_identification",
			"location_tracking",
			"movement_detection",
			"vpn_detection",
			"authenticity_verification",
			"premium_features",
		},
	})
}
