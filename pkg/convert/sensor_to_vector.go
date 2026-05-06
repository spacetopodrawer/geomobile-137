package convert

import (
	"encoding/json"
	"fmt"
	"time"

	"cadastreia/pkg/model"
	"github.com/google/uuid"
)

// SensorToVectorConverter converts raw sensor data to VectorObject
type SensorToVectorConverter struct {
	DefaultCoordFrame model.CoordFrame
	DefaultOwner      string
}

// NewSensorToVectorConverter creates a new converter
func NewSensorToVectorConverter(owner string) *SensorToVectorConverter {
	return &SensorToVectorConverter{
		DefaultCoordFrame: model.CoordFrameWGS84,
		DefaultOwner:      owner,
	}
}

// FromGNSS creates a VectorObject from GNSS data (point object)
func (c *SensorToVectorConverter) FromGNSS(gnssData *model.GNSSData, name string) *model.VectorObject {
	if gnssData == nil {
		return nil
	}

	vo := model.New(model.ObjectTypeParcel, name)
	vo.Owner = c.DefaultOwner
	vo.Classification = "gnss_point"

	// Create Point geometry
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "Point",
		Coordinates: json.RawMessage(fmt.Sprintf(
			`[%.6f, %.6f, %.2f]`,
			gnssData.Position[1], // longitude
			gnssData.Position[0], // latitude
			gnssData.Position[2], // altitude
		)),
		BBox: [4]float64{
			gnssData.Position[1] - 0.0001, // minLon
			gnssData.Position[0] - 0.0001, // minLat
			gnssData.Position[1] + 0.0001, // maxLon
			gnssData.Position[0] + 0.0001, // maxLat
		},
	}

	vo.Accuracy = gnssData.PositionAccuracy
	vo.CoordinateFrame = model.CoordFrameWGS84

	// Store sensor data
	vo.SensorData = &model.SensorDataBundle{
		GNSS: gnssData,
	}
	vo.ExtractedAt = gnssData.Timestamp

	// Properties
	vo.Properties = map[string]interface{}{
		"fix_quality":     gnssData.FixQuality,
		"satellite_count": gnssData.SatelliteCount,
		"hdop":            gnssData.HDOP,
		"vdop":            gnssData.VDOP,
		"heading":         gnssData.Heading,
		"speed":           gnssData.Speed,
	}

	return vo
}

// FromPhotogrammetry creates a VectorObject from photogrammetry/3D data
func (c *SensorToVectorConverter) FromPhotogrammetry(photoData *model.PhotogramData, name string) *model.VectorObject {
	if photoData == nil {
		return nil
	}

	vo := model.New(model.ObjectTypeStructure, name)
	vo.Owner = c.DefaultOwner
	vo.Classification = "photogrammetry_model"

	// Create Polygon geometry from bounding box
	if photoData.BoundingBox != nil {
		minX := photoData.BoundingBox.Min[0]
		minY := photoData.BoundingBox.Min[1]
		maxX := photoData.BoundingBox.Max[0]
		maxY := photoData.BoundingBox.Max[1]

		vo.Geometry = &model.GeoJSONGeometry{
			Type: "Polygon",
			Coordinates: json.RawMessage(fmt.Sprintf(
				`[[[%.6f, %.6f], [%.6f, %.6f], [%.6f, %.6f], [%.6f, %.6f], [%.6f, %.6f]]]`,
				minX, minY,
				maxX, minY,
				maxX, maxY,
				minX, maxY,
				minX, minY,
			)),
			BBox: [4]float64{float64(minX), float64(minY), float64(maxX), float64(maxY)},
		}
	}

	vo.Accuracy = 0.05 // Photogrammetry typically accurate to 5cm

	// Store 3D model
	vo.ThreeDModel = &model.ThreeDModel{
		MeshURL:    photoData.MeshURL,
		TextureURL: photoData.TextureURL,
		Scale:      [3]float32{photoData.Scale, photoData.Scale, photoData.Scale},
	}

	// Store sensor data
	vo.SensorData = &model.SensorDataBundle{
		Photogrammetry: photoData,
	}
	vo.ExtractedAt = photoData.ProcessedAt

	// Properties
	vo.Properties = map[string]interface{}{
		"algorithm":       photoData.Algorithm,
		"confidence":      photoData.Confidence,
		"point_count":     photoData.PointCount,
		"triangle_count":  photoData.TriangleCount,
		"point_cloud_url": photoData.PointCloudURL,
	}

	return vo
}

