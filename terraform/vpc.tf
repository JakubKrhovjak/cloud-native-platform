# =============================================================================
# STEP 2: VPC Network
# =============================================================================
# Creates the network infrastructure for GKE and Cloud SQL.
#
# Dependencies:
#   apis.tf (compute API) → this file
#
# This file creates:
#   vpc.tf → gke.tf (GKE needs VPC)
#   vpc.tf → cloudsql.tf (Cloud SQL needs VPC for private peering)
# =============================================================================

# VPC Network - the main network for all resources
# Depends on: compute API (apis.tf)
resource "google_compute_network" "vpc" {
  name                    = var.network_name
  auto_create_subnetworks = false

  depends_on = [google_project_service.compute]
}

# Subnet with secondary ranges for GKE pods and services
# Depends on: VPC network (above)
resource "google_compute_subnetwork" "subnet" {
  name          = "${var.network_name}-subnet"
  ip_cidr_range = "10.0.0.0/24"       # Node IPs
  region        = var.region
  network       = google_compute_network.vpc.id

  secondary_ip_range {
    range_name    = "pods"
    ip_cidr_range = "10.100.0.0/16"   # Pod IPs (64k addresses) - avoids peering overlap
  }

  secondary_ip_range {
    range_name    = "services"
    ip_cidr_range = "10.101.0.0/20"   # Service IPs (4k addresses)
  }
}

# Cloud Router for NAT gateway
# Depends on: VPC network
resource "google_compute_router" "router" {
  name    = "${var.network_name}-router"
  region  = var.region
  network = google_compute_network.vpc.id
}

# NAT Gateway - allows private nodes to access internet (pull images, etc.)
# Depends on: Cloud Router
resource "google_compute_router_nat" "nat" {
  name                               = "${var.network_name}-nat"
  router                             = google_compute_router.router.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

# Firewall rule for GKE private cluster
# Allows master → node communication (kubectl logs, exec, etc.)
# Depends on: VPC network
resource "google_compute_firewall" "gke_master_to_kubelet" {
  name    = "${var.cluster_name}-master-kubelet"
  network = google_compute_network.vpc.name

  direction = "INGRESS"
  priority  = 900

  allow {
    protocol = "tcp"
    ports    = ["10250", "443", "8443"]
  }

  source_ranges = [var.master_ipv4_cidr_block]
}
