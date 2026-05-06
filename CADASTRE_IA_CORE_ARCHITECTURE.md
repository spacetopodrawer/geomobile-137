# Cadastre_IA Core Architecture
**Version:** 3.0 Native Autonomous  
**Status:** 🚀 Development Phase (2026-05-05)  
**Objective:** Vector-based geospatial game engine with real-time sensor fusion

---

## 📋 System Overview

```
SENSOR INPUT LAYER
├─ GNSS (GPS/positioning)
├─ IMU (motion/orientation)
├─ Photogrammetry (3D meshes)
├─ Drone telemetry (surveys)
├─ Camera (visual recognition)
└─ Custom sensors (extensible)
        │
        ▼
VECTOR OBJECT LAYER (Core Data Model)
├─ Standardized encapsulation
├─ Geometry + Metadata fusion
├─ Temporal versioning
└─ Conflict resolution (OT)
        │
    ┌───┴───┬───────────┬──────────┐
    ▼       ▼           ▼          ▼
ARCADE  WEB UI      3D VIEWER   MOBILE
(Neo-Geo) (React)  (Babylon)   (WebGL)
    │       │           │          │
    └───────┴───────────┴──────────┘
            │
            ▼
    SYNC ENGINE (P2P WiFi)
    ├─ Real-time replication
    ├─ Conflict detection
    ├─ Change streaming
    └─ Multi-device coordination
```

---

## 🏗️ Core Components to Build

### 1. Vector Object Schema (Data Model)
- Complete GeoJSON + sensor fusion
- Protobuf serialization (compact)
- SQLite persistence (embedded)
- Version control (CRDT-ready)

### 2. Sync Engine
- P2P WebSocket (no server required)
- Operational Transform (conflict resolution)
- Change tracking (event log)
- Bandwidth optimized (delta sync)

### 3. Converters
- `sensor → vector` (data normalization)
- `vector → sprite` (arcade rendering)
- `vector → 3D` (glTF export)
- `3D → arcade` (reverse projection)

### 4. Renderers
- **Arcade:** 256×224 @ 16 colors, 60 FPS
- **Vector:** SVG/Canvas (web)
- **3D:** glTF/Babylon.js (viewers)

### 5. Game Engine
- Joystick input handling (8-dir)
- State machine (exploration, scanning, editing)
- Real-time sensor overlay
- Multi-player synchronization

### 6. Storage
- SQLite (local, embedded)
- Binary blob storage (meshes, textures)
- Append-only event log (audit trail)
- Auto-backup + versioning

---

## 📦 Vector Object Structure

```go
type VectorObject struct {
    // Identity
    ID              uuid.UUID         `json:"id"`
    Type            ObjectType        `json:"type"` // parcel, building, tree, etc.
    Name            string            `json:"name"`
    
    // Geometry
    Geometry        GeoJSONGeometry   `json:"geometry"`
    CoordinateFrame CoordFrame        `json:"coordinate_frame"` // WGS84, etc.
    Accuracy        float32           `json:"accuracy_meters"`
    
    // Sensor Fusion Data
    SensorData      SensorDataBundle  `json:"sensor_data"`
    ExtractedAt     time.Time         `json:"extracted_at"`
    
    // Properties & Metadata
    Properties      map[string]interface{} `json:"properties"`
    Tags            []string          `json:"tags"`
    Owner           string            `json:"owner"`
    
    // Rendering Hints
    RenderStyle     RenderStyle       `json:"render_style"`
    ArcadeSprite    *ArcadeSprite     `json:"arcade_sprite,omitempty"`
    ThreeDModel     *ThreeDModel      `json:"3d_model,omitempty"`
    
    // Versioning
    Version         int32             `json:"version"`
    CreatedAt       time.Time         `json:"created_at"`
    ModifiedAt      time.Time         `json:"modified_at"`
    LastModifiedBy  string            `json:"last_modified_by"`
    
    // Sync Metadata
    SyncID          string            `json:"sync_id"` // For P2P sync
    LastSyncAt      time.Time         `json:"last_sync_at"`
    IsDeleted       bool              `json:"is_deleted"` // Soft delete
}

type SensorDataBundle struct {
    GNSS            *GNSSData         `json:"gnss,omitempty"`
    IMU             *IMUData          `json:"imu,omitempty"`
    Photogrammetry  *PhotogramData    `json:"photogram,omitempty"`
    DroneData       *DroneData        `json:"drone,omitempty"`
    Camera          *CameraData       `json:"camera,omitempty"`
    CustomSensors   map[string]interface{} `json:"custom,omitempty"`
}

type GNSSData struct {
    Position        [3]float64        // [lat, lon, alt]
    Accuracy        float32           // meters
    SatelliteCount  int32
    HDOP            float32
    Timestamp       time.Time
}

type IMUData struct {
    Orientation     [3]float32        // [pitch, roll, yaw]
    Acceleration    [3]float32        // [x, y, z] m/s²
    AngularVelocity [3]float32        // [x, y, z] rad/s
    Timestamp       time.Time
}

type PhotogramData struct {
    MeshURL         string            // Local or remote
    TextureURL      string
    PointCloudURL   string
    BoundingBox     BBox3D
    Scale           float32
    Confidence      float32           // 0-1
    ProcessedAt     time.Time
}

type DroneData struct {
    OrthoGeoTIFF    []byte            // Binary raster
    PointCloud      []XYZPoint        // Sparse or dense
    FlightAltitude  float32
    CapturedAt      time.Time
}

type CameraData struct {
    ImageURL        string
    ExifData        map[string]string
    FeaturePoints   []FeaturePoint    // Detected corners/edges
    CapturedAt      time.Time
}

// Rendering specifications
type RenderStyle struct {
    ArcadeColor     uint8             // 0-255 (palette index)
    VectorFill      string            // CSS color or pattern
    VectorStroke    string
    VectorOpacity   float32           // 0-1
    AnimationFrames int32
}

type ArcadeSprite struct {
    Data            []byte            // Raw sprite bitmap
    Width           int32
    Height          int32
    PaletteIndex    uint8
    Collision       [][]bool          // Collision map
}

type ThreeDModel struct {
    GLTFBinary      []byte            // Embedded glTF
    MeshURL         string
    TextureURL      string
    Material        PBRMaterial
    Scale           [3]float32
}
```