// FromDroneData creates a VectorObject from drone survey data
func (c *SensorToVectorConverter) FromDroneData(droneData *model.DroneData, name string) *model.VectorObject {
	if droneData == nil {
		return nil
	}

	vo := model.New(model.ObjectTypeParcel, name)
	vo.Owner = c.DefaultOwner
	vo.Classification = "drone_survey"

	// For orthophoto, create a large polygon (footprint)
	// In practice, would use the actual GeoTIFF bounds
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "MultiPolygon",
		Coordinates: json.RawMessage(
			`[[[[0, 0], [1, 0], [1, 1], [0, 1], [0, 0]]]]`,
		),
	}

	vo.Accuracy = droneData.Resolution

	// Store 3D data
	vo.SensorData = &model.SensorDataBundle{
		DroneData: droneData,
	}
	vo.ExtractedAt = droneData.CapturedAt

	// Properties with drone-specific metadata
	vo.Properties = map[string]interface{}{
		"drone_type":    droneData.DroneType,
		"altitude":      droneData.FlightAltitude,
		"resolution_cm": droneData.Resolution,
		"coverage_pct":  droneData.Coverage,
	}

	return vo
}

// FromCamera creates a VectorObject from camera image + features
func (c *SensorToVectorConverter) FromCamera(cameraData *model.CameraData, lat, lon float64) *model.VectorObject {
	if cameraData == nil {
		return nil
	}

	vo := model.New(model.ObjectTypeLandmark, "Camera Observation")
	vo.Owner = c.DefaultOwner
	vo.Classification = "camera_image"

	// Store as point at location
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "Point",
		Coordinates: json.RawMessage(fmt.Sprintf(
			`[%.6f, %.6f]`,
			lon, lat,
		)),
	}

	vo.SensorData = &model.SensorDataBundle{
		Camera: cameraData,
	}
	vo.ExtractedAt = cameraData.CapturedAt

	// Extract and store detected features
	featureCount := len(cameraData.FeaturePoints)
	vo.Properties = map[string]interface{}{
		"image_url":      cameraData.ImageURL,
		"camera_model":   cameraData.CameraModel,
		"focal_length":   cameraData.FocalLength,
		"features_found": featureCount,
		"image_size":     cameraData.ImageSize,
	}

	if featureCount > 0 {
		// Add tags for strong features
		for i, fp := range cameraData.FeaturePoints {
			if fp.Strength > 0.8 && i < 5 { // Keep top 5 features
				vo.Tags = append(vo.Tags, fmt.Sprintf("feature_%d", i))
			}
		}
	}

	return vo
}

// FromLiDAR creates a VectorObject from LiDAR point cloud
func (c *SensorToVectorConverter) FromLiDAR(lidarData *model.LiDARData, lat, lon float64) *model.VectorObject {
	if lidarData == nil {
		return nil
	}

	vo := model.New(model.ObjectTypeStructure, "LiDAR Point Cloud")
	vo.Owner = c.DefaultOwner
	vo.Classification = "lidar_scan"

	// Store as point at location
	vo.Geometry = &model.GeoJSONGeometry{
		Type: "Point",
		Coordinates: json.RawMessage(fmt.Sprintf(
			`[%.6f, %.6f]`,
			lon, lat,
		)),
	}

	vo.Accuracy = lidarData.Resolution

	vo.SensorData = &model.SensorDataBundle{
		LiDAR: lidarData,
	}
	vo.ExtractedAt = lidarData.CapturedAt

	// Properties
	vo.Properties = map[string]interface{}{
		"scanner_type":  lidarData.ScannerType,
		"resolution_cm": lidarData.Resolution,
		"point_count":   len(lidarData.PointCloud),
		"scan_lines":    lidarData.ScanLines,
	}

	return vo
}

