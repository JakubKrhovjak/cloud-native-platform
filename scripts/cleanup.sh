#!/bin/bash

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}⚠️  This will delete the entire Kind cluster and all data${NC}"
echo -e "${YELLOW}⚠️  Are you sure? (yes/no)${NC}"
read -r response

if [ "$response" != "yes" ]; then
    echo -e "${BLUE}Cancelled${NC}"
    exit 0
fi

echo -e "${BLUE}🗑️  Deleting Kind cluster...${NC}"
kind delete cluster --name grud-cluster

echo -e "${GREEN}✅ Cluster deleted${NC}"
