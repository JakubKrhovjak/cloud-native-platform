.PHONY: build build-student build-project version test kind/setup kind/deploy kind/status kind/wait kind/stop kind/start kind/cleanup gke/auth gke/connect gke/deploy gke/status gke/full-deploy gke/ingress gke/resources gke/clean gke/prometheus gke/grafana tf/init tf/plan tf/apply tf/destroy tf/output tf/fmt tf/validate helm/template-kind helm/template-gke helm/uninstall infra/setup infra/deploy infra/deploy-gke infra/deploy-prometheus infra/deploy-prometheus-gke infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts infra/status infra/cleanup secrets/generate-kind secrets/list-kind secrets/list-gke secrets/view-gke help

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

kind/build: ## Build and push images to local registry
	@echo "📦 Building and pushing images to local registry..."
	@echo "🔨 Building student-service..."
	@cd services/student-service && KO_DOCKER_REPO=localhost:5001/student-service \
		ko build --bare --insecure-registry -t latest ./cmd/student-service
	@echo "🔨 Building project-service..."
	@cd services/project-service && KO_DOCKER_REPO=localhost:5001/project-service \
		ko build --bare --insecure-registry -t latest ./cmd/project-service
	@echo "🔨 Building admin-panel..."
	@docker build -t localhost:5001/admin-panel:latest services/admin
	@docker push localhost:5001/admin-panel:latest
	@echo "✅ All images built and pushed to localhost:5001"

kind/deploy: ## Deploy to Kind with Helm (requires images in local registry)
	@echo "🚀 Deploying to Kind with Helm..."
	@helm upgrade --install grud k8s/grud \
		-n grud --create-namespace \
		-f k8s/grud/values-kind.yaml \
		--wait
	@echo "✅ Deployed to Kind"

kind/build-deploy: kind/build kind/deploy ## Build images and deploy to Kind

kind/status: ## Show Kind cluster status
	@kubectl config use-context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || true
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
	@kubectl config use-context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || true
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
	@if ! docker ps -a --format '{{.Names}}' | grep -q "$(KIND_CLUSTER_NAME)-control-plane"; then \
		echo "❌ Kind cluster doesn't exist. Run 'make kind/setup' first."; \
		exit 1; \
	fi
	@docker start $(KIND_CLUSTER_NAME)-control-plane $(KIND_CLUSTER_NAME)-worker $(KIND_CLUSTER_NAME)-worker2 $(KIND_CLUSTER_NAME)-worker3 2>/dev/null || true
	@kubectl config use-context kind-$(KIND_CLUSTER_NAME)
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
	@gcloud auth application-default login
	@gcloud config set project $(GCP_PROJECT)
	@gcloud auth configure-docker $(GCP_REGION)-docker.pkg.dev
	@echo "✅ GCP authentication complete"

gke/connect: ## Connect to GKE cluster via Connect Gateway
	@echo "🔗 Connecting to GKE cluster via Connect Gateway..."
	@gcloud container fleet memberships get-credentials $(GKE_CLUSTER) \
		--location=$(GCP_REGION) \
		--project=$(GCP_PROJECT)
	@echo "✅ Connected to $(GKE_CLUSTER) via Connect Gateway"

gke/build: ## Build and push images to Artifact Registry
	@echo "📦 Building and pushing images to Artifact Registry (linux/amd64)..."
	@KO_DOCKER_REPO=$(GKE_REGISTRY)/student-service ko build --bare -t latest --platform=linux/amd64 ./services/student-service/cmd/student-service
	@KO_DOCKER_REPO=$(GKE_REGISTRY)/project-service ko build --bare -t latest --platform=linux/amd64 ./services/project-service/cmd/project-service
	@echo "📦 Building admin-panel via Cloud Build (AMD64)..."
	@gcloud builds submit services/admin --tag=$(GKE_REGISTRY)/admin-panel:latest --project=$(GCP_PROJECT) --quiet
	@echo "✅ Images pushed to $(GKE_REGISTRY)"