// MergeMultipleSensors creates a single VectorObject combining multiple sensor sources
func (c *SensorToVectorConverter) MergeMultipleSensors(
	name string,
	gnss *model.GNSSData,
	imu *model.IMUData,
	photogram *model.PhotogramData,
	camera *model.CameraData,
) *model.VectorObject {

	vo := model.New(model.ObjectTypeParcel, name)
	vo.Owner = c.DefaultOwner
	vo.Classification = "multi_sensor_fusion"

	// Use GNSS for primary geometry
	if gnss != nil {
		vo.Geometry = &model.GeoJSONGeometry{
			Type: "Point",
			Coordinates: json.RawMessage(fmt.Sprintf(
				`[%.6f, %.6f, %.2f]`,
				gnss.Position[1], gnss.Position[0], gnss.Position[2],
			)),
		}
		vo.Accuracy = gnss.PositionAccuracy
	}

	// Combine all sensor data
	vo.SensorData = &model.SensorDataBundle{
		GNSS:           gnss,
		IMU:            imu,
		Photogrammetry: photogram,
		Camera:         camera,
	}

	// Use earliest timestamp
	timestamps := []time.Time{}
	if gnss != nil {
		timestamps = append(timestamps, gnss.Timestamp)
	}
	if photogram != nil {
		timestamps = append(timestamps, photogram.ProcessedAt)
	}
	if camera != nil {
		timestamps = append(timestamps, camera.CapturedAt)
	}

	if len(timestamps) > 0 {
		earliest := timestamps[0]
		for _, ts := range timestamps {
			if ts.Before(earliest) {
				earliest = ts
			}
		}
		vo.ExtractedAt = earliest
	}

	// Aggregate properties
	props := make(map[string]interface{})

	if gnss != nil {
		props["gnss_fix"] = gnss.FixQuality
		props["satellites"] = gnss.SatelliteCount
	}

	if imu != nil {
		props["orientation"] = imu.Orientation
	}

	if photogram != nil {
		props["3d_model"] = "available"
		props["confidence"] = photogram.Confidence
	}

	if camera != nil {
		props["image"] = "available"
		props["features"] = len(camera.FeaturePoints)
	}

	vo.Properties = props
	vo.Tags = []string{"multi-sensor", "fused"}

	return vo
}

// ValidateConversion checks if conversion was successful
func (c *SensorToVectorConverter) ValidateConversion(vo *model.VectorObject) error {
	if vo == nil {
		return fmt.Errorf("vector object is nil")
	}

	if vo.Geometry == nil {
		return fmt.Errorf("geometry is empty")
	}

	if vo.SensorData == nil {
		return fmt.Errorf("sensor data is missing")
	}

	if vo.ID == uuid.Nil {
		return fmt.Errorf("object ID not set")
	}

	return nil
}

// ConversionStats provides statistics on the conversion
type ConversionStats struct {
	ObjectType     string
	SensorType     string
	GeometryType   string
	PropertyCount  int
	TagCount       int
	HasModel       bool
	Confidence     float32
}

// GetConversionStats returns stats about a converted object
func (c *SensorToVectorConverter) GetConversionStats(vo *model.VectorObject) *ConversionStats {
	stats := &ConversionStats{
		ObjectType: string(vo.Type),
		TagCount:   len(vo.Tags),
	}

	if vo.Geometry != nil {
		stats.GeometryType = vo.Geometry.Type
	}

	if vo.Properties != nil {
		stats.PropertyCount = len(vo.Properties)
	}

	stats.HasModel = vo.ThreeDModel != nil

	if vo.SensorData != nil {
		if vo.SensorData.GNSS != nil {
			stats.SensorType = "GNSS"
			stats.Confidence = 0.95
		}
		if vo.SensorData.Photogrammetry != nil {
			stats.SensorType = "Photogrammetry"
			stats.Confidence = vo.SensorData.Photogrammetry.Confidence
		}
		if vo.SensorData.DroneData != nil {
			stats.SensorType = "Drone"
			stats.Confidence = 0.90
		}
	}

	return stats
}
