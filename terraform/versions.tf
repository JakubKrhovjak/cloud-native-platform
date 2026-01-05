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
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
