#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Running all tests in monorepo...${NC}"
echo ""

# Array of services to test
services=("student-service" "project-service")

# Track failures
failed=0

# Test each service
for service in "${services[@]}"; do
    echo -e "${BLUE}📦 Testing $service...${NC}"
    if go test ./services/$service/...; then
        echo -e "${GREEN}✅ $service tests passed${NC}"
    else
        echo -e "${RED}❌ $service tests failed${NC}"
        failed=1
    fi
    echo ""
done

# Summary
if [ $failed -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
