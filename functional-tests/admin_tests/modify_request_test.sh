#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}
TEST_URL=${TEST_URL:-"http://localhost:8888"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli -Vf namespace create \
    name=modify_request_test-ns

dgate-cli -Vf domain create \
    name=modify_request_test-dm \
    patterns:='["modify_request_test.example.com"]' \
    namespace=modify_request_test-ns

MOD_B64="$(base64 < $DIR/modify_request.ts)"
dgate-cli -Vf module create \
    name=printer payload="$MOD_B64" \
    namespace=modify_request_test-ns

dgate-cli -Vf service create \
    name=base_svc \
    urls="$TEST_URL" \
    namespace=modify_request_test-ns

dgate-cli -Vf route create \
    name=base_rt \
    paths:='["/modify_request_test"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=modify_request_test-ns \
    service='base_svc'

curl -sf ${PROXY_URL}/modify_request_test \
    -H Host:modify_request_test.example.com \
    -H X-Forwarded-For:1.1.1.1

echo "Modify Request Test Passed"