gke/deploy: gke/build  ## Deploy to GKE with Helm
	@echo "🔗 Connecting to GKE cluster via Connect Gateway..."
	@gcloud container fleet memberships get-credentials $(GKE_CLUSTER) --location=$(GCP_REGION) --project=$(GCP_PROJECT)
	@echo "🚀 Deploying to GKE with Helm..."
	@CLOUDSQL_IP=$$(cd $(TF_DIR) && terraform output -raw cloudsql_private_ip) && \
	helm upgrade --install grud k8s/grud \
		-n grud --create-namespace \
		-f k8s/grud/values-gke.yaml \
		--set studentService.image.repository=$(GKE_REGISTRY)/student-service \
		--set projectService.image.repository=$(GKE_REGISTRY)/project-service \
		--set adminPanel.image.repository=$(GKE_REGISTRY)/admin-panel \
		--set cloudSql.privateIp=$$CLOUDSQL_IP \
		--set secrets.gcp.projectId=$(GCP_PROJECT) \
		--set secrets.gcp.clusterLocation=$(GCP_ZONE) \
		--wait
	@kubectl rollout restart deployment -n grud
	@echo "🌐 Deploying Gateway API..."
	@kubectl apply -f k8s/gateway/
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

gke/gateway: ## Deploy Gateway API resources
	@echo "🌐 Deploying Gateway API..."
	@kubectl apply -f k8s/gateway/
	@echo "✅ Gateway deployed"
	@echo ""
	@echo "Check Gateway status:"
	@echo "  kubectl get gateway -n grud"
	@echo "  kubectl get httproute -A"

gke/gateway-status: ## Show Gateway and HTTPRoute status
	@echo "=== Gateway ==="
	@kubectl get gateway -n grud -o wide
	@echo ""
	@echo "=== HTTPRoutes ==="
	@kubectl get httproute -A
	@echo ""
	@echo "=== Gateway Details ==="
	@kubectl describe gateway grud-gateway -n grud | tail -20

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
	@echo "🔄 Importing existing resources if they exist..."
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_dns_managed_zone.grudapp projects/$(GCP_PROJECT)/managedZones/grudapp-zone 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_dns_record_set.root $(GCP_PROJECT)/grudapp-zone/grudapp.com./A 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_dns_record_set.grafana $(GCP_PROJECT)/grudapp-zone/grafana.grudapp.com./A 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_dns_record_set.admin $(GCP_PROJECT)/grudapp-zone/admin.grudapp.com./A 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_compute_managed_ssl_certificate.grud projects/$(GCP_PROJECT)/global/sslCertificates/grud-cert 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_certificate_manager_certificate_map.grud projects/$(GCP_PROJECT)/locations/global/certificateMaps/grud-certmap 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_certificate_manager_dns_authorization.grudapp projects/$(GCP_PROJECT)/locations/global/dnsAuthorizations/grudapp-dns-auth 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_certificate_manager_certificate.grud projects/$(GCP_PROJECT)/locations/global/certificates/grud-gateway-cert 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_certificate_manager_certificate_map_entry.root projects/$(GCP_PROJECT)/locations/global/certificateMaps/grud-certmap/certificateMapEntries/grud-root-entry 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_certificate_manager_certificate_map_entry.wildcard projects/$(GCP_PROJECT)/locations/global/certificateMaps/grud-certmap/certificateMapEntries/grud-wildcard-entry 2>/dev/null || true
	@cd $(TF_DIR) && terraform import -var="skip_kubernetes_provider=true" google_sql_database_instance.postgres grud-postgres 2>/dev/null || true
	@echo "📦 Stage 1: Bootstrap (VPC, GKE, Cloud SQL, DNS) - skip k8s providers..."
	@cd $(TF_DIR) && terraform apply -var="skip_kubernetes_provider=true" -auto-approve
	@echo "📦 Stage 2: Full apply with k8s providers..."
	@cd $(TF_DIR) && terraform apply -auto-approve
	@echo "✅ Terraform applied"

