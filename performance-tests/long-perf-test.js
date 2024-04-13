import http from "k6/http";
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '5s', target: 200},
    { duration: '1h', target: 300},
    { duration: '1h', target: 100},
  ],
};

let url = "http://localhost:80";
let i = 0;

export default async function() {
  let res = http.get(url + "/modtest", {
    headers: { Host: 'dgate.dev' },
  });
    check(res, { 'status: 204': (r) => r.status == 204 });
};
