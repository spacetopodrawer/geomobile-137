package arcade

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// NEOGEOCompiler converts game assets to NEO-GEO ROM format
type NEOGEOCompiler struct {
	outputPath string
	palette    [16]color.RGBA
	romSize    int64 // Typical NEO-GEO ROM: 256-512MB
}

// NEOGEOROMHeader represents the NEO-GEO ROM file header
type NEOGEOROMHeader struct {
	Magic          [4]byte   // "NEOP" or "PSYQ"
	Version        uint32    // ROM version
	GameTitle      [32]byte  // Game name
	ReleaseDate    uint32    // Unix timestamp
	ScreenWidth    uint16    // 320 or 304
	ScreenHeight   uint16    // 224
	ColorMode      uint8     // 4-bit or 8-bit
	Reserved       [259]byte // Padding
}

// SpriteData represents a NEO-GEO sprite tile (16x16 pixels)
type SpriteData struct {
	TileID   uint16
	X        int16
	Y        int16
	Width    uint8
	Height   uint8
	Rotation uint8
	Palette  uint8
	Data     [256]byte // 16x16 pixels, 4-bit
}

// NEOGEOProgram represents the main program ROM
type NEOGEOProgram struct {
	Header    NEOGEOROMHeader
	CRC32     uint32
	Version   uint32
	CheckSum  uint16
	Hardware  uint16 // NEO-GEO hardware revision
	Code      []byte // Z80 compiled code (68K assembly)
	DataSize  uint32
}

// NewNEOGEOCompiler creates a new compiler instance
func NewNEOGEOCompiler(outputPath string) *NEOGEOCompiler {
	return &NEOGEOCompiler{
		outputPath: outputPath,
		palette:    GetNEOGEOPalette(),
		romSize:    512 * 1024 * 1024, // 512MB default
	}
}

// CompileGameState converts current game state to NEO-GEO ROM
func (nc *NEOGEOCompiler) CompileGameState(gameState *GameState, outputFile string) error {
	if gameState == nil {
		return fmt.Errorf("game state is nil")
	}

	// Create output file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create ROM file: %w", err)
	}
	defer f.Close()

	// Create header
	header := nc.createHeader()
	program := &NEOGEOProgram{
		Header:   header,
		CRC32:    0xDEADBEEF, // Placeholder CRC
		Version:  0x01000000, // v1.0.0.0
		CheckSum: 0x0000,
		Hardware: 0x0100, // NEO-GEO MV1
	}

	// Compile game logic to Z80 bytecode
	bytecode := nc.compileGameLogic(gameState)
	program.Code = bytecode
	program.DataSize = uint32(len(bytecode))

	// Write program ROM
	if err := nc.writeProgram(f, program); err != nil {
		return fmt.Errorf("failed to write program ROM: %w", err)
	}

	// Write sprite ROM (graphics)
	if err := nc.writeSpriteROM(f, gameState); err != nil {
		return fmt.Errorf("failed to write sprite ROM: %w", err)
	}

	// Write palette ROM
	if err := nc.writePaletteROM(f); err != nil {
		return fmt.Errorf("failed to write palette ROM: %w", err)
	}

	// Write sound ROM (if applicable)
	if err := nc.writeSoundROM(f); err != nil {
		return fmt.Errorf("failed to write sound ROM: %w", err)
	}

	return nil
}

// createHeader creates a NEO-GEO ROM header
func (nc *NEOGEOCompiler) createHeader() NEOGEOROMHeader {
	header := NEOGEOROMHeader{
		Magic:       [4]byte{'N', 'E', 'O', 'P'},
		Version:     0x01000000,
		ScreenWidth: 320,
		ScreenHeight: 224,
		ColorMode:   0x04, // 4-bit = 16 colors
	}

	// Game title
	title := "CADASTRE_IA v0.1.0"
	copy(header.GameTitle[:], title)

	return header
}

