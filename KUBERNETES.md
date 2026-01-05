# Kubernetes Deployment

Deploy GRUD microservices on Kind cluster using **Helm + Ko** - cloud-native, production-ready approach.

## Quick Start

```bash
# 1. Create Kind cluster with 4 nodes
make kind/setup

# 2. Deploy infrastructure (Prometheus, Grafana, NATS, Loki, Tempo)
make infra/deploy

# 3. Deploy application services with Helm
make kind/deploy

# 4. Wait for all resources
make kind/wait

# 5. Check status
make kind/status
```

**Access services:**
- Student Service: http://localhost:8080
- Admin Panel: http://localhost:9080
- Grafana: http://localhost:30300 (admin/admin)
- Prometheus: http://localhost:30090

## Why Helm?

✅ **Production-ready** - Industry standard for Kubernetes deployments
✅ **Templating** - DRY configuration with values files
✅ **Package management** - Version, rollback, upgrade
✅ **Environment overlays** - Easy dev/staging/prod configs via values files
✅ **Ko integration** - Build Go apps without Docker
✅ **Dependency management** - Infrastructure stack via Helm charts

## Prerequisites

```bash
# macOS
brew install kind kubectl helm ko

# Verify
kind version
kubectl version --client
helm version
ko version
```

## Architecture

```
grud-cluster (Kind) - 4 nodes
├── control-plane (1 node) - Kubernetes control plane
│
├── app node (workload=app) - Application services
│   ├── student-service (2 replicas) - HTTP API, gRPC client, NATS producer
│   ├── project-service (2 replicas) - HTTP API, gRPC server, NATS consumer
│   └── admin-panel (2 replicas) - React UI
│
├── infra node (workload=infra) - Infrastructure & observability
│   ├── prometheus - Metrics storage
│   ├── grafana - Visualization
│   ├── grafana-alloy - OpenTelemetry collector
│   ├── loki - Log aggregation
│   ├── tempo - Distributed tracing
│   └── nats - Message broker
│
└── db node (workload=db) - Databases (CloudNativePG)
    ├── student-db (3 PostgreSQL pods) - HA cluster
    └── project-db (3 PostgreSQL pods) - HA cluster
```

### Communication Flow

1. **Admin Panel** (HTTP) → **Student Service** (REST API)
2. **Student Service** (gRPC) → **Project Service** (GetProjectsByStudent)
3. **Student Service** (NATS) → **Project Service** (async events)
4. All services → **Grafana Alloy** (OTLP traces/metrics) → **Prometheus/Tempo**
5. All services → **Loki** (JSON logs)

## Deployment Methods

### Method 1: Makefile (Recommended)

All deployment commands from project root:

```bash
# Complete setup
make kind/setup          # Create Kind cluster (4 nodes)
make infra/deploy        # Deploy infrastructure stack
make kind/deploy         # Deploy services with Helm
make kind/wait           # Wait for all resources

# Check status
make kind/status         # Show all resources

# View logs
kubectl logs -n grud -l app=student-service -f
kubectl logs -n grud -l app=project-service -f
kubectl logs -n grud -l app=admin-panel -f

# Cleanup
make kind/stop           # Stop cluster (keeps data)
make kind/cleanup        # Delete cluster completely
```

### Method 2: Helm (Direct)

Deploy services manually with Helm:

```bash
# Build Go services with Ko
cd services/student-service
KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=grud-cluster \
  ko build --bare ./cmd/server > /tmp/student-image.txt

cd ../project-service
KO_DOCKER_REPO=kind.local KIND_CLUSTER_NAME=grud-cluster \
  ko build --bare ./cmd/server > /tmp/project-image.txt

# Build admin panel
cd ../admin
docker build -t admin-panel:latest .
kind load docker-image admin-panel:latest --name grud-cluster

# Deploy with Helm
cd ../../k8s
helm upgrade --install grud ./grud \
  -n grud --create-namespace \
  -f grud/values-kind.yaml \
  --set studentService.image.repository=$(cat /tmp/student-image.txt) \
  --set projectService.image.repository=$(cat /tmp/project-image.txt) \
  --wait
```

### Method 3: Infrastructure Only (for development)

Deploy only infrastructure, run services locally:

