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
