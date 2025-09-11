#!/bin/bash

set -x

CONTENT_PATH=$1

mkdir -p /var/snap/k8s/common/images

find -L "$CONTENT_PATH" -name "*.tar" -type f | while read -r tarfile; do
    cp "$tarfile" /var/snap/k8s/common/images/
done
