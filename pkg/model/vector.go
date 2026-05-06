package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ObjectType defines the type of vector object
type ObjectType string

const (
	ObjectTypeParcel         ObjectType = "parcel"
	ObjectTypeBuilding       ObjectType = "building"
	ObjectTypeTree           ObjectType = "tree"
	ObjectTypeLandmark       ObjectType = "landmark"
	ObjectTypeRoute          ObjectType = "route"
	ObjectTypeSensor         ObjectType = "sensor"
	ObjectTypeStructure      ObjectType = "structure"
	ObjectTypeVegetation     ObjectType = "vegetation"
	ObjectTypeWaterFeature   ObjectType = "water"
	ObjectTypeCustom         ObjectType = "custom"
)

// CoordFrame defines coordinate system reference
type CoordFrame string

const (
	CoordFrameWGS84   CoordFrame = "WGS84"
	CoordFrameUTM     CoordFrame = "UTM"
	CoordFrameLocal   CoordFrame = "LOCAL"
	CoordFrameRelative CoordFrame = "RELATIVE"
)

// VectorObject is the core data model - encapsulates all sensor data + geometry
type VectorObject struct {
	// Identity
	ID              uuid.UUID                  `json:"id" db:"id"`
	Type            ObjectType                 `json:"type" db:"type"`
	Name            string                     `json:"name" db:"name"`
	Description     string                     `json:"description" db:"description"`

	// Geometry (GeoJSON-compatible)
	Geometry        *GeoJSONGeometry           `json:"geometry" db:"geometry"` // JSON blob
	CoordinateFrame CoordFrame                 `json:"coordinate_frame" db:"coord_frame"`
	Accuracy        float32                    `json:"accuracy_meters" db:"accuracy"`

	// Sensor Fusion Data
	SensorData      *SensorDataBundle          `json:"sensor_data" db:"sensor_data"` // JSON blob
	ExtractedAt     time.Time                  `json:"extracted_at" db:"extracted_at"`

	// Properties & Metadata
	Properties      map[string]interface{}     `json:"properties" db:"properties"` // JSONB
	Tags            []string                   `json:"tags" db:"tags"`             // TEXT[] in DB
	Owner           string                     `json:"owner" db:"owner"`
	Classification  string                     `json:"classification" db:"classification"` // e.g., "agricultural", "urban"

	// Rendering Hints
	RenderStyle     *RenderStyle               `json:"render_style" db:"render_style"`
	ArcadeSprite    *ArcadeSprite              `json:"arcade_sprite,omitempty" db:"-"`
	ThreeDModel     *ThreeDModel               `json:"3d_model,omitempty" db:"-"`

	// Versioning & Audit
	Version         int32                      `json:"version" db:"version"`
	CreatedAt       time.Time                  `json:"created_at" db:"created_at"`
	ModifiedAt      time.Time                  `json:"modified_at" db:"modified_at"`
	LastModifiedBy  string                     `json:"last_modified_by" db:"last_modified_by"`

	// Synchronization Metadata
	SyncID          string                     `json:"sync_id" db:"sync_id"`           // For P2P tracking
	LastSyncAt      time.Time                  `json:"last_sync_at" db:"last_sync_at"`
	IsDeleted       bool                       `json:"is_deleted" db:"is_deleted"`     // Soft delete
	DeletedAt       *time.Time                 `json:"deleted_at,omitempty" db:"deleted_at"`
}

// GeoJSONGeometry represents geometry in GeoJSON format
type GeoJSONGeometry struct {
	Type        string          `json:"type"` // Point, LineString, Polygon, MultiPolygon, etc.
	Coordinates json.RawMessage `json:"coordinates"` // Can be various nested arrays
	BBox        [4]float64      `json:"bbox,omitempty"` // [minLon, minLat, maxLon, maxLat]
}

// SensorDataBundle aggregates all available sensor data
type SensorDataBundle struct {
	GNSS            *GNSSData              `json:"gnss,omitempty"`
	IMU             *IMUData               `json:"imu,omitempty"`
	Photogrammetry  *PhotogramData         `json:"photogrammetry,omitempty"`
	DroneData       *DroneData             `json:"drone,omitempty"`
	Camera          *CameraData            `json:"camera,omitempty"`
	LiDAR           *LiDARData             `json:"lidar,omitempty"`
	CustomSensors   map[string]interface{} `json:"custom,omitempty"`
}

