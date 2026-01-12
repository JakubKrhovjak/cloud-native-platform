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

# App node pool - for application workloads (student-service, project-service)
variable "infra_node_count" {
  description = "Number of app nodes"
  type        = number
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
}

variable "app_machine_type" {
  description = "Machine type for app nodes"
  type        = string
  default     = "e2-medium"
}

variable "app_min_node_count" {
  description = "Minimum app nodes for autoscaling"
  type        = number
  default     = 1
}

variable "app_max_node_count" {
  description = "Maximum app nodes for autoscaling"
  type        = number
  default     = 4
}

# Private cluster
variable "enable_private_endpoint" {
  description = "Enable private endpoint (API server not accessible from internet)"
  type        = bool
  default     = false
}

variable "master_ipv4_cidr_block" {
  description = "CIDR block for GKE master (control plane) - must be /28 and not overlap with other ranges"
  type        = string
  default     = "172.16.0.0/28"
}

variable "master_authorized_networks" {
  description = "List of CIDR blocks authorized to access the Kubernetes master"
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = [
    {
      cidr_block   = "10.0.0.0/24"
      display_name = "VPC subnet"
    },
    {
      cidr_block   = "141.170.140.27/32"
      display_name = "Terraform destroy"
    }
  ]
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

# Connect Gateway
variable "connect_gateway_users" {
  description = "List of users/service accounts for Connect Gateway access (format: user:email or serviceAccount:email)"
  type        = list(string)
  default     = ["user:cloudarunning@gmail.com"]
}
