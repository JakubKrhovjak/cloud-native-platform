# ArgoCD Setup

ArgoCD is a declarative GitOps continuous delivery tool for Kubernetes.

## Quick Start

```bash
# Deploy infrastructure (includes ArgoCD)
make infra/deploy

# Or install ArgoCD separately
make argocd/install

# Get admin password
make argocd/password

# Access UI
open http://localhost:30080
```

ArgoCD is deployed in the `infra` namespace together with Prometheus, Grafana, Loki, Tempo, and NATS.

## Features

- **GitOps Workflow**: All application configurations stored in Git
- **Automated Sync**: Automatically sync cluster state with Git
- **Self-Healing**: Automatically correct drift from desired state
- **Rollback**: Easy rollback to previous versions

## Applications Managed

### apps
- **Path**: `k8s/apps`
- **Type**: Helm chart
- **Namespace**: grud
- **Services**: student-service, project-service, admin-panel

### monitoring-stack
- **Chart**: kube-prometheus-stack
- **Namespace**: infra
- **Components**: Prometheus, Grafana, AlertManager

### nats
- **Path**: `k8s/infra/nats.yaml`
- **Namespace**: infra
- **Component**: NATS messaging system

## Access

### UI Access (Kind)
- **URL**: http://localhost:30080
- **Username**: admin
- **Password**: Run `make argocd/password`

### CLI Login

```bash
# Get password
ARGOCD_PASSWORD=$(kubectl -n infra get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

# Login
argocd login localhost:30080 --username admin --password $ARGOCD_PASSWORD --insecure
```

## Common Operations

### Sync Application
```bash
argocd app sync apps
argocd app sync monitoring-stack
```

### View Application Status
```bash
argocd app list
argocd app get apps
```

### Manual Sync (disable auto-sync)
```bash
kubectl patch application apps -n infra --type json -p='[{"op": "remove", "path": "/spec/syncPolicy/automated"}]'
```

### Enable Auto-Sync
```bash
kubectl patch application apps -n infra --type json -p='[{"op": "add", "path": "/spec/syncPolicy/automated", "value": {"prune": true, "selfHeal": true}}]'
```

## Troubleshooting

### Application OutOfSync
```bash
# Check diff
argocd app diff apps

# Force sync
argocd app sync apps --force
```

### Application Health Unknown
```bash
# Check events
kubectl get events -n grud --sort-by='.lastTimestamp'

# Check pod logs
kubectl logs -n grud -l app=student-service
```

## Configuration

### Update Git Repository
Edit the applications in `k8s/infra/argocd/application-*.yaml`:

```yaml
spec:
  source:
    repoURL: https://github.com/your-username/your-repo.git
```

### Change Sync Policy
```yaml
spec:
  syncPolicy:
    automated:
      prune: true      # Delete resources removed from git
      selfHeal: true   # Sync when cluster state deviates
```

## Best Practices

1. **Keep Git as Source of Truth**: Always modify resources via Git commits
2. **Use Auto-Sync**: Let ArgoCD automatically sync changes
3. **Monitor Health**: Check application health regularly in UI
4. **Review Diffs**: Always review diffs before syncing major changes
5. **Use Projects**: Organize applications into projects for multi-tenancy

## Security

### Production Setup
For GKE, use:
- Cloud IAP for UI authentication
- GitHub/GitLab OAuth integration
- RBAC for fine-grained access control
- Sealed Secrets for sensitive data

## Resources

- [Official Documentation](https://argo-cd.readthedocs.io/)
- [Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
- [Disaster Recovery](https://argo-cd.readthedocs.io/en/stable/operator-manual/disaster_recovery/)
