# =============================================================================
# STEP 3a: Cloud SQL (runs in parallel with GKE)
# =============================================================================
# Creates Cloud SQL PostgreSQL instance with private IP via VPC peering.
#
# Dependencies:
#   apis.tf (sqladmin, servicenetworking APIs)
#   vpc.tf (VPC network for peering)
#
# This creates resources used by:
#   → Application services (student-service, project-service)
#   → secrets.tf (database credentials reference Cloud SQL users)
#
# VPC Peering Flow:
#   1. Reserve IP range for Google services (10.64.0.0/16)
#   2. Create VPC peering connection to Google's network
#   3. Cloud SQL gets private IP from reserved range (e.g., 10.67.0.3)
#   4. GKE pods can reach Cloud SQL via private IP
# =============================================================================

# Reserve IP range for Google managed services (Cloud SQL, Memorystore, etc.)
# This range will be used by VPC peering - Cloud SQL gets IP from this range
# Depends on: VPC network (vpc.tf)
resource "google_compute_global_address" "private_ip_range" {
  name          = "google-managed-services-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16                          # /16 = 65,536 IPs
  network       = google_compute_network.vpc.id
}

# VPC Peering connection to Google's service producer network
# This allows Cloud SQL to get a private IP accessible from our VPC
# Depends on: IP range reservation, servicenetworking API
resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_range.name]

  depends_on = [google_project_service.servicenetworking]
}

# Cloud SQL PostgreSQL instance
# Depends on: sqladmin API, VPC peering connection
resource "google_sql_database_instance" "postgres" {
  name             = "grud-postgres"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier              = "db-custom-1-3840" # 1 vCPU, 3.75GB RAM (smallest)
    availability_type = "ZONAL"            # Single zone (no HA)
    disk_size         = 10                 # 10GB minimum
    disk_type         = "PD_HDD"           # HDD (cheaper than SSD)

    ip_configuration {
      ipv4_enabled    = false              # No public IP
      private_network = google_compute_network.vpc.id
    }

    backup_configuration {
      enabled = false                      # Disabled for dev
    }
  }

  deletion_protection = false

  depends_on = [
    google_project_service.sqladmin,
    google_service_networking_connection.private_vpc_connection
  ]
}

# =============================================================================
# Databases and Users
# =============================================================================
# Depends on: Cloud SQL instance (above)
# Used by: secrets.tf (credentials stored in Secret Manager)

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

# User for student-service
# Password from secrets.tf (random_password.student_db_password)
resource "google_sql_user" "student_user" {
  name     = "student_user"
  instance = google_sql_database_instance.postgres.name
  password = random_password.student_db_password.result
}

# User for project-service
# Password from secrets.tf (random_password.project_db_password)
resource "google_sql_user" "project_user" {
  name     = "project_user"
  instance = google_sql_database_instance.postgres.name
  password = random_password.project_db_password.result
}
