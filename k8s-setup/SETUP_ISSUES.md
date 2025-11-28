# K8s Setup Issues and Resolution Attempts

## Summary

The K8s test environment setup is currently failing due to fundamental limitations of running Kubernetes distributions (MicroK8s and K3s) inside Podman containers.

## Issues Encountered

### 1. MicroK8s with Snap (Original Approach)

**Problem:** MicroK8s requires snap, and snap requires AppArmor to be fully functional.

**Error:**
```
error: cannot perform the following tasks:
- Run install hook of "microk8s" snap if present (run hook "install": aa_is_enabled() failed unexpectedly (No such file or directory): No such file or directory)
```

**Root Cause:**
- Snapd's install hooks call `aa_is_enabled()` from libapparmor
- AppArmor's `/sys/kernel/security/apparmor` filesystem is mounted read-only from the host
- Snapd cannot write AppArmor profiles, causing the installation to fail

**Attempted Solutions:**
1. Disabled AppArmor service in containers
2. Created fake `aa-enabled` and `aa_is_enabled` scripts
3. Tried mounting `/sys/kernel/security` as read-write (not possible with Podman)
4. Attempted to use `--security-opt apparmor=unconfined`
5. Tried to extract and modify the snap package
6. Attempted to create LD_PRELOAD library to intercept `aa_is_enabled()`

**Result:** All attempts failed because the `aa_is_enabled()` function is called by snapd in a confined environment where workarounds don't work.

### 2. K3s (Alternative Approach)

**Problem 1:** K3s containerd cannot use overlayfs inside a container.

**Error:**
```
"overlayfs" snapshotter cannot be enabled: failed to mount overlay: invalid argument
```

**Solution:** Use `--snapshotter=fuse-overlayfs` and install `fuse-overlayfs` package.

**Problem 2:** K3s kubelet cannot access `/dev/kmsg`.

**Error:**
```
failed to create kubelet: open /dev/kmsg: operation not permitted
```

**Solution:** Use `--kubelet-arg='feature-gates=KubeletInUserNamespace=true'`

**Problem 3:** K3s cannot create cgroups.

**Error:**
```
Failed to create cgroup: mkdir /sys/fs/cgroup/kubepods: permission denied
```

**Root Cause:**
- Podman containers don't have permission to create cgroups in `/sys/fs/cgroup`
- Even with `--privileged` and `--cgroupns=host`, the container cannot modify the host's cgroup hierarchy
- This is a fundamental security boundary in Podman

**Result:** K3s fails to start because it cannot manage cgroups, which is essential for Kubernetes operation.

## Recommended Solutions

### Option 1: Use Real VMs Instead of Containers

Replace Podman containers with actual VMs using:
- **libvirt/KVM** with `virt-install`
- **Vagrant** with libvirt provider
- **Multipass** (Ubuntu VMs)

**Pros:**
- Full system isolation
- Proper cgroup support
- AppArmor works correctly
- No nested containerization issues

**Cons:**
- Requires more resources
- Slower to start/stop
- More complex setup

### Option 2: Use Kind (Kubernetes in Docker)

Kind is specifically designed to run Kubernetes in containers:

```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Create cluster
kind create cluster --name gym-jinni
```

**Pros:**
- Designed for this use case
- Handles all the container/cgroup complexities
- Fast and lightweight
- Works with both Docker and Podman

**Cons:**
- Different from production Kubernetes
- Limited to single-node or simulated multi-node

### Option 3: Use k3d (K3s in Docker)

k3d wraps K3s in Docker/Podman containers properly:

```bash
# Install k3d
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Create cluster
k3d cluster create gym-jinni --servers 1 --agents 2
```

**Pros:**
- Uses K3s (closer to production)
- Handles container complexities
- Multi-node support
- Works with Podman

**Cons:**
- Additional abstraction layer
- Requires k3d tool

### Option 4: Use Minikube with Podman Driver

Minikube has a Podman driver that handles these issues:

```bash
# Install minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Start with podman driver
minikube start --driver=podman --container-runtime=containerd
```

**Pros:**
- Well-maintained and popular
- Good documentation
- Handles Podman complexities

**Cons:**
- Single-node only (for Podman driver)
- More resource-intensive

## Current Status

- **teardown.sh**: ✅ Working - Successfully cleans up containers, volumes, and networks
- **setup.sh**: ❌ Failing - Cannot complete K8s cluster setup due to cgroup/AppArmor issues
- **Containers**: ✅ Created successfully with systemd support
- **Networking**: ✅ Podman network configured correctly
- **K8s Installation**: ❌ Blocked by fundamental container limitations

## Next Steps

1. **Immediate**: Choose one of the recommended solutions above
2. **Update**: Modify Ansible playbooks to use the chosen solution
3. **Test**: Verify the new setup works end-to-end
4. **Document**: Update README with the new approach

## Files Modified

- `k8s-setup/roles/podman-nodes/tasks/main.yml` - Added AppArmor workarounds
- `k8s-setup/roles/microk8s/tasks/main.yml` - Added devmode fallback
- `k8s-setup/roles/k3s/` - Created new K3s role (incomplete)
- `k8s-setup/playbooks/setup-cluster-k3s.yml` - Created K3s playbook
- `k8s-setup/inventory.yml` - Fixed IP addresses
- `k8s-setup/setup.sh` - Updated to use K3s playbook

## Recommendation

**Use Kind or k3d** as they are specifically designed for running Kubernetes in containers and handle all the edge cases we encountered. They abstract away the complexity while still providing a functional multi-node-like environment.

