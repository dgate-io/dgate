#!/bin/bash

set -eo xtrace

ADMIN_URL=${ADMIN_URL:-"http://localhost:9080"}
PROXY_URL=${PROXY_URL:-"http://localhost"}

DIR="$( cd "$( dirname "$0" )" && pwd )"

export DGATE_ADMIN_API=$ADMIN_URL

dgate-cli namespace create \
    name=multimod-test-ns

dgate-cli domain create \
    name=multimod-test-dm \
    patterns:='["multimod-test.com"]' \
    namespace=multimod-test-ns

MOD_B64=$(base64 <<-END
export {
    requestModifier,
} from './multimod2';
import {
    responseModifier as resMod,
} from './multimod2';
const responseModifier = async (ctx) => {
    console.log('responseModifier executed from multimod1')
    return resMod(ctx);
};
END

)

dgate-cli module create \
    name=multimod1 \
    payload="$MOD_B64" \
    namespace=multimod-test-ns

MOD_B64=$(base64 <<-END
const reqMod = (ctx) => ctx.request().writeJson({a: 1});
const resMod = async (ctx) => ctx.upstream()?.writeJson({
    upstream_body: await ctx.upstream()?.readJson(),
    upstream_headers: ctx.upstream()?.headers,
    upsteam_status: ctx.upstream()?.statusCode,
    upstream_statusText: ctx.upstream()?.statusText,
});
export {
    reqMod as requestModifier,
    resMod as responseModifier,
};
END

)

dgate-cli module create name=multimod2 \
    payload="$MOD_B64" namespace=multimod-test-ns

URL='http://localhost:8888'
dgate-cli service create name=base_svc \
    urls="$URL/a","$URL/b","$URL/c" \
    namespace=multimod-test-ns

dgate-cli route create name=base_rt \
    paths=/,/multimod-test \
    methods:='["GET"]' \
    modules=multimod1,multimod2 \
    service=base_svc \
    stripPath:=true \
    preserveHost:=true \
    namespace=multimod-test-ns


curl -s --fail-with-body ${PROXY_URL}/ -H Host:multimod-test.com
curl -s --fail-with-body ${PROXY_URL}/multimod-test -H Host:multimod-test.com

echo "Multi Module Test Passed"