.PHONY: test test-student test-project test-integration test-all test-coverage test-verbose clean admin-dev admin-build admin-install k8s/setup k8s/deploy k8s/deploy-dev k8s/deploy-prod k8s/status k8s/wait k8s/logs k8s/stop k8s/start k8s/cleanup k8s/port-forward-admin k8s/port-forward-student k8s/port-forward-project setup deploy deploy-dev deploy-prod status wait logs stop start cleanup port-forward-admin port-forward-student port-forward-project build build-student build-project run-student run-project version

# Build configuration
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags
STUDENT_LDFLAGS := -X 'student-service/internal/app.Version=$(VERSION)' \
                   -X 'student-service/internal/app.GitCommit=$(GIT_COMMIT)' \
                   -X 'student-service/internal/app.BuildTime=$(BUILD_TIME)'

PROJECT_LDFLAGS := -X 'project-service/internal/app.Version=$(VERSION)' \
                   -X 'project-service/internal/app.GitCommit=$(GIT_COMMIT)' \
                   -X 'project-service/internal/app.BuildTime=$(BUILD_TIME)'

# Build targets
build: build-student build-project
	@echo "✅ All services built successfully"

build-student:
	@echo "🔨 Building student-service $(VERSION) ($(GIT_COMMIT))..."
	@mkdir -p bin
	@cd services/student-service && \
	go build -ldflags="$(STUDENT_LDFLAGS)" \
	  -o ../../bin/student-service \
	  ./cmd/server
	@echo "✅ student-service → bin/student-service"

build-project:
	@echo "🔨 Building project-service $(VERSION) ($(GIT_COMMIT))..."
	@mkdir -p bin
	@cd services/project-service && \
	go build -ldflags="$(PROJECT_LDFLAGS)" \
	  -o ../../bin/project-service \
	  ./cmd/server
	@echo "✅ project-service → bin/project-service"

run-student:
	@cd services/student-service && go run ./cmd/server

run-project:
	@cd services/project-service && go run ./cmd/server

version:
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

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

k8s/port-forward-project:
	@$(MAKE) -C k8s port-forward-project

# Kubernetes aliases (without k8s/ prefix)
setup:
	@$(MAKE) -C k8s setup

deploy:
	@$(MAKE) -C k8s deploy

deploy-dev:
	@$(MAKE) -C k8s deploy-dev

deploy-prod:
	@$(MAKE) -C k8s deploy-prod

status:
	@$(MAKE) -C k8s status

wait:
	@$(MAKE) -C k8s wait

logs:
	@$(MAKE) -C k8s logs

stop:
	@$(MAKE) -C k8s stop

start:
	@$(MAKE) -C k8s start

cleanup:
	@$(MAKE) -C k8s cleanup

port-forward-admin:
	@$(MAKE) -C k8s port-forward-admin

port-forward-student:
	@$(MAKE) -C k8s port-forward-student

port-forward-project:
	@$(MAKE) -C k8s port-forward-project

# Observability stack (Helm-based)
infra/setup:
	@echo "📦 Adding Helm repositories..."
	@helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	@helm repo add grafana https://grafana.github.io/helm-charts
	@helm repo update
	@echo "✅ Helm repositories added"

infra/deploy-prometheus:
	@echo "🔥 Deploying Prometheus stack..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		-n infra \
		-f k8s/infra/prometheus-values.yaml \
		--wait
	@echo "📊 Deploying Grafana dashboards..."
	@kubectl apply -f k8s/infra/grafana-dashboard-configmap.yaml
	@echo "✅ Prometheus stack deployed"
	@echo "📊 Grafana: http://localhost:30300 (admin/admin)"

infra/deploy-alloy:
	@echo "📡 Deploying Grafana Alloy..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install alloy grafana/alloy \
		-n infra \
		-f k8s/infra/alloy-values.yaml \
		--wait
	@echo "✅ Grafana Alloy deployed"

infra/deploy-nats:
	@echo "💬 Deploying NATS..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f k8s/infra/nats.yaml
	@echo "✅ NATS deployed"

