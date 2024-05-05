import http from "k6/http";
import { check, sleep } from 'k6';

// export let options = {
//   stages: [
//     { duration: '5s', target: 200},
//     { duration: '1h', target: 300},
//     { duration: '1h', target: 100},
//   ],
// };

const n = 15;
export let options = {
  scenarios: {
    modtest: {
      executor: 'constant-vus',
      vus: n,
      duration: '20m',
      exec: 'dgatePath',
      env: { DGATE_PATH: '/modtest' },
      // startTime: '25s',
      gracefulStop: '10s',
    },
    svctest: {
      executor: 'constant-vus',
      vus: n,
      duration: '20m',
      exec: 'dgatePath',
      env: { DGATE_PATH: "/svctest" },
      // startTime: '10m',
      gracefulStop: '5s',
    },
    svctest_500ms: {
      executor: 'constant-vus',
      vus: n,
      duration: '20m',
      exec: 'dgatePath',
      env: { DGATE_PATH: "/svctest?wait=30ms" },
      // startTime: '20m',
      gracefulStop: '5s',
    },
  },
  discardResponseBodies: true,
};

export function dgatePath() {
  const dgatePath = __ENV.PROXY_URL || 'http://localhost';
  const path = __ENV.DGATE_PATH;
  let res = http.get(dgatePath + path, {
    headers: { Host: 'dgate.dev' },
  });
  let results = {};
  results[path + ': status is ' + res.status] =
    (r) => r.status >= 200 && r.status < 400;
  check(res, results);
};
