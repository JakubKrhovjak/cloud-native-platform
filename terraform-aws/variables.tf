# =============================================================================
# [6/6] VARIABLES
# =============================================================================
# All input variables in one place. Sensitive values (passwords) should be
# provided via terraform.tfvars (gitignored) or -var flags.
#
# See terraform.tfvars.example for a template.
# =============================================================================

# -- General ------------------------------------------------------------------

variable "region" {
  description = "AWS Region"
  type        = string
  default     = "eu-central-1"
}

variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
  default     = "grud-cluster"
}

variable "cluster_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.31"
}

variable "skip_kubernetes_provider" {
  description = "Skip kubernetes/helm provider configuration (for bootstrap)"
  type        = bool
  default     = false
}

# -- VPC ----------------------------------------------------------------------

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

# -- EKS Node Pools -----------------------------------------------------------

variable "system_instance_type" {
  description = "Instance type for system nodes"
  type        = string
  default     = "t3.small"
}

variable "app_instance_type" {
  description = "Instance type for app nodes"
  type        = string
  default     = "t3.small"
}

variable "infra_instance_type" {
  description = "Instance type for infra nodes (NATS, monitoring)"
  type        = string
  default     = "t3.small"
}

variable "disk_size_gb" {
  description = "Node disk size in GB"
  type        = number
  default     = 20
}

# -- RDS PostgreSQL ------------------------------------------------------------

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t4g.micro"
}

variable "db_master_password" {
  description = "RDS master password (grud_admin user)"
  type        = string
  sensitive   = true
}

variable "db_password_student" {
  description = "Password for student_user (university DB)"
  type        = string
  sensitive   = true
}

variable "db_password_project" {
  description = "Password for project_user (projects DB)"
  type        = string
  sensitive   = true
}
