# Terraform Infrastructure

This directory contains Terraform configuration for deploying GRUD to Google Cloud Platform (GKE).

## Architecture

```
GCP Project
├── VPC Network (grud-network)
│   ├── Subnet (10.0.0.0/24)
│   ├── Pod IP range (10.1.0.0/16)
│   ├── Service IP range (10.2.0.0/20)
│   ├── Cloud Router + NAT
│   └── Firewall rules
│
├── GKE Cluster (zonal: europe-west1-b)
│   ├── Private nodes (no public IPs)
│   ├── Workload Identity enabled
│   ├── Connect Gateway access (no IP whitelist needed)
│   ├── Infra Node Pool (3x e2-medium, spot)
│   │   └── Prometheus, Grafana, NATS, Loki, Tempo, Alloy
│   └── App Node Pool (1-4x e2-medium, spot, autoscaling)
│       └── student-service, project-service
│
├── GKE Fleet & Connect Gateway
│   ├── Fleet membership (grud-cluster)
│   └── IAM bindings for gateway access
│
├── Cloud SQL (PostgreSQL 15)
│   ├── Private IP via VPC peering
│   ├── Database: university (student-service)
│   └── Database: projects (project-service)
│
├── Cloud DNS (grudapp.com)
│   ├── A record: grudapp.com → Ingress IP
│   └── A record: grafana.grudapp.com → Grafana IP
│
├── Static IPs (Global)
│   ├── grud-ingress-ip (application)
│   └── grafana-ingress-ip (Grafana)
│
├── Artifact Registry (grud)
│   └── Container images with vulnerability scanning
│
├── Google Secret Manager
│   ├── grud-jwt-secret
│   ├── grud-student-db-credentials
│   └── grud-project-db-credentials
│
├── Cloud IAP (Identity-Aware Proxy)
│   └── Protects Grafana with Google authentication
│
└── External Secrets Operator (Helm)
    └── Syncs GSM secrets to Kubernetes
```

## Security Features

| Feature | Description |
|---------|-------------|
| **Connect Gateway** | Access kubectl/Terraform from anywhere without IP whitelisting |
| **Cloud IAP** | Google authentication for Grafana access |
| **Private Nodes** | GKE nodes have no public IPs |
| **Workload Identity** | Secure GCP API access without service account keys |
| **VPC Peering** | Cloud SQL accessible only via private IP |
| **Secret Manager** | Secrets stored securely, synced via ESO |
| **HTTPS** | Google-managed SSL certificates |

## Prerequisites

1. **Google Cloud SDK** installed and configured
2. **Terraform** >= 1.0
3. **kubectl** and **Helm** installed
4. **Ko** for building Go container images
5. **gke-gcloud-auth-plugin** for kubectl authentication

```bash
# Install on macOS
brew install google-cloud-sdk terraform kubectl helm ko

# Ensure gke-gcloud-auth-plugin is in PATH
echo 'export PATH="/opt/homebrew/share/google-cloud-sdk/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Quick Start

### 1. Authenticate with GCP

```bash
make gke/auth
# Or manually:
gcloud auth login
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### 2. Create terraform.tfvars

```bash
cd terraform
cat > terraform.tfvars << EOF
project_id = "your-project-id"
region     = "europe-west1"
zone       = "europe-west1-b"

# Node configuration
infra_node_count = 3
app_node_count   = 1

# Database passwords (used for initial creation)
db_password_student = "initial-password"
db_password_project = "initial-password"

# Connect Gateway users (optional, defaults to your email)
connect_gateway_users = ["user:your-email@gmail.com"]
EOF
```

### 3. Deploy Infrastructure

```bash
make tf/init
make tf/plan
make tf/apply
```

### 4. Connect to Cluster (via Connect Gateway)

```bash
make gke/connect

# This runs:
# gcloud container fleet memberships get-credentials grud-cluster \
#   --location=europe-west1 --project=YOUR_PROJECT_ID
```

### 5. Deploy Observability Stack

```bash
make infra/deploy-gke
```

### 6. Deploy Application

```bash
make gke/deploy
```

## File Structure

| File | Description |
|------|-------------|
| `apis.tf` | Enable required GCP APIs |
| `vpc.tf` | VPC network, subnets, NAT, firewall |
| `gke.tf` | GKE cluster and node pools |
| `fleet.tf` | GKE Fleet membership and Connect Gateway |
| `cloudsql.tf` | Cloud SQL PostgreSQL instance |
| `dns.tf` | Cloud DNS zone and records |
| `ingress.tf` | Static IP addresses (data sources) |
| `registry.tf` | Artifact Registry with vulnerability scanning |
| `secrets.tf` | Google Secret Manager secrets |
| `iam.tf` | Service accounts and IAM bindings |
| `helm.tf` | External Secrets Operator |
| `iap.tf` | Cloud IAP configuration notes |
| `outputs.tf` | Terraform outputs |
| `variables.tf` | Input variables |
| `versions.tf` | Provider version constraints |

## Connect Gateway

Connect Gateway allows kubectl and Terraform access to the GKE cluster from anywhere without IP whitelisting. Authentication is handled via Google IAM.

### How it works

1. Cluster is registered to GKE Fleet
2. Users with `roles/gkehub.gatewayReader` and `roles/gkehub.viewer` can access
3. kubectl connects through `connectgateway.googleapis.com` instead of cluster IP

