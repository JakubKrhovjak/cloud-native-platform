# External Secrets Operator namespace
resource "kubernetes_namespace" "external_secrets" {
  metadata {
    name = "external-secrets-system"
    labels = {
      name = "external-secrets-system"
    }
  }
}

# External Secrets Operator Helm release
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
