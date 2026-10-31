#!/bin/bash
set -e

# Use the same socket selection as setup.sh.
if [ -z "${DOCKER_HOST:-}" ] && [ ! -S /var/run/docker.sock ] && [ -S /run/podman/podman.sock ]; then
    export DOCKER_HOST=unix:///run/podman/podman.sock
fi
export DOCKER_HOST="${DOCKER_HOST:-unix:///var/run/docker.sock}"

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
    echo "  - Optionally remove local container images (docker)"
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
    if kubectl delete namespace gym-jinni --ignore-not-found=true --timeout=60s; then
        echo -e "${GREEN}✓ Kubernetes resources deleted${NC}"
    else
        echo -e "${YELLOW}⚠ Namespace deletion failed; continuing with cluster deletion${NC}" >&2
    fi
else
    echo -e "${YELLOW}⚠ Kubeconfig not found, skipping K8s resource deletion${NC}"
fi
echo ""

# Step 2: Delete k3d cluster
echo -e "${YELLOW}Step 2: Deleting k3d cluster...${NC}"

if command -v k3d >/dev/null 2>&1; then
    if ! CLUSTERS=$(sudo -E k3d cluster list --no-headers); then
        echo -e "${RED}Error: Cannot list clusters using DOCKER_HOST=$DOCKER_HOST${NC}" >&2
        exit 1
    fi
    if echo "$CLUSTERS" | grep -q '^gym-jinni[[:space:]]'; then
        sudo -E k3d cluster delete gym-jinni
        echo -e "${GREEN}✓ k3d cluster deleted${NC}"
    else
        echo -e "${YELLOW}⚠ k3d cluster not found${NC}"
    fi
else
    echo -e "${RED}Error: k3d is not installed; cannot delete the cluster${NC}" >&2
    exit 1
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
remove_images() {
    local images image
    images=$(sudo docker image ls --format '{{.Repository}}:{{.Tag}}')
    for image in docker.io/gym-jinni/service:latest docker.io/gym-jinni/ui:latest; do
        if echo "$images" | grep -qxF "$image" || echo "$images" | grep -qxF "${image#docker.io/}"; then
            sudo docker rmi "$image"
        fi
    done
    echo -e "${GREEN}✓ Container image cleanup complete${NC}"
}

echo -e "${YELLOW}Step 4: Cleaning up container images...${NC}"
if [ "$FORCE" = false ]; then
    read -p "Do you want to remove built container images (docker)? (yes/no): " -r
    echo ""
    if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        remove_images
    else
        echo -e "${YELLOW}⚠ Skipping container image removal${NC}"
    fi
else
    remove_images
fi
echo ""

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

