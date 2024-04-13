#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

CALL='http --check-status -p=mb -F'

$CALL PUT ${ADMIN_URL}/namespace \
    name=test-lb-ns

$CALL PUT ${ADMIN_URL}/domain \
    name=test-lb-dm \
    patterns:='["test-lb.com"]' \
    namespace=test-lb-ns

$CALL ${ADMIN_URL}/domain

MOD_B64="$(base64 < $DIR/iphash_load_balancer.ts)"
$CALL PUT ${ADMIN_URL}/module \
    name=printer \
    payload=$MOD_B64 \
    namespace=test-lb-ns


http -m PUT ${ADMIN_URL}/service \
    name=base_svc \
    urls:='["http://localhost:8888/a","http://localhost:8888/b","http://localhost:8888/c"]' \
    namespace=test-lb-ns

$CALL PUT ${ADMIN_URL}/route \
    name=base_rt \
    paths:='["/test-lb","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    service=base_svc \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-lb-ns

http -m -p=hbm ${PROXY_URL}/test-lb Host:test-lb.com

http -m -p=hbm ${PROXY_URL}/test-lb Host:test-lb.com X-Forwarded-For:192.168.0.1
