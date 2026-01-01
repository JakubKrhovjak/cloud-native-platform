.PHONY: build build-student build-project version test kind/setup kind/deploy kind/status kind/wait kind/stop kind/start kind/cleanup gke/auth gke/enable-apis gke/create-registry gke/create-cluster gke/connect gke/deploy gke/status gke/delete-cluster helm/template-kind helm/template-gke helm/uninstall infra/setup infra/deploy infra/deploy-prometheus infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts infra/status infra/cleanup help

# =============================================================================
# Build Configuration
# =============================================================================
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

STUDENT_LDFLAGS := -X 'student-service/internal/app.Version=$(VERSION)' \
                   -X 'student-service/internal/app.GitCommit=$(GIT_COMMIT)' \
                   -X 'student-service/internal/app.BuildTime=$(BUILD_TIME)'

PROJECT_LDFLAGS := -X 'project-service/internal/app.Version=$(VERSION)' \
                   -X 'project-service/internal/app.GitCommit=$(GIT_COMMIT)' \
                   -X 'project-service/internal/app.BuildTime=$(BUILD_TIME)'

# =============================================================================
# Build Targets
# =============================================================================
build: build-student build-project ## Build all services
	@echo "✅ All services built successfully"

build-student: ## Build student-service
	@echo "🔨 Building student-service $(VERSION) ($(GIT_COMMIT))..."
	@mkdir -p bin
	@cd services/student-service && go build -ldflags="$(STUDENT_LDFLAGS)" -o ../../bin/student-service ./cmd/server
	@echo "✅ student-service → bin/student-service"

build-project: ## Build project-service
	@echo "🔨 Building project-service $(VERSION) ($(GIT_COMMIT))..."
	@mkdir -p bin
	@cd services/project-service && go build -ldflags="$(PROJECT_LDFLAGS)" -o ../../bin/project-service ./cmd/server
	@echo "✅ project-service → bin/project-service"

version: ## Show version info
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# =============================================================================
# Test Targets
# =============================================================================
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@go test ./services/student-service/... ./services/project-service/...

# =============================================================================
# Kind Cluster
# =============================================================================
KIND_CLUSTER_NAME := grud-cluster

kind/setup: ## Create Kind cluster
	@echo "🚀 Creating Kind cluster..."
	@./scripts/kind-setup.sh

kind/deploy: ## Deploy to Kind with Helm
	@echo "🚀 Deploying to Kind with Helm..."
	@echo "📦 Building Go services with ko..."
	@cd services/student-service && KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=grud-cluster ko build --bare ./cmd/server 2>&1 | grep "Loading" | sed 's/.*Loading //' > /tmp/student-image.txt
	@cd services/project-service && KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=grud-cluster ko build --bare ./cmd/server 2>&1 | grep "Loading" | sed 's/.*Loading //' > /tmp/project-image.txt
	@echo "📦 Building admin-panel..."
	@docker build -t admin-panel:latest services/admin
	@kind load docker-image admin-panel:latest --name grud-cluster
	@echo "🚀 Deploying with Helm..."
	@helm upgrade --install grud k8s/grud \
		-n grud --create-namespace \
		-f k8s/grud/values-kind.yaml \
		--set studentService.image.repository=$$(cat /tmp/student-image.txt) \
		--set projectService.image.repository=$$(cat /tmp/project-image.txt) \
		--wait
	@echo "✅ Deployed to Kind"

kind/status: ## Show Kind cluster status
	@echo "📋 Kind Cluster Status"
	@echo ""
	@echo "Nodes:"
	@kubectl get nodes -o wide
	@echo ""
	@echo "Deployments:"
	@kubectl get deployments -n grud
	@echo ""
	@echo "Pods:"
	@kubectl get pods -n grud -o wide
	@echo ""
	@echo "Services:"
	@kubectl get services -n grud

kind/wait: ## Wait for all resources to be ready
	@echo "⏳ Waiting for databases..."
	@kubectl wait --for=condition=Ready pod -l app=student-db -n grud --timeout=300s
	@kubectl wait --for=condition=Ready pod -l app=project-db -n grud --timeout=300s
	@echo "⏳ Waiting for services..."
	@kubectl wait --for=condition=Available deployment/student-service -n grud --timeout=300s
	@kubectl wait --for=condition=Available deployment/project-service -n grud --timeout=300s
	@kubectl wait --for=condition=Available deployment/admin-panel -n grud --timeout=300s
	@echo "✅ All resources ready!"

kind/stop: ## Stop Kind cluster (without deleting)
	@echo "⏸️  Stopping Kind cluster..."
	@docker stop $(KIND_CLUSTER_NAME)-control-plane $(KIND_CLUSTER_NAME)-worker $(KIND_CLUSTER_NAME)-worker2 $(KIND_CLUSTER_NAME)-worker3 2>/dev/null || true
	@echo "✅ Cluster stopped"

