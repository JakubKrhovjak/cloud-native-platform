.PHONY: help setup install-operator deploy deploy-dev deploy-prod status logs stop start cleanup

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

# Environment
export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME=grud-cluster

help: ## Show this help
	@echo "$(BLUE)GRUD Kubernetes Deployment$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'

setup: ## Create Kind cluster (required first step)
	@echo "$(BLUE)🚀 Creating Kind cluster...$(NC)"
	@../scripts/kind-setup.sh

deploy: ## Deploy everything with Ko + Kustomize (base config)
	@echo "$(BLUE)🚀 Building and deploying services...$(NC)"
	@kustomize build . | ko resolve -f - | kubectl apply -f -
	@echo "$(GREEN)✅ Deployment complete!$(NC)"

deploy-dev: ## Deploy with development overlay (1 replica each)
	@echo "$(BLUE)🚀 Deploying to development...$(NC)"
	@echo "$(BLUE)📦 Building admin panel Docker image...$(NC)"
	@docker build -t admin-panel:latest ../services/admin
	@echo "$(BLUE)📤 Loading admin panel image into Kind cluster...$(NC)"
	@kind load docker-image admin-panel:latest --name $(KIND_CLUSTER_NAME)
	@echo "$(BLUE)🚢 Deploying services with Kustomize...$(NC)"
	@kustomize build overlays/dev/ | ko resolve -f - | kubectl apply -f -
	@echo "$(BLUE)🔄 Restarting admin-panel to use new image...$(NC)"
	@kubectl rollout restart deployment/admin-panel -n grud
	@echo "$(GREEN)✅ Development deployment complete!$(NC)"

deploy-prod: ## Deploy with production overlay (3 replicas each)
	@echo "$(BLUE)🚀 Deploying to production...$(NC)"
	@kustomize build overlays/prod/ | ko resolve -f - | kubectl apply -f -
	@echo "$(GREEN)✅ Production deployment complete!$(NC)"

build-only: ## Only build images without deploying
	@echo "$(BLUE)🔨 Building images with Ko...$(NC)"
	@ko build ./student-service/cmd/server
	@ko build ./project-service/cmd/server

status: ## Show cluster status
	@echo "$(BLUE)📋 Cluster Status$(NC)"
	@echo ""
	@echo "$(YELLOW)Nodes:$(NC)"
	@kubectl get nodes -o wide
	@echo ""
	@echo "$(YELLOW)Deployments:$(NC)"
	@kubectl get deployments -n grud
	@echo ""
	@echo "$(YELLOW)StatefulSets:$(NC)"
	@kubectl get statefulsets -n grud
	@echo ""
	@echo "$(YELLOW)Pods:$(NC)"
	@kubectl get pods -n grud -o wide
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@kubectl get services -n grud
	@echo ""
	@echo "$(YELLOW)PersistentVolumeClaims:$(NC)"
	@kubectl get pvc -n grud

wait: ## Wait for all resources to be ready
	@echo "$(BLUE)⏳ Waiting for databases...$(NC)"
	@kubectl wait --for=condition=Ready pod -l app=student-db -n grud --timeout=300s
	@kubectl wait --for=condition=Ready pod -l app=project-db -n grud --timeout=300s
	@echo "$(BLUE)⏳ Waiting for services...$(NC)"
	@kubectl wait --for=condition=Available deployment/student-service -n grud --timeout=300s
	@kubectl wait --for=condition=Available deployment/project-service -n grud --timeout=300s
	@kubectl wait --for=condition=Available deployment/admin-panel -n grud --timeout=300s
	@echo "$(GREEN)✅ All resources ready!$(NC)"

logs-db: ## Follow database logs
	@kubectl logs -n grud -l app=student-db -f --tail=50 &
	@kubectl logs -n grud -l app=project-db -f --tail=50

logs-admin: ## Follow admin-panel logs
	@kubectl logs -n grud -l app=admin-panel -f --tail=100

test: ## Test services
	@echo "$(BLUE)🧪 Testing services...$(NC)"
	@echo ""
	@echo "$(YELLOW)Testing student-service:$(NC)"
	@curl -s http://localhost:8080/api/students | jq '.' || echo "Student service not responding"
	@echo ""
	@echo "$(YELLOW)Testing project-service:$(NC)"
	@curl -s http://localhost:8081/api/projects | jq '.' || echo "Project service not responding"

port-forward-student: ## Port-forward student-service to localhost:9080
	@echo "$(BLUE)🔌 Port-forwarding student-service to localhost:9080...$(NC)"
	@kubectl port-forward -n grud service/student-service 9080:8080

port-forward-project: ## Port-forward project-service to localhost:9052
	@echo "$(BLUE)🔌 Port-forwarding project-service to localhost:9052...$(NC)"
	@kubectl port-forward -n grud service/project-service 9052:50052

port-forward-admin: ## Port-forward admin-panel to localhost:3000
	@echo "$(BLUE)🔌 Port-forwarding admin-panel to localhost:3000...$(NC)"
	@kubectl port-forward -n grud service/admin-panel 3000:80

port-forward-prometheus: ## Port-forward Prometheus to localhost:9090
	@echo "$(BLUE)🔌 Port-forwarding Prometheus to localhost:9090...$(NC)"
	@kubectl port-forward -n infra service/prometheus-kube-prometheus-prometheus 9090:9090

stop: ## Stop Kind cluster (without deleting)
	@echo "$(YELLOW)⏸️  Stopping Kind cluster...$(NC)"
	@docker stop $(KIND_CLUSTER_NAME)-control-plane $(KIND_CLUSTER_NAME)-worker $(KIND_CLUSTER_NAME)-worker2 $(KIND_CLUSTER_NAME)-worker3 2>/dev/null || true
	@echo "$(GREEN)✅ Cluster stopped$(NC)"

start: ## Start Kind cluster
	@echo "$(BLUE)▶️  Starting Kind cluster...$(NC)"
	@docker start $(KIND_CLUSTER_NAME)-control-plane $(KIND_CLUSTER_NAME)-worker $(KIND_CLUSTER_NAME)-worker2 $(KIND_CLUSTER_NAME)-worker3 2>/dev/null || true
	@echo "$(BLUE)⏳ Waiting for cluster to be ready...$(NC)"
	@kubectl wait --for=condition=Ready nodes --all --timeout=120s
	@echo "$(GREEN)✅ Cluster started and ready!$(NC)"

cleanup: ## Delete the entire cluster
	@../scripts/cleanup.sh

# Full workflow targets
all: setup deploy wait status ## Complete setup (cluster + deploy)

dev: setup deploy-dev wait status ## Complete development setup

prod: setup deploy-prod wait status ## Complete production setup
