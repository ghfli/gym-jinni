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
#   - Build Docker images
#   - Deploy all services

# 4. Access services
export KUBECONFIG=$HOME/.kube/gym-jinni-config
kubectl get pods -n gym-jinni

# 5. Port forward to test (user + RBAC share main-service HTTP)
kubectl port-forward -n gym-jinni svc/main-service 8080:8080 8081:8081 &

# 6. Test endpoints
curl http://localhost:8081/v1/roles
curl http://localhost:8081/v1/users
```

## Prerequisites

```bash
# Install Docker and Ansible
sudo apt-get install -y docker.io ansible kubectl  # Ubuntu/Debian
sudo pacman -S docker ansible kubectl              # Arch Linux
brew install docker ansible kubectl                # macOS

# Add user to docker group (Linux only)
sudo usermod -aG docker $USER
# Log out and back in for group changes to take effect

# k3d will be automatically installed by the setup script
```

## What Gets Deployed?

- **k3d cluster** with 1 server + 2 agent nodes
- **PostgreSQL** database with persistent storage (5Gi)
- **Main Service** (User, Class, RBAC) - 2 replicas
- **UI** (Flutter web) - 2 replicas
- **Traefik ingress** controller (included with k3d)

## Service Ports

| Service | gRPC | HTTP | Description |
|---------|------|------|-------------|
| Main Service | 8080 | 8081 | Users, classes, RBAC (same process) |
| UI | - | 3000 | Flutter web interface |
| PostgreSQL | 5432 | - | Database |
| K8s API | 6443 | - | Kubernetes API server |

## Common Commands

```bash
# View all pods
kubectl get pods -n gym-jinni

# View cluster nodes
kubectl get nodes

# View k3d cluster info
k3d cluster list
k3d node list

# View logs
kubectl logs -f deployment/main-service -n gym-jinni

# Scale a service
kubectl scale deployment main-service --replicas=3 -n gym-jinni

# Restart a service
kubectl rollout restart deployment/main-service -n gym-jinni

# Access a k3d node (if needed)
docker exec -it k3d-gym-jinni-server-0 sh

# Stop cluster (preserves state)
k3d cluster stop gym-jinni

# Start cluster
k3d cluster start gym-jinni

# Teardown everything
./teardown.sh
```

## Troubleshooting

### Setup fails - Docker not running
```bash
# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Verify Docker is running
docker ps
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
docker images | grep gym-jinni

# Import images to k3d if needed
k3d image import gym-jinni/service:latest -c gym-jinni
```

### Can't connect to services
```bash
# Verify services are running
kubectl get svc -n gym-jinni

# Check endpoints
kubectl get endpoints -n gym-jinni

# Port forward manually
kubectl port-forward -n gym-jinni svc/main-service 8081:8081
```

### Port already in use
```bash
# Check what's using the port
sudo ss -tulpn | grep :8080

# Kill the process or use different ports
# Edit roles/k3d/defaults/main.yml to change port mappings
```

### kubectl can't connect
```bash
# Refresh kubeconfig
k3d kubeconfig get gym-jinni > ~/.kube/gym-jinni-config
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
- View k3d logs: `k3d cluster list` and `docker logs k3d-gym-jinni-server-0`
