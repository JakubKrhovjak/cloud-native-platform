output "cluster_name" {
  description = "GKE cluster name"
  value       = google_container_cluster.primary.name
}

output "cluster_endpoint" {
  description = "GKE cluster endpoint"
  value       = google_container_cluster.primary.endpoint
  sensitive   = true
}

output "registry_url" {
  description = "Artifact Registry URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.grud.repository_id}"
}

output "get_credentials_command" {
  description = "Command to get cluster credentials"
  value       = "gcloud container clusters get-credentials ${google_container_cluster.primary.name} --zone ${var.zone} --project ${var.project_id}"
}

output "configure_docker_command" {
  description = "Command to configure Docker for Artifact Registry"
  value       = "gcloud auth configure-docker ${var.region}-docker.pkg.dev"
}

output "secrets_operator_sa" {
  description = "Secrets operator GCP service account email"
  value       = google_service_account.secrets_operator.email
}

# Cloud SQL
output "cloudsql_instance_name" {
  description = "Cloud SQL instance connection name"
  value       = google_sql_database_instance.postgres.connection_name
}

output "cloudsql_private_ip" {
  description = "Cloud SQL private IP address"
  value       = google_sql_database_instance.postgres.private_ip_address
}

# Ingress
output "ingress_ip" {
  description = "Static IP for application Ingress"
  value       = data.google_compute_global_address.ingress_ip.address
}

# Secrets
output "jwt_secret_name" {
  description = "Google Secret Manager JWT secret name"
  value       = google_secret_manager_secret.jwt_secret.secret_id
}

output "student_db_secret_name" {
  description = "Google Secret Manager student DB secret name"
  value       = google_secret_manager_secret.student_db_credentials.secret_id
}

output "project_db_secret_name" {
  description = "Google Secret Manager project DB secret name"
  value       = google_secret_manager_secret.project_db_credentials.secret_id
}

output "external_secrets_namespace" {
  description = "External Secrets Operator namespace"
  value       = kubernetes_namespace.external_secrets.metadata[0].name
}

