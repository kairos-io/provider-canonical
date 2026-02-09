# Canonical Kubernetes Networking

## Overview

Provider-canonical networking includes **Calico CNI by default** (unlike kubeadm which requires manual CNI installation). This skill covers network configuration, Calico CNI, NetworkPolicy, and firewall requirements.

## Key Concepts

### Built-in Calico CNI

**Critical Difference from kubeadm**:
- **kubeadm**: NO CNI included - manual installation required
- **K3s**: Includes Flannel CNI by default
- **RKE2**: Includes Canal CNI (Calico + Flannel) by default
- **Canonical K8s**: **Includes Calico CNI by default**

**Implications**:
1. After `k8s bootstrap`, nodes immediately become `Ready`
2. Calico provides NetworkPolicy support out-of-the-box
3. No manual CNI installation step required

### Network Layers

**Provider-canonical networking has two separate layers**:

1. **Node-to-Node (Host Networking)**:
   - Physical network interfaces (eth0, eth1)
   - Optional: Stylus overlay (VxLAN for multi-site)
   - Control plane communication (API server port 6443)

2. **Pod-to-Pod (Calico CNI)**:
   - Calico CNI (included by default)
   - Pod IP assignment from pod-cidr
   - Service IP assignment from service-cidr
   - NetworkPolicy enforcement

## Implementation Patterns

### Pattern 1: Default Network Configuration

**Use Case**: Standard pod and service networking with Calico.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  # Uses default CIDRs:
  # pod-cidr: 10.1.0.0/16
  # service-cidr: 10.152.183.0/24
```

**Execution**:
```bash
# Bootstrap with defaults
k8s bootstrap

# Verify Calico is running
k8s kubectl get pods -n kube-system -l k8s-app=calico-node
# NAME               READY   STATUS    RESTARTS   AGE
# calico-node-abc    1/1     Running   0          1m

# Nodes immediately Ready (Calico included)
k8s kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   2m    v1.28.0
```

---

### Pattern 2: Custom Network CIDRs

**Use Case**: Non-default pod/service subnets.

**Cloud-init Configuration**:
```yaml
cluster:
  role: init
  config: |
    pod-cidr: 10.100.0.0/16
    service-cidr: 10.200.0.0/16
```

**Execution**:
```bash
# Bootstrap with custom CIDRs
k8s bootstrap --pod-cidr 10.100.0.0/16 --service-cidr 10.200.0.0/16

# Calico automatically uses pod-cidr
k8s kubectl get ippool -o yaml
# spec:
#   cidr: 10.100.0.0/16
```

---

### Pattern 3: NetworkPolicy Enforcement

**Use Case**: Implement network segmentation and security policies.

**NetworkPolicy Example**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
  namespace: production
spec:
  podSelector: {}
  policyTypes:
  - Ingress
```

```bash
k8s kubectl apply -f deny-all-ingress.yaml

# Now pods in production namespace cannot receive traffic
# (unless explicitly allowed by other NetworkPolicy)
```

**Allow Specific Traffic**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-frontend
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: backend
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: frontend
    ports:
    - protocol: TCP
      port: 8080
```

---

### Pattern 4: Calico Network Modes

**Use Case**: Choose between VXLAN and BGP networking.

**VXLAN Mode** (Default):
```bash
# Calico uses VXLAN overlay by default
# UDP port 4789
# Encapsulates pod traffic
```

**BGP Mode** (Advanced):
```bash
# Disable VXLAN, enable BGP
k8s kubectl patch felixconfiguration default --type merge --patch='{"spec":{"vxlanEnabled":false}}'
k8s kubectl patch ippools default-ipv4-ippool --type merge --patch='{"spec":{"vxlanMode":"Never"}}'

# Requires BGP peering configuration
# TCP port 179 for BGP
```

---

### Pattern 5: Multi-Interface Nodes

**Use Case**: Nodes with multiple network interfaces.

**Cloud-init Configuration**:
```yaml
cluster:
  config: |
    # Specify interface for Calico
    calico-interface: eth1
```

**Manual Configuration**:
```bash
# Edit Calico daemonset
k8s kubectl edit daemonset calico-node -n kube-system

# Add env var
env:
- name: IP_AUTODETECTION_METHOD
  value: "interface=eth1"
```

---

## Firewall Configuration

### Required Ports

**Control Plane**:
- **6443**: Kubernetes API server (TCP)
- **2379-2380**: etcd client/peer (TCP)
- **10250**: Kubelet API (TCP)

**Worker Nodes**:
- **10250**: Kubelet API (TCP)
- **30000-32767**: NodePort services (TCP/UDP)

**Calico CNI**:
- **VXLAN mode**: UDP 4789
- **BGP mode**: TCP 179
- **Typha** (Calico component): TCP 5473

**Stylus Overlay** (if enabled):
- **UDP 4789**: Stylus VXLAN tunnels (same port as Calico VXLAN)

**Note**: Calico VXLAN and Stylus overlay both use UDP 4789 but on different interfaces - no conflict.

**Example firewalld Rules**:
```bash
# Control plane
firewall-cmd --permanent --add-port=6443/tcp
firewall-cmd --permanent --add-port=2379-2380/tcp
firewall-cmd --permanent --add-port=10250/tcp