tf/destroy: ## Destroy Terraform resources (preserves DNS, Gateway certs, IPs)
	@echo "🗑️  Destroying Terraform resources..."
	@echo "🧹 Cleaning up Kubernetes resources first..."
	@echo "    - Deleting grud namespace (Gateway API, apps)..."
	@kubectl delete namespace grud --wait=true --timeout=5m 2>/dev/null || echo "    ⚠️  grud namespace not found (already deleted)"
	@echo "    - Deleting infra resources (Prometheus, Grafana, Alloy, etc.)..."
	@kubectl delete namespace infra --wait=true --timeout=5m 2>/dev/null || echo "    ⚠️  infra namespace not found (already deleted)"
	@echo "    - Waiting for GCP load balancers to cleanup (30s)..."
	@sleep 30
	@echo "    - Cleaning up Cloud SQL dependencies..."
	@echo "      Dropping student_user role from PostgreSQL..."
	@gcloud sql databases delete university --instance=grud-postgres --quiet 2>/dev/null || echo "      ⚠️  Database already deleted"
	@gcloud sql databases delete projects --instance=grud-postgres --quiet 2>/dev/null || echo "      ⚠️  Database already deleted"
	@echo "✅ Kubernetes cleanup complete"
	@echo ""
	@echo "🛡️  Removing protected resources from state..."
	@echo "    - DNS zone and records"
	@cd $(TF_DIR) && terraform state rm google_dns_managed_zone.grudapp 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_dns_record_set.root 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_dns_record_set.grafana 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_dns_record_set.admin 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_dns_record_set.cert_validation 2>/dev/null || true
	@echo "    - Static IP"
	@cd $(TF_DIR) && terraform state rm 'data.google_compute_global_address.ingress_ip' 2>/dev/null || true
	@echo "    - Certificate Manager (Gateway API)"
	@cd $(TF_DIR) && terraform state rm google_certificate_manager_certificate_map.grud 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_certificate_manager_dns_authorization.grudapp 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_certificate_manager_certificate.grud 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_certificate_manager_certificate_map_entry.root 2>/dev/null || true
	@cd $(TF_DIR) && terraform state rm google_certificate_manager_certificate_map_entry.wildcard 2>/dev/null || true
	@echo "📝 Note: Old Ingress SSL cert (google_compute_managed_ssl_certificate.grud) WILL be deleted"
	@echo "🚀 Running terraform destroy..."
	@cd $(TF_DIR) && terraform destroy -auto-approve -refresh=false

tf/output: ## Show Terraform outputs
	@cd $(TF_DIR) && terraform output

gke/ingress: ## Show Ingress status and external IP
	@echo "=== Shared Static IP (Terraform) ==="
	@cd $(TF_DIR) && terraform output ingress_ip 2>/dev/null || echo "Not created yet"
	@echo ""
	@echo "=== App Ingress (grud namespace) ==="
	@kubectl get ingress -n grud 2>/dev/null || echo "No ingress found"
	@echo ""
	@echo "=== Grafana Ingress (infra namespace) ==="
	@kubectl get ingress -n infra 2>/dev/null || echo "No ingress found"
	@echo ""
	@echo "URLs (after deployment):"
	@echo "  API:     https://grudapp.com/api"
	@echo "  Grafana: https://grafana.grudapp.com"

gcp/resources: ## List all GCP resources in project
	@echo "=== GKE Clusters ==="
	@gcloud container clusters list --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== GKE Node Pools ==="
	@gcloud container node-pools list --cluster=$(GKE_CLUSTER) --zone=$(GCP_ZONE) --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== Cloud SQL Instances ==="
	@gcloud sql instances list --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
	@echo ""
	@echo "=== Cloud SQL Databases ==="
	@gcloud sql databases list --instance=grud-postgres --project=$(GCP_PROJECT) 2>/dev/null || echo "None"
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

infra/deploy: infra/setup infra/deploy-prometheus infra/deploy-alloy infra/deploy-nats infra/deploy-loki infra/deploy-tempo infra/deploy-alerts argocd/install ## Deploy full observability stack (Kind)
	@echo "✅ Full observability stack deployed"

