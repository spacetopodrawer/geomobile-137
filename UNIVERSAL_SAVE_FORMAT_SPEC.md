# Universal Save Format Specification v0.3.0

**Project**: Cadastre_IA - Arcade Emulator Framework  
**Version**: 0.3.0 (Phase 4.5B - Planned)  
**Status**: SPECIFICATION (Pre-Implementation)  
**Last Updated**: May 8, 2026

---

## Overview

The **Universal Save Format** is a system-agnostic game state serialization format enabling:

1. **Cross-ROM Migration** - Load a save from NEO-GEO game in MAME or FBNeo
2. **SVG Asset Compression** - Quick compression/decompression of vector graphics
3. **Backward Compatibility** - Phase 3 saves upgrade automatically
4. **Minimal File Size** - Compressed SVG + DEFLATE for fast I/O
5. **Format Independence** - Works across all 6 supported arcade systems

---

## File Structure

```
Cadastre_IA_Save_v0.3.0.usf (Universal Save Format)
├── [Magic] (4 bytes)           "CIAU" (Cadastre_IA_Arcade_Universal)
├── [Version] (2 bytes)         0x0003 (v0.3.0)
├── [Flags] (2 bytes)           Compression, encryption, system hints
├── [GameState] (variable)      Core game logic state
│   ├── Player state            Position, inventory, status
│   ├── Object states           All objects in scene
│   ├── World metadata          Tilemap, collision, logic
│   └── Progress tracking       Level, score, time
├── [SVGAssets] (variable)      Compressed vector graphics
│   ├── Player sprite SVG       <svg> + compression dict
│   ├── Enemy sprites SVG       <svg> + compression dict
│   ├── Environment SVG         <svg> + compression dict
│   ├── UI elements SVG         <svg> + compression dict
│   └── Effects SVG             <svg> + compression dict
└── [Checksum] (4 bytes)        CRC32 of entire file
```

---

## Magic Header & Versioning

### Magic Bytes

```
Position: 0x00 (4 bytes)
Value: "CIAU" (0x43 0x49 0x41 0x55)
Meaning: Cadastre_IA_Arcade_Universal

Verification:
  if file[0:4] != "CIAU" {
    return error("Not a valid Universal Save Format")
  }
```

### Version Number

```
Position: 0x04 (2 bytes, Big-Endian)
Value: 0x0003 (v0.3.0)
Format: Major (1 byte) + Minor (1 byte)
  0x00 = v0.0.x
  0x01 = v0.1.x
  0x02 = v0.2.x
  0x03 = v0.3.0 (current)

Compatibility:
  Can load: v0.0.0 - v0.3.0
  Cannot load: v1.0.0+ (major version change = breaking)
```

### Flags Byte (Position 0x06)

```
Bit 0: SVG Compression (0=None, 1=DEFLATE)
Bit 1: Encryption (0=None, 1=XOR with device ID)
Bit 2: System Hint (0=Auto-detect, 1=Specific system)
Bit 3: Reserved
Bit 4: Reserved
Bit 5: Reserved
Bit 6: Reserved
Bit 7: Reserved

Example: 0x01 = SVG compression enabled, no encryption
```

---

## Game State Section

### Structure

```json
{
  "gameState": {
    "metadata": {
      "sourceSystem": "neogeo",          // System that created save
      "targetSystem": "mame",            // System to load in
      "timestamp": "2026-05-08T15:30:45Z",
      "saveSlot": 1,
      "playtimeSeconds": 3600
    },
    
    "player": {
      "position": { "x": 160.5, "y": 112.25 },
      "velocity": { "x": 0.0, "y": 0.0 },
      "action": "idle",
      "direction": "right",
      "health": 100,
      "score": 5000
    },
    
    "inventory": [
      { "itemID": "weapon_sword", "quantity": 1, "equipped": true },
      { "itemID": "potion_health", "quantity": 5, "equipped": false }
    ],
    
    "objects": [
      {
        "id": "enemy_001",
        "type": "enemy",
        "position": { "x": 200, "y": 80 },
        "state": "patrol",
        "health": 50
      }
    ],
    
    "world": {
      "level": 1,
      "tilemap": "base64_encoded_compressed_data",
      "collisionMap": "base64_encoded_compressed_data",
      "spawnPoints": [
        { "id": "spawn_player", "x": 160, "y": 224 }
      ]
    }
  }
}
```

