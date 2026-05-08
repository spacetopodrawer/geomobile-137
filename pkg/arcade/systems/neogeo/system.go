package neogeo

import (
	"cadastreia/pkg/arcade"
)

// System provides NEO-GEO system information and factories
type System struct {
	name string
	id   string
	spec *arcade.SystemSpec
}

// GetSystem returns NEO-GEO system information
func GetSystem() *arcade.SystemInfo {
	return &arcade.SystemInfo{
		Name:         "NEO-GEO",
		SystemID:     "neogeo",
		Description:  "SNK NEO-GEO arcade system",
		Version:      "v0.2.0",
		Manufacturer: "SNK",
		ReleaseYear:  1990,
		Spec: &arcade.SystemSpec{
			DisplayWidth:      320,
			DisplayHeight:     224,
			RefreshRate:       60,
			ColorDepth:        4,
			MaxColors:         16,
			DefaultPalette:    8,
			Directions:        8,
			ActionButtons:     4,
			SupportedButtons:  []string{"button_a", "button_b", "button_c", "button_d"},
			CPU:               "Motorola 68K",
			AudioChip:         "YM2610",
			MaxAudioChannels:  4,
			ROMFormat:         ".bin",
			MinROMSize:        512 * 1024,
			MaxROMSize:        512 * 1024 * 1024,
			ChecksumType:      "CRC32",
			FrameTime:         16667,
			CPUClockHz:        12000000,
			AudioSampleRate:   18518,
		},
		Enabled:  true,
		Priority: 1,
	}
}

// RegisterNEOGEOSystem registers NEO-GEO in the arcade framework
func RegisterNEOGEOSystem() error {
	spec := GetSystem().Spec
	return arcade.RegisterSystem("neogeo", Factory, spec)
}

// GetCompiler creates a new NEO-GEO ROM compiler
func GetCompiler(outputPath string) *Compiler {
	return NewCompiler(outputPath)
}

// GetController creates a new NEO-GEO controller mapper
func GetController() *Controller {
	return NewController()
}
