.PHONY: help setup install-operator deploy deploy-dev deploy-prod status logs cleanup

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
	@kustomize build overlays/dev/ | ko resolve -f - | kubectl apply -f -
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
	@echo "$(GREEN)✅ All resources ready!$(NC)"

logs: ## Follow logs for all services
	@echo "$(BLUE)📜 Following service logs...$(NC)"
	@kubectl logs -n grud -l 'app in (student-service,project-service)' -f --tail=50

logs-student: ## Follow student-service logs
	@kubectl logs -n grud -l app=student-service -f --tail=100

logs-project: ## Follow project-service logs
	@kubectl logs -n grud -l app=project-service -f --tail=100

logs-db: ## Follow database logs
	@kubectl logs -n grud -l app=student-db -f --tail=50 &
	@kubectl logs -n grud -l app=project-db -f --tail=50

test: ## Test services
	@echo "$(BLUE)🧪 Testing services...$(NC)"
	@echo ""
	@echo "$(YELLOW)Testing student-service:$(NC)"
	@curl -s http://localhost:8080/api/students | jq '.' || echo "Student service not responding"
	@echo ""
	@echo "$(YELLOW)Testing project-service:$(NC)"
	@curl -s http://localhost:8081/api/projects | jq '.' || echo "Project service not responding"

cleanup: ## Delete the entire cluster
	@../scripts/cleanup.sh

# Full workflow targets
all: setup deploy wait status ## Complete setup (cluster + deploy)

dev: setup deploy-dev wait status ## Complete development setup

prod: setup deploy-prod wait status ## Complete production setup
