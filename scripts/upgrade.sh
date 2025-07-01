#!/bin/bash

set -x

trap 'echo -n $(date)' DEBUG

exec   > >(tee -ia /var/log/canonical-upgrade.log)
exec  2> >(tee -ia /var/log/canonical-upgrade.log >& 2)

node_role=$1
current_node_name=$(cat /etc/hostname)

current_installed_revision=$(snap list k8s | grep k8s | awk '{print $3}')
echo "current k8s revision: $current_installed_revision"

upcoming_revision=$(cat /opt/canonical/k8s.revision)
if [ "$current_installed_revision" = "$upcoming_revision" ]; then
    echo "k8s is already up to date"
    exit 0
fi

export KUBECONFIG=/etc/kubernetes/admin.conf

get_current_upgrading_node_name() {
  kubectl get configmap upgrade-lock -n kube-system -o jsonpath="{['data']['node']}"
}

delete_lock_config_map() {
  # Delete the configmap lock once the upgrade completes
  if [ "$node_role" != "worker" ]
  then
    kubectl delete configmap upgrade-lock -n kube-system
  fi
}

do_upgrade() {
    if [ "$node_role" != "worker" ]; then
      until kubectl create configmap upgrade-lock -n kube-system --from-literal=node="${current_node_name}" > /dev/null
      do
        upgrade_node=$(get_current_upgrading_node_name)
        if [ "$upgrade_node" = "$current_node_name" ]; then
          echo "resuming upgrade"
          break
        fi
        echo "failed to create configmap for upgrade lock, upgrading is going on the node ${upgrade_node}, retrying in 60 sec"
        sleep 60
      done
    fi

    snap wait system seed.loaded

    snapd_revision=$(cat /opt/canonical/snapd.revision)
    core_revision=$(cat /opt/canonical/core.revision)
    k8s_revision=$(cat /opt/canonical/k8s.revision)

    cd /opt/canonical-k8s
    snap ack snapd_"${snapd_revision}".assert && sudo snap install ./snapd_"${snapd_revision}".snap
    snap ack core*_"${core_revision}".assert && sudo snap install ./core*_"${core_revision}".snap --classic
    snap ack k8s_"${k8s_revision}".assert && sudo snap install ./k8s_"${k8s_revision}".snap --classic

    snap refresh k8s --hold

    delete_lock_config_map
}

do_upgrade