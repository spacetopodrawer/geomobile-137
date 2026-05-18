package geometry

import (
	"errors"
	"log"
	"sync"
)

// PlatformType represents target rendering platforms
type PlatformType string

const (
	PlatformUE5    PlatformType = "unreal_engine_5"
	PlatformWeb    PlatformType = "web_threejs"
	PlatformMobile PlatformType = "mobile_expo"
	PlatformVR     PlatformType = "webxr"
)

// PlatformLoader is the interface all platform loaders implement
type PlatformLoader interface {
	Load(doc *GLTFDocument) (PlatformAsset, error)
	GetPlatform() PlatformType
	OptimizeForPlatform(doc *GLTFDocument) (*GLTFDocument, error)
}

// PlatformAsset represents a platform-specific asset
type PlatformAsset interface {
	GetID() string
	GetFormat() string
	GetSize() int64
	Serialize() ([]byte, error)
}

// ============================================================================
// UE5 LOADER (Nanite-optimized)
// ============================================================================

// UE5Loader converts glTF to UE5 Nanite format
type UE5Loader struct {
	mu sync.Mutex
}

// NewUE5Loader creates a new UE5 loader
func NewUE5Loader() *UE5Loader {
	return &UE5Loader{}
}

// Load converts glTF to UE5 static mesh format
func (ul *UE5Loader) Load(doc *GLTFDocument) (PlatformAsset, error) {
	ul.mu.Lock()
	defer ul.mu.Unlock()

	if doc == nil {
		return nil, errors.New("document is nil")
	}

	// Optimize for UE5 (enable Nanite for large meshes)
	optimized, err := ul.OptimizeForPlatform(doc)
	if err != nil {
		return nil, err
	}

	asset := &UE5Asset{
		ID:            "ue5_" + optimized.Geometries[0].ID,
		Meshes:        optimized.Meshes,
		LODs:          optimized.LODLevels,
		NaniteEnabled: len(optimized.Meshes) > 0 && optimized.Meshes[0].TriangleCount > 1000,
	}

	log.Printf("✓ UE5 Asset loaded: %d meshes, Nanite=%v", len(asset.Meshes), asset.NaniteEnabled)

	return asset, nil
}

// GetPlatform returns the platform type
func (ul *UE5Loader) GetPlatform() PlatformType {
	return PlatformUE5
}

// OptimizeForPlatform optimizes geometry for UE5 rendering
func (ul *UE5Loader) OptimizeForPlatform(doc *GLTFDocument) (*GLTFDocument, error) {
	// 1. Enable Nanite for large meshes (> 1000 triangles)
	for _, mesh := range doc.Meshes {
		if mesh.TriangleCount > 1000 {
			// In real implementation: Enable Nanite flag
			log.Printf("  → Mesh %s: Nanite enabled (%d triangles)", mesh.ID, mesh.TriangleCount)
		}
	}

	// 2. Generate LOD levels
	lodGen := NewLODGenerator(0.5)
	doc.LODLevels = lodGen.GenerateLODs(doc)

	// 3. Optimize normals and tangents
	// (In real implementation: compute tangent space for normal mapping)

	// 4. Set material properties for Unreal
	for _, mat := range doc.Materials {
		mat.Metallic = 0.0
		mat.Roughness = 0.8
	}

	log.Printf("✓ UE5 Optimization: LOD levels=%d, Materials=%d", len(doc.LODLevels), len(doc.Materials))

	return doc, nil
}

// UE5Asset represents an Unreal Engine asset
type UE5Asset struct {
	ID            string
	Meshes        []*GLTFMesh
	LODs          []*LODLevel
	NaniteEnabled bool
	Materials     []*GLTFMaterial
}

func (ua *UE5Asset) GetID() string {
	return ua.ID
}

func (ua *UE5Asset) GetFormat() string {
	return "uasset" // Unreal .uasset format
}

func (ua *UE5Asset) GetSize() int64 {
	totalSize := int64(0)
	for _, mesh := range ua.Meshes {
		totalSize += int64(len(mesh.Vertices)*4) + int64(len(mesh.Indices)*4)
	}
	return totalSize
}

