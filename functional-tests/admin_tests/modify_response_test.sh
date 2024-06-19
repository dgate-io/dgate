#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli -Vf namespace create \
    name=test-ns

dgate-cli -Vf domain create \
    name=test-dm \
    patterns:='["test.example.com"]' \
    namespace=test-ns

MOD_B64="$(base64 < $DIR/modify_response.ts)"
dgate-cli -Vf module create \
    name=printer payload="$MOD_B64" \
    namespace=test-ns

dgate-cli -Vf service create \
    name=base_svc \
    urls:='["http://localhost:8888"]' \
    namespace=test-ns

dgate-cli -Vf route create \
    name=base_rt \
    paths:='["/test","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-ns \
    service='base_svc'

curl -s ${PROXY_URL}/test -H Host:test.example.com

echo "Modify Response Test Passed"
