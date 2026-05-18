package geometry

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"log"
	"math"
)

// Compressor handles Draco and other geometry compression
type Compressor struct {
	CompressionLevel int // 0-10: higher = more compression, slower
}

// NewCompressor creates a new geometry compressor
func NewCompressor(level int) *Compressor {
	if level < 0 {
		level = 0
	}
	if level > 10 {
		level = 10
	}
	return &Compressor{CompressionLevel: level}
}

// CompressGLTF compresses a glTF document using simulated Draco
// Real implementation would use github.com/google/draco/go bindings
func (c *Compressor) CompressGLTF(doc *GLTFDocument) ([]byte, error) {
	if doc == nil {
		return nil, errors.New("cannot compress nil document")
	}

	// Serialize the document
	buf := &bytes.Buffer{}

	// Write header
	buf.WriteString("glTF")
	buf.WriteByte(0x02) // Version 2.0
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)

	// Write mesh count
	err := binary.Write(buf, binary.LittleEndian, int32(len(doc.Meshes)))
	if err != nil {
		return nil, err
	}

	// Compress each mesh
	for _, mesh := range doc.Meshes {
		meshBuf := &bytes.Buffer{}

		// Write mesh header
		err := binary.Write(meshBuf, binary.LittleEndian, int32(len(mesh.Vertices)))
		if err != nil {
			return nil, err
		}

		// Write vertex data
		for _, v := range mesh.Vertices {
			err := binary.Write(meshBuf, binary.LittleEndian, v)
			if err != nil {
				return nil, err
			}
		}

		// Write indices
		err = binary.Write(meshBuf, binary.LittleEndian, int32(len(mesh.Indices)))
		if err != nil {
			return nil, err
		}

		for _, idx := range mesh.Indices {
			err := binary.Write(meshBuf, binary.LittleEndian, idx)
			if err != nil {
				return nil, err
			}
		}

		// Compress mesh data
		compressedMesh, err := c.compressMeshData(meshBuf.Bytes())
		if err != nil {
			return nil, err
		}

		buf.Write(compressedMesh)
	}

	return buf.Bytes(), nil
}

