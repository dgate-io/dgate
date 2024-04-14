#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

# SETUP BASE

CALL='http --ignore-stdin --check-status -p=mb -F'

# domain setup

$CALL PUT ${ADMIN_URL}/namespace \
    name=test-ns1

$CALL PUT ${ADMIN_URL}/domain \
    name=test-dm patterns:='["dgate.dev"]' \
    namespace=test-ns1

$CALL PUT ${ADMIN_URL}/service \
    name=test-svc urls:='["http://localhost:8080"]' \
    namespace=test-ns1
    
MOD_B64="$(base64 < $DIR/performance_test_prep.ts)"
$CALL PUT ${ADMIN_URL}/module \
    name=test-mod payload="$MOD_B64" \
    namespace=test-ns1

$CALL PUT ${ADMIN_URL}/route \
    name=base-rt1 \
    service=test-svc \
    methods:='["GET"]' \
    paths:='["/svctest"]' \
    preserveHost:=false \
    stripPath:=true \
    namespace=test-ns1

$CALL PUT ${ADMIN_URL}/route \
    name=test-rt2 \
    paths:='["/modtest","/modview"]' \
    methods:='["GET"]' \
    modules:='["test-mod"]' \
    stripPath:=false \
    preserveHost:=false \
    namespace=test-ns1

$CALL PUT ${ADMIN_URL}/route \
    name=test-rt3 \
    paths:='["/blank"]' \
    methods:='["GET"]' \
    stripPath:=false \
    preserveHost:=false \
    namespace=test-ns1


http ${PROXY_URL}/svctest Host:dgate.dev

http ${PROXY_URL}/modtest Host:dgate.dev

http ${PROXY_URL}/blank Host:dgate.dev