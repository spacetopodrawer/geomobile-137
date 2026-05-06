package convert

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"cadastreia/pkg/model"
)

// ArcadePalette defines Neo-Geo compatible 16-color palette
var DefaultArcadePalette = [16]color.Color{
	color.RGBA{0, 0, 0, 255},       // 0: Black
	color.RGBA{255, 0, 0, 255},     // 1: Red
	color.RGBA{0, 255, 0, 255},     // 2: Green
	color.RGBA{255, 255, 0, 255},   // 3: Yellow
	color.RGBA{0, 0, 255, 255},     // 4: Blue
	color.RGBA{255, 0, 255, 255},   // 5: Magenta
	color.RGBA{0, 255, 255, 255},   // 6: Cyan
	color.RGBA{255, 255, 255, 255}, // 7: White
	color.RGBA{128, 0, 0, 255},     // 8: Dark Red
	color.RGBA{0, 128, 0, 255},     // 9: Dark Green
	color.RGBA{128, 128, 0, 255},   // 10: Olive
	color.RGBA{0, 0, 128, 255},     // 11: Navy
	color.RGBA{128, 0, 128, 255},   // 12: Purple
	color.RGBA{0, 128, 128, 255},   // 13: Teal
	color.RGBA{128, 128, 128, 255}, // 14: Gray
	color.RGBA{192, 192, 192, 255}, // 15: Light Gray
}

// VectorToArcadeConverter converts VectorObject to arcade sprites
type VectorToArcadeConverter struct {
	SpriteWidth  int
	SpriteHeight int
	Palette      [16]color.Color
}

// NewVectorToArcadeConverter creates a new converter
func NewVectorToArcadeConverter(width, height int) *VectorToArcadeConverter {
	return &VectorToArcadeConverter{
		SpriteWidth:  width,
		SpriteHeight: height,
		Palette:      DefaultArcadePalette,
	}
}

// ConvertToSprite converts a VectorObject to an ArcadeSprite
func (c *VectorToArcadeConverter) ConvertToSprite(vo *model.VectorObject) (*model.ArcadeSprite, error) {
	if vo == nil {
		return nil, fmt.Errorf("vector object is nil")
	}

	if vo.Geometry == nil {
		return nil, fmt.Errorf("no geometry to render")
	}

	// Create image buffer
	img := image.NewPaletted(
		image.Rect(0, 0, c.SpriteWidth, c.SpriteHeight),
		c.Palette[:],
	)

	// Render based on geometry type and object properties
	var paletteIndex uint8 = 7 // Default: white

	// Assign color based on object type
	switch vo.Type {
	case model.ObjectTypeParcel:
		paletteIndex = 2 // Green
	case model.ObjectTypeBuilding:
		paletteIndex = 1 // Red
	case model.ObjectTypeTree:
		paletteIndex = 9 // Dark Green
	case model.ObjectTypeLandmark:
		paletteIndex = 3 // Yellow
	case model.ObjectTypeRoute:
		paletteIndex = 4 // Blue
	case model.ObjectTypeSensor:
		paletteIndex = 5 // Magenta
	case model.ObjectTypeStructure:
		paletteIndex = 8 // Dark Red
	case model.ObjectTypeVegetation:
		paletteIndex = 2 // Green
	case model.ObjectTypeWaterFeature:
		paletteIndex = 6 // Cyan
	case model.ObjectTypeCustom:
		paletteIndex = 7 // White
	}

	// Use custom render style if available
	if vo.RenderStyle != nil {
		paletteIndex = vo.RenderStyle.ArcadeColor
	}

	// Render geometry
	switch vo.Geometry.Type {
	case "Point":
		c.renderPoint(img, paletteIndex)
	case "Polygon", "MultiPolygon":
		c.renderPolygon(img, paletteIndex)
	case "LineString":
		c.renderLine(img, paletteIndex)
	default:
		c.renderDefault(img, paletteIndex)
	}

	// Add animation frames if needed
	frameCount := int32(1)
	if vo.RenderStyle != nil && vo.RenderStyle.AnimationFrames > 0 {
		frameCount = vo.RenderStyle.AnimationFrames
	}

	// Convert to sprite
	sprite := &model.ArcadeSprite{
		Data:          img.Pix,
		Width:         int32(c.SpriteWidth),
		Height:        int32(c.SpriteHeight),
		PaletteIndex:  paletteIndex,
		Collision:     c.generateCollisionMap(img),
		Frames:        frameCount,
		FrameDuration: 100, // 100ms per frame
	}

	return sprite, nil
}

// renderPoint renders a point as a small circle
func (c *VectorToArcadeConverter) renderPoint(img *image.Paletted, colorIndex uint8) {
	centerX := c.SpriteWidth / 2
	centerY := c.SpriteHeight / 2
	radius := c.SpriteWidth / 4

	// Draw filled circle
	for y := 0; y < c.SpriteHeight; y++ {
		for x := 0; x < c.SpriteWidth; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist <= float64(radius) {
				img.SetColorIndex(x, y, colorIndex)
			}
		}
	}
}

