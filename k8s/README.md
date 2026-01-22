# Kubernetes Deployment

Deploy Cloud Native Platform using **Helm** for local Kind and GKE production environments.

## Quick Start

### Local (Kind)

```bash
make kind/setup      # Create Kind cluster
make infra/deploy    # Deploy observability stack
make kind/deploy     # Deploy services
```

**Access:**
- Student Service: http://localhost:8080
- Admin Panel: http://localhost:30082
- Grafana: http://localhost:30300 (admin/admin)

### GKE Production

```bash
make tf/apply           # Deploy infrastructure via Terraform
make gke/connect        # Connect via Connect Gateway
make infra/deploy-gke   # Deploy observability
make gke/deploy         # Build & deploy services
```

**Access:**
- API: https://grudapp.com
- Admin: https://admin.grudapp.com
- Grafana: https://grafana.grudapp.com (IAP protected)

## Structure

```
k8s/
├── grud/                       # Helm chart for services
│   ├── Chart.yaml
│   ├── values.yaml             # Kind defaults
│   ├── values-gke.yaml         # GKE overrides
│   └── templates/
│       ├── student-service.yaml
│       ├── project-service.yaml
│       ├── admin-panel.yaml
│       ├── student-db.yaml
│       ├── project-db.yaml
│       └── external-secrets.yaml
│
├── infra/                      # Observability stack
│   ├── prometheus/
│   ├── grafana/
│   ├── loki/
│   ├── tempo/
│   ├── alloy/
│   └── nats/
│
└── gateway/                    # GKE Gateway API
    ├── gateway.yaml
    ├── routes.yaml
    └── backend-policies.yaml
```

## Helm Chart

### Values

| Environment | File | Description |
|-------------|------|-------------|
| Kind | `values.yaml` | Local development, NodePort, local DBs |
| GKE | `values-gke.yaml` | Production, Cloud SQL, HPA, External Secrets |

### Key Differences

| Feature | Kind | GKE |
|---------|------|-----|
| Database | CloudNativePG pods | Cloud SQL (managed) |
| Secrets | Kubernetes Secrets | Google Secret Manager |
| Ingress | NodePort | Gateway API + Cloud Armor |
| SSL | None | Google-managed certificates |
| Replicas | 1 | 2+ with HPA |

### Deploy Commands

```bash
# Kind
helm upgrade --install grud k8s/apps -n grud --create-namespace

# GKE
helm upgrade --install grud k8s/apps \
  -n grud --create-namespace \
  -f k8s/apps/values-gke.yaml \
  --set cloudSql.privateIp=$CLOUDSQL_IP \
  --set secrets.gcp.projectId=$PROJECT_ID
```

## Observability Stack

### Components

| Component | Purpose | Port |
|-----------|---------|------|
| Prometheus | Metrics collection | 9090 |
| Grafana | Visualization | 3000 |
| Loki | Log aggregation | 3100 |
| Tempo | Distributed tracing | 4317 |
| Alloy | OTLP collector | 4317 |
| NATS | Messaging | 4222 |

### Deploy

```bash
# Kind
make infra/deploy

# GKE
make infra/deploy-gke
```

## Gateway API (GKE)

Gateway API provides advanced traffic management:

```yaml
# gateway.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: grud-gateway
spec:
  gatewayClassName: gke-l7-global-external-managed
  listeners:
    - name: https
      port: 443
      protocol: HTTPS
```

### Routes

| Host | Backend |
|------|---------|
| grudapp.com | student-service |
| admin.grudapp.com | admin-panel |
| grafana.grudapp.com | grafana (IAP) |

### Security Policies

- **Cloud Armor**: WAF, DDoS, rate limiting
- **Cloud IAP**: Google authentication for Grafana
- **SSL**: Google-managed certificates

## Node Affinity

Workloads are scheduled on dedicated nodes:

| Node Type | Workloads |
|-----------|-----------|
| `app` | student-service, project-service, admin |
| `infra` | Prometheus, Grafana, Loki, Tempo |
| `db` | PostgreSQL (Kind only) |

```yaml
nodeSelector:
  node-type: app
tolerations:
  - key: workload
    value: app
    effect: NoSchedule
```

## Monitoring

```bash
# Pod status
kubectl get pods -n grud
kubectl get pods -n infra

# Logs
kubectl logs -n grud deployment/student-service
kubectl logs -n infra deployment/grafana

# Resources
kubectl top pods -n grud
```

## Troubleshooting

### Pods not starting

```bash
kubectl describe pod -n grud <pod-name>
kubectl logs -n grud <pod-name>
```

### Database connection issues

```bash
# Check Cloud SQL connectivity (GKE)
kubectl exec -it -n grud deployment/student-service -- nc -zv $CLOUDSQL_IP 5432

# Check local DB (Kind)
kubectl get pods -n grud -l app=student-db
```

### External Secrets not syncing

```bash
kubectl get externalsecret -n grud
kubectl describe externalsecret -n grud student-db-secret
```

### Gateway not routing

```bash
kubectl get gateway -n grud
kubectl get httproute -n grud
kubectl describe gateway grud-gateway -n grud
```

## Makefile Commands

```bash
# Kind
make kind/setup      # Create cluster
make kind/deploy     # Deploy services
make kind/status     # Show status
make kind/cleanup    # Delete cluster

# GKE
make gke/connect     # Connect to cluster
make gke/deploy      # Build & deploy
make gke/status      # Show status

# Infrastructure
make infra/deploy       # Deploy to Kind
make infra/deploy-gke   # Deploy to GKE
make infra/status       # Show status
```
