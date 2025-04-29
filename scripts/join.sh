#!/bin/bash

exec   > >(tee -ia /var/log/canonical-join.log)
exec  2> >(tee -ia /var/log/canonical-join.log >& 2)
exec 19>> /var/log/canonical-join.log

export BASH_XTRACEFD="19"
set -ex

token=$1
advertise_address=$2

cd /opt/canonical-k8s
snap ack core20.assert && sudo snap install ./core20.snap
snap ack k8s.assert && sudo snap install ./k8s.snap --classic

join_cmd=''

if [ -n "$advertise_address" ]; then
  join_cmd="k8s join-cluster $token --file /opt/canonical/join-config.yaml --address $advertise_address"
else
  join_cmd="k8s join-cluster $token --file /opt/canonical/join-config.yaml"
fi

until $join_cmd > /dev/null
do
  echo "retrying in 10s"
  sleep 10;
done

k8s status --wait-ready

snap refresh k8s --hold

touch /opt/canonical/canonical.join