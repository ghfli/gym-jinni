# Kubernetes Test Environment for gym-jinni

This directory contains the complete setup for a Kubernetes cluster using k3d, orchestrated by Ansible, with CI/CD pipelines for the gym-jinni project.

## Architecture

- **k3d cluster** (K3s in Linux containers via Podman) with 1 server + 2 agent nodes
  - Lightweight Kubernetes distribution; k3d drives the engine through the Docker-compatible API (`DOCKER_HOST`)
  - Handles all cgroup and AppArmor complexities automatically
  - Fast setup and teardown
- **Services deployed**:
  - PostgreSQL (StatefulSet with persistent storage)
  - CSR Service (RBAC) - same `main-service` endpoints as user/class API (gRPC `8080`, HTTP `8081` in-cluster; see `configmaps.yml`)
  - Main Service (User/Class) - gRPC: 8080, HTTP: 8081 (in-cluster)
  - UI (Flutter web) - Service port 80, **NodePort** `30000` (required so k3d LB `39300:30000` reaches the pods)
- **Host access via k3d load balancer** (`roles/k3d/defaults/main.yml`): `39080`, `39081`, `39300` on localhost (avoids binding `808x`/`3000` on the host so `kubectl port-forward` and local dev tools can use those ports)
- **Ansible** playbooks for automation
- **CI/CD** pipelines (GitHub Actions & GitLab CI)

## Prerequisites

Install the following on your host machine:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y ansible python3-pip podman kubectl
sudo systemctl enable --now podman.socket

# Arch Linux
sudo pacman -S ansible podman kubectl
sudo systemctl enable --now podman.socket

# macOS (Homebrew; enable the Podman machine / socket per Podman docs)
brew install ansible podman kubectl
```

**Podman socket:** Ansible and k3d expect the **rootful** API socket at `unix:///run/podman/podman.sock` (the default after `systemctl enable --now podman.socket` on Linux). Override `k3d_docker_host` / `k3d_podman_socket_path` in `group_vars/all.yml` if you use a different layout.

**Note:** k3d is invoked with `sudo -E` so `DOCKER_HOST` from Ansible reaches the same engine `podman build` uses under `sudo`.

## Quick Start

### 1. Setup the entire environment

```bash
cd k8s-setup
chmod +x setup.sh
./setup.sh
```

This will:
1. Install k3d (if not already installed)
2. Create a k3d cluster with 1 server + 2 agents
3. Configure kubectl access
4. Build container images for all services (Podman)
5. Deploy all services to Kubernetes

### 2. Access the cluster

```bash
# Set KUBECONFIG (already done by setup.sh)
export KUBECONFIG=$HOME/.kube/gym-jinni-config

# View pods
kubectl get pods -n gym-jinni

# View services
kubectl get svc -n gym-jinni

# View ingress
kubectl get ingress -n gym-jinni
```

### 3. Access services

After `./setup.sh`, the k3d server load balancer publishes **high host ports** so they do not overlap `docker-proxy` bindings that used to block `3000` and `8080`–`8083`.

**Default (via k3d LB on localhost):**

- Main Service gRPC: `localhost:39080`
- Main Service HTTP: http://localhost:39081/v1/roles
- CSR (RBAC): same URLs as main service (`CSR_SERVICE_*` in `configmaps.yml` points at `main-service:8080` / `:8081`)
- UI: http://localhost:39300

**Optional: `kubectl port-forward`** (uses the same numeric ports on your machine as in the cluster):

```bash
# Main + CSR APIs (same Deployment / Service)
kubectl port-forward -n gym-jinni svc/main-service 8080:8080 8081:8081

# UI
kubectl port-forward -n gym-jinni svc/ui 3000:80

# PostgreSQL (for debugging)
kubectl port-forward -n gym-jinni svc/postgres 5432:5432
```

Then access (port-forward path):

- Main / CSR HTTP: http://localhost:8081/v1/roles
- UI: http://localhost:3000

### 4. Teardown

```bash
cd k8s-setup
chmod +x teardown.sh
./teardown.sh
```

## Manual Setup (Step by Step)

If you want more control, run Ansible playbooks manually:

### 1. Setup cluster

```bash
cd k8s-setup
ansible-playbook playbooks/setup-cluster-k3d.yml
```

### 2. Build and deploy services

```bash
# Build images only
ansible-playbook playbooks/deploy-services.yml --tags build

# Deploy services only
ansible-playbook playbooks/deploy-services.yml --tags deploy

# Both
ansible-playbook playbooks/deploy-services.yml
```

### 3. Configure kubectl

```bash
export KUBECONFIG=$HOME/.kube/gym-jinni-config
kubectl get nodes
```

## Directory Structure

```
k8s-setup/
├── ansible.cfg                 # Ansible configuration
├── inventory.yml               # Simplified inventory for localhost
├── playbooks/
│   ├── setup-cluster-k3d.yml   # Setup k3d cluster
│   └── deploy-services.yml     # Build + deploy services
├── roles/
│   ├── k3d/                    # Install k3d and create cluster
│   ├── build-images/           # Build container images (Podman)
│   └── gym-jinni/              # Deploy to K8s
├── manifests/
│   ├── namespace.yml
│   ├── secrets.yml
│   ├── configmaps.yml
│   ├── postgres-statefulset.yml
│   ├── main-service-deployment.yml
│   ├── ui-deployment.yml
│   └── ingress.yml
├── container/
│   ├── Dockerfile.service
│   └── Dockerfile.ui
├── group_vars/
│   └── all.yml                 # k3d DOCKER_HOST / Podman socket paths
├── archive/                    # Old Podman/MicroK8s/K3s implementation
├── setup.sh                    # One-command setup
├── teardown.sh                 # One-command cleanup
└── README.md                   # This file
```

