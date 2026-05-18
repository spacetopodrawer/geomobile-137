package geometry

import (
	"encoding/json"
	"errors"
	"math"

	"github.com/go-echarts/go-echarts/v2/charts"
)

// GeometryFormat represents supported input formats
type GeometryFormat string

const (
	FormatGeoJSON    GeometryFormat = "geojson"
	FormatGeoTIFF    GeometryFormat = "geotiff"
	FormatShapefile  GeometryFormat = "shapefile"
	FormatPointCloud GeometryFormat = "pointcloud"
)

// GLTFDocument represents a unified glTF 2.0 geometry document
type GLTFDocument struct {
	Meshes      []*GLTFMesh      `json:"meshes"`
	Materials   []*GLTFMaterial  `json:"materials"`
	Nodes       []*GLTFNode      `json:"nodes"`
	Geometries  []*GLTFGeometry  `json:"geometries"`
	Metadata    map[string]interface{} `json:"metadata"`
	BoundingBox *BoundingBox     `json:"bounding_box"`
	LODLevels   []*LODLevel      `json:"lod_levels"`
}

// GLTFMesh represents a glTF mesh primitive
type GLTFMesh struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Vertices      []float32           `json:"vertices"`      // x,y,z interleaved
	Normals       []float32           `json:"normals"`       // x,y,z interleaved
	TexCoords     []float32           `json:"tex_coords"`    // u,v interleaved
	Indices       []uint32            `json:"indices"`
	Attributes    map[string]interface{} `json:"attributes"` // Custom: parcel_id, owner, area, etc.
	Material      *GLTFMaterial       `json:"material"`
	VertexCount   int                 `json:"vertex_count"`
	TriangleCount int                 `json:"triangle_count"`
}

// GLTFMaterial represents material properties
type GLTFMaterial struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Color         [4]float32 `json:"color"` // RGBA
	Metallic      float32    `json:"metallic"`
	Roughness     float32    `json:"roughness"`
	NormalMap     string     `json:"normal_map_url,omitempty"`
	AmbientOcclusion float32 `json:"ambient_occlusion"`
}

// GLTFNode represents a transform node
type GLTFNode struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MeshID       string    `json:"mesh_id"`
	Translation  [3]float64 `json:"translation"`
	Rotation     [4]float32 `json:"rotation"` // Quaternion
	Scale        [3]float32 `json:"scale"`
	Children     []string  `json:"children,omitempty"`
}

// GLTFGeometry represents geometric data
type GLTFGeometry struct {
	ID         string    `json:"id"`
	Format     string    `json:"format"`      // Source format
	SourcePath string    `json:"source_path"`
	Vertices   int       `json:"vertices"`
	Triangles  int       `json:"triangles"`
	FileSize   int64     `json:"file_size_bytes"`
}

// BoundingBox represents spatial bounds
type BoundingBox struct {
	MinX float64 `json:"min_x"`
	MaxX float64 `json:"max_x"`
	MinY float64 `json:"min_y"`
	MaxY float64 `json:"max_y"`
	MinZ float64 `json:"min_z"`
	MaxZ float64 `json:"max_z"`
}

// LODLevel represents a Level-of-Detail version
type LODLevel struct {
	Level        int      `json:"level"` // 0=high, 1=medium, 2=low
	VertexCount  int      `json:"vertex_count"`
	TriangleCount int     `json:"triangle_count"`
	FileSizeBytes int64   `json:"file_size_bytes"`
	MeshIDs      []string `json:"mesh_ids"`
}

// GeoJSONConverter converts GeoJSON features to glTF
type GeoJSONConverter struct {
	SourcePath string
	Data       map[string]interface{}
}

// NewGeoJSONConverter creates a new GeoJSON converter
func NewGeoJSONConverter(path string, data map[string]interface{}) *GeoJSONConverter {
	return &GeoJSONConverter{
		SourcePath: path,
		Data:       data,
	}
}

// ToGLTF converts GeoJSON to glTF document
func (gc *GeoJSONConverter) ToGLTF() (*GLTFDocument, error) {
	doc := &GLTFDocument{
		Meshes:     []*GLTFMesh{},
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   make(map[string]interface{}),
		LODLevels:  []*LODLevel{},
	}

	// Extract features from GeoJSON
	features, ok := gc.Data["features"].([]interface{})
	if !ok {
		return nil, errors.New("invalid GeoJSON structure: missing features array")
	}

	bbox := &BoundingBox{
		MinX: math.MaxFloat64,
		MaxX: -math.MaxFloat64,
		MinY: math.MaxFloat64,
		MaxY: -math.MaxFloat64,
		MinZ: 0,
		MaxZ: 0,
	}

	vertexCount := 0
	triangleCount := 0

	// Process each feature (parcel)
	for i, f := range features {
		feature, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract parcel properties
		properties := feature["properties"].(map[string]interface{})
		parcelID := properties["parcel_id"].(string)
		owner := properties["owner"].(string)
		area := properties["area"].(float64)

		// Create mesh for this parcel
		mesh := &GLTFMesh{
			ID:         parcelID,
			Name:       parcelID,
			Attributes: map[string]interface{}{},
			Vertices:   []float32{},
			Normals:    []float32{},
			Indices:    []uint32{},
		}

		// Add custom properties to mesh
		mesh.Attributes["parcel_id"] = parcelID
		mesh.Attributes["owner"] = owner
		mesh.Attributes["area"] = area

		// Create material for parcel
		material := &GLTFMaterial{
			ID:    parcelID + "_mat",
			Name:  parcelID + "_material",
			Color: [4]float32{0.2, 0.6, 0.8, 0.9}, // Default blue
		}
		doc.Materials = append(doc.Materials, material)
		mesh.Material = material

		// Create node for parcel
		node := &GLTFNode{
			ID:     parcelID + "_node",
			Name:   parcelID,
			MeshID: parcelID,
			Scale:  [3]float32{1, 1, 1},
		}
		doc.Nodes = append(doc.Nodes, node)

		// Update bounds (simplified - in real implementation would parse geometry)
		bbox.MinX = math.Min(bbox.MinX, area*0.001)
		bbox.MaxX = math.Max(bbox.MaxX, area*0.002)
		bbox.MinY = math.Min(bbox.MinY, area*0.001)
		bbox.MaxY = math.Max(bbox.MaxY, area*0.002)

		mesh.VertexCount = 4  // Simplified: would be parsed from geometry
		mesh.TriangleCount = 2 // Simplified: would be calculated

		vertexCount += mesh.VertexCount
		triangleCount += mesh.TriangleCount

		doc.Meshes = append(doc.Meshes, mesh)
	}

	doc.BoundingBox = bbox
	doc.Geometries = append(doc.Geometries, &GLTFGeometry{
		ID:         "geojson_source",
		Format:     "geojson",
		SourcePath: gc.SourcePath,
		Vertices:   vertexCount,
		Triangles:  triangleCount,
		FileSize:   int64(len(json.Marshal(gc.Data))),
	})

	return doc, nil
}

