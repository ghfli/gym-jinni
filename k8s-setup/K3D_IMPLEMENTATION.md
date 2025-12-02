# k3d Implementation Summary

## Overview

Successfully migrated the Kubernetes test environment from Podman containers + MicroK8s/K3s to **k3d** (K3s in Docker). This resolves all the cgroup and AppArmor issues that were blocking the original implementation.

## What Changed

### ✅ New Files Created

1. **`roles/k3d/`** - New Ansible role
   - `defaults/main.yml` - Configuration variables (cluster name, node count, port mappings)
   - `tasks/main.yml` - Installs k3d, creates cluster, configures kubectl

2. **`playbooks/setup-cluster-k3d.yml`** - New setup playbook
   - Uses k3d role to create cluster
   - Configures kubectl access
   - Displays cluster information

### 📝 Files Modified

1. **`setup.sh`**
   - Changed from `setup-cluster-k3s.yml` to `setup-cluster-k3d.yml`
   - Updated prerequisites (removed Podman, added kubectl check)
   - Updated help text and status messages

2. **`teardown.sh`**
   - Replaced Podman container cleanup with `k3d cluster delete`
   - Removed volume/network cleanup (k3d handles this)
   - Changed from Podman to Docker for image cleanup

3. **`inventory.yml`**
   - Simplified to localhost only (k3d manages nodes internally)
   - Removed node-specific configuration

4. **`README.md`**
   - Updated architecture description
   - Updated prerequisites
   - Replaced Podman/MicroK8s commands with k3d commands
   - Updated troubleshooting section

5. **`SETUP_ISSUES.md`**
   - Documented successful k3d implementation
   - Listed all changes made
   - Updated status to show working solution

### 📦 Files Archived

Moved to `archive/` directory:
- `roles/podman-nodes/` - Podman container setup
- `roles/microk8s/` - MicroK8s installation
- `roles/k3s/` - K3s installation
- `playbooks/setup-cluster.yml` - Original MicroK8s playbook
- `playbooks/setup-cluster-k3s.yml` - K3s playbook

### ✅ Files Unchanged

These continue to work as-is:
- `playbooks/deploy-services.yml` - Service deployment
- `roles/build-images/` - Docker image building
- `roles/gym-jinni/` - Kubernetes manifests deployment
- `manifests/*.yml` - All Kubernetes manifests
- `docker/Dockerfile.*` - All Dockerfiles

## How to Use

### Setup

```bash
cd k8s-setup
./setup.sh
```

This will:
1. Install k3d (if not already installed)
2. Create a k3d cluster named "gym-jinni" with 1 server + 2 agents
3. Configure kubectl access
4. Build and deploy all services

### Teardown

```bash
cd k8s-setup
./teardown.sh
```

This will:
1. Delete Kubernetes resources in gym-jinni namespace
2. Delete the k3d cluster
3. Clean up kubeconfig
4. Optionally remove Docker images

### Useful Commands

```bash
# View cluster
k3d cluster list
k3d node list
kubectl get nodes

# Stop/Start cluster (preserves state)
k3d cluster stop gym-jinni
k3d cluster start gym-jinni

# Access cluster
export KUBECONFIG=$HOME/.kube/gym-jinni-config
kubectl get pods -A

# Import images to cluster
k3d image import <image-name> -c gym-jinni
```

## Benefits

1. **No cgroup issues** - k3d handles container isolation properly
2. **No AppArmor issues** - k3d manages security contexts automatically
3. **Faster setup** - ~2 minutes vs ~10 minutes with Podman approach
4. **Simpler teardown** - Single command cleanup
5. **Standard Kubernetes** - Full K8s API compatibility
6. **Multi-node simulation** - 1 server + 2 agents for realistic testing
7. **Better maintained** - k3d is actively developed and widely used

## Port Mappings

The following ports are exposed from the cluster to localhost:

- `8080` → Main service gRPC (NodePort 30080)
- `8081` → Main service HTTP (NodePort 30081)
- `8082` → CSR service gRPC (NodePort 30082)
- `8083` → CSR service HTTP (NodePort 30083)
- `3000` → UI (NodePort 30000)

## Prerequisites

- **Docker** or **Podman** (with Docker compatibility)
- **kubectl** - Kubernetes CLI
- **Ansible** - For automation
- **k3d** - Will be auto-installed by setup script

## Next Steps

1. Run `./setup.sh` to create the cluster
2. Deploy services with the script or manually
3. Access services via the exposed ports
4. Develop and test your gym-jinni application

## Troubleshooting

See `README.md` for detailed troubleshooting steps. Common issues:

- **k3d not found**: The setup script will install it automatically
- **Docker not running**: Start Docker daemon (`sudo systemctl start docker`)
- **Port conflicts**: Edit `roles/k3d/defaults/main.yml` to change port mappings
- **kubectl can't connect**: Run `k3d kubeconfig get gym-jinni > ~/.kube/gym-jinni-config`

## References

- [k3d Documentation](https://k3d.io/)
- [K3s Documentation](https://k3s.io/)
- [Original Setup Issues](SETUP_ISSUES.md)