// renderPolygon renders a polygon as a filled rectangle
func (c *VectorToArcadeConverter) renderPolygon(img *image.Paletted, colorIndex uint8) {
	// Fill most of the sprite area (polygon footprint)
	margin := 4

	for y := margin; y < c.SpriteHeight-margin; y++ {
		for x := margin; x < c.SpriteWidth-margin; x++ {
			img.SetColorIndex(x, y, colorIndex)
		}
	}

	// Draw border
	borderColor := uint8((colorIndex + 1) % 16)
	for y := margin; y < c.SpriteHeight-margin; y++ {
		img.SetColorIndex(margin, y, borderColor)
		img.SetColorIndex(c.SpriteWidth-margin-1, y, borderColor)
	}
	for x := margin; x < c.SpriteWidth-margin; x++ {
		img.SetColorIndex(x, margin, borderColor)
		img.SetColorIndex(x, c.SpriteHeight-margin-1, borderColor)
	}
}

// renderLine renders a line
func (c *VectorToArcadeConverter) renderLine(img *image.Paletted, colorIndex uint8) {
	// Draw horizontal line across sprite
	centerY := c.SpriteHeight / 2

	for x := 0; x < c.SpriteWidth; x++ {
		img.SetColorIndex(x, centerY, colorIndex)
		if centerY > 0 {
			img.SetColorIndex(x, centerY-1, colorIndex)
		}
	}
}

// renderDefault renders a default pattern
func (c *VectorToArcadeConverter) renderDefault(img *image.Paletted, colorIndex uint8) {
	// Fill with pattern
	pattern := false

	for y := 0; y < c.SpriteHeight; y++ {
		for x := 0; x < c.SpriteWidth; x++ {
			if pattern {
				img.SetColorIndex(x, y, colorIndex)
			}
			pattern = !pattern
		}
		pattern = !pattern // Start next row with opposite
	}
}

// generateCollisionMap generates a collision map from the sprite
func (c *VectorToArcadeConverter) generateCollisionMap(img *image.Paletted) [][]bool {
	collision := make([][]bool, c.SpriteHeight)

	for y := 0; y < c.SpriteHeight; y++ {
		collision[y] = make([]bool, c.SpriteWidth)
		for x := 0; x < c.SpriteWidth; x++ {
			// Pixels with non-zero color index are solid
			colorIndex := img.ColorIndexAt(x, y)
			collision[y][x] = colorIndex > 0
		}
	}

	return collision
}

// SpriteSheet creates a sprite sheet for multiple objects
type SpriteSheet struct {
	Sprites    []*model.ArcadeSprite
	Width      int
	Height     int
	TileWidth  int
	TileHeight int
}

// NewSpriteSheet creates a sprite sheet
func (c *VectorToArcadeConverter) NewSpriteSheet(objects []*model.VectorObject) *SpriteSheet {
	sheet := &SpriteSheet{
		Sprites:    make([]*model.ArcadeSprite, 0),
		TileWidth:  c.SpriteWidth,
		TileHeight: c.SpriteHeight,
	}

	// Convert each object
	for _, vo := range objects {
		if sprite, err := c.ConvertToSprite(vo); err == nil {
			sheet.Sprites = append(sheet.Sprites, sprite)
		} else {
			log.Printf("Error converting object %s: %v\n", vo.ID, err)
		}
	}

	// Calculate sheet dimensions (layout as grid)
	spritesPerRow := 4
	rows := (len(sheet.Sprites) + spritesPerRow - 1) / spritesPerRow

	sheet.Width = spritesPerRow * c.SpriteWidth
	sheet.Height = rows * c.SpriteHeight

	return sheet
}

// GetSpriteData returns raw sprite data
func (c *VectorToArcadeConverter) GetSpriteData(sprite *model.ArcadeSprite) []byte {
	return sprite.Data
}

// OptimizeForArcade optimizes sprite for arcade limitations
func (c *VectorToArcadeConverter) OptimizeForArcade(sprite *model.ArcadeSprite) *model.ArcadeSprite {
	if sprite == nil {
		return nil
	}

	// Ensure within size limits (Neo-Geo max 512x512)
	if sprite.Width > 512 {
		sprite.Width = 512
	}
	if sprite.Height > 512 {
		sprite.Height = 512
	}

	// Limit frames (Neo-Geo practical limit)
	if sprite.Frames > 16 {
		sprite.Frames = 16
	}

	// Ensure frame duration is reasonable
	if sprite.FrameDuration < 16 {
		sprite.FrameDuration = 16 // Min 16ms (60 FPS)
	}

	return sprite
}

// ConvertStats provides conversion statistics
type ConvertStats struct {
	SpriteSize    int64  // Bytes
	CompressionRatio float32
	TotalPixels   int32
	OpaquePixels  int32
	TransparentPixels int32
	PaletteUsage  int32  // Out of 16 colors
}

// GetConvertStats returns statistics about the conversion
func (c *VectorToArcadeConverter) GetConvertStats(sprite *model.ArcadeSprite) *ConvertStats {
	if sprite == nil {
		return nil
	}

	stats := &ConvertStats{
		SpriteSize: int64(len(sprite.Data)),
		TotalPixels: sprite.Width * sprite.Height,
	}

	// Count opaque vs transparent
	for _, px := range sprite.Data {
		if px > 0 {
			stats.OpaquePixels++
		} else {
			stats.TransparentPixels++
		}
	}

	// Estimate palette usage (colors 0-15)
	colorSet := make(map[uint8]bool)
	for _, px := range sprite.Data {
		if px < 16 {
			colorSet[px] = true
		}
	}
	stats.PaletteUsage = int32(len(colorSet))

	// Calculate compression ratio (theoretical)
	uncompressed := stats.TotalPixels * 8 // 8-bit per pixel
	if uncompressed > 0 {
		stats.CompressionRatio = float32(stats.SpriteSize) / float32(uncompressed)
	}

	return stats
}
