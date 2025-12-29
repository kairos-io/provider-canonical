#!/bin/bash

source "$(dirname "$0")/common.sh"
setup_logging /var/log/canonical-join.log
set -u

token=$1
advertise_address=$2
node_role=$3

log "starting canonical k8s join"

install_all_snaps

join_cmd="k8s join-cluster $token --file /opt/canonical/join-config.yaml"
if [ -n "$advertise_address" ]; then
  join_cmd="$join_cmd --address $advertise_address"
fi

log "joining k8s cluster with command: $join_cmd"

with_retry "k8s join-cluster" eval "$join_cmd"

if [ "$node_role" != "worker" ]; then
  wait_for_k8s_ready
fi

hold_k8s_snap_refresh

touch /opt/canonical/canonical.join