### Usage

```bash
# Get Connect Gateway credentials
gcloud container fleet memberships get-credentials grud-cluster \
  --location=europe-west1 \
  --project=rugged-abacus-483006-r5

# Or use make
make gke/connect

# kubectl now works from anywhere
kubectl get nodes
```

### Adding users

Add users to `connect_gateway_users` in `terraform.tfvars`:

```hcl
connect_gateway_users = [
  "user:developer1@company.com",
  "user:developer2@company.com",
  "serviceAccount:ci-cd@project.iam.gserviceaccount.com"
]
```

## Cloud IAP (Grafana)

Grafana is protected by Cloud Identity-Aware Proxy (IAP). Users must authenticate with Google before accessing Grafana.

### Setup (already done in Terraform)

1. IAP API enabled (`apis.tf`)
2. OAuth consent screen configured (manual in Console)
3. OAuth credentials created (manual in Console)
4. BackendConfig with IAP enabled (`k8s/infra/grafana-ingress.yaml`)

### Adding users

1. Go to: https://console.cloud.google.com/security/iap
2. Select the Grafana backend service
3. Add users with "IAP-secured Web App User" role

Or via gcloud:
```bash
gcloud iap web add-iam-policy-binding \
  --member="user:newuser@company.com" \
  --role="roles/iap.httpsResourceAccessor" \
  --project=rugged-abacus-483006-r5
```

## DNS Configuration

Cloud DNS manages the `grudapp.com` domain:

| Record | Type | Value |
|--------|------|-------|
| `grudapp.com` | A | Ingress IP (35.201.103.144) |
| `grafana.grudapp.com` | A | Grafana IP (34.49.153.44) |

### Nameservers

Configure these in your domain registrar:
```
ns-cloud-c1.googledomains.com
ns-cloud-c2.googledomains.com
ns-cloud-c3.googledomains.com
ns-cloud-c4.googledomains.com
```

## Makefile Commands

```bash
# Terraform
make tf/init      # Initialize Terraform
make tf/plan      # Plan changes
make tf/apply     # Apply configuration
make tf/destroy   # Destroy all resources
make tf/output    # Show outputs

# GKE (uses Connect Gateway)
make gke/connect  # Get cluster credentials via Connect Gateway
make gke/deploy   # Build and deploy application
make gke/status   # Show cluster status
make gke/ingress  # Show Ingress IPs and status
make gke/clean    # Uninstall Helm release

# Full deployment
make gke/full-deploy  # tf/init + tf/apply + infra + app
```

## Outputs

```bash
terraform output
```

| Output | Description |
|--------|-------------|
| `cluster_name` | GKE cluster name |
| `connect_gateway_command` | kubectl context switch command |
| `registry_url` | Artifact Registry URL |
| `cloudsql_private_ip` | Cloud SQL private IP |
| `ingress_ip` | Application Ingress IP |
| `grafana_ip` | Grafana Ingress IP |
| `dns_nameservers` | Nameservers for domain registrar |

## Cost Optimization

This configuration is optimized for development/demo:

| Optimization | Savings |
|--------------|---------|
| Spot VMs | Up to 91% cheaper |
| Zonal cluster | No cross-zone traffic |
| e2-medium nodes | Smallest practical size |
| HDD storage | Cheaper than SSD |
| Single Cloud SQL | No HA |
| No backups | Disabled for Cloud SQL |

For production, consider:
- Regional cluster with multiple zones
- On-demand or mixed node pools
- SSD storage for Cloud SQL
- HA Cloud SQL (REGIONAL)
- Enable automated backups

## Troubleshooting

### Connect Gateway Issues

```bash
# Verify membership exists
gcloud container fleet memberships list

# Check IAM permissions
gcloud projects get-iam-policy PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.role:gkehub"

# Ensure gke-gcloud-auth-plugin is installed
which gke-gcloud-auth-plugin
# If not found, add to PATH:
export PATH="/opt/homebrew/share/google-cloud-sdk/bin:$PATH"
```

### Cloud SQL Connection Issues

```bash
# Verify private IP
terraform output cloudsql_private_ip

# Check VPC peering
gcloud services vpc-peerings list --network=grud-network
```

### External Secrets Not Syncing

```bash
# Check ESO logs
kubectl logs -n external-secrets-system deployment/external-secrets

# Check ExternalSecret status
kubectl describe externalsecret -n grud

# Check SecretStore
kubectl describe secretstore -n grud
```

### IAP Not Working

```bash
# Check BackendConfig
kubectl get backendconfig -n infra grafana-backend-config -o yaml

# Check IAP secret exists
kubectl get secret -n infra grafana-iap-secret

# Verify OAuth redirect URI includes:
# https://iap.googleapis.com/v1/oauth/clientIds/CLIENT_ID:handleRedirect
```

## Destroying Infrastructure

```bash
# Remove Helm releases first
make gke/clean
make infra/cleanup

# Destroy Terraform resources
make tf/destroy
```

**Note**: Static IPs and SSL certificates are preserved (data sources). To delete them:
```bash
gcloud compute addresses delete grud-ingress-ip --global
gcloud compute addresses delete grafana-ingress-ip --global
```

## URLs

After deployment:

| Service | URL |
|---------|-----|
| Application API | https://grudapp.com/api |
| Grafana | https://grafana.grudapp.com (requires IAP login) |
