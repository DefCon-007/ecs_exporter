import { check } from 'k6';
import http from 'k6/http';

export const options = {
  thresholds: {
    checks: ['rate>0.9'],
  },
};

export default function () {
  const res = http.get('http://localhost:9779/metrics');

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response body is not empty': (r) => r.body.length > 0,
    'response body contains metrics': (r) =>
      r.body.includes('ecs_memory_bytes'),
  });
}
