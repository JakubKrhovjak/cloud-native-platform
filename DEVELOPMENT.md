# Development Guide

## Running Services Locally in GoLand/IntelliJ IDEA

### Quick Setup

Both services support running from the project root directory. Config files are automatically discovered in `services/<service-name>/configs/`.

### Run Configuration Settings

When creating a run configuration in GoLand:

1. **Working Directory**: Set to project root
   ```
   /Users/jakubkrhovjak/GolandProjects/grud
   ```

2. **Package path** (for student-service):
   ```
   grud/services/student-service/cmd/server
   ```

3. **Package path** (for project-service):
   ```
   grud/services/project-service/cmd/server
   ```

4. **Environment Variables** (REQUIRED):
   ```
   ENV=local
   JWT_SECRET=your-secret-key-change-this-in-production
   ```

   Environment variables explained:
   - `ENV=local` - Loads `config.local.yaml`, disables secure-only cookies
   - `JWT_SECRET` - Secret key for JWT token signing (required for authentication)

### Important: Scale Down Kubernetes Services

When running services locally in GoLand, scale down Kubernetes deployments to avoid conflicts:

```bash
# Scale down student-service
kubectl scale deployment student-service -n grud --replicas=0

# Scale down project-service
kubectl scale deployment project-service -n grud --replicas=0
```

After finishing local development:

```bash
# Scale back up
kubectl scale deployment student-service -n grud --replicas=2
kubectl scale deployment project-service -n grud --replicas=2
```

### Database Setup

For local development, you have three options:

#### Option 1: Docker Compose (Recommended for IDE development)
```bash
docker-compose up postgres postgres_projects nats -d
```

This starts:
- Student database: `localhost:5439`
- Project database: `localhost:5440`
- NATS: `localhost:4222`

#### Option 2: Kind Kubernetes (Full Stack)
```bash
# Port forward databases from Kubernetes
kubectl port-forward -n grud svc/student-db-rw 5439:5432 &
kubectl port-forward -n grud svc/project-db-rw 5440:5432 &
kubectl port-forward -n infra svc/nats 4222:4222 &
```

#### Option 3: Hybrid (Kind databases + local services)
Best for testing with real infrastructure:
```bash
# Use Kind infrastructure
make kind/setup
make infra/deploy

# Port forward
kubectl port-forward -n grud svc/student-db-rw 5439:5432 &
kubectl port-forward -n grud svc/project-db-rw 5440:5432 &
kubectl port-forward -n infra svc/nats 4222:4222 &

# Run services locally in IDE
```

### Config Files Location

Config files are located in:
- `services/student-service/configs/config.local.yaml`
- `services/project-service/configs/config.local.yaml`

The application automatically searches these paths:
1. `./configs` (for Docker/K8s runtime)
2. `./services/<service-name>/configs` (for IDE from root)
3. `../configs` (for IDE from cmd/server)
4. `../../configs` (for other locations)

## Project Structure

```
grud/
├── services/                    # Microservices
│   ├── student-service/        # HTTP API, gRPC client, NATS producer
│   │   ├── cmd/server/         # Main entry point
│   │   ├── internal/           # Business logic (DDD)
│   │   │   ├── student/        # Student domain
│   │   │   ├── auth/           # JWT authentication
│   │   │   ├── config/         # Configuration
│   │   │   └── app/            # App bootstrap
│   │   └── configs/            # YAML config files
│   ├── project-service/        # HTTP API, gRPC server, NATS consumer
│   │   ├── cmd/server/
│   │   ├── internal/
│   │   │   ├── project/        # Project domain
│   │   │   ├── grpc/           # gRPC server
│   │   │   ├── nats/           # NATS consumer
│   │   │   └── app/
│   │   └── configs/
│   └── admin/                  # React admin panel
│       ├── src/
│       ├── public/
│       └── Dockerfile
├── api/                        # Protobuf definitions
│   ├── proto/                  # .proto files
│   └── gen/                    # Generated Go code
├── common/                     # Shared Go packages
│   ├── telemetry/             # OpenTelemetry setup
│   ├── metrics/               # Prometheus metrics
│   ├── logger/                # Structured logging
│   └── httputil/              # HTTP utilities
├── testing/                    # Test utilities
│   ├── testdb/                # PostgreSQL test containers
│   └── testnats/              # NATS test helpers
├── k8s/                       # Kubernetes manifests (Helm)
│   ├── grud/                  # Application Helm chart
│   └── infra/                 # Infrastructure stack
├── terraform/                 # GKE infrastructure
└── scripts/                   # Deployment scripts
```

## Testing

Run all tests:
```bash
make test              # Fast shared container tests (~5s)
```

See [TESTING.md](TESTING.md) for complete testing guide.

## Building Services

```bash
make build             # Build all services
make build-student     # Build student-service only
make build-project     # Build project-service only
make version           # Show version info
```

Binaries are output to `bin/` directory.

## Kubernetes Deployment

Deploy to local Kind cluster:
```bash
# Complete setup
make kind/setup        # Create Kind cluster
make infra/deploy      # Deploy infrastructure
make kind/deploy       # Deploy services with Helm
make kind/wait         # Wait for all resources
```

See [KUBERNETES.md](KUBERNETES.md) for complete deployment guide.

## Troubleshooting

### Config file not found
If you see `Config File "config.local" Not Found`:
1. Ensure working directory is set to project root in IDE
2. Check config file exists at `services/<service-name>/configs/config.local.yaml`
3. Verify `ENV=local` is set

### Database connection refused
Ensure databases are running:
```bash
# Check Docker containers
docker ps | grep postgres

# Check Kind pods
kubectl get pods -n grud

# Test connection
psql -h localhost -p 5439 -U postgres -d university
```

### Port already in use
Check if services are already running:
```bash
lsof -i :8080  # student-service
lsof -i :8081  # project-service HTTP
lsof -i :9090  # project-service gRPC
lsof -i :9080  # admin panel
```

### Authentication not working
If login fails with "invalid credentials":
1. Verify `JWT_SECRET` is set in IDE run configuration
2. Ensure `ENV=local` is set (disables secure-only cookies)
3. Check database has seed users (see migrations)
4. Test database connection: `lsof -i :5439`

### gRPC connection issues
If student-service can't reach project-service:
1. Ensure project-service is running on port 9090
2. Check gRPC server logs: `kubectl logs -n grud -l app=project-service | grep grpc`
3. Test gRPC: `grpcurl -plaintext localhost:9090 list`

### NATS connection issues
If NATS messages aren't delivered:
1. Ensure NATS is running: `docker ps | grep nats` or `kubectl get pods -n infra -l app=nats`
2. Check NATS URL in config: `nats://localhost:4222` (local) or `nats://nats.infra.svc.cluster.local:4222` (k8s)
3. View NATS logs: `kubectl logs -n infra -l app=nats`

## Admin Panel Development

```bash
cd services/admin

# Install dependencies
npm install

# Start dev server (hot reload)
npm run dev

# Build for production
npm run build
```

Access at: http://localhost:5173

## Generating Protobuf Code

When updating `.proto` files:

```bash
# Install protoc and plugins
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
./scripts/generate-proto.sh
```

Generated files are in `api/gen/`.
