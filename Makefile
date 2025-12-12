.PHONY: test test-student test-project test-integration test-all test-coverage test-verbose clean admin-dev admin-build admin-install k8s/setup k8s/deploy k8s/deploy-dev k8s/deploy-prod k8s/status k8s/wait k8s/logs k8s/cleanup setup deploy deploy-dev deploy-prod status wait logs cleanup

# Default: Run all tests (shared container, fast)
test:
	@echo "🧪 Running all tests (shared container)..."
	@go test ./services/student-service/... ./services/project-service/...

# Test individual services
test-student:
	@echo "🧪 Testing student-service..."
	@go test ./services/student-service/...

test-project:
	@echo "🧪 Testing project-service..."
	@go test ./services/project-service/...

# Integration tests (isolated containers, slower)
test-integration:
	@echo "🧪 Running integration tests (isolated containers)..."
	@go test -tags=integration ./services/student-service/... ./services/project-service/...

# All tests (shared + integration)
test-all:
	@echo "🧪 Running ALL tests (shared + integration)..."
	@go test ./services/student-service/... ./services/project-service/...
	@go test -tags=integration ./services/student-service/... ./services/project-service/...

# Tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	@go test -cover ./services/student-service/... ./services/project-service/...


# Test with race detector
test-race:
	@echo "🏁 Running tests with race detector..."
	@go test -race ./services/student-service/... ./services/project-service/...

# Clean test cache
clean:
	@echo "🧹 Cleaning test cache..."
	@go clean -testcache

# Watch tests (requires entr: brew install entr)
test-watch:
	@echo "👀 Watching for changes..."
	@find . -name '*.go' | entr -c make test

# Admin panel commands
admin-install:
	@echo "📦 Installing admin panel dependencies..."
	@cd services/admin && npm install

admin-dev:
	@echo "🚀 Starting admin panel dev server..."
	@cd services/admin && npm run dev

admin-build:
	@echo "🏗️  Building admin panel..."
	@cd services/admin && npm run build

generate-proto:
	@echo "🔨 Generating protobuf files..."
	./scripts/generate-proto.sh

# Kubernetes commands (proxy to k8s/Makefile)
k8s/setup:
	@$(MAKE) -C k8s setup

k8s/deploy:
	@$(MAKE) -C k8s deploy

k8s/deploy-dev:
	@$(MAKE) -C k8s deploy-dev

k8s/deploy-prod:
	@$(MAKE) -C k8s deploy-prod

k8s/status:
	@$(MAKE) -C k8s status

k8s/wait:
	@$(MAKE) -C k8s wait

k8s/logs:
	@$(MAKE) -C k8s logs

k8s/cleanup:
	@$(MAKE) -C k8s cleanup

# Kubernetes aliases (without k8s/ prefix)
setup: k8s/setup
deploy: k8s/deploy
deploy-dev: k8s/deploy-dev
deploy-prod: k8s/deploy-prod
status: k8s/status
wait: k8s/wait
logs: k8s/logs
cleanup: k8s/cleanup

# Help
help:
	@echo "Available commands:"
	@echo "  make test              - Run all tests (default, fast)"
	@echo "  make test-student      - Test student-service only"
	@echo "  make test-project      - Test project-service only"
	@echo "  make test-integration  - Run integration tests (slow)"
	@echo "  make test-all          - Run all tests (shared + integration)"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make test-verbose      - Run tests with verbose output"
	@echo "  make test-race         - Run tests with race detector"
	@echo "  make clean             - Clean test cache"
	@echo "  make test-watch        - Watch and auto-run tests on change"
	@echo ""
	@echo "Admin Panel:"
	@echo "  make admin-install     - Install admin panel dependencies"
	@echo "  make admin-dev         - Start admin panel dev server"
	@echo "  make admin-build       - Build admin panel for production"
	@echo ""
	@echo "Kubernetes:"
	@echo "  make setup             - Create Kind cluster"
	@echo "  make deploy-dev        - Deploy to development"
	@echo "  make deploy-prod       - Deploy to production"
	@echo "  make status            - Show cluster status"
	@echo "  make logs              - Follow service logs"
	@echo "  make cleanup           - Delete cluster"
	@echo ""
	@echo "  make help              - Show this help message"
