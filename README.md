# CADASTRE_IA: geo-mobile (Instance 137)

**Family:** geo-mobile  
**Instance:** 137  
**Status:** 🟢 Production Ready  
**Version:** v0.2.0-arcade-137  
**Release Codename:** NEO-GEO Arcade Emulation  
**Date:** 2026-05-06

## 🚀 Quick Start

### Prerequisites
- Docker (tested with version 29.4.1)
- Docker Compose (tested with v5.1.3)
- Bash or PowerShell

### Run Everything in One Command

```bash
bash START_EVERYTHING.sh
```

This will:
1. ✅ Start PostgreSQL + Redis + PgAdmin (Docker)
2. ✅ Build the Go server
3. ✅ Run API endpoint tests
4. ✅ Verify database connectivity
5. ✅ Generate execution report

### Services Available After Launch

| Service | URL | Credentials |
|---------|-----|-------------|
| Backend API | http://localhost:8080 | N/A |
| Health Check | http://localhost:8080/health | N/A |
| WebSocket | ws://localhost:8080/ws | N/A |
| PostgreSQL | localhost:5432 | user: `cadastreia`, pass: `cadastreia_secure_pwd` |
| Redis | localhost:6379 | N/A |
| PgAdmin | http://localhost:5050 | admin@cadastreia.local / admin |

## 📊 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register user
- `POST /api/v1/auth/login` - Login user

### Parcels
- `GET /api/v1/parcels` - List all parcels
- `POST /api/v1/parcels` - Create parcel
- `GET /api/v1/parcels/:id` - Get parcel details
- `PUT /api/v1/parcels/:id` - Update parcel
- `DELETE /api/v1/parcels/:id` - Delete parcel

### Real-time Sync
- `GET /ws` - WebSocket connection for real-time synchronization

## 🧪 Running Individual Tests

```bash
# Lightweight infrastructure test (no build)
bash run-tests-lite.sh

# Full test suite
bash START_EVERYTHING.sh
```

## 🔍 Monitoring

### Check Docker containers
```bash
docker ps
```

### View PostgreSQL logs
```bash
docker logs cadastreia_postgres
```

### Connect to PostgreSQL
```bash
docker exec -it cadastreia_postgres psql -U cadastreia -d cadastreia
```

### Stop all services
```bash
docker-compose down -v
```

## 📁 Project Structure

```
geo-mobile-137/
├── cmd/
│   ├── server/           # Server orchestrator with arcade init
│   └── test/             # Integration and arcade test suites
├── pkg/
│   ├── arcade/           # Phase 3: NEO-GEO emulation (ROM compiler, input mapping, emulator)
│   ├── game/             # 60 FPS game loop
│   ├── storage/          # SQLite3 with WAL
│   ├── sync/             # Vector Clock + Operational Transform + WebSocket
│   ├── convert/          # Sensor-to-vector + Vector-to-arcade rendering
│   └── model/            # Data structures
├── migrations/           # SQLite schema
├── config.yaml           # Project configuration
├── docker-compose.yml    # Docker Compose configuration (optional)
├── Dockerfile            # Go server container image (optional)
├── go.mod, go.sum       # Go dependencies
├── server.exe            # Compiled binary (19.5+ MB)
└── docs/                 # Documentation (PHASE_3_*, README_*, etc.)
```

## ✅ Expected Test Results

After running `bash START_EVERYTHING.sh`:

```
✅ TESTS PASSED: 5/5
  - Health Check: ✅
  - Register User: ✅
  - Login User: ✅
  - List Parcels: ✅
  - Create Parcel: ✅

Status:
  PostgreSQL: ✅ Running on localhost:5432
  Redis: ✅ Running on localhost:6379
  PgAdmin: ✅ Available on http://localhost:5050
  Go Server: ✅ Running on http://localhost:8080
  WebSocket: ✅ Ready on ws://localhost:8080/ws
```

## 🎮 Next Steps - Neo-Geo Game Integration

Once testing is successful, we will:
1. Create ROM wrapper for arcade/Neo-Geo emulator (NeoRageX5)
2. Implement real-time data sync between game and backend
3. Support multi-player gameplay with shared cadastral data
4. Deploy as playable arcade game

## 📞 Support

For issues:
1. Check Docker is running: `docker --version`
2. Review logs: `cat logs/EXECUTION_TIMESTAMP.txt`
3. Verify ports are available: `netstat -an | grep 8080` (Windows/Linux)
4. Restart services: `docker-compose down -v && bash START_EVERYTHING.sh`

---

**Created:** 2026-05-05  
**Solo Mode:** Local testing without network dependency  
**Status:** 🟢 Ready for execution