infra/deploy-loki:
	@echo "📝 Deploying Loki (logging)..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install loki grafana/loki \
		-n infra \
		-f k8s/infra/loki-values.yaml \
		--wait
	@echo "✅ Loki deployed (logs collected by Alloy)"

infra/deploy-tempo:
	@echo "🔍 Deploying Tempo (tracing)..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install tempo grafana/tempo \
		-n infra \
		-f k8s/infra/tempo-values.yaml \
		--wait
	@echo "✅ Tempo deployed"

infra/deploy-alerts:
	@echo "🚨 Deploying alerting rules..."
	@kubectl apply -f k8s/infra/alerting-rules.yaml
	@echo "✅ Alerting rules deployed"

infra/deploy: infra/setup infra/deploy-prometheus infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts
	@echo "✅ Full observability stack deployed"

infra/status:
	@echo "📊 Observability stack status:"
	@kubectl get pods -n infra

infra/cleanup:
	@echo "🧹 Cleaning up observability stack..."
	@helm uninstall loki -n infra 2>/dev/null || true
	@helm uninstall tempo -n infra 2>/dev/null || true
	@helm uninstall prometheus -n infra 2>/dev/null || true
	@helm uninstall alloy -n infra 2>/dev/null || true
	@kubectl delete -f k8s/infra/nats.yaml 2>/dev/null || true
	@kubectl delete -f k8s/infra/alerting-rules.yaml 2>/dev/null || true
	@kubectl delete namespace infra 2>/dev/null || true
	@echo "✅ Cleanup complete"

infra/port-forward-grafana:
	@echo "📊 Port-forwarding Grafana to localhost:3000..."
	@kubectl port-forward -n infra svc/prometheus-grafana 3000:80

infra/port-forward-prometheus:
	@echo "📈 Port-forwarding Prometheus to localhost:9090..."
	@kubectl port-forward -n infra svc/prometheus-kube-prometheus-prometheus 9090:9090

infra/port-forward-nats:
	@echo "💬 Port-forwarding NATS monitoring to localhost:8222..."
	@kubectl port-forward -n infra svc/nats 8222:8222

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
	@echo "  make setup                  - Create Kind cluster"
	@echo "  make deploy-dev             - Deploy to development"
	@echo "  make deploy-prod            - Deploy to production"
	@echo "  make status                 - Show cluster status"
	@echo "  make logs                   - Follow service logs"
	@echo "  make stop                   - Stop Kind cluster (without deleting)"
	@echo "  make start                  - Start Kind cluster"
	@echo "  make port-forward-admin     - Port-forward admin-panel to localhost:3000"
	@echo "  make port-forward-student   - Port-forward student-service to localhost:9080"
	@echo "  make port-forward-project   - Port-forward project-service to localhost:9052"
	@echo "  make cleanup                - Delete cluster"
	@echo ""
	@echo "Observability:"
	@echo "  make infra/setup            - Add Helm repositories"
	@echo "  make infra/deploy           - Deploy full infra stack (Prometheus + Alloy + NATS + Loki)"
	@echo "  make infra/deploy-prometheus - Deploy Prometheus stack only"
	@echo "  make infra/deploy-alloy     - Deploy Grafana Alloy only"
	@echo "  make infra/deploy-nats      - Deploy NATS only"
	@echo "  make infra/deploy-loki      - Deploy Loki logging stack"
	@echo "  make infra/deploy-tempo     - Deploy Tempo tracing"
	@echo "  make infra/deploy-alerts    - Deploy alerting rules"
	@echo "  make infra/status           - Show infra pods status"
	@echo "  make infra/port-forward-grafana    - Port-forward Grafana to localhost:3000"
	@echo "  make infra/port-forward-prometheus - Port-forward Prometheus to localhost:9090"
	@echo "  make infra/port-forward-nats       - Port-forward NATS monitoring to localhost:8222"
	@echo "  make infra/cleanup          - Remove infra stack"
	@echo ""
	@echo "  make help                   - Show this help message"
