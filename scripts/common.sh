#!/bin/bash
#
# Common functions shared across bootstrap, join, and upgrade scripts.
#

# -------- Logging --------
setup_logging() {
  local log_file="$1"
  exec   > >(tee -ia "$log_file")
  exec  2> >(tee -ia "$log_file" >&2)
  exec 19>> "$log_file"
  export BASH_XTRACEFD="19"
}

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# -------- Retry helpers --------
with_retry() {
  local desc="$1"; shift
  local delay="${RETRY_DELAY:-10}"
  until "$@"; do
    log "${desc} failed; retrying in ${delay}s..."
    sleep "$delay"
  done
  log "${desc} succeeded"
}

# -------- Snap helpers --------
snap_is_busy() {
  snap changes 2>/dev/null | awk 'NR>1 {print $2}' | grep -qiE 'doing|undoing'
}

wait_for_snap_idle() {
  until ! snap_is_busy; do
    log "snapd has a change in progress; waiting 10s..."
    sleep 10
  done
}

# Install snapd snap
install_snapd() {
  local revision
  revision=$(cat /opt/canonical/snapd.revision)
  local assert_file="snapd_${revision}.assert"
  local snap_file="./snapd_${revision}.snap"

  if [[ ! -f "$assert_file" || ! -f "$snap_file" ]]; then
    log "snapd files not found yet"
    return 1
  fi

  wait_for_snap_idle
  sudo snap ack "$assert_file"
  sudo snap install "$snap_file"
}

# Install core snap (uses wildcard for core20/core22/core24)
install_core() {
  local revision
  revision=$(cat /opt/canonical/core.revision)

  shopt -s nullglob
  local assert_files=( core*_"${revision}".assert )
  local snap_files=( ./core*_"${revision}".snap )
  shopt -u nullglob

  if [[ ${#assert_files[@]} -eq 0 || ${#snap_files[@]} -eq 0 ]]; then
    log "core snap files not found yet"
    return 1
  fi

  wait_for_snap_idle
  sudo snap ack "${assert_files[0]}"
  sudo snap install "${snap_files[0]}"
}

# Install k8s snap
install_k8s() {
  local revision
  revision=$(cat /opt/canonical/k8s.revision)
  local assert_file="k8s_${revision}.assert"
  local snap_file="./k8s_${revision}.snap"

  if [[ ! -f "$assert_file" || ! -f "$snap_file" ]]; then
    log "k8s files not found yet"
    return 1
  fi

  wait_for_snap_idle
  sudo snap ack "$assert_file"
  sudo snap install "$snap_file" --classic
}

# Install all three snaps with retry
install_all_snaps() {
  snap wait system seed.loaded
  cd /opt/canonical-k8s

  with_retry "snapd install" install_snapd
  with_retry "core install" install_core
  with_retry "k8s install" install_k8s
}

# -------- K8s helpers --------
wait_for_k8s_ready() {
  until k8s status --wait-ready; do
    log "waiting for k8s status..."
    sleep 10
  done
  log "k8s is ready"
}

hold_k8s_snap_refresh() {
  log "holding k8s snap refresh"
  snap refresh k8s --hold
}
