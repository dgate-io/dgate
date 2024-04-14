#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

CALL='http --ignore-stdin --check-status -p=mb -F'

$CALL PUT ${ADMIN_URL}/namespace \
    name=modify_request_test-ns

$CALL PUT ${ADMIN_URL}/domain \
    name=modify_request_test-dm \
    patterns:='["modify_request_test.com"]' \
    namespace=modify_request_test-ns

MOD_B64="$(base64 < $DIR/modify_request.ts)"
$CALL PUT ${ADMIN_URL}/module \
    name=printer payload="$MOD_B64" \
    namespace=modify_request_test-ns

$CALL PUT ${ADMIN_URL}/service \
    name=base_svc \
    urls:='["http://localhost:8888"]' \
    namespace=modify_request_test-ns

$CALL PUT ${ADMIN_URL}/route \
    name=base_rt \
    paths:='["/modify_request_test"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=modify_request_test-ns \
    service='base_svc'

http -m -p=hmb ${PROXY_URL}/modify_request_test Host:modify_request_test.com X-Forwarded-For:1.1.1.1
