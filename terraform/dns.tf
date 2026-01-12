# =============================================================================
# Cloud DNS Configuration
# =============================================================================
# Manages DNS zone and records for grudapp.com
#
# These resources have prevent_destroy = true to survive terraform destroy.
# To actually delete them, remove the lifecycle block first.
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

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.dns]
}

# Root domain A record
resource "google_dns_record_set" "root" {
  name         = google_dns_managed_zone.grudapp.dns_name
  managed_zone = google_dns_managed_zone.grudapp.name
  type         = "A"
  ttl          = 300
  rrdatas      = [data.google_compute_global_address.ingress_ip.address]

  lifecycle {
    prevent_destroy = true
  }
}

# Grafana subdomain
resource "google_dns_record_set" "grafana" {
  name         = "grafana.${google_dns_managed_zone.grudapp.dns_name}"
  managed_zone = google_dns_managed_zone.grudapp.name
  type         = "A"
  ttl          = 300
  rrdatas      = [data.google_compute_global_address.grafana_ip.address]

  lifecycle {
    prevent_destroy = true
  }
}

# =============================================================================
# Google-managed SSL Certificates
# =============================================================================

resource "google_compute_managed_ssl_certificate" "grudapp" {
  name = "grudapp-cert"

  managed {
    domains = ["grudapp.com"]
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_compute_managed_ssl_certificate" "grafana" {
  name = "grafana-cert"

  managed {
    domains = ["grafana.grudapp.com"]
  }

  lifecycle {
    prevent_destroy = true
  }
}