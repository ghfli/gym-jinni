#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Gym-Jinni K8s Test Environment Setup${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

command -v ansible-playbook >/dev/null 2>&1 || { echo -e "${RED}Error: ansible is not installed${NC}" >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo -e "${RED}Error: kubectl is not installed${NC}" >&2; exit 1; }
command -v podman >/dev/null 2>&1 || { echo -e "${RED}Error: podman is not installed${NC}" >&2; exit 1; }

if [ ! -S /run/podman/podman.sock ]; then
    echo -e "${RED}Error: Podman API socket not found at /run/podman/podman.sock${NC}" >&2
    echo "Enable it with: sudo systemctl enable --now podman.socket" >&2
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"
echo ""

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Parse command line arguments
SKIP_BUILD=false
SKIP_DEPLOY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-deploy)
            SKIP_DEPLOY=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-build    Skip building container images"
            echo "  --skip-deploy   Skip deploying services"
            echo "  --help          Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Step 1: Setup cluster
echo -e "${YELLOW}Step 1: Setting up k3d Kubernetes cluster...${NC}"
ansible-playbook playbooks/setup-cluster-k3d.yml

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to setup cluster${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Cluster setup complete${NC}"
echo ""

# Step 2: Export KUBECONFIG
echo -e "${YELLOW}Step 2: Configuring kubectl access...${NC}"
export KUBECONFIG=$HOME/.kube/gym-jinni-config
echo "export KUBECONFIG=$HOME/.kube/gym-jinni-config" >> $HOME/.bashrc

echo -e "${GREEN}✓ kubectl configured${NC}"
echo ""

# Wait for cluster to be fully ready
echo -e "${YELLOW}Waiting for cluster to be fully ready...${NC}"
sleep 10

# Step 3: Build and deploy services
if [ "$SKIP_BUILD" = false ] && [ "$SKIP_DEPLOY" = false ]; then
    echo -e "${YELLOW}Step 3: Building container images and deploying services...${NC}"
    ansible-playbook playbooks/deploy-services.yml
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to deploy services${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Services deployed${NC}"
    echo ""
elif [ "$SKIP_BUILD" = false ]; then
    echo -e "${YELLOW}Step 3: Building container images...${NC}"
    ansible-playbook playbooks/deploy-services.yml --tags build
    echo -e "${GREEN}✓ Images built${NC}"
    echo ""
elif [ "$SKIP_DEPLOY" = false ]; then
    echo -e "${YELLOW}Step 3: Deploying services...${NC}"
    ansible-playbook playbooks/deploy-services.yml --tags deploy
    echo -e "${GREEN}✓ Services deployed${NC}"
    echo ""
fi

# Display cluster info
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Setup Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}Cluster Information:${NC}"
echo ""
echo "Kubectl config: $HOME/.kube/gym-jinni-config"
echo "To use kubectl: export KUBECONFIG=$HOME/.kube/gym-jinni-config"
echo ""
echo -e "${YELLOW}Useful Commands:${NC}"
echo ""
echo "  # View all pods"
echo "  kubectl get pods -n gym-jinni"
echo ""
echo "  # View all services"
echo "  kubectl get svc -n gym-jinni"
echo ""
echo "  # View logs"
echo "  kubectl logs -f deployment/main-service -n gym-jinni"
echo "  kubectl logs -f deployment/ui -n gym-jinni"
echo ""
echo "  # Port forward to access services"
echo "  kubectl port-forward -n gym-jinni svc/main-service 8080:8080 8081:8081"
echo "  kubectl port-forward -n gym-jinni svc/ui 3000:80"
echo ""
echo "  # View k3d cluster info"
echo "  k3d cluster list"
echo "  k3d node list"
echo ""
echo "  # Access k3d nodes directly (if needed)"
echo "  podman exec -it k3d-gym-jinni-server-0 sh"
echo ""
echo -e "${YELLOW}Service Endpoints (after port-forward):${NC}"
echo ""
echo "  CSR Service gRPC:  localhost:8082"
echo "  CSR Service HTTP:  http://localhost:8083/v1/roles"
echo "  Main Service gRPC: localhost:8080"
echo "  Main Service HTTP: http://localhost:8081/v1/users"
echo "  UI:                http://localhost:3000"
echo ""
echo -e "${GREEN}Happy testing!${NC}"

