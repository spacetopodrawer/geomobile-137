package neogeo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"cadastreia/pkg/arcade"
)

// Compiler handles NEO-GEO ROM compilation
// Implements arcade.ROMCompiler interface
type Compiler struct {
	outputPath string
	palette    [16]color.RGBA
	romSize    int64
}

// NEOGEOROMHeader represents the NEO-GEO ROM file header
type NEOGEOROMHeader struct {
	Magic          [4]byte
	Version        uint32
	GameTitle      [32]byte
	ReleaseDate    uint32
	ScreenWidth    uint16
	ScreenHeight   uint16
	ColorMode      uint8
	Reserved       [259]byte
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
	Data     [256]byte
}

// NEOGEOProgram represents the main program ROM
type NEOGEOProgram struct {
	Header    NEOGEOROMHeader
	CRC32     uint32
	Version   uint32
	CheckSum  uint16
	Hardware  uint16
	Code      []byte
	DataSize  uint32
}

// NewCompiler creates a new NEO-GEO compiler instance
func NewCompiler(outputPath string) *Compiler {
	return &Compiler{
		outputPath: outputPath,
		palette:    GetNEOGEOPalette(),
		romSize:    512 * 1024 * 1024,
	}
}

// Compile converts game state to NEO-GEO ROM format
func (c *Compiler) Compile(gameState *arcade.GameState, outputFile string) error {
	if gameState == nil {
		return fmt.Errorf("game state is nil")
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create ROM file: %w", err)
	}
	defer f.Close()

	// Create header
	header := NEOGEOROMHeader{
		Magic:        [4]byte{'N', 'E', 'O', 'P'},
		Version:      0x01000000,
		ScreenWidth:  320,
		ScreenHeight: 224,
		ColorMode:    4,
	}
	copy(header.GameTitle[:], []byte("CADASTRE_IA v0.1.0"))

	program := &NEOGEOProgram{
		Header:   header,
		CRC32:    0xDEADBEEF,
		Version:  0x01000000,
		CheckSum: 0x0000,
		Hardware: 0x0100,
	}

	bytecode := c.compileGameLogic(gameState)
	program.Code = bytecode
	program.DataSize = uint32(len(bytecode))

	if err := c.writeProgram(f, program); err != nil {
		return fmt.Errorf("failed to write program ROM: %w", err)
	}

	if err := c.writeSpriteROM(f, gameState); err != nil {
		return fmt.Errorf("failed to write sprite ROM: %w", err)
	}

	if err := c.writePaletteROM(f); err != nil {
		return fmt.Errorf("failed to write palette ROM: %w", err)
	}

	if err := c.writeSoundROM(f); err != nil {
		return fmt.Errorf("failed to write sound ROM: %w", err)
	}

	return nil
}

// GetFormat returns the ROM format identifier
func (c *Compiler) GetFormat() string {
	return "neogeo"
}

// GetSpec returns the system specification
func (c *Compiler) GetSpec() *arcade.SystemSpec {
	return arcade.GetSystemSpec("neogeo")
}

// GetROMInfo retrieves metadata about a compiled ROM
func (c *Compiler) GetROMInfo(romPath string) (*arcade.ROMInfo, error) {
	fileInfo, err := os.Stat(romPath)
	if err != nil {
		return nil, err
	}

	return &arcade.ROMInfo{
		Format:      "neogeo",
		Size:        fileInfo.Size(),
		CRC32:       0xDEADBEEF,
		Title:       "NEO-GEO Game",
		ReleaseDate: "2026-05-07",
		Resolution: [2]int{320, 224},
		ColorDepth:  4,
		AudioChip:   "YM2610",
	}, nil
}

// CompileSprites compiles sprite data with palette
func (c *Compiler) CompileSprites(spriteData []byte, paletteID uint8) ([]byte, error) {
	// Simple sprite compilation: prepend palette ID and size
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint16(len(spriteData)))
	buf.WriteByte(paletteID)
	buf.Write(spriteData)
	return buf.Bytes(), nil
}

// CompileAudio compiles audio data
func (c *Compiler) CompileAudio(audioData []byte) ([]byte, error) {
	// YM2610 audio format (placeholder)
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(audioData)))
	buf.Write(audioData)
	return buf.Bytes(), nil
}

// compileGameLogic converts game state to Z80 bytecode
func (c *Compiler) compileGameLogic(gameState *arcade.GameState) []byte {
	var buf bytes.Buffer

	buf.Write([]byte{
		0xF3,             // DI (disable interrupts)
		0x31, 0xFF, 0xDF, // LD SP, 0xDFFF (stack pointer)
		0xCD, 0x00, 0x10, // CALL initialize
		0xC9,             // RET
	})

	buf.WriteString("GAME_INIT:")
	buf.Write([]byte{0x00, 0x00, 0x00})

	return buf.Bytes()
}