// compileGameLogic converts game state to Z80 bytecode
func (nc *NEOGEOCompiler) compileGameLogic(gameState *GameState) []byte {
	var buf bytes.Buffer

	// NEO-GEO Z80 bytecode structure
	// This is a simplified representation - real compilation would use assembler

	// Header: program initialization
	buf.Write([]byte{
		0xF3,                   // DI (disable interrupts)
		0x31, 0xFF, 0xDF,       // LD SP, 0xDFFF (stack pointer)
		0xCD, 0x00, 0x10,       // CALL initialize
		0xC9,                   // RET
	})

	// Game loop initialization
	buf.WriteString("GAME_INIT:")
	buf.Write([]byte{0x00, 0x00, 0x00}) // NOP padding

	// Player state encoding
	if gameState != nil && gameState.Player != nil {
		playerX := uint16(gameState.Player.X)
		playerY := uint16(gameState.Player.Y)

		// LD A, playerX
		buf.Write([]byte{0x3E, byte(playerX)})
		// LD B, playerY
		buf.Write([]byte{0x06, byte(playerY)})
		// Store in RAM at 0xC000
		buf.Write([]byte{0xC3, 0x00, 0x10}) // JP main_loop
	}

	// VBlank interrupt handler
	buf.WriteString("VBLANK_HANDLER:")
	buf.Write([]byte{
		0xF5,       // PUSH AF
		0xC5,       // PUSH BC
		0xD5,       // PUSH DE
		0xE5,       // PUSH HL
		0xCD, 0x00, 0x20, // CALL update_game
		0xE1,       // POP HL
		0xD1,       // POP DE
		0xC1,       // POP BC
		0xF1,       // POP AF
		0xFB,       // EI (enable interrupts)
		0xED, 0x4D, // RETI
	})

	return buf.Bytes()
}

// writeSpriteROM writes graphics data to ROM
func (nc *NEOGEOCompiler) writeSpriteROM(f *os.File, gameState *GameState) error {
	// NEO-GEO sprite format: 16x16 tiles, 4-bit per pixel
	// Typical sprite ROM: 64MB

	spriteCount := 0

	// Write player sprite
	if gameState != nil && gameState.Player != nil {
		playerSprite := nc.createPlayerSprite(gameState.Player)
		if err := nc.writeSprite(f, playerSprite); err != nil {
			return err
		}
		spriteCount++
	}

	// Write object sprites
	if gameState != nil {
		for _, obj := range gameState.Objects {
			objSprite := nc.createObjectSprite(obj)
			if err := nc.writeSprite(f, objSprite); err != nil {
				return err
			}
			spriteCount++
		}
	}

	fmt.Printf("Wrote %d sprites to sprite ROM\n", spriteCount)
	return nil
}

// createPlayerSprite generates a player sprite
func (nc *NEOGEOCompiler) createPlayerSprite(player *PlayerState) *SpriteData {
	sprite := &SpriteData{
		TileID:   0,
		X:        int16(player.X),
		Y:        int16(player.Y),
		Width:    4,   // 4x4 tiles = 64x64 pixels (Neo-Geo max)
		Height:   4,
		Rotation: 0,
		Palette:  1, // Yellow palette
	}

	// Fill with yellow color (palette index 3)
	for i := range sprite.Data {
		sprite.Data[i] = 0x33 // Yellow pattern
	}

	return sprite
}

// createObjectSprite generates an object sprite
func (nc *NEOGEOCompiler) createObjectSprite(obj *ObjectState) *SpriteData {
	sprite := &SpriteData{
		TileID:   1,
		X:        0, // Would calculate from geometry
		Y:        0,
		Width:    2,
		Height:   2,
		Rotation: 0,
		Palette:  2, // Green palette
	}

	// Fill with green color based on object type
	paletteIdx := byte(0x22) // Green default
	for i := range sprite.Data {
		sprite.Data[i] = paletteIdx
	}

	return sprite
}

// writeSprite writes a single sprite to ROM
func (nc *NEOGEOCompiler) writeSprite(f *os.File, sprite *SpriteData) error {
	buf := bytes.Buffer{}

	// Write sprite header
	binary.Write(&buf, binary.LittleEndian, sprite.TileID)
	binary.Write(&buf, binary.LittleEndian, sprite.X)
	binary.Write(&buf, binary.LittleEndian, sprite.Y)
	buf.WriteByte(sprite.Width)
	buf.WriteByte(sprite.Height)
	buf.WriteByte(sprite.Rotation)
	buf.WriteByte(sprite.Palette)

	// Write sprite data
	buf.Write(sprite.Data[:])

	_, err := f.Write(buf.Bytes())
	return err
}

// writeProgram writes the program ROM (Z80 bytecode) to file
func (nc *NEOGEOCompiler) writeProgram(f *os.File, program *NEOGEOProgram) error {
	var buf bytes.Buffer

	// Write header
	buf.Write(program.Header.Magic[:])
	binary.Write(&buf, binary.BigEndian, program.Header.Version)
	buf.Write(program.Header.GameTitle[:])
	binary.Write(&buf, binary.BigEndian, program.Header.ReleaseDate)
	binary.Write(&buf, binary.BigEndian, program.Header.ScreenWidth)
	binary.Write(&buf, binary.BigEndian, program.Header.ScreenHeight)
	buf.WriteByte(program.Header.ColorMode)
	buf.Write(program.Header.Reserved[:])

	// Write program metadata
	binary.Write(&buf, binary.BigEndian, program.CRC32)
	binary.Write(&buf, binary.BigEndian, program.Version)
	binary.Write(&buf, binary.BigEndian, program.CheckSum)
	binary.Write(&buf, binary.BigEndian, program.Hardware)

	// Write code
	buf.Write(program.Code)

	_, err := f.Write(buf.Bytes())
	return err
}

