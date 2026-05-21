package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"cadastre_ia/internal/sync"
	"cadastre_ia/internal/rtk"
	"cadastre_ia/internal/garmin"
)

// ============ SYNC ENDPOINTS (Phase 2A) ============

// InitSync initializes device for synchronization
// POST /api/v1/sync/init
func (h *Handler) InitSync(c *gin.Context) {
	var req struct {
		DeviceID    string `json:"device_id" binding:"required"`
		Fingerprint string `json:"fingerprint" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.syncEngine.InitializeDevice(c.Request.Context(), req.DeviceID, req.Fingerprint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize sync"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": req.DeviceID,
		"status":    "initialized",
		"timestamp": time.Now().Unix(),
	})
}

// GetSyncState retrieves device sync state
// GET /api/v1/sync/state
func (h *Handler) GetSyncState(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	state, err := h.syncEngine.GetSyncState(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sync state not found"})
		return
	}

	c.JSON(http.StatusOK, state)
}

// SyncUpload uploads changes from device
// POST /api/v1/sync/upload
func (h *Handler) SyncUpload(c *gin.Context) {
	var req struct {
		DeviceID string            `json:"device_id" binding:"required"`
		Changes  []sync.SyncChange `json:"changes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.syncEngine.SyncUpload(c.Request.Context(), req.DeviceID, req.Changes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync upload failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "synced",
		"changes_accepted": len(req.Changes),
		"timestamp":        time.Now().Unix(),
	})
}

// SyncDownload downloads changes for device
// GET /api/v1/sync/download
func (h *Handler) SyncDownload(c *gin.Context) {
	deviceID := c.Query("device_id")
	sinceStr := c.Query("since")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	var since int64 = 0
	if sinceStr != "" {
		// Parse timestamp
	}

	changes, err := h.syncEngine.SyncDownload(c.Request.Context(), deviceID, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync download failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"changes":   changes,
		"count":     len(changes),
		"timestamp": time.Now().Unix(),
	})
}

// ResolveSyncConflict resolves a conflict
// POST /api/v1/sync/resolve
func (h *Handler) ResolveSyncConflict(c *gin.Context) {
	var req struct {
		ParcelID  string `json:"parcel_id" binding:"required"`
		Strategy  string `json:"strategy" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.syncEngine.ResolveConflict(c.Request.Context(), req.ParcelID, req.Strategy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "conflict resolution failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSyncStatus gets overall sync status
// GET /api/v1/sync/status
func (h *Handler) GetSyncStatus(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	pending, _ := h.syncEngine.GetPendingChanges(c.Request.Context(), deviceID)
	conflicts, _ := h.syncEngine.DetectConflicts(c.Request.Context(), "")

	c.JSON(http.StatusOK, gin.H{
		"device_id":            deviceID,
		"pending_changes":      len(pending),
		"active_conflicts":     len(conflicts),
		"last_sync":            time.Now().Unix(),
		"sync_enabled":         true,
	})
}

// ============ RTK ENDPOINTS (Phase 2B) ============

// EnableRTK enables RTK for a device
// POST /api/v1/rtk/enable
func (h *Handler) EnableRTK(c *gin.Context) {
	var req struct {
		DeviceID           string `json:"device_id" binding:"required"`
		NTRIPUrl           string `json:"ntrip_url" binding:"required"`
		NTRIPUsername      string `json:"ntrip_username"`
		NTRIPPassword      string `json:"ntrip_password"`
		NTRIPMountPoint    string `json:"ntrip_mount_point"`
		ReferenceStationID string `json:"reference_station_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := rtk.RTKConfig{
		DeviceID:            req.DeviceID,
		RTKEnabled:          true,
		NTRIPUrl:            req.NTRIPUrl,
		NTRIPUsername:       req.NTRIPUsername,
		NTRIPPassword:       req.NTRIPPassword,
		NTRIPMountPoint:     req.NTRIPMountPoint,
		ReferenceStationID:  req.ReferenceStationID,
	}

	err := h.rtkService.EnableRTK(c.Request.Context(), req.DeviceID, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable RTK"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "rtk_enabled",
		"device_id": req.DeviceID,
		"timestamp": time.Now().Unix(),
	})
}

