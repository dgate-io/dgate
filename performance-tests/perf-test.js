import http from "k6/http";
import { check } from 'k6';

const n = 10;
export let options = {
  scenarios: {
    modtest: {
      executor: 'constant-vus',
      vus: n,
      duration: '20s',
      // same function as the scenario above, but with different env vars
      exec: 'dgatePath',
      env: { DGATE_PATH: '/modtest' },
      // startTime: '25s',
      gracefulStop: '5s',
    },
    // svctest: {
    //   executor: 'constant-vus',
    //   vus: n,
    //   duration: '20s',
    //   exec: 'dgatePath', // same function as the scenario above, but with different env vars
    //   env: { DGATE_PATH: "/svctest" },
    //   // startTime: '25s',
    //   gracefulStop: '5s',
    // },
    // blank: {
    //   executor: 'constant-vus',
    //   vus: n,
    //   duration: '20s',
    //   exec: 'dgatePath', // same function as the scenario above, but with different env vars
    //   env: { DGATE_PATH: "/blank" },
    //   // startTime: '50s',
    //   gracefulStop: '5s',
    // },
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
  results[path + ': status is ' + res.status] = (r) => r.status < 400;
  check(res, results);
};