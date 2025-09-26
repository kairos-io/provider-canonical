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

# snapd (exact file names)
until [[ -f "snapd_${snapd_revision}.assert" && -f "./snapd_${snapd_revision}.snap" ]] && \
      sudo snap ack "snapd_${snapd_revision}.assert" && \
      sudo snap install "./snapd_${snapd_revision}.snap"; do
  echo "waiting for snapd files or retrying install in 10s"
  sleep 10
done

# core (wildcard, tolerate file not existing yet)
until core_assert=( core*_"${core_revision}".assert ) && \
      core_snap=( ./core*_"${core_revision}".snap ) && \
      [[ ${#core_assert[@]} -ge 1 && ${#core_snap[@]} -ge 1 ]] && \
      sudo snap ack "${core_assert[0]}" && \
      sudo snap install "${core_snap[0]}" --classic; do
  echo "waiting for core files or retrying install in 10s"
  sleep 10
done

# k8s (exact file names)
until [[ -f "k8s_${k8s_revision}.assert" && -f "./k8s_${k8s_revision}.snap" ]] && \
      sudo snap ack "k8s_${k8s_revision}.assert" && \
      sudo snap install "./k8s_${k8s_revision}.snap" --classic; do
  echo "waiting for k8s files or retrying install in 10s"
  sleep 10
done

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