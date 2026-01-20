# Node Pool Architecture

Production-like node pool structure for workload isolation and resource management.

## Kind Cluster (Local Development)

### Node Pools:

```
┌─────────────────────────────────────────────────────────────────┐
│ Control Plane Node                                              │
│ - Kubernetes control plane components                           │
│ - No workload scheduling                                        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ System Node (grud-system-node)                                  │
│ Label: node-type=system                                         │
│ Taint: NONE                                                     │
│                                                                 │
│ Purpose: GKE system components that need flexible scheduling    │
│ - kube-dns                                                      │
│ - metrics-server                                                │
│ - local-path-provisioner                                        │
│ - Other system pods without specific tolerations                │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Infra Node (grud-infra-node)                                    │
│ Label: node-type=infra                                          │
│ Taint: workload=infra:NoSchedule                                │
│                                                                 │
│ Purpose: Monitoring and observability stack                     │
│ - ArgoCD (GitOps)                                               │
│ - Prometheus (metrics)                                          │
│ - Grafana (dashboards)                                          │
│ - Loki (logs)                                                   │
│ - Tempo (traces)                                                │
│ - Alloy (OTLP collector)                                        │
│ - NATS (messaging)                                              │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ App Node (grud-app-node)                                        │
│ Label: node-type=app                                            │
│ Taint: workload=app:NoSchedule                                  │
│                                                                 │
│ Purpose: Application workloads                                  │
│ - student-service                                               │
│ - project-service                                               │
│ - admin-panel                                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ DB Node (grud-db-node)                                          │
│ Label: node-type=db                                             │
│ Taint: workload=db:NoSchedule                                   │
│                                                                 │
│ Purpose: Database workloads (Kind only)                         │
│ - student-db (PostgreSQL)                                       │
│ - project-db (PostgreSQL)                                       │
└─────────────────────────────────────────────────────────────────┘
```

### Recreate Kind Cluster with New Structure:

```bash
# Delete old cluster
make kind/cleanup

# Create new cluster with updated config
make kind/setup

# Deploy infrastructure (includes ArgoCD)
make infra/deploy

# Build and deploy applications
make kind/build-deploy
```

## GKE Cluster (Production)

### Node Pools:

```
┌─────────────────────────────────────────────────────────────────┐
│ System Pool (system-pool)                                       │
│ Machine: e2-medium (2 vCPU, 4GB RAM)                            │
│ Count: 2 nodes (fixed, non-spot)                                │
│ Label: node-type=system                                         │
│ Taint: NONE                                                     │
│                                                                 │
│ Purpose: GKE managed system components                          │
│ - kube-dns                                                      │
│ - kube-proxy                                                    │
│ - metrics-server                                                │
│ - event-exporter                                                │
│ - fluentbit (GKE logging agent)                                 │
│ - gke-metrics-agent                                             │
│                                                                 │
│ Why no taint? GKE system pods don't have tolerations            │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Infra Pool (infra-pool)                                         │
│ Machine: e2-standard-4 (4 vCPU, 16GB RAM)                       │
│ Count: 2 nodes (fixed, spot instances)                          │
│ Label: node-type=infra                                          │
│ Taint: workload=infra:NoSchedule                                │
│                                                                 │
│ Purpose: Monitoring and observability stack                     │
│ - ArgoCD                                                        │
│ - Prometheus                                                    │
│ - Grafana                                                       │
│ - Loki                                                          │
│ - Tempo                                                         │
│ - Alloy                                                         │
│ - NATS                                                          │
│                                                                 │
│ Cost: ~$40-60/month with spot instances                         │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ App Pool (app-pool)                                             │
│ Machine: e2-standard-2 (2 vCPU, 8GB RAM)                        │
│ Count: Auto-scaling (1-4 nodes, spot instances)                 │
│ Label: node-type=app                                            │
│ Taint: workload=app:NoSchedule                                  │
│                                                                 │
│ Purpose: Application workloads                                  │
│ - student-service                                               │
│ - project-service                                               │
│ - admin-panel                                                   │
│                                                                 │
│ Cost: ~$20-80/month depending on load                           │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Database: Cloud SQL (NOT in Kubernetes)                         │
│ Machine: db-f1-micro                                            │
│ HA: Optional                                                    │
│                                                                 │
│ Why not in Kubernetes?                                          │
│ - Managed backups                                               │
│ - Automatic failover                                            │
│ - Lower operational overhead                                    │
│ - Better security (VPC private IP)                              │
│                                                                 │
│ Cost: ~$15-30/month                                             │
└─────────────────────────────────────────────────────────────────┘
```

### Deploy to GKE:

```bash
# Apply Terraform (creates node pools)
make tf/apply

# Connect to cluster
make gke/connect

# Deploy infrastructure
make infra/deploy-gke

# Deploy applications
make gke/deploy
```

## Benefits of This Architecture

### ✅ Workload Isolation
- Monitoring can't steal resources from applications
- Database workloads don't impact app performance
- System components always have resources

### ✅ Independent Scaling
- App nodes scale based on traffic
- Infra nodes fixed size (monitoring doesn't scale)
- System nodes stable (no spot instances)

### ✅ Cost Optimization
- Spot instances for non-critical workloads (60-91% cheaper)
- Right-sized machines for each workload type
- System pool stable, app pool elastic

### ✅ Security & Compliance
- Clear separation of concerns
- Easier to apply different security policies per pool
- Audit trail (who can deploy where)

### ✅ Production-Ready
- Follows GKE best practices
- Same structure as enterprise Kubernetes
- Easy to add more pools (cache, ML, etc.)

## When to Use Each Pool

| Workload Type | Pool | Example |
|---------------|------|---------|
| System components | `system` | kube-dns, metrics-server |
| Monitoring/Observability | `infra` | Prometheus, Grafana, ArgoCD |
| Application services | `app` | student-service, APIs |
| Databases (Kind only) | `db` | PostgreSQL, Redis |
| Databases (GKE) | Cloud SQL | Managed PostgreSQL |

## Tolerations Example

To schedule a pod on infra nodes:

```yaml
tolerations:
  - key: workload
    operator: Equal
    value: infra
    effect: NoSchedule
nodeSelector:
  node-type: infra
```

## Cost Estimates (GKE)

**Monthly costs with spot instances:**
- System pool (2x e2-medium): ~$20-30
- Infra pool (2x e2-standard-4): ~$40-60
- App pool (1-4x e2-standard-2): ~$20-80
- Cloud SQL (db-f1-micro): ~$15-30
- **Total: $95-200/month** depending on load

**Without spot instances: ~$300-400/month**

## Monitoring Node Usage

```bash
# Check node labels
kubectl get nodes --show-labels

# Check node taints
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints

# Check pods per node
kubectl get pods -A -o wide | grep <node-name>

# Check resource usage
kubectl top nodes
```