## Useful Commands

### Cluster Management

```bash
# View cluster nodes
kubectl get nodes

# View all resources in gym-jinni namespace
kubectl get all -n gym-jinni

# Describe a pod
kubectl describe pod <pod-name> -n gym-jinni

# View logs
kubectl logs -f deployment/main-service -n gym-jinni
kubectl logs -f deployment/main-service -n gym-jinni
kubectl logs -f deployment/ui -n gym-jinni

# Execute command in pod
kubectl exec -it <pod-name> -n gym-jinni -- /bin/sh

# Scale deployments
kubectl scale deployment main-service --replicas=3 -n gym-jinni
```

### k3d Cluster Management

```bash
# List k3d clusters
k3d cluster list

# List k3d nodes
k3d node list

# Access k3d node shell (if needed)
podman exec -it k3d-gym-jinni-server-0 sh

# Stop cluster (preserves state)
k3d cluster stop gym-jinni

# Start cluster
k3d cluster start gym-jinni

# View k3d cluster info
kubectl cluster-info
```

### Debugging

```bash
# Check if PostgreSQL is ready
kubectl get statefulset postgres -n gym-jinni
kubectl exec -it postgres-0 -n gym-jinni -- psql -U root -d gj -c '\dt'

# Check service endpoints
kubectl get endpoints -n gym-jinni

# View events
kubectl get events -n gym-jinni --sort-by='.lastTimestamp'

# Check pod resource usage
kubectl top pods -n gym-jinni
```

## CI/CD Pipelines

### GitHub Actions

The `.github/workflows/deploy.yml` pipeline:
1. Builds container images with Podman for all services
2. Pushes to GitHub Container Registry
3. Deploys to Kubernetes cluster (on push to main/master)

**Setup:**
1. Add `KUBECONFIG` secret to GitHub repository (base64 encoded)
2. Push to main/master branch to trigger deployment

### GitLab CI

The `.gitlab-ci.yml` pipeline:
1. Builds container images with Podman
2. Runs integration tests
3. Deploys to Kubernetes (manual trigger)

**Setup:**
1. Add `KUBECONFIG_CONTENT` variable to GitLab CI/CD settings (base64 encoded)
2. Configure GitLab Container Registry
3. Manually trigger deployment from GitLab UI

## Troubleshooting

### k3d cluster won't start

```bash
# Check Podman API socket and engine
ls -l /run/podman/podman.sock
sudo podman ps

# Check k3d version
k3d version

# Delete and recreate cluster
k3d cluster delete gym-jinni
./setup.sh
```

### Services not deploying

```bash
# Check if images are available
sudo podman images | grep gym-jinni

# Import images to k3d cluster (same DOCKER_HOST as Ansible)
export DOCKER_HOST=unix:///run/podman/podman.sock
sudo -E k3d image import gym-jinni/service:latest -c gym-jinni
sudo -E k3d image import gym-jinni/ui:latest -c gym-jinni

# Check pod events
kubectl describe pod <pod-name> -n gym-jinni

# Check logs
kubectl logs <pod-name> -n gym-jinni
```

### Network/Port issues

```bash
# Check k3d cluster ports
k3d cluster list

# Check if ports are in use
netstat -tulpn | grep -E '39080|39081|39300|6443'

# Recreate cluster with different ports if needed
k3d cluster delete gym-jinni
# Edit roles/k3d/defaults/main.yml to change ports
./setup.sh
```

### kubectl can't connect

```bash
# Check kubeconfig
export KUBECONFIG=$HOME/.kube/gym-jinni-config
kubectl cluster-info

# Get fresh kubeconfig
k3d kubeconfig get gym-jinni > $HOME/.kube/gym-jinni-config
```

### Database connection issues

```bash
# Check PostgreSQL logs
kubectl logs postgres-0 -n gym-jinni

# Test connection from service pod
kubectl exec -it <service-pod> -n gym-jinni -- nc -zv postgres 5432

# Check ConfigMap
kubectl get configmap gym-jinni-config -n gym-jinni -o yaml
```

## Performance Tuning

### Adjust resource limits

Edit deployment manifests to increase/decrease resources:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### Scale replicas

```bash
kubectl scale deployment main-service --replicas=5 -n gym-jinni
```

### Persistent storage

PostgreSQL uses persistent volumes. To increase storage:

```yaml
# In postgres-statefulset.yml
volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      resources:
        requests:
          storage: 20Gi  # Increase from 5Gi
```

## Development Workflow

1. Make code changes in `service/`
2. Rebuild images: `ansible-playbook playbooks/deploy-services.yml --tags build`
3. Redeploy: `kubectl rollout restart deployment/main-service -n gym-jinni`
4. View logs: `kubectl logs -f deployment/main-service -n gym-jinni`

## Security Notes

- This setup is for **testing only**, not production
- Database credentials are stored in plain text
- No TLS/SSL configured
- Services run with default security contexts

For production:
- Use proper secrets management (Vault, Sealed Secrets)
- Enable TLS for all services
- Configure network policies
- Use non-root containers
- Enable RBAC policies

## License

Same as gym-jinni project.

