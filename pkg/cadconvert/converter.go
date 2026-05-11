package cadconvert

import "encoding/json"

type Converter interface {
	Parse(data []byte) (*UnifiedGeometry, error)
	Export(geom *UnifiedGeometry) ([]byte, error)
}

type UnifiedGeometry struct {
	Type        string          `json:"type"`
	Coordinates [][]float64     `json:"coordinates,omitempty"`
	Geometries  []interface{}   `json:"geometries,omitempty"`
	Layers      []string        `json:"layers,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	SourceFile  string          `json:"source_file,omitempty"`
}

type ConversionResult struct {
	Status   string         `json:"status"`
	Geometry UnifiedGeometry `json:"geometry"`
	Error    string         `json:"error,omitempty"`
}
