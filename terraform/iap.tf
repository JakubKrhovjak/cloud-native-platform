# =============================================================================
# Cloud Identity-Aware Proxy (IAP) Configuration
# =============================================================================
# Protects Grafana with Google authentication.
#
# TODO: Implement Workforce Identity Federation with Okta
#       - Replace Google Identity with Okta SSO
#       - Requires: GCP Organization, Okta OIDC app
#       - See: https://cloud.google.com/iam/docs/workforce-identity-federation
#
# TODO: Setup Tailscale VPN for private cluster access
#       - Free for personal use (up to 100 devices)
#       - Deploy Tailscale operator to GKE
#       - Use as subnet router for private services
#       - See: https://tailscale.com/kb/1236/kubernetes-operator
#
# TODO: Add Cloud Armor for API protection
#       - WAF rules (SQL injection, XSS protection)
#       - Rate limiting per IP
#       - Optional geo-blocking
#       - ~$5/month + $0.75/million requests
#       - See: https://cloud.google.com/armor/docs
#
# Prerequisites:
#   - OAuth consent screen must be configured in GCP Console
#     (APIs & Services → OAuth consent screen)
#
# After terraform apply:
#   1. Go to: https://console.cloud.google.com/security/iap
#   2. Enable IAP for the Grafana backend service
#   3. Add authorized users (IAP-secured Web App User role)
#
# Dependencies:
#   apis.tf (IAP API)
# =============================================================================

# Get project info for brand reference
data "google_project" "current" {}

# =============================================================================
# IAP OAuth Brand (OAuth consent screen)
# =============================================================================
# Note: If you already have an OAuth consent screen configured,
# import it: terraform import google_iap_brand.grafana projects/PROJECT_NUMBER/brands/PROJECT_NUMBER

resource "google_iap_brand" "grafana" {
  support_email     = "cloudarunning@gmail.com"
  application_title = "GRUD Grafana"
  project           = data.google_project.current.number
}

# =============================================================================
# IAP OAuth Client
# =============================================================================
resource "google_iap_client" "grafana" {
  display_name = "Grafana IAP Client"
  brand        = google_iap_brand.grafana.name
}

# =============================================================================
# Store IAP credentials in Secret Manager
# =============================================================================
resource "google_secret_manager_secret" "grafana_iap_credentials" {
  secret_id = "grafana-iap-credentials"

  replication {
    auto {}
  }

  labels = {
    app       = "grafana"
    component = "iap"
  }

  depends_on = [google_project_service.secret_manager]
}

resource "google_secret_manager_secret_version" "grafana_iap_credentials" {
  secret      = google_secret_manager_secret.grafana_iap_credentials.id
  secret_data = jsonencode({
    client_id     = google_iap_client.grafana.client_id
    client_secret = google_iap_client.grafana.secret
  })
}

# =============================================================================
# IAP Authorized Users
# =============================================================================
# Users who can access Grafana through IAP

resource "google_iap_web_iam_member" "grafana_users" {
  for_each = toset([
    "user:cloudarunning@gmail.com",
    "user:jakub.krhovjak@protonmail.com"
  ])

  project = var.project_id
  role    = "roles/iap.httpsResourceAccessor"
  member  = each.value
}

# Note: IAP for GKE Ingress backends is configured via BackendConfig
# in Kubernetes, not directly in Terraform.
# See: k8s/infra/grafana-ingress.yaml
