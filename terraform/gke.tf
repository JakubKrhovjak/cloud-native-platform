resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.zone # Zonal cluster (single zone = fewer nodes)

  network    = google_compute_network.vpc.name
  subnetwork = google_compute_subnetwork.subnet.name

  # Allow terraform to delete cluster
  deletion_protection = false

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }

  # Workload Identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Release channel
  release_channel {
    channel = "REGULAR"
  }

  # Dataplane V2 (eBPF/Cilium) - better networking & observability
  datapath_provider = "ADVANCED_DATAPATH"

  depends_on = [google_project_service.container]
}

# Infra node pool - for NATS only
resource "google_container_node_pool" "infra" {
  name       = "infra-pool"
  location   = var.zone
  cluster    = google_container_cluster.primary.name
  node_count = 1

  node_config {
    machine_type = var.infra_machine_type
    disk_size_gb = var.disk_size_gb
    spot         = true

    labels = {
      "node-type" = "infra"
    }

    taint {
      key    = "workload"
      value  = "infra"
      effect = "NO_SCHEDULE"
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# App node pool - for application workloads
resource "google_container_node_pool" "app" {
  name       = "app-pool"
  location   = var.zone
  cluster    = google_container_cluster.primary.name
  node_count = var.app_node_count

  autoscaling {
    min_node_count = var.app_min_node_count
    max_node_count = var.app_max_node_count
  }

  node_config {
    machine_type = var.app_machine_type
    disk_size_gb = var.disk_size_gb
    spot         = true

    labels = {
      "node-type" = "app"
    }

    taint {
      key    = "workload"
      value  = "app"
      effect = "NO_SCHEDULE"
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}
