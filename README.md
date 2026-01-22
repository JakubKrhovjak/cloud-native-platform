# Cloud Native Platform

**Showcase project** demonstrating production-ready infrastructure for a mid-sized company (10-50 engineers, multiple teams).

This is intentionally "enterprise-grade" - the goal was to build a reference architecture that handles real-world concerns: observability, security, CI/CD, infrastructure as code, and operational excellence. Not every project needs this complexity, but when you do, this is how it's done.

## Why These Technologies?

### Backend Stack

| Technology | Why? |
|------------|------|
| **Go + Gin** | High-performance HTTP framework with minimal overhead. Gin provides fast routing, middleware support, JSON validation, and error handling out of the box. Battle-tested in production at scale. |
| **Bun ORM** | Modern, fast SQL-first ORM for Go. Generates efficient queries, supports PostgreSQL natively, and provides type-safe database operations without the complexity of GORM. |
| **PostgreSQL** | Battle-tested relational database. ACID compliance, JSON support, excellent performance. Cloud SQL provides managed HA with automatic backups. |
| **JWT Authentication** | Stateless authentication using access + refresh tokens. HttpOnly cookies prevent XSS, bcrypt for password hashing, configurable expiration. |
| **gRPC + Protocol Buffers** | Efficient inter-service communication with type safety. Smaller payloads and faster serialization than REST/JSON. |
| **NATS** | Lightweight, high-performance messaging. Simpler than Kafka for event-driven architecture, perfect for real-time notifications. |

### Observability Stack

| Technology | Why? |
|------------|------|
| **OpenTelemetry** | Vendor-neutral observability standard. Single SDK for traces, metrics, and logs. Future-proof - switch backends without code changes. |
| **Grafana** | Unified visualization for all telemetry data. Dashboards for metrics (Prometheus), logs (Loki), and traces (Tempo) in one place. |
| **Prometheus** | Industry standard for metrics. Pull-based model works well with Kubernetes, extensive ecosystem of exporters and alerts. |
| **Loki + Tempo** | Grafana-native log and trace storage. Cost-effective (log labels, not full-text indexing), seamless integration with Grafana dashboards. |

### Infrastructure

| Technology | Why? |
|------------|------|
| **Terraform** | Infrastructure as code for reproducible deployments. State management, dependency graph, extensive GCP provider support. |
| **GKE + Gateway API** | Managed Kubernetes with Google's next-gen ingress. Native HTTPS, Cloud Armor integration, global load balancing. |
| **Cloud Armor** | WAF and DDoS protection at the edge. Rate limiting, geo-blocking, OWASP rule sets - all managed by Google. |
| **Workload Identity** | Secure GCP API access without service account keys. Pods authenticate using Kubernetes service accounts mapped to GCP IAM. |

## When NOT to Use This Architecture

This is a **showcase for mid-sized companies** (10-50 engineers). It's **overkill** if:

| Scenario | Better Alternative |
|----------|-------------------|
| **MVP / Prototype** | Single Go binary + SQLite, deploy to Cloud Run |
| **< 1000 DAU** | Monolith on a single VM, PostgreSQL on the same box |
| **Solo developer** | Skip Kubernetes, use Cloud Run or Railway |
| **No ops team** | Use managed PaaS (Render, Fly.io, Heroku) |
| **Budget < $100/mo** | Single container, managed database |
| **Simple CRUD app** | Skip microservices, gRPC, and message queues |

**Signs you DON'T need this:**
- You can count your services on one hand
- Your team doesn't have dedicated DevOps/SRE
- You're not hitting scaling limits with a monolith
- You don't need independent deployment of services
- "Because microservices" is your only reason

**This architecture makes sense when:**
- Multiple teams need to deploy independently
- Different services have different scaling requirements
- You need fault isolation between components
- You have dedicated platform/infrastructure engineers
- Monthly cloud spend justifies the operational complexity

> **For most projects, Cloud Run + Cloud SQL is the right answer.** This architecture is for when you've outgrown that.

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
│  Cloud Armor ──► WAF + DDoS protection + Rate limiting                     │
│  Gateway API ──► HTTPS with managed certificates                           │
│  Cloud IAP ──► Grafana (Google authentication)                             │
│  Connect Gateway ──► kubectl/Terraform (IAM auth, no IP whitelist)         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Security Features

| Feature | Description |
|---------|-------------|
| **Cloud Armor** | WAF with OWASP rules, DDoS protection, rate limiting, geo-blocking |
| **JWT Authentication** | Stateless auth with access/refresh tokens, HttpOnly cookies, bcrypt passwords |
| **Gateway API** | HTTPS with Google-managed SSL certificates |
| **Cloud IAP** | Google authentication for Grafana (automated via Terraform) |
| **Connect Gateway** | Secure cluster access without IP whitelisting |
| **Private Nodes** | GKE nodes have no public IPs |
| **Workload Identity** | Secure GCP API access without service account keys |
| **Secret Manager** | Secrets stored in GSM, synced via External Secrets Operator |
| **Database SSL** | Encrypted connections to Cloud SQL (sslmode=require) |

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
open https://grudapp.com          # API
open https://admin.grudapp.com    # Admin Panel
open https://grafana.grudapp.com  # Grafana (IAP protected)
```

## Services

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| **student-service** | 8080 | HTTP | Student management, JWT auth, NATS producer |
| **project-service** | 50052 | gRPC | Project management, NATS consumer |
| **admin** | 80 | HTTP | React admin panel |

## Authentication

### JWT Flow

```
┌──────────┐     POST /auth/login      ┌──────────────────┐
│  Client  │ ────────────────────────► │  student-service │
│          │                           │                  │
│          │ ◄──────────────────────── │  bcrypt verify   │
│          │   Set-Cookie: refresh     │  generate JWT    │
│          │   Body: { accessToken }   │                  │
└──────────┘                           └──────────────────┘

