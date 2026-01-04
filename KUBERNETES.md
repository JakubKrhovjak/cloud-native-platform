# Kubernetes Deployment

Deploy GRUD microservices on Kind cluster using **Kustomize + Ko** - declarative, GitOps-friendly approach.

## Quick Start (3 commands)

```bash
cd k8s

# 1. Create cluster + install operator + deploy everything
make all

# Or step by step:
make setup              # Create Kind cluster
make install-operator   # Install CloudNativePG
make deploy             # Deploy with Ko + Kustomize
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

## Prerequisites

```bash
# macOS
brew install kind kubectl kustomize

# Install Ko
go install github.com/google/ko@latest

# Verify
kind version
kubectl version --client
kustomize version
ko version
```

## Architecture

```
grud-cluster (Kind)
├── control-plane (1 node)
├── app node (workload=app) - Applications
│   ├── student-service (2 replicas)
│   └── project-service (2 replicas)
├── infra node (workload=infra) - Reserved
└── db node (workload=db) - Databases
    ├── student-db (3 PostgreSQL pods)
    └── project-db (3 PostgreSQL pods)
```

## Deployment Methods

### Method 1: Makefile (Recommended)

```bash
cd k8s

# Complete setup
make all                 # Cluster + operator + deploy + wait

# Or step by step
make setup              # Create Kind cluster
make install-operator   # Install CloudNativePG operator
make deploy             # Deploy everything with Kustomize

# Development (1 replica, 1 DB instance)
make deploy-dev

# Production (3 replicas, 3 DB instances)
make deploy-prod

# Check status
make status

# View logs
make logs
make logs-student
make logs-project

# Test services
make test

# Cleanup
make cleanup
```

### Method 2: Ko + Kustomize (Direct)

```bash
# Set environment
export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME=grud-cluster

# Deploy everything
ko resolve -f k8s/ | kubectl apply -f -

# With development overlay
ko resolve -f k8s/overlays/dev/ | kubectl apply -f -

# With production overlay
ko resolve -f k8s/overlays/prod/ | kubectl apply -f -
```

### Method 3: Pure Kustomize (without Ko)

If you already have Docker images:

```bash
# Build configuration
kustomize build k8s/ > deployment.yaml

# Apply
kubectl apply -k k8s/

# Or with overlay
kubectl apply -k k8s/overlays/dev/
```

## Kustomize Structure

```
k8s/
├── kustomization.yaml          # Base configuration
├── namespace.yaml
├── student-service/
│   ├── configmap.yaml
│   ├── deployment.yaml
│   └── service.yaml
├── project-service/
│   ├── configmap.yaml
│   ├── deployment.yaml
│   └── service.yaml
├── postgres/
│   ├── secrets.yaml
│   ├── student-db.yaml
│   └── project-db.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml  # 1 replica, 1 DB instance
    └── prod/
        └── kustomization.yaml  # 3 replicas, 3 DB instances
```

## Overlays

### Development Overlay

Reduces resources for local testing:

```bash
make deploy-dev
# or
ko resolve -f k8s/overlays/dev/ | kubectl apply -f -
```

Changes:
- Services: 1 replica each (instead of 2)
- Databases: 1 instance each (instead of 3)
- Lower resource limits

### Production Overlay

Full HA setup:

```bash
make deploy-prod
# or
ko resolve -f k8s/overlays/prod/ | kubectl apply -f -
```

Changes:
- Services: 3 replicas each
- Databases: 3 instances each
- Higher resource limits

## How Ko Works with Kustomize

Ko resolves `image: ko://...` references before applying:

```yaml
# In deployment.yaml:
image: ko://student-service/cmd/student-service

# Ko resolves to:
image: kind.local/student-service-abc123@sha256:xyz...
```

