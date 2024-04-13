#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

CALL='http --check-status -p=mhb -F'

$CALL PUT ${ADMIN_URL}/namespace \
    name=url_shortener-ns

$CALL PUT ${ADMIN_URL}/domain \
    name=url_shortener-dm \
    patterns:='["url_shortener.com"]' \
    namespace=url_shortener-ns

$CALL PUT ${ADMIN_URL}/collection \
    schema:='{"type":"object","properties":{"url":{"type":"string"}}}' \
    name=short_link \
    type=document \
    namespace=url_shortener-ns

MOD_B64="$(base64 < $DIR/url_shortener.ts)"
$CALL PUT ${ADMIN_URL}/module \
    name=printer \
    payload=$MOD_B64 \
    namespace=url_shortener-ns

$CALL PUT ${ADMIN_URL}/route \
    name=base_rt \
    paths:='["/test","/hello"]' \
    methods:='["GET","POST"]' \
    modules:='["printer"]' \
    stripPath:=true \
    preserveHost:=true \
    namespace=url_shortener-ns #\ service='base_svc'

$CALL POST \
    ${PROXY_URL}/test\?url\=${PROXY_URL}/hello \
    Host:url_shortener.com

URL_ID=$(http POST \
    ${PROXY_URL}/test\?url\=${PROXY_URL}/hello \
    Host:url_shortener.com | jq -re '.id')

http -m -p=hbm ${PROXY_URL}/test\?id\=$URL_ID Host:url_shortener.com
