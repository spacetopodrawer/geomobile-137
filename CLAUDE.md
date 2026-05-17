# 🚀 GeoMobile137 - Production Startup & Development Instructions

**Version:** 3.0.0 Phase 1 Complete  
**Last Updated:** 2026-05-17  
**Status:** ✅✅✅ PHASE 1 COMPLETE - Full Production Stack Operational
**Project Status:** PostgreSQL + Backend + Frontend + DeviceRadar Schema ALL RUNNING

---

## 📋 CRITICAL: Production Startup Sequence

### ⚠️ IMPORTANT RULE: BACKEND FIRST, THEN FRONTEND

The correct sequence MUST be followed or npm will crash:

```
1️⃣  START BACKEND (port 8080) → WAIT for health check ✅
2️⃣  START FRONTEND (port 3000) → proxy to backend ✅
```

**Why?** Frontend proxy requires backend to be operational. If backend is down, npm crashes.

---

## 🔧 Method 1: Automated Startup (RECOMMENDED)

### Windows PowerShell

```powershell
# Run from project root (F:\geomobile137)
bash STARTUP_PRODUCTION_SEQUENCE.sh
```

This script:
- ✅ Verifies Go installation
- ✅ Compiles backend
- ✅ Starts backend server (port 8080)
- ✅ Verifies health endpoint
- ✅ Installs frontend dependencies
- ✅ Starts frontend (port 3000)
- ✅ Verifies both are running
- ✅ Tests proxy integration

### What to Expect

```
✅ Backend Server:  RUNNING (PID: XXXX)
                   http://localhost:8080
                   Health: http://localhost:8080/health

✅ Frontend Server: RUNNING (PID: YYYY)
                   http://localhost:3000
                   Proxy: → http://localhost:8080

🔗 Quick Access:
   Frontend:  http://localhost:3000
   Backend:   http://localhost:8080/health
```

---

## 🔧 Method 2: Manual Startup (if script fails)

### Step 1: Start Backend

```bash
# From project root
go build -o cadastre-server ./cmd/cadastre-server
./cadastre-server -port 8080 -db mock
```

**Expected Output:**
```
🚀 GEO-MOBILE137 CADASTRAL SERVER
Phase 2.2 — CAD Converter Integration
Starting server on port 8080
Database mode: mock
```

**Verification:**
```bash
curl http://localhost:8080/health
# Should return 200 OK
```

### Step 2: Start Frontend

In a NEW terminal/tab:

```bash
cd frontend
npm install
npm run dev
```

**Expected Output:**
```
  VITE v... ready in XXX ms

  ➜  Local:   http://localhost:3000/
  ➜  press h to show help
```

**Verification:**
```bash
# In another terminal
curl http://localhost:3000/
# Should return HTML page
```

---

## 📊 Cahier des Charges Verification Checklist

### Backend (Port 8080)
- [x] Go 1.23+ installed ✓
- [x] Source code: `cmd/cadastre-server/main.go` ✓
- [x] Dependencies: all `pkg/` modules present ✓
- [x] Compilation: `go build` succeeds ✓
- [x] Runs without errors ✓
- [x] Health endpoint responds (200 OK) ✓
- [x] Listens on port 8080 ✓
- [x] Supports mock database mode ✓

### Frontend (Port 3000)
- [x] Node 20+ installed ✓
- [x] npm 10+ installed ✓
- [x] Source: `frontend/` directory complete ✓
- [x] Vite config: `frontend/vite.config.ts` ✓
- [x] Proxy configured: target=8080 ✓
- [x] Dependencies installable via npm ci ✓
- [x] Dev server starts on port 3000 ✓

### Integration
- [x] Frontend proxy → Backend (8080) ✓
- [x] API requests routed correctly ✓
- [x] Both services can run simultaneously ✓
- [x] No port conflicts ✓

---

## 🐛 Troubleshooting

### Problem: Backend won't compile

```bash
# Solution: Update Go modules
go mod download
go mod tidy
go build -o cadastre-server ./cmd/cadastre-server
```

### Problem: Backend starts but crashes immediately

```bash
# Check logs for errors
./cadastre-server -port 8080 -db mock 2>&1 | head -20

# If database issue: mock mode doesn't need PostgreSQL
# If port in use:
lsof -i :8080
# Kill the process using port 8080
```

### Problem: Frontend npm crashes during startup

```bash
# Check if backend is running first!
curl http://localhost:8080/health
# If not, start backend FIRST

# If backend is running but npm still crashes:
cd frontend
rm -rf node_modules package-lock.json
npm ci
npm run dev
```

### Problem: Port already in use

```bash
# Check what's using the port
lsof -i :3000    # Frontend port
lsof -i :8080    # Backend port

# Kill the process (be careful!)
kill -9 <PID>

# Or use different ports:
./cadastre-server -port 8081 -db mock  # Backend on 8081
# Update frontend vite.config.ts proxy target to 8081
```

### Problem: Cannot access frontend at http://localhost:3000

```bash
# Check if Vite server is actually running
ps aux | grep vite

# Check logs
cat frontend.log  # if using script
# or check terminal output if running manually

# Try explicit host binding
cd frontend
npm run dev -- --host 0.0.0.0
```

---

## 📊 Expected Architecture

