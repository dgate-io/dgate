#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

CALL='http --check-status -p=mb -F'

$CALL PUT ${ADMIN_URL}/namespace \
    name=test-ns

$CALL PUT ${ADMIN_URL}/domain \
    name=test-dm \
    patterns:='["test.com"]' \
    namespace=test-ns

MOD_B64="$(base64 < $DIR/modify_response.ts)"
$CALL PUT ${ADMIN_URL}/module \
    name=printer payload=$MOD_B64 \
    namespace=test-ns

$CALL PUT ${ADMIN_URL}/service \
    name=base_svc \
    urls:='["http://localhost:8888"]' \
    namespace=test-ns

$CALL PUT ${ADMIN_URL}/route \
    name=base_rt \
    paths:='["/test","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-ns \
    service='base_svc'

http -m -p=hmb ${PROXY_URL}/test Host:test.com
