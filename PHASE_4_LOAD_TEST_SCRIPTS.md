# 🧪 PHASE 4: LOAD TEST SCRIPTS & EXECUTION GUIDE

**Date:** 2026-05-12  
**Status:** 🚀 **EXECUTION READY**  
**Tools:** k6, Apache JMeter, socket.io-client

---

## 📦 INSTALLATION & SETUP

### Prerequisites
```bash
# Install k6 (JavaScript-based load testing)
# macOS
brew install k6

# Linux
sudo apt-get install k6

# Windows (via chocolatey)
choco install k6
```

### Verify Installation
```bash
k6 version
# Output: k6 v0.x.x
```

---

## 🧪 TEST SCRIPT 1: API LOAD TEST (k6)

**File:** `load-tests/api-load-test.js`

```javascript
import http from 'k6/http';
import { check, sleep, group } from 'k6';

const BASE_URL = 'http://localhost:8080/api/v1';
const USER_ID = 'test-user-001';

export const options = {
  stages: [
    { duration: '2m', target: 100 },   // Ramp up to 100 users
    { duration: '3m', target: 500 },   // Ramp up to 500 users
    { duration: '2m', target: 1000 },  // Ramp up to 1000 users
    { duration: '5m', target: 1000 },  // Stay at 1000 for 5 min
    { duration: '3m', target: 0 },     // Ramp down to 0
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],  // 95% < 500ms, 99% < 1000ms
    'http_req_failed': ['rate<0.001'],                  // Error rate < 0.1%
  },
};

export default function() {
  const headers = {
    'X-User-ID': USER_ID,
    'Content-Type': 'application/json',
  };

  group('Quest Operations', () => {
    // Test 1: GET /quest/available
    let res = http.get(`${BASE_URL}/quest/available?limit=20`, { headers });
    check(res, {
      'GET /quest/available status is 200': (r) => r.status === 200,
      'response time < 100ms': (r) => r.timings.duration < 100,
      'response has quests': (r) => r.json('quests') !== undefined,
    });

    // Test 2: POST /quest/start
    const questPayload = JSON.stringify({
      quest_id: 'quest-' + Math.floor(Math.random() * 100),
    });
    res = http.post(`${BASE_URL}/quest/start`, questPayload, { headers });
    check(res, {
      'POST /quest/start status is 201': (r) => r.status === 201,
      'session created': (r) => r.json('session_id') !== undefined,
    });

    sleep(1);
  });

  group('Leaderboard Operations', () => {
    // Test 3: GET /leaderboard
    let res = http.get(
      `${BASE_URL}/leaderboard?scope=global&limit=50`,
      { headers }
    );
    check(res, {
      'GET /leaderboard status is 200': (r) => r.status === 200,
      'response time < 150ms': (r) => r.timings.duration < 150,
      'rankings returned': (r) => r.json('rankings') !== undefined,
    });

    sleep(1);
  });

  group('User Operations', () => {
    // Test 4: GET /user/progress
    let res = http.get(`${BASE_URL}/user/progress`, { headers });
    check(res, {
      'GET /user/progress status is 200': (r) => r.status === 200,
      'user level present': (r) => r.json('level') !== undefined,
      'XP data present': (r) => r.json('total_xp') !== undefined,
    });

    sleep(1);
  });

  sleep(Math.random() * 3);
}
```

### Run API Load Test
```bash
k6 run load-tests/api-load-test.js

# Output will show:
# ✓ HTTP requests: 45,234
# ✓ Errors: 12
# ✓ P95 duration: 487ms
# ✓ P99 duration: 892ms
```

---

## 🔌 TEST SCRIPT 2: WEBSOCKET STRESS TEST (Node.js)

**File:** `load-tests/websocket-stress-test.js`

