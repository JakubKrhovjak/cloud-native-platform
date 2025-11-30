# Kubernetes Deployment Guide

Complete guide for deploying GRUD microservices on Kind (Kubernetes in Docker) cluster.

## Architecture

### Cluster Layout

```
grud-cluster (Kind)
├── control-plane (1 node)
│   └── API server, scheduler, controller-manager
│
├── app node (worker 1) - taint: workload=app
│   ├── student-service (2 replicas)
│   └── project-service (2 replicas)
│
├── infra node (worker 2) - taint: workload=infra
│   └── Reserved for monitoring, logging, etc.
│
└── db node (worker 3) - taint: workload=db
    ├── student-db (PostgreSQL cluster - 3 instances)
    └── project-db (PostgreSQL cluster - 3 instances)
```

### Key Features

✅ **Node Affinity & Taints**
- Applications run only on `app` nodes
- Databases run only on `db` nodes
- Infrastructure services reserved on `infra` nodes

✅ **Ko Build Tool**
- No Docker required for building Go applications
- Direct Kubernetes integration
- Smaller images with Chainguard base

✅ **CloudNativePG Operator**
- Production-grade PostgreSQL management
- High availability with automatic failover
- Built-in backup and restore
- Connection pooling

✅ **High Availability**
- 2 replicas per application service
- 3 PostgreSQL instances per database cluster
- Pod anti-affinity rules

## Prerequisites

Install required tools:

```bash
# macOS
brew install kind kubectl

# Install ko (Go image builder)
go install github.com/google/ko@latest

# Verify installations
kind version
kubectl version --client
ko version
```

## Quick Start

Deploy everything in 3 commands:

```bash
# 1. Create Kind cluster (3 worker nodes with taints)
./scripts/kind-setup.sh

# 2. Install PostgreSQL operator
./scripts/install-cnpg.sh

# 3. Deploy databases
./scripts/deploy-databases.sh

# 4. Build and deploy services with Ko
./scripts/deploy-services.sh
```

Total setup time: ~5 minutes

## Step-by-Step Deployment

### 1. Create Kind Cluster

```bash
./scripts/kind-setup.sh
```

This creates a cluster with:
- 1 control-plane node
- 3 worker nodes (app, infra, db)
- Port mappings: 8080, 8081

Verify cluster:
```bash
kubectl get nodes -o wide
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
```

Expected output:
```
NAME                         TAINTS
grud-cluster-control-plane   [map[effect:NoSchedule key:node-role.kubernetes.io/control-plane]]
grud-app-node               [map[effect:NoSchedule key:workload value:app]]
grud-infra-node             [map[effect:NoSchedule key:workload value:infra]]
grud-db-node                [map[effect:NoSchedule key:workload value:db]]
```

### 2. Install CloudNativePG Operator

```bash
./scripts/install-cnpg.sh
```

This installs the PostgreSQL operator that manages database clusters.

Verify operator:
```bash
kubectl get pods -n cnpg-system
```

### 3. Deploy PostgreSQL Databases

```bash
./scripts/deploy-databases.sh
```

This creates:
- `grud` namespace
- Database secrets
- 2 PostgreSQL clusters (student-db, project-db)
- Each cluster has 3 instances (1 primary + 2 replicas)

Wait for databases (2-3 minutes):
```bash
# Watch database pods starting
kubectl get pods -n grud -w
```

Verify databases:
```bash
kubectl get clusters -n grud
kubectl get pods -n grud -l postgresql
```

Expected output:
```
NAME         AGE   INSTANCES   READY   STATUS
student-db   2m    3           3       Cluster in healthy state
project-db   2m    3           3       Cluster in healthy state
```

### 4. Build and Deploy Services

```bash
./scripts/deploy-services.sh
```

This:
- Builds Go applications with Ko (no Docker!)
- Creates container images
- Deploys to Kubernetes
- Waits for readiness

Verify deployments:
```bash
kubectl get deployments -n grud
kubectl get pods -n grud -l 'app in (student-service,project-service)'
kubectl get services -n grud
```

## Access Services

Services are exposed via NodePort:

```bash
# Student Service
curl http://localhost:8080/api/students

# Project Service
curl http://localhost:8081/api/projects

# Create a student
curl -X POST http://localhost:8080/api/students \
  -H "Content-Type: application/json" \
  -d '{"firstName":"John","lastName":"Doe","email":"john@example.com","major":"CS","year":2}'

# Create a project
curl -X POST http://localhost:8081/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project","description":"Test project"}'
```

## Verify Node Placement

Check that pods are scheduled on correct nodes:

```bash
# Applications should be on app node
kubectl get pods -n grud -o wide -l 'app in (student-service,project-service)'

# Databases should be on db node
kubectl get pods -n grud -o wide -l postgresql
```

## Monitoring & Debugging

### View Logs

```bash
# Student service logs
kubectl logs -n grud -l app=student-service -f

# Project service logs
kubectl logs -n grud -l app=project-service -f

# Database logs
kubectl logs -n grud -l postgresql=student-db -f
```

