# =============================================================================
# Static IP for Ingress (managed separately from Terraform lifecycle)
# =============================================================================
# The static IP is created ONCE manually and referenced here as a data source.
# This prevents the IP from being destroyed/recreated on terraform destroy.
#
# Dependencies:
#   apis.tf (compute API)
#   Manual: gcloud compute addresses create grud-ingress-ip --global
#
# This is used by:
#   → GKE Ingress for grud app (namespace: grud)
#   → GKE Ingress for Grafana (namespace: infra)
#   → DNS (grudapp.com and grafana.grudapp.com point to this IP)
#
# Why data source instead of resource?
#   - Static IP should survive terraform destroy
#   - DNS points to this IP (changing it = downtime)
#   - Once created, it never needs to change
#
# Initial setup (run once):
#   gcloud compute addresses create grud-ingress-ip --global
#
# To import existing IP:
#   terraform import google_compute_global_address.ingress_ip grud-ingress-ip
# =============================================================================

data "google_compute_global_address" "ingress_ip" {
  name = "grud-ingress-ip"
}
