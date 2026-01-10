# =============================================================================
# STEP 4: Secrets (runs after APIs, parallel with other resources)
# =============================================================================
# Creates secrets in Google Secret Manager and generates random passwords.
#
# Dependencies:
#   apis.tf (secret_manager API)
#
# This creates resources used by:
#   → cloudsql.tf (Cloud SQL users get passwords from random_password)
#   → iam.tf (IAM bindings grant access to these secrets)
#   → External Secrets Operator (syncs GSM secrets to Kubernetes)
#
# Password Strategy:
#   - URL-safe characters only (_-) because passwords are embedded in DSN URLs
#   - 32 characters for DB passwords, 64 for JWT
#   - lifecycle.ignore_changes prevents regeneration on each apply
# =============================================================================

# =============================================================================
# Secret Manager Secrets (containers for secret data)
# =============================================================================

resource "google_secret_manager_secret" "jwt_secret" {
  secret_id = "grud-jwt-secret"

  replication {
    auto {}
  }

  labels = {
    app       = "grud"
    component = "auth"
  }

  depends_on = [google_project_service.secret_manager]
}

resource "google_secret_manager_secret" "student_db_credentials" {
  secret_id = "grud-student-db-credentials"

  replication {
    auto {}
  }

  labels = {
    app       = "grud"
    component = "database"
    service   = "student"
  }

  depends_on = [google_project_service.secret_manager]
}

resource "google_secret_manager_secret" "project_db_credentials" {
  secret_id = "grud-project-db-credentials"

  replication {
    auto {}
  }

  labels = {
    app       = "grud"
    component = "database"
    service   = "project"
  }

  depends_on = [google_project_service.secret_manager]
}

# =============================================================================
# Random Password Generation
# =============================================================================
# These are used by: cloudsql.tf (Cloud SQL users), secret versions (below)

resource "random_password" "jwt_secret" {
  length  = 64
  special = true

  lifecycle {
    ignore_changes = [length, special]
  }
}

# URL-safe passwords for database DSN strings
# Special chars like @:/ break URL parsing, so we only allow _ and -
resource "random_password" "student_db_password" {
  length           = 32
  special          = true
  override_special = "_-"

  lifecycle {
    ignore_changes = [length, special, override_special]
  }
}

resource "random_password" "project_db_password" {
  length           = 32
  special          = true
  override_special = "_-"

  lifecycle {
    ignore_changes = [length, special, override_special]
  }
}

# =============================================================================
# Secret Versions (actual secret values)
# =============================================================================
# These store the generated passwords in Secret Manager

resource "google_secret_manager_secret_version" "jwt_secret" {
  secret      = google_secret_manager_secret.jwt_secret.id
  secret_data = random_password.jwt_secret.result
}

# Database credentials stored as JSON (External Secrets extracts individual fields)
resource "google_secret_manager_secret_version" "student_db_credentials" {
  secret = google_secret_manager_secret.student_db_credentials.id
  secret_data = jsonencode({
    username = "student_user"
    password = random_password.student_db_password.result
    database = "university"
  })
}

resource "google_secret_manager_secret_version" "project_db_credentials" {
  secret = google_secret_manager_secret.project_db_credentials.id
  secret_data = jsonencode({
    username = "project_user"
    password = random_password.project_db_password.result
    database = "projects"
  })
}
