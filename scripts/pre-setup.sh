#!/bin/bash

set -x

load_provider_environment

systemctl enable snapd
systemctl restart snapd