# =============================================================================
# [8/8] COGNITO — Zero Trust Authentication for Grafana
# =============================================================================
# Creates Cognito User Pool, App Client, and Domain for ALB OIDC integration.
# The ALB authenticates users via Cognito before forwarding to Grafana.
#
# Auth flow:
#   Browser -> ALB -> Cognito hosted UI -> ALB callback (/oauth2/idpresponse)
#   -> ALB sets encrypted session cookie -> Grafana (authenticated)
#
# Callback URL is fixed by AWS ALB: https://<domain>/oauth2/idpresponse
#
# After terraform apply, run:
#   terraform output cognito_user_pool_id     -> use in admin-create-user
#   terraform output cognito_user_pool_arn    -> paste into ingress-grafana-eks.yaml
#   terraform output cognito_app_client_id    -> paste into ingress-grafana-eks.yaml
# =============================================================================

resource "aws_cognito_user_pool" "main" {
  name = "${var.cluster_name}-user-pool"

  # Disable self-registration — admin creates users only
  admin_create_user_config {
    allow_admin_create_user_only = true
    invite_message_template {
      email_subject = "Your GRUD Platform access"
      email_message = "Your username is {username} and temporary password is {####}"
      sms_message   = "Your username is {username} and temporary password is {####}"
    }
  }

  auto_verified_attributes = ["email"]

  password_policy {
    minimum_length                   = 6
    require_uppercase                = false
    require_lowercase                = false
    require_numbers                  = false
    require_symbols                  = false
    temporary_password_validity_days = 365
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  tags = {
    Environment = "test"
    Project     = "grud"
  }
}

resource "aws_cognito_user_pool_domain" "main" {
  # Domain prefix must be globally unique across all AWS accounts
  domain       = "${var.cluster_name}-auth"
  user_pool_id = aws_cognito_user_pool.main.id
}

resource "aws_cognito_user_pool_client" "alb" {
  name         = "${var.cluster_name}-alb-client"
  user_pool_id = aws_cognito_user_pool.main.id

  generate_secret = true

  allowed_oauth_flows                  = ["code"]
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_scopes                 = ["openid", "email", "profile"]

  supported_identity_providers = ["COGNITO"]

  # Fixed by AWS ALB — do not change
  callback_urls = ["https://aws.grudapp.com/oauth2/idpresponse"]
  logout_urls   = ["https://aws.grudapp.com/grafana"]

  access_token_validity  = 1
  id_token_validity      = 1
  refresh_token_validity = 30

  token_validity_units {
    access_token  = "hours"
    id_token      = "hours"
    refresh_token = "days"
  }

  explicit_auth_flows = [
    "ALLOW_USER_SRP_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  prevent_user_existence_errors = "ENABLED"
}
