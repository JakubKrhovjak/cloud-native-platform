# GRUD - University Management Platform

Cloud-native microservices platform for university management with observability, distributed tracing, and real-time messaging.

## Quick Start

### Local Development (Kind)
```bash
make kind/setup          # Create Kind cluster
make infra/deploy        # Prometheus, Grafana, NATS, Loki, Tempo
make kind/deploy         # Deploy all services

# Access services
open http://localhost:8080    # Student Service API
open http://localhost:9080    # Admin Panel
open http://localhost:30300   # Grafana
```

### GKE Production
```bash
make tf/apply            # Deploy infrastructure
make gke/connect         # Connect via Connect Gateway
make infra/deploy-gke    # Deploy observability stack
make gke/deploy          # Deploy application

# Access services
open https://grudapp.com/api          # API
open https://grafana.grudapp.com      # Grafana (IAP protected)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              GKE Cluster                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐        │
│  │ student-service │◄──►│ project-service │◄──►│      NATS       │        │
│  │   (HTTP/gRPC)   │    │   (gRPC/HTTP)   │    │   (messaging)   │        │
│  └────────┬────────┘    └────────┬────────┘    └─────────────────┘        │
│           │                      │                                          │
│           ▼                      ▼                                          │
│  ┌─────────────────────────────────────────┐                               │
│  │            Cloud SQL (PostgreSQL)        │                               │
│  │     university DB    │    projects DB    │                               │
│  └─────────────────────────────────────────┘                               │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Observability Stack                               │   │
│  │  Prometheus ◄─── Alloy ◄─── Services (OTLP)                         │   │
│  │      │            │                                                  │   │
│  │      ▼            ▼                                                  │   │
│  │   Grafana ──► Loki (logs) + Tempo (traces)                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Security Layer                                     │
│                                                                             │
│  Connect Gateway ──► kubectl/Terraform (IAM auth, no IP whitelist)         │
│  Cloud IAP ──► Grafana (Google authentication)                             │
│  GCE Ingress ──► HTTPS with managed certificates                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Services

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| **student-service** | 8080 | HTTP | Student management, JWT auth, NATS producer |
| **project-service** | 8081/9090 | HTTP/gRPC | Project management, NATS consumer |
| **admin** | 9080 | HTTP | React admin panel |

## Security Features

| Feature | Description |
|---------|-------------|
| **Connect Gateway** | Access cluster from anywhere without IP whitelisting |
| **Cloud IAP** | Google authentication for Grafana |
| **HTTPS** | Google-managed SSL certificates |
| **Private Nodes** | GKE nodes have no public IPs |
| **Workload Identity** | Secure GCP API access without keys |
| **Secret Manager** | Secrets stored in GSM, synced via ESO |

## Technologies

### Backend
- Go 1.25, PostgreSQL 16, gRPC + Protocol Buffers
- NATS (messaging), Bun ORM
- OpenTelemetry (traces + metrics)

### Frontend
- React 19, TypeScript, Material UI, Vite

### Infrastructure
- Kubernetes + Helm, Ko (container builder)
- Kind (local) + CloudNativePG
- GKE (production) + Cloud SQL + Terraform
- Google Secret Manager + External Secrets Operator

### Observability
- Prometheus, Grafana, Loki, Tempo, Grafana Alloy

## GKE Deployment

### Prerequisites

```bash
brew install google-cloud-sdk terraform kubectl helm ko

# Add gke-gcloud-auth-plugin to PATH
echo 'export PATH="/opt/homebrew/share/google-cloud-sdk/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Deploy

```bash
# 1. Authenticate
make gke/auth

# 2. Create terraform.tfvars
cd terraform
cat > terraform.tfvars << EOF
project_id = "your-project-id"
infra_node_count = 3
app_node_count = 1
db_password_student = "initial"
db_password_project = "initial"
connect_gateway_users = ["user:your-email@gmail.com"]
EOF

# 3. Deploy infrastructure
make tf/init && make tf/apply

# 4. Connect to cluster (via Connect Gateway)
make gke/connect

# 5. Deploy observability stack
make infra/deploy-gke

# 6. Deploy application
make gke/deploy
```

### Access

| Service | URL | Auth |
|---------|-----|------|
| API | https://grudapp.com/api | Public |
| Grafana | https://grafana.grudapp.com | Google IAP |
| kubectl | `make gke/connect` | Google IAM |

See [terraform/README.md](terraform/README.md) for detailed guide.

## Local Development (Kind)

```bash
# Create cluster
make kind/setup

# Deploy infrastructure
make infra/deploy

# Deploy application
make kind/deploy

# Check status
make kind/status
```

### Access Services

- Student Service API: http://localhost:8080
- Project Service API: http://localhost:8081
- Admin Panel: http://localhost:9080
- Grafana: http://localhost:30300 (admin/admin)

## Makefile Commands

### Build
```bash
make build           # Build all services
make test            # Run all tests
make version         # Show version info
```

### Kind Cluster
```bash
make kind/setup      # Create Kind cluster
make kind/deploy     # Deploy with Helm
make kind/status     # Show status
make kind/cleanup    # Delete cluster
```

### GKE Cluster
```bash
make gke/auth        # Authenticate with GCP
make gke/connect     # Connect via Connect Gateway
make gke/deploy      # Build and deploy
make gke/status      # Show status
make gke/ingress     # Show Ingress IPs
```

### Terraform
```bash
make tf/init         # Initialize
make tf/plan         # Plan changes
make tf/apply        # Apply
make tf/destroy      # Destroy
```

### Infrastructure
```bash
make infra/deploy       # Deploy to Kind
make infra/deploy-gke   # Deploy to GKE
make infra/status       # Show status
make infra/cleanup      # Remove
```

## API Documentation

### Authentication

```bash
POST /api/auth/login
{
  "email": "test@example.com",
  "password": "password123"
}
```

### Students (requires JWT)

```bash
GET    /api/students          # List all
GET    /api/students/{id}     # Get by ID
POST   /api/students          # Create
PUT    /api/students/{id}     # Update
DELETE /api/students/{id}     # Delete
```

### Projects

```bash
GET    /api/projects          # List all
GET    /api/projects/{id}     # Get by ID
POST   /api/projects          # Create
```

## Documentation

| Document | Description |
|----------|-------------|
| [terraform/README.md](terraform/README.md) | GKE infrastructure |
| [k8s/infra/README.md](k8s/infra/README.md) | Observability stack |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Local development |
| [TESTING.md](TESTING.md) | Testing guide |
| [SECRETS.md](SECRETS.md) | Secret management |
| [KUBERNETES.md](KUBERNETES.md) | Kubernetes deployment |

## Troubleshooting

### Connect Gateway not working

```bash
# Check membership
gcloud container fleet memberships list

# Ensure gke-gcloud-auth-plugin is in PATH
which gke-gcloud-auth-plugin
export PATH="/opt/homebrew/share/google-cloud-sdk/bin:$PATH"
```

### Grafana IAP error

```bash
# Check IAP secret
kubectl get secret -n infra grafana-iap-secret

# Check BackendConfig
kubectl get backendconfig -n infra grafana-backend-config
```

### Pods not starting

```bash
kubectl get pods -n grud
kubectl describe pod -n grud <pod-name>
kubectl logs -n grud <pod-name>
```

## License

This project is for educational purposes.
