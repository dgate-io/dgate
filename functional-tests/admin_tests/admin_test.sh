#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

# domain setup

id=$(uuid)

dgate-cli namespace create name=ns-$id

dgate-cli domain create name=dm-$id \
    namespace=ns-$id priority:=$RANDOM patterns="$id.example.com"

dgate-cli service create \
    name=svc-$id namespace=ns-$id \
    urls="http://localhost:8888/$RANDOM"

dgate-cli module create name=module1 \
    payload@=$DIR/admin_test.ts \
    namespace=ns-$id

dgate-cli route create \
    name=rt-$id \
    service=svc-$id \
    namespace=ns-$id \
    paths="/,/{},/$id,/$id/{id}" \
    methods=GET,POST,PUT \
    modules=module1 \
    preserveHost:=false \
    stripPath:=false

curl -f $ADMIN_URL/readyz

curl -f ${PROXY_URL}/$id/$RANDOM-$j -H Host:$id.example.com

echo "Admin Test Succeeded"