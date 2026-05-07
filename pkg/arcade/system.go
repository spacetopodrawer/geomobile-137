package arcade

// SystemInfo describes an arcade system instance
type SystemInfo struct {
	Name         string // "NEO-GEO", "MAME", "FBNeo", etc.
	SystemID     string // Unique identifier ("neogeo", "mame", "fbneo")
	Description  string
	Version      string       // ROM version
	Manufacturer string       // Hardware manufacturer
	ReleaseYear  int
	Spec         *SystemSpec
	Enabled      bool
	Priority     int // Lower = higher priority
}

// SystemSpec describes hardware specifications of an arcade system
type SystemSpec struct {
	// Display
	DisplayWidth      int
	DisplayHeight     int
	RefreshRate       int // Hz (usually 50 or 60)
	ColorDepth        int // bits per pixel (4, 8, 16, 32)
	MaxColors         int // Total palette size
	DefaultPalette    int // Number of color palettes

	// Input
	Directions       int      // 2 (up/down), 4, 8, etc.
	ActionButtons    int      // Number of action buttons
	SupportedButtons []string // ["button_a", "button_b", ...]

	// CPU & Audio
	CPU              string // "Z80", "6502", "68000", etc.
	AudioChip        string // "YM2610", "SID", "AY-3-8910", etc.
	MaxAudioChannels int

	// ROM format
	ROMFormat       string // ".bin", ".zip", ".rom", etc.
	MinROMSize      int64  // Bytes
	MaxROMSize      int64  // Bytes
	ChecksumType    string // "CRC32", "MD5", "SHA1"

	// Timing
	FrameTime       int // microseconds per frame
	CPUClockHz      int64
	AudioSampleRate int
}

// GetSystemInfo returns info for a given system
func GetSystemInfo(systemID string) *SystemInfo {
	specs := getSystemSpec(systemID)
	if specs == nil {
		return nil
	}

	return &SystemInfo{
		Name:      getSystemName(systemID),
		SystemID:  systemID,
		Spec:      specs,
		Enabled:   true,
		Priority:  getPriority(systemID),
	}
}

func getSystemName(systemID string) string {
	names := map[string]string{
		"neogeo":     "NEO-GEO",
		"mame":       "MAME",
		"fbneo":      "FBNeo",
		"cps1":       "CPS1",
		"cps2":       "CPS2",
		"atari2600":  "Atari 2600",
		"commodore64": "Commodore 64",
	}
	if name, ok := names[systemID]; ok {
		return name
	}
	return "Unknown"
}

func getSystemSpec(systemID string) *SystemSpec {
	specs := map[string]*SystemSpec{
		"neogeo": {
			DisplayWidth:   320,
			DisplayHeight:  224,
			RefreshRate:    60,
			ColorDepth:     4,
			MaxColors:      16,
			DefaultPalette: 8,
			Directions:     8,
			ActionButtons:  4,
			SupportedButtons: []string{"button_a", "button_b", "button_c", "button_d"},
			CPU:            "Motorola 68K",
			AudioChip:      "YM2610",
			MaxAudioChannels: 4,
			ROMFormat:      ".bin",
			MinROMSize:     512 * 1024,
			MaxROMSize:     512 * 1024 * 1024,
			ChecksumType:   "CRC32",
			FrameTime:      16667,
			CPUClockHz:     12000000,
			AudioSampleRate: 18518,
		},
		"mame": {
			DisplayWidth:   320,
			DisplayHeight:  240,
			RefreshRate:    60,
			ColorDepth:     8,
			MaxColors:      256,
			DefaultPalette: 1,
			Directions:     8,
			ActionButtons:  6,
			SupportedButtons: []string{"button_1", "button_2", "button_3", "button_4", "button_5", "button_6"},
			CPU:            "Multiple (varies by ROM)",
			AudioChip:      "Multiple (varies by ROM)",
			MaxAudioChannels: 8,
			ROMFormat:      ".zip",
			MinROMSize:     1024,
			MaxROMSize:     4 * 1024 * 1024 * 1024,
			ChecksumType:   "CRC32",
			FrameTime:      16667,
			CPUClockHz:     10000000,
			AudioSampleRate: 44100,
		},
		"fbneo": {
			DisplayWidth:   320,
			DisplayHeight:  224,
			RefreshRate:    60,
			ColorDepth:     8,
			MaxColors:      256,
			DefaultPalette: 1,
			Directions:     8,
			ActionButtons:  6,
			SupportedButtons: []string{"button_1", "button_2", "button_3", "button_4", "button_5", "button_6"},
			CPU:            "Motorola 68K",
			AudioChip:      "YM2610",
			MaxAudioChannels: 4,
			ROMFormat:      ".zip",
			MinROMSize:     512 * 1024,
			MaxROMSize:     64 * 1024 * 1024,
			ChecksumType:   "CRC32",
			FrameTime:      16667,
			CPUClockHz:     12000000,
			AudioSampleRate: 18518,
		},
		"atari2600": {
			DisplayWidth:   160,
			DisplayHeight:  192,
			RefreshRate:    60,
			ColorDepth:     4,
			MaxColors:      128,
			DefaultPalette: 1,
			Directions:     4,
			ActionButtons:  1,
			SupportedButtons: []string{"button_fire"},
			CPU:            "MOS 6502",
			AudioChip:      "TIA",
			MaxAudioChannels: 2,
			ROMFormat:      ".bin",
			MinROMSize:     2 * 1024,
			MaxROMSize:     32 * 1024,
			ChecksumType:   "CRC32",
			FrameTime:      16667,
			CPUClockHz:     1193182,
			AudioSampleRate: 31400,
		},
		"commodore64": {
			DisplayWidth:   320,
			DisplayHeight:  200,
			RefreshRate:    50,
			ColorDepth:     4,
			MaxColors:      16,
			DefaultPalette: 1,
			Directions:     8,
			ActionButtons:  1,
			SupportedButtons: []string{"button_fire"},
			CPU:            "MOS 6502",
			AudioChip:      "SID",
			MaxAudioChannels: 3,
			ROMFormat:      ".prg",
			MinROMSize:     1024,
			MaxROMSize:     64 * 1024,
			ChecksumType:   "CRC32",
			FrameTime:      20000,
			CPUClockHz:     985248,
			AudioSampleRate: 44100,
		},
	}

	if spec, ok := specs[systemID]; ok {
		return spec
	}
	return nil
}

func getPriority(systemID string) int {
	priorities := map[string]int{
		"neogeo":      1,
		"mame":        2,
		"fbneo":       2,
		"cps1":        3,
		"cps2":        3,
		"atari2600":   3,
		"commodore64": 3,
	}
	if priority, ok := priorities[systemID]; ok {
		return priority
	}
	return 99
}
