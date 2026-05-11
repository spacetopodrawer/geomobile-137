package geospatial

import (
	"encoding/json"
	"log"
	"math"
)

type Point struct {
	Lat float64
	Lon float64
}

type BoundingBox struct {
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64
}

func PointDistance(p1, p2 Point) float64 {
	const earthRadiusKm = 6371.0

	lat1 := toRad(p1.Lat)
	lon1 := toRad(p1.Lon)
	lat2 := toRad(p2.Lat)
	lon2 := toRad(p2.Lon)

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func ValidateBounds(bbox BoundingBox) bool {
	if bbox.MinLat < -90 || bbox.MaxLat > 90 {
		return false
	}
	if bbox.MinLon < -180 || bbox.MaxLon > 180 {
		return false
	}
	return bbox.MinLat < bbox.MaxLat && bbox.MinLon < bbox.MaxLon
}

func ToGeoJSON(geometry json.RawMessage) string {
	geojson := map[string]interface{}{
		"type":        "Feature",
		"properties":  map[string]interface{}{},
		"geometry":    json.RawMessage(geometry),
	}

	data, err := json.MarshalIndent(geojson, "", "  ")
	if err != nil {
		log.Printf("Error marshaling GeoJSON: %v", err)
		return "{}"
	}
	return string(data)
}

func toRad(degrees float64) float64 {
	return degrees * math.Pi / 180
}
