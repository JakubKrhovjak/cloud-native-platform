# =============================================================================
# STEP 5: IAM (runs after secrets)
# =============================================================================
# Creates service account and IAM bindings for External Secrets Operator.
#
# Dependencies:
#   secrets.tf (GSM secrets must exist before IAM bindings)
#
# This creates resources used by:
#   → helm.tf (External Secrets Operator uses this service account)
#   → Kubernetes ServiceAccount (via Workload Identity annotation)
#
# Workload Identity Flow:
#   1. GCP Service Account (grud-secrets-sa@...)
#   2. Workload Identity binding allows K8s SA to impersonate GCP SA
#   3. K8s ServiceAccount annotated with GCP SA email
#   4. Pod uses K8s SA → gets GCP SA permissions
# =============================================================================

# GCP Service Account for External Secrets Operator
resource "google_service_account" "secrets_operator" {
  account_id   = "grud-secrets-sa"
  display_name = "GRUD Secrets Operator"
  description  = "Service account for External Secrets Operator to access Secret Manager"
}

# Workload Identity binding
# Allows K8s ServiceAccounts to act as this GCP SA
resource "google_service_account_iam_binding" "secrets_operator_workload_identity" {
  service_account_id = google_service_account.secrets_operator.name
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[grud/grud-secrets-sa]",
    "serviceAccount:${var.project_id}.svc.id.goog[infra/infra-secrets-sa]"
  ]
}

# =============================================================================
# Secret Manager Access (one binding per secret)
# =============================================================================
# Grant the service account access to read each secret

resource "google_secret_manager_secret_iam_member" "secrets_operator_jwt" {
  secret_id = google_secret_manager_secret.jwt_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"

  depends_on = [google_secret_manager_secret.jwt_secret]
}

resource "google_secret_manager_secret_iam_member" "secrets_operator_student_db" {
  secret_id = google_secret_manager_secret.student_db_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"

  depends_on = [google_secret_manager_secret.student_db_credentials]
}

resource "google_secret_manager_secret_iam_member" "secrets_operator_project_db" {
  secret_id = google_secret_manager_secret.project_db_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"

  depends_on = [google_secret_manager_secret.project_db_credentials]
}

# =============================================================================
# IAP Credentials Secret Access
# =============================================================================
# Secret is managed manually, referenced via data source in iap.tf

resource "google_secret_manager_secret_iam_member" "secrets_operator_grafana_iap" {
  secret_id = data.google_secret_manager_secret.grafana_iap_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.secrets_operator.email}"
}
