import http from "k6/http";
import { check, sleep } from 'k6';

const n = 20;
const inc = 5;
let curWait = -inc;
export let options = {
  scenarios: {
    modtest: {
      executor: 'constant-vus',
      vus: n,
      duration: inc + 'm',
      startTime: (curWait += inc) + 'm',
      exec: 'dgatePath',
      env: { DGATE_PATH: '/modtest' },
      gracefulStop: '5s',
    },
    modtest_wait: {
      executor: 'constant-vus',
      vus: n*3,
      duration: inc + 'm',
      startTime: (curWait += inc) + 'm',
      exec: 'dgatePath',
      env: { DGATE_PATH: '/modtest?wait=30ms' },
      gracefulStop: '5s',
    },
    svctest: {
      executor: 'constant-vus',
      vus: n,
      duration: inc + 'm',
      startTime: (curWait += inc) + 'm',
      exec: 'dgatePath',
      env: { DGATE_PATH: "/svctest" },
      gracefulStop: '5s',
    },
    svctest_wait: {
      executor: 'constant-vus',
      vus: n*3,
      duration: inc + 'm',
      startTime: (curWait += inc) + 'm',
      exec: 'dgatePath',
      env: { DGATE_PATH: "/svctest?wait=30ms" },
      gracefulStop: '5s',
    },
    // test_server_direct: {
    //   executor: 'constant-vus',
    //   vus: n,
    //   duration: inc + 'm',
    //   startTime: (curWait += inc) + 'm',
    //   exec: 'dgatePath',
    //   env: { DGATE_PATH: ":8888/direct" },
    //   gracefulStop: '5s',
    // },
    // test_server_direct_wait: {
    //   executor: 'constant-vus',
    //   vus: n*5,
    //   duration: inc + 'm',
    //   startTime: (curWait += inc) + 'm',
    //   exec: 'dgatePath',
    //   env: { DGATE_PATH: ":8888/svctest?wait=30ms" },
    //   gracefulStop: '5s',
    // },
  },
  discardResponseBodies: true,
};

let i = 0;
let ports = [8888];
// const raftMode = !!__ENV.RAFT_MODE || false;
// 'http://localhost:' + ports[i++ % ports.length];

export function dgatePath() {
  const dgatePath = __ENV._PROXY_URL || 'http://localhost'
  const path = __ENV.DGATE_PATH;
  let res = http.get(dgatePath + path, {
    headers: { Host: 'performance.example.com' },
  });
  let results = {};
  results[path + ': status is ' + res.status] =
    (r) => (r.status >= 200 && r.status < 400);
  check(res, results);
};
