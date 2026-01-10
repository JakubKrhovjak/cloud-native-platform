# Google Secret Manager secrets
resource "google_secret_manager_secret" "jwt_secret" {
  secret_id = "grud-jwt-secret"

  replication {
    auto {}
  }

  labels = {
    app       = "grud"
    component = "auth"
  }

  depends_on = [
    google_project_service.secret_manager
  ]
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

  depends_on = [
    google_project_service.secret_manager
  ]
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

  depends_on = [
    google_project_service.secret_manager
  ]
}

# Generate random secrets
resource "random_password" "jwt_secret" {
  length  = 64
  special = true

  lifecycle {
    ignore_changes = [
      length,
      special,
    ]
  }
}

resource "random_password" "student_db_password" {
  length           = 32
  special          = true
  override_special = "_-"  # URL-safe special characters only

  lifecycle {
    ignore_changes = [
      length,
      special,
      override_special,
    ]
  }
}

resource "random_password" "project_db_password" {
  length           = 32
  special          = true
  override_special = "_-"  # URL-safe special characters only

  lifecycle {
    ignore_changes = [
      length,
      special,
      override_special,
    ]
  }
}

# Store JWT secret
resource "google_secret_manager_secret_version" "jwt_secret" {
  secret      = google_secret_manager_secret.jwt_secret.id
  secret_data = random_password.jwt_secret.result
}

# Store student database credentials
resource "google_secret_manager_secret_version" "student_db_credentials" {
  secret = google_secret_manager_secret.student_db_credentials.id
  secret_data = jsonencode({
    username = "student_user"
    password = random_password.student_db_password.result
    database = "university"
  })
}

# Store project database credentials
resource "google_secret_manager_secret_version" "project_db_credentials" {
  secret = google_secret_manager_secret.project_db_credentials.id
  secret_data = jsonencode({
    username = "project_user"
    password = random_password.project_db_password.result
    database = "projects"
  })
}