// GNSSData - Global Navigation Satellite System (GPS)
type GNSSData struct {
	Position        [3]float64 `json:"position"`        // [latitude, longitude, altitude]
	PositionAccuracy float32   `json:"position_accuracy"` // meters
	SatelliteCount  int32      `json:"satellite_count"`
	HDOP            float32    `json:"hdop"`             // Horizontal Dilution of Precision
	VDOP            float32    `json:"vdop"`             // Vertical DOP
	Timestamp       time.Time  `json:"timestamp"`
	FixQuality      string     `json:"fix_quality"`      // "float", "rtk_fixed", "dgps", etc.
	Heading         float32    `json:"heading"`          // degrees (0-360)
	Speed           float32    `json:"speed"`            // m/s
}

// IMUData - Inertial Measurement Unit
type IMUData struct {
	Orientation     [3]float32 `json:"orientation"`      // [pitch, roll, yaw] in degrees
	Acceleration    [3]float32 `json:"acceleration"`     // [x, y, z] m/s²
	AngularVelocity [3]float32 `json:"angular_velocity"` // [x, y, z] rad/s
	MagneticField   [3]float32 `json:"magnetic_field"`   // [x, y, z] Tesla
	Temperature     float32    `json:"temperature"`      // Celsius
	Timestamp       time.Time  `json:"timestamp"`
}

// PhotogramData - 3D reconstruction from images
type PhotogramData struct {
	MeshURL         string     `json:"mesh_url"`
	TextureURL      string     `json:"texture_url"`
	PointCloudURL   string     `json:"point_cloud_url"`
	BoundingBox     *BBox3D    `json:"bounding_box"`
	Scale           float32    `json:"scale"`            // Scale factor
	Confidence      float32    `json:"confidence"`       // 0-1 quality score
	PointCount      int32      `json:"point_count"`
	TriangleCount   int32      `json:"triangle_count"`
	ProcessedAt     time.Time  `json:"processed_at"`
	Algorithm       string     `json:"algorithm"`        // e.g., "structure_from_motion"
}

// DroneData - Aerial survey data
type DroneData struct {
	OrthoGeoTIFF    []byte     `json:"ortho_geotiff,omitempty"` // Binary GeoTIFF
	PointCloud      []XYZPoint `json:"point_cloud,omitempty"`
	FlightAltitude  float32    `json:"flight_altitude"`
	DroneType       string     `json:"drone_type"`
	CapturedAt      time.Time  `json:"captured_at"`
	Resolution      float32    `json:"resolution"`       // cm per pixel
	Coverage        float32    `json:"coverage"`         // percentage
}

// CameraData - Visual sensor data
type CameraData struct {
	ImageURL        string           `json:"image_url"`
	ExifData        map[string]string `json:"exif_data"`
	FeaturePoints   []FeaturePoint   `json:"feature_points"`
	ImageSize       [2]int32         `json:"image_size"`  // [width, height]
	FocalLength     float32          `json:"focal_length"`
	CameraMake      string           `json:"camera_make"`
	CameraModel     string           `json:"camera_model"`
	CapturedAt      time.Time        `json:"captured_at"`
}

// LiDARData - Light Detection and Ranging
type LiDARData struct {
	PointCloud      []XYZPoint `json:"point_cloud"`
	Intensity       []float32  `json:"intensity"`        // Per-point intensity
	Resolution      float32    `json:"resolution"`       // cm per point
	ScanLines       int32      `json:"scan_lines"`
	PointsPerLine   int32      `json:"points_per_line"`
	CapturedAt      time.Time  `json:"captured_at"`
	ScannerType     string     `json:"scanner_type"`
}

// XYZPoint represents a 3D point
type XYZPoint struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// BBox3D represents a 3D bounding box
type BBox3D struct {
	Min [3]float32 `json:"min"` // [x, y, z] minimum
	Max [3]float32 `json:"max"` // [x, y, z] maximum
}

