# =============================================================================
# STEP 3c: Artifact Registry (runs in parallel with GKE and Cloud SQL)
# =============================================================================
# Creates container registry for application images.
#
# Dependencies:
#   apis.tf (artifact_registry API)
#
# This creates resources used by:
#   → Ko build process (pushes images here)
#   → GKE deployments (pulls images from here)
#
# Image URL format:
#   europe-west1-docker.pkg.dev/PROJECT_ID/grud/SERVICE_NAME:TAG
# =============================================================================

resource "google_artifact_registry_repository" "grud" {
  location      = var.region
  repository_id = "grud"
  description   = "GRUD container images"
  format        = "DOCKER"

  # Vulnerability scanning - scans images on push
  docker_config {
    immutable_tags = false
  }

  depends_on = [
    google_project_service.artifact_registry,
    google_project_service.containeranalysis,
    google_project_service.containerscanning
  ]
}

# Enable vulnerability scanning for the project
resource "google_project_service_identity" "containerscanning" {
  provider = google-beta
  project  = var.project_id
  service  = "containerscanning.googleapis.com"

  depends_on = [google_project_service.containerscanning]
}
