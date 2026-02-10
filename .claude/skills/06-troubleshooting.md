---
skill: Canonical Kubernetes Troubleshooting
description: Skill documentation for provider-canonical
type: general
repository: provider-canonical
team: edge
topics: [kubernetes, provider, edge, cluster]
difficulty: intermediate
last_updated: 2026-02-09
related_skills: []
memory_references: []
---

# Canonical Kubernetes Troubleshooting

## Overview

This skill provides troubleshooting guidance for common provider-canonical issues including snap problems, Calico CNI issues, token management, and deployment mode complications.

## Common Issues

### Issue 1: Snap Installation Fails

**Symptom**:
```bash
snap install k8s --classic
# error: unable to contact snap store
```

**Root Causes**:
1. No internet connectivity
2. Proxy blocking snapcraft.io
3. Snap daemon not running

**Diagnosis**:
```bash
# Check internet connectivity
ping -c 3 snapcraft.io

# Check snap daemon
systemctl status snapd
# Active: active (running)

# Check proxy
env | grep -i proxy
```

**Solution**:
```bash
# If proxy issue - configure snap proxy
snap set system proxy.http="http://proxy.corp.com:8080"
snap set system proxy.https="http://proxy.corp.com:8080"

# Retry install
snap install k8s --classic

# OR install from local snap file (air-gap)
snap install k8s_1.28_amd64.snap --classic --dangerous
```

---

### Issue 2: Snap Auto-Update During Operation

**Symptom**:
```bash
# Deployed cluster yesterday
k8s kubectl version
# Server Version: v1.28.0

# Today
k8s kubectl version
# Server Version: v1.29.0
# ❌ Snap auto-updated to newer version!
```

**Root Cause**: Snap auto-refreshes by default.

**Diagnosis**:
```bash
# Check snap refresh status
snap refresh --time
# schedule: 00:00-04:59/06:00-11:59/12:00-17:59/18:00-23:59
# last: today at 02:00 UTC
# next: today at 18:00 UTC
```

**Solution**:
```bash
# Hold snap at current version
snap refresh --hold k8s

# Verify hold
snap info k8s | grep held
# held: 2024-02-08T10:00:00Z
```

**Prevention**: Always hold snap after installation:
```bash
snap install k8s --classic
snap refresh --hold k8s
```

---

### Issue 3: Token Expired

**Symptom**:
```bash
k8s join-cluster eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
# error: token has expired
```

**Root Cause**: Join tokens expire (default: 1 year).

**Diagnosis**:
```bash
# On control plane - check token validity
k8s get-join-token
# eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Decode token to check expiry (optional)
echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." | cut -d. -f2 | base64 -d | jq
```

**Solution**:
```bash
# Generate new token
k8s get-join-token
# eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... (new token)

# Join with new token
k8s join-cluster eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... (new token)
```

---

### Issue 4: Calico Pods Not Running

**Symptom**:
```bash
k8s kubectl get pods -n kube-system -l k8s-app=calico-node
# NAME               READY   STATUS             RESTARTS   AGE
# calico-node-abc    0/1     CrashLoopBackOff   5          3m
```

**Root Cause**: Calico configuration issue or network problem.

**Diagnosis**:
```bash
# Check Calico logs
k8s kubectl logs -n kube-system calico-node-abc

# Common errors:
# - Failed to reach API server
# - Network interface not found
# - VXLAN configuration error
```

**Solution**:
```bash
# If interface issue - specify interface
k8s kubectl edit daemonset calico-node -n kube-system

# Add env var
env:
- name: IP_AUTODETECTION_METHOD
  value: "interface=eth1"

# Restart Calico pods
k8s kubectl delete pod -n kube-system -l k8s-app=calico-node
```

---

### Issue 5: Proxy Blocks Container Registry

**Symptom**:
```bash
k8s bootstrap

# Check pod status
k8s kubectl get pods -n kube-system
# NAME                          READY   STATUS         RESTARTS   AGE
# coredns-abc123                0/1     ErrImagePull   0          2m
```

**Root Cause**: Proxy blocks container registry access.

**Diagnosis**:
```bash
# Check proxy settings
cat /etc/default/kubelet | grep PROXY

# Test registry access
curl -I https://rocks.canonical.com/
# ❌ Connection timeout
```

**Solution**:
```bash
# Add registry to NO_PROXY
export NO_PROXY="rocks.canonical.com,ghcr.io"

# OR configure proxy to allow registry access
```

---

### Issue 6: Pods Cannot Reach Services

**Symptom**:
```bash
k8s kubectl exec test-pod -- curl kubernetes.default.svc
# curl: (28) Connection timed out
```

**Root Cause**: Service CIDR not in NO_PROXY.

