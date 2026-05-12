# 🧪 PHASE 4 COMPLETE TEST EXECUTION GUIDE

**Status:** 🚀 **READY TO RUN**  
**Expected Duration:** 40-50 minutes  
**Required:** Running backend + frontend servers

---

## ✅ PRE-TEST CHECKLIST (5 minutes)

```bash
# 1. Start Backend Server
cd F:\geomobile137
go build -o cadastre-server ./cmd/cadastre-server
./cadastre-server
# Expected output: "Server listening on :8080"

# 2. Start Frontend Dev Server (new terminal)
cd F:\geomobile137\frontend
npm install  # if needed
npm run dev
# Expected output: "Local: http://localhost:3000"

# 3. Verify Database
psql -h localhost -U postgres -c "SELECT version();"
# Should show PostgreSQL version

# 4. Verify WebSocket
# Open browser console at http://localhost:3000
# Should connect without errors

# 5. Test API manually
curl -H "X-User-ID: test-user" http://localhost:8080/api/v1/quest/available
# Should return 200 with quests array
```

---

## 🚀 TEST EXECUTION SEQUENCE

### TEST 1: API LOAD TEST (12 minutes)

**Install k6 first:**
```bash
# macOS
brew install k6

# Windows (via chocolatey or direct download)
# From: https://github.com/grafana/k6/releases

# Verify
k6 version
```

**Run test:**
```bash
cd F:\geomobile137
k6 run load-tests/api-load-test.js

# Output will show real-time progress:
#   0%   ▓░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  (100/1000 VUs, 0s elapsed / 15s)
#   ...
#   100%  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  (0/1000 VUs, 15s elapsed / 15s)

# Summary at end:
#   http_reqs................................: 45234 12.7/s
#   http_req_duration........................: avg=213ms p(95)=487ms p(99)=892ms
#   http_req_failed...........................: 12 0.03%
#   iteration_duration........................: avg=4.3s min=1.1s med=3.9s max=12.4s
#   iterations................................: 45234 12.7/s
#   vus....................................... : 0 min=0 max=1000
#   vus_max...................................: 1000
```

**Expected Results:**
```
✅ P95 latency: 487ms (target: < 500ms)
✅ P99 latency: 892ms (target: < 1000ms)
✅ Error rate: 0.03% (target: < 0.1%)
✅ Throughput: 12.7 req/sec * 1000 users = 12,700 total
```

---

### TEST 2: WEBSOCKET STRESS TEST (10 minutes)

**Install dependencies:**
```bash
cd F:\geomobile137\load-tests
npm init -y
npm install socket.io-client
```

**Run test:**
```bash
node websocket-stress-test.js

# Output shows real-time status every 10 seconds:
# ╔════════════════════════════════════════╗
# ║   WebSocket Stress Test Status         ║
# ╠════════════════════════════════════════╣
# ║ Connected:         850                 ║
# ║ Errors:            5                   ║
# ║ Events Sent:       42500               ║
# ║ Messages Received: 42350               ║
# ║ Rate (msg/sec):    423.50              ║
# ║ Uptime:            100.2s              ║
# ╚════════════════════════════════════════╝
```

**Expected Results:**
```
✅ Connections: 980+/1000
✅ Message delivery: 99%+
✅ Latency: < 100ms average
✅ No resource leaks
```

---

### TEST 3: SPIKE TEST (8 minutes)

**Run test:**
```bash
k6 run load-tests/spike-test.js

# Stages:
#   1 min:   0 → 500 users (normal load)
#   30 sec:  500 → 2000 users (SPIKE!)
#   1 min:   2000 users (hold)
#   30 sec:  2000 → 500 users (recovery)
#   1 min:   500 → 0 users (wind down)
```

**Expected Results:**
```
✅ P95 during spike: < 1000ms
✅ Recovery time: < 30 seconds
✅ Error rate during spike: < 1%
✅ No hanging connections
```

---

### TEST 4: DATABASE STRESS (5 minutes)

**Option A: Using pgbench (PostgreSQL included)**
```bash
# Test simple query performance
pgbench -h localhost -U postgres -d geo_mobile_db \
  -c 50 -j 10 -t 1000 \
  -f <(echo "SELECT * FROM quests LIMIT 20;")

# Results show:
#   tps = 230.45 (transactions per second)
#   latency = 216.78 ms
```

**Option B: Using psql directly**
```bash
psql -h localhost -U postgres -d geo_mobile_db <<EOF
-- Test 1: Quest query
EXPLAIN ANALYZE
SELECT * FROM quests LIMIT 20;

-- Test 2: Leaderboard query (check if index is used)
EXPLAIN ANALYZE
SELECT * FROM user_progress ORDER BY total_xp DESC LIMIT 50;

-- Test 3: Check slow queries
SELECT query, calls, mean_time
FROM pg_stat_statements
WHERE mean_time > 100
ORDER BY mean_time DESC
LIMIT 10;
EOF
```

**Expected Results:**
```
✅ Quest select: 15-30ms
✅ Leaderboard: 50-150ms
✅ User updates: 20-50ms
✅ All queries use indexes (Seq Scan = BAD)
```

---

## 📊 RESULT ANALYSIS

### After all tests complete:

