.PHONY: build build-student build-project version test kind/setup kind/deploy kind/status kind/wait kind/stop kind/start kind/cleanup gke/auth gke/connect gke/deploy gke/status gke/full-deploy gke/ingress gke/resources gke/clean gke/prometheus gke/grafana tf/init tf/plan tf/apply tf/destroy tf/output tf/fmt tf/validate helm/template-kind helm/template-gke helm/uninstall infra/setup infra/deploy infra/deploy-gke infra/deploy-prometheus infra/deploy-prometheus-gke infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts infra/status infra/cleanup help

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
	@cd services/student-service && go build -ldflags="$(STUDENT_LDFLAGS)" -o ../../bin/student-service ./cmd/student-service
	@echo "✅ student-service → bin/student-service"

build-project: ## Build project-service
	@echo "🔨 Building project-service $(VERSION) ($(GIT_COMMIT))..."
	@mkdir -p bin
	@cd services/project-service && go build -ldflags="$(PROJECT_LDFLAGS)" -o ../../bin/project-service ./cmd/project-service
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
	@cd services/student-service && KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=grud-cluster ko build --bare ./cmd/student-service 2>&1 | grep "Loading" | sed 's/.*Loading //' > /tmp/student-image.txt
	@cd services/project-service && KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=grud-cluster ko build --bare ./cmd/project-service 2>&1 | grep "Loading" | sed 's/.*Loading //' > /tmp/project-image.txt
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
# GKE Cluster (infrastructure via Terraform)
# =============================================================================
GCP_PROJECT := rugged-abacus-483006-r5
GCP_REGION := europe-west1
GCP_ZONE := europe-west1-b
GKE_CLUSTER := grud-cluster
GKE_REGISTRY := $(GCP_REGION)-docker.pkg.dev/$(GCP_PROJECT)/grud

gke/auth: ## Authenticate with GCP
	@echo "🔐 Authenticating with GCP..."
	@gcloud auth login
	@gcloud config set project $(GCP_PROJECT)
	@gcloud auth configure-docker $(GCP_REGION)-docker.pkg.dev
	@echo "✅ GCP authentication complete"

gke/connect: ## Connect to GKE cluster
	@echo "🔗 Connecting to GKE cluster..."
	@gcloud container clusters get-credentials $(GKE_CLUSTER) \
		--zone=$(GCP_ZONE) \
		--project=$(GCP_PROJECT)
	@echo "✅ Connected to $(GKE_CLUSTER)"

gke/build: ## Build and push images to Artifact Registry
	@echo "📦 Building and pushing images to Artifact Registry..."
	@KO_DOCKER_REPO=$(GKE_REGISTRY)/student-service ko build --bare -t latest ./services/student-service/cmd/student-service
	@KO_DOCKER_REPO=$(GKE_REGISTRY)/project-service ko build --bare -t latest ./services/project-service/cmd/project-service
	@echo "✅ Images pushed to $(GKE_REGISTRY)"

gke/deploy: gke/connect gke/build  ## Deploy to GKE with Helm
	@echo "🚀 Deploying to GKE with Helm..."
	@CLOUDSQL_IP=$$(cd $(TF_DIR) && terraform output -raw cloudsql_private_ip) && \
	helm upgrade --install grud k8s/grud \
		-n grud --create-namespace \
		-f k8s/grud/values-gke.yaml \
		--set studentService.image.repository=$(GKE_REGISTRY)/student-service \
		--set projectService.image.repository=$(GKE_REGISTRY)/project-service \
		--set cloudSql.privateIp=$$CLOUDSQL_IP \
		--wait
	@kubectl rollout restart deployment -n grud
	@echo "✅ Deployed to GKE"

gke/status: ## Show GKE cluster status
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

gke/resources: ## Show resource utilization for grud namespace and nodes
	@echo "📊 Resource Utilization"
	@echo ""
	@echo "=== Node Resources ==="
	@kubectl top nodes
	@echo ""
	@echo "=== Pod Resources (grud namespace) ==="
	@kubectl top pods -n grud --containers
	@echo ""
	@echo "=== Resource Requests/Limits ==="
	@kubectl get pods -n grud -o custom-columns="\
NAME:.metadata.name,\
CPU_REQ:.spec.containers[*].resources.requests.cpu,\
CPU_LIM:.spec.containers[*].resources.limits.cpu,\
MEM_REQ:.spec.containers[*].resources.requests.memory,\
MEM_LIM:.spec.containers[*].resources.limits.memory"

gke/clean: ## Clean uninstall grud helm release and all pods
	@echo "🧹 Cleaning grud namespace..."
	@helm uninstall grud -n grud --wait 2>/dev/null || true
	@echo "✅ Cleanup complete"

gke/prometheus: ## Port-forward Prometheus (localhost:9090)
	@echo "📊 Port-forwarding Prometheus to localhost:9090..."
	@kubectl port-forward -n infra svc/prometheus-kube-prometheus-prometheus 9090:9090

gke/grafana: ## Port-forward Grafana (localhost:3000)
	@echo "📈 Port-forwarding Grafana to localhost:3000..."
	@kubectl port-forward -n infra svc/prometheus-grafana 3000:80

