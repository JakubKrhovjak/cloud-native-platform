# =============================================================================
# GKE Fleet & Connect Gateway
# =============================================================================
# Enables secure access to GKE cluster without IP whitelisting.
# Uses Google IAM for authentication instead of network-based access.
#
# After applying:
#   kubectl config use-context connectgateway_rugged-abacus-483006-r5_europe-west1_grud-cluster
#
# Dependencies:
#   apis.tf (gkehub, gkeconnect, connectgateway APIs)
#   gke.tf (cluster must exist)
# =============================================================================

# Register cluster to Fleet
resource "google_gke_hub_membership" "primary" {
  membership_id = "grud-cluster"
  project       = var.project_id
  location      = var.region

  endpoint {
    gke_cluster {
      resource_link = "//container.googleapis.com/${google_container_cluster.primary.id}"
    }
  }

  authority {
    issuer = "https://container.googleapis.com/v1/${google_container_cluster.primary.id}"
  }

  depends_on = [
    google_project_service.gkehub,
    google_project_service.gkeconnect,
    google_container_cluster.primary
  ]
}

# IAM binding for Connect Gateway access
resource "google_project_iam_member" "connect_gateway_user" {
  for_each = toset(var.connect_gateway_users)

  project = var.project_id
  role    = "roles/gkehub.gatewayReader"
  member  = each.value
}

# Additional role needed for kubectl access
resource "google_project_iam_member" "connect_gateway_viewer" {
  for_each = toset(var.connect_gateway_users)

  project = var.project_id
  role    = "roles/gkehub.viewer"
  member  = each.value
}