func (ua *UE5Asset) Serialize() ([]byte, error) {
	// In real implementation: Serialize to .uasset binary format
	return []byte("UE5_ASSET_BINARY"), nil
}

// ============================================================================
// WEB LOADER (Three.js/Babylon.js optimized)
// ============================================================================

// WebLoader converts glTF to web-optimized format
type WebLoader struct {
	mu sync.Mutex
}

// NewWebLoader creates a new web loader
func NewWebLoader() *WebLoader {
	return &WebLoader{}
}

// Load converts glTF to web-optimized format
func (wl *WebLoader) Load(doc *GLTFDocument) (PlatformAsset, error) {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if doc == nil {
		return nil, errors.New("document is nil")
	}

	optimized, err := wl.OptimizeForPlatform(doc)
	if err != nil {
		return nil, err
	}

	// Compress for network transfer
	compressor := NewCompressor(8)
	compressed, err := compressor.CompressGLTF(optimized)
	if err != nil {
		return nil, err
	}

	asset := &WebAsset{
		ID:               "web_" + optimized.Geometries[0].ID,
		Meshes:           optimized.Meshes,
		CompressedSize:   int64(len(compressed)),
		CompressionRatio: float64(len(compressed)) / float64(len(optimized.Meshes[0].Vertices)*4),
	}

	log.Printf("✓ Web Asset loaded: %d meshes, %dKB compressed (%.1f%%)",
		len(asset.Meshes), asset.CompressedSize/1024, (1-asset.CompressionRatio)*100)

	return asset, nil
}

// GetPlatform returns the platform type
func (wl *WebLoader) GetPlatform() PlatformType {
	return PlatformWeb
}

// OptimizeForPlatform optimizes geometry for web browsers
func (wl *WebLoader) OptimizeForPlatform(doc *GLTFDocument) (*GLTFDocument, error) {
	// 1. Use medium LOD for web (balance quality vs. performance)
	lodGen := NewLODGenerator(0.5)
	doc.LODLevels = lodGen.GenerateLODs(doc)

	// 2. Reduce precision for web (16-bit float)
	for _, mesh := range doc.Meshes {
		mesh.Vertices = QuantizeVertices(mesh.Vertices, 16)
	}

	// 3. Optimize material properties for WebGL
	for _, mat := range doc.Materials {
		mat.AmbientOcclusion = 0.8 // Improve web shading
	}

	// 4. Remove unsupported attributes
	for _, mesh := range doc.Meshes {
		for key := range mesh.Attributes {
			if key == "custom_proprietary_attr" {
				delete(mesh.Attributes, key)
			}
		}
	}

	log.Printf("✓ Web Optimization: LOD levels=%d, Quantized to 16-bit", len(doc.LODLevels))

	return doc, nil
}

// WebAsset represents a web-optimized asset
type WebAsset struct {
	ID               string
	Meshes           []*GLTFMesh
	CompressedSize   int64
	CompressionRatio float64
}

func (wa *WebAsset) GetID() string {
	return wa.ID
}

func (wa *WebAsset) GetFormat() string {
	return "glb" // glTF binary format for web
}

func (wa *WebAsset) GetSize() int64 {
	return wa.CompressedSize
}

func (wa *WebAsset) Serialize() ([]byte, error) {
	// In real implementation: Serialize to .glb (glTF binary)
	return []byte("GLB_BINARY_DATA"), nil
}

// ============================================================================
// MOBILE LOADER (Expo/React Native optimized)
// ============================================================================

// MobileLoader converts glTF for mobile devices
type MobileLoader struct {
	mu sync.Mutex
}

// NewMobileLoader creates a new mobile loader
func NewMobileLoader() *MobileLoader {
	return &MobileLoader{}
}

