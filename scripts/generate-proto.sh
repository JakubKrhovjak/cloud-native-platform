#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Generating protobuf files...${NC}"

# Root directory
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/api/proto"
OUT_DIR="${ROOT_DIR}/api/gen"

# Create output directory
mkdir -p "${OUT_DIR}"

echo -e "${GREEN}Generating to shared api/gen directory...${NC}"

# Generate Go code
protoc \
    --proto_path="${PROTO_DIR}" \
    --go_out="${OUT_DIR}" \
    --go_opt=paths=source_relative \
    --go-grpc_out="${OUT_DIR}" \
    --go-grpc_opt=paths=source_relative \
    "${PROTO_DIR}/project/v1/project.proto"

echo -e "${GREEN}✓ Generated protobuf files${NC}"
echo -e "${BLUE}Done!${NC}"
