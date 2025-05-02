#!/bin/bash

exec   > >(tee -ia /var/log/canonical-bootstrap.log)
exec  2> >(tee -ia /var/log/canonical-bootstrap.log >& 2)
exec 19>> /var/log/canonical-bootstrap.log

export BASH_XTRACEFD="19"
set -ex

cd /opt/canonical-k8s
snap ack snapd.assert && sudo snap install ./snapd.snap
snap ack core20.assert && sudo snap install ./core20.snap --classic
snap ack k8s.assert && sudo snap install ./k8s.snap --classic

until k8s bootstrap --file /opt/canonical/bootstrap-config.yaml > /dev/null
do
  echo "retrying in 10s"
  sleep 10;
done

k8s status --wait-ready

snap refresh k8s --hold

touch /opt/canonical/canonical.bootstrap