### Serialization

```go
// Before compression
gameStateJSON := json.Marshal(gameState)

// DEFLATE compression
compressed := flate.Compress(gameStateJSON)

// Encode as Base64 for JSON storage
encoded := base64.StdEncoding.EncodeToString(compressed)

// Write to USF file
file.Write([]byte(encoded))
```

---

## SVG Assets Section

### SVG Compression Strategy

**Problem**: SVG sprites are text-based and highly compressible, but decompression must be fast for real-time gameplay.

**Solution**: Multi-level compression with dictionary optimization:

```
1. Minification     → Remove whitespace, reduce precision
                    (e.g., "1.23456" → "1.235")
2. Dictionary Ref   → Replace common patterns with tokens
                    (e.g., '<circle cx="' → [0x01])
3. DEFLATE          → Standard compression (Level 6, balance speed vs ratio)
4. Base64 Encode    → Store in JSON-safe format
```

### SVG Asset Structure

```json
{
  "svgAssets": {
    "sprites": [
      {
        "id": "player_sprite",
        "type": "player",
        "originalSize": 2048,
        "compressed": {
          "algorithm": "deflate",
          "level": 6,
          "compressedSize": 512,
          "ratio": 0.25,
          "data": "eJxLTEopTiwBAM8/AJc=",
          "minificationDict": {
            "[0x01]": "<circle cx=\"",
            "[0x02]": "\" cy=\"",
            "[0x03]": "\" r=\"",
            "[0x04]": "<path d=\""
          }
        },
        "checksums": {
          "original": "abc123def456",
          "decompressed": "abc123def456"
        }
      },
      
      {
        "id": "enemy_01",
        "type": "enemy",
        "originalSize": 1536,
        "compressed": {
          "algorithm": "deflate",
          "level": 6,
          "compressedSize": 384,
          "ratio": 0.25,
          "data": "eJxLTEopTiwBAM8/AJc=",
          "minificationDict": { /* ... */ }
        }
      },
      
      {
        "id": "effect_explosion",
        "type": "effect",
        "originalSize": 3072,
        "compressed": {
          "algorithm": "deflate",
          "level": 6,
          "compressedSize": 768,
          "ratio": 0.25,
          "data": "eJxLTEopTiwBAM8/AJc=",
          "minificationDict": { /* ... */ }
        }
      }
    ]
  }
}
```

### Compression Performance

**Target Metrics**:

| Operation | Target | Acceptable |
|-----------|--------|------------|
| Compress SVG | <50ms | <100ms |
| Decompress SVG | <10ms | <20ms |
| Decompress all assets | <100ms | <200ms |
| Load entire save | <500ms | <1000ms |

**Typical Compression Ratios**:

| Asset Type | Original | Compressed | Ratio |
|-----------|----------|-----------|-------|
| Player sprite | 2.0 KB | 0.5 KB | 25% |
| Enemy sprites (5×) | 7.5 KB | 1.9 KB | 25% |
| Environment | 10 KB | 2.5 KB | 25% |
| Effects | 6 KB | 1.5 KB | 25% |
| UI elements | 4 KB | 1.0 KB | 25% |
| **Total** | **29.5 KB** | **7.4 KB** | **25%** |

---

## Decompression Algorithm

### Fast Decompression (Phase 4.5B)