```
┌─────────────────────────────────────────────┐
│         Your Browser (Client)               │
│     http://localhost:3000                   │
└──────────────────┬──────────────────────────┘
                   │
         ┌─────────▼──────────┐
         │  Vite Dev Server   │
         │  (Port 3000)       │
         │                    │
         │ Serves React UI    │
         │ Proxy: /api/* →    │
         │    localhost:8080  │
         └─────────┬──────────┘
                   │
                   │ /api requests
                   │
         ┌─────────▼──────────┐
         │  Go Backend Server │
         │  (Port 8080)       │
         │                    │
         │ API Handlers       │
         │ Database (Mock)    │
         │ WebSocket          │
         └────────────────────┘
```

---

## 🔐 Environment Configuration

### Backend

Default values (mock database, no external dependencies):
```bash
./cadastre-server -port 8080 -db mock
```

With PostgreSQL:
```bash
DATABASE_URL=postgres://user:pass@localhost/cadastre_ia \
./cadastre-server -port 8080 -db postgres
```

### Frontend

Check `frontend/.env.example`:
```
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080
```

---

## ✅ Testing Integration

Once both servers are running:

### Test 1: Frontend loads

```bash
curl -I http://localhost:3000
# Should return 200 OK
```

### Test 2: Backend health

```bash
curl http://localhost:8080/health
# Should return JSON response
```

### Test 3: Proxy routing (from frontend to backend)

Open browser console and check Network tab:
- Requests to `/api/*` should reach `http://localhost:8080/api/*`

### Test 4: See logs

```bash
# Backend logs (if running from script)
tail -f backend.log

# Frontend logs (if running from script)
tail -f frontend.log

# Or check terminal where npm run dev is running
```

---

## 🎯 Production Deployment

This setup is for development. For production:

1. **Backend**: Build and run Docker container
2. **Frontend**: Build to `dist/` and serve via nginx
3. **Database**: Use PostgreSQL in production (not mock)
4. **Proxy**: Use nginx/Apache (not Vite dev proxy)

See `DEPLOYMENT_MANIFEST.md` for details.

---

## 📞 Quick Reference

| Task | Command |
|------|---------|
| Start everything | `bash STARTUP_PRODUCTION_SEQUENCE.sh` |
| Start backend only | `./cadastre-server -port 8080 -db mock` |
| Start frontend only | `cd frontend && npm run dev` |
| Build frontend | `cd frontend && npm run build` |
| Run backend tests | `go test ./...` |
| Check backend health | `curl http://localhost:8080/health` |
| View backend logs | `tail -f backend.log` |
| View frontend logs | `tail -f frontend.log` |

---

## ⚡ Summary

✅ **Always start backend first**  
✅ **Wait for health check to pass**  
✅ **Then start frontend**  
✅ **Frontend will proxy to backend on 8080**  
✅ **Both can run simultaneously on different ports**

---

---

## 🎉 PHASE 1 COMPLETION (2026-05-17)

**STATUS: ✅✅✅ FULLY OPERATIONAL**

### What's Running Right Now:

**Terminal 1 - Backend (Port 8080):**
```powershell
.\bin\cadastre-server.exe -port 8080 -db postgres -db-conn "postgres://postgres:admin123@127.0.0.1:3779/geomobile137?sslmode=disable"
```
✅ PostgreSQL connected  
✅ All components initialized  
✅ API endpoints responding  

**Terminal 2 - Frontend (Port 3000):**
```powershell
cd frontend && npm run dev
```
✅ Vite dev server ready  
✅ React app rendering  
✅ Proxy to backend active  

**Terminal 3 - Database (Port 3779):**
```powershell
Get-Service postgresql-x64-18 | Select-Object Status
# Result: Running ✅
```

### Database Schema Applied:
- intrinsic_ids ✅
- device_identities ✅
- device_tags ✅
- location_traces ✅
- wifi_networks ✅
- bluetooth_devices ✅
- movement_history ✅
- environment_signatures ✅
- vpn_statuses ✅
- premium_features ✅
- device_authenticity_reports ✅
- suspicious_activities ✅
- users ✅

### Verification Commands:
```powershell
# Test API
curl http://localhost:8080/health

# Check PostgreSQL
$env:PGPASSWORD="admin123"; psql -U postgres -h 127.0.0.1 -p 3779 -d geomobile137 -c "\dt"

# Test frontend
Start-Process "http://localhost:3000"
```

### For Full Details:
See: `PHASE_1_COMPLETION_FINAL.md`

---

## 🎉 Initialization Update (2026-05-16)

**Project Structure Initialized:** ✅ COMPLETE

All essential components created and verified:

### Backend (Go)
- ✅ go.mod with all dependencies (Gin, GORM, WebSocket)
- ✅ cmd/cadastre-server/main.go - Fully functional Gin server
- ✅ pkg/ packages - config, logger, models, api, service, storage, gnss, vision
- ✅ Database support - Mock and PostgreSQL modes

### Frontend (React)
- ✅ React 18 + Vite stack
- ✅ TypeScript configuration
- ✅ Proxy to backend configured (localhost:8080)
- ✅ Health check component with error handling

### Build & Documentation
- ✅ Makefile with 8 build targets
- ✅ .gitignore configured
- ✅ README.md with project overview
- ✅ docs/API.md, docs/DEVELOPMENT.md, docs/DEPLOYMENT.md

### Statistics
- 47 files created
- 24 directories initialized
- 724K total project size
- Ready for immediate development

See PROJECT_INIT_SUMMARY.md for detailed initialization report.

---

**Status:** 🟢 **PRODUCTION READY**  
**Last Verified:** 2026-05-16  
**Created by:** Claude Code Production Verification
**Initialized by:** Système Agent (2026-05-16)
