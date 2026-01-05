# GRUD - University Management Platform

Cloud-native microservices platform for university management with observability, distributed tracing, and real-time messaging.

## Quick Start

### Local Development (Kind)
```bash
# Create Kind cluster with all infrastructure
make kind/setup
make infra/deploy        # Prometheus, Grafana, NATS, Loki, Tempo
make kind/deploy         # Deploy all services

# Access services
open http://localhost:8080    # Student Service API
open http://localhost:9080    # Admin Panel
open http://localhost:30300   # Grafana
```

### Run Tests
```bash
make test              # Fast tests (~5s)
```

See [TESTING.md](TESTING.md) for complete testing guide.

## Architecture

The platform consists of three microservices communicating via HTTP, gRPC, and NATS:

1. **student-service** - Student management (HTTP API, NATS producer)
2. **project-service** - Project management (gRPC API, NATS consumer)
3. **admin** - React admin panel (UI)

### Communication Patterns
- **HTTP REST**: Admin panel → Student service
- **gRPC**: Student service → Project service (synchronous)
- **NATS**: Student service → Project service (async events)

### Observability Stack
- **OpenTelemetry**: Distributed tracing with context propagation
- **Prometheus**: Metrics collection
- **Grafana**: Visualization and dashboards
- **Loki**: Log aggregation
- **Tempo**: Distributed trace storage
- **Grafana Alloy**: OpenTelemetry collector

## Technologies

### Backend
- Go 1.25
- PostgreSQL 16 (CloudNativePG operator)
- gRPC + Protocol Buffers
- NATS (messaging)
- Bun ORM

### Frontend
- React 19
- TypeScript
- Material UI
- Vite

### Infrastructure
- Kubernetes + Helm
- Ko (container builder)
- Kind (local development)
- GKE (production via Terraform)
- CloudNativePG (HA PostgreSQL)

### Observability
- OpenTelemetry (traces + metrics)
- Prometheus (metrics storage)
- Grafana (visualization)
- Loki (logs)
- Tempo (traces)
- Grafana Alloy (collector)

## Services

### Student Service
- **Port**: 8080 (HTTP)
- **Database**: `university`
- **API**: REST `/api/students`, `/api/auth`
- **Features**: JWT authentication, CRUD operations, NATS events
- **Communication**: Publishes events to NATS, calls project-service via gRPC

### Project Service
- **Port**: 8081 (HTTP), 9090 (gRPC)
- **Database**: `projects`
- **API**: REST `/api/projects`, gRPC ProjectService
- **Features**: gRPC server, NATS consumer, project management
- **Communication**: Consumes NATS events, serves gRPC requests

### Admin Panel
- **Port**: 9080 (HTTP)
- **Technology**: React 19 + TypeScript + Material UI
- **Features**: Student management UI, authentication, responsive design

## Project Structure

```
grud/
├── services/                    # Microservices
│   ├── student-service/        # Student management (HTTP + gRPC client + NATS producer)
│   │   ├── cmd/server/         # Main entry point
│   │   ├── internal/           # Business logic (DDD)
│   │   │   ├── student/        # Student domain
│   │   │   ├── auth/           # Authentication
│   │   │   ├── config/         # Configuration
│   │   │   └── app/            # App bootstrap
│   │   └── configs/            # Config files (YAML)
│   │
│   ├── project-service/        # Project management (HTTP + gRPC server + NATS consumer)
│   │   ├── cmd/server/
│   │   ├── internal/
│   │   │   ├── project/        # Project domain
│   │   │   ├── grpc/           # gRPC server
│   │   │   ├── nats/           # NATS consumer
│   │   │   └── app/
│   │   └── configs/
│   │
│   └── admin/                  # React admin panel
│       ├── src/
│       ├── public/
│       └── Dockerfile
│
├── api/                        # Shared Protobuf definitions
│   ├── proto/                  # .proto files
│   │   ├── project/
│   │   └── message/
│   └── gen/                    # Generated Go code
│       ├── project/
│       └── message/
│
├── common/                     # Shared Go packages
│   ├── telemetry/             # OpenTelemetry setup
│   ├── metrics/               # Prometheus metrics collectors
│   ├── logger/                # Structured logging
│   └── httputil/              # HTTP utilities
│
├── testing/                    # Test utilities
│   ├── testdb/                # PostgreSQL test containers
│   └── testnats/              # NATS test helpers
│
├── k8s/                       # Kubernetes manifests
│   ├── grud/                  # Helm chart
│   │   ├── templates/
│   │   ├── values-kind.yaml   # Kind config
│   │   └── values-gke.yaml    # GKE config
│   └── infra/                 # Infrastructure (Prometheus, Grafana, NATS, Loki)
│
├── terraform/                 # GKE infrastructure
│   ├── main.tf
│   ├── gke.tf
│   ├── database.tf
│   └── variables.tf
│
├── scripts/                   # Deployment scripts
│   ├── kind-setup.sh
│   └── generate-traffic.sh
│
├── Makefile                   # Build & deployment automation
├── go.work                    # Go workspace
└── docker-compose.yml         # Local development
```

