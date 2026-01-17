terraform {
  required_version = ">= 1.14.0"

  backend "gcs" {
    bucket = "grud-terraform-state-rugged-abacus"
    prefix = "terraform/state"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 7.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

data "google_client_config" "default" {}

# Data source to look up existing GKE cluster (may not exist during bootstrap)
data "google_container_cluster" "primary" {
  count    = var.skip_kubernetes_provider ? 0 : 1
  name     = var.cluster_name
  location = var.zone
}

# Kubernetes/Helm providers - will be invalid during initial bootstrap
# Use: terraform apply -var="skip_kubernetes_provider=true" for first run
provider "kubernetes" {
  host                   = var.skip_kubernetes_provider ? "https://localhost" : "https://${data.google_container_cluster.primary[0].endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = var.skip_kubernetes_provider ? "" : base64decode(data.google_container_cluster.primary[0].master_auth[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = var.skip_kubernetes_provider ? "https://localhost" : "https://${data.google_container_cluster.primary[0].endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = var.skip_kubernetes_provider ? "" : base64decode(data.google_container_cluster.primary[0].master_auth[0].cluster_ca_certificate)
  }
}
