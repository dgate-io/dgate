#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

dgate-cli namespace create \
    name=test-ns

dgate-cli domain create \
    name=test-dm \
    patterns:='["test.com"]' \
    namespace=test-ns

dgate-cli domain

MOD_B64="$(base64 < $DIR/merge_responses.ts)"
dgate-cli module create \
    name=printer \
    payload=$MOD_B64 \
    namespace=test-ns

dgate-cli route create \
    name=base_rt \
    paths:='["/test","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-ns #\ service='base_svc'

curl ${PROXY_URL}/hello -H Host:test.com

echo "Merge Responses Test Passed"
