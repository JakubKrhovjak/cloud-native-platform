# =============================================================================
# STEP 1: Enable GCP APIs
# =============================================================================
# These must be enabled FIRST before any other resources can be created.
# All other resources depend on these APIs being active.
#
# Dependency chain:
#   apis.tf (this file) → vpc.tf, gke.tf, cloudsql.tf, registry.tf, secrets.tf
# =============================================================================

# Required for: GKE cluster (gke.tf)
resource "google_project_service" "container" {
  service            = "container.googleapis.com"
  disable_on_destroy = false
}

# Required for: Container images (registry.tf)
resource "google_project_service" "artifact_registry" {
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

# Required for: VPC, Subnets, Firewall, NAT, Static IPs (vpc.tf, ingress.tf)
resource "google_project_service" "compute" {
  service            = "compute.googleapis.com"
  disable_on_destroy = false
}

# Required for: Google Secret Manager secrets (secrets.tf)
resource "google_project_service" "secret_manager" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

# Required for: Cloud SQL instance (cloudsql.tf)
resource "google_project_service" "sqladmin" {
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}

# Required for: VPC Peering to Cloud SQL (cloudsql.tf)
resource "google_project_service" "servicenetworking" {
  service            = "servicenetworking.googleapis.com"
  disable_on_destroy = false
}