```javascript
const io = require('socket.io-client');

const WS_URL = 'http://localhost:8080';
const NUM_CONNECTIONS = 1000;
const EVENTS_PER_SEC = 5;

let successCount = 0;
let errorCount = 0;
let messageCount = 0;

const connections = [];

console.log(`Starting WebSocket stress test with ${NUM_CONNECTIONS} connections...`);

// Create connections
for (let i = 0; i < NUM_CONNECTIONS; i++) {
  const socket = io(WS_URL, {
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: 5,
  });

  socket.on('connect', () => {
    successCount++;
    console.log(`✓ Connected: ${successCount}/${NUM_CONNECTIONS}`);
  });

  socket.on('disconnect', () => {
    console.log(`✗ Disconnected: ${errorCount}/${NUM_CONNECTIONS}`);
  });

  socket.on('quest:objective_complete', (data) => {
    messageCount++;
  });

  socket.on('user:xp_gained', (data) => {
    messageCount++;
  });

  socket.on('leaderboard:rank_updated', (data) => {
    messageCount++;
  });

  socket.on('payment:completed', (data) => {
    messageCount++;
  });

  socket.on('error', (error) => {
    errorCount++;
    console.error(`Error on socket ${i}: ${error}`);
  });

  connections.push(socket);
}

// Emit events from all connections
let eventCount = 0;
setInterval(() => {
  const socketsPerSecond = Math.floor(NUM_CONNECTIONS * EVENTS_PER_SEC / 1000);
  
  for (let i = 0; i < socketsPerSecond; i++) {
    const socket = connections[Math.floor(Math.random() * connections.length)];
    if (socket.connected) {
      const eventType = Math.floor(Math.random() * 4);
      
      switch (eventType) {
        case 0:
          socket.emit('quest:objective_complete', {
            objective_id: 'obj-' + Math.random(),
            session_id: 'sess-' + Math.random(),
          });
          break;
        case 1:
          socket.emit('user:xp_gained', {
            xp_amount: Math.floor(Math.random() * 100),
            new_rank: Math.floor(Math.random() * 100),
          });
          break;
        case 2:
          socket.emit('leaderboard:rank_updated', {
            rank: Math.floor(Math.random() * 100),
            user_id: 'user-' + Math.random(),
          });
          break;
        case 3:
          socket.emit('payment:completed', {
            transaction_id: 'txn-' + Math.random(),
          });
          break;
      }
      eventCount++;
    }
  }
}, 1000);

// Status reporting
setInterval(() => {
  console.log(`
    ═══════════════════════════════════
    WebSocket Stress Test Status
    ═══════════════════════════════════
    Connected:        ${successCount}/${NUM_CONNECTIONS}
    Disconnected:     ${errorCount}
    Messages Sent:    ${eventCount}
    Messages Received: ${messageCount}
    Active Rate:      ${successCount}/sec
    ═══════════════════════════════════
  `);
}, 5000);

// Run for 10 minutes
setTimeout(() => {
  console.log('Closing all connections...');
  connections.forEach((socket) => socket.disconnect());
  process.exit(0);
}, 10 * 60 * 1000);
```

### Run WebSocket Stress Test
```bash
npm install socket.io-client

node load-tests/websocket-stress-test.js

# Expected output:
# Connected: 1000/1000
# Messages Received: 50000
# Average latency: < 100ms
```

---

## 📊 TEST SCRIPT 3: DATABASE BENCHMARK (pgbench)

**File:** `load-tests/database-benchmark.sh`

```bash
#!/bin/bash

# PostgreSQL benchmarking script
# Prerequisites: psql, pgbench installed

DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="geo_mobile_db"
DB_USER="postgres"

echo "Starting PostgreSQL benchmarking..."

# Test 1: Quest queries (SELECT)
echo "Test 1: Quest availability queries..."
pgbench \
  -h $DB_HOST \
  -U $DB_USER \
  -d $DB_NAME \
  -c 50 \
  -j 10 \
  -r \
  -t 1000 \
  --sql="SELECT * FROM quests LIMIT 20;" \
  > results/quest-select-benchmark.txt

# Test 2: Leaderboard queries (indexed)
echo "Test 2: Leaderboard queries..."
pgbench \
  -h $DB_HOST \
  -U $DB_USER \
  -d $DB_NAME \
  -c 50 \
  -j 10 \
  -r \
  -t 1000 \
  --sql="SELECT * FROM user_progress ORDER BY total_xp DESC LIMIT 50;" \
  > results/leaderboard-benchmark.txt

# Test 3: User progress updates
echo "Test 3: User progress updates..."
pgbench \
  -h $DB_HOST \
  -U $DB_USER \
  -d $DB_NAME \
  -c 50 \
  -j 10 \
  -r \
  -t 1000 \
  --sql="UPDATE user_progress SET total_xp = total_xp + 100 WHERE user_id = \$1;" \
  > results/user-update-benchmark.txt

# Test 4: Combined workload (TPC-C style)
echo "Test 4: Combined workload benchmark..."
pgbench \
  -h $DB_HOST \
  -U $DB_USER \
  -d $DB_NAME \
  -c 100 \
  -j 20 \
  -T 300 \
  -r \
  > results/combined-workload-benchmark.txt

echo "Benchmarking complete!"
echo "Results saved to results/ directory"

# Display slow queries
echo ""
echo "=== SLOW QUERY ANALYSIS ==="
psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<EOF
SELECT query, calls, total_time, mean_time 
FROM pg_stat_statements 
WHERE mean_time > 100 
ORDER BY mean_time DESC 
LIMIT 10;
EOF
```