kind/start: ## Start Kind cluster
	@echo "▶️  Starting Kind cluster..."
	@docker start $(KIND_CLUSTER_NAME)-control-plane $(KIND_CLUSTER_NAME)-worker $(KIND_CLUSTER_NAME)-worker2 $(KIND_CLUSTER_NAME)-worker3 2>/dev/null || true
	@echo "⏳ Waiting for cluster to be ready..."
	@kubectl wait --for=condition=Ready nodes --all --timeout=120s
	@echo "✅ Cluster started and ready!"

kind/cleanup: ## Delete Kind cluster
	@./scripts/cleanup.sh

# =============================================================================
# GKE Cluster
# =============================================================================
GCP_PROJECT := rugged-abacus-483006-r5
GCP_REGION := europe-west1
GKE_CLUSTER := grud-cluster
GKE_REGISTRY := $(GCP_REGION)-docker.pkg.dev/$(GCP_PROJECT)/grud

gke/auth: ## Authenticate with GCP
	@echo "🔐 Authenticating with GCP..."
	@gcloud auth login
	@gcloud config set project $(GCP_PROJECT)
	@gcloud auth configure-docker $(GCP_REGION)-docker.pkg.dev
	@echo "✅ GCP authentication complete"

gke/enable-apis: ## Enable required GCP APIs
	@echo "🔧 Enabling required APIs..."
	@gcloud services enable container.googleapis.com --project=$(GCP_PROJECT)
	@gcloud services enable artifactregistry.googleapis.com --project=$(GCP_PROJECT)
	@echo "✅ APIs enabled"

gke/create-registry: ## Create Artifact Registry repository
	@echo "📦 Creating Artifact Registry..."
	@gcloud artifacts repositories create grud \
		--repository-format=docker \
		--location=$(GCP_REGION) \
		--description="GRUD container images" || echo "Repository already exists"
	@echo "✅ Artifact Registry ready"

gke/create-cluster: ## Create GKE Standard cluster
	@echo "🚀 Creating GKE Standard cluster..."
	@gcloud container clusters create $(GKE_CLUSTER) \
		--region=$(GCP_REGION) \
		--project=$(GCP_PROJECT) \
		--num-nodes=1 \
		--machine-type=e2-small \
		--disk-size=20GB
	@echo "✅ GKE cluster created"

gke/connect: ## Connect to existing GKE cluster
	@echo "🔗 Connecting to GKE cluster..."
	@gcloud container clusters get-credentials $(GKE_CLUSTER) \
		--region=$(GCP_REGION) \
		--project=$(GCP_PROJECT)
	@echo "✅ Connected to $(GKE_CLUSTER)"

gke/setup: gke/auth gke/create-registry ## Full GKE setup (auth + registry)
	@echo "✅ GKE setup complete"

gke/full-setup: gke/auth gke/enable-apis gke/create-registry gke/create-cluster gke/connect ## Full GKE setup including cluster creation
	@echo "✅ GKE full setup complete"

gke/deploy: gke/connect ## Deploy to GKE with Helm
	@echo "🚀 Deploying to GKE with Helm..."
	@helm upgrade --install grud k8s/grud \
		-n grud --create-namespace \
		-f k8s/grud/values-gke.yaml \
		--set studentService.image.repository=$(GKE_REGISTRY)/student-service \
		--set projectService.image.repository=$(GKE_REGISTRY)/project-service \
		--wait
	@echo "✅ Deployed to GKE"

gke/status: gke/connect ## Show GKE cluster status
	@echo "📋 GKE Cluster Status"
	@echo ""
	@echo "Nodes:"
	@kubectl get nodes -o wide
	@echo ""
	@echo "Deployments:"
	@kubectl get deployments -n grud
	@echo ""
	@echo "Pods:"
	@kubectl get pods -n grud -o wide
	@echo ""
	@echo "Services:"
	@kubectl get services -n grud

gke/cleanup: ## Delete GKE cluster
	@echo "🗑️  Deleting GKE cluster..."
	@gcloud container clusters delete $(GKE_CLUSTER) \
		--region=$(GCP_REGION) \
		--project=$(GCP_PROJECT) \
		--quiet
	@echo "✅ GKE cluster deleted"

# =============================================================================
# Helm Utilities
# =============================================================================
helm/template-kind: ## Show rendered templates for Kind
	@helm template grud k8s/grud -f k8s/grud/values-kind.yaml

helm/template-gke: ## Show rendered templates for GKE
	@helm template grud k8s/grud -f k8s/grud/values-gke.yaml

