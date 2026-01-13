# =============================================================================
# STEP 3b: GKE Cluster (runs in parallel with Cloud SQL)
# =============================================================================
# Creates GKE cluster with node pools for infrastructure and applications.
#
# Dependencies:
#   apis.tf (container API)
#   vpc.tf (VPC network and subnet)
#
# This creates resources used by:
#   → helm.tf (External Secrets Operator deployed to cluster)
#   → Application deployments (via Helm)
#
# Node Pool Strategy:
#   - infra-pool: Monitoring stack (Prometheus, Grafana, NATS, etc.) - no taint
#   - app-pool: Application services - taint workload=app:NoSchedule
# =============================================================================

# GKE Cluster (control plane)
# Depends on: container API, VPC network
resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.zone                          # Zonal cluster (cheaper)

  network    = google_compute_network.vpc.name
  subnetwork = google_compute_subnetwork.subnet.name

  deletion_protection = false

  # Remove default node pool, we manage our own
  remove_default_node_pool = true
  initial_node_count       = 1

  # Use secondary IP ranges from subnet for pods/services
  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"     # 10.1.0.0/16
    services_secondary_range_name = "services" # 10.2.0.0/20
  }

  # Private cluster - nodes have no public IPs
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = var.enable_private_endpoint
    master_ipv4_cidr_block  = var.master_ipv4_cidr_block
  }

  # Restrict who can access the Kubernetes API
  master_authorized_networks_config {
    dynamic "cidr_blocks" {
      for_each = var.master_authorized_networks
      content {
        cidr_block   = cidr_blocks.value.cidr_block
        display_name = cidr_blocks.value.display_name
      }
    }
  }

  # Workload Identity - allows pods to use GCP service accounts
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  release_channel {
    channel = "REGULAR"
  }

  # Dataplane V2 (eBPF/Cilium) - better networking
  datapath_provider = "ADVANCED_DATAPATH"

  # Disable GKE managed monitoring - we use our own Prometheus stack
  monitoring_config {
    enable_components = []
    managed_prometheus {
      enabled = false
    }
  }

  # Security Posture - vulnerability scanning and compliance
  security_posture_config {
    mode               = "BASIC"
    vulnerability_mode = "VULNERABILITY_BASIC"
  }

  # Enable Gateway API
  gateway_api_config {
    channel = "CHANNEL_STANDARD"
  }

  depends_on = [google_project_service.container]
}

# =============================================================================
# Node Pools
# =============================================================================
# Depends on: GKE cluster (above)
# Used by: All Kubernetes workloads

# Infra node pool - monitoring stack, NATS, system components
# NO TAINT - allows kube-dns and other system pods
resource "google_container_node_pool" "infra" {
  name       = "infra-pool"
  location   = var.zone
  cluster    = google_container_cluster.primary.name
  node_count = 3                               # Fixed count for infra

  node_config {
    machine_type = var.infra_machine_type      # e2-medium
    disk_size_gb = var.disk_size_gb
    spot         = true                        # 60-91% cheaper

    labels = {
      "node-type" = "infra"
    }

    # No taint - system components (kube-dns, etc.) can run here

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    workload_metadata_config {
      mode = "GKE_METADATA"                    # Required for Workload Identity
    }
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# App node pool - application workloads (student-service, project-service)
# HAS TAINT - only pods with toleration can run here
resource "google_container_node_pool" "app" {
  name       = "app-pool"
  location   = var.zone
  cluster    = google_container_cluster.primary.name
  node_count = var.app_node_count

  autoscaling {
    min_node_count = var.app_min_node_count    # 1
    max_node_count = var.app_max_node_count    # 4
  }

  node_config {
    machine_type = var.app_machine_type        # e2-medium
    disk_size_gb = var.disk_size_gb
    spot         = true

    labels = {
      "node-type" = "app"
    }

    # Taint - only app pods with toleration can be scheduled here
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
