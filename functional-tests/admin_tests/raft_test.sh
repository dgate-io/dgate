#!/bin/bash

set -ea

DIR="$( cd "$( dirname "$0" )" && pwd )"

ADMIN_URL=${ADMIN_URL:-"http://localhost:9081/api/v1"}
PROXY_URL=${PROXY_URL:-"http://localhost:81"}

# SETUP BASE

# make sure we're talking to the leader
dgate-cli raftadmin/VerifyLeader

. $DIR/namespace_test.sh

set +a