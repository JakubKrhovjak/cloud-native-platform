# Admin Panel

React-based admin panel for the GRUD application.

## Tech Stack

- React 19
- TypeScript
- Vite
- Material UI
- React Router
- React Hook Form
- Axios

## Features

- **Student Management** - List, create, update, delete students
- **Authentication** - Login with JWT tokens
- **Responsive Design** - Material UI components
- **Form Validation** - React Hook Form with validation
- **API Integration** - Axios HTTP client with JWT auth

## Development

```bash
# Install dependencies
npm install

# Start dev server (hot reload)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

Dev server runs on: http://localhost:5173

## Environment Variables

Create `.env` file:

```env
VITE_API_URL=http://localhost:8080
```

- **Local development**: `http://localhost:8080` (student-service)
- **Kubernetes**: `http://student-service.grud.svc.cluster.local:8080`

## Kubernetes Deployment

### 1. Build Docker Image

```bash
# From services/admin directory
docker build -t admin-panel:latest .
```

### 2. Load to Kind (for local testing)

```bash
kind load docker-image admin-panel:latest --name grud-cluster
```

### 3. Deploy with Helm

The admin panel is deployed as part of the main Helm chart:

```bash
# From project root
make kind/deploy
```

### 4. Access the Admin Panel

```bash
# Get NodePort
kubectl get svc admin-panel -n grud

# Access via NodePort
# http://localhost:9080
```

Or use port-forward:
```bash
kubectl port-forward -n grud svc/admin-panel 9080:80
```

## Project Structure

```
services/admin/
├── public/              # Static assets
├── src/
│   ├── components/      # React components
│   ├── pages/           # Page components
│   ├── services/        # API services
│   ├── hooks/           # Custom React hooks
│   ├── types/           # TypeScript types
│   ├── App.tsx          # Main app component
│   └── main.tsx         # Entry point
├── index.html
├── vite.config.ts       # Vite configuration
├── tsconfig.json        # TypeScript configuration
├── Dockerfile           # Multi-stage Docker build
└── package.json
```

## API Integration

The admin panel communicates with student-service:

```typescript
// src/services/api.ts
import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // Send cookies
});

// Add JWT token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

## Authentication

JWT tokens are stored in:
1. **localStorage** - For API requests
2. **HTTP-only cookies** - Set by backend

Login flow:
1. User submits credentials
2. Admin panel calls `/api/auth/login`
3. Backend returns JWT token + sets HTTP-only cookie
4. Admin panel stores token in localStorage
5. All subsequent requests include token in `Authorization` header

Logout flow:
1. Admin panel calls `/api/auth/logout`
2. Remove token from localStorage
3. Backend clears cookie

## Production Build

### Build Configuration

The Dockerfile uses multi-stage build:

```dockerfile
# Stage 1: Build
FROM node:18-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Serve with nginx
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Environment-specific Builds

For production:

```bash
# Build with production API URL
VITE_API_URL=https://api.example.com npm run build
```

In Kubernetes, the API URL is configured via ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: admin-panel-config
data:
  config.js: |
    window.ENV = {
      VITE_API_URL: 'http://student-service:8080'
    }
```

## Development Tips

### Hot Reload

Vite provides instant hot module replacement (HMR):
- Edit a component → see changes immediately
- No full page reload needed

### TypeScript

All components use TypeScript for type safety:

```typescript
interface Student {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  major: string;
  year: number;
}
```

### Material UI

Using Material UI components:

```tsx
import { Button, TextField, Card } from '@mui/material';

function StudentForm() {
  return (
    <Card>
      <TextField label="First Name" />
      <Button variant="contained">Save</Button>
    </Card>
  );
}
```

## Quick Commands (from project root)

```bash
# Start dev server
make admin-dev

# Build for production
make admin-build

# Install dependencies
make admin-install
```

## Troubleshooting

### API Connection Issues

If admin panel can't connect to student-service:

1. **Check API URL**:
   ```bash
   echo $VITE_API_URL
   ```

2. **Verify student-service is running**:
   ```bash
   curl http://localhost:8080/health/live
   ```

3. **Check CORS settings** in student-service

4. **View browser console** for error messages

### Authentication Issues

If login fails:

1. **Check credentials** - Default: test@example.com / password123
2. **Verify JWT_SECRET** is set in student-service
3. **Check browser cookies** - Should see HTTP-only cookie
4. **Check localStorage** - Should have `token` key

### Build Issues

```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install

# Clear Vite cache
rm -rf node_modules/.vite
npm run dev
```

## License

This project is for educational purposes.