// writePaletteROM writes color palette to ROM
func (nc *NEOGEOCompiler) writePaletteROM(f *os.File) error {
	// NEO-GEO palette: 16 colors × 8 palettes = 128 colors total
	// Each color: 16-bit RGB555 format

	paletteCount := 8
	colorsPerPalette := 16

	for p := 0; p < paletteCount; p++ {
		for c := 0; c < colorsPerPalette; c++ {
			// Get color from palette
			idx := (p*colorsPerPalette + c) % len(nc.palette)
			color := nc.palette[idx]

			// Convert RGBA to RGB555
			r := uint16((color.R >> 3) & 0x1F)
			g := uint16((color.G >> 3) & 0x1F)
			b := uint16((color.B >> 3) & 0x1F)
			rgb555 := (r << 10) | (g << 5) | b

			// Write as little-endian 16-bit
			colorBytes := []byte{
				byte(rgb555 & 0xFF),
				byte((rgb555 >> 8) & 0xFF),
			}
			if _, err := f.Write(colorBytes); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeSoundROM writes audio ROM (currently placeholder)
func (nc *NEOGEOCompiler) writeSoundROM(f *os.File) error {
	// NEO-GEO sound ROM: YM2610 sound chip
	// Typical sound ROM: 16-32MB

	// Write placeholder sound data
	soundData := make([]byte, 1024*1024) // 1MB placeholder
	_, err := f.Write(soundData)
	return err
}

// ExportAsPNG exports sprite data as PNG for visualization
func (nc *NEOGEOCompiler) ExportAsPNG(sprite *SpriteData, outputPath string) error {
	// Create 16x16 image
	img := image.NewPaletted(
		image.Rect(0, 0, 16, 16),
		nc.paletteToColorMap(),
	)

	// Fill pixels from sprite data
	for i := 0; i < 256; i++ {
		x := i % 16
		y := i / 16
		paletteIdx := sprite.Data[i] & 0x0F // 4-bit color index
		img.SetColorIndex(x, y, paletteIdx)
	}

	// Write PNG
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// paletteToColorMap converts NEO-GEO palette to color.Palette
func (nc *NEOGEOCompiler) paletteToColorMap() color.Palette {
	p := make(color.Palette, 16)
	for i := 0; i < 16; i++ {
		p[i] = nc.palette[i]
	}
	return p
}

// GetNEOGEOPalette returns the NEO-GEO compatible 16-color palette
func GetNEOGEOPalette() [16]color.RGBA {
	return [16]color.RGBA{
		{0, 0, 0, 255},         // 0: Black
		{255, 255, 255, 255},   // 1: White
		{255, 0, 0, 255},       // 2: Red
		{0, 255, 0, 255},       // 3: Green
		{0, 0, 255, 255},       // 4: Blue
		{255, 255, 0, 255},     // 5: Yellow
		{255, 0, 255, 255},     // 6: Magenta
		{0, 255, 255, 255},     // 7: Cyan
		{128, 0, 0, 255},       // 8: Dark Red
		{0, 128, 0, 255},       // 9: Dark Green
		{0, 0, 128, 255},       // 10: Dark Blue
		{128, 128, 0, 255},     // 11: Dark Yellow
		{128, 0, 128, 255},     // 12: Dark Magenta
		{0, 128, 128, 255},     // 13: Dark Cyan
		{128, 128, 128, 255},   // 14: Gray
		{192, 192, 192, 255},   // 15: Light Gray
	}
}

// CompileToFile compiles game state and writes to output file
func (nc *NEOGEOCompiler) CompileToFile(gameState *GameState, outputFile string) error {
	return nc.CompileGameState(gameState, outputFile)
}

// GetROMInfo returns information about the compiled ROM
func (nc *NEOGEOCompiler) GetROMInfo(romPath string) (map[string]interface{}, error) {
	fileInfo, err := os.Stat(romPath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"filename": filepath.Base(romPath),
		"size":     fileInfo.Size(),
		"modified": fileInfo.ModTime(),
		"format":   "NEO-GEO .bin",
		"hardware": "MVS 1A",
	}, nil
}
