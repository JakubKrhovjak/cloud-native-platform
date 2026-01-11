# =============================================================================
# Cloud Identity-Aware Proxy (IAP) Configuration
# =============================================================================
# Protects Grafana with Google authentication.
#
# MANUAL STEPS REQUIRED after terraform apply:
#   1. Go to: https://console.cloud.google.com/security/iap
#   2. Configure OAuth consent screen (if not done)
#   3. Enable IAP for the Grafana backend service
#   4. Add authorized users (IAP-secured Web App User role)
#
# Dependencies:
#   apis.tf (IAP API)
# =============================================================================

# Note: IAP for GKE Ingress backends is configured via BackendConfig
# in Kubernetes, not directly in Terraform.
# See: k8s/infra/grafana-ingress.yaml
