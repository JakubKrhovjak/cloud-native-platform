# =============================================================================
# [2/6] VPC NETWORK
# =============================================================================
# Creates the VPC where EKS and RDS live.
#
# What gets created:
#   - VPC (10.0.0.0/16)
#   - 2 private subnets (10.0.1.0/24, 10.0.2.0/24) — EKS nodes + RDS
#   - 2 public subnets (10.0.101.0/24, 10.0.102.0/24) — NAT Gateway, load balancers
#   - NAT Gateway (single, for cost savings) — allows private subnets to reach internet
#   - Subnet tags for K8s service discovery (ELB/internal-ELB)
#
# Depends on: [1/6] versions.tf (AWS provider)
# Used by:    [3/6] eks.tf, [4/6] rds.tf
# =============================================================================

data "aws_availability_zones" "available" {
  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.cluster_name}-vpc"
  cidr = var.vpc_cidr

  azs             = slice(data.aws_availability_zones.available.names, 0, 2)
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }

  tags = {
    Environment = "test"
    Project     = "grud"
  }
}

# =============================================================================
# VPC Flow Logs → CloudWatch
# =============================================================================
# Logs all network traffic (accept/reject) for security audit and debugging.
# GuardDuty also uses flow logs for better threat detection.
# =============================================================================

resource "aws_flow_log" "vpc" {
  vpc_id               = module.vpc.vpc_id
  traffic_type         = "ALL"
  log_destination_type = "cloud-watch-logs"
  log_destination      = aws_cloudwatch_log_group.vpc_flow_logs.arn
  iam_role_arn         = aws_iam_role.vpc_flow_logs.arn

  tags = {
    Environment = "test"
    Project     = "grud"
  }
}

resource "aws_cloudwatch_log_group" "vpc_flow_logs" {
  name              = "/aws/vpc/flow-logs/${var.cluster_name}"
  retention_in_days = 7

  tags = {
    Environment = "test"
    Project     = "grud"
  }
}

resource "aws_iam_role" "vpc_flow_logs" {
  name = "${var.cluster_name}-vpc-flow-logs"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "vpc-flow-logs.amazonaws.com"
      }
    }]
  })

  tags = {
    Environment = "test"
    Project     = "grud"
  }
}

resource "aws_iam_role_policy" "vpc_flow_logs" {
  name = "vpc-flow-logs-publish"
  role = aws_iam_role.vpc_flow_logs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ]
      Effect   = "Allow"
      Resource = "*"
    }]
  })
}