## Installation and Setup

### Prerequisites
- **Docker Desktop** (for Kind)
- **kubectl** - Kubernetes CLI
- **Helm** - Package manager for Kubernetes
- **Kind** - Local Kubernetes cluster
- **Ko** - Go container builder (optional, for building images)
- **Go 1.25+** (for development)
- **Node.js 18+** (for admin panel development)

### Install Tools (macOS)

```bash
brew install kubectl helm kind ko
```

### Complete Setup (Kind + Infrastructure)

```bash
# 1. Create Kind cluster with 4 nodes
make kind/setup

# 2. Deploy infrastructure stack (Prometheus, Grafana, NATS, Loki, Tempo)
make infra/deploy

# 3. Deploy application services
make kind/deploy

# 4. Wait for all resources
make kind/wait

# 5. Verify deployment
make kind/status
```

### Access Services

After deployment:
- **Student Service API**: http://localhost:8080
- **Project Service API**: http://localhost:8081
- **Admin Panel**: http://localhost:9080
- **Grafana**: http://localhost:30300 (admin/admin)
- **Prometheus**: http://localhost:30090

### View Logs

```bash
# All services
kubectl logs -n grud -l app=student-service -f
kubectl logs -n grud -l app=project-service -f

# Grafana Alloy (collector)
kubectl logs -n infra -l app.kubernetes.io/name=alloy -f

# NATS
kubectl logs -n infra -l app=nats -f
```

### Cleanup

```bash
# Stop cluster (keeps data)
kind stop --name grud-cluster

# Delete cluster completely
make kind/cleanup
```

## API Documentation

### Authentication

All student endpoints require JWT authentication.

#### Login
```bash
POST http://localhost:8080/api/auth/login
Content-Type: application/json

{
  "email": "test@example.com",
  "password": "password123"
}

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "test@example.com",
    "firstName": "Test",
    "lastName": "User"
  }
}
```

The JWT token is also set as an HTTP-only cookie.

### Student Service API (HTTP)

#### Student Model
```json
{
  "id": 1,
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@university.com",
  "major": "Computer Science",
  "year": 2,
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
```

#### Endpoints

**Create Student** (requires authentication)
```bash
POST http://localhost:8080/api/students
Authorization: Bearer <token>
Content-Type: application/json

{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@university.com",
  "major": "Computer Science",
  "year": 2
}
```

**Get All Students** (requires authentication)
```bash
GET http://localhost:8080/api/students
Authorization: Bearer <token>
```

**Get Student by ID** (requires authentication)
```bash
GET http://localhost:8080/api/students/{id}
Authorization: Bearer <token>
```

**Update Student** (requires authentication)
```bash
PUT http://localhost:8080/api/students/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@university.com",
  "major": "Software Engineering",
  "year": 3
}
```