**Workflow:**
1. `ko resolve` scans manifests for `ko://` images
2. Builds Go binaries (CGO_ENABLED=0)
3. Creates container images (Chainguard base)
4. Pushes to Kind registry
5. Replaces `ko://` with actual image reference
6. Outputs resolved YAML for kubectl

## Development Workflow

### Update Code → Deploy

```bash
# After code changes
cd k8s
make deploy              # Rebuild and redeploy

# Or for specific service
ko apply -f student-service/deployment.yaml
```

Ko automatically:
- ✅ Detects code changes
- ✅ Rebuilds binary
- ✅ Creates new image
- ✅ Triggers rolling update

### Scale Services

```bash
# Scale up
kubectl scale deployment student-service -n grud --replicas=5

# Or patch via Kustomize
cat <<EOF > k8s/overlays/scaled/kustomization.yaml
resources:
  - ../../
patches:
  - target:
      kind: Deployment
      name: student-service
    patch: |-
      - op: replace
        path: /spec/replicas
        value: 5
EOF

ko resolve -f k8s/overlays/scaled/ | kubectl apply -f -
```

### View Resources

```bash
# All resources
make status

# Or manually
kubectl get all -n grud
kubectl get clusters -n grud

# Check pod placement on nodes
kubectl get pods -n grud -o wide
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
ko build --local ./student-service/cmd/student-service
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
cd k8s
make cleanup

# Or manually
kind delete cluster --name grud-cluster
```

## Scripts vs Kustomize

**Scripts are ONLY used for:**
1. ✅ `kind-setup.sh` - Creating Kind cluster (can't do with Kustomize)
2. ✅ `install-cnpg.sh` - Installing operator (external resource)
3. ✅ `cleanup.sh` - Deleting cluster

**Kustomize handles:**
- ✅ All application deployments
- ✅ Database provisioning
- ✅ Configuration management
- ✅ Environment overlays (dev/prod)

**Why no deployment scripts?**
- ❌ Scripts are imperative (order matters)
- ❌ Not GitOps-friendly
- ❌ Hard to version control state
- ✅ Kustomize is declarative
- ✅ Idempotent (safe to reapply)
- ✅ Perfect for GitOps (ArgoCD/Flux)

## GitOps Integration

Ready for ArgoCD/Flux:

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

## Best Practices Implemented

✅ **Declarative Infrastructure** - Everything in YAML
✅ **Ko Build** - No Docker, smaller images
✅ **Kustomize Overlays** - Easy environment management
✅ **Node Affinity** - Proper workload placement
✅ **High Availability** - Multiple replicas, DB replication
✅ **Security** - Non-root, read-only filesystem
✅ **Resource Limits** - CPU/memory management
✅ **Health Probes** - Liveness and readiness checks

## Complete Makefile Reference

```bash
make help              # Show all available commands

# Setup
make setup             # Create Kind cluster
make install-operator  # Install CloudNativePG

# Deployment
make deploy            # Deploy with base config
make deploy-dev        # Deploy with dev overlay
make deploy-prod       # Deploy with prod overlay

# Monitoring
make status            # Show cluster status
make logs              # Follow all service logs
make logs-student      # Student service logs
make logs-project      # Project service logs
make logs-db           # Database logs
make test              # Test services

# Utilities
make wait              # Wait for all resources
make cleanup           # Delete cluster

# Complete workflows
make all               # Setup + operator + deploy + wait
make dev               # Setup + operator + deploy-dev + wait
make prod              # Setup + operator + deploy-prod + wait
```

## Next Steps

1. **Add monitoring**: Deploy Prometheus/Grafana on infra node
2. **Add ingress**: Replace NodePort with Ingress controller
3. **Add TLS**: Use cert-manager for HTTPS
4. **GitOps**: Connect to ArgoCD or Flux
5. **Helm**: Convert to Helm charts for easier distribution

## References

- [Kustomize Documentation](https://kustomize.io/)
- [Ko Documentation](https://ko.build/)
- [CloudNativePG](https://cloudnative-pg.io/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
