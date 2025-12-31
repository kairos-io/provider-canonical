#!/bin/bash

source "$(dirname "$0")/common.sh"
setup_logging /var/log/canonical-join.log
set -u

token=$1
advertise_address=$2
node_role=$3

log "starting canonical k8s join"

install_all_snaps

join_cmd="k8s join-cluster $token --file /opt/canonical/join-config.yaml"
if [ -n "$advertise_address" ]; then
  join_cmd="$join_cmd --address $advertise_address"
fi

# -------- BEGIN: Token refresh logic (PE-7944 - remove when upstream issue is fixed) --------

# override if required in refresh_token
CLUSTER_TOKEN=$token

CERT_DIR="/oem/.spectrocloud/mtls"

hostname=$(cat /etc/hostname)

get_publisher_id(){
  # Get node name from /run/stylus/userdata stylus.site.name
  local userdata_file="/run/stylus/userdata"
  local edge_id=""
  if [ -f "$userdata_file" ]; then
    edge_id=$(grep -A5 'site:' "$userdata_file" | grep 'name:' | head -1 | sed 's/.*name:[[:space:]]*//' | tr -d '[:space:]')
  fi
  echo "$edge_id"
}

get_port(){
  # Get port from /run/stylus/userdata stylus.localUI.port, default 5080
  local userdata_file="/run/stylus/userdata"
  local api_port="5080"
  if [ -f "$userdata_file" ]; then
    local parsed_port=$(grep -A5 'localUI:' "$userdata_file" | grep 'port:' | head -1 | sed 's/.*port:[[:space:]]*//' | tr -d '[:space:]')
    [ -n "$parsed_port" ] && api_port="$parsed_port"
  fi
  echo "$api_port"
}

# Fetches cluster token from leader node, retries until success
# Usage: fetch_cluster_token <base64_token> <node_role>
# Output: prints token to stdout
fetch_cluster_token() {
  local encoded_token="$1"
  local role="$2"
  local edge_id=$(get_publisher_id)
  local cert_opts=""

  # Decode base64 token
  local decoded_token=$(echo "$encoded_token" | base64 -d 2>/dev/null)
  if [ -z "$decoded_token" ]; then
    log "ERROR: Failed to decode base64 token"
    return 1
  fi

  # Get join addresses
  local join_addresses=$(echo "$decoded_token" | jq -r '.join_addresses[]' 2>/dev/null)
  if [ -z "$join_addresses" ]; then
    log "ERROR: No join_addresses found in token"
    return 1
  fi

  # Build cert options
  if [ -f "$CERT_DIR/spectro-client.crt" ] && \
     [ -f "$CERT_DIR/spectro-client.key" ] && \
     [ -f "$CERT_DIR/spectro-ca.crt" ]; then
    cert_opts="--cert $CERT_DIR/spectro-client.crt --key $CERT_DIR/spectro-client.key --cacert $CERT_DIR/spectro-ca.crt"
  else
    log "WARNING: mTLS certs not found, using insecure"
    cert_opts="--insecure"
  fi

  # Retry until we get a token

  while true; do
    for addr in $join_addresses; do
      local ip="${addr%:*}"
      
      # Get port from /run/stylus/userdata stylus.localUI.port if overridden, default 5080
      local api_port=$(get_port)  
      
      local url="https://internal.spectrocloud.com:${api_port}/v1/internal/edgehosts/current/actions/cluster-token"
      local request_body=$(jq -n \
        --arg expiry "24h" \
        --arg engine "canonical" \
        --arg name "$hostname" \
        --arg role "$role" \
        '{expiry: $expiry, k8sEngine: $engine, nodeName: $name, nodeRole: $role}')

      log "=== Request ==="
      log "URL: $url (will be resolved to $ip:$api_port)"
      log "Headers: Content-Type: application/json, Accept: application/json, Publisher-Host-Id: ${edge_id}"
      log "Body: $request_body"
      log "Certs: $cert_opts"

      local response=$(curl --silent --show-error --max-time 30 \
        --location "$url" \
        --resolve "internal.spectrocloud.com:${api_port}:${ip}" \
        --noproxy internal.spectrocloud.com \
        --header 'Content-Type: application/json' \
        --header 'Accept: application/json' \
        --header "Publisher-Host-Id: ${edge_id}" \
        $cert_opts \
        --data "$request_body" 2>&1)

      if [ $? -eq 0 ] && [ -n "$response" ]; then
        echo "got response: $response"
        local local_token=$(echo "$response" | jq -r '.token // empty' 2>/dev/null)
        if [ -n "$local_token" ]; then
          CLUSTER_TOKEN=$local_token
          return 0
        fi
        local err=$(echo "$response" | jq -r '.message // .error // empty' 2>/dev/null)
        [ -n "$err" ] && log "Error from $ip: $err"
      fi
    done

    log "All addresses failed, waiting 10s..."
    sleep 10
  done
}

# Join cluster with automatic token refresh on CoreTokenRecord errors
join_with_token_refresh() {
  local delay="${RETRY_DELAY:-10}"
  local token_error_count=0
  local token_error_threshold=3

  local cmd="$join_cmd"

  log "Join command: $cmd"

  local output
  until output=$(eval "$cmd" 2>&1); do
    log "Join failed: $output"

    if echo "$output" | grep -q "CoreTokenRecord not found"; then
      token_error_count=$((token_error_count + 1))
      log "CoreTokenRecord error ($token_error_count/$token_error_threshold)"

      if [ "$token_error_count" -ge "$token_error_threshold" ]; then
        log "Refreshing token..."
        fetch_cluster_token "$token" "$node_role"
        cmd="k8s join-cluster $CLUSTER_TOKEN --file /opt/canonical/join-config.yaml"
        [ -n "$advertise_address" ] && cmd="$cmd --address $advertise_address"
        log "New join command: $cmd"
        token_error_count=0
      fi
    else
      log "Retrying in ${delay}s..."
      sleep "$delay"
    fi

    
  done

  log "k8s join-cluster succeeded"
}

join_with_token_refresh
# -------- END: Token refresh logic (PE-7944) --------

# TODO: uncomment this once token refresh is fixed upstream
# with_retry "k8s join-cluster" eval "$join_cmd"

if [ "$node_role" != "worker" ]; then
  wait_for_k8s_ready
fi

hold_k8s_snap_refresh

touch /opt/canonical/canonical.join
