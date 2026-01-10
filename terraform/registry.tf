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

  depends_on = [google_project_service.artifact_registry]
}
