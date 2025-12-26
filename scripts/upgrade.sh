#!/bin/bash

set -xeuo pipefail

exec   > >(tee -ia /var/log/canonical-upgrade.log)
exec  2> >(tee -ia /var/log/canonical-upgrade.log >& 2)
exec 19>> /var/log/canonical-bootstrap.log

export BASH_XTRACEFD="19"

export KUBECONFIG=/etc/kubernetes/admin.conf
current_node_name=$(cat /etc/hostname)

# -------- inputs --------
node_role=$1

current_installed_revision=$(snap list k8s | grep k8s | awk '{print $3}')
echo "current k8s revision: $current_installed_revision"

upcoming_revision=$(cat /opt/canonical/k8s.revision)
if [ "$current_installed_revision" = "$upcoming_revision" ]; then
	echo "k8s is already up to date"
	exit 0
fi

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

snap_is_busy() {
	# Check for any running snapd changes
	snap changes 2>/dev/null | awk 'NR>1 {print $2}' | grep -qiE 'doing|undoing'
}

wait_for_snap_idle() {
	until ! snap_is_busy ; do
		echo "snapd has a change in progress; waiting 10s..."
		sleep 10
	done
}

acquire_lock() {
	if [ "$node_role" != "worker" ]; then
		until kubectl create configmap upgrade-lock -n kube-system --from-literal=node="${current_node_name}" > /dev/null; do
			upgrade_node=$(get_current_upgrading_node_name)
			if [ "$upgrade_node" = "$current_node_name" ]; then
				echo "resuming upgrade"
				break
			fi
			echo "failed to create configmap for upgrade lock, upgrading is going on the node ${upgrade_node}, retrying in 60 sec"
			sleep 60
		done
	fi
}

with_retry() {
	local desc="$1"; shift
	until "$@"; do
		echo "${desc} failed; retrying in 10s..."
		sleep 10
	done
}

do_upgrade() {
	acquire_lock

	snap wait system seed.loaded

	snapd_revision=$(cat /opt/canonical/snapd.revision)
	core_revision=$(cat /opt/canonical/core.revision)
	k8s_revision=$(cat /opt/canonical/k8s.revision)

	cd /opt/canonical-k8s

	with_retry "snapd install" bash -c "
		wait_for_snap_idle
		snap ack snapd_${snapd_revision}.assert &&
		sudo snap install ./snapd_${snapd_revision}.snap
	"
	with_retry "core install" bash -c "
		wait_for_snap_idle
		snap ack core20_${core_revision}.assert &&
		sudo snap install ./core20_${core_revision}.snap
	"
	with_retry "k8s install" bash -c "
		wait_for_snap_idle
		snap ack "k8s_${k8s_revision}.assert" &&
		sudo snap install "./k8s_${k8s_revision}.snap" --classic
	"

	if [ "$node_role" != "worker" ]; then
		until k8s status --wait-ready
		do
			echo "waiting for status"
			sleep 10
		done
	fi

	snap refresh k8s --hold

	delete_lock_config_map
}

do_upgrade