# Development Guide

## Local Development Setup

### Prerequisites

```bash
brew install go docker kubectl helm kind
```

### Running Services in GoLand/IntelliJ IDEA

#### Run Configuration

1. **Working Directory**: Project root (where `go.work` is located)

2. **Package path**:
   - student-service: `cloud-native-platform/services/student-service/cmd/student-service`
   - project-service: `cloud-native-platform/services/project-service/cmd/project-service`

3. **Environment Variables**:
   ```
   ENV=local
   JWT_SECRET=dev-secret-key-change-in-production
   ```

### Database Setup

#### Option 1: Docker Compose (Recommended)

```bash
docker-compose up -d postgres postgres_projects
```

Databases:
- Student DB: `localhost:5433`
- Project DB: `localhost:5440`

#### Option 2: Kind Kubernetes

```bash
make kind/setup
make kind/deploy

# Port forward databases
kubectl port-forward -n apps svc/student-db 5433:5432 &
kubectl port-forward -n apps svc/project-db 5440:5432 &
```

### Scale Down K8s When Running Locally

When running services locally, scale down K8s deployments to avoid conflicts:

```bash
# Scale down
kubectl scale deployment student-service -n apps --replicas=0
kubectl scale deployment project-service -n apps --replicas=0

# Scale back up when done
kubectl scale deployment student-service -n apps --replicas=2
kubectl scale deployment project-service -n apps --replicas=2
```

## Project Structure

```
cloud-native-platform/
├── services/
│   ├── student-service/
│   │   ├── cmd/student-service/    # Entry point
│   │   ├── internal/               # Business logic
│   │   └── configs/                # YAML configs
│   ├── project-service/
│   │   ├── cmd/project-service/
│   │   ├── internal/
│   │   └── configs/
│   └── admin/                      # React frontend
├── common/                         # Shared Go packages
├── api/                            # Protobuf definitions
├── k8s/                            # Helm charts & manifests
├── terraform/                      # GKE infrastructure
└── testing/                        # Test utilities
```

## Config Files

Config files per environment:
- `services/student-service/configs/config.local.yaml`
- `services/project-service/configs/config.local.yaml`

The application searches these paths:
1. `/configs` (Kubernetes mount)
2. `./services/<service>/configs` (IDE from root)
3. `../configs` (IDE from cmd/)

## Testing

```bash
# All tests
make test

# With coverage
go test ./... -cover

# Specific service
cd services/student-service && go test ./...
```

## Building

```bash
# Build all
make build

# Build specific service
go build ./services/student-service/cmd/student-service

# Build container images (Ko)
export KO_DOCKER_REPO=kind.local
ko build ./services/student-service/cmd/student-service
```

## Kubernetes Deployment

### Local (Kind)

```bash
make kind/setup      # Create cluster
make infra/deploy    # Deploy observability
make kind/deploy     # Deploy services
```

### GKE

```bash
make gke/connect     # Connect to cluster
make gke/deploy      # Build & deploy
```

## Troubleshooting

### Config file not found

```
Config File "config.local" Not Found
```

Fix:
1. Set working directory to project root
2. Ensure `ENV=local` is set
3. Check config file exists at `services/<service>/configs/config.local.yaml`

### Database connection refused

```bash
# Check Docker
docker ps | grep postgres

# Check K8s
kubectl get pods -n apps
```

### Port already in use

```bash
lsof -i :8080  # student-service
lsof -i :9090  # project-service gRPC
```

### Authentication not working

1. Ensure `JWT_SECRET` is set
2. Ensure `ENV=local` (disables secure cookies)
3. Check database has users with bcrypt hashed passwords