```go
type SVGAsset struct {
    ID                string            `json:"id"`
    Type              string            `json:"type"`
    OriginalSize      int               `json:"originalSize"`
    Compressed        CompressedData    `json:"compressed"`
    Checksums         ChecksumData      `json:"checksums"`
}

type CompressedData struct {
    Algorithm        string            `json:"algorithm"`
    Level            int               `json:"level"`
    CompressedSize   int               `json:"compressedSize"`
    Ratio            float32           `json:"ratio"`
    Data             string            `json:"data"` // Base64
    MinificationDict map[string]string `json:"minificationDict"`
}

func DecompressSVGAsset(asset *SVGAsset) (string, error) {
    // 1. Base64 decode
    compressedBytes, err := base64.StdEncoding.DecodeString(asset.Compressed.Data)
    if err != nil {
        return "", err
    }
    
    // 2. DEFLATE decompress
    decompressed := flate.Decompress(compressedBytes)
    
    // 3. Restore minified patterns from dictionary
    svgText := string(decompressed)
    for token, pattern := range asset.Compressed.MinificationDict {
        svgText = strings.ReplaceAll(svgText, token, pattern)
    }
    
    // 4. Verify checksum
    decompressedHash := md5([]byte(svgText))
    if decompressedHash != asset.Checksums.Decompressed {
        return "", fmt.Errorf("checksum mismatch: expected %s, got %s",
            asset.Checksums.Decompressed, decompressedHash)
    }
    
    return svgText, nil
}

func DecompressAllAssets(save *UniversalSave) error {
    for _, asset := range save.SVGAssets.Sprites {
        svg, err := DecompressSVGAsset(asset)
        if err != nil {
            return err
        }
        // Render or cache SVG
        cache.Store(asset.ID, svg)
    }
    return nil
}
```

---

## File Format Layout (Binary)

```
Offset  Size  Field
------  ----  -----
0x0000  4     Magic ("CIAU")
0x0004  2     Version (0x0003)
0x0006  2     Flags
0x0008  4     GameState size (big-endian uint32)
0x000C  ?     GameState (JSON gzip)
...     4     SVGAssets size (big-endian uint32)
...     ?     SVGAssets (JSON gzip)
...     4     CRC32 checksum
```

### Example Binary Layout

```
00000000: 4349 4155 0003 0001 0000 0100 7b22 6761  |CIAU......{"ga|
00000010: 6d65 5374 6174 6522 3a7b 226d 6574 6164  |meState":{"metad|
...
(GameState compressed JSON data)
...
0000xxxx: 0000 0200 7b22 7376 6741 7373 6574 7322  |...{"svgAssets"|
(SVGAssets compressed JSON data)
...
0000yyyy: 1a2b 3c4d  |....|
(CRC32 checksum as last 4 bytes)
```

---

## System-Specific Compatibility

### Mapping Across Systems

When loading a save from System A into System B:

```
NEO-GEO (320×224, 4-bit, 8-dir+4btn)
  ↓
  Convert player position: (160, 112) → (160, 120) for MAME
  Convert buttons: 4 buttons → 6 buttons (pad unused)
  Rerender SVG sprites: Use stored SVG, render at system resolution
  ↓
MAME (320×240, 8-bit, 8-dir+6btn)
```

### Resolution Scaling

```go
// NEO-GEO → MAME
srcResolution := [2]int{320, 224}
dstResolution := [2]int{320, 240}

// Player at NEO-GEO position
playerX := 160
playerY := 112

// Scale to MAME resolution
scaleX := float32(dstResolution[0]) / float32(srcResolution[0])
scaleY := float32(dstResolution[1]) / float32(srcResolution[1])

newPlayerX := float32(playerX) * scaleX  // 160.0
newPlayerY := float32(playerY) * scaleY  // 120.0 (approx)
```

### SVG Rerendering

```go
// SVG assets are resolution-agnostic (vector)
// Decompress original SVG
svgText, err := DecompressSVGAsset(asset)

// Render at target system resolution
// Example: NEO-GEO SVG → render at 320×224 or 320×240
renderSVG := func(svgText string, targetWidth, targetHeight int) image.Image {
    // Use svg package to render to target dimensions
    return renderToImage(svgText, targetWidth, targetHeight)
}

targetSprite := renderSVG(svgText, 320, 240)  // Render for MAME
```

---

## Backward Compatibility (Phase 3 → Phase 4.5B)

### Phase 3 Save Format

Phase 3 saves are binary NEO-GEO specific format:
```
[NEO-GEO Header] [Game State] [Sprites] [Palette]
```

### Migration Path

```
Phase 3 Save (.neo)
  ↓
1. Parse NEO-GEO specific format
2. Extract player position, inventory, objects
3. Convert sprites from NEO-GEO format to SVG
4. Create new GameState structure
5. Compress with DEFLATE
6. Write as v0.3.0 Universal Save Format (.usf)
  ↓
v0.3.0 Universal Save Format
```

