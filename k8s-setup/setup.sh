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

command -v podman >/dev/null 2>&1 || { echo -e "${RED}Error: podman is not installed${NC}" >&2; exit 1; }
command -v ansible-playbook >/dev/null 2>&1 || { echo -e "${RED}Error: ansible is not installed${NC}" >&2; exit 1; }

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
            echo "  --skip-build    Skip building Docker images"
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
echo -e "${YELLOW}Step 1: Setting up Podman containers and MicroK8s cluster...${NC}"
ansible-playbook playbooks/setup-cluster.yml

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
    echo -e "${YELLOW}Step 3: Building Docker images and deploying services...${NC}"
    ansible-playbook playbooks/deploy-services.yml
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to deploy services${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Services deployed${NC}"
    echo ""
elif [ "$SKIP_BUILD" = false ]; then
    echo -e "${YELLOW}Step 3: Building Docker images...${NC}"
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
echo "  kubectl logs -f deployment/csr-service -n gym-jinni"
echo "  kubectl logs -f deployment/main-service -n gym-jinni"
echo "  kubectl logs -f deployment/ui -n gym-jinni"
echo ""
echo "  # Port forward to access services"
echo "  kubectl port-forward -n gym-jinni svc/csr-service 8082:8082 8083:8083"
echo "  kubectl port-forward -n gym-jinni svc/main-service 8080:8080 8081:8081"
echo "  kubectl port-forward -n gym-jinni svc/ui 8000:80"
echo ""
echo "  # Access Podman containers directly"
echo "  podman exec -it gym-jinni-node1 bash"
echo ""
echo "  # View MicroK8s status"
echo "  podman exec gym-jinni-node1 microk8s status"
echo ""
echo -e "${YELLOW}Service Endpoints (after port-forward):${NC}"
echo ""
echo "  CSR Service gRPC:  localhost:8082"
echo "  CSR Service HTTP:  http://localhost:8083/v1/roles"
echo "  Main Service gRPC: localhost:8080"
echo "  Main Service HTTP: http://localhost:8081/v1/users"
echo "  UI:                http://localhost:8000"
echo ""
echo -e "${GREEN}Happy testing!${NC}"

