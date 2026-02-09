# Canonical Kubernetes Deployment Modes

## Overview

Provider-canonical supports both **appliance mode** and **agent mode** via Stylus integration using the `STYLUS_ROOT` environment variable, similar to other providers.

## Key Concepts

### Deployment Modes

**Appliance Mode** (Local Cluster Management):
- `STYLUS_ROOT=/`
- Configuration files in standard locations
- Uses snap k8s containerd

**Agent Mode** (Palette-Managed Cluster):
- `STYLUS_ROOT=/persistent/spectro`
- Configuration files in persistent storage
- May use spectro-containerd variant

### STYLUS_ROOT Environment Variable

| Mode | STYLUS_ROOT | kubelet config | containerd |
|------|-------------|----------------|------------|
| Appliance | `/` | `/etc/default/kubelet` | snap.k8s.containerd |
| Agent | `/persistent/spectro` | `/persistent/spectro/etc/default/kubelet` | snap.k8s.containerd or spectro variant |

## Implementation Patterns

### Pattern 1: Appliance Mode Deployment

**Use Case**: Standalone edge clusters managed locally by Stylus.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  # No STYLUS_ROOT - defaults to appliance mode
```

**Generated File Paths**:
```bash
# kubelet config
/etc/default/kubelet

# Snap containerd proxy
/etc/systemd/system/snap.k8s.containerd.service.d/http-proxy.conf

# Snap data
/var/snap/k8s/current/
```

---

### Pattern 2: Agent Mode Deployment

**Use Case**: Palette-managed edge clusters.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  env:
    STYLUS_ROOT: "/persistent/spectro"   # ← Agent mode
```

**Generated File Paths**:
```bash
# kubelet config
${STYLUS_ROOT}/etc/default/kubelet = /persistent/spectro/etc/default/kubelet

# Snap containerd proxy (if using standard snap)
/etc/systemd/system/snap.k8s.containerd.service.d/http-proxy.conf

# OR spectro-containerd (if using Palette variant)
/etc/systemd/system/snap.spectro-k8s.containerd.service.d/http-proxy.conf
```

---

### Pattern 3: Mode Detection in Scripts

**Use Case**: Shell scripts need to detect deployment mode.

**Script Detection**:
```bash
#!/bin/bash

# Detect STYLUS_ROOT
root_path="${STYLUS_ROOT:-/}"

# Appliance mode
if [ "$root_path" = "/" ]; then
  echo "Running in appliance mode"
  kubelet_config="/etc/default/kubelet"

# Agent mode
elif [ "$root_path" = "/persistent/spectro" ]; then
  echo "Running in agent mode"
  kubelet_config="${root_path}/etc/default/kubelet"
fi
```

---

## Mode Comparison

| Aspect | Appliance Mode | Agent Mode |
|--------|----------------|------------|
| **STYLUS_ROOT** | `/` | `/persistent/spectro` |
| **kubelet config** | `/etc/default/kubelet` | `/persistent/spectro/etc/default/kubelet` |
| **Snap location** | `/snap/k8s/` | `/snap/k8s/` (same) |
| **Palette Integration** | No | Yes (via Stylus) |
| **Provider Events** | No | Yes |
| **Management** | Local (Stylus) | Remote (Palette) |

## Common Pitfalls

### ❌ WRONG: Hardcoded paths without STYLUS_ROOT

```bash
# Script
kubelet_config="/etc/default/kubelet"
# ❌ Ignores STYLUS_ROOT - breaks agent mode
```

### ✅ CORRECT: Use STYLUS_ROOT prefix

```bash
root_path="${STYLUS_ROOT:-/}"
kubelet_config="${root_path}/etc/default/kubelet"
# ✅ Works in both modes
```

---

## Integration Points

### With Stylus

**Appliance Mode**:
- Stylus calls provider-canonical with `STYLUS_ROOT=/`
- Stylus manages cluster lifecycle locally
- No Palette communication

**Agent Mode**:
- Stylus calls provider-canonical with `STYLUS_ROOT=/persistent/spectro`
- Stylus forwards provider events to Palette
- Palette provides cluster configuration

### With Snap

- Snap installation location same for both modes (`/snap/k8s/`)
- Snap data location same for both modes (`/var/snap/k8s/`)
- Only kubelet config path differs based on STYLUS_ROOT

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-canonical/pkg/provider/provider.go` - Mode detection logic
- `/Users/rishi/work/src/provider-canonical/stages/proxy.go` - STYLUS_ROOT handling

## Related Skills

- See `provider-canonical:01-architecture` for deployment mode overview
- See `provider-canonical:03-proxy-configuration` for mode-specific proxy setup

**Related Provider Skills**:
- See `provider-kubeadm:06-deployment-modes` for kubeadm deployment modes (same concept)

## Documentation References

**Stylus Documentation**:
- See Stylus ai-knowledge-base for appliance vs agent mode architecture
