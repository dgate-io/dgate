#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli -Vf namespace create \
    name=url_shortener-ns

dgate-cli -Vf domain create \
    name=url_shortener-dm \
    patterns:='["url_shortener.example.com"]' \
    namespace=url_shortener-ns

dgate-cli -Vf collection create \
    schema:='{"type":"object","properties":{"url":{"type":"string"}}}' \
    name=short_link type=document namespace=url_shortener-ns

dgate-cli -Vf module create name=url_shortener-mod \
    payload@=$DIR/url_shortener.ts \
    namespace=url_shortener-ns

dgate-cli -Vf route create \
    name=base_rt paths:='["/", "/{id}"]' \
    modules:='["url_shortener-mod"]' \
    methods:='["GET","POST"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=url_shortener-ns

JSON_RESP=$(curl -fsG -X POST \
    -H Host:url_shortener.example.com ${PROXY_URL}/ \
    --data-urlencode 'url=https://dgate.io/'$(uuid))

URL_ID=$(echo $JSON_RESP | jq -r '.id')

curl -s --fail-with-body \
    ${PROXY_URL}/$URL_ID \
    -H Host:url_shortener.example.com
