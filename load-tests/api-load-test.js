import http from 'k6/http';
import { check, sleep, group } from 'k6';

const BASE_URL = 'http://localhost:8080/api/v1';
const USER_ID = 'test-user-' + Math.floor(Math.random() * 10000);

export const options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '3m', target: 500 },
    { duration: '2m', target: 1000 },
    { duration: '5m', target: 1000 },
    { duration: '3m', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'http_req_failed': ['rate<0.001'],
  },
};

export default function() {
  const headers = {
    'X-User-ID': USER_ID,
    'Content-Type': 'application/json',
  };

  group('Quest Operations', () => {
    let res = http.get(`${BASE_URL}/quest/available?limit=20`, { headers });
    check(res, {
      'GET /quest/available status is 200': (r) => r.status === 200,
      'response time < 100ms': (r) => r.timings.duration < 100,
    });

    const questPayload = JSON.stringify({
      quest_id: 'quest-' + Math.floor(Math.random() * 100),
    });
    res = http.post(`${BASE_URL}/quest/start`, questPayload, { headers });
    check(res, {
      'POST /quest/start status is 201': (r) => r.status === 201,
    });

    sleep(1);
  });

  group('Leaderboard Operations', () => {
    let res = http.get(
      `${BASE_URL}/leaderboard?scope=global&limit=50`,
      { headers }
    );
    check(res, {
      'GET /leaderboard status is 200': (r) => r.status === 200,
      'response time < 150ms': (r) => r.timings.duration < 150,
    });

    sleep(1);
  });

  group('User Operations', () => {
    let res = http.get(`${BASE_URL}/user/progress`, { headers });
    check(res, {
      'GET /user/progress status is 200': (r) => r.status === 200,
    });

    sleep(1);
  });

  sleep(Math.random() * 3);
}
