# Admin Panel

React admin panel pro GRUD aplikaci.

## Tech Stack

- React 19
- TypeScript
- Vite
- Material UI
- React Router
- React Hook Form
- Axios

## Development

```bash
# Install dependencies
npm install

# Start dev server
npm run dev

# Build for production
npm run build
```

## Kubernetes Deployment

### 1. Build Docker image

```bash
# From services/admin directory
docker build -t admin-panel:latest .
```

### 2. Load to Kind (pro lokální testing)

```bash
kind load docker-image admin-panel:latest --name grud-cluster
```

### 3. Deploy to Kubernetes

```bash
# From k8s directory
kubectl apply -k base/

# Or using kustomize
kustomize build base/ | kubectl apply -f -
```

### 4. Access the admin panel

```bash
# Get node port
kubectl get svc admin-panel -n grud

# Access via NodePort
# http://localhost:30081
```

## Environment Variables

- `VITE_API_URL` - Student service API URL (default: http://localhost:9080)

## Production Build

Pro produkční build se používá `.env.production`:

```bash
VITE_API_URL=http://student-service.grud.svc.cluster.local:8080
```

## Quick Commands

```bash
# From root grud directory
make admin-dev        # Start dev server
make admin-build      # Build for production
make admin-install    # Install dependencies
```
