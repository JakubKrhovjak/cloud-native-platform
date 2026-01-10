#!/bin/bash

# Generate secrets for GRUD application
# Usage: ./scripts/generate-secrets.sh [kind|gke]

set -e

ENVIRONMENT=${1:-kind}
NAMESPACE=${2:-grud}

echo "🔐 Generating secrets for environment: $ENVIRONMENT"

# Generate random JWT secret (256-bit)
JWT_SECRET=$(openssl rand -base64 32)

# Generate random database passwords
STUDENT_DB_PASSWORD=$(openssl rand -base64 32)
PROJECT_DB_PASSWORD=$(openssl rand -base64 32)
a;
case $ENVIRONMENT in
  kind)
    echo "📦 Creating Kubernetes Secrets for Kind cluster..."

    # Create namespace if it doesn't exist
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

    # Create JWT secret
    kubectl create secret generic jwt-secret \
      --from-literal=jwt-secret="$JWT_SECRET" \
      --namespace=$NAMESPACE \
      --dry-run=client -o yaml | kubectl apply -f -

    # Create student database secret
    kubectl create secret generic student-db-secret \
      --from-literal=username=app \
      --from-literal=password="$STUDENT_DB_PASSWORD" \
      --from-literal=database=university \
      --namespace=$NAMESPACE \
      --dry-run=client -o yaml | kubectl apply -f -

    # Create project database secret
    kubectl create secret generic project-db-secret \
      --from-literal=username=app \
      --from-literal=password="$PROJECT_DB_PASSWORD" \
      --from-literal=database=projects \
      --namespace=$NAMESPACE \
      --dry-run=client -o yaml | kubectl apply -f -

    echo "✅ Secrets created in Kubernetes namespace: $NAMESPACE"
    echo ""
    echo "📝 JWT Secret (save this for development):"
    echo "   export JWT_SECRET=\"$JWT_SECRET\""
    ;;

  gke)
    echo "☁️  Creating secrets in Google Secret Manager..."

    # Check if gcloud is installed
    if ! command -v gcloud &> /dev/null; then
      echo "❌ gcloud CLI is not installed. Please install it first."
      exit 1
    fi

    # Get GCP project ID
    GCP_PROJECT=$(gcloud config get-value project)

    if [ -z "$GCP_PROJECT" ]; then
      echo "❌ GCP project not set. Run: gcloud config set project PROJECT_ID"
      exit 1
    fi

    echo "📍 Using GCP Project: $GCP_PROJECT"

    # Create JWT secret in Secret Manager
    echo "Creating grud-jwt-secret..."
    echo -n "$JWT_SECRET" | gcloud secrets create grud-jwt-secret \
      --data-file=- \
      --replication-policy=automatic \
      2>/dev/null || \
    echo -n "$JWT_SECRET" | gcloud secrets versions add grud-jwt-secret \
      --data-file=-

    # Create student database credentials
    echo "Creating grud-student-db-credentials..."
    cat <<EOF | gcloud secrets create grud-student-db-credentials \
      --data-file=- \
      --replication-policy=automatic \
      2>/dev/null || \
    cat <<EOF | gcloud secrets versions add grud-student-db-credentials \
      --data-file=-
{
  "username": "app",
  "password": "$STUDENT_DB_PASSWORD",
  "database": "university"
}
EOF

    # Create project database credentials
    echo "Creating grud-project-db-credentials..."
    cat <<EOF | gcloud secrets create grud-project-db-credentials \
      --data-file=- \
      --replication-policy=automatic \
      2>/dev/null || \
    cat <<EOF | gcloud secrets versions add grud-project-db-credentials \
      --data-file=-
{
  "username": "app",
  "password": "$PROJECT_DB_PASSWORD",
  "database": "projects"
}
EOF

    echo "✅ Secrets created in Google Secret Manager"
    echo ""
    echo "🔑 Grant access to secrets:"
    echo "   gcloud secrets add-iam-policy-binding grud-jwt-secret \\"
    echo "     --member='serviceAccount:grud-sa@$GCP_PROJECT.iam.gserviceaccount.com' \\"
    echo "     --role='roles/secretmanager.secretAccessor'"
    echo ""
    echo "   gcloud secrets add-iam-policy-binding grud-student-db-credentials \\"
    echo "     --member='serviceAccount:grud-sa@$GCP_PROJECT.iam.gserviceaccount.com' \\"
    echo "     --role='roles/secretmanager.secretAccessor'"
    echo ""
    echo "   gcloud secrets add-iam-policy-binding grud-project-db-credentials \\"
    echo "     --member='serviceAccount:grud-sa@$GCP_PROJECT.iam.gserviceaccount.com' \\"
    echo "     --role='roles/secretmanager.secretAccessor'"
    ;;

  *)
    echo "❌ Unknown environment: $ENVIRONMENT"
    echo "Usage: $0 [kind|gke]"
    exit 1
    ;;
esac

echo ""
echo "🎉 Done!"
