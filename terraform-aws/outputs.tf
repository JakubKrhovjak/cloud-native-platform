# =============================================================================
# OUTPUTS
# =============================================================================
# Values printed after terraform apply. Use `terraform output` to see them.
# The configure_kubectl command is needed to connect to the cluster.
# The rds_endpoint goes into k8s/apps/values-eks.yaml database.host fields.
# =============================================================================

output "cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = module.eks.cluster_endpoint
}

output "cluster_version" {
  description = "Kubernetes version"
  value       = module.eks.cluster_version
}

output "configure_kubectl" {
  description = "Command to configure kubectl"
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint (put this in values-eks.yaml database.host)"
  value       = aws_db_instance.main.address
}

output "rds_port" {
  description = "RDS PostgreSQL port"
  value       = aws_db_instance.main.port
}

output "lb_controller_role_arn" {
  description = "IAM role ARN for AWS Load Balancer Controller"
  value       = aws_iam_role.lb_controller.arn
}

output "acm_certificate_arn" {
  description = "ACM certificate ARN for aws.grudapp.com (use in Ingress annotations)"
  value       = data.aws_acm_certificate.main.arn
}

output "cognito_user_pool_id" {
  description = "Cognito User Pool ID (use in admin-create-user CLI command)"
  value       = aws_cognito_user_pool.main.id
}

output "cognito_user_pool_arn" {
  description = "Cognito User Pool ARN (paste into ingress-grafana-eks.yaml auth-idp-cognito)"
  value       = aws_cognito_user_pool.main.arn
}

output "cognito_app_client_id" {
  description = "Cognito App Client ID (paste into ingress-grafana-eks.yaml auth-idp-cognito)"
  value       = aws_cognito_user_pool_client.alb.id
}

output "cognito_domain" {
  description = "Cognito hosted UI domain (for reference)"
  value       = "https://${aws_cognito_user_pool_domain.main.domain}.auth.${var.region}.amazoncognito.com"
}
