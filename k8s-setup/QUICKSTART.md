# Quick Start Guide - K8s Test Environment

## TL;DR - Get Running in 5 Minutes

```bash
# 1. Navigate to k8s-setup directory
cd k8s-setup

# 2. Run the setup script
chmod +x setup.sh
./setup.sh

# 3. Wait for completion (5-10 minutes)
# The script will automatically:
#   - Create 3 Podman containers
#   - Install MicroK8s cluster
#   - Build Docker images
#   - Deploy all services

# 4. Access services
export KUBECONFIG=$HOME/.kube/gym-jinni-config
kubectl get pods -n gym-jinni

# 5. Port forward to test
kubectl port-forward -n gym-jinni svc/csr-service 8083:8083 &
kubectl port-forward -n gym-jinni svc/main-service 8081:8081 &

# 6. Test endpoints
curl http://localhost:8083/v1/roles
curl http://localhost:8081/v1/users
```

## Prerequisites

```bash
# Install Podman and Ansible
sudo apt-get install -y podman ansible  # Ubuntu/Debian
sudo pacman -S podman ansible           # Arch Linux
brew install podman ansible             # macOS
```

## What Gets Deployed?

- **3-node MicroK8s cluster** in Podman containers
- **PostgreSQL** database with persistent storage
- **CSR Service** (RBAC) - 2 replicas
- **Main Service** (User/Class) - 2 replicas
- **UI** (Flutter web) - 2 replicas
- **Ingress controller** for routing

## Service Ports

| Service | gRPC | HTTP | Description |
|---------|------|------|-------------|
| CSR Service | 8082 | 8083 | Role-based access control |
| Main Service | 8080 | 8081 | User and class management |
| UI | - | 80 | Flutter web interface |
| PostgreSQL | 5432 | - | Database |

## Common Commands

```bash
# View all pods
kubectl get pods -n gym-jinni

# View logs
kubectl logs -f deployment/csr-service -n gym-jinni

# Scale a service
kubectl scale deployment csr-service --replicas=3 -n gym-jinni

# Restart a service
kubectl rollout restart deployment/csr-service -n gym-jinni

# Access a container
podman exec -it gym-jinni-node1 bash

# Teardown everything
./teardown.sh
```

## Troubleshooting

### Setup fails at MicroK8s installation
```bash
# Manually check snapd
podman exec gym-jinni-node1 systemctl status snapd

# Restart and retry
./teardown.sh
./setup.sh
```

### Services not starting
```bash
# Check pod status
kubectl describe pod <pod-name> -n gym-jinni

# Check logs
kubectl logs <pod-name> -n gym-jinni

# Check if images are imported
podman exec gym-jinni-node1 microk8s ctr images ls | grep gym-jinni
```

### Can't connect to services
```bash
# Verify services are running
kubectl get svc -n gym-jinni

# Check endpoints
kubectl get endpoints -n gym-jinni

# Port forward manually
kubectl port-forward -n gym-jinni svc/csr-service 8083:8083
```

## Next Steps

1. Read the full [README.md](README.md) for detailed documentation
2. Explore the [manifests/](manifests/) directory to understand K8s resources
3. Check [playbooks/](playbooks/) for Ansible automation
4. Review CI/CD pipelines in `.github/workflows/` and `.gitlab-ci.yml`

## Clean Up

```bash
# Remove everything
./teardown.sh

# Keep images but remove cluster
./teardown.sh
# Answer "no" when asked about removing images
```

## Support

For issues or questions:
- Check the main [README.md](README.md)
- Review Ansible logs in the terminal output
- Check Kubernetes events: `kubectl get events -n gym-jinni`
- Inspect pod logs: `kubectl logs <pod-name> -n gym-jinni`

