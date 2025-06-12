#!/bin/bash

exec   > >(tee -ia /var/log/canonical-join.log)
exec  2> >(tee -ia /var/log/canonical-join.log >& 2)
exec 19>> /var/log/canonical-join.log

export BASH_XTRACEFD="19"
set -ex

token=$1
advertise_address=$2
node_role=$3

snap wait system seed.loaded

snapd_revision=$(cat /opt/canonical/snapd.revision)
core_revision=$(cat /opt/canonical/core.revision)
k8s_revision=$(cat /opt/canonical/k8s.revision)

cd /opt/canonical-k8s
snap ack snapd_"${snapd_revision}".assert && sudo snap install ./snapd_"${snapd_revision}".snap
snap ack core*_"${core_revision}".assert && sudo snap install ./core*_"${core_revision}".snap --classic
snap ack k8s_"${k8s_revision}".assert && sudo snap install ./k8s_"${k8s_revision}".snap --classic


join_cmd=''

if [ -n "$advertise_address" ]; then
  join_cmd="k8s join-cluster $token --file /opt/canonical/join-config.yaml --address $advertise_address"
else
  join_cmd="k8s join-cluster $token --file /opt/canonical/join-config.yaml"
fi

until eval "$join_cmd" > /dev/null
do
  echo "retrying in 10s"
  sleep 10;
done

if [ "$node_role" != "worker" ]; then
  until k8s status --wait-ready
  do
    echo "waiting for status"
    sleep 10
  done
fi

snap refresh k8s --hold

touch /opt/canonical/canonical.join