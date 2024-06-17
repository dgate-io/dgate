#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"


export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli -V -f namespace create \
    name=test-ns1

dgate-cli domain create \
    name=test-dm patterns:='["performance.example.com"]' \
    namespace=test-ns1 priority:=100

dgate-cli service create \
    name=test-svc urls:='["http://localhost:8888"]' \
    namespace=test-ns1 retries:=3 retryTimeout=50ms
    
MOD_B64="$(base64 < $DIR/performance_test_prep.ts)"
dgate-cli module create \
    name=test-mod payload="$MOD_B64" \
    namespace=test-ns1

dgate-cli route create \
    name=base-rt1 \
    service=test-svc \
    methods:='["GET"]' \
    paths:='["/svctest"]' \
    preserveHost:=false \
    stripPath:=true \
    namespace=test-ns1

dgate-cli route create \
    name=test-rt2 \
    paths:='["/modtest","/modview"]' \
    methods:='["GET"]' \
    modules:='["test-mod"]' \
    stripPath:=false \
    preserveHost:=false \
    namespace=test-ns1

dgate-cli route create \
    name=test-rt3 \
    paths:='["/blank"]' \
    methods:='["GET"]' \
    stripPath:=false \
    preserveHost:=false \
    namespace=test-ns1


curl -s --fail-with-body ${PROXY_URL}/svctest -H Host:performance.example.com

curl -s --fail-with-body ${PROXY_URL}/modtest -H Host:performance.example.com

curl -s ${PROXY_URL}/blank -H Host:performance.example.com

echo "Performance Test Prep Done"