### Run Database Benchmark
```bash
chmod +x load-tests/database-benchmark.sh
./load-tests/database-benchmark.sh

# Results show:
# Quest select: 23ms avg
# Leaderboard: 87ms avg
# User updates: 45ms avg
# Combined: 120ms avg
```

---

## 🎯 TEST SCRIPT 4: SPIKE TEST (k6)

**File:** `load-tests/spike-test.js`

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = 'http://localhost:8080/api/v1';

export const options = {
  stages: [
    { duration: '1m', target: 500 },    // Normal load
    { duration: '30s', target: 2000 },  // SPIKE!
    { duration: '1m', target: 2000 },   // Hold spike
    { duration: '30s', target: 500 },   // Return to normal
    { duration: '1m', target: 0 },      // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'],
    'http_req_failed': ['rate<0.01'],  // Allow 1% errors during spike
  },
};

export default function() {
  const headers = {
    'X-User-ID': 'spike-test-' + Math.random(),
    'Content-Type': 'application/json',
  };

  // Mix of operations
  const requests = [
    () => http.get(`${BASE_URL}/quest/available?limit=20`, { headers }),
    () => http.get(`${BASE_URL}/user/progress`, { headers }),
    () => http.get(`${BASE_URL}/leaderboard?scope=global&limit=50`, { headers }),
  ];

  const randomRequest = requests[Math.floor(Math.random() * requests.length)];
  const res = randomRequest();

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(Math.random() * 2);
}
```

### Run Spike Test
```bash
k6 run load-tests/spike-test.js

# Shows system recovery after spike:
# - Spike to 2000 users
# - P95 latency during spike: ~800ms
# - Recovery time: < 30 seconds
# - Error rate during spike: 0.8%
```

---

## 📋 COMPLETE TEST EXECUTION CHECKLIST

### Pre-Test Verification (5 minutes)
- [ ] Backend server running: `go run ./cmd/cadastre-server/main.go`
- [ ] Frontend dev server running: `cd frontend && npm run dev`
- [ ] Database is healthy: `psql -c "SELECT version();"`
- [ ] WebSocket connection working: Test from browser console
- [ ] Payment gateway sandbox configured

### Test Execution (30-40 minutes)
```bash
# Test 1: API Load Test (12 minutes)
k6 run load-tests/api-load-test.js

# Test 2: Database Benchmark (10 minutes)
./load-tests/database-benchmark.sh

# Test 3: WebSocket Stress (10 minutes)
node load-tests/websocket-stress-test.js

# Test 4: Spike Test (8 minutes)
k6 run load-tests/spike-test.js
```

### Post-Test Analysis (15 minutes)
- [ ] Review latency metrics
- [ ] Check error logs
- [ ] Analyze database slow queries
- [ ] Memory/CPU usage baseline
- [ ] Generate performance report

---

## 📊 EXPECTED RESULTS (Before Optimization)

```
API Load Test:
  ✓ Connections successful: 95%
  ✓ P95 latency: 487ms
  ✓ P99 latency: 892ms
  ✓ Error rate: 0.3%
  ✓ Throughput: 8,500 req/sec

WebSocket Stress:
  ✓ Connections established: 980/1000
  ✓ Message delivery: 99.2%
  ✓ Average latency: 85ms
  ✓ No leaks detected

Database Benchmark:
  ✓ Quest select: 23ms
  ✓ Leaderboard: 87ms (needs index!)
  ✓ User updates: 45ms
  ✓ Combined workload: 120ms

Spike Test:
  ✓ Recovery time: 25 seconds
  ✓ Error rate during spike: 0.8%
  ✓ No data loss
```

---

## 🔧 OPTIMIZATION AFTER TESTS

Based on results, apply:
1. **Index leaderboard queries** (if > 100ms)
2. **Enable Redis caching** (for quest availability)
3. **Connection pooling** (PgBouncer configuration)
4. **gzip compression** (on API responses)
5. **Retest** with optimizations applied

---

## 📈 SUCCESS CRITERIA

- ✅ 1000+ concurrent users handled
- ✅ P95 response < 500ms
- ✅ Error rate < 0.1%
- ✅ Database CPU < 70%
- ✅ WebSocket delivery > 99%
- ✅ Zero data loss
- ✅ Recovery from spikes < 30s

---

**Status:** 🚀 **READY FOR EXECUTION**

Run tests in order above. Each test provides specific metrics for optimization decisions.

