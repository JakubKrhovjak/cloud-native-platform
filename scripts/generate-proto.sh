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

# Services
SERVICES=("project-service" "student-service")

# Generate for each service
for SERVICE in "${SERVICES[@]}"; do
    echo -e "${GREEN}Generating for ${SERVICE}...${NC}"

    OUT_DIR="${ROOT_DIR}/${SERVICE}/pkg/gen"
    mkdir -p "${OUT_DIR}"

    # Generate Go code
    protoc \
        --proto_path="${PROTO_DIR}" \
        --go_out="${OUT_DIR}" \
        --go_opt=paths=source_relative \
        --go-grpc_out="${OUT_DIR}" \
        --go-grpc_opt=paths=source_relative \
        "${PROTO_DIR}/project/v1/project.proto"

    echo -e "${GREEN}✓ Generated for ${SERVICE}${NC}"
done

echo -e "${BLUE}Done!${NC}"
