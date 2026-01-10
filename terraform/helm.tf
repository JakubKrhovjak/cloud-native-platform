# =============================================================================
# STEP 6: Helm (runs after GKE cluster is ready)
# =============================================================================
# Deploys External Secrets Operator to the GKE cluster via Helm.
#
# Dependencies:
#   gke.tf (cluster and node pool must be ready)
#
# This creates resources used by:
#   → Application Helm deployment (SecretStore, ExternalSecret resources)
#
# External Secrets Operator Flow:
#   1. ESO watches ExternalSecret resources in cluster
#   2. ExternalSecret references SecretStore (GCP Secret Manager config)
#   3. ESO reads secrets from GSM using Workload Identity
#   4. ESO creates Kubernetes Secrets from GSM data
#   5. Pods mount Kubernetes Secrets as env vars
# =============================================================================

# Namespace for External Secrets Operator
resource "kubernetes_namespace" "external_secrets" {
  metadata {
    name = "external-secrets-system"
    labels = {
      name = "external-secrets-system"
    }
  }
}

# External Secrets Operator Helm release
# Depends on: GKE cluster and infra node pool (needs somewhere to run)
resource "helm_release" "external_secrets" {
  name       = "external-secrets"
  repository = "https://charts.external-secrets.io"
  chart      = "external-secrets"
  version    = "0.9.11"
  namespace  = kubernetes_namespace.external_secrets.metadata[0].name

  set {
    name  = "installCRDs"
    value = "true"
  }

  set {
    name  = "webhook.port"
    value = "9443"
  }

  depends_on = [
    google_container_cluster.primary,
    google_container_node_pool.infra
  ]
}