### Database Access

Connect to PostgreSQL:

```bash
# Student database
kubectl exec -it -n grud student-db-1 -- psql -U app university

# Project database
kubectl exec -it -n grud project-db-1 -- psql -U app projects
```

### Check Resource Usage

```bash
kubectl top nodes
kubectl top pods -n grud
```

### Describe Resources

```bash
# Check pod scheduling
kubectl describe pod -n grud <pod-name>

# Check database cluster status
kubectl describe cluster -n grud student-db
```

## Development Workflow

### Update Application Code

After code changes:

```bash
# Rebuild and redeploy student-service
cd student-service
ko apply -f ../k8s/student-service/deployment.yaml
cd ..

# Rebuild and redeploy project-service
cd project-service
ko apply -f ../k8s/project-service/deployment.yaml
cd ..
```

Ko automatically:
1. Builds the Go binary
2. Creates a new container image
3. Pushes to Kind's registry
4. Updates the deployment
5. Triggers rolling update

### Scale Services

```bash
# Scale student-service to 3 replicas
kubectl scale deployment student-service -n grud --replicas=3

# Scale project-service to 5 replicas
kubectl scale deployment project-service -n grud --replicas=5
```

### Restart Services

```bash
# Rolling restart
kubectl rollout restart deployment student-service -n grud
kubectl rollout restart deployment project-service -n grud
```

## Cleanup

Delete the entire cluster:

```bash
./scripts/cleanup.sh
```

Or manually:
```bash
kind delete cluster --name grud-cluster
```

## Configuration Files

```
k8s/
├── kind-config.yaml              # Kind cluster configuration
├── namespace.yaml                # Namespace definition
│
├── student-service/
│   ├── deployment.yaml           # App deployment with node affinity
│   ├── service.yaml              # NodePort service (port 30080)
│   └── configmap.yaml            # Application configuration
│
├── project-service/
│   ├── deployment.yaml           # App deployment with node affinity
│   ├── service.yaml              # NodePort service (port 30081)
│   └── configmap.yaml            # Application configuration
│
└── postgres/
    ├── student-db.yaml           # PostgreSQL cluster (3 instances)
    ├── project-db.yaml           # PostgreSQL cluster (3 instances)
    └── secrets.yaml              # Database credentials
```

## Ko Configuration

Ko configuration in `.ko.yaml`:

```yaml
defaultBaseImage: cgr.dev/chainguard/static:latest

builds:
  - id: student-service
    main: ./student-service/cmd/server
    env: [CGO_ENABLED=0, GOOS=linux, GOARCH=amd64]

  - id: project-service
    main: ./project-service/cmd/server
    env: [CGO_ENABLED=0, GOOS=linux, GOARCH=amd64]
```

## Best Practices Implemented

### Security
✅ Non-root containers (UID 65532)
✅ Read-only root filesystem
✅ Dropped all capabilities
✅ Security contexts enforced
✅ Secrets for database credentials

### High Availability
✅ Multiple replicas per service
✅ Pod anti-affinity rules
✅ PostgreSQL with automatic failover
✅ Liveness and readiness probes

### Resource Management
✅ Resource requests and limits
✅ Node affinity and taints
✅ Proper scheduling constraints
✅ Storage classes for persistence

### Observability
✅ Structured logging (JSON)
✅ Health check endpoints
✅ PostgreSQL monitoring enabled
✅ Labels for service discovery

## Troubleshooting

### Pods Not Starting

Check node placement:
```bash
kubectl describe pod -n grud <pod-name>
```

Look for:
- Tolerations matching node taints
- Node affinity requirements
- Resource availability

### Database Connection Issues

Verify database is ready:
```bash
kubectl get clusters -n grud
kubectl logs -n grud student-db-1
```

Check service DNS:
```bash
kubectl exec -n grud deployment/student-service -- nslookup student-db-rw.grud.svc.cluster.local
```

### Ko Build Failures

Ensure Go workspace is clean:
```bash
go work sync
go mod tidy -C student-service
go mod tidy -C project-service
```

### Port Already in Use

Change NodePort in service.yaml or stop conflicting service:
```bash
lsof -i :8080
lsof -i :8081
```

## Production Considerations

For production deployments, consider:

1. **Ingress Controller**: Replace NodePort with Ingress
2. **TLS Certificates**: Use cert-manager for HTTPS
3. **External Secrets**: Use sealed-secrets or external-secrets-operator
4. **Database Backups**: Configure S3/Azure/GCS for CloudNativePG backups
5. **Monitoring**: Add Prometheus and Grafana
6. **Logging**: Deploy EFK/ELK stack on infra node
7. **Service Mesh**: Consider Istio or Linkerd
8. **GitOps**: Use ArgoCD or Flux for deployments
9. **Resource Limits**: Tune based on load testing
10. **Network Policies**: Add network segmentation

## References

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Ko Documentation](https://ko.build/)
- [CloudNativePG Documentation](https://cloudnative-pg.io/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
