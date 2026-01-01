variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "europe-west1"
}

variable "zone" {
  description = "GCP Zone (for zonal cluster)"
  type        = string
  default     = "europe-west1-b"
}

variable "cluster_name" {
  description = "GKE cluster name"
  type        = string
  default     = "grud-cluster"
}

variable "network_name" {
  description = "VPC network name"
  type        = string
  default     = "grud-network"
}

variable "disk_size_gb" {
  description = "Node disk size in GB"
  type        = number
  default     = 20
}

# Infra node pool - for observability stack (Prometheus, Grafana, Loki, Tempo, NATS)
variable "infra_machine_type" {
  description = "Machine type for infra node"
  type        = string
  default     = "e2-medium"
}

# App node pool - for application workloads (student-service, project-service)
variable "app_node_count" {
  description = "Number of app nodes"
  type        = number
  default     = 2
}

variable "app_machine_type" {
  description = "Machine type for app nodes"
  type        = string
  default     = "e2-medium"
}

variable "app_min_node_count" {
  description = "Minimum app nodes for autoscaling"
  type        = number
  default     = 2
}

variable "app_max_node_count" {
  description = "Maximum app nodes for autoscaling"
  type        = number
  default     = 4
}

# Cloud SQL
variable "db_password_student" {
  description = "Password for student_user"
  type        = string
  sensitive   = true
}

variable "db_password_project" {
  description = "Password for project_user"
  type        = string
  sensitive   = true
}