// writeProgram writes the program ROM to file
func (c *Compiler) writeProgram(f *os.File, program *NEOGEOProgram) error {
	var buf bytes.Buffer

	buf.Write(program.Header.Magic[:])
	binary.Write(&buf, binary.BigEndian, program.Header.Version)
	buf.Write(program.Header.GameTitle[:])
	binary.Write(&buf, binary.BigEndian, program.Header.ReleaseDate)
	binary.Write(&buf, binary.BigEndian, program.Header.ScreenWidth)
	binary.Write(&buf, binary.BigEndian, program.Header.ScreenHeight)
	buf.WriteByte(program.Header.ColorMode)
	buf.Write(program.Header.Reserved[:])

	binary.Write(&buf, binary.BigEndian, program.CRC32)
	binary.Write(&buf, binary.BigEndian, program.Version)
	binary.Write(&buf, binary.BigEndian, program.CheckSum)
	binary.Write(&buf, binary.BigEndian, program.Hardware)

	buf.Write(program.Code)

	_, err := f.Write(buf.Bytes())
	return err
}

// writeSpriteROM writes graphics data to ROM
func (c *Compiler) writeSpriteROM(f *os.File, gameState *arcade.GameState) error {
	var buf bytes.Buffer

	if gameState.Player != nil {
		sprite := c.createPlayerSprite(gameState.Player)
		if sprite != nil {
			buf.Write(sprite.Data[:])
		}
	}

	for _, obj := range gameState.Objects {
		sprite := c.createObjectSprite(obj)
		if sprite != nil {
			buf.Write(sprite.Data[:])
		}
	}

	_, err := f.Write(buf.Bytes())
	return err
}

// createPlayerSprite generates player sprite data
func (c *Compiler) createPlayerSprite(player *arcade.PlayerState) *SpriteData {
	sprite := &SpriteData{
		TileID:   0,
		X:        int16(player.X),
		Y:        int16(player.Y),
		Width:    2,
		Height:   2,
		Rotation: 0,
		Palette:  1,
	}

	for i := 0; i < 256; i++ {
		sprite.Data[i] = byte((player.Health / 255) * 255)
	}

	return sprite
}

// createObjectSprite generates object sprite data
func (c *Compiler) createObjectSprite(obj *arcade.ObjectState) *SpriteData {
	sprite := &SpriteData{
		TileID:   1,
		X:        int16(obj.X),
		Y:        int16(obj.Y),
		Width:    2,
		Height:   2,
		Rotation: obj.Rotation,
		Palette:  obj.PaletteID,
	}

	for i := 0; i < 256; i++ {
		sprite.Data[i] = byte((i % 16) * 16)
	}

	return sprite
}

// writePaletteROM writes color palette to ROM
func (c *Compiler) writePaletteROM(f *os.File) error {
	var buf bytes.Buffer

	paletteCount := 8
	for p := 0; p < paletteCount; p++ {
		for i := 0; i < 16; i++ {
			color := c.palette[i]
			r := uint16(color.R) >> 3
			g := uint16(color.G) >> 3
			b := uint16(color.B) >> 3
			rgb555 := (r << 10) | (g << 5) | b
			binary.Write(&buf, binary.BigEndian, rgb555)
		}
	}

	_, err := f.Write(buf.Bytes())
	return err
}

// writeSoundROM writes audio data to ROM
func (c *Compiler) writeSoundROM(f *os.File) error {
	var buf bytes.Buffer

	// YM2610 audio placeholder
	soundData := []byte{0x00, 0x00, 0x00, 0x00}
	buf.Write(soundData)

	_, err := f.Write(buf.Bytes())
	return err
}

// ExportAsPNG exports sprite data as PNG image
func (c *Compiler) ExportAsPNG(spriteData *SpriteData, outputPath string) error {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			idx := (y * 16) + x
			colorIdx := spriteData.Data[idx] & 0x0F
			color := c.palette[colorIdx]
			img.SetRGBA(x, y, color)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// GetNEOGEOPalette returns the default NEO-GEO color palette
func GetNEOGEOPalette() [16]color.RGBA {
	return [16]color.RGBA{
		{0, 0, 0, 255},          // Black
		{255, 0, 0, 255},        // Red
		{0, 255, 0, 255},        // Green
		{255, 255, 0, 255},      // Yellow
		{0, 0, 255, 255},        // Blue
		{255, 0, 255, 255},      // Magenta
		{0, 255, 255, 255},      // Cyan
		{192, 192, 192, 255},    // Light gray
		{128, 128, 128, 255},    // Dark gray
		{255, 128, 0, 255},      // Orange
		{128, 255, 0, 255},      // Light green
		{0, 128, 255, 255},      // Sky blue
		{255, 0, 128, 255},      // Rose
		{128, 0, 255, 255},      // Violet
		{0, 255, 128, 255},      // Spring green
		{255, 255, 255, 255},    // White
	}
}
