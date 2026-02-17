# =============================================================================
# GuardDuty — Threat Detection
# =============================================================================
# GuardDuty is enabled manually (persists across cluster destroy/create).
# This data source just reads the existing detector ID.
# =============================================================================

data "aws_guardduty_detector" "main" {}
