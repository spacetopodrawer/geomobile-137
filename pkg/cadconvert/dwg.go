package cadconvert

type DWGParser struct{}

func NewDWGParser() *DWGParser {
	return &DWGParser{}
}

func (p *DWGParser) Parse(data []byte) (*UnifiedGeometry, error) {
	return &UnifiedGeometry{
		Type:   "MultiPolygon",
		Layers: []string{"layer_1"},
	}, nil
}

func (p *DWGParser) Export(geom *UnifiedGeometry) ([]byte, error) {
	return []byte("DWG export not yet implemented"), nil
}
