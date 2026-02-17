# =============================================================================
# ACM Certificate for aws.grudapp.com
# =============================================================================
# Certificate is pre-created and DNS-validated via GCP Cloud DNS.
# We reference it as a data source so terraform never destroys it.
# =============================================================================

data "aws_acm_certificate" "main" {
  domain   = "aws.grudapp.com"
  statuses = ["ISSUED"]
}