**Delete Student** (requires authentication)
```bash
DELETE http://localhost:8080/api/students/{id}
Authorization: Bearer <token>
```

### Project Service API (HTTP + gRPC)

#### Project Model
```json
{
  "id": 1,
  "name": "Web Application",
  "description": "Modern web app with Go backend",
  "studentId": 42,
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
```

#### HTTP Endpoints

**Get All Projects**
```bash
GET http://localhost:8081/api/projects
```

**Get Project by ID**
```bash
GET http://localhost:8081/api/projects/{id}
```

**Create Project**
```bash
POST http://localhost:8081/api/projects
Content-Type: application/json

{
  "name": "Web Application",
  "description": "Modern web app with Go backend",
  "studentId": 42
}
```

#### gRPC API

Project service exposes a gRPC server on port 9090.

**GetProjectsByStudent** - Get all projects for a student
```protobuf
service ProjectService {
  rpc GetProjectsByStudent(GetProjectsByStudentRequest) returns (GetProjectsByStudentResponse);
}

message GetProjectsByStudentRequest {
  int64 student_id = 1;
}

message GetProjectsByStudentResponse {
  repeated Project projects = 1;
}
```

Student service calls this endpoint when fetching student details.

#### NATS Events

Project service consumes events from NATS:

**Topic**: `student.viewed`
**Payload**:
```json
{
  "student_id": 42,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

This event is published by student-service when a student is viewed.

## Local Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed development guide including:
- Running services in GoLand/IntelliJ IDEA
- Environment configuration
- Database setup
- Debugging

### Quick Start (GoLand/IntelliJ)

1. Set working directory to project root
2. Set environment variables:
   ```
   ENV=local
   JWT_SECRET=your-secret-key
   ```
3. Start databases:
   ```bash
   docker-compose up postgres postgres_projects -d
   ```
4. Run service from IDE

### Go Workspace

The project uses Go workspace:

```bash
# Sync workspace
go work sync

# Build services
make build              # Build all
make build-student      # Student service only
make build-project      # Project service only
```

## Observability

### Distributed Tracing

The platform uses OpenTelemetry for distributed tracing:
- Automatic trace propagation across HTTP, gRPC, and NATS
- Traces are exported to Tempo via Grafana Alloy
- View traces in Grafana (http://localhost:30300)

Example trace flow:
1. HTTP request to student-service
2. gRPC call to project-service
3. NATS message published
4. NATS message consumed by project-service

All operations share the same `trace_id` for correlation.

### Metrics

Services expose metrics in OpenTelemetry format:
- **HTTP metrics**: Request rate, latency (p50/p95/p99), errors
- **gRPC metrics**: RPC duration, request/response sizes
- **Database metrics**: Connection pool stats, query performance
- **NATS metrics**: Message publish/consume rate, processing time
- **Runtime metrics**: Goroutines, memory, GC

View in Prometheus: http://localhost:30090
View dashboards in Grafana: http://localhost:30300

### Logs

Structured JSON logs are collected by Loki:
- All logs include `trace_id` for correlation
- View logs in Grafana Explore
- Filter by service, level, or trace ID

## Deployment Options

### 1. Local Development (Kind)
```bash
make kind/setup      # Create Kind cluster
make infra/deploy    # Deploy infrastructure
make kind/deploy     # Deploy services
```

### 2. Google Kubernetes Engine (GKE)
```bash
cd terraform
terraform init
terraform apply      # Create GKE cluster + Cloud SQL