**Diagnosis**:
```bash
# Check NO_PROXY
cat /etc/default/kubelet | grep NO_PROXY
# NO_PROXY=localhost,127.0.0.1
# ❌ Missing service CIDR (10.152.183.0/24)
```

**Solution**:
```bash
# Provider should auto-append service CIDR
# If missing, add manually
echo 'NO_PROXY=10.1.0.0/16,10.152.183.0/24,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1' >> /etc/default/kubelet

systemctl restart snap.k8s.kubelet
```

---

### Issue 7: Wrong Snap Containerd Socket (Agent Mode)

**Symptom**:
```bash
# Agent mode (STYLUS_ROOT=/persistent/spectro)
k8s bootstrap
# error: cannot connect to containerd
```

**Root Cause**: Using wrong containerd socket for agent mode.

**Diagnosis**:
```bash
# Check STYLUS_ROOT
echo $STYLUS_ROOT
# /persistent/spectro  ← Agent mode

# Check containerd running
systemctl status snap.k8s.containerd
# OR
systemctl status snap.spectro-k8s.containerd
```

**Solution**: Ensure correct containerd variant for mode.

---

### Issue 8: NetworkPolicy Not Enforced

**Symptom**:
```bash
k8s kubectl apply -f deny-all.yaml

# But pods can still communicate
k8s kubectl exec pod-1 -- curl pod-2:8080
# ✅ Success (should be blocked)
```

**Root Cause**: Calico felix not processing policies.

**Diagnosis**:
```bash
# Check NetworkPolicy exists
k8s kubectl get networkpolicy
# NAME       POD-SELECTOR   AGE
# deny-all   <none>         1m

# Check Calico felix status
k8s kubectl exec -n kube-system calico-node-abc -- calico-node -felix-live
```

**Solution**:
```bash
# Check Calico logs
k8s kubectl logs -n kube-system -l k8s-app=calico-node

# Restart Calico if needed
k8s kubectl delete pod -n kube-system -l k8s-app=calico-node
```

---

## Diagnostic Commands

### Check Cluster Status
```bash
# Nodes
k8s kubectl get nodes -o wide

# Pods
k8s kubectl get pods -A -o wide

# Services
k8s kubectl get svc -A
```

### Check Snap Status
```bash
# Snap info
snap info k8s

# Snap services
snap services k8s
# Service                      Startup  Current
# k8s.containerd               enabled  active
# k8s.k8s-apiserver            enabled  active
# k8s.k8s-controller-manager   enabled  active
# k8s.kubelet                  enabled  active

# Snap logs
snap logs k8s -f
```

### Check Calico CNI
```bash
# Calico pods
k8s kubectl get pods -n kube-system -l k8s-app=calico-node

# Calico logs
k8s kubectl logs -n kube-system -l k8s-app=calico-node

# Calico status
k8s kubectl get felixconfiguration default -o yaml
k8s kubectl get ippools
```

### Check Networking
```bash
# Pod IP assignment
k8s kubectl get pods -A -o wide

# Test pod-to-pod
k8s kubectl run test-1 --image=nginx
k8s kubectl run test-2 --image=nginx
k8s kubectl exec test-1 -- ping -c 3 $(k8s kubectl get pod test-2 -o jsonpath='{.status.podIP}')

# DNS resolution
k8s kubectl exec test-1 -- nslookup kubernetes.default.svc
```

---

## Integration Points

### With Snap

- Most issues stem from snap management (updates, proxy, installation)
- Always verify snap daemon is running
- Check snap logs for detailed errors

### With Calico CNI

- Calico included by default (no manual installation)
- Check Calico logs for networking issues
- Verify VXLAN interface and routes exist

### With Deployment Modes

- Agent mode may require different containerd variant
- File paths differ between appliance and agent modes
- Verify STYLUS_ROOT is set correctly

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-canonical/scripts/` - Shell scripts with error handling

**Log Files**:
- Snap logs: `snap logs k8s`
- Kubelet logs: `k8s kubectl logs -n kube-system -l component=kubelet`
- Calico logs: `k8s kubectl logs -n kube-system -l k8s-app=calico-node`

## Related Skills

- See `provider-canonical:01-architecture` for Canonical K8s overview
- See `provider-canonical:02-configuration-patterns` for configuration details
- See `provider-canonical:03-proxy-configuration` for proxy setup
- See `provider-canonical:05-networking` for Calico CNI details

## Documentation References

**Canonical Kubernetes Troubleshooting**:
- https://canonical-kubernetes.readthedocs-hosted.com/en/latest/snap/howto/troubleshoot/

**Snap Troubleshooting**:
- https://snapcraft.io/docs/troubleshooting-snap-install

**Calico Troubleshooting**:
- https://docs.tigera.io/calico/latest/operations/troubleshoot/
