# Canonical Kubernetes Provider Architecture

## Overview

The provider-canonical is a Kairos/C3OS cluster provider that configures Canonical Kubernetes (MicroK8s/k8s-snap) installations for Ubuntu-based edge deployments. Canonical Kubernetes is snap-based and optimized for Ubuntu/Canonical ecosystem with opinionated defaults.

## Key Concepts

### Kubernetes Distribution: Canonical Kubernetes

Canonical Kubernetes is Ubuntu's snap-packaged Kubernetes distribution:

- **Snap-Based**: Uses snap packaging for installation and updates
- **Ubuntu-Optimized**: Designed for Ubuntu/Canonical Linux distributions
- **Opinionated Defaults**: Includes Calico CNI by default
- **Automatic Updates**: Snap auto-updates (can be disabled)
- **Confined**: Snap confinement provides security isolation
- **Simple Management**: Single snap command for lifecycle management

### Canonical K8s vs kubeadm vs K3s

| Feature | K3s | kubeadm | Canonical K8s |
|---------|-----|---------|---------------|
| **Package Format** | Binary | Binary | Snap |
| **Distribution** | Rancher | Upstream | Canonical/Ubuntu |
| **CNI Included** | Flannel | **None** | Calico |
| **Ecosystem** | General | General | Ubuntu-specific |
| **Updates** | Manual | Manual | Snap auto-update |
| **Complexity** | Low | High | Medium |

### Snap Packaging

**Snap Characteristics**:
- **Isolated**: Snap runs in confined environment
- **Read-only**: Snap files immutable at `/snap/k8s/`
- **Writable Data**: Config and data in `/var/snap/k8s/`
- **Auto-updates**: Snap refreshes automatically (configurable)

**Snap Paths**:
```
/snap/k8s/current/            # Read-only snap files (binaries, libs)
/var/snap/k8s/current/        # Writable data (config, etcd, certs)
/var/snap/k8s/common/         # Persistent data across snap revisions
```

### Kairos/C3OS Integration

Provider-canonical integrates with Kairos immutable Linux distribution:

- **Cloud-init Configuration**: Declarative cluster setup via cluster section
- **Immutable OS**: A/B partition updates with atomic upgrades
- **Boot Stages**: Yip-based stage execution during boot.before phase
- **Service Management**: systemd service orchestration (snap services)
- **Ubuntu Base**: Typically used with Ubuntu-based Kairos images

### Component Architecture

```
┌─────────────────────────────────────────────────────┐
│              Cloud-Init (User Configuration)         │
│  cluster:                                           │
│    cluster_token: token123                         │
│    control_plane_host: 10.0.1.100                  │
│    role: init|controlplane|worker                  │
│    config: |                                       │
│      # Canonical K8s specific config              │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│     Provider-Canonical (Cluster Plugin)             │
│  • Parse cluster configuration                      │
│  • Install k8s snap                                 │
│  • Configure proxy settings                         │
│  • Generate join commands                           │
│  • Handle role-specific setup                       │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│               Yip Stage Execution                   │
│  boot.before:                                       │
│    1. Install k8s snap                              │
│    2. Configure snap settings                       │
│    3. Install config files                          │
│    4. Start k8s services                            │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│          Canonical Kubernetes (Snap)                │
│  ┌──────────────┐  ┌────────────────┐              │
│  │ K8s Control  │  │  K8s Worker    │              │
│  │ Plane        │  │                │              │
│  │              │  │                │              │
│  │ • API Server │  │ • Kubelet      │              │
│  │ • Scheduler  │  │ • Kube-proxy   │              │
│  │ • Controller │  │ • Container    │              │
│  │ • etcd       │  │   Runtime      │              │
│  │ • Calico CNI │  │ • Calico CNI   │              │
│  └──────────────┘  └────────────────┘              │
│                                                      │
│  ✅ Calico CNI included by default                  │
└─────────────────────────────────────────────────────┘
```

### Configuration Flow

1. **Cluster Definition**: User defines cluster configuration in cloud-init
2. **Provider Execution**: Provider-canonical processes configuration via ClusterProvider()
3. **Snap Installation**: Install k8s snap package
4. **Config Generation**: Creates config files in `/var/snap/k8s/`
5. **Service Start**: systemd starts snap.k8s.daemon service
6. **Cluster Join**: Node joins cluster using join token

### File Structure

```
/snap/k8s/current/              # Read-only snap installation
├── bin/
│   ├── k8s                     # Main k8s command
│   ├── kubectl
│   └── ...

/var/snap/k8s/current/          # Writable config and data
├── args/                       # Service arguments
│   ├── kube-apiserver
│   ├── kube-controller-manager
│   ├── kube-scheduler
│   └── kubelet
├── certs/                      # Cluster certificates
│   ├── ca.crt
│   └── ...
├── credentials/                # Cluster credentials
│   └── client.config           # kubeconfig
└── var/
    ├── kubernetes/             # Kubernetes data
    └── lib/
        └── etcd/               # etcd database

/etc/default/
└── kubelet                     # Kubelet proxy environment variables

/etc/systemd/system/
└── snap.k8s.containerd.service.d/
    └── http-proxy.conf         # Containerd proxy config
```

