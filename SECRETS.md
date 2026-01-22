# Secret Management

This document describes how secrets are managed in the GRUD application for both Kind (local development) and GKE (production) environments.

## Overview

GRUD uses different secret management approaches depending on the deployment environment:

- **Kind (Local Development)**: Direct Kubernetes Secrets
- **GKE (Production)**: Google Secret Manager + External Secrets Operator

## Secrets Used

The application uses three main secrets:

1. **JWT Secret** (`jwt-secret`)
   - Used for signing and verifying JWT authentication tokens
   - Required by: student-service

2. **Student Database Credentials** (`student-db-secret`)
   - Database connection credentials for the student service
   - Contains: username, password, database name
   - Required by: student-service

3. **Project Database Credentials** (`project-db-secret`)
   - Database connection credentials for the project service
   - Contains: username, password, database name
   - Required by: project-service

## Kind (Local Development)

### Architecture

For Kind clusters, secrets are created as standard Kubernetes Secret resources. These are suitable for local development and testing.

### Setup

Generate secrets for Kind:

```bash
make secrets/generate-kind
```

This will:
- Create the `grud` namespace if it doesn't exist
- Generate random 256-bit secrets for JWT and database passwords
- Create three Kubernetes Secrets in the cluster:
  - `jwt-secret`
  - `student-db-secret`
  - `project-db-secret`

### Manual Creation

You can also manually create secrets:

```bash
# Create JWT secret
kubectl create secret generic jwt-secret \
  --from-literal=jwt-secret="your-secret-key" \
  --namespace=grud

# Create student database secret
kubectl create secret generic student-db-secret \
  --from-literal=username=app \
  --from-literal=password="your-password" \
  --from-literal=database=university \
  --namespace=grud

# Create project database secret
kubectl create secret generic project-db-secret \
  --from-literal=username=app \
  --from-literal=password="your-password" \
  --from-literal=database=projects \
  --namespace=grud
```

### Listing Secrets

View secrets in Kind cluster:

```bash
make secrets/list-kind
```

Or directly:

```bash
kubectl get secrets -n apps -l app=grud,component=secrets
```

## GKE (Production)

### Architecture

For GKE, secrets are stored in Google Secret Manager and synchronized to Kubernetes using the External Secrets Operator. This provides:

- Centralized secret management
- Automatic secret rotation
- IAM-based access control
- Audit logging
- Secret versioning

### Prerequisites

1. **Google Cloud Project** with Secret Manager API enabled
2. **GKE Cluster** with Workload Identity enabled
3. **Service Account** with Secret Manager access
4. **External Secrets Operator** installed in the cluster

### Setup

All GKE secrets and External Secrets Operator are managed by Terraform. The setup is automated during infrastructure deployment.

#### 1. Deploy Infrastructure with Terraform

```bash
cd terraform
terraform init
terraform apply
```

This will:
- Create Google Secret Manager secrets:
  - `grud-jwt-secret`
  - `grud-student-db-credentials` (JSON format)
  - `grud-project-db-credentials` (JSON format)
- Generate random secure passwords and JWT secret
- Install External Secrets Operator via Helm
- Grant service accounts access to secrets via IAM bindings

#### 2. Deploy Application

When deploying with Helm, use the GKE values file:

```bash
make gke/deploy
```

Or manually:

```bash
helm upgrade --install grud k8s/apps \
  -n apps --create-namespace \
  -f k8s/apps/values-gke.yaml \
  --set secrets.gcp.projectId=YOUR_PROJECT_ID \
  --wait
```

Note: Project ID and other settings are automatically configured by the Makefile from Terraform outputs.

### How It Works

1. **Secrets are stored in Google Secret Manager**
   - Centralized, encrypted storage
   - Version controlled
   - Access controlled via IAM

2. **SecretStore resource** (`secret-store.yaml`)
   - Configures connection to Google Secret Manager
   - Uses Workload Identity for authentication
   - Specifies GCP project and cluster details

3. **ExternalSecret resources** (`external-secret.yaml`)
   - Define mappings from GSM secrets to Kubernetes Secrets
   - Automatically sync secrets from GSM
   - Create standard Kubernetes Secrets that pods can use

4. **Services consume secrets** as environment variables
   - Standard Kubernetes secret mounting
   - No code changes required
   - Transparent to the application

### Secret Rotation

To rotate secrets in GKE:

#### Option 1: Terraform-managed (Recommended)

Update the secret values in Terraform:

```bash
# This will regenerate all random passwords and update secrets
cd terraform
terraform taint random_password.jwt_secret
terraform taint random_password.student_db_password
terraform taint random_password.project_db_password
terraform apply
```

#### Option 2: Manual rotation

1. **Generate new secret version** in Google Secret Manager:
   ```bash
   echo -n "new-jwt-secret" | gcloud secrets versions add grud-jwt-secret --data-file=-
   ```

2. **External Secrets Operator** will automatically sync the new version (within 1 hour, configurable)

3. **Restart deployments** to pick up new secrets:
   ```bash
   kubectl rollout restart deployment -n apps
   ```

