#!/bin/bash

set -x

CONTENT_PATH=$1

mkdir -p /var/snap/k8s/common/images

find "$CONTENT_PATH" -name "*.tar" -type f -exec cp {} /var/snap/k8s/common/images \;