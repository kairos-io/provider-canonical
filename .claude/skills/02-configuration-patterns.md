---
skill: Canonical Kubernetes Configuration Patterns
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

# Canonical Kubernetes Configuration Patterns

## Overview

Provider-canonical uses snap-based Canonical Kubernetes with simplified configuration. This skill covers cluster configuration patterns for init/worker nodes, networking setup, and snap-specific options.

## Key Concepts

### Canonical K8s Configuration

Canonical Kubernetes uses **`k8s` snap commands** instead of kubeadm YAML files:

**Bootstrap** (init node):
```bash
k8s bootstrap
```

**Join** (worker/control plane):
```bash
k8s join-cluster <token>
```

**Configuration Files**: `/var/snap/k8s/current/args/`

### Cloud-init Integration

Provider-canonical processes cloud-init configuration:

```yaml
#cloud-config
cluster:
  cluster_token: "<join-token>"
  control_plane_host: "10.0.1.100"
  role: init  # or controlplane, or worker
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
  config: |
    # Canonical K8s specific configuration (if needed)
```

## Implementation Patterns

### Pattern 1: Init Node (Bootstrap First Node)

**Use Case**: First node that creates the cluster.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  control_plane_host: "10.0.1.100"   # This node's IP
  role: init
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    HTTPS_PROXY: "https://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1,.corp.com"
```

**Execution**:
```bash
# Provider installs snap
snap install k8s --classic

# Bootstrap cluster
k8s bootstrap

# Get join token for other nodes
k8s get-join-token
# eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Get kubeconfig
k8s config > ~/.kube/config

# Verify cluster
k8s kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   1m    v1.28.0
```

---

### Pattern 2: Worker Node

**Use Case**: Worker nodes running application pods.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  control_plane_host: "10.0.1.100"
  role: worker
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
```

**Execution**:
```bash
# Provider installs snap
snap install k8s --classic

# Join cluster as worker
k8s join-cluster eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Verify join
k8s kubectl get nodes
# NAME          STATUS   ROLES    AGE   VERSION
# control-1     Ready    <none>   5m    v1.28.0
# worker-1      Ready    <none>   1m    v1.28.0
```

---

### Pattern 3: ControlPlane Node (HA)

**Use Case**: Additional control plane nodes for HA.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  control_plane_host: "10.0.1.100"   # First control plane IP
  role: controlplane
```

**Execution**:
```bash
# Provider installs snap
snap install k8s --classic

# Join cluster as control plane
k8s join-cluster eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... --control-plane

# Verify HA setup
k8s kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   10m   v1.28.0
# control-2     Ready    control-plane   1m    v1.28.0
```

---

### Pattern 4: Custom Network CIDRs

**Use Case**: Non-default pod/service subnets.

**Canonical K8s Bootstrap with Custom CIDRs**:
```bash
# Bootstrap with custom pod subnet
k8s bootstrap --pod-cidr 10.100.0.0/16 --service-cidr 10.200.0.0/16
```

**Cloud-init Configuration**:
```yaml
cluster:
  role: init
  config: |
    pod-cidr: 10.100.0.0/16
    service-cidr: 10.200.0.0/16
```

**Calico Configuration**: Calico CNI automatically uses the pod-cidr specified during bootstrap.

---

### Pattern 5: Snap Version Control

**Use Case**: Control Canonical K8s version and updates.

**Install Specific Version**:
```bash
# Install specific channel (version track)
snap install k8s --classic --channel=1.28/stable

# Check available channels
snap info k8s
# channels:
#   1.29/stable:     1.29.0
#   1.28/stable:     1.28.5
#   1.27/stable:     1.27.10
```

**Disable Auto-Updates**:
```bash
# Hold snap at current version (no auto-updates)
snap refresh --hold k8s

# Check refresh status
snap refresh --time
```

**Cloud-init Configuration**:
```yaml
stages:
  boot.before:
    - commands:
      - snap install k8s --classic --channel=1.28/stable
      - snap refresh --hold k8s
```

---

## Common Pitfalls

### ❌ WRONG: Using kubeadm commands

```bash
kubeadm init
# kubeadm: command not found
# ❌ Canonical K8s doesn't use kubeadm
```

### ✅ CORRECT: Use k8s snap commands

```bash
k8s bootstrap
# ✅ Canonical K8s bootstrap command
```

---

### ❌ WRONG: Snap auto-update during cluster operation

```yaml
# No snap refresh hold
snap install k8s --classic
k8s bootstrap

# Days later: snap auto-updates to newer version
# ❌ Cluster upgraded unintentionally
```

### ✅ CORRECT: Hold snap version

```bash
snap install k8s --classic
snap refresh --hold k8s
# ✅ Snap won't auto-update
```

---

### ❌ WRONG: Modifying read-only snap files

```bash
vi /snap/k8s/current/bin/k8s
# Read-only file system
# ❌ Snap files are immutable
```

### ✅ CORRECT: Configure via /var/snap

```bash
vi /var/snap/k8s/current/args/kubelet
# ✅ Writable config location
```

---

## Integration Points

### With Provider

- Provider parses `cluster.role` and generates snap commands
- Provider creates role-specific shell scripts
- Provider merges user config with snap defaults

### With Proxy

- Provider configures proxy in `/etc/default/kubelet`
- Provider configures proxy for snap containerd service
- (See `03-proxy-configuration.md`)

### With CNI

- Calico CNI included by default
- No manual CNI installation required (unlike kubeadm)
- Calico automatically configured with pod-cidr from bootstrap

### With Stylus

- Appliance mode: Standard snap paths
- Agent mode: STYLUS_ROOT-prefixed config paths

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-canonical/pkg/provider/provider.go` - Config parsing
- `/Users/rishi/work/src/provider-canonical/scripts/` - Shell scripts

## Related Skills

- See `provider-canonical:01-architecture` for Canonical K8s overview
- See `provider-canonical:03-proxy-configuration` for proxy setup
- See `provider-canonical:04-deployment-modes` for appliance vs agent mode
- See `provider-canonical:05-networking` for Calico CNI details

## Documentation References

**Canonical Kubernetes**:
- https://canonical-kubernetes.readthedocs-hosted.com/
- https://snapcraft.io/k8s

**Snap Channels**:
- https://snapcraft.io/docs/channels
