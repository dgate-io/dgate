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

dgate-cli -Vf namespace create name=ns-$id

dgate-cli -Vf domain create name=dm-$id \
    namespace=ns-$id priority:=$RANDOM patterns="$id.example.com"

dgate-cli -Vf service create \
    name=svc-$id namespace=ns-$id \
    urls="http://localhost:8081/$RANDOM"

dgate-cli -Vf route create \
    name=rt-$id \
    service=svc-$id \
    namespace=ns-$id \
    paths="/,/{},/$id,/$id/{id}" \
    methods=GET,POST,PUT \
    preserveHost:=false \
    stripPath:=false

curl -f $ADMIN_URL1/readyz

for i in {1..1}; do
    for j in {1..3}; do
        proxy_url=PROXY_URL$i
        curl -f ${!proxy_url}/$id/$RANDOM-$j -H Host:$id.example.com
    done
done

# if dgate-cli --admin $ADMIN_URL4 namespace create name=0; then
#     echo "Expected error when creating namespace on non-voter"
#     exit 1
# fi

# export DGATE_ADMIN_API=$ADMIN_URL5

# if dgate-cli --admin $ADMIN_URL5 namespace create name=0; then
#     echo "Expected error when creating namespace on non-voter"
#     exit 1
# fi

echo "Raft Test Succeeded"
