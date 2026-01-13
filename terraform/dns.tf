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

# Admin panel subdomain
resource "google_dns_record_set" "admin" {
  name         = "admin.${google_dns_managed_zone.grudapp.dns_name}"
  managed_zone = google_dns_managed_zone.grudapp.name
  type         = "A"
  ttl          = 300
  rrdatas      = [data.google_compute_global_address.ingress_ip.address]
}

# =============================================================================
# SSL Certificate (for legacy Ingress - keeping for backwards compatibility)
# =============================================================================
resource "google_compute_managed_ssl_certificate" "grud" {
  name = "grud-cert"

  managed {
    domains = ["grudapp.com", "grafana.grudapp.com", "admin.grudapp.com"]
  }
}

# =============================================================================
# Certificate Manager (for Gateway API)
# =============================================================================
# Gateway API uses Certificate Manager instead of compute SSL certificates.
# This provides more flexibility and supports wildcard certificates.

resource "google_project_service" "certificatemanager" {
  service            = "certificatemanager.googleapis.com"
  disable_on_destroy = false
}

# Certificate Map - associates certificates with Gateway
resource "google_certificate_manager_certificate_map" "grud" {
  name        = "grud-certmap"
  description = "Certificate map for GRUD Gateway"

  depends_on = [google_project_service.certificatemanager]
}

# DNS Authorization for domain validation (covers root + wildcard)
resource "google_certificate_manager_dns_authorization" "grudapp" {
  name        = "grudapp-dns-auth"
  domain      = "grudapp.com"
  description = "DNS authorization for grudapp.com and *.grudapp.com"

  depends_on = [google_project_service.certificatemanager]
}

# CNAME record for certificate validation
resource "google_dns_record_set" "cert_validation" {
  name         = google_certificate_manager_dns_authorization.grudapp.dns_resource_record[0].name
  managed_zone = google_dns_managed_zone.grudapp.name
  type         = google_certificate_manager_dns_authorization.grudapp.dns_resource_record[0].type
  ttl          = 300
  rrdatas      = [google_certificate_manager_dns_authorization.grudapp.dns_resource_record[0].data]
}

# Google-managed wildcard certificate
resource "google_certificate_manager_certificate" "grud" {
  name        = "grud-gateway-cert"
  description = "Wildcard certificate for GRUD Gateway"

  managed {
    domains = ["grudapp.com", "*.grudapp.com"]
    dns_authorizations = [
      google_certificate_manager_dns_authorization.grudapp.id
    ]
  }

  depends_on = [google_project_service.certificatemanager]
}

# Map certificate to domains
resource "google_certificate_manager_certificate_map_entry" "root" {
  name         = "grud-root-entry"
  map          = google_certificate_manager_certificate_map.grud.name
  hostname     = "grudapp.com"
  certificates = [google_certificate_manager_certificate.grud.id]
}

resource "google_certificate_manager_certificate_map_entry" "wildcard" {
  name         = "grud-wildcard-entry"
  map          = google_certificate_manager_certificate_map.grud.name
  hostname     = "*.grudapp.com"
  certificates = [google_certificate_manager_certificate.grud.id]
}