# Quick Start Guide - K8s Test Environment

## TL;DR - Get Running in 2 Minutes

```bash
# 1. Navigate to k8s-setup directory
cd k8s-setup

# 2. Run the setup script
chmod +x setup.sh
./setup.sh

# 3. Wait for completion (~2 minutes)
# The script will automatically:
#   - Install k3d (if not present)
#   - Create k3d cluster (1 server + 2 agents)
#   - Build container images (Podman)
#   - Deploy all services

# 4. Access services
export KUBECONFIG=$HOME/.kube/gym-jinni-config
kubectl get pods -n gym-jinni

# 5. Hit the API via k3d LB (no port-forward needed), or port-forward if you prefer localhost:808x
curl http://localhost:39081/v1/roles

# Optional: kubectl port-forward -n gym-jinni svc/main-service 8080:8080 8081:8081 &
# curl http://localhost:8081/v1/roles
```

## Prerequisites

```bash
# Install Podman, Ansible, and kubectl
sudo apt-get install -y podman ansible kubectl  # Ubuntu/Debian
sudo pacman -S podman ansible kubectl           # Arch Linux
brew install podman ansible kubectl             # macOS

# Linux: expose the rootful Podman API socket (required for k3d + Ansible)
sudo systemctl enable --now podman.socket

# k3d will be automatically installed by the setup script
```

## What Gets Deployed?

- **k3d cluster** with 1 server + 2 agent nodes
- **PostgreSQL** database with persistent storage (5Gi)
- **Main Service** (User, Class, RBAC) - 2 replicas
- **UI** (Flutter web) - 2 replicas
- **Traefik ingress** controller (included with k3d)

## Service Ports

| Service | gRPC (in-cluster) | HTTP (in-cluster) | Host via k3d LB |
|---------|-------------------|-------------------|-----------------|
| Main Service | 8080 | 8081 | 39080 / 39081 |
| CSR (RBAC) | 8080 | 8081 | same as main (`main-service` in-cluster) |
| UI | - | Service :80, NodePort 30000 | http://localhost:39300 |
| PostgreSQL | - | 5432 | not published by default |
| K8s API | - | HTTPS | localhost:6443 |

`main-service` and `ui` use **NodePort** `30080` / `30081` / `30000` so k3d’s LB port publishes (`39080` / `39081` / `39300`) reach real backends. **390xx / 39300** are the **host** ports so they do not conflict with `docker-proxy` on `808x`/`3000` or with local `kubectl port-forward`.

## Common Commands

```bash
# View all pods
kubectl get pods -n gym-jinni

# View cluster nodes
kubectl get nodes

# View k3d cluster info (use same DOCKER_HOST as setup)
export DOCKER_HOST=unix:///run/podman/podman.sock
sudo -E k3d cluster list
sudo -E k3d node list

# View logs
kubectl logs -f deployment/main-service -n gym-jinni

# Scale a service
kubectl scale deployment main-service --replicas=3 -n gym-jinni

# Restart a service
kubectl rollout restart deployment/main-service -n gym-jinni

# Access a k3d node (if needed)
podman exec -it k3d-gym-jinni-server-0 sh

# Stop cluster (preserves state)
sudo -E k3d cluster stop gym-jinni

# Start cluster
sudo -E k3d cluster start gym-jinni

# Teardown everything
./teardown.sh
```

## Troubleshooting

### Setup fails - Podman socket missing
```bash
# Enable the API socket (Linux)
sudo systemctl enable --now podman.socket
ls -l /run/podman/podman.sock
```

### Setup fails - k3d installation
```bash
# Manually install k3d
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Verify installation
k3d version
```

### Services not starting
```bash
# Check pod status
kubectl describe pod <pod-name> -n gym-jinni

# Check logs
kubectl logs <pod-name> -n gym-jinni

# Check if images are available
sudo podman images | grep gym-jinni

# Import images to k3d if needed
export DOCKER_HOST=unix:///run/podman/podman.sock
sudo -E k3d image import gym-jinni/service:latest -c gym-jinni
```

### Can't connect to services
```bash
# Verify services are running
kubectl get svc -n gym-jinni

# Check endpoints
kubectl get endpoints -n gym-jinni

# Port forward manually (optional; LB URLs use 39081 etc.)
kubectl port-forward -n gym-jinni svc/main-service 8081:8081
```

### Port already in use
```bash
# Check what's using the port (k3d LB publishes 39080, 39081, 39300)
sudo ss -tulpn | grep -E ':3908|:39300|:6443'

# Kill the process or use different ports
# Edit roles/k3d/defaults/main.yml to change port mappings
```

### kubectl can't connect
```bash
# Refresh kubeconfig
sudo -E k3d kubeconfig get gym-jinni > ~/.kube/gym-jinni-config
export KUBECONFIG=$HOME/.kube/gym-jinni-config

# Verify connection
kubectl cluster-info
```

## Next Steps

1. Read the full [README.md](README.md) for detailed documentation
2. Explore the [manifests/](manifests/) directory to understand K8s resources
3. Check [playbooks/](playbooks/) for Ansible automation
4. Review [K3D_IMPLEMENTATION.md](K3D_IMPLEMENTATION.md) for implementation details
5. See [TEST_RESULTS.md](TEST_RESULTS.md) for test results

## Clean Up

```bash
# Remove everything
./teardown.sh

# Keep images but remove cluster
./teardown.sh
# Answer "no" when asked about removing images
```

## Performance

- **Setup time**: ~2 minutes (vs 10+ minutes with old approach)
- **Cluster creation**: ~20 seconds
- **Service deployment**: ~60 seconds
- **Resource usage**: ~1.5GB RAM (vs ~2.5GB with old approach)

## Why k3d?

- ✅ **Fast**: 5x faster than Podman + MicroK8s approach
- ✅ **Reliable**: No cgroup or AppArmor issues
- ✅ **Simple**: Single command setup and teardown
- ✅ **Standard**: Full Kubernetes API compatibility
- ✅ **Lightweight**: Uses K3s (lightweight Kubernetes)
- ✅ **Multi-node**: Simulates 3-node cluster

## Support

For issues or questions:
- Check the main [README.md](README.md)
- Review [SETUP_ISSUES.md](SETUP_ISSUES.md) for known issues and solutions
- Check Kubernetes events: `kubectl get events -n gym-jinni`
- Inspect pod logs: `kubectl logs <pod-name> -n gym-jinni`
- View k3d logs: `sudo -E k3d cluster list` and `sudo podman logs k3d-gym-jinni-server-0`