### Automatic Upgrade

```go
func LoadSave(filePath string) (*UniversalSave, error) {
    file, _ := os.Open(filePath)
    magic := make([]byte, 4)
    file.Read(magic)
    
    if string(magic) == "CIAU" {
        // v0.3.0 Universal Save Format
        return loadUniversalSave(file)
    } else if string(magic) == "NEOP" || string(magic) == "NEOS" {
        // Phase 3 NEO-GEO format - auto-upgrade
        return upgradePhase3Save(filePath)
    } else {
        return nil, fmt.Errorf("unknown save format")
    }
}

func upgradePhase3Save(phase3Path string) (*UniversalSave, error) {
    // 1. Load Phase 3 save
    phase3Save := loadPhase3Format(phase3Path)
    
    // 2. Convert sprites to SVG
    svgAssets := convertSpritesToSVG(phase3Save.Sprites)
    
    // 3. Create Universal format
    universalSave := &UniversalSave{
        Magic:     "CIAU",
        Version:   0x0003,
        GameState: phase3Save.GameState,
        SVGAssets: svgAssets,
    }
    
    // 4. Compress and write
    return universalSave, nil
}
```

---

## Checksum & Validation

### File Integrity

```go
type UniversalSave struct {
    Magic       string
    Version     uint16
    Flags       uint16
    GameState   GameState
    SVGAssets   SVGAssetCollection
    FileChecksum uint32  // CRC32 of entire file minus checksum field
}

func CalculateChecksum(save *UniversalSave) uint32 {
    // Serialize everything except checksum
    buffer := serializeUniversalSave(save)
    
    // Calculate CRC32
    crc32Table := crc32.MakeTable(crc32.IEEE)
    return crc32.Checksum(buffer, crc32Table)
}

func ValidateSave(filePath string) error {
    save := loadUniversalSave(filePath)
    
    calculated := calculateChecksum(save)
    if calculated != save.FileChecksum {
        return fmt.Errorf("save file corrupted: checksum mismatch")
    }
    
    return nil
}
```

---

## Encryption (Optional)

### XOR with Device ID

For cross-device saves, optional XOR encryption with device ID:

```go
func EncryptSVGAsset(asset *SVGAsset, deviceID string) error {
    decoded, _ := base64.StdEncoding.DecodeString(asset.Compressed.Data)
    
    // XOR with device ID (repeated)
    for i := 0; i < len(decoded); i++ {
        decoded[i] ^= deviceID[i % len(deviceID)]
    }
    
    asset.Compressed.Data = base64.StdEncoding.EncodeToString(decoded)
    return nil
}
```

**Note**: This is light obfuscation, not cryptographic security. For production, use AES-256.

---

## Implementation Timeline (Phase 4.5B)

### Week 1: Core Format & Compression

- [ ] Implement UniversalSave struct
- [ ] Add DEFLATE compression/decompression
- [ ] Create SVG minification dictionary
- [ ] Add CRC32 validation
- [ ] Write file I/O functions

### Week 2: Cross-System Migration

- [ ] Implement resolution scaling
- [ ] Add SVG rerendering for target systems
- [ ] Test save loading across NEO-GEO → MAME → FBNeo
- [ ] Validate sprite quality post-render

### Week 3: Backward Compatibility

- [ ] Implement Phase 3 save format parser
- [ ] Auto-upgrade Phase 3 → v0.3.0
- [ ] Convert NEO-GEO sprites to SVG
- [ ] Test migration paths

### Week 4: Performance & Testing

- [ ] Benchmark compression ratios
- [ ] Measure decompression times
- [ ] Load testing (100 saves, 1000 assets)
- [ ] Documentation & examples

---

## Example: Complete Save/Load Cycle

### Save Game

```go
// Player completes level 3 on NEO-GEO
gameState := &GameState{
    SourceSystem: "neogeo",
    Player: PlayerState{
        Position: Point{X: 160.5, Y: 112.25},
        Action: "idle",
        Health: 100,
    },
    Objects: []ObjectState{ /* ... */ },
    World: WorldState{ /* ... */ },
}

// Compress SVG sprites
svgAssets := &SVGAssetCollection{
    Sprites: []*SVGAsset{
        // Each asset is DEFLATE compressed + minified
    },
}

// Create Universal Save
save := &UniversalSave{
    Magic: "CIAU",
    Version: 0x0003,
    Flags: 0x01,  // SVG compression enabled
    GameState: gameState,
    SVGAssets: svgAssets,
}

// Write to file
saveFile := "save_slot_1.usf"
save.Write(saveFile)
```

