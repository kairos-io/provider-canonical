#!/bin/bash

set -x

node_name=$(cat /etc/hostname)

k8s remove-node "$node_name"

sleep 10

snap remove k8s --purge
snap remove core20 --purge

rm -rf /opt/canonical
rm -rf /opt/canonical-k8s
rm -rf /opt/containerd
rm -rf /opt/*init
rm -rf /opt/*join

rm -rf /etc/kubernetes/*

rm -rf /var/log/provider-canonical.log
rm -rf /var/log/canonical*.log
rm -rf /var/log/pods