**1. Compare Results to Baselines**
```
Baseline (from PHASE_4_LOAD_TESTING_PLAN.md):
  P95: ~487ms ← API Load Test result should match
  P99: ~892ms ← API Load Test result
  Error Rate: 0.3% ← Should be similar

Your Results:
  [Fill in from actual test runs above]
```

**2. Check Performance Issues**
- If P95 > 500ms: Database needs indexing or caching
- If error rate > 0.5%: Check backend logs, resource limits
- If WebSocket delivery < 99%: Connection pool exhaustion

**3. Memory & CPU Usage**
```bash
# While tests run, monitor in separate terminal:
# macOS/Linux
top -p $(pgrep cadastre-server)

# Windows
tasklist /FI "IMAGENAME eq cadastre-server*" /V
```

---

## 🔧 OPTIMIZATION ACTIONS

**If Tests Show Issues:**

### Slow Database Queries (> 100ms)
```sql
-- Add missing indexes
CREATE INDEX idx_quests_user_id ON quests(user_id);
CREATE INDEX idx_user_progress_xp_desc ON user_progress(total_xp DESC);
CREATE INDEX idx_leaderboard_tier ON user_progress(tier_level);

-- Then rerun tests
```

### High Error Rate (> 0.1%)
```bash
# Check backend logs
tail -f logs/cadastre-server.log

# Common issues:
# - Connection pool exhausted
# - Memory limit hit
# - Database connections maxed out
```

### Low WebSocket Delivery
```bash
# Check system limits
ulimit -n  # File descriptors
# Should be > 2000 for 1000 WebSocket connections

# Increase if needed
ulimit -n 10000
```

---

## 📈 FINAL REPORT TEMPLATE

Create `load-test-results-2026-05-12.txt`:

```
╔════════════════════════════════════════════════════════╗
║       PHASE 4 LOAD TESTING RESULTS — 2026-05-12       ║
╠════════════════════════════════════════════════════════╣

API LOAD TEST:
  Duration:        15 minutes (0→1000→0 users)
  Max Concurrency: 1000 simultaneous users
  Total Requests:  [YOUR_NUMBER]
  Success Rate:    [YOUR_NUMBER]%
  
  Latency (HTTP):
    Average:       [YOUR_MS]ms
    P95:           [YOUR_MS]ms  (target: < 500ms)
    P99:           [YOUR_MS]ms  (target: < 1000ms)
    Max:           [YOUR_MS]ms
  
  Error Rate:      [YOUR_NUMBER]%  (target: < 0.1%)
  Throughput:      [YOUR_NUMBER] req/sec

WEBSOCKET STRESS TEST:
  Connections:     [YOUR_NUMBER]/1000
  Success Rate:    [YOUR_NUMBER]%
  Messages Sent:   [YOUR_NUMBER]
  Messages Rcvd:   [YOUR_NUMBER]
  Delivery Rate:   [YOUR_NUMBER]%  (target: > 99%)
  Latency:         [YOUR_MS]ms  (target: < 100ms)

SPIKE TEST:
  Normal Load (500 users):
    P95 Latency:   [YOUR_MS]ms
    Error Rate:    [YOUR_NUMBER]%
  
  Spike (2000 users):
    P95 Latency:   [YOUR_MS]ms
    Error Rate:    [YOUR_NUMBER]%
  
  Recovery Time:   [YOUR_SEC]s  (target: < 30s)

DATABASE BENCHMARK:
  Quest Query:     [YOUR_MS]ms  (target: < 30ms)
  Leaderboard:     [YOUR_MS]ms  (target: < 100ms)
  User Updates:    [YOUR_MS]ms  (target: < 50ms)

SYSTEM RESOURCES:
  Peak CPU:        [YOUR_NUMBER]%  (target: < 70%)
  Peak Memory:     [YOUR_GB]GB  (target: stable)
  Database CPU:    [YOUR_NUMBER]%  (target: < 70%)
  Connections:     [YOUR_NUMBER] (target: < max_connections)

ISSUES FOUND:
  ☐ None - All tests passed!
  ☐ Performance: [DESCRIBE]
  ☐ Stability: [DESCRIBE]
  ☐ Database: [DESCRIBE]
  ☐ WebSocket: [DESCRIBE]

RECOMMENDATIONS:
  1. [RECOMMENDATION]
  2. [RECOMMENDATION]
  3. [RECOMMENDATION]

NEXT STEPS:
  ☐ Apply optimization fixes
  ☐ Retest critical scenarios
  ☐ Document results
  ☐ Proceed to Phase 5 (Production Ready)

╚════════════════════════════════════════════════════════╝

Test Executed By: [NAME]
Date: 2026-05-12
Environment: [LOCALHOST/AWS/OTHER]
```

---

## ✅ COMPLETION CHECKLIST

- [ ] API Load Test completed (12 min)
- [ ] WebSocket Stress Test completed (10 min)
- [ ] Spike Test completed (8 min)
- [ ] Database Benchmarking completed (5 min)
- [ ] Results documented in report
- [ ] Issues identified and categorized
- [ ] Optimization recommendations created
- [ ] Performance meets success criteria (or gaps identified)

---

**Status:** 🚀 **READY FOR EXECUTION**

Start with TEST 1 and progress through sequentially. Each test builds on previous insights.

Expected completion time: **50 minutes total**

Let the user know when tests are complete!

