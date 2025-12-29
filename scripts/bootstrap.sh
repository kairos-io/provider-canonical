#!/bin/bash

source "$(dirname "$0")/common.sh"
setup_logging /var/log/canonical-bootstrap.log
set -ex

advertise_address=$1

install_all_snaps

bootstrap_cmd='k8s bootstrap --file /opt/canonical/bootstrap-config.yaml'
if [ -n "$advertise_address" ]; then
  bootstrap_cmd="$bootstrap_cmd --address $advertise_address"
fi

with_retry "k8s bootstrap" eval "$bootstrap_cmd"

wait_for_k8s_ready
hold_k8s_snap

touch /opt/canonical/canonical.bootstrap
