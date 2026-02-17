# =============================================================================
# [1/6] PROVIDERS & BACKEND
# =============================================================================
# Terraform version, required providers, and provider configuration.
# This file is processed first — it tells Terraform what plugins to download.
#
# Providers used:
#   - aws:        VPC, EKS, RDS, security groups
#   - kubernetes: (reserved for future K8s resources managed by Terraform)
#   - helm:       (reserved for future Helm releases managed by Terraform)
#   - postgresql: not used — DB init via K8s Job (k8s/jobs/rds-init.yaml)
#
# State is stored in S3 bucket. Bucket must be created before terraform init.
# Create it with: aws s3 mb s3://grud-terraform-state --region eu-central-1
# =============================================================================

terraform {
  required_version = ">= 1.14.0"

  backend "s3" {
    bucket = "grud-terraform-state"
    key    = "eks/terraform.tfstate"
    region = "eu-central-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.4"
    }

  }
}

provider "aws" {
  region = var.region
}

data "aws_eks_cluster" "cluster" {
  count = var.skip_kubernetes_provider ? 0 : 1
  name  = module.eks.cluster_name
}

data "aws_eks_cluster_auth" "cluster" {
  count = var.skip_kubernetes_provider ? 0 : 1
  name  = module.eks.cluster_name
}

provider "kubernetes" {
  host                   = var.skip_kubernetes_provider ? "https://localhost" : module.eks.cluster_endpoint
  token                  = var.skip_kubernetes_provider ? "" : data.aws_eks_cluster_auth.cluster[0].token
  cluster_ca_certificate = var.skip_kubernetes_provider ? "" : base64decode(module.eks.cluster_certificate_authority_data)
}

provider "helm" {
  kubernetes {
    host                   = var.skip_kubernetes_provider ? "https://localhost" : module.eks.cluster_endpoint
    token                  = var.skip_kubernetes_provider ? "" : data.aws_eks_cluster_auth.cluster[0].token
    cluster_ca_certificate = var.skip_kubernetes_provider ? "" : base64decode(module.eks.cluster_certificate_authority_data)
  }
}