cd ..
make gke/deploy      # Deploy services with Helm
```

See [terraform/README.md](terraform/README.md) for GKE deployment guide.

### 3. Docker Compose (Legacy)
```bash
docker-compose up -d
```

## Makefile Commands

### Build
```bash
make build           # Build all services
make build-student   # Build student-service
make build-project   # Build project-service
make version         # Show version info
```

### Test
```bash
make test            # Run all tests (~5s)
```

### Kind Cluster
```bash
make kind/setup      # Create Kind cluster
make kind/deploy     # Deploy with Helm
make kind/status     # Show status
make kind/wait       # Wait for ready
make kind/stop       # Stop cluster
make kind/cleanup    # Delete cluster
```

### Infrastructure
```bash
make infra/deploy    # Deploy all infrastructure
make infra/status    # Show infrastructure status
make infra/cleanup   # Remove infrastructure
```

### GKE
```bash
make gke/create-cluster  # Create GKE cluster
make gke/deploy          # Deploy services
make gke/status          # Show status
make gke/delete-cluster  # Delete cluster
```

## Database Access

### In Kubernetes (Kind)
```bash
# Student database
kubectl exec -it -n grud student-db-1 -- psql -U app university

# Project database
kubectl exec -it -n grud project-db-1 -- psql -U app projects

# Port-forward for local access
kubectl port-forward -n grud svc/student-db-rw 5432:5432
psql -h localhost -U app -d university
```

### Docker Compose
```bash
docker exec -it university_db psql -U postgres -d university
docker exec -it projects_db psql -U postgres -d projects
```

## Architecture Best Practices

1. **Microservices** - Independent deployment, fault isolation, scalability
2. **DDD Structure** - Model, Repository, Service, HTTP/gRPC layers
3. **Dependency Injection** - Constructor-based injection
4. **Interface Segregation** - Each layer defines interfaces
5. **Error Handling** - Wrapped errors with context
6. **Observability** - Distributed tracing, metrics, structured logs
7. **Security** - JWT authentication, HTTP-only cookies
8. **Cloud Native** - Kubernetes, Helm, CloudNativePG
9. **GitOps Ready** - Declarative infrastructure

## Troubleshooting

### Services not starting
```bash
# Check pod status
kubectl get pods -n grud

# Check logs
kubectl logs -n grud -l app=student-service --tail=50

# Describe pod
kubectl describe pod -n grud <pod-name>
```

### Database connection issues
```bash
# Check database status
kubectl get clusters -n grud

# Check database logs
kubectl logs -n grud student-db-1
```

### Traces not visible in Grafana
```bash
# Check Alloy logs
kubectl logs -n infra -l app.kubernetes.io/name=alloy

# Check service OTEL configuration
kubectl logs -n grud -l app=student-service | grep -i otel
```

### NATS messages not delivered
```bash
# Check NATS logs
kubectl logs -n infra -l app=nats

# Check publisher
kubectl logs -n grud -l app=student-service | grep -i nats

# Check consumer
kubectl logs -n grud -l app=project-service | grep -i nats
```

## Testing

See [TESTING.md](TESTING.md) for complete testing guide.

### Quick Start

```bash
make test              # Run all tests (~5s)
```

### Test Architecture

The project uses shared PostgreSQL and NATS containers for fast testing:
- **Shared Container Tests**: ~5s for all tests
- **Integration Tests**: ~40s with isolated containers
- **8× speedup** with shared containers!

Tests include:
- Unit tests for business logic
- Integration tests with real PostgreSQL
- NATS messaging tests
- gRPC client/server tests

## Documentation

### Main Documentation
- [README.md](README.md) - This file
- [DEVELOPMENT.md](DEVELOPMENT.md) - Local development guide
- [KUBERNETES.md](KUBERNETES.md) - Kubernetes deployment guide
- [TESTING.md](TESTING.md) - Complete testing guide
- [ENV_SETUP.md](ENV_SETUP.md) - Environment configuration

### Service Documentation
- [services/student-service/README.md](services/student-service/README.md) - Student service
- [services/project-service/README.md](services/project-service/README.md) - Project service (TODO)
- [services/admin/README.md](services/admin/README.md) - Admin panel

### Infrastructure Documentation
- [k8s/README.md](k8s/README.md) - Kubernetes manifests
- [k8s/infra/README.md](k8s/infra/README.md) - Infrastructure stack
- [common/metrics/README.md](common/metrics/README.md) - Metrics package
- [terraform/README.md](terraform/README.md) - GKE deployment (TODO)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is for educational purposes.
