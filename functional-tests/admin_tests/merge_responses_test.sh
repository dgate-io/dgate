#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli namespace create \
    name=test-ns

dgate-cli domain create \
    name=test-dm \
    patterns:='["test.com"]' \
    namespace=test-ns

MOD_B64="$(base64 < $DIR/merge_responses.ts)"
dgate-cli module create \
    name=printer \
    payload="$MOD_B64" \
    namespace=test-ns

dgate-cli route create \
    name=base_rt \
    paths:='["/test","/hello"]' \
    methods:='["GET"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=test-ns

curl -s --fail-with-body ${PROXY_URL}/hello -H Host:test.com

echo "Merge Responses Test Passed"
