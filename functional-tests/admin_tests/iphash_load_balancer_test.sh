#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli namespace create \
    name=test-lb-ns

dgate-cli domain create \
    name=test-lb-dm \
    patterns:='["test-lb.example.com"]' \
    namespace=test-lb-ns

MOD_B64="$(base64 < $DIR/iphash_load_balancer.ts)"
dgate-cli module create \
    name=printer \
    payload="$MOD_B64" \
    namespace=test-lb-ns


dgate-cli service create \
    name=base_svc \
    urls:='["http://localhost:8888/a","http://localhost:8888/b","http://localhost:8888/c"]' \
    namespace=test-lb-ns

dgate-cli route create \
    name=base_rt \
    paths:='["/test-lb","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    service=base_svc \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-lb-ns

path1="$(curl -s --fail-with-body ${PROXY_URL}/test-lb -H Host:test-lb.example.com | jq -r '.data.path')"

path2="$(curl -s --fail-with-body ${PROXY_URL}/test-lb -H Host:test-lb.example.com -H X-Forwarded-For:192.168.0.1 | jq -r '.data.path')"

if [ "$path1" != "$path2" ]; then
    echo "IP Hash Load Balancer Test Passed"
else
    echo "IP Hash Load Balancer Test Failed"
    exit 1
fi