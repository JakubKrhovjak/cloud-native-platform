resource "google_compute_network" "vpc" {
  name                    = var.network_name
  auto_create_subnetworks = false

  depends_on = [google_project_service.compute]
}

resource "google_compute_subnetwork" "subnet" {
  name          = "${var.network_name}-subnet"
  ip_cidr_range = "10.0.0.0/24"
  region        = var.region
  network       = google_compute_network.vpc.id

  secondary_ip_range {
    range_name    = "pods"
    ip_cidr_range = "10.1.0.0/16"
  }

  secondary_ip_range {
    range_name    = "services"
    ip_cidr_range = "10.2.0.0/20"
  }
}

resource "google_compute_router" "router" {
  name    = "${var.network_name}-router"
  region  = var.region
  network = google_compute_network.vpc.id
}

resource "google_compute_router_nat" "nat" {
  name                               = "${var.network_name}-nat"
  router                             = google_compute_router.router.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

# Firewall rule to allow GKE master to access kubelet on nodes
# Required for private clusters to allow master -> node communication (logs, exec, etc.)
resource "google_compute_firewall" "gke_master_to_kubelet" {
  name    = "${var.cluster_name}-master-kubelet"
  network = google_compute_network.vpc.name

  direction = "INGRESS"
  priority  = 900  # Higher priority than default GKE rules

  allow {
    protocol = "tcp"
    ports    = ["10250", "443", "8443"]
  }

  # Only from master CIDR - applies to all instances in the network
  source_ranges = [var.master_ipv4_cidr_block]
}