infra/deploy-gke: infra/setup ## Deploy full observability stack (GKE)
	@echo "🔗 Connecting to GKE cluster via Connect Gateway..."
	@gcloud container fleet memberships get-credentials $(GKE_CLUSTER) --location=$(GCP_REGION) --project=$(GCP_PROJECT)
	@$(MAKE) infra/deploy-prometheus-gke
	@$(MAKE) infra/deploy-alloy
	@$(MAKE) infra/deploy-nats
	@$(MAKE) infra/deploy-loki
	@$(MAKE) infra/deploy-tempo
	@$(MAKE) infra/deploy-alerts
	@$(MAKE) argocd/install
	@echo "✅ Full observability stack deployed for GKE"

infra/status: ## Show infra pods status
	@echo "📊 Observability stack status:"
	@kubectl get pods -n infra

infra/resources: ## Show infra node resource utilization
	@echo "📊 Infra node resource utilization:"
	@kubectl describe node -l node-type=infra | grep -A10 "Allocated resources:"

infra/cleanup: ## Remove observability stack
	@echo "🧹 Cleaning up observability stack..."
	@$(MAKE) argocd/uninstall
	@helm uninstall loki -n infra 2>/dev/null || true
	@helm uninstall tempo -n infra 2>/dev/null || true
	@helm uninstall prometheus -n infra 2>/dev/null || true
	@helm uninstall alloy -n infra 2>/dev/null || true
	@kubectl delete -f k8s/infra/nats.yaml 2>/dev/null || true
	@kubectl delete -f k8s/infra/alerting-rules.yaml 2>/dev/null || true
	@echo "✅ Cleanup complete"

# =============================================================================
# ArgoCD
# =============================================================================
argocd/install: ## Install ArgoCD
	@echo "🚀 Installing ArgoCD..."
	@kubectl create namespace infra --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -n infra -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	@kubectl apply -f k8s/infra/argocd/install.yaml
	@echo "🔧 Applying node selectors and tolerations..."
	@kubectl patch deployment argocd-server -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@kubectl patch deployment argocd-repo-server -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@kubectl patch deployment argocd-redis -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@kubectl patch deployment argocd-dex-server -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@kubectl patch deployment argocd-notifications-controller -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@kubectl patch deployment argocd-applicationset-controller -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@kubectl patch statefulset argocd-application-controller -n infra -p '{"spec":{"template":{"spec":{"nodeSelector":{"node-type":"infra"},"tolerations":[{"key":"workload","operator":"Equal","value":"infra","effect":"NoSchedule"}]}}}}'
	@echo "⏳ Waiting for ArgoCD to be ready..."
	@kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n infra
	@echo "✅ ArgoCD installed successfully"
	@echo ""
	@echo "Access ArgoCD:"
	@echo "  URL: http://localhost:30080"
	@echo "  Username: admin"
	@echo "  Password: Run 'make argocd/password'"

argocd/password: ## Get ArgoCD admin password
	@echo "🔑 ArgoCD Admin Password:"
	@kubectl -n infra get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
	@echo ""

argocd/login: ## Login to ArgoCD CLI
	@echo "🔐 Logging into ArgoCD..."
	@argocd login localhost:30080 --username admin --password $$(kubectl -n infra get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d) --insecure

argocd/deploy-apps: ## Deploy ArgoCD applications
	@echo "📦 Deploying ArgoCD applications..."
	@kubectl apply -f k8s/infra/argocd/application-grud.yaml
	@kubectl apply -f k8s/infra/argocd/application-infra.yaml
	@echo "✅ ArgoCD applications deployed"

argocd/status: ## Show ArgoCD applications status
	@echo "📊 ArgoCD Applications Status:"
	@kubectl get applications -n infra
	@echo ""
	@echo "Pods in infra namespace:"
	@kubectl get pods -n infra