---

## 🔄 Sync Engine Protocol

```
WebSocket Message Format:

{
  "type": "sync_operation" | "sync_request" | "sync_response" | "conflict",
  "version": 1,
  "timestamp": 1714867200000,
  
  // For sync_operation
  "operation": {
    "id": "op-uuid",
    "type": "create" | "update" | "delete",
    "object_id": "obj-uuid",
    "before_state": {...},
    "after_state": {...},
    "vector_clock": {"device-1": 5, "device-2": 3}
  },
  
  // For conflict
  "conflict": {
    "object_id": "obj-uuid",
    "operation_a": {...},
    "operation_b": {...},
    "resolution": "merge" | "operational_transform" | "user_choice"
  }
}

Real-time sync ensures:
✓ <100ms latency between devices
✓ Automatic conflict resolution (OT)
✓ Offline capability (queue + sync on reconnect)
✓ Bandwidth optimized (delta encoding)
✓ No central server required (P2P over WiFi)
```

---

## 🎮 Neo-Geo Arcade Game Flow

```
┌─ GAME START ─┐
│              │
│  Load ROM    │
│  Initialize  │
│  Game State  │
│              ▼
│    ┌─────────────────────────────┐
│    │ MAIN MENU                   │
│    │ • New Game                  │
│    │ • Continue                  │
│    │ • Sync Status               │
│    │ • Settings                  │
│    └─────────────────────────────┘
│              │
│              ▼
│    ┌─────────────────────────────┐
│    │ GAME WORLD (Exploration)    │
│    │                             │
│    │ [Vector objects rendered]   │
│    │ [Real sensor data overlay]  │
│    │ [Multi-player markers]      │
│    │                             │
│    │ Controls:                   │
│    │  Joystick = Move            │
│    │  Button A = Scan/Interact   │
│    │  Button B = Zoom Map        │
│    │  Button C = Inventory       │
│    └─────────────────────────────┘
│              │
│    ┌────┬────┴────┬────┐
│    ▼    ▼         ▼    ▼
│  SCAN EDIT ZOOM SYNC
│    │    │        │    │
│    └────┴────────┴────┘
│         │
│         ▼
│    [Upload to backend]
│    [Sync with other players]
│         │
└─────────┴─────────────────────────→ GAME LOOP (60 FPS)

Game State:
• Player position (lat/lon)
• Viewport (visible objects)
• Inventory (collected data)
• Network status
• FPS counter
```

---

## 📊 Data Flow Example

### Scenario: User scans a parcel with drone

```
1. SENSOR INPUT
   Drone captures: [Image, GPS, Altitude, Timestamp]
   ↓
2. NORMALIZATION (sensor → vector)
   Raw data → VectorObject
   {
     type: "parcel",
     geometry: GeoJSON from drone GPS
     photogrammetry: mesh + texture,
     drone_data: point cloud,
     extracted_at: now()
   }
   ↓
3. LOCAL STORAGE
   SQLite INSERT + Event Log
   ↓
4. SYNC BROADCAST
   WebSocket: "sync_operation" {create, parcel, vector_data}
   ↓
5. REMOTE DEVICES
   PC 2 receives → Renders on web UI
   PC 3 receives → Renders in 3D viewer
   ↓
6. ARCADE RENDERING
   Vector → Sprite (arcade renderer)
   Displays in Neo-Geo ROM with real sensor data
   ↓
7. PERSISTENCE
   Event logged for audit trail
   Versioning incremented
   Timestamp recorded
```

---

## 🎯 Optimization Targets

### Performance
- **ROM size:** <512 KB (single ROM file)
- **Latency:** <100ms sync (WiFi)
- **Memory:** <50 MB runtime (both devices)
- **CPU:** <30% on Raspberry Pi
- **FPS:** 60 stable (arcade)

### Reliability
- **Uptime:** 99.9% (no external dependencies)
- **Data integrity:** ACID (SQLite)
- **Sync accuracy:** 100% (OT + event log)
- **Offline duration:** Unlimited (queue)

### Compatibility
- **Windows/Linux/Mac:** Native binary
- **Arcade:** NeoRageX5, MAME
- **Mobile:** iOS/Android (WebGL wrapper)
- **Browsers:** Modern (Chrome, Firefox, Safari)

---

## 🚀 Implementation Phases

### Phase 1: Core Data Model ✓ (This Night)
- [ ] Vector Object struct (Go)
- [ ] SQLite schema
- [ ] Protobuf definitions
- [ ] Serialization/deserialization

### Phase 2: Sync Engine (This Night)
- [ ] WebSocket P2P implementation
- [ ] Operational Transform (conflict resolution)
- [ ] Change tracking
- [ ] Replication logic

### Phase 3: Converters (Early Morning)
- [ ] Sensor → Vector normalizer
- [ ] Vector → Sprite renderer
- [ ] Vector → 3D (glTF) exporter
- [ ] 3D → Arcade reverse projection

### Phase 4: Game Engine (Morning)
- [ ] Input handler (joystick)
- [ ] Game loop (60 FPS)
- [ ] State machine
- [ ] UI overlay system

### Phase 5: ROM Compiler (Morning)
- [ ] Arcade sprite assembler
- [ ] ROM binary generator
- [ ] Emulator compatibility (NeoRageX5)

### Phase 6: Integration (Tomorrow)
- [ ] End-to-end testing
- [ ] Multi-device synchronization
- [ ] Sensor adaptation (real GNSS, IMU, drone)
- [ ] Performance optimization

---

## 📁 Project Structure (To Create)

```
cadastre-ia-core/
├── cmd/
│   ├── server/main.go
│   └── rom-compiler/main.go
├── pkg/
│   ├── model/
│   │   ├── vector.go
│   │   ├── sensor.go
│   │   └── schema.go
│   ├── storage/
│   │   ├── sqlite.go
│   │   └── migrations/
│   ├── sync/
│   │   ├── websocket.go
│   │   ├── ot.go
│   │   └── replicator.go
│   ├── render/
│   │   ├── arcade.go
│   │   ├── vector.go
│   │   └── 3d.go
│   ├── convert/
│   │   ├── sensor_to_vector.go
│   │   ├── vector_to_sprite.go
│   │   └── vector_to_3d.go
│   └── game/
│       ├── engine.go
│       ├── input.go
│       └── state.go
├── proto/
│   ├── vector_object.proto
│   ├── sync_message.proto
│   └── game_state.proto
├── migrations/
│   └── schema.sql
├── rom/
│   ├── sprites/
│   ├── audio/
│   └── build/
├── tests/
├── docs/
└── go.mod
```

---

## 🔧 Key Technologies Selected

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.21 | Native performance, cross-platform, goroutines |
| Storage | SQLite | Embedded, zero setup, ACID, fast |
| Sync | Custom OT | Conflict-free, peer-to-peer, offline-capable |
| Serialization | Protobuf | Compact, fast, strongly typed |
| Arcade | Custom ROM | Full control, optimal size, emulator-agnostic |
| 3D Export | glTF | Standard, lightweight, widely supported |
| Web Frontend | React+WebGL | Modern, responsive, real-time capable |

---

**Next Morning:** Code implementation ready for integration and testing!

