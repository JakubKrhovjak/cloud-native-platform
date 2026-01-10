# Service Account for External Secrets Operator
resource "google_service_account" "secrets_operator" {
  account_id   = "grud-secrets-sa"
  display_name = "GRUD Secrets Operator"
  description  = "Service account for External Secrets Operator to access Secret Manager"
}

# Workload Identity binding for External Secrets Operator
resource "google_service_account_iam_binding" "secrets_operator_workload_identity" {
  service_account_id = google_service_account.secrets_operator.name
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[grud/grud-secrets-sa]"
  ]
}

# Grant Secret Manager access to External Secrets Operator (all secrets)
resource "google_secret_manager_secret_iam_member" "secrets_operator_jwt" {
  secret_id = google_secret_manager_secret.jwt_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"

  depends_on = [
    google_secret_manager_secret.jwt_secret
  ]
}

resource "google_secret_manager_secret_iam_member" "secrets_operator_student_db" {
  secret_id = google_secret_manager_secret.student_db_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"

  depends_on = [
    google_secret_manager_secret.student_db_credentials
  ]
}

resource "google_secret_manager_secret_iam_member" "secrets_operator_project_db" {
  secret_id = google_secret_manager_secret.project_db_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"

  depends_on = [
    google_secret_manager_secret.project_db_credentials
  ]
}