### Listing Secrets

View secrets in Google Secret Manager:

```bash
make secrets/list-gke
```

Or directly:

```bash
gcloud secrets list --filter="name:grud-"
```

## Configuration Reference

### Kind Values (`values-kind.yaml`)

```yaml
secrets:
  createKubernetesSecrets: true   # Create Kubernetes Secrets
  useExternalSecrets: false        # Don't use External Secrets

databases:
  student:
    host: student-db-rw.apps.svc.cluster.local
    port: 5432
    database: university

  project:
    host: project-db-rw.apps.svc.cluster.local
    port: 5432
    database: projects
```

Note: For Kind, database credentials are managed by CloudNativePG operator and secrets are generated by the `scripts/generate-secrets.sh` script.

### GKE Values (`values-gke.yaml`)

```yaml
secrets:
  createKubernetesSecrets: false   # Don't create direct secrets
  useExternalSecrets: true         # Use External Secrets

  gcp:
    projectId: ""                  # Set via --set secrets.gcp.projectId
    clusterLocation: europe-west1-b  # Zone for zonal clusters, region for regional
    clusterName: grud-cluster
    serviceAccountName: grud-secrets-sa
    jwtSecretName: grud-jwt-secret
    studentDbSecretName: grud-student-db-credentials
    projectDbSecretName: grud-project-db-credentials
```

**Important**: For zonal GKE clusters, `clusterLocation` must be the zone (e.g., `europe-west1-b`), not the region. For regional clusters, use the region (e.g., `europe-west1`).

## Environment Variables

Services consume secrets through environment variables:

### Student Service

```yaml
env:
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: jwt-secret
        key: jwt-secret

  - name: DB_USER
    valueFrom:
      secretKeyRef:
        name: student-db-secret
        key: username

  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: student-db-secret
        key: password

  - name: DB_NAME
    valueFrom:
      secretKeyRef:
        name: student-db-secret
        key: database
```

### Project Service

```yaml
env:
  - name: DB_USER
    valueFrom:
      secretKeyRef:
        name: project-db-secret
        key: username

  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: project-db-secret
        key: password

  - name: DB_NAME
    valueFrom:
      secretKeyRef:
        name: project-db-secret
        key: database
```

## Security Best Practices

### General

1. **Never commit secrets to version control**
   - Use `.gitignore` for any files containing secrets
   - Use environment-specific configuration

2. **Use strong, random secrets**
   - Minimum 256-bit entropy for JWT secrets
   - Use cryptographically secure random generators

3. **Use URL-safe passwords for database credentials**
   - Database passwords are embedded in connection DSN URLs
   - Special characters like `@`, `:`, `/` can break URL parsing
   - Terraform is configured with `override_special = "_-"` for URL-safe passwords

4. **Rotate secrets regularly**
   - Implement secret rotation policy
   - Document rotation procedures

### Kind (Development)

1. **Don't use production secrets**
   - Use different secrets for development
   - Auto-generated secrets are fine for local testing

2. **Clean up when done**
   - Delete Kind cluster when not in use
   - Don't persist sensitive data

### GKE (Production)

1. **Use Google Secret Manager**
   - Centralized, encrypted storage
   - Audit logging enabled
   - IAM-based access control

2. **Limit access with IAM**
   - Use least privilege principle
   - Grant access only to specific service accounts
   - Review access regularly

3. **Enable audit logging**
   - Monitor secret access
   - Alert on unusual patterns
   - Keep audit logs for compliance

4. **Use Workload Identity**
   - No service account keys in cluster
   - Automatic credential rotation
   - Fine-grained IAM permissions

5. **Implement secret rotation**
   - Regular rotation schedule
   - Automated rotation where possible
   - Zero-downtime rotation strategy

## Troubleshooting

### Kind Issues

**Secrets not found:**
```bash
# Check if secrets exist
kubectl get secrets -n apps

# Regenerate secrets
make secrets/generate-kind
```

**Permission denied:**
```bash
# Ensure namespace exists
kubectl create namespace apps

# Check RBAC permissions
kubectl auth can-i create secrets -n apps
```

### GKE Issues

**External Secrets not syncing:**
```bash
# Check External Secrets Operator logs
kubectl logs -n external-secrets-system deployment/external-secrets

# Check ExternalSecret status
kubectl describe externalsecret -n apps

# Check SecretStore status
kubectl describe secretstore -n apps
```

**Permission denied accessing GSM:**
```bash
# Verify service account has access
gcloud secrets get-iam-policy grud-jwt-secret

# Grant access if missing
make secrets/grant-access-gke
```

**Workload Identity not working:**
```bash
# Verify annotation on service account
kubectl get serviceaccount student-service -n apps -o yaml | grep gcp-service-account

# Verify GCP service account binding
gcloud iam service-accounts get-iam-policy \
  grud-sa@PROJECT_ID.iam.gserviceaccount.com
```

## References

- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Google Secret Manager](https://cloud.google.com/secret-manager/docs)
- [External Secrets Operator](https://external-secrets.io/)
- [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
