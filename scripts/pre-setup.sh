#!/bin/bash

set -x

load_provider_environment

# legacy vs systemd-extension compatibility
# Check if /usr/lib/containerd/config.toml exists. If it exists, copy it to /etc/containerd/config.toml.
# Canonical systemd extensions store config.toml in /usr/lib/containerd.
if [[ -f /usr/lib/containerd/config.toml ]]; then
    mkdir -p /etc/containerd
    cp -f /usr/lib/containerd/config.toml /etc/containerd/config.toml
fi

systemctl enable snapd
systemctl restart snapd