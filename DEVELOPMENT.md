# Development Guide

## Running Services Locally in GoLand/IntelliJ IDEA

### Quick Setup

Both services now support running from the project root directory. The config loading has been updated to search in the correct paths after the restructuring to `services/` directory.

### Run Configuration Settings

When creating a run configuration in GoLand:

1. **Working Directory**: Set to project root
   ```
   /Users/jakubkrhovjak/GolandProjects/grud
   ```

2. **Package path** (for student-service):
   ```
   grud/services/student-service/cmd/student-service
   ```

3. **Package path** (for project-service):
   ```
   grud/services/project-service/cmd/project-service
   ```

4. **Environment Variables** (REQUIRED):
   - `ENV=local` (required for config loading)
   - `ENV=local` (required for local development, disables secure cookies)
   - `JWT_SECRET=your-secret-key` (required for authentication)

   Example for local development:
   ```
   ENV=local
   ENV=local
   JWT_SECRET=production-secret-change-this-in-real-deployment
   ```

### Important: Scale Down Kubernetes Services

When running services locally in GoLand, you must scale down the corresponding Kubernetes deployments to avoid database connection conflicts:

```bash
# Scale down student-service in Kubernetes
kubectl scale deployment student-service -n grud --replicas=0

# Scale down project-service in Kubernetes (if testing project-service)
kubectl scale deployment project-service -n grud --replicas=0
```

After finishing local development, scale back up:

```bash
# Scale back up
kubectl scale deployment student-service -n grud --replicas=2
kubectl scale deployment project-service -n grud --replicas=2
```

### Database Setup

For local development, you have two options:

#### Option 1: Docker Compose (Recommended)
```bash
docker-compose up postgres postgres_projects
```

This will start:
- Student database on `localhost:5439`
- Project database on `localhost:5440`

#### Option 2: Kind Kubernetes (Full Stack)
```bash
# Port forward the databases from Kubernetes
kubectl port-forward -n grud svc/student-db 5432:5432
kubectl port-forward -n grud svc/project-db 5440:5432
```

### Config Files Location

Config files are located in:
- `services/student-service/configs/config.local.yaml`
- `services/project-service/configs/config.qa.yaml`

The application automatically searches these paths:
1. `./configs` (for Docker/K8s runtime)
2. `./services/<service-name>/configs` (for IDE from root)
3. `./services/<service-name>/configs` (legacy path support)
4. `../configs` (for IDE from cmd/)
5. `../../configs` (for other locations)

## Project Structure

```
grud/
├── services/                    # Microservices
│   ├── student-service/
│   │   ├── cmd/student-service/  # Main entry point
│   │   ├── internal/             # Internal packages
│   │   └── configs/              # Service configs
│   └── project-service/
│       ├── cmd/project-service/
│       ├── internal/
│       └── configs/
├── api/                        # Shared protobuf definitions
├── common/                     # Shared utilities
├── testing/                    # Shared test utilities
└── k8s/                       # Kubernetes manifests
```

## Testing

Run all tests:
```bash
make test
```

Run service-specific tests:
```bash
make test-student
make test-project
```

## Kubernetes Deployment

Deploy to local Kind cluster:
```bash
export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME=grud-cluster
kustomize build k8s/overlays/dev | ko resolve -f - | kubectl apply -f -
```

## Troubleshooting

### Config file not found
If you see `Config File "config.local" Not Found`, ensure:
1. Working directory is set to project root in your IDE run configuration
2. Config file exists at `services/<service-name>/configs/config.local.yaml`
3. `ENV` is set correctly (defaults to `local`)

### Database connection refused
Ensure database is running:
```bash
# Check docker containers
docker ps | grep postgres

# Or check K8s pods
kubectl get pods -n grud
```

### Port already in use
Check if service is already running:
```bash
lsof -i :8080  # student-service
lsof -i :8081  # project-service
```

### Authentication not working (invalid credentials)
If login always returns "invalid email or password":
1. Ensure `JWT_SECRET` environment variable is set in your IDE run configuration
2. Ensure `ENV=local` is set (disables secure-only cookies)
3. Check that database contains users with hashed passwords
4. Verify port-forward to database is active: `lsof -i :5432`
