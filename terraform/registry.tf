resource "google_artifact_registry_repository" "grud" {
  location      = var.region
  repository_id = "grud"
  description   = "GRUD container images"
  format        = "DOCKER"

  depends_on = [google_project_service.artifact_registry]
}