```bash
# Deploy infrastructure
make kind/setup
make infra/deploy

# Port-forward for local development
kubectl port-forward -n grud svc/student-db-rw 5439:5432 &
kubectl port-forward -n grud svc/project-db-rw 5440:5432 &
kubectl port-forward -n infra svc/nats 4222:4222 &
kubectl port-forward -n infra svc/alloy 4317:4317 &

# Run services locally in IDE
```

## Helm Chart Structure

```
k8s/grud/                           # Helm chart
├── Chart.yaml                      # Chart metadata
├── values-kind.yaml                # Kind configuration
├── values-gke.yaml                 # GKE configuration
└── templates/
    ├── namespace.yaml              # grud namespace
    ├── student-db.yaml             # CloudNativePG cluster
    ├── project-db.yaml             # CloudNativePG cluster
    ├── student-service/
    │   ├── deployment.yaml         # Student service
    │   ├── service.yaml            # ClusterIP + NodePort
    │   ├── configmap.yaml          # Configuration
    │   └── serviceaccount.yaml
    ├── project-service/
    │   ├── deployment.yaml         # Project service
    │   ├── service.yaml            # ClusterIP (HTTP + gRPC)
    │   ├── configmap.yaml
    │   └── serviceaccount.yaml
    └── admin-panel/
        ├── deployment.yaml         # React UI
        ├── service.yaml            # NodePort
        ├── configmap.yaml
        └── serviceaccount.yaml
```

## Values Files

### Kind Configuration (`values-kind.yaml`)

```yaml
studentService:
  replicaCount: 2
  image:
    repository: kind.local/student-service  # Set via Ko
    tag: latest
  resources:
    requests:
      memory: "256Mi"
      cpu: "200m"

projectService:
  replicaCount: 2
  grpc:
    enabled: true
    port: 9090

databases:
  student:
    instances: 3
    storage: 1Gi
  project:
    instances: 3
    storage: 1Gi
```

### GKE Configuration (`values-gke.yaml`)

```yaml
studentService:
  replicaCount: 3
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"

databases:
  student:
    instances: 3
    storage: 10Gi
    storageClass: pd-ssd
```

## How Ko Works with Helm

Ko builds Go services without Docker:

1. **Build**: `ko build --bare ./cmd/server` creates minimal container image
2. **Load to Kind**: Image is automatically loaded to Kind registry
3. **Helm Deploy**: Image reference is passed to Helm via `--set`

**Benefits:**
- No Dockerfile needed
- Minimal base image (Chainguard)
- Fast builds (< 10 seconds)
- Automatic multi-arch support

## Development Workflow

### Update Code → Deploy

```bash
# After code changes, redeploy services
make kind/deploy
```

This automatically:
1. Rebuilds Go services with Ko
2. Builds admin panel Docker image
3. Loads images to Kind
4. Upgrades Helm release
5. Triggers rolling update

### Scale Services

```bash
# Scale up via kubectl
kubectl scale deployment student-service -n grud --replicas=5

# Or update values file
helm upgrade grud k8s/grud \
  -n grud \
  -f k8s/grud/values-kind.yaml \
  --set studentService.replicaCount=5 \
  --wait
```

### Rollback Deployment

```bash
# View release history
helm history grud -n grud

# Rollback to previous version
helm rollback grud -n grud

# Rollback to specific revision
helm rollback grud 3 -n grud
```

### View Resources

```bash
# All resources via Makefile
make kind/status

# Or manually
kubectl get all -n grud
kubectl get all -n infra
kubectl get clusters -n grud  # CloudNativePG clusters

# Check pod placement on nodes
kubectl get pods -n grud -o wide
kubectl get pods -n infra -o wide
```

## Testing

```bash
# Automated tests
make test

# Manual tests
curl http://localhost:8080/api/students
curl http://localhost:8081/api/projects

# Create student
curl -X POST http://localhost:8080/api/students \
  -H "Content-Type: application/json" \
  -d '{"firstName":"John","lastName":"Doe","email":"john@test.com","major":"CS","year":2}'

# Create project
curl -X POST http://localhost:8081/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Project","description":"Testing K8s deployment"}'
```

## Monitoring

```bash
# Follow logs
make logs              # All services
make logs-student      # Student service only
make logs-project      # Project service only
make logs-db           # Database logs

# Resource usage
kubectl top nodes
kubectl top pods -n grud

# Events
kubectl get events -n grud --sort-by='.lastTimestamp'

# Describe
kubectl describe deployment student-service -n grud
kubectl describe cluster student-db -n grud
```

