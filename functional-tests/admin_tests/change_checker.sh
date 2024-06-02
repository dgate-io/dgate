#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli namespace create \
    name=change_checker-ns

dgate-cli domain create \
    name=change_checker-dm \
    patterns:='["change_checker.com"]' \
    namespace=change_checker-ns

dgate-cli module create name=change_checker-mod \
    payload@=$DIR/change_checker_1.ts \
    namespace=change_checker-ns

dgate-cli route create \
    name=base_rt paths:='["/", "/{id}"]' \
    modules:='["change_checker-mod"]' \
    methods:='["GET","POST"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=change_checker-ns

MODID1=$(curl -sG -H Host:change_checker.com ${PROXY_URL}/ | jq -r '.mod')

if [ "$MODID1" != "module1" ]; then
    echo "Initial assert failed"
    exit 1
fi


dgate-cli module create name=change_checker-mod \
    payload@=$DIR/change_checker_2.ts \
    namespace=change_checker-ns

# dgate-cli r.ker-ns

MODID2=$(curl -sG -H Host:change_checker.com ${PROXY_URL}/ | jq -r '.mod')

if [ "$MODID2" != "module2" ]; then
    echo "module update failed"
    exit 1
fi