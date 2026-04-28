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

# Create local registry container if it doesn't exist
echo -e "${BLUE}📦 Setting up local Docker registry...${NC}"
if [ ! "$(docker ps -q -f name=kind-registry)" ]; then
    if [ "$(docker ps -aq -f name=kind-registry)" ]; then
        docker rm kind-registry
    fi
    docker run -d --restart=always -p "127.0.0.1:5001:5000" --name kind-registry registry:2
    echo -e "${GREEN}✅ Local registry created at localhost:5001${NC}"
else
    echo -e "${GREEN}✅ Local registry already running${NC}"
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

# Connect the registry to the cluster network
echo -e "${BLUE}🔗 Connecting registry to cluster network...${NC}"
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' kind-registry)" = 'null' ]; then
    docker network connect "kind" "kind-registry"
    echo -e "${GREEN}✅ Registry connected to kind network${NC}"
else
    echo -e "${GREEN}✅ Registry already connected${NC}"
fi

# Add a mirror entry for localhost:5001 -> kind-registry:5000 on every node.
# Newer kindest/node images run containerd >= 2.x which requires the
# hosts.toml mechanism; the legacy inline `mirrors` block is rejected.
echo -e "${BLUE}🔧 Wiring local registry into containerd on every node...${NC}"
REGISTRY_DIR="/etc/containerd/certs.d/localhost:5001"
for node in $(kind get nodes --name grud-cluster); do
    docker exec "$node" mkdir -p "$REGISTRY_DIR"
    cat <<EOF | docker exec -i "$node" tee "$REGISTRY_DIR/hosts.toml" >/dev/null
[host."http://kind-registry:5000"]
EOF
done
echo -e "${GREEN}✅ Registry mirror configured on all nodes${NC}"

# Document the local registry
echo -e "${BLUE}📝 Configuring local registry...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:5001"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

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
