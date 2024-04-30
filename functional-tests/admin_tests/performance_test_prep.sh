#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

# domain setup

dgate-cli namespace create \
    name=test-ns1

dgate-cli domain create \
    name=test-dm patterns:='["dgate.dev"]' \
    namespace=test-ns1

dgate-cli service create \
    name=test-svc urls:='["http://localhost:8888"]' \
    namespace=test-ns1
    
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


curl -s --fail-with-body ${PROXY_URL}/svctest -H Host:dgate.dev

curl -s --fail-with-body ${PROXY_URL}/modtest -H Host:dgate.dev

curl -s ${PROXY_URL}/blank -H Host:dgate.dev

echo "Performance Test Prep Done"