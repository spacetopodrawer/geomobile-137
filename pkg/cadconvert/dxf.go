package cadconvert

type DXFParser struct{}

func NewDXFParser() *DXFParser {
	return &DXFParser{}
}

func (p *DXFParser) Parse(data []byte) (*UnifiedGeometry, error) {
	return &UnifiedGeometry{
		Type:   "MultiPolygon",
		Layers: []string{"layer_1"},
	}, nil
}

func (p *DXFParser) Export(geom *UnifiedGeometry) ([]byte, error) {
	return []byte("DXF export not yet implemented"), nil
}