### Load Game (Same System)

```go
// Player loads save on NEO-GEO
save, err := LoadUniversalSave("save_slot_1.usf")
if err != nil {
    return err
}

// Decompress SVG assets (fast, <100ms for all)
err = DecompressAllAssets(save)
if err != nil {
    return err
}

// Load game state
gameState := save.GameState
player := gameState.Player  // Position, inventory, health intact

// Render decompressed sprites at NEO-GEO resolution (320×224)
for _, asset := range cache.GetAll() {
    renderSprite(asset.ID, asset.SVG)
}

game.RestoreState(gameState)
```

### Load Game (Cross-System)

```go
// Player loads Phase 3 NEO-GEO save on MAME
save, err := LoadSaveAnyFormat("save_slot_1.neo")  // Auto-detects format
if err != nil {
    return err
}

// If Phase 3 format, auto-upgrade to v0.3.0
if save.Version < 0x0003 {
    save, err = UpgradePhase3Save(save)
    if err != nil {
        return err
    }
}

// Decompress all SVG assets
err = DecompressAllAssets(save)

// Scale player position from NEO-GEO (320×224) → MAME (320×240)
save.GameState.Player.Position.X *= 320.0 / 320.0  // 1:1
save.GameState.Player.Position.Y *= 240.0 / 224.0  // Scale Y

// Rerender SVG sprites at MAME resolution
for _, asset := range save.SVGAssets.Sprites {
    mameResolution := [2]int{320, 240}
    sprite := renderSVGAtResolution(asset.SVG, mameResolution)
    cache.Store(asset.ID, sprite)
}

// Load state and continue playing on MAME
game.RestoreState(save.GameState)
```

---

## Performance Targets

### Compression

| Operation | Target | Notes |
|-----------|--------|-------|
| Compress 1 SVG sprite | <50ms | DEFLATE level 6 + dict |
| Compress all 5 enemy sprites | <150ms | Parallel possible |
| Compress environment | <30ms | Larger file, DEFLATE fast |
| **Total save game** | **<500ms** | Player sees "Saving..." message |

### Decompression

| Operation | Target | Notes |
|-----------|--------|-------|
| Decompress 1 SVG sprite | <10ms | Sub-frame, fast DEFLATE |
| Decompress all sprites (10×) | <100ms | Parallel possible |
| Validate checksums | <20ms | CRC32 fast |
| **Total load game** | **<500ms** | Player sees "Loading..." screen |

### File Size

| Content | Original | Compressed | Ratio |
|---------|----------|-----------|-------|
| GameState JSON | 8 KB | 2 KB | 25% |
| 10 SVG sprites | 20 KB | 5 KB | 25% |
| World tilemap | 12 KB | 3 KB | 25% |
| **Total save file** | **40 KB** | **10 KB** | **25%** |

---

## Related Files

- [ARCADE_FRAMEWORK.md](./ARCADE_FRAMEWORK.md) - Framework overview
- [RELEASE_NOTES.md](./RELEASE_NOTES.md) - v0.3.0 features and roadmap
- [API_ARCADE.md](./API_ARCADE.md) - HTTP endpoints for save/load

---

## Version History

### v0.3.0 (Planned - Phase 4.5B)
- Universal Save Format specification
- SVG compression/decompression
- Cross-system migration support
- Phase 3 backward compatibility

### v0.2.0 (Current - Phase 3)
- NEO-GEO specific binary format
- Not compatible with other systems

### v0.1.0
- Early phase save mechanism

---

## Future Enhancements (v0.4.0+)

- [ ] AES-256 encryption for sensitive saves
- [ ] Cloud save synchronization
- [ ] Save file versioning/branching
- [ ] Replay recording (input-based)
- [ ] Mod support (custom asset injection)
- [ ] Save state snapshots (quicksave/quickload)
