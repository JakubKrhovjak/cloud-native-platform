  # Service Account for student-service
resource "google_service_account" "student_service" {
  account_id   = "student-service"
  display_name = "Student Service"
  description  = "Service account for student-service workload"
}

# Service Account for project-service
resource "google_service_account" "project_service" {
  account_id   = "project-service"
  display_name = "Project Service"
  description  = "Service account for project-service workload"
}

# Workload Identity binding for student-service
resource "google_service_account_iam_binding" "student_service_workload_identity" {
  service_account_id = google_service_account.student_service.name
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[grud/student-service]"
  ]
}

# Workload Identity binding for project-service
resource "google_service_account_iam_binding" "project_service_workload_identity" {
  service_account_id = google_service_account.project_service.name
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[grud/project-service]"
  ]
}

# Grant Cloud SQL access to student-service
resource "google_project_iam_member" "student_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.student_service.email}"
}

# Grant Cloud SQL access to project-service
resource "google_project_iam_member" "project_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.project_service.email}"
}

# Grant Secret Manager access to student-service
resource "google_secret_manager_secret_iam_member" "student_jwt_secret" {
  secret_id = google_secret_manager_secret.jwt_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.student_service.email}"

  depends_on = [
    google_secret_manager_secret.jwt_secret
  ]
}

resource "google_secret_manager_secret_iam_member" "student_db_secret" {
  secret_id = google_secret_manager_secret.student_db_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.student_service.email}"

  depends_on = [
    google_secret_manager_secret.student_db_credentials
  ]
}

# Grant Secret Manager access to project-service
resource "google_secret_manager_secret_iam_member" "project_db_secret" {
  secret_id = google_secret_manager_secret.project_db_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.project_service.email}"

  depends_on = [
    google_secret_manager_secret.project_db_credentials
  ]
}
