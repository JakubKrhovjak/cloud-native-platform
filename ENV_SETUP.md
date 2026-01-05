# Environment Configuration Guide

## Available Environments

- **local** - Local IDE development (localhost, Docker Compose databases)
- **kind** - Kind Kubernetes cluster (local development)
- **gke** - Google Kubernetes Engine (production)

## Configuration Files

Each service has YAML configuration files in `services/<service-name>/configs/`:

```
services/
├── student-service/configs/
│   ├── config.local.yaml      # IDE development
│   └── config.kind.yaml       # Kind cluster (mounted via ConfigMap)
└── project-service/configs/
    ├── config.local.yaml
    └── config.kind.yaml
```

## Environment Variable: ENV

The `ENV` variable determines which config file to load:
- `ENV=local` → loads `config.local.yaml`
- `ENV=kind` → loads `config.kind.yaml`
- Default: `local`

## Local Development (IDE)

### Configuration

**student-service** (`config.local.yaml`):
```yaml
server:
  port: 8080
  shutdownTimeout: 30s

database:
  host: localhost
  port: 5439
  user: postgres
  password: postgres
  database: university

projectService:
  grpcAddress: localhost:9090

nats:
  url: nats://localhost:4222

otel:
  endpoint: http://localhost:4317
  insecure: true
```

**project-service** (`config.local.yaml`):
```yaml
server:
  httpPort: 8081
  grpcPort: 9090

database:
  host: localhost
  port: 5440
  user: postgres
  password: postgres
  database: projects

nats:
  url: nats://localhost:4222
  subject: student.viewed

otel:
  endpoint: http://localhost:4317
  insecure: true
```

### Start Dependencies

```bash
# Start PostgreSQL databases and NATS
docker-compose up postgres postgres_projects nats -d
```

### Environment Variables (IDE)

Set in GoLand/IntelliJ run configuration:
```
ENV=local
JWT_SECRET=your-secret-key-change-this-in-production
```

## Kind Cluster

### Configuration

In Kind, services use Kubernetes service DNS names:

**student-service**:
- Database: `student-db-rw.grud.svc.cluster.local:5432`
- Project service gRPC: `project-service.grud.svc.cluster.local:9090`
- NATS: `nats://nats.infra.svc.cluster.local:4222`
- OTEL: `alloy.infra.svc.cluster.local:4317`

**project-service**:
- Database: `project-db-rw.grud.svc.cluster.local:5432`
- NATS: `nats://nats.infra.svc.cluster.local:4222`
- OTEL: `alloy.infra.svc.cluster.local:4317`

### ConfigMaps

Configuration is injected via Kubernetes ConfigMaps:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: student-service-config
data:
  config.yaml: |
    server:
      port: 8080
    database:
      host: student-db-rw
      port: 5432
      # ...
```

Helm values in `k8s/grud/values-kind.yaml` control these configurations.

## GKE (Production)

### Configuration

In GKE, services use:
- **CloudSQL**: Via Cloud SQL Proxy sidecar
- **Managed NATS**: Kubernetes service
- **Grafana Cloud**: Remote OTEL endpoint

See `k8s/grud/values-gke.yaml` for production Helm values.

### Secrets Management

Secrets are stored in Kubernetes Secrets (or GCP Secret Manager):
- Database passwords
- JWT secret
- TLS certificates

```bash
kubectl create secret generic student-db-secret \
  -n grud \
  --from-literal=password=<generated-password>
```

## Configuration Hierarchy

Configuration values are loaded in this order (later overrides earlier):

1. **Default values** in code
2. **YAML config file** (`config.{env}.yaml`)
3. **Environment variables** (e.g., `JWT_SECRET`)
4. **Kubernetes Secrets** (mounted as files or env vars)

## Key Differences Between Environments

| Configuration | Local (IDE) | Kind | GKE |
|---------------|-------------|------|-----|
| **Database Host** | `localhost:5439/5440` | `student-db-rw:5432` | Cloud SQL Proxy |
| **gRPC** | `localhost:9090` | `project-service:9090` | Internal LB |
| **NATS** | `localhost:4222` | `nats.infra:4222` | Managed NATS |
| **OTEL** | `localhost:4317` | `alloy.infra:4317` | Grafana Cloud |
| **TLS** | No | No | Yes (Ingress) |
| **Cookies** | Not secure | Not secure | Secure + SameSite |

## Switching Environments

### From Local to Kind

```bash
# Deploy to Kind
make kind/setup
make infra/deploy
make kind/deploy

# Services automatically use Kind configuration
```

### From Kind to GKE

```bash
# Create GKE cluster
cd terraform
terraform apply

# Deploy services
cd ..
make gke/deploy
```

Helm automatically uses the correct values file based on the target cluster.

## Environment Variables Reference

### Required for All Environments

- `ENV` - Environment name (local/kind/gke)
- `JWT_SECRET` - JWT signing secret (must be same across all instances)

### Optional

- `LOG_LEVEL` - Log level (debug/info/warn/error), default: info
- `OTEL_SERVICE_NAME` - Override service name for telemetry
- `OTEL_ENVIRONMENT` - Override environment for telemetry

## Debugging Configuration

### Print Active Configuration

Services log configuration on startup (with secrets redacted):

```bash
# Local
go run ./services/student-service/cmd/server

# Kind
kubectl logs -n grud -l app=student-service | grep config
```

### Verify ConfigMap

```bash
# View ConfigMap
kubectl get configmap student-service-config -n grud -o yaml

# Verify mounted config in pod
kubectl exec -n grud student-service-xxxx -- cat /etc/config/config.yaml
```

## Best Practices

1. **Never commit secrets** - Use `.gitignore` for `.env.prod`
2. **Use different JWT secrets** per environment
3. **Rotate secrets regularly** in production
4. **Keep local config in sync** with Kind/GKE structure
5. **Use CloudSQL Proxy** in GKE, never direct connections
6. **Enable secure cookies** only in production (HTTPS)

## Troubleshooting

### Config file not found

```bash
# Check ENV variable
echo $ENV

# Verify file exists
ls services/student-service/configs/

# Check search paths in logs
```

### Database connection refused

```bash
# Local: Check Docker
docker ps | grep postgres

# Kind: Check database pods
kubectl get pods -n grud -l app=student-db

# Test connection
psql -h localhost -p 5439 -U postgres -d university
```

### Wrong NATS endpoint

```bash
# Check NATS URL in config
kubectl get configmap student-service-config -n grud -o yaml | grep nats

# Verify NATS is running
kubectl get pods -n infra -l app=nats
```

### OTEL not receiving traces

```bash
# Check OTEL endpoint in config
kubectl get configmap student-service-config -n grud -o yaml | grep otel

# Check Alloy logs
kubectl logs -n infra -l app.kubernetes.io/name=alloy
```