## Implementation Patterns

### Role-Based Configuration

The provider handles three distinct roles:

**Init Role (Bootstrap First Node)**:
```go
case clusterplugin.RoleInit:
    // Install snap
    snapInstall("k8s")

    // Initialize cluster
    k8sBootstrap()
```

**ControlPlane Role (Additional Control Plane)**:
```go
case clusterplugin.RoleControlPlane:
    // Install snap
    snapInstall("k8s")

    // Join as control plane
    k8sJoin(controlPlane=true)
```

**Worker Role (Worker Node)**:
```go
case clusterplugin.RoleWorker:
    // Install snap
    snapInstall("k8s")

    // Join as worker
    k8sJoin(controlPlane=false)
```

### Snap Commands

**Install Snap**:
```bash
snap install k8s --classic
```

**Bootstrap Cluster** (init role):
```bash
k8s bootstrap
```

**Get Join Token**:
```bash
k8s get-join-token
# eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Join Cluster**:
```bash
# Worker
k8s join-cluster <token>

# Control plane
k8s join-cluster <token> --control-plane
```

### Proxy Configuration

Provider-canonical configures proxy settings in two locations:

1. **kubelet environment variables**: `/etc/default/kubelet`
2. **containerd systemd override**: `/etc/systemd/system/snap.k8s.containerd.service.d/http-proxy.conf`

(Similar to provider-kubeadm - see `03-proxy-configuration.md`)

### Deployment Modes

Provider-canonical supports both Stylus deployment modes:

**Appliance Mode** (STYLUS_ROOT=/):
- Config files in `/etc/default/kubelet`
- Standard snap containerd

**Agent Mode** (STYLUS_ROOT=/persistent/spectro):
- Config files in `/persistent/spectro/etc/default/kubelet`
- Spectro containerd (if used)

(See `04-deployment-modes.md` for details)

## Common Pitfalls

### ❌ WRONG: Expecting kubeadm commands

```bash
# Trying kubeadm commands
kubeadm init
# kubeadm: command not found
# ❌ Canonical K8s uses `k8s` command, not kubeadm
```

### ✅ CORRECT: Use k8s snap commands

```bash
k8s bootstrap
k8s get-join-token
k8s join-cluster <token>
# ✅ Correct Canonical K8s commands
```

---

### ❌ WRONG: Modifying snap files

```bash
# Try to edit snap file
vi /snap/k8s/current/bin/k8s
# Read-only file system
# ❌ Snap files are immutable
```

### ✅ CORRECT: Modify config in /var/snap

```bash
# Edit writable config
vi /var/snap/k8s/current/args/kubelet
# ✅ Config files are writable
```

---

### ❌ WRONG: Forgetting snap auto-updates

```yaml
# Deploy cluster
k8s bootstrap

# 24 hours later
k8s version
# v1.29.0  ← Snap auto-updated!
# ❌ Cluster upgraded without explicit action
```

### ✅ CORRECT: Disable snap auto-updates if needed

```bash
# Hold snap at current version
snap refresh --hold k8s

# Or set refresh timer
snap set system refresh.timer=fri,23:00-01:00
```

---

## Integration Points

### Dependencies

- **gomi**: Common Go libraries (logging, k8s utilities)
- **hapi**: API schema definitions (for Palette integration)
- **Kairos/C3OS**: Immutable Linux distribution framework
- **snap**: Snap package manager (required on hosts)
- **Canonical K8s snap**: k8s snap package from Canonical

### Consumers

- **Stylus**: Edge orchestration agent (appliance and agent modes)
  - Appliance mode: Local cluster management
  - Agent mode: Palette-managed cluster
- **Palette**: Cluster orchestration platform
  - Provides cluster configuration via Stylus
  - Receives cluster status and events

### With CNI

- **Calico CNI included by default** (unlike kubeadm)
- Calico provides NetworkPolicy support out-of-the-box
- No manual CNI installation required

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-canonical/pkg/provider/provider.go` - Main provider implementation
- `/Users/rishi/work/src/provider-canonical/pkg/domain/types.go` - Configuration data structures

**Shell Scripts**:
- `/Users/rishi/work/src/provider-canonical/scripts/` - Cluster lifecycle scripts

## Related Skills

- See `provider-canonical:02-configuration-patterns` for cluster configuration
- See `provider-canonical:03-proxy-configuration` for proxy setup
- See `provider-canonical:04-deployment-modes` for appliance vs agent mode
- See `provider-canonical:05-networking` for Calico CNI details
- See `provider-canonical:06-troubleshooting` for common issues

**Related Provider Skills**:
- See `provider-kubeadm:01-architecture` for comparison with upstream kubeadm
- See `provider-k3s:01-architecture` for comparison with K3s

## Documentation References

**Canonical Kubernetes**:
- https://canonical-kubernetes.readthedocs-hosted.com/
- https://snapcraft.io/k8s

**Snap Documentation**:
- https://snapcraft.io/docs

**Kairos Documentation**:
- https://kairos.io/docs/

**Provider Repository**:
- https://github.com/kairos-io/provider-canonical
