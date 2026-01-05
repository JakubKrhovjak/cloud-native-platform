# Static IP for Ingress
# This ensures the external IP doesn't change on redeploy

# =============================================================================
# Application Ingress IP (grudapp.com)
# =============================================================================
# This IP is managed OUTSIDE of terraform destroy/apply cycle.
# It must exist before running terraform apply.
#
# Create once manually:
#   gcloud compute addresses create grud-ingress-ip --global
#
# Or import existing:
#   terraform import google_compute_global_address.ingress_ip grud-ingress-ip
#
# This data source references the existing IP (doesn't create/destroy it)
data "google_compute_global_address" "ingress_ip" {
  name = "grud-ingress-ip"
}