argocd/sync: ## Sync all ArgoCD applications
	@echo "🔄 Syncing ArgoCD applications..."
	@argocd app sync grud-app
	@argocd app sync monitoring-stack
	@argocd app sync nats

argocd/ui: ## Open ArgoCD UI
	@echo "🌐 Opening ArgoCD UI..."
	@open http://localhost:30080

argocd/uninstall: ## Uninstall ArgoCD
	@echo "🗑️  Uninstalling ArgoCD..."
	@kubectl delete -f k8s/infra/argocd/application-grud.yaml 2>/dev/null || true
	@kubectl delete -f k8s/infra/argocd/application-infra.yaml 2>/dev/null || true
	@kubectl delete -n infra -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml 2>/dev/null || true
	@echo "✅ ArgoCD uninstalled"

# =============================================================================
# Secret Management
# =============================================================================
secrets/generate-kind: ## Generate secrets for Kind cluster
	@echo "🔐 Generating secrets for Kind cluster..."
	@./scripts/generate-secrets.sh kind
	@echo "✅ Secrets generated for Kind"

secrets/list-kind: ## List secrets in Kind cluster
	@echo "📋 Secrets in Kind cluster:"
	@kubectl get secrets -n grud -l app=grud,component=secrets

secrets/list-gke: ## List secrets in Google Secret Manager
	@echo "📋 Secrets in Google Secret Manager:"
	@gcloud secrets list --filter="name:grud-"

secrets/view-gke: ## View secret values in Google Secret Manager (for debugging)
	@echo "🔍 JWT Secret:"
	@gcloud secrets versions access latest --secret=grud-jwt-secret
	@echo ""
	@echo "🔍 Student DB Credentials:"
	@gcloud secrets versions access latest --secret=grud-student-db-credentials | jq
	@echo ""
	@echo "🔍 Project DB Credentials:"
	@gcloud secrets versions access latest --secret=grud-project-db-credentials | jq

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
	@echo "  make gke/auth           - Authenticate with GCP"
	@echo "  make gke/connect        - Connect to GKE via Connect Gateway"
	@echo "  make gke/deploy         - Build and deploy to GKE with Helm"
	@echo "  make gke/full-deploy    - Full GKE deployment (terraform + helm)"
	@echo "  make gke/status         - Show GKE status"
	@echo "  make gke/resources      - Show resource utilization"
	@echo "  make gke/ingress        - Show Ingress status and IPs"
	@echo "  make gke/clean          - Clean uninstall helm release"
	@echo "  make gke/prometheus     - Port-forward Prometheus (localhost:9090)"
	@echo "  make gke/grafana        - Port-forward Grafana (localhost:3000)"
	@echo ""
	@echo "Observability:"
	@echo "  make infra/deploy       - Deploy full observability stack"
	@echo "  make infra/status       - Show infra pods status"
	@echo "  make infra/cleanup      - Remove observability stack"
	@echo ""
	@echo "ArgoCD:"
	@echo "  make argocd/install     - Install ArgoCD"
	@echo "  make argocd/password    - Get ArgoCD admin password"
	@echo "  make argocd/deploy-apps - Deploy ArgoCD applications"
	@echo "  make argocd/status      - Show ArgoCD applications status"
	@echo "  make argocd/ui          - Open ArgoCD UI"
	@echo "  make argocd/uninstall   - Uninstall ArgoCD"
	@echo ""
	@echo "Secret Management:"
	@echo "  make secrets/generate-kind - Generate secrets for Kind"
	@echo "  make secrets/list-kind     - List secrets in Kind cluster"
	@echo "  make secrets/list-gke      - List secrets in Google Secret Manager"
	@echo "  make secrets/view-gke      - View secret values in GSM (debug)"
	@echo ""
	@echo "  Note: GKE secrets are managed by Terraform (see terraform/secrets.tf)"
	@echo ""
	@echo "Helm:"
	@echo "  make helm/template-kind - Show Kind templates"
	@echo "  make helm/template-gke  - Show GKE templates"
	@echo "  make helm/uninstall     - Uninstall Helm release"
