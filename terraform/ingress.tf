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
#   → GKE Ingress (kubernetes.io/ingress.global-static-ip-name annotation)
#   → DNS (grudapp.com A record points to this IP)
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

# Static IP for Grafana LoadBalancer
resource "google_compute_address" "grafana_ip" {
  name         = "grafana-lb-ip"
  region       = var.region
  address_type = "EXTERNAL"
  network_tier = "PREMIUM"

  depends_on = [google_project_service.compute]
}
