# Infrastructure Stack

This directory contains configuration for the observability and infrastructure stack.

## Components

| Component | Purpose | Endpoint |
|-----------|---------|----------|
| **Prometheus** | Metrics collection and storage | `prometheus-kube-prometheus-prometheus.infra:9090` |
| **Grafana** | Metrics visualization | https://grafana.grudapp.com (GKE) |
| **Alertmanager** | Alerting | `alertmanager-operated.infra:9093` |
| **Grafana Alloy** | OTLP receiver, metrics/traces export | `alloy.infra:4317` |
| **Loki** | Log aggregation | `loki.infra:3100` |
| **Tempo** | Distributed tracing | `tempo.infra:4317` |
| **NATS** | Messaging | `nats.infra:4222` |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              GKE Cluster                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        infra namespace                               │   │
│  │                                                                      │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │   │
│  │  │   Grafana    │◄───│  Prometheus  │◄───│    Alloy     │          │   │
│  │  │   (IAP)      │    │              │    │   (OTLP)     │          │   │
│  │  └──────────────┘    └──────────────┘    └──────────────┘          │   │
│  │         │                   │                   ▲                   │   │
│  │         │            ┌──────┴──────┐            │                   │   │
│  │         ▼            ▼             ▼            │                   │   │
│  │  ┌──────────────┐ ┌─────────┐ ┌─────────┐      │                   │   │
│  │  │     Loki     │ │  Tempo  │ │  NATS   │      │                   │   │
│  │  │   (logs)     │ │(traces) │ │ (msgs)  │      │                   │   │
│  │  └──────────────┘ └─────────┘ └─────────┘      │                   │   │
│  └────────────────────────────────────────────────┼────────────────────┘   │
│                                                    │                        │
│  ┌─────────────────────────────────────────────────┼────────────────────┐   │
│  │                        grud namespace           │                    │   │
│  │                                                 │                    │   │
│  │  ┌──────────────────┐    ┌──────────────────┐  │                    │   │
│  │  │  student-service │────│  project-service │──┘                    │   │
│  │  │   (HTTP/OTLP)    │    │   (gRPC/OTLP)    │                       │   │
│  │  └──────────────────┘    └──────────────────┘                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          External Access                                    │
│                                                                             │
│  https://grafana.grudapp.com ──► GCE Ingress ──► Cloud IAP ──► Grafana     │
│                                  (SSL cert)     (Google auth)               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Deployment

### Kind (local development)

```bash
make infra/deploy
```

### GKE (production)

```bash
# Connect to cluster
make gke/connect

# Deploy full stack
make infra/deploy-gke
```

This deploys:
1. Prometheus + Grafana (kube-prometheus-stack)
2. Grafana Alloy (OTLP collector)
3. Loki (log aggregation)
4. Tempo (distributed tracing)
5. NATS (messaging)
6. Alerting rules

## Grafana Access

### GKE (with Cloud IAP)

Grafana is protected by Cloud Identity-Aware Proxy:

1. **URL**: https://grafana.grudapp.com
2. **Authentication**: Google account (IAP)
3. **Grafana login**: admin / admin (after IAP auth)

**Add IAP users:**
```bash
gcloud iap web add-iam-policy-binding \
  --member="user:newuser@company.com" \
  --role="roles/iap.httpsResourceAccessor" \
  --project=rugged-abacus-483006-r5
```

### Kind (local)

```bash
# Port-forward
make gke/grafana
# Open http://localhost:3000
# Login: admin / admin
```

## Configuration Files

| File | Description |
|------|-------------|
| `prometheus-values.yaml` | Base Prometheus/Grafana config |
| `prometheus-values-gke.yaml` | GKE-specific overrides |
| `alloy-values.yaml` | Grafana Alloy configuration |
| `loki-values.yaml` | Loki configuration |
| `tempo-values.yaml` | Tempo configuration |
| `nats.yaml` | NATS deployment |
| `alerting-rules.yaml` | PrometheusRule for alerts |
| `grafana-ingress.yaml` | GCE Ingress + IAP for Grafana |
| `grafana-dashboard-configmap.yaml` | Custom dashboards |
| `grafana-datasources.yaml` | Loki/Tempo datasources |

## Grafana Ingress (GKE)

The `grafana-ingress.yaml` configures:

1. **ManagedCertificate**: Google-managed SSL for `grafana.grudapp.com`
2. **BackendConfig**: Health checks and IAP configuration
3. **Service**: ClusterIP service for GCE Ingress
4. **Ingress**: GCE Ingress with static IP

### IAP Setup

IAP requires OAuth credentials:

1. Create OAuth consent screen in GCP Console
2. Create OAuth 2.0 Client ID (Web application)
3. Add redirect URI: `https://iap.googleapis.com/v1/oauth/clientIds/CLIENT_ID:handleRedirect`
4. Create Kubernetes secret:
   ```bash
   kubectl create secret generic grafana-iap-secret \
     -n infra \
     --from-literal=client_id=YOUR_CLIENT_ID \
     --from-literal=client_secret=YOUR_CLIENT_SECRET
   ```

## Metrics

### Application metrics (via OTLP)

Services send metrics to Alloy via OTLP:

```
http_server_duration_seconds{service_name="student-service", http_route="/api/students"}
rpc_server_duration_seconds{service_name="project-service", rpc_method="GetProject"}
```

### Prometheus queries

```promql
# Request rate
rate(http_server_duration_seconds_count{service_name="student-service"}[5m])

# Latency p95
histogram_quantile(0.95, rate(http_server_duration_seconds_bucket[5m]))

# Error rate
rate(http_server_duration_seconds_count{http_status_code=~"5.."}[5m])
```

## Alerting

Alerts are defined in `alerting-rules.yaml`:

| Alert | Condition |
|-------|-----------|
| HighErrorRate | Error rate > 5% for 5m |
| HighLatency | P95 latency > 1s for 5m |
| PodNotReady | Pod not ready for 5m |
| PodCrashLooping | Pod restarting frequently |

## Cleanup

```bash
# Kind
make infra/cleanup

# GKE
make infra/cleanup
```

## Troubleshooting

### Grafana not accessible (GKE)

```bash
# Check Ingress status
kubectl get ingress -n infra grafana-ingress

# Check backend health
kubectl describe ingress -n infra grafana-ingress

# Check IAP secret
kubectl get secret -n infra grafana-iap-secret

# Check certificate status
kubectl get managedcertificate -n infra grafana-managed-cert
```

### Alloy not receiving metrics

```bash
# Check Alloy logs
kubectl logs -n infra -l app.kubernetes.io/name=alloy

# Check service endpoint
kubectl get endpoints -n infra alloy
```

### Loki not receiving logs

```bash
# Check Loki logs
kubectl logs -n infra -l app.kubernetes.io/name=loki

# Verify Alloy is forwarding
kubectl logs -n infra -l app.kubernetes.io/name=alloy | grep loki
```

### NATS connection issues

```bash
# Check NATS logs
kubectl logs -n infra -l app=nats

# Verify service
kubectl get svc -n infra nats
```
