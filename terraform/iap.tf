# =============================================================================
# Cloud Identity-Aware Proxy (IAP) Configuration
# =============================================================================
# Protects Grafana with Google authentication.
#
# Note: IAP Brand and OAuth Client are managed manually in GCP Console
#       because the IAP OAuth Admin APIs require a GCP Organization.
#       See: APIs & Services → OAuth consent screen → Credentials
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
# After terraform apply:
#   1. Go to: https://console.cloud.google.com/security/iap
#   2. Enable IAP for the Grafana backend service
#   3. Add authorized users (IAP-secured Web App User role)
#
# Dependencies:
#   apis.tf (IAP API)
# =============================================================================

# =============================================================================
# Reference existing IAP credentials secret (created manually)
# =============================================================================
data "google_secret_manager_secret" "grafana_iap_credentials" {
  secret_id = "grafana-iap-credentials"
}

# =============================================================================
# IAP Authorized Users
# =============================================================================
# Users who can access Grafana through IAP

resource "google_iap_web_iam_member" "grafana_users" {
  for_each = toset([
    "user:cloudarunning@gmail.com",
    "user:jakubkrhovjak@gmail.com"
  ])

  project = var.project_id
  role    = "roles/iap.httpsResourceAccessor"
  member  = each.value
}

# Note: IAP for GKE Ingress backends is configured via BackendConfig
# in Kubernetes, not directly in Terraform.
# See: k8s/infra/grafana-ingress.yaml