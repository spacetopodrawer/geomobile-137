const io = require('socket.io-client');

const WS_URL = 'http://localhost:8080';
const NUM_CONNECTIONS = 1000;
const TEST_DURATION = 10 * 60 * 1000; // 10 minutes

let successCount = 0;
let errorCount = 0;
let messageCount = 0;
let eventsSent = 0;

const connections = [];

console.log(`🚀 Starting WebSocket stress test with ${NUM_CONNECTIONS} connections...`);
console.log(`Duration: ${TEST_DURATION / 1000 / 60} minutes`);

// Create connections
for (let i = 0; i < NUM_CONNECTIONS; i++) {
  const socket = io(WS_URL, {
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: 5,
    transports: ['websocket'],
  });

  socket.on('connect', () => {
    successCount++;
    if (successCount % 100 === 0) {
      console.log(`✓ Connected: ${successCount}/${NUM_CONNECTIONS}`);
    }
  });

  socket.on('disconnect', () => {
    console.log(`✗ Disconnected: ${errorCount + 1}/${NUM_CONNECTIONS}`);
    errorCount++;
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
    console.error(`Error on socket ${i}: ${error}`);
  });

  connections.push(socket);
}

// Emit events from all connections
setInterval(() => {
  const eventsThisTick = 5; // 5 events per tick

  for (let i = 0; i < eventsThisTick; i++) {
    const socketIndex = Math.floor(Math.random() * connections.length);
    const socket = connections[socketIndex];

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
      eventsSent++;
    }
  }
}, 100);

// Status reporting
const statusInterval = setInterval(() => {
  const uptime = (Date.now() - startTime) / 1000;
  const msgPerSec = messageCount > 0 ? (messageCount / uptime).toFixed(2) : '0.00';

  console.log(`
╔════════════════════════════════════════╗
║   WebSocket Stress Test Status         ║
╠════════════════════════════════════════╣
║ Connected:         ${successCount.toString().padEnd(24)} ║
║ Errors:            ${errorCount.toString().padEnd(24)} ║
║ Events Sent:       ${eventsSent.toString().padEnd(24)} ║
║ Messages Received: ${messageCount.toString().padEnd(24)} ║
║ Rate (msg/sec):    ${msgPerSec.padEnd(24)} ║
║ Uptime:            ${uptime.toFixed(1)}s${' '.repeat(28 - uptime.toFixed(1).length)} ║
╚════════════════════════════════════════╝
  `);
}, 10000);

const startTime = Date.now();

// Run for specified duration
setTimeout(() => {
  console.log('\n⏸️  Closing all connections...');
  clearInterval(statusInterval);

  connections.forEach((socket) => socket.disconnect());

  console.log(`
╔════════════════════════════════════════╗
║   Final Test Results                   ║
╠════════════════════════════════════════╣
║ Total Connected:   ${successCount.toString().padEnd(24)} ║
║ Total Errors:      ${errorCount.toString().padEnd(24)} ║
║ Total Events Sent: ${eventsSent.toString().padEnd(24)} ║
║ Total Msgs Rcvd:   ${messageCount.toString().padEnd(24)} ║
║ Success Rate:      ${((successCount / NUM_CONNECTIONS) * 100).toFixed(2)}%${' '.repeat(18)} ║
║ Delivery Rate:     ${((messageCount / eventsSent) * 100).toFixed(2)}%${' '.repeat(18)} ║
╚════════════════════════════════════════╝
  `);

  process.exit(0);
}, TEST_DURATION);