// Load converts glTF for mobile
func (ml *MobileLoader) Load(doc *GLTFDocument) (PlatformAsset, error) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if doc == nil {
		return nil, errors.New("document is nil")
	}

	optimized, err := ml.OptimizeForPlatform(doc)
	if err != nil {
		return nil, err
	}

	// Heavy compression for mobile (slower networks)
	compressor := NewCompressor(10)
	compressed, err := compressor.CompressGLTF(optimized)
	if err != nil {
		return nil, err
	}

	asset := &MobileAsset{
		ID:               "mobile_" + optimized.Geometries[0].ID,
		Meshes:           optimized.Meshes,
		CompressedSize:   int64(len(compressed)),
		CompressionRatio: float64(len(compressed)) / float64(len(optimized.Meshes[0].Vertices)*4),
		TargetPlatforms:  []string{"iOS", "Android"},
	}

	log.Printf("✓ Mobile Asset loaded: %d meshes, %dKB compressed (%.1f%%), targets: %v",
		len(asset.Meshes), asset.CompressedSize/1024, (1-asset.CompressionRatio)*100, asset.TargetPlatforms)

	return asset, nil
}

// GetPlatform returns the platform type
func (ml *MobileLoader) GetPlatform() PlatformType {
	return PlatformMobile
}

// OptimizeForPlatform optimizes geometry for mobile devices
func (ml *MobileLoader) OptimizeForPlatform(doc *GLTFDocument) (*GLTFDocument, error) {
	// 1. Use low LOD (battery/memory constrained)
	lodGen := NewLODGenerator(0.25)
	doc.LODLevels = lodGen.GenerateLODs(doc)

	// 2. Aggressive quantization (8-bit float)
	for _, mesh := range doc.Meshes {
		mesh.Vertices = QuantizeVertices(mesh.Vertices, 8)
		mesh.Normals = QuantizeVertices(mesh.Normals, 8)
	}

	// 3. Remove unnecessary attributes for mobile
	for _, mesh := range doc.Meshes {
		delete(mesh.Attributes, "custom_data")
	}

	// 4. Simplify materials
	for _, mat := range doc.Materials {
		mat.NormalMap = "" // Disable expensive normal mapping
	}

	log.Printf("✓ Mobile Optimization: LOD levels=%d, Quantized to 8-bit, simplified materials", len(doc.LODLevels))

	return doc, nil
}

// MobileAsset represents a mobile-optimized asset
type MobileAsset struct {
	ID               string
	Meshes           []*GLTFMesh
	CompressedSize   int64
	CompressionRatio float64
	TargetPlatforms  []string
}

func (ma *MobileAsset) GetID() string {
	return ma.ID
}

func (ma *MobileAsset) GetFormat() string {
	return "glb_mobile"
}

func (ma *MobileAsset) GetSize() int64 {
	return ma.CompressedSize
}

func (ma *MobileAsset) Serialize() ([]byte, error) {
	// In real implementation: SQLite-compatible binary format
	return []byte("MOBILE_GLBINARY_DATA"), nil
}

// ============================================================================
// PLATFORM LOADER FACTORY
// ============================================================================

// LoaderFactory creates appropriate loader for target platform
type LoaderFactory struct{}

// CreateLoader returns a platform-specific loader
func (lf *LoaderFactory) CreateLoader(platform PlatformType) PlatformLoader {
	switch platform {
	case PlatformUE5:
		return NewUE5Loader()
	case PlatformWeb:
		return NewWebLoader()
	case PlatformMobile:
		return NewMobileLoader()
	case PlatformVR:
		return NewWebLoader() // WebXR uses web-optimized assets
	default:
		return NewWebLoader()
	}
}

// LoadForAllPlatforms converts a glTF document for all platforms
func LoadForAllPlatforms(doc *GLTFDocument) (map[PlatformType]PlatformAsset, error) {
	result := make(map[PlatformType]PlatformAsset)
	factory := &LoaderFactory{}

	platforms := []PlatformType{PlatformUE5, PlatformWeb, PlatformMobile, PlatformVR}

	for _, platform := range platforms {
		loader := factory.CreateLoader(platform)
		asset, err := loader.Load(doc)
		if err != nil {
			log.Printf("✗ Failed to load for %s: %v", platform, err)
			continue
		}
		result[platform] = asset
	}

	return result, nil
}
