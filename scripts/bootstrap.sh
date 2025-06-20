#!/bin/bash

exec   > >(tee -ia /var/log/canonical-bootstrap.log)
exec  2> >(tee -ia /var/log/canonical-bootstrap.log >& 2)
exec 19>> /var/log/canonical-bootstrap.log

export BASH_XTRACEFD="19"
set -ex

advertise_address=$1

snap wait system seed.loaded

snapd_revision=$(cat /opt/canonical/snapd.revision)
core_revision=$(cat /opt/canonical/core.revision)
k8s_revision=$(cat /opt/canonical/k8s.revision)

cd /opt/canonical-k8s
snap ack snapd_"${snapd_revision}".assert && sudo snap install ./snapd_"${snapd_revision}".snap
snap ack core*_"${core_revision}".assert && sudo snap install ./core*_"${core_revision}".snap --classic
snap ack k8s_"${k8s_revision}".assert && sudo snap install ./k8s_"${k8s_revision}".snap --classic

bootstrap_cmd=''

if [ -n "$advertise_address" ]; then
  bootstrap_cmd="k8s bootstrap --file /opt/canonical/bootstrap-config.yaml --address $advertise_address"
else
  bootstrap_cmd="k8s bootstrap --file /opt/canonical/bootstrap-config.yaml"
fi

until eval "$bootstrap_cmd" > /dev/null
do
  echo "retrying in 10s"
  sleep 10;
done

until k8s status --wait-ready
do
  echo "waiting for status"
  sleep 10
done

snap refresh k8s --hold

touch /opt/canonical/canonical.bootstrap