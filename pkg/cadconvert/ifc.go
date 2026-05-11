package cadconvert

type IFCParser struct{}

func NewIFCParser() *IFCParser {
	return &IFCParser{}
}

func (p *IFCParser) Parse(data []byte) (*UnifiedGeometry, error) {
	return &UnifiedGeometry{
		Type:   "MultiPolygon",
		Layers: []string{"building_layer"},
	}, nil
}

func (p *IFCParser) Export(geom *UnifiedGeometry) ([]byte, error) {
	return []byte("IFC export not yet implemented"), nil
}
