#!/bin/bash

set -eo xtrace

ADMIN_URL1=${ADMIN_URL1:-"http://localhost:9081"}
PROXY_URL1=${PROXY_URL1:-"http://localhost:81"}

ADMIN_URL2=${ADMIN_URL2:-"http://localhost:9082"}
PROXY_URL2=${PROXY_URL2:-"http://localhost:82"}

ADMIN_URL3=${ADMIN_URL3:-"http://localhost:9083"}
PROXY_URL3=${PROXY_URL3:-"http://localhost:83"}

ADMIN_URL4=${ADMIN_URL4:-"http://localhost:9084"}
PROXY_URL4=${PROXY_URL4:-"http://localhost:84"}

ADMIN_URL5=${ADMIN_URL5:-"http://localhost:9085"}
PROXY_URL5=${PROXY_URL5:-"http://localhost:85"}


DIR="$( cd "$( dirname "$0" )" && pwd )"

# domain setup

export DGATE_ADMIN_API=$ADMIN_URL1


id=$(uuid)

dgate-cli -f namespace create name=ns-$id

dgate-cli -f domain create name=dm-$id \
    namespace=ns-$id priority:=$RANDOM patterns="$id.example.com"

dgate-cli -f service create \
    name=svc-$id namespace=ns-$id \
    urls="http://localhost:8888/$RANDOM"

dgate-cli -f route create \
    name=rt-$id \
    service=svc-$id \
    namespace=ns-$id \
    paths="/$id/{id}" \
    methods:='["GET"]' \
    preserveHost:=false \
    stripPath:=true

for i in {1..5}; do
    curl -f $PROXY_URL1/$id/$i -H Host:$id.example.com
    curl -f $PROXY_URL2/$id/$i -H Host:$id.example.com
    curl -f $PROXY_URL3/$id/$i -H Host:$id.example.com
    curl -f $PROXY_URL4/$id/$i -H Host:$id.example.com
    curl -f $PROXY_URL5/$id/$i -H Host:$id.example.com
done

dgate-cli -V --admin $ADMIN_URL1 route get name=rt-$id namespace=ns-$id
dgate-cli -V --admin $ADMIN_URL2 route get name=rt-$id namespace=ns-$id
dgate-cli -V --admin $ADMIN_URL3 route get name=rt-$id namespace=ns-$id
dgate-cli -V --admin $ADMIN_URL4 route get name=rt-$id namespace=ns-$id
dgate-cli -V --admin $ADMIN_URL5 route get name=rt-$id namespace=ns-$id


if dgate-cli --admin $ADMIN_URL4 namespace create name=0; then
    echo "Expected error when creating namespace"
    exit 1
fi

export DGATE_ADMIN_API=$ADMIN_URL5

if dgate-cli --admin $ADMIN_URL5 namespace create name=0; then
    echo "Expected error when creating namespace"
    exit 1
fi

echo "Raft Test Succeeded"