# Worker
firewall-cmd --permanent --add-port=10250/tcp
firewall-cmd --permanent --add-port=30000-32767/tcp
firewall-cmd --permanent --add-port=30000-32767/udp

# Calico VXLAN
firewall-cmd --permanent --add-port=4789/udp

# Calico Typha
firewall-cmd --permanent --add-port=5473/tcp

firewall-cmd --reload
```

---

## Common Pitfalls

### ❌ WRONG: Expecting nodes to be NotReady (like kubeadm)

```bash
k8s bootstrap

# Wait for CNI installation...
# ❌ No need to wait - Calico included by default!

k8s kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   1m    v1.28.0
# ✅ Already Ready - Calico running
```

### ✅ CORRECT: Nodes immediately Ready

```bash
k8s bootstrap

k8s kubectl get nodes
# ✅ Nodes Ready immediately (Calico included)
```

---

### ❌ WRONG: Trying to install additional CNI

```bash
# Calico already included
k8s kubectl apply -f flannel.yml
# ❌ Conflict! Two CNIs running
```

### ✅ CORRECT: Use included Calico

```bash
# Calico already running - no additional CNI needed
k8s kubectl get pods -n kube-system -l k8s-app=calico-node
# ✅ Calico running by default
```

---

### ❌ WRONG: Firewall blocks Calico VXLAN

```bash
# Firewall blocking UDP 4789
k8s kubectl exec pod-1 -- ping 10.1.1.5  # Pod on node-2
# ❌ No response - VXLAN blocked
```

### ✅ CORRECT: Open UDP 4789

```bash
firewall-cmd --permanent --add-port=4789/udp
firewall-cmd --reload

# Test again
k8s kubectl exec pod-1 -- ping 10.1.1.5
# ✅ Success
```

---

## Troubleshooting

### Issue: Pods cannot communicate

**Symptom**:
```bash
k8s kubectl exec pod-1 -- ping 10.1.1.5
# No response
```

**Diagnosis**:
```bash
# Check Calico pods running
k8s kubectl get pods -n kube-system -l k8s-app=calico-node
# NAME               READY   STATUS    RESTARTS   AGE
# calico-node-abc    1/1     Running   0          5m

# Check routes
ip route | grep cali
# 10.1.1.0/26 via 10.1.1.1 dev vxlan.calico

# Check VXLAN interface
ip link show vxlan.calico
# vxlan.calico@NONE: <BROADCAST,MULTICAST,UP,LOWER_UP>
```

**Solution**: If routes/interfaces missing, check Calico logs:
```bash
k8s kubectl logs -n kube-system calico-node-abc
```

---

### Issue: NetworkPolicy not working

**Symptom**:
```bash
# Applied deny-all NetworkPolicy
k8s kubectl apply -f deny-all.yaml

# But pods can still communicate
k8s kubectl exec pod-1 -- curl pod-2:8080
# ✅ Success (should be blocked)
```

**Diagnosis**:
```bash
# Check NetworkPolicy exists
k8s kubectl get networkpolicy
# NAME           POD-SELECTOR   AGE
# deny-all       <none>         1m

# Check Calico felix logs
k8s kubectl logs -n kube-system -l k8s-app=calico-node
```

**Solution**: Ensure Calico felix is running and processing policies.

---

## Integration Points

### With Stylus Overlay

- Stylus overlay provides host-to-host connectivity across sites
- Calico CNI provides pod-to-pod connectivity within cluster
- Both use UDP 4789 but no conflict (different interfaces)

### With Proxy

- Proxy settings apply to Calico pods (image pulls)
- pod/service CIDRs must be in NO_PROXY
- Calico pod-to-pod traffic bypasses proxy

### With Deployment Modes

- Networking same for appliance and agent modes
- Calico configuration identical

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-canonical/pkg/provider/provider.go` - Network configuration

## Related Skills

- See `provider-canonical:01-architecture` for Calico CNI overview
- See `provider-canonical:02-configuration-patterns` for pod/service CIDR config
- See `provider-canonical:03-proxy-configuration` for proxy NO_PROXY with networks
- See `provider-canonical:06-troubleshooting` for network troubleshooting

**Related Provider Skills**:
- See `provider-kubeadm:02-cni-installation` for manual CNI installation (contrast)
- See `provider-rke2:05-networking` for Canal CNI (Calico + Flannel)

## Documentation References

**Calico Documentation**:
- https://docs.tigera.io/calico/latest/
- https://docs.tigera.io/calico/latest/networking/
- https://docs.tigera.io/calico/latest/network-policy/

**NetworkPolicy**:
- https://kubernetes.io/docs/concepts/services-networking/network-policies/