helm/uninstall: ## Uninstall Helm release
	@echo "🗑️  Uninstalling Helm release..."
	@helm uninstall grud -n grud || true
	@echo "✅ Helm release uninstalled"

# =============================================================================
# Observability Stack
# =============================================================================
infra/setup: ## Add Helm repositories
	@echo "📦 Adding Helm repositories..."
	@helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	@helm repo add grafana https://grafana.github.io/helm-charts
	@helm repo update
	@echo "✅ Helm repositories added"

infra/deploy-prometheus: ## Deploy Prometheus stack
	@echo "🔥 Deploying Prometheus stack..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		-n infra \
		-f k8s/infra/prometheus-values.yaml \
		--wait
	@echo "📊 Deploying Grafana dashboards..."
	@kubectl apply -f k8s/infra/grafana-dashboard-configmap.yaml
	@echo "✅ Prometheus stack deployed"

infra/deploy-alloy: ## Deploy Grafana Alloy
	@echo "📡 Deploying Grafana Alloy..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install alloy grafana/alloy \
		-n infra \
		-f k8s/infra/alloy-values.yaml \
		--wait
	@echo "✅ Grafana Alloy deployed"

infra/deploy-nats: ## Deploy NATS
	@echo "💬 Deploying NATS..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f k8s/infra/nats.yaml
	@echo "✅ NATS deployed"

infra/deploy-loki: ## Deploy Loki logging
	@echo "📝 Deploying Loki..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install loki grafana/loki \
		-n infra \
		-f k8s/infra/loki-values.yaml \
		--wait
	@echo "✅ Loki deployed"

infra/deploy-tempo: ## Deploy Tempo tracing
	@echo "🔍 Deploying Tempo..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install tempo grafana/tempo \
		-n infra \
		-f k8s/infra/tempo-values.yaml \
		--wait
	@echo "✅ Tempo deployed"

infra/deploy-alerts: ## Deploy alerting rules
	@echo "🚨 Deploying alerting rules..."
	@kubectl apply -f k8s/infra/alerting-rules.yaml
	@echo "✅ Alerting rules deployed"

infra/deploy: infra/setup infra/deploy-prometheus infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts ## Deploy full observability stack
	@echo "✅ Full observability stack deployed"

infra/status: ## Show infra pods status
	@echo "📊 Observability stack status:"
	@kubectl get pods -n infra

infra/cleanup: ## Remove observability stack
	@echo "🧹 Cleaning up observability stack..."
	@helm uninstall loki -n infra 2>/dev/null || true
	@helm uninstall tempo -n infra 2>/dev/null || true
	@helm uninstall prometheus -n infra 2>/dev/null || true
	@helm uninstall alloy -n infra 2>/dev/null || true
	@kubectl delete -f k8s/infra/nats.yaml 2>/dev/null || true
	@kubectl delete -f k8s/infra/alerting-rules.yaml 2>/dev/null || true
	@kubectl delete namespace infra 2>/dev/null || true
	@echo "✅ Cleanup complete"

# =============================================================================
# Help
# =============================================================================
help: ## Show this help
	@echo "GRUD - Available Commands"
	@echo ""
	@echo "Build:"
	@echo "  make build              - Build all services"
	@echo "  make version            - Show version info"
	@echo "  make test               - Run all tests"
	@echo ""
	@echo "Kind Cluster:"
	@echo "  make kind/setup         - Create Kind cluster"
	@echo "  make kind/deploy        - Deploy to Kind with Helm"
	@echo "  make kind/status        - Show cluster status"
	@echo "  make kind/wait          - Wait for resources to be ready"
	@echo "  make kind/stop          - Stop Kind cluster"
	@echo "  make kind/start         - Start Kind cluster"
	@echo "  make kind/cleanup       - Delete Kind cluster"
	@echo ""
	@echo "GKE Cluster:"
	@echo "  make gke/setup          - Full GKE setup (auth + registry)"
	@echo "  make gke/full-setup     - Full setup including cluster creation"
	@echo "  make gke/deploy         - Deploy to GKE with Helm"
	@echo "  make gke/status         - Show GKE status"
	@echo "  make gke/cleanup        - Delete GKE cluster"
	@echo ""
	@echo "Observability:"
	@echo "  make infra/deploy       - Deploy full observability stack"
	@echo "  make infra/status       - Show infra pods status"
	@echo "  make infra/cleanup      - Remove observability stack"
	@echo ""
	@echo "Helm:"
	@echo "  make helm/template-kind - Show Kind templates"
	@echo "  make helm/template-gke  - Show GKE templates"
	@echo "  make helm/uninstall     - Uninstall Helm release"