## Database Access

```bash
# Connect to student database
kubectl exec -it -n grud student-db-1 -- psql -U app university

# Connect to project database
kubectl exec -it -n grud project-db-1 -- psql -U app projects

# Port-forward for local access
kubectl port-forward -n grud svc/student-db-rw 5432:5432
psql -h localhost -U app -d university
```

## Troubleshooting

### Check Deployment Status

```bash
make status
```

### Pods Not Starting

```bash
kubectl describe pod -n grud <pod-name>

# Common issues:
# - Node taint/toleration mismatch
# - Image pull errors (check Ko build)
# - Resource constraints
# - Database not ready
```

### Ko Build Issues

```bash
# Clean Go workspace
go work sync
go mod tidy -C student-service
go mod tidy -C project-service

# Test Ko build
ko build --local ./student-service/cmd/server
```

### Database Connection Issues

```bash
# Verify databases are ready
kubectl get clusters -n grud

# Should show:
# NAME         INSTANCES   READY   STATUS
# student-db   3           3       Cluster in healthy state

# Check service endpoints
kubectl get endpoints -n grud

# Test DNS
kubectl exec -n grud deployment/student-service -- \
  nslookup student-db-rw.grud.svc.cluster.local
```

## Cleanup

```bash
# Stop cluster (keeps data)
make kind/stop

# Delete cluster completely
make kind/cleanup

# Or manually
kind delete cluster --name grud-cluster
helm uninstall grud -n grud
```

## Best Practices Implemented

✅ **Declarative Infrastructure** - Helm charts, version controlled
✅ **Ko Build** - No Dockerfile, minimal images (Chainguard)
✅ **Helm Values** - Environment-specific configurations
✅ **Node Affinity** - Proper workload placement (app/infra/db nodes)
✅ **High Availability** - Multiple replicas, PostgreSQL replication
✅ **Security** - Non-root containers, ServiceAccounts, NetworkPolicies
✅ **Resource Management** - CPU/memory requests and limits
✅ **Health Probes** - Liveness and readiness checks
✅ **Observability** - OpenTelemetry, Prometheus, Loki, Tempo
✅ **Rolling Updates** - Zero-downtime deployments

## Makefile Reference

All commands from project root:

### Setup
```bash
make kind/setup        # Create Kind cluster with 4 nodes
```

### Infrastructure
```bash
make infra/deploy      # Deploy all infrastructure (Helm)
make infra/status      # Show infrastructure status
make infra/cleanup     # Remove infrastructure
```

### Application Deployment
```bash
make kind/deploy       # Deploy services with Helm
make kind/wait         # Wait for all resources
make kind/status       # Show application status
make kind/stop         # Stop cluster (keeps data)
make kind/cleanup      # Delete cluster
```

### GKE Deployment
```bash
make gke/auth               # Authenticate with GCP
make gke/create-cluster     # Create GKE cluster
make gke/deploy             # Deploy services to GKE
make gke/status             # Show GKE status
make gke/delete-cluster     # Delete GKE cluster
```

### Utilities
```bash
make build             # Build all services locally
make test              # Run all tests
make version           # Show version info
```

## GKE Deployment

For production deployment on Google Kubernetes Engine:

```bash
# 1. Create GKE cluster with Terraform
cd terraform
terraform init
terraform apply

# 2. Connect to cluster
make gke/auth

# 3. Deploy infrastructure
make infra/deploy

# 4. Deploy services
make gke/deploy
```

See [terraform/README.md](terraform/README.md) for details.

## GitOps Integration

Ready for ArgoCD/FluxCD:

```yaml
# ArgoCD Application
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: grud
spec:
  source:
    repoURL: https://github.com/your-org/grud
    path: k8s/grud
    targetRevision: main
    helm:
      valueFiles:
        - values-gke.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: grud
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## References

- [Helm Documentation](https://helm.sh/docs/)
- [Ko Documentation](https://ko.build/)
- [CloudNativePG](https://cloudnative-pg.io/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [OpenTelemetry](https://opentelemetry.io/)
- [Grafana Alloy](https://grafana.com/docs/alloy/)