// DisableRTK disables RTK for a device
// POST /api/v1/rtk/disable
func (h *Handler) DisableRTK(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.rtkService.DisableRTK(c.Request.Context(), req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable RTK"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "rtk_disabled",
		"device_id": req.DeviceID,
	})
}

// GetRTKState gets current RTK state
// GET /api/v1/rtk/state
func (h *Handler) GetRTKState(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	state, err := h.rtkService.GetRTKState(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get RTK state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":  deviceID,
		"rtk_state":  state,
		"timestamp":  time.Now().Unix(),
	})
}

// SubmitRTKPosition submits RTK-corrected position
// POST /api/v1/rtk/submit-position
func (h *Handler) SubmitRTKPosition(c *gin.Context) {
	var req struct {
		DeviceID  string  `json:"device_id" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
		Height    float64 `json:"height"`
		Accuracy  float32 `json:"accuracy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gnssPos := rtk.GNSSPosition{
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Height:    req.Height,
		Accuracy:  float64(req.Accuracy),
	}

	correction, err := h.rtkService.CorrectPosition(c.Request.Context(), req.DeviceID, gnssPos)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "position correction failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":         req.DeviceID,
		"corrected_lat":     correction.Latitude,
		"corrected_lon":     correction.Longitude,
		"corrected_alt":     correction.Height,
		"accuracy_m":        correction.Accuracy,
		"is_fixed":          correction.IsFixed,
		"timestamp":         time.Now().Unix(),
	})
}

// ============ GARMIN ENDPOINTS (Phase 2B) ============

// PairGarmin pairs a Garmin device
// POST /api/v1/garmin/pair
func (h *Handler) PairGarmin(c *gin.Context) {
	var req struct {
		DeviceID         string `json:"device_id" binding:"required"`
		SerialNumber     string `json:"serial_number" binding:"required"`
		ConnectionMethod string `json:"connection_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pairing, err := h.garminService.PairGarmin(
		c.Request.Context(),
		req.DeviceID,
		req.SerialNumber,
		garmin.ConnectionMethod(req.ConnectionMethod),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to pair Garmin device"})
		return
	}

	c.JSON(http.StatusOK, pairing)
}

// DisconnectGarmin disconnects a Garmin device
// POST /api/v1/garmin/disconnect
func (h *Handler) DisconnectGarmin(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.garminService.DisconnectGarmin(c.Request.Context(), req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect Garmin device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "disconnected",
		"device_id": req.DeviceID,
	})
}

// GetGarminStatus gets Garmin device status
// GET /api/v1/garmin/status
func (h *Handler) GetGarminStatus(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	pairing, err := h.garminService.GetGarminStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Garmin device not paired"})
		return
	}

	c.JSON(http.StatusOK, pairing)
}

// SubmitGarminSensorData submits sensor data from Garmin
// POST /api/v1/garmin/sensors
func (h *Handler) SubmitGarminSensorData(c *gin.Context) {
	var req struct {
		DeviceID   string                 `json:"device_id" binding:"required"`
		SensorType string                 `json:"sensor_type" binding:"required"`
		RawData    map[string]interface{} `json:"raw_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reading, err := h.garminService.ReceiveSensorData(
		c.Request.Context(),
		req.DeviceID,
		garmin.SensorType(req.SensorType),
		req.RawData,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store sensor data"})
		return
	}

	c.JSON(http.StatusOK, reading)
}

// ============ Handler struct (with Phase 2 services) ============

// Handler is the main API handler (extended for Phase 2)
type Handler struct {
	// Existing fields
	db *sql.DB
	// Phase 2 services
	syncEngine     *sync.HierarchicalSyncEngine
	rtkService     *rtk.RTKService
	garminService  *garmin.GarminService
}