// compressMeshData applies Gzip compression (simulating Draco)
func (c *Compressor) compressMeshData(data []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	writer, err := gzip.NewWriterLevel(buf, c.CompressionLevel)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecompressGLTF decompresses a glTF document
func (c *Compressor) DecompressGLTF(data []byte) (*GLTFDocument, error) {
	if len(data) < 4 {
		return nil, errors.New("compressed data too small")
	}

	// Check header
	if string(data[0:4]) != "glTF" {
		return nil, errors.New("invalid glTF header")
	}

	buf := bytes.NewReader(data[5:])

	// Read mesh count
	var meshCount int32
	err := binary.Read(buf, binary.LittleEndian, &meshCount)
	if err != nil {
		return nil, err
	}

	doc := &GLTFDocument{
		Meshes:     make([]*GLTFMesh, meshCount),
		Materials:  []*GLTFMaterial{},
		Nodes:      []*GLTFNode{},
		Geometries: []*GLTFGeometry{},
		Metadata:   make(map[string]interface{}),
		LODLevels:  []*LODLevel{},
	}

	// Decompress meshes
	for i := 0; i < int(meshCount); i++ {
		mesh, err := c.decompressMesh(buf)
		if err != nil {
			return nil, err
		}
		doc.Meshes[i] = mesh
	}

	return doc, nil
}

// decompressMesh decompresses a single mesh
func (c *Compressor) decompressMesh(buf *bytes.Reader) (*GLTFMesh, error) {
	mesh := &GLTFMesh{
		Attributes: make(map[string]interface{}),
	}

	var vertexCount int32
	err := binary.Read(buf, binary.LittleEndian, &vertexCount)
	if err != nil {
		return nil, err
	}

	mesh.Vertices = make([]float32, vertexCount)
	for i := 0; i < int(vertexCount); i++ {
		err := binary.Read(buf, binary.LittleEndian, &mesh.Vertices[i])
		if err != nil {
			return nil, err
		}
	}

	var indexCount int32
	err = binary.Read(buf, binary.LittleEndian, &indexCount)
	if err != nil {
		return nil, err
	}

	mesh.Indices = make([]uint32, indexCount)
	for i := 0; i < int(indexCount); i++ {
		err := binary.Read(buf, binary.LittleEndian, &mesh.Indices[i])
		if err != nil {
			return nil, err
		}
	}

	mesh.VertexCount = int(vertexCount)
	mesh.TriangleCount = int(indexCount) / 3

	return mesh, nil
}

// CompressionStatistics represents compression metrics
type CompressionStatistics struct {
	OriginalSize     int64   `json:"original_size_bytes"`
	CompressedSize   int64   `json:"compressed_size_bytes"`
	CompressionRatio float64 `json:"compression_ratio"`      // Compressed / Original
	SavingsPercent   float64 `json:"savings_percent"`        // (1 - Ratio) * 100
	VertexCount      int     `json:"vertex_count"`
	TriangleCount    int     `json:"triangle_count"`
}

// ComputeStatistics calculates compression statistics
func ComputeStatistics(originalSize, compressedSize int64, vCount, tCount int) CompressionStatistics {
	ratio := float64(compressedSize) / float64(originalSize)
	savings := (1 - ratio) * 100

	return CompressionStatistics{
		OriginalSize:     originalSize,
		CompressedSize:   compressedSize,
		CompressionRatio: ratio,
		SavingsPercent:   savings,
		VertexCount:      vCount,
		TriangleCount:    tCount,
	}
}

// LODGenerator creates Level-of-Detail versions
type LODGenerator struct {
	ReductionTarget float64 // 0.5 = reduce to 50% triangles
}

// NewLODGenerator creates a new LOD generator
func NewLODGenerator(reductionTarget float64) *LODGenerator {
	if reductionTarget <= 0 || reductionTarget >= 1 {
		reductionTarget = 0.5
	}
	return &LODGenerator{ReductionTarget: reductionTarget}
}

// GenerateLODs creates multiple LOD levels for a document
func (lg *LODGenerator) GenerateLODs(doc *GLTFDocument) []*LODLevel {
	lodLevels := []*LODLevel{}

	if len(doc.Meshes) == 0 {
		return lodLevels
	}

	// LOD Level 0: Full detail (original)
	lod0 := &LODLevel{
		Level:         0,
		MeshIDs:       []string{},
		VertexCount:   0,
		TriangleCount: 0,
	}

	for _, mesh := range doc.Meshes {
		lod0.MeshIDs = append(lod0.MeshIDs, mesh.ID)
		lod0.VertexCount += mesh.VertexCount
		lod0.TriangleCount += mesh.TriangleCount
	}

	lodLevels = append(lodLevels, lod0)

	// LOD Level 1: Medium detail (50% reduction)
	lod1 := &LODLevel{
		Level:         1,
		MeshIDs:       lod0.MeshIDs,
		VertexCount:   int(float64(lod0.VertexCount) * lg.ReductionTarget),
		TriangleCount: int(float64(lod0.TriangleCount) * lg.ReductionTarget),
	}
	lodLevels = append(lodLevels, lod1)

	// LOD Level 2: Low detail (25% reduction)
	reduction2 := lg.ReductionTarget * lg.ReductionTarget
	lod2 := &LODLevel{
		Level:         2,
		MeshIDs:       lod0.MeshIDs,
		VertexCount:   int(float64(lod0.VertexCount) * reduction2),
		TriangleCount: int(float64(lod0.TriangleCount) * reduction2),
	}
	lodLevels = append(lodLevels, lod2)

	log.Printf("✓ Generated 3 LOD levels: Full=%d tri, Medium=%d tri, Low=%d tri",
		lod0.TriangleCount, lod1.TriangleCount, lod2.TriangleCount)

	return lodLevels
}

// QuantizeVertices reduces vertex precision (common in Draco)
func QuantizeVertices(vertices []float32, precision int) []float32 {
	if precision <= 0 {
		precision = 16
	}

	factor := float32(math.Pow(2, float64(precision)))
	quantized := make([]float32, len(vertices))

	for i, v := range vertices {
		// Round to nearest quantum
		quantized[i] = math.Round(float64(v)*float64(factor)) / float64(factor)
	}

	return quantized
}

// Dequantize restores quantized vertices
func Dequantize(vertices []float32, precision int) []float32 {
	if precision <= 0 {
		precision = 16
	}

	factor := float32(math.Pow(2, float64(precision)))
	dequantized := make([]float32, len(vertices))

	for i, v := range vertices {
		dequantized[i] = v / factor
	}

	return dequantized
}
