#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

dgate-cli namespace create \
    name=url_shortener-ns

dgate-cli domain create \
    name=url_shortener-dm \
    patterns:='["url_shortener.com"]' \
    namespace=url_shortener-ns

dgate-cli collection create \
    schema:='{"type":"object","properties":{"url":{"type":"string"}}}' \
    name=short_link \
    type=document \
    namespace=url_shortener-ns

MOD_B64="$(base64 < $DIR/url_shortener.ts)"
dgate-cli module create \
    name=printer \
    payload="$MOD_B64" \
    namespace=url_shortener-ns

dgate-cli route create \
    name=base_rt \
    paths:='["/test","/hello"]' \
    methods:='["GET","POST"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=url_shortener-ns #\ service='base_svc'

JSON_RESP=$(curl -sG -X POST -H Host:url_shortener.com ${PROXY_URL}/test --data-urlencode "url=${PROXY_URL}/hello")
echo $JSON_RESP

URL_ID=$(echo $JSON_RESP | jq -r '.id')

curl -s --fail-with-body ${PROXY_URL}/test\?id=$URL_ID -H Host:url_shortener.com
