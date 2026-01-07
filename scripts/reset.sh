#!/bin/bash

set -x

load_provider_environment

node_role=$1
node_name=$(cat /etc/hostname)

if [ "$node_role" != "worker" ]; then
  k8s remove-node "$node_name"
fi

sleep 10

snap remove k8s --purge

# remove all core snaps
snap list | awk '/^core[0-9]+/ {print $1}' | xargs -n1 snap remove --purge

rm -rf /opt/canonical
rm -rf /opt/canonical-k8s
rm -rf /opt/containerd
rm -rf /opt/*init
rm -rf /opt/*join

rm -rf /etc/kubernetes/*

rm -rf /var/log/provider-canonical.log
rm -rf /var/log/canonical*.log
rm -rf /var/log/pods