// GeoTIFFConverter converts GeoTIFF DEM to glTF terrain mesh
type GeoTIFFConverter struct {
	SourcePath string
	Width      int
	Height     int
	Data       []float32 // Height grid
}

// NewGeoTIFFConverter creates a new GeoTIFF converter
func NewGeoTIFFConverter(path string, width, height int, data []float32) *GeoTIFFConverter {
	return &GeoTIFFConverter{
		SourcePath: path,
		Width:      width,
		Height:     height,
		Data:       data,
	}
}

// ToGLTF converts GeoTIFF DEM to glTF mesh
func (gc *GeoTIFFConverter) ToGLTF() (*GLTFDocument, error) {
	doc := &GLTFDocument{
		Meshes:     []*GLTFMesh{},
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   make(map[string]interface{}),
		LODLevels:  []*LODLevel{},
	}

	// Create terrain material
	material := &GLTFMaterial{
		ID:    "terrain_material",
		Name:  "Terrain",
		Color: [4]float32{0.6, 0.8, 0.4, 1.0}, // Green
	}
	doc.Materials = append(doc.Materials, material)

	// Create procedural terrain mesh
	mesh := &GLTFMesh{
		ID:         "terrain_mesh",
		Name:       "DEM Terrain",
		Vertices:   []float32{},
		Normals:    []float32{},
		Indices:    []uint32{},
		Attributes: map[string]interface{}{},
		Material:   material,
	}

	// Generate grid vertices from height data
	vertexIndex := 0
	bbox := &BoundingBox{MinX: 0, MaxX: float64(gc.Width), MinY: 0, MaxY: float64(gc.Height), MinZ: 0, MaxZ: 0}

	for y := 0; y < gc.Height; y++ {
		for x := 0; x < gc.Width; x++ {
			idx := y*gc.Width + x
			if idx < len(gc.Data) {
				height := gc.Data[idx]
				mesh.Vertices = append(mesh.Vertices, float32(x), float32(y), height)
				mesh.Normals = append(mesh.Normals, 0, 0, 1) // Simplified normals

				bbox.MaxZ = math.Max(bbox.MaxZ, float64(height))
				vertexIndex++
			}
		}
	}

	// Generate triangle indices
	for y := 0; y < gc.Height-1; y++ {
		for x := 0; x < gc.Width-1; x++ {
			idx := uint32(y*gc.Width + x)
			// Two triangles per quad
			mesh.Indices = append(mesh.Indices, idx, idx+1, idx+uint32(gc.Width))
			mesh.Indices = append(mesh.Indices, idx+1, idx+uint32(gc.Width)+1, idx+uint32(gc.Width))
		}
	}

	mesh.VertexCount = vertexIndex
	mesh.TriangleCount = len(mesh.Indices) / 3

	doc.Meshes = append(doc.Meshes, mesh)
	doc.BoundingBox = bbox

	// Create node
	node := &GLTFNode{
		ID:     "terrain_node",
		Name:   "Terrain",
		MeshID: "terrain_mesh",
		Scale:  [3]float32{1, 1, 1},
	}
	doc.Nodes = append(doc.Nodes, node)

	doc.Geometries = append(doc.Geometries, &GLTFGeometry{
		ID:         "geotiff_source",
		Format:     "geotiff",
		SourcePath: gc.SourcePath,
		Vertices:   mesh.VertexCount,
		Triangles:  mesh.TriangleCount,
		FileSize:   int64(len(gc.Data) * 4),
	})

	return doc, nil
}

// ShapefileConverter converts shapefile to glTF
type ShapefileConverter struct {
	SourcePath string
	ShapeCount int
}

// NewShapefileConverter creates a new shapefile converter
func NewShapefileConverter(path string) *ShapefileConverter {
	return &ShapefileConverter{SourcePath: path}
}

// ToGLTF converts shapefile to glTF (stub)
func (sc *ShapefileConverter) ToGLTF() (*GLTFDocument, error) {
	doc := &GLTFDocument{
		Meshes:     []*GLTFMesh{},
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   make(map[string]interface{}),
		LODLevels:  []*LODLevel{},
	}

	// Placeholder: Would parse shapefile and convert geometries
	doc.Geometries = append(doc.Geometries, &GLTFGeometry{
		ID:         "shapefile_source",
		Format:     "shapefile",
		SourcePath: sc.SourcePath,
		Vertices:   0,
		Triangles:  0,
	})

	return doc, nil
}
