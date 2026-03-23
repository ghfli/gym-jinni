#!/bin/bash
set -e

# k3d uses the Docker API; with Podman this must point at the rootful API socket.
export DOCKER_HOST="${DOCKER_HOST:-unix:///run/podman/podman.sock}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Gym-Jinni K8s Test Environment Teardown${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Parse command line arguments
FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --force|-f)
            FORCE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --force, -f     Skip confirmation prompt"
            echo "  --help          Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Confirmation prompt
if [ "$FORCE" = false ]; then
    echo -e "${RED}WARNING: This will destroy the entire K8s test environment!${NC}"
    echo ""
    echo "This will:"
    echo "  - Delete all Kubernetes resources in gym-jinni namespace"
    echo "  - Delete the k3d cluster"
    echo "  - Remove kubeconfig file"
    echo "  - Optionally remove local container images (podman)"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo "Teardown cancelled."
        exit 0
    fi
fi

echo -e "${YELLOW}Starting teardown...${NC}"
echo ""

# Step 1: Delete Kubernetes resources
echo -e "${YELLOW}Step 1: Deleting Kubernetes resources...${NC}"
export KUBECONFIG=$HOME/.kube/gym-jinni-config

if [ -f "$KUBECONFIG" ]; then
    kubectl delete namespace gym-jinni --ignore-not-found=true --timeout=60s || true
    echo -e "${GREEN}✓ Kubernetes resources deleted${NC}"
else
    echo -e "${YELLOW}⚠ Kubeconfig not found, skipping K8s resource deletion${NC}"
fi
echo ""

# Step 2: Delete k3d cluster
echo -e "${YELLOW}Step 2: Deleting k3d cluster...${NC}"

if command -v k3d >/dev/null 2>&1; then
    if sudo -E k3d cluster list 2>/dev/null | grep -q "gym-jinni"; then
        sudo -E k3d cluster delete gym-jinni 2>/dev/null || true
        echo -e "${GREEN}✓ k3d cluster deleted${NC}"
    else
        echo -e "${YELLOW}⚠ k3d cluster not found${NC}"
    fi
else
    echo -e "${YELLOW}⚠ k3d not installed, skipping cluster deletion${NC}"
fi
echo ""

# Step 3: Clean up kubeconfig
echo -e "${YELLOW}Step 3: Cleaning up kubeconfig...${NC}"

if [ -f "$HOME/.kube/gym-jinni-config" ]; then
    rm -f "$HOME/.kube/gym-jinni-config"
    echo -e "${GREEN}✓ Kubeconfig removed${NC}"
else
    echo -e "${YELLOW}⚠ Kubeconfig not found${NC}"
fi
echo ""

# Step 4: Clean up local container images (optional)
echo -e "${YELLOW}Step 4: Cleaning up container images...${NC}"
if [ "$FORCE" = false ]; then
    read -p "Do you want to remove built container images (podman)? (yes/no): " -r
    echo ""
    if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        sudo podman rmi gym-jinni/service:latest 2>/dev/null || true
        sudo podman rmi gym-jinni/ui:latest 2>/dev/null || true
        echo -e "${GREEN}✓ Container images removed${NC}"
    else
        echo -e "${YELLOW}⚠ Skipping container image removal${NC}"
    fi
else
    sudo podman rmi gym-jinni/service:latest 2>/dev/null || true
    sudo podman rmi gym-jinni/ui:latest 2>/dev/null || true
    echo -e "${GREEN}✓ Container images removed${NC}"
fi
echo ""

# Final cleanup
echo -e "${YELLOW}Performing final cleanup...${NC}"
sudo podman system prune -f 2>/dev/null || true

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Teardown Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "The K8s test environment has been completely removed."
echo ""
echo "To set it up again, run:"
echo "  cd k8s-setup && ./setup.sh"
echo ""

