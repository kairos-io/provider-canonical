#!/bin/bash

set -x

do_dqlite_cleanup=$1

node_name=$(cat /etc/hostname)

if [ "$do_dqlite_cleanup" = true ]; then
  k8s remove-node "$node_name" --force
fi

snap remove k8s --purge
snap remove core20 --purge

rm -rf /opt/canonical
rm -rf /opt/canonical-k8s
rm -rf /opt/containerd
rm -rf /opt/*init
rm -rf /opt/*join

rm -rf /etc/kubernetes
