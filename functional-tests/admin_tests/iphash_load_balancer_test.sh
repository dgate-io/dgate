#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}
TEST_URL=${TEST_URL:-"http://localhost:8888"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli -Vf namespace create \
    name=test-lb-ns

dgate-cli -Vf domain create \
    name=test-lb-dm \
    patterns:='["test-lb.example.com"]' \
    namespace=test-lb-ns

MOD_B64="$(base64 < $DIR/iphash_load_balancer.ts)"
dgate-cli -Vf module create \
    name=printer \
    payload="$MOD_B64" \
    namespace=test-lb-ns


dgate-cli -Vf service create \
    name=base_svc \
    urls:="$TEST_URL/a","$TEST_URL/b","$TEST_URL/c" \
    namespace=test-lb-ns

dgate-cli -Vf route create \
    name=base_rt \
    paths:='["/test-lb","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    service=base_svc \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-lb-ns

path1="$(curl -sf ${PROXY_URL}/test-lb -H Host:test-lb.example.com | jq -r '.data.path')"

path2="$(curl -sf ${PROXY_URL}/test-lb -H Host:test-lb.example.com -H X-Forwarded-For:192.168.0.1 | jq -r '.data.path')"

if [ "$path1" != "$path2" ]; then
    echo "IP Hash Load Balancer Test Passed"
else
    echo "IP Hash Load Balancer Test Failed"
    exit 1
fi