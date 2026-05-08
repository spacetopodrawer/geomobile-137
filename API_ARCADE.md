# Arcade Emulator Framework - HTTP API

**Base URL**: `http://localhost:8080`  
**WebSocket**: `ws://localhost:8080/ws`  
**Version**: v0.3.0

## Overview

The Arcade Emulator Framework exposes REST and WebSocket endpoints for system status monitoring, game frame retrieval, and P2P synchronization.

## Endpoints

### Health Check

**GET** `/health`

Returns basic server health status.

**Response** (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2026-05-08T15:30:45Z"
}
```

---

### System Status

**GET** `/status`

Returns comprehensive system status including arcade emulator state.

**Response** (200 OK):
```json
{
  "status": {
    "sync": {
      "connected_peers": 2,
      "pending_operations": 5,
      "last_sync": "2026-05-08T15:30:40Z"
    },
    "game": {
      "running": true,
      "objects_count": 12,
      "player_position": [160, 112],
      "inventory_items": 3
    },
    "storage": {
      "total_rows": 1250,
      "query_time_ms": 15
    },
    "arcade": {
      "running": true,
      "frame_count": 54000,
      "fps": 60,
      "connected_clients": 1,
      "system": "neogeo"
    }
  }
}
```

**Fields**:
- `arcade.running` - Emulator active (boolean)
- `arcade.frame_count` - Total frames rendered since start (uint64)
- `arcade.fps` - Target frames per second (int, typically 60)
- `arcade.connected_clients` - Number of connected P2P peers (int)
- `arcade.system` - Active arcade system ID (string, e.g., "neogeo")

---

### System Statistics

**GET** `/stats`

Returns aggregated statistics across all subsystems.

**Response** (200 OK):
```json
{
  "connected_devices": 2,
  "arcade_clients": 1,
  "game_objects": 12,
  "player_inventory": 3,
  "sync_ops": 47,
  "timestamp": "2026-05-08T15:30:45Z"
}
```

**Fields**:
- `connected_devices` - Total WebSocket connections (int)
- `arcade_clients` - Emulator peer connections (int)
- `game_objects` - Objects in current game state (int)
- `player_inventory` - Items in player inventory (int)
- `sync_ops` - Total sync operations completed (int)
- `timestamp` - UTC timestamp (RFC3339)

---

### Arcade Emulator Status

**GET** `/arcade/status`

Detailed arcade emulator statistics specific to active system.

**Response** (200 OK):
```json
{
  "running": true,
  "frame_count": 54000,
  "fps": 60,
  "connected_clients": 1,
  "system": "neogeo"
}
```

**Fields**:
- `running` - Emulator active (boolean)
- `frame_count` - Frames rendered since startup (uint64)
- `fps` - Frames per second (int)
- `connected_clients` - Connected P2P peers (int)
- `system` - System ID ("neogeo", "mame", "fbneo", etc.)

**Notes**:
- Endpoint only available if `config.arcade.enabled = true`
- Returns active system's status, not all registered systems
- Frame counter resets on emulator restart

---

### WebSocket Connection (P2P Sync)

**WS** `/ws`

Bidirectional WebSocket for peer-to-peer game state synchronization.

**Message Format** (JSON):
```json
{
  "type": "game_frame|sync_op|device_info",
  "deviceID": "local-device-0",
  "data": { /* message-specific */ },
  "timestamp": "2026-05-08T15:30:45Z"
}
```

#### Message Types

**game_frame**: Current game state from arcade emulator
```json
{
  "type": "game_frame",
  "deviceID": "local-device-0",
  "data": {
    "frameID": 54000,
    "playerState": {
      "x": 160,
      "y": 112,
      "action": "idle"
    },
    "objects": [
      { "id": 1, "x": 100, "y": 50, "type": "enemy" }
    ],
    "sourceDevice": "neogeo"
  },
  "timestamp": "2026-05-08T15:30:45Z"
}
```

**sync_op**: Synchronization operation (Vector Clock, OT)
```json
{
  "type": "sync_op",
  "deviceID": "remote-device-1",
  "data": {
    "operation": "set_player_position",
    "path": ["player", "x"],
    "value": 165,
    "vectorClock": {
      "local-device-0": 54000,
      "remote-device-1": 12345
    },
    "timestamp": "2026-05-08T15:30:44Z"
  }
}
```

**device_info**: Peer identification and capabilities
```json
{
  "type": "device_info",
  "deviceID": "remote-device-1",
  "data": {
    "name": "Player 2",
    "capabilities": ["game_frame", "sync_op"],
    "arcadeSystem": "neogeo"
  }
}
```

#### Events

**on open**: Connected to sync hub
```
Server sends device_info confirming connection
```

**on message**: Receive game frame or sync operation
```
Client processes message and applies to local state
```

**on close**: Peer disconnected
```
Client updates peer list and resync if needed
```

---

## Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Bad request (invalid parameters) |
| 404 | Endpoint not found |
| 500 | Server error |

---

## Example Usage

### Get Arcade Status (cURL)

```bash
curl http://localhost:8080/arcade/status
```

```json
{
  "running": true,
  "frame_count": 54000,
  "fps": 60,
  "connected_clients": 1,
  "system": "neogeo"
}
```

### Get Full Status (cURL)

```bash
curl http://localhost:8080/status | jq '.status.arcade'
```

### Connect to WebSocket (wscat)

```bash
wscat -c ws://localhost:8080/ws
```

Send device info:
```json
{"type":"device_info","deviceID":"player-1","data":{"name":"Local Player"}}
```

Receive game frames:
```json
{"type":"game_frame","deviceID":"arcade-1","data":{...}}
```

### Monitor Real-Time Stats

```bash
while true; do
  curl -s http://localhost:8080/stats | jq '.'
  sleep 1
done
```

---

## Configuration

Arcade API behavior controlled in `config.yaml`:

```yaml
server:
  port: 8080

arcade:
  enabled: true
  systems:
    neogeo:
      enabled: true
      port: 9001
```

## Performance Notes

- Frame generation: <16ms (60 FPS)
- Status endpoint: <5ms (simple lookup)
- Stats endpoint: <10ms (aggregation)
- WebSocket latency: <100ms (P2P network dependent)

## Error Handling

**Missing arcade system**:
```
GET /arcade/status (if arcade.enabled = false)
→ 404 Not Found
```

**Invalid WebSocket message**:
```json
{"type":"invalid_type"}
```
Server ignores invalid messages and waits for next valid message.

**Emulator crash**:
```
GET /arcade/status (if emulator stopped)
→ "running": false
→ "frame_count": 54000 (frozen at last value)
```

---

## Versioning

API versioning follows semantic versioning:
- **v0.3.0**: Current stable release
- **Breaking changes**: Major version increment (v1.0.0)
- **New endpoints**: Minor version increment (v0.4.0)
- **Bug fixes**: Patch version increment (v0.3.1)

Current endpoints will not break through v0.3.x releases.

---

## Rate Limiting

No rate limiting implemented. Production deployments should add:
- Per-IP rate limits on HTTP endpoints
- WebSocket connection limits per origin
- Memory limits for arcade emulator instances

---

## See Also

- [ARCADE_FRAMEWORK.md](./ARCADE_FRAMEWORK.md) - Framework architecture
- [RELEASE_NOTES.md](./RELEASE_NOTES.md) - v0.3.0 changes
- [config.yaml](./config.yaml) - Configuration reference
