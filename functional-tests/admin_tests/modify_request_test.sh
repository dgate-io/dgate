#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

dgate-cli namespace create \
    name=modify_request_test-ns

dgate-cli domain create \
    name=modify_request_test-dm \
    patterns:='["modify_request_test.com"]' \
    namespace=modify_request_test-ns

MOD_B64="$(base64 < $DIR/modify_request.ts)"
dgate-cli module create \
    name=printer payload="$MOD_B64" \
    namespace=modify_request_test-ns

dgate-cli service create \
    name=base_svc \
    urls:='["http://localhost:8888"]' \
    namespace=modify_request_test-ns

dgate-cli route create \
    name=base_rt \
    paths:='["/modify_request_test"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=modify_request_test-ns \
    service='base_svc'

curl http://localhost/modify_request_test \
    -H Host:modify_request_test.com \
    -H X-Forwarded-For:1.1.1.1

echo "Modify Request Test Passed"