gke/full-deploy: ## Full GKE deployment (terraform + helm)
	@$(MAKE) tf/init
	@$(MAKE) tf/plan
	@$(MAKE) tf/apply
	@$(MAKE) gke/connect
	@$(MAKE) infra/setup
	@$(MAKE) infra/deploy-gke
	@$(MAKE) gke/deploy
	@echo "✅ Full GKE deployment complete"

# =============================================================================
# Terraform
# =============================================================================
TF_DIR := terraform

tf/init: ## Initialize Terraform
	@echo "🔧 Initializing Terraform..."
	@cd $(TF_DIR) && terraform init
	@echo "✅ Terraform initialized"

tf/plan: ## Plan Terraform changes
	@echo "📋 Planning Terraform changes..."
	@cd $(TF_DIR) && terraform plan

tf/apply: ## Apply Terraform configuration
	@echo "🚀 Applying Terraform configuration..."
	@cd $(TF_DIR) && terraform apply -auto-approve
	@echo "✅ Terraform applied"

tf/destroy: ## Destroy Terraform resources
	@echo "🗑️  Destroying Terraform resources..."
	@cd $(TF_DIR) && terraform destroy -auto-approve

tf/output: ## Show Terraform outputs
	@cd $(TF_DIR) && terraform output

gke/ingress: ## Show Ingress status and external IP
	@echo "=== Reserved Static IPs (Terraform) ==="
	@cd $(TF_DIR) && terraform output ingress_ip 2>/dev/null || echo "Not created yet"
	@cd $(TF_DIR) && terraform output grafana_ip 2>/dev/null || echo "Not created yet"
	@echo ""
	@echo "=== App Ingress (grud namespace) ==="
	@kubectl get ingress -n grud 2>/dev/null || echo "No ingress found"
	@echo ""
	@echo "=== Grafana Ingress (infra namespace) ==="
	@kubectl get ingress -n infra 2>/dev/null || echo "No ingress found"
	@echo ""
	@echo "URLs (after deployment):"
	@echo "  API:     http://$$(cd $(TF_DIR) && terraform output -raw ingress_ip 2>/dev/null)/api"
	@echo "  Grafana: http://$$(cd $(TF_DIR) && terraform output -raw grafana_ip 2>/dev/null)"

gcp/resources: ## List all GCP resources in project
	@echo "=== GKE Clusters ==="
	@gcloud container clusters list --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== Cloud SQL Instances ==="
	@gcloud sql instances list --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== Compute Instances (VMs) ==="
	@gcloud compute instances list --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== VPC Networks ==="
	@gcloud compute networks list --project=$(GCP_PROJECT) 2>/dev/null
	@echo ""
	@echo "=== Artifact Registry ==="
	@gcloud artifacts repositories list --project=$(GCP_PROJECT) --location=$(GCP_REGION) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== Service Accounts (app) ==="
	@gcloud iam service-accounts list --project=$(GCP_PROJECT) --filter="email~student-service OR email~project-service" 2>/dev/null || echo "None"

tf/fmt: ## Format Terraform files
	@cd $(TF_DIR) && terraform fmt -recursive

tf/validate: ## Validate Terraform configuration
	@cd $(TF_DIR) && terraform validate

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

infra/deploy-prometheus: ## Deploy Prometheus stack (Kind)
	@echo "🔥 Deploying Prometheus stack..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		-n infra \
		-f k8s/infra/prometheus-values.yaml \
		--wait
	@echo "📊 Deploying Grafana dashboards and datasources..."
	@kubectl apply -f k8s/infra/grafana-dashboard-configmap.yaml
	@kubectl apply -f k8s/infra/grafana-datasources.yaml
	@echo "✅ Prometheus stack deployed"

infra/deploy-prometheus-gke: ## Deploy Prometheus stack (GKE with Ingress)
	@echo "🔥 Deploying Prometheus stack for GKE..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		-n infra \
		-f k8s/infra/prometheus-values.yaml \
		-f k8s/infra/prometheus-values-gke.yaml \
		--wait
	@echo "📊 Deploying Grafana dashboards and datasources..."
	@kubectl apply -f k8s/infra/grafana-dashboard-configmap.yaml
	@kubectl apply -f k8s/infra/grafana-datasources.yaml
	@echo "✅ Prometheus stack deployed with Ingress"

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

infra/deploy: infra/setup infra/deploy-prometheus infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts ## Deploy full observability stack (Kind)
	@echo "✅ Full observability stack deployed"

infra/deploy-gke: infra/setup infra/deploy-prometheus-gke infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts ## Deploy full observability stack (GKE)
	@echo "✅ Full observability stack deployed for GKE"

infra/status: ## Show infra pods status
	@echo "📊 Observability stack status:"
	@kubectl get pods -n infra

infra/resources: ## Show infra node resource utilization
	@echo "📊 Infra node resource utilization:"
	@kubectl describe node -l node-type=infra | grep -A10 "Allocated resources:"

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
	@echo "  make gke/resources      - Show resource utilization"
	@echo "  make gke/clean          - Clean uninstall helm release"
	@echo "  make gke/prometheus     - Port-forward Prometheus (localhost:9090)"
	@echo "  make gke/grafana        - Port-forward Grafana (localhost:3000)"
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
