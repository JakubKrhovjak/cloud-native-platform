#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Setting up Kind cluster for GRUD project${NC}"
echo ""

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo -e "${RED}❌ kind is not installed${NC}"
    echo "Install with: brew install kind (macOS) or go install sigs.k8s.io/kind@latest"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl is not installed${NC}"
    echo "Install with: brew install kubectl (macOS)"
    exit 1
fi

# Check if ko is installed
if ! command -v ko &> /dev/null; then
    echo -e "${YELLOW}⚠️  ko is not installed${NC}"
    echo "Installing ko..."
    go install github.com/google/ko@latest
fi

# Delete existing cluster if it exists
if kind get clusters | grep -q "grud-cluster"; then
    echo -e "${YELLOW}⚠️  Deleting existing grud-cluster...${NC}"
    kind delete cluster --name grud-cluster
fi

# Create Kind cluster
echo -e "${BLUE}📦 Creating Kind cluster with 3 worker nodes...${NC}"
# Determine the correct path to kind-config.yaml
if [ -f "k8s/kind-config.yaml" ]; then
    CONFIG_PATH="k8s/kind-config.yaml"
elif [ -f "kind-config.yaml" ]; then
    CONFIG_PATH="kind-config.yaml"
else
    echo -e "${RED}❌ Cannot find kind-config.yaml${NC}"
    exit 1
fi
kind create cluster --config "$CONFIG_PATH"

# Wait for cluster to be ready
echo -e "${BLUE}⏳ Waiting for cluster to be ready...${NC}"
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# Display nodes
echo -e "${GREEN}✅ Cluster created successfully!${NC}"
echo ""
echo -e "${BLUE}📋 Cluster nodes:${NC}"
kubectl get nodes -o wide

echo ""
echo -e "${BLUE}🏷️  Node labels and taints:${NC}"
kubectl get nodes -o custom-columns=NAME:.metadata.name,LABELS:.metadata.labels,TAINTS:.spec.taints

echo ""
echo -e "${GREEN}✅ Kind cluster setup complete!${NC}"
echo -e "${BLUE}Next steps:${NC}"
echo "  2. Deploy databases: ./scripts/deploy-databases.sh"
echo "  3. Build and deploy services: ./scripts/deploy-services.sh"
