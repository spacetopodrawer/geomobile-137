import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = 'http://localhost:8080/api/v1';

export const options = {
  stages: [
    { duration: '1m', target: 500 },
    { duration: '30s', target: 2000 },
    { duration: '1m', target: 2000 },
    { duration: '30s', target: 500 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'],
    'http_req_failed': ['rate<0.01'],
  },
};

export default function() {
  const headers = {
    'X-User-ID': 'spike-test-' + Math.random(),
    'Content-Type': 'application/json',
  };

  const requests = [
    () => http.get(`${BASE_URL}/quest/available?limit=20`, { headers }),
    () => http.get(`${BASE_URL}/user/progress`, { headers }),
    () => http.get(`${BASE_URL}/leaderboard?scope=global&limit=50`, { headers }),
  ];

  const randomRequest = requests[Math.floor(Math.random() * requests.length)];
  const res = randomRequest();

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time acceptable': (r) => r.timings.duration < 2000,
  });

  sleep(Math.random() * 2);
}
