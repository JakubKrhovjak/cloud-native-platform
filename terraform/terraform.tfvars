project_id   = "rugged-abacus-483006-r5"
region       = "europe-west1"
cluster_name = "grud-cluster"

# Disk size for all nodes
disk_size_gb = 20

# Infra node pool (1 node for observability: Prometheus, Grafana, Loki, Tempo, NATS)
infra_machine_type = "e2-medium"
infra_node_count     = 2

# App node pool (2 nodes for services: student-service, project-service)
app_node_count     = 1
app_machine_type   = "e2-medium"
app_min_node_count = 1
app_max_node_count = 2

# Cloud SQL passwords
db_password_student = "student_password_change_me"
db_password_project = "project_password_change_me"