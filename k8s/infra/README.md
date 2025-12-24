# Infrastructure Stack

This directory contains configuration for the infrastructure stack (Prometheus, Grafana, OpenTelemetry Collector, NATS).

## Components

### 1. **Prometheus + Grafana** (kube-prometheus-stack)
- **Prometheus**: Metrics collection and storage
- **Grafana**: Metrics visualization and dashboards
- **Alertmanager**: Alerting
- **Node Exporter**: Node metrics
- **Kube State Metrics**: Kubernetes object metrics

### 2. **OpenTelemetry Collector**
- Receives OTLP metrics from Go services (student-service, project-service)
- Exports metrics to Prometheus
- Endpoint: `otel-collector.infra.svc.cluster.local:4317`

### 3. **NATS**
- Lightweight messaging system for development
- Used by student-service (producer) and project-service (consumer)
- Endpoint: `nats://nats.infra.svc.cluster.local:4222`
- Monitoring UI: http://localhost:8222 (via port-forward)

## Deployment

### Step 1: Deploy infrastructure stack
```bash
make infra/deploy
```

This command:
1. Adds Helm repositories (prometheus-community, open-telemetry)
2. Creates namespace `infra`
3. Deploys kube-prometheus-stack
4. Deploys OpenTelemetry Collector
5. Deploys NATS

### Step 2: Verify deployment
```bash
make infra/status
```

Expected output:
```
NAME                                                   READY   STATUS    RESTARTS   AGE
alertmanager-prometheus-kube-prometheus-alertmanager-0 2/2     Running   0          2m
nats-...                                               1/1     Running   0          2m
otel-collector-...                                     1/1     Running   0          2m
prometheus-grafana-...                                 3/3     Running   0          2m
prometheus-kube-prometheus-operator-...                1/1     Running   0          2m
prometheus-kube-state-metrics-...                      1/1     Running   0          2m
prometheus-prometheus-kube-prometheus-prometheus-0     2/2     Running   0          2m
```

### Step 3: Access Grafana
```bash
# Via NodePort (available at http://localhost:30300)
open http://localhost:30300

# Or via port-forward (available at http://localhost:3000)
make infra/port-forward-grafana
```

**Login credentials:**
- Username: `admin`
- Password: `admin`

### Step 4: Deploy application services
```bash
make deploy-dev
```

After deployment, services will automatically start sending metrics to OTel Collector.

## UI Access

### Grafana
- **NodePort**: http://localhost:30300
- **Port-forward**: `make infra/port-forward-grafana` → http://localhost:3000
- **Login**: admin / admin

### Prometheus
- **Port-forward**: `make infra/port-forward-prometheus` → http://localhost:9090

### NATS Monitoring
- **Port-forward**: `make infra/port-forward-nats` → http://localhost:8222

## Metrics

### HTTP metrics (student-service)
Go services automatically collect:
- `http_server_duration_seconds` - Request duration histogram
- `http_server_request_size_bytes` - Request size histogram
- `http_server_response_size_bytes` - Response size histogram

### gRPC metriky (project-service)
- `rpc_server_duration_seconds` - RPC duration histogram
- `rpc_server_request_size_bytes` - RPC request size
- `rpc_server_response_size_bytes` - RPC response size

### Prometheus queries (examples)

Request rate:
```promql
rate(http_server_duration_seconds_count{service_name="student-service"}[5m])
```

Request latency (p95):
```promql
histogram_quantile(0.95, rate(http_server_duration_seconds_bucket[5m]))
```

Error rate:
```promql
rate(http_server_duration_seconds_count{http_status_code=~"5.."}[5m])
```

## Grafana Dashboards

### Import pre-built dashboards

1. Open Grafana (http://localhost:30300)
2. Navigate to **Dashboards** → **Import**
3. Import these dashboards by ID:
   - **15661**: Kubernetes Cluster Monitoring
   - **15760**: Kubernetes / Views / Global
   - **3662**: Prometheus 2.0 Stats
   - **13639**: OTel Collector Dashboard

### Custom dashboard for GRUD services

Dashboard can be created manually with these panels:

**HTTP Requests Rate:**
```promql
sum(rate(http_server_duration_seconds_count{service_name="student-service"}[5m])) by (http_route)
```

**gRPC Requests Rate:**
```promql
sum(rate(rpc_server_duration_seconds_count{service_name="project-service"}[5m])) by (rpc_method)
```

**Response Time (p95):**
```promql
histogram_quantile(0.95,
  sum(rate(http_server_duration_seconds_bucket{service_name="student-service"}[5m])) by (le, http_route)
)
```

## Cleanup

Remove infrastructure stack:
```bash
make infra/cleanup
```

This removes:
- Prometheus + Grafana
- OpenTelemetry Collector
- NATS
- `infra` namespace

## Troubleshooting

### OTel Collector not running
```bash
kubectl logs -n infra deployment/otel-collector
```

### Services not sending metrics
```bash
# Check service logs
kubectl logs -n grud deployment/student-service | grep -i otel

# Check OTel Collector logs
kubectl logs -n infra deployment/otel-collector | grep -i receive
```

### Prometheus not scraping OTel Collector
```bash
# Check ServiceMonitor
kubectl get servicemonitor -n infra

# Check Prometheus targets
# Port-forward Prometheus and open http://localhost:9090/targets
make infra/port-forward-prometheus
```

### NATS not receiving messages
```bash
# Check NATS logs
kubectl logs -n infra deployment/nats

# Check NATS monitoring UI
make infra/port-forward-nats
# Open http://localhost:8222 to see connections and subscriptions

# Check if services can connect
kubectl logs -n grud deployment/student-service | grep -i nats
kubectl logs -n grud deployment/project-service | grep -i nats
```

## Next steps (Loki for logs)

To add log aggregation:
```bash
# TODO: Add Loki stack
helm install loki grafana/loki-stack -n infra
```