// FeaturePoint represents a detected image feature
type FeaturePoint struct {
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Descriptor []byte  `json:"descriptor,omitempty"` // SIFT/SURF/ORB descriptor
	Strength  float32 `json:"strength"`              // 0-1 confidence
}

// RenderStyle defines how object should be rendered
type RenderStyle struct {
	ArcadeColor     uint8   `json:"arcade_color"`
	VectorFill      string  `json:"vector_fill"`       // CSS color
	VectorStroke    string  `json:"vector_stroke"`
	VectorOpacity   float32 `json:"vector_opacity"`
	AnimationFrames int32   `json:"animation_frames"`
	Priority        int32   `json:"priority"`          // Z-order
	Visibility      string  `json:"visibility"`        // "visible", "hidden", "dim"
}

// ArcadeSprite represents rendered arcade sprite
type ArcadeSprite struct {
	Data            []byte      `json:"data"`             // Raw sprite bitmap
	Width           int32       `json:"width"`
	Height          int32       `json:"height"`
	PaletteIndex    uint8       `json:"palette_index"`
	Collision       [][]bool    `json:"collision"`        // Collision map
	Frames          int32       `json:"frames"`           // For animated sprites
	FrameDuration   int32       `json:"frame_duration"`   // ms per frame
}

// ThreeDModel represents 3D model data
type ThreeDModel struct {
	GLTFBinary      []byte     `json:"gltf_binary,omitempty"` // Embedded glTF binary
	MeshURL         string     `json:"mesh_url"`
	TextureURL      string     `json:"texture_url"`
	Material        *PBRMaterial `json:"material"`
	Scale           [3]float32 `json:"scale"`
	Rotation        [4]float32 `json:"rotation"`           // Quaternion [x, y, z, w]
}

// PBRMaterial - Physically Based Rendering material
type PBRMaterial struct {
	BaseColor       [4]float32 `json:"base_color"`        // RGBA
	Metallic        float32    `json:"metallic"`          // 0-1
	Roughness       float32    `json:"roughness"`         // 0-1
	NormalMapURL    string     `json:"normal_map_url"`
	MetallicMapURL  string     `json:"metallic_map_url"`
	RoughnessMapURL string     `json:"roughness_map_url"`
}

// New creates a new VectorObject with defaults
func New(objType ObjectType, name string) *VectorObject {
	return &VectorObject{
		ID:              uuid.New(),
		Type:            objType,
		Name:            name,
		CoordinateFrame: CoordFrameWGS84,
		Version:         1,
		CreatedAt:       time.Now(),
		ModifiedAt:      time.Now(),
		IsDeleted:       false,
		SyncID:          uuid.New().String(),
	}
}

// WithGeometry sets the geometry
func (vo *VectorObject) WithGeometry(geom *GeoJSONGeometry) *VectorObject {
	vo.Geometry = geom
	vo.ModifiedAt = time.Now()
	return vo
}

// WithSensorData sets the sensor data
func (vo *VectorObject) WithSensorData(sd *SensorDataBundle) *VectorObject {
	vo.SensorData = sd
	vo.ExtractedAt = time.Now()
	vo.ModifiedAt = time.Now()
	return vo
}

// IncrementVersion bumps the version number
func (vo *VectorObject) IncrementVersion(modifiedBy string) {
	vo.Version++
	vo.ModifiedAt = time.Now()
	vo.LastModifiedBy = modifiedBy
}

// SoftDelete marks the object as deleted
func (vo *VectorObject) SoftDelete() {
	now := time.Now()
	vo.IsDeleted = true
	vo.DeletedAt = &now
	vo.ModifiedAt = now
}

// MarshalJSON custom JSON marshaling
func (vo *VectorObject) MarshalJSON() ([]byte, error) {
	type Alias VectorObject
	return json.Marshal(&struct {
		*Alias
		CreatedAt  string `json:"created_at"`
		ModifiedAt string `json:"modified_at"`
	}{
		Alias:      (*Alias)(vo),
		CreatedAt:  vo.CreatedAt.Format(time.RFC3339Nano),
		ModifiedAt: vo.ModifiedAt.Format(time.RFC3339Nano),
	})
}
