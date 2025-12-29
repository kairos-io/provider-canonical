#!/bin/bash

source "$(dirname "$0")/common.sh"
setup_logging /var/log/canonical-upgrade.log
set -u

export KUBECONFIG=/etc/kubernetes/admin.conf
current_node_name=$(cat /etc/hostname)

# -------- inputs --------
node_role=$1

log "starting canonical k8s upgrade"

current_installed_revision=$(snap list k8s | grep k8s | awk '{print $3}')

upcoming_revision=$(cat /opt/canonical/k8s.revision)
log "current installed k8s revision: $current_installed_revision"
log "upcoming k8s revision to install: $upcoming_revision"
if [ "$current_installed_revision" = "$upcoming_revision" ]; then
	log "k8s is already up to date"
	exit 0
fi

log "upgrading k8s from $current_installed_revision to $upcoming_revision"

get_current_upgrading_node_name() {
	kubectl get configmap upgrade-lock -n kube-system -o jsonpath="{['data']['node']}"
}

delete_lock_config_map() {
	# Delete the configmap lock once the upgrade completes
	if [ "$node_role" != "worker" ]; then
		kubectl delete configmap upgrade-lock -n kube-system
	fi
}

acquire_lock() {
	if [ "$node_role" != "worker" ]; then
		until kubectl create configmap upgrade-lock -n kube-system --from-literal=node="${current_node_name}" > /dev/null; do
			upgrade_node=$(get_current_upgrading_node_name)
			if [ "$upgrade_node" = "$current_node_name" ]; then
				log "resuming upgrade"
				break
			fi
			log "failed to create configmap for upgrade lock, upgrade in progress on node ${upgrade_node}; retrying in 60s..."
			sleep 60
		done
	fi
}

do_upgrade() {
	acquire_lock

	install_all_snaps

	if [ "$node_role" != "worker" ]; then
		wait_for_k8s_ready
	fi

	hold_k8s_snap_refresh

	delete_lock_config_map
}

do_upgrade
