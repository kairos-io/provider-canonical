# Canonical Kubernetes Proxy Configuration

## Overview

Provider-canonical supports HTTP/HTTPS proxy configuration similar to provider-kubeadm. Proxy settings are applied to kubelet and snap containerd, with automatic NO_PROXY calculation.

## Key Concepts

### Proxy Configuration Files

Provider-canonical creates two proxy configuration files:

1. **`/etc/default/kubelet`** - Kubelet environment variables
2. **`/etc/systemd/system/snap.k8s.containerd.service.d/http-proxy.conf`** - Snap containerd systemd override

### NO_PROXY Auto-Calculation

The provider automatically appends Kubernetes internal networks to user NO_PROXY:

- **podSubnet**: From k8s bootstrap --pod-cidr (default: 10.1.0.0/16)
- **serviceSubnet**: From k8s bootstrap --service-cidr (default: 10.152.183.0/24)
- **k8s service domains**: `.svc,.svc.cluster,.svc.cluster.local`

**Constant** (same as all providers):
```go
k8sNoProxy = ".svc,.svc.cluster,.svc.cluster.local"
```

## Implementation Patterns

### Pattern 1: Basic Proxy Configuration

**Use Case**: Simple corporate proxy.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    HTTPS_PROXY: "https://proxy.corp.com:8443"
    NO_PROXY: "localhost,127.0.0.1,.corp.com"
```

**Generated `/etc/default/kubelet`**:
```bash
HTTP_PROXY=http://proxy.corp.com:8080
HTTPS_PROXY=https://proxy.corp.com:8443
NO_PROXY=10.1.0.0/16,10.152.183.0/24,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,.corp.com
```

**Generated `/etc/systemd/system/snap.k8s.containerd.service.d/http-proxy.conf`**:
```ini
[Service]
Environment="HTTP_PROXY=http://proxy.corp.com:8080"
Environment="HTTPS_PROXY=https://proxy.corp.com:8443"
Environment="NO_PROXY=10.1.0.0/16,10.152.183.0/24,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,.corp.com"
```

**Systemd Service Reload**:
```bash
systemctl daemon-reload
systemctl restart snap.k8s.containerd
```

---

### Pattern 2: Agent Mode with Spectro Containerd

**Use Case**: Palette-managed clusters using spectro-containerd.

**Cloud-init Configuration**:
```yaml
cluster:
  role: init
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    STYLUS_ROOT: "/persistent/spectro"   # Agent mode
```

**Generated Files**:
1. **`${STYLUS_ROOT}/etc/default/kubelet`** = `/persistent/spectro/etc/default/kubelet`
2. **`/etc/systemd/system/snap.spectro-k8s.containerd.service.d/http-proxy.conf`** (if using spectro variant)

---

### Pattern 3: Snap-Specific Proxy

**Use Case**: Proxy for snap package manager itself (for snap installs/updates).

**Snap Proxy Configuration**:
```bash
# Set proxy for snap commands
snap set system proxy.http="http://proxy.corp.com:8080"
snap set system proxy.https="http://proxy.corp.com:8080"

# Verify
snap get system proxy
# Key          Value
# proxy.http   http://proxy.corp.com:8080
# proxy.https  http://proxy.corp.com:8080
```

**Cloud-init Configuration**:
```yaml
stages:
  boot.before:
    - commands:
      - snap set system proxy.http="http://proxy.corp.com:8080"
      - snap set system proxy.https="http://proxy.corp.com:8080"
```

---

## Common Pitfalls

### ❌ WRONG: Missing service subnet in NO_PROXY

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ❌ Missing pod and service CIDRs
```

**Result**: Pods cannot communicate, services unreachable

### ✅ CORRECT: Provider auto-appends subnets

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ✅ Provider appends pod/service subnets automatically
```

---

### ❌ WRONG: Proxy blocks snap package registry

```bash
snap install k8s --classic
# error: unable to contact snap store
# ❌ Proxy blocks snapcraft.io
```

### ✅ CORRECT: Add snapcraft.io to NO_PROXY or snap proxy config

```bash
# Option 1: NO_PROXY
export NO_PROXY="snapcraft.io,api.snapcraft.io"
snap install k8s --classic

# Option 2: Snap proxy config
snap set system proxy.http="http://proxy.corp.com:8080"
snap install k8s --classic
```

---

## Integration Points

### With Canonical K8s

- Proxy settings apply to kubelet and snap containerd
- Calico CNI pods inherit kubelet proxy settings
- snap install requires proxy config for package downloads

### With Stylus

- Appliance mode: `/etc/default/kubelet`
- Agent mode: `${STYLUS_ROOT}/etc/default/kubelet`

### With Container Runtime

- Snap containerd uses proxy for image pulls
- Proxy config in `/etc/systemd/system/snap.k8s.containerd.service.d/`

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-canonical/stages/proxy.go` - Proxy configuration logic
- `/Users/rishi/work/src/provider-canonical/utils/proxy.go` - NO_PROXY calculation

## Related Skills

- See `provider-canonical:01-architecture` for proxy overview
- See `provider-canonical:02-configuration-patterns` for pod/service subnet config
- See `provider-canonical:05-networking` for Calico CNI and proxy interaction

**Related Provider Skills**:
- See `provider-kubeadm:04-proxy-configuration` for kubeadm proxy (same logic)

## Documentation References

**Snap Proxy**:
- https://snapcraft.io/docs/system-options#heading--proxy

**Kubernetes Proxy**:
- https://kubernetes.io/docs/tasks/administer-cluster/configure-kubernetes-proxy/
