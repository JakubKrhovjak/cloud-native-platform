# Kubernetes Deployment

Deploy GRUD microservices using **Kustomize + Ko** - declarative, GitOps-friendly approach.

## Quick Start

```bash
cd k8s

# Complete setup
make all

# Or step by step:
make setup              # Create Kind cluster
make install-operator   # Install CloudNativePG
make deploy             # Deploy with Kustomize + Ko
make status             # Check deployment
```

**Access services:**
- Student Service: http://localhost:8080/api/students
- Project Service: http://localhost:8081/api/projects

## Why Kustomize?

✅ **Declarative** - GitOps-friendly, version controlled
✅ **No scripts** - Pure Kubernetes manifests
✅ **Overlays** - Easy dev/staging/prod configs
✅ **Ko integration** - Build Go apps without Docker
✅ **Idempotent** - Apply multiple times safely

## Architecture

```
grud-cluster (Kind)
├── control-plane (1 node)
├── app node (workload=app) - Applications
│   ├── student-service (2 replicas)
│   └── project-service (2 replicas)
├── infra node (workload=infra) - Reserved for monitoring
└── db node (workload=db) - Databases
    ├── student-db (3 PostgreSQL pods)
    └── project-db (3 PostgreSQL pods)
```

## Deployment Methods

### Method 1: Makefile (Recommended)

```bash
# Complete setup
make all                 # Cluster + operator + deploy + wait

# Development (1 replica)
make deploy-dev

# Production (3 replicas)
make deploy-prod

# Check status
make status

# View logs
make logs
make logs-student
make logs-project

# Cleanup
make cleanup
```

### Method 2: Ko + Kustomize (Direct)

```bash
# Set environment
export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME=grud-cluster

# Deploy everything
ko resolve -f . | kubectl apply -f -

# With overlays
ko resolve -f overlays/dev/ | kubectl apply -f -
ko resolve -f overlays/prod/ | kubectl apply -f -
```

### Method 3: Pure Kustomize

If you already have Docker images:

```bash
kubectl apply -k .
kubectl apply -k overlays/dev/
kubectl apply -k overlays/prod/
```

## Structure

```
k8s/
├── Makefile                    # Deployment automation
├── kustomization.yaml          # Base configuration
├── namespace.yaml
│
├── student-service/
│   ├── configmap.yaml
│   ├── deployment.yaml         # Ko image: ko://student-service/cmd/student-service
│   └── service.yaml
│
├── project-service/
│   ├── configmap.yaml
│   ├── deployment.yaml         # Ko image: ko://project-service/cmd/project-service
│   └── service.yaml
│
├── postgres/
│   ├── secrets.yaml
│   ├── student-db.yaml         # CloudNativePG cluster
│   └── project-db.yaml         # CloudNativePG cluster
│
└── overlays/
    ├── dev/                    # 1 replica, 1 DB instance
    └── prod/                   # 3 replicas, 3 DB instances
```

## How It Works

### Ko + Kustomize Integration

Ko resolves `image: ko://...` references:

```yaml
# In deployment.yaml
image: ko://student-service/cmd/student-service

# Ko builds and replaces with
image: kind.local/student-service-abc123@sha256:xyz...
```

**Workflow:**
1. `ko resolve` scans manifests
2. Builds Go binaries (CGO_ENABLED=0)
3. Creates images (Chainguard base)
4. Pushes to Kind registry
5. Replaces `ko://` with real image
6. Outputs YAML for kubectl

### Kustomize Overlays

Base configuration + environment patches:

**Development:**
- 1 replica per service
- 1 DB instance per cluster
- Lower resource limits

**Production:**
- 3 replicas per service
- 3 DB instances per cluster
- Higher resource limits

## Development Workflow

```bash
# After code changes
make deploy              # Rebuild and redeploy

# Or specific service
ko apply -f student-service/deployment.yaml
```

Ko automatically:
- ✅ Detects code changes
- ✅ Rebuilds binary
- ✅ Creates new image
- ✅ Triggers rolling update

## Makefile Commands

```bash
make help              # Show all commands

# Setup
make setup             # Create Kind cluster
make install-operator  # Install CloudNativePG

# Deployment
make deploy            # Base config (2 replicas)
make deploy-dev        # Dev overlay (1 replica)
make deploy-prod       # Prod overlay (3 replicas)

# Monitoring
make status            # Show cluster status
make logs              # All service logs
make test              # Test services

# Utilities
make wait              # Wait for resources
make cleanup           # Delete cluster

# Complete workflows
make all               # Setup + deploy + wait
make dev               # Setup + dev deploy
make prod              # Setup + prod deploy
```

## Testing

```bash
# Automated
make test

# Manual
curl http://localhost:8080/api/students
curl http://localhost:8081/api/projects

# Create student
curl -X POST http://localhost:8080/api/students \
  -H "Content-Type: application/json" \
  -d '{"firstName":"John","lastName":"Doe","email":"john@test.com","major":"CS","year":2}'

# Create project
curl -X POST http://localhost:8081/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","description":"Testing"}'
```

## Monitoring

```bash
# Logs
make logs              # All services
make logs-student      # Student only
make logs-project      # Project only
make logs-db           # Databases

# Resources
kubectl top nodes
kubectl top pods -n grud

# Status
make status
kubectl get all -n grud
kubectl get clusters -n grud
```

## Database Access

```bash
# Connect to database
kubectl exec -it -n grud student-db-1 -- psql -U app university
kubectl exec -it -n grud project-db-1 -- psql -U app projects

# Port-forward
kubectl port-forward -n grud svc/student-db-rw 5432:5432
psql -h localhost -U app -d university
```

## Troubleshooting

```bash
# Check status
make status

# Describe pod
kubectl describe pod -n grud <pod-name>

# Check databases
kubectl get clusters -n grud

# Ko build test
ko build --local ./student-service/cmd/student-service
```

## Scripts vs Kustomize

**Scripts ONLY for:**
- ✅ `kind-setup.sh` - Create cluster
- ✅ `install-cnpg.sh` - Install operator
- ✅ `cleanup.sh` - Delete cluster

**Kustomize for:**
- ✅ All deployments
- ✅ Configuration management
- ✅ Environment overlays

**Why?**
- ❌ Scripts are imperative
- ❌ Not GitOps-friendly
- ✅ Kustomize is declarative
- ✅ Idempotent
- ✅ Perfect for GitOps

## GitOps Ready

```yaml
# ArgoCD Application
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: grud
spec:
  source:
    repoURL: https://github.com/your-org/grud
    path: k8s/overlays/prod
    plugin:
      name: ko
  destination:
    server: https://kubernetes.default.svc
    namespace: grud
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Best Practices

✅ **Declarative Infrastructure** - Everything in YAML
✅ **Ko Build** - No Docker needed
✅ **Kustomize Overlays** - Environment management
✅ **Node Affinity** - Workload placement
✅ **High Availability** - Multiple replicas
✅ **Security** - Non-root containers
✅ **Resource Limits** - CPU/memory management
✅ **Health Probes** - Liveness/readiness checks

## Full Documentation

See [KUBERNETES.md](../KUBERNETES.md) for complete documentation including:
- Detailed architecture
- Advanced configuration
- Production considerations
- Troubleshooting guide
- GitOps integration

## References

- [Kustomize Docs](https://kustomize.io/)
- [Ko Docs](https://ko.build/)
- [CloudNativePG](https://cloudnative-pg.io/)
- [Kind Docs](https://kind.sigs.k8s.io/)
