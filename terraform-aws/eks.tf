# =============================================================================
# [3/6] EKS CLUSTER
# =============================================================================
# Creates the Kubernetes cluster with three node pools.
#
# What gets created:
#   - EKS control plane (managed by AWS, ~$0.10/h)
#   - system-pool: 1x t3.small ON_DEMAND — kube-system (CoreDNS, kube-proxy, etc.)
#   - app-pool:    1x t3.small SPOT      — app workloads (taint: workload=app)
#   - infra-pool:  1x t3.medium SPOT     — NATS (taint: workload=infra)
#   - Cluster addons: CoreDNS, kube-proxy, VPC-CNI
#   - IRSA enabled (IAM Roles for Service Accounts)
#
# Depends on: [2/6] vpc.tf (VPC, subnets)
# Used by:    [4/6] rds.tf (node security group for DB access)
# =============================================================================

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  cluster_endpoint_public_access = true

  # Allow current IAM user to manage the cluster
  enable_cluster_creator_admin_permissions = true

  # GitHub Actions role - potrebuje admin pristup pro helm deploy
  access_entries = {
    github-actions = {
      principal_arn = aws_iam_role.github_actions_deploy.arn
      policy_associations = {
        admin = {
          policy_arn = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
          access_scope = {
            type = "cluster"
          }
        }
      }
    }
  }

  # Name tags for Security Groups (visible in AWS Console)
  node_security_group_name = "${var.cluster_name}-node"
  node_security_group_tags = {
    "Name" = "${var.cluster_name}-node"
  }
  cluster_security_group_name = "${var.cluster_name}-cluster"
  cluster_security_group_tags = {
    "Name" = "${var.cluster_name}-cluster"
  }

  enable_irsa = true
  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
    amazon-cloudwatch-observability = {
      most_recent = true
    }
  }

  eks_managed_node_groups = {
    system-pool = {
      instance_types = [var.system_instance_type]
      capacity_type  = "ON_DEMAND"

      min_size     = 1
      max_size     = 1
      desired_size = 1

      disk_size = var.disk_size_gb

      labels = {
        "node-type" = "system"
      }

      iam_role_additional_policies = {
        CloudWatchAgent = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
        SSM             = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
      }
    }

    app-pool = {
      instance_types = [var.app_instance_type]
      capacity_type  = "SPOT"

      min_size     = 1
      max_size     = 1
      desired_size = 1

      disk_size = var.disk_size_gb

      labels = {
        "node-type" = "app"
      }

      taints = [
        {
          key    = "workload"
          value  = "app"
          effect = "NO_SCHEDULE"
        }
      ]

      iam_role_additional_policies = {
        CloudWatchAgent = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
        SSM             = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
      }
    }

    infra-pool = {
      instance_types = [var.infra_instance_type]
      capacity_type  = "SPOT"

      min_size     = 1
      max_size     = 1
      desired_size = 1

      disk_size = var.disk_size_gb

      labels = {
        "node-type" = "infra"
      }

      taints = [
        {
          key    = "workload"
          value  = "infra"
          effect = "NO_SCHEDULE"
        }
      ]

      iam_role_additional_policies = {
        CloudWatchAgent = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
        SSM             = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
      }
    }
  }

  tags = {
    Environment = "test"
    Project     = "grud"
  }
}
