# Enable Cloud SQL API
resource "google_project_service" "sqladmin" {
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}

# Cloud SQL PostgreSQL instance
resource "google_sql_database_instance" "postgres" {
  name             = "grud-postgres"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier              = "db-custom-1-3840" # Smallest for PostgreSQL (1 vCPU, 3.75GB RAM)
    availability_type = "ZONAL"            # Single zone (cheaper)
    disk_size         = 10                 # 10GB minimum
    disk_type         = "PD_HDD"           # HDD (cheaper than SSD)

    ip_configuration {
      ipv4_enabled    = false # No public IP
      private_network = google_compute_network.vpc.id
    }

    backup_configuration {
      enabled = false # Disable for dev (enable for prod)
    }
  }

  deletion_protection = false # Allow terraform destroy

  depends_on = [
    google_project_service.sqladmin,
    google_service_networking_connection.private_vpc_connection
  ]
}

# Private VPC connection for Cloud SQL
resource "google_compute_global_address" "private_ip_range" {
  name          = "google-managed-services-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_range.name]

  depends_on = [google_project_service.servicenetworking]
}

resource "google_project_service" "servicenetworking" {
  service            = "servicenetworking.googleapis.com"
  disable_on_destroy = false
}

# Database: university (for student-service)
resource "google_sql_database" "university" {
  name     = "university"
  instance = google_sql_database_instance.postgres.name
}

# Database: projects (for project-service)
resource "google_sql_database" "projects" {
  name     = "projects"
  instance = google_sql_database_instance.postgres.name
}

# User for student-service (university database)
resource "google_sql_user" "student_user" {
  name     = "student_user"
  instance = google_sql_database_instance.postgres.name
  password = random_password.student_db_password.result
}

# User for project-service (projects database)
resource "google_sql_user" "project_user" {
  name     = "project_user"
  instance = google_sql_database_instance.postgres.name
  password = random_password.project_db_password.result
}
