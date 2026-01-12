# =============================================================================
# Cloud DNS Configuration
# =============================================================================
# Manages DNS zone, records and SSL certificates for grudapp.com
#
# These resources are protected from terraform destroy via Makefile.
# The tf/destroy target removes them from state before destroying.
#
# After applying, update nameservers in Squarespace to the ones from:
#   terraform output dns_nameservers
# =============================================================================

# Enable Cloud DNS API
resource "google_project_service" "dns" {
  service            = "dns.googleapis.com"
  disable_on_destroy = false
}

# DNS Zone
resource "google_dns_managed_zone" "grudapp" {
  name        = "grudapp-zone"
  dns_name    = "grudapp.com."
  description = "DNS zone for GRUD application"

  depends_on = [google_project_service.dns]
}

# Root domain A record
resource "google_dns_record_set" "root" {
  name         = google_dns_managed_zone.grudapp.dns_name
  managed_zone = google_dns_managed_zone.grudapp.name
  type         = "A"
  ttl          = 300
  rrdatas      = [data.google_compute_global_address.ingress_ip.address]
}

# Grafana subdomain - shares same LB as main app
resource "google_dns_record_set" "grafana" {
  name         = "grafana.${google_dns_managed_zone.grudapp.dns_name}"
  managed_zone = google_dns_managed_zone.grudapp.name
  type         = "A"
  ttl          = 300
  rrdatas      = [data.google_compute_global_address.ingress_ip.address]
}

# =============================================================================
# SSL Certificate (shared by all Ingresses)
# =============================================================================
resource "google_compute_managed_ssl_certificate" "grud" {
  name = "grud-cert"

  managed {
    domains = ["grudapp.com", "grafana.grudapp.com"]
  }
}