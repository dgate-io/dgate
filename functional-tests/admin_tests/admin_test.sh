#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}
TEST_URL=${TEST_URL:-"http://localhost:8888"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

# domain setup
# check if uuid is available
if ! command -v uuid > /dev/null; then
    id=X$RANDOM-$RANDOM-$RANDOM
else
    id=$(uuid)
fi

dgate-cli -Vf namespace create name=ns-$id

dgate-cli -Vf domain create name=dm-$id \
    namespace=ns-$id priority:=$RANDOM patterns="$id.example.com"

dgate-cli -Vf service create \
    name=svc-$id namespace=ns-$id \
    urls="$TEST_URL/$RANDOM"

dgate-cli -Vf module create name=module1 \
    payload@=$DIR/admin_test.ts \
    namespace=ns-$id

dgate-cli -Vf route create \
    name=rt-$id \
    service=svc-$id \
    namespace=ns-$id \
    paths="/,/{id},/$id,/$id/{id}" \
    methods=GET,POST,PUT \
    modules=module1 \
    preserveHost:=false \
    stripPath:=false

curl -sf $ADMIN_URL/readyz > /dev/null

curl -f ${PROXY_URL}/$id/$RANDOM -H Host:$id.example.com

echo "Admin Test Succeeded"
