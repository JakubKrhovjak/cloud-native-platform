# =============================================================================
# GITHUB ACTIONS OIDC + DEPLOY ROLE
# =============================================================================
# Flow:
#   1. git push main
#   2. GitHub Actions job zacne - GitHub vydá JWT token pro tento job
#   3. configure-aws-credentials action posle JWT na AWS STS
#   4. AWS overi JWT pomoci verejneho klice z GitHub OIDC endpointu
#   5. AWS zkontroluje podmínky v teto roli (repo + branch)
#   6. AWS vrati docasne credentials platne 15 minut
#   7. Zbytek jobu pouziva credentials pro ECR push + helm deploy
# =============================================================================

# Registruje GitHub jako duveryhodny vydavatel JWT tokenu v AWS.
# AWS sem bude chodit pro verejny klic kdyz overi podpis JWT.
#
# url          - GitHub OIDC endpoint, AWS stahne verejny klic pro overeni JWT podpisu
# client_id    - JWT musi byt urcen pro AWS STS (pole "aud" v tokenu)
# thumbprint   - otisk TLS certifikatu GitHub serveru (ochrana pred MITM)
resource "aws_iam_openid_connect_provider" "github_actions" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]

  tags = {
    Environment = "production"
    Project     = "grud"
  }
}

# IAM role kterou GitHub Actions "prebere" (AssumeRole) po uspesnem overeni JWT.
# Samotna role nema zadna opravneni - ta jsou pripojena nize (ECR, EKS).
resource "aws_iam_role" "github_actions_deploy" {
  name = "${var.cluster_name}-github-actions-deploy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        # Tuto roli smi prevzit pouze OIDC identita od GitHub providera.
        # Ne clovek, ne jina AWS sluzba - jen GitHub JWT token.
        Principal = {
          Federated = aws_iam_openid_connect_provider.github_actions.arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        # Obe podmínky musi platit zaroven.
        Condition = {
          StringEquals = {
            # JWT musi byt urcen pro AWS STS (pole "aud" v tokenu)
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            # JWT musi rikat ze pochazi z tohoto repa a main branche.
            # Blokuje forky a feature branche - ty deployovat nesmeji.
            # Pole "sub" v JWT vyplni GitHub automaticky pri spusteni jobu.
            "token.actions.githubusercontent.com:sub" = "repo:JakubKrhovjak/cloud-native-platform:ref:refs/heads/main"
          }
        }
      }
    ]
  })

  tags = {
    Environment = "production"
    Project     = "grud"
  }
}

# Opravneni pushovat Docker images do ECR (potreba pro ko build + docker push).
# PowerUser = push/pull images, vytvorit repo. Nemuze mazat registry.
resource "aws_iam_role_policy_attachment" "github_actions_ecr" {
  role       = aws_iam_role.github_actions_deploy.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser"
}

# Minimalni opravneni pro "aws eks update-kubeconfig".
# DescribeCluster = stahne endpoint + certifikat clusteru pro kubectl.
# Samotny helm deploy pak ridi Kubernetes RBAC uvnitr clusteru, ne IAM.
resource "aws_iam_role_policy" "github_actions_eks" {
  name = "eks-deploy"
  role = aws_iam_role.github_actions_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "eks:DescribeCluster",
          "eks:ListClusters"
        ]
        Resource = "*"
      }
    ]
  })
}

# ARN role - po "terraform apply" zkopiruj tuto hodnotu
# a vloz ji do GitHub Secrets jako AWS_DEPLOY_ROLE_ARN.
output "github_actions_role_arn" {
  description = "IAM role ARN for GitHub Actions - paste into GitHub Actions workflow"
  value       = aws_iam_role.github_actions_deploy.arn
}