┌──────────┐     GET /api/students     ┌──────────────────┐
│  Client  │ ────────────────────────► │  student-service │
│          │   Authorization: Bearer   │                  │
│          │                           │  validate JWT    │
│          │ ◄──────────────────────── │  extract claims  │
│          │   { students: [...] }     │                  │
└──────────┘                           └──────────────────┘
```

### Endpoints

```bash
# Register
POST /auth/register
{"firstName": "John", "lastName": "Doe", "email": "john@example.com", "password": "password123"}

# Login
POST /auth/login
{"email": "john@example.com", "password": "password123"}
# Returns: { accessToken, refreshToken, student }

# Refresh token
POST /auth/refresh
# Uses HttpOnly cookie, returns new accessToken

# Logout
POST /auth/logout
# Invalidates refresh token
```

### Protected Routes

All `/api/*` routes require valid JWT in Authorization header:
```bash
curl -H "Authorization: Bearer <token>" https://grudapp.com/api/students
```

## Cloud Armor

Cloud Armor provides WAF and DDoS protection at the Google Cloud edge.

### Features Enabled

| Rule | Description |
|------|-------------|
| **OWASP Top 10** | SQL injection, XSS, LFI/RFI protection |
| **Rate Limiting** | 100 requests/minute per IP |
| **DDoS Protection** | Automatic mitigation at edge |

### Configuration (Terraform)

```hcl
resource "google_compute_security_policy" "api_policy" {
  name = "api-security-policy"

  # Default allow
  rule {
    action   = "allow"
    priority = "2147483647"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
  }

  # Rate limiting
  rule {
    action   = "rate_based_ban"
    priority = "1000"
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      rate_limit_threshold {
        count        = 100
        interval_sec = 60
      }
    }
  }

  # OWASP SQL injection
  rule {
    action   = "deny(403)"
    priority = "2000"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-v33-stable')"
      }
    }
  }
}
```

## API Documentation

### Students (requires JWT)

```bash
GET    /api/students          # List all
GET    /api/students/{id}     # Get by ID
POST   /api/students          # Create
PUT    /api/students/{id}     # Update
DELETE /api/students/{id}     # Delete
```

### Projects (via gRPC)

```bash
GET    /api/projects          # List all
GET    /api/projects/{id}     # Get by ID
POST   /api/projects          # Create
```

### Messages (NATS)

```bash
POST   /api/messages          # Send message via NATS
```

## GKE Deployment

### Prerequisites

```bash
brew install google-cloud-sdk terraform kubectl helm ko
```

### Deploy

```bash
# 1. Authenticate
make gke/auth

# 2. Create terraform.tfvars
cd terraform
cat > terraform.tfvars << EOF
project_id = "your-project-id"
connect_gateway_users = ["user:your-email@gmail.com"]
EOF

# 3. Deploy infrastructure
make tf/init && make tf/apply

# 4. Connect and deploy
make gke/connect
make infra/deploy-gke
make gke/deploy
```

### Access

| Service | URL | Auth |
|---------|-----|------|
| API | https://grudapp.com | Public (Cloud Armor protected) |
| Admin | https://admin.grudapp.com | Public |
| Grafana | https://grafana.grudapp.com | Google IAP |
| kubectl | `make gke/connect` | Google IAM |

## Makefile Commands

```bash
# Build & Test
make build              # Build all services
make test               # Run all tests

# Kind (Local)
make kind/setup         # Create cluster
make kind/deploy        # Deploy services
make kind/cleanup       # Delete cluster

# GKE (Production)
make gke/connect        # Connect via Connect Gateway
make gke/deploy         # Build and deploy
make gke/status         # Show status

# Terraform
make tf/init            # Initialize
make tf/apply           # Apply infrastructure
make tf/destroy         # Destroy

# Infrastructure
make infra/deploy       # Deploy to Kind
make infra/deploy-gke   # Deploy to GKE
```

## Troubleshooting

### Connect Gateway not working

```bash
gcloud container fleet memberships list
which gke-gcloud-auth-plugin
```

### Grafana IAP error

```bash
kubectl get secret -n infra grafana-iap-secret
kubectl describe externalsecret -n infra grafana-iap-secret
```

### Pods not starting

```bash
kubectl get pods -n apps
kubectl describe pod -n apps <pod-name>
kubectl logs -n apps <pod-name>
```

## Documentation

| Document | Description |
|----------|-------------|
| [terraform/README.md](terraform/README.md) | GKE infrastructure |
| [k8s/infra/README.md](k8s/infra/README.md) | Observability stack |

## License

Educational project demonstrating cloud-native architecture patterns.
