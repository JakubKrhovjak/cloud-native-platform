# =============================================================================
# [5/6] DATABASES & USERS
# =============================================================================
# RDS is in private subnets — cannot be reached from local machine.
# Databases and users are created via a K8s Job that runs inside the cluster.
#
# After terraform apply + kubectl configured, run:
#   kubectl apply -f k8s/jobs/rds-init.yaml
#   kubectl logs -n apps job/rds-init -f
# =============================================================================
