# Student Service

Microservice for university student management with PostgreSQL database, JWT authentication, gRPC client, and NATS event producer.

## Technologies

- Go 1.25
- PostgreSQL 16
- Bun ORM
- Gorilla Mux (HTTP router)
- JWT Authentication
- gRPC Client (calls project-service)
- NATS Producer (publishes events)
- OpenTelemetry (tracing + metrics)
- Slog Logger (structured logging)

## Architecture

The project uses Domain-Driven Design (DDD) structure:

```
student-service/
├── cmd/
│   └── server/
│       └── main.go              # Entry point with graceful shutdown
│
├── internal/
│   ├── config/
│   │   └── config.go            # Application configuration
│   │
│   ├── db/
│   │   └── db.go                # Database connection and migrations
│   │
│   ├── logger/
│   │   └── logger.go            # Slog logger configuration
│   │
│   ├── student/                 # STUDENT DOMAIN
│   │   ├── model.go             # Student entity
│   │   ├── repository.go        # DB operations
│   │   ├── service.go           # Business logic with logging
│   │   └── http.go              # HTTP handlers
│   │
│   ├── auth/                    # AUTHENTICATION DOMAIN
│   │   ├── model.go             # User entity
│   │   ├── repository.go        # User DB operations
│   │   ├── service.go           # JWT token generation/validation
│   │   ├── http.go              # Login/logout endpoints
│   │   └── middleware.go        # JWT authentication middleware
│   │
│   └── app/
│       └── app.go               # Application bootstrap
│
├── configs/
│   ├── config.local.yaml        # Local development
│   └── config.kind.yaml         # Kind Kubernetes
│
├── go.mod
└── Dockerfile
```

## Features

1. **Student Management** - CRUD operations for students
2. **JWT Authentication** - Secure token-based auth with HTTP-only cookies
3. **gRPC Integration** - Calls project-service to fetch student's projects
4. **NATS Events** - Publishes `student.viewed` events when student is accessed
5. **OpenTelemetry** - Distributed tracing across HTTP, gRPC, and NATS
6. **Structured Logging** - JSON logs with trace IDs
7. **Health Checks** - Liveness and readiness endpoints

## Student Model

```json
{
  "id": 1,
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@university.com",
  "major": "Computer Science",
  "year": 2,
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
```

## API Endpoints

### Authentication

#### Login
```bash
POST /api/auth/login
Content-Type: application/json

{
  "email": "test@example.com",
  "password": "password123"
}

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "test@example.com",
    "firstName": "Test",
    "lastName": "User"
  }
}
```

JWT token is also set as HTTP-only cookie.

#### Logout
```bash
POST /api/auth/logout
Authorization: Bearer <token>
```

### Student Management

All student endpoints require JWT authentication.

#### Create Student
```bash
POST /api/students
Authorization: Bearer <token>
Content-Type: application/json

{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@university.com",
  "major": "Computer Science",
  "year": 2
}
```

#### Get All Students
```bash
GET /api/students
Authorization: Bearer <token>
```

#### Get Student by ID
```bash
GET /api/students/{id}
Authorization: Bearer <token>
```

This endpoint:
1. Fetches student from database
2. Calls project-service via gRPC to get student's projects
3. Publishes `student.viewed` event to NATS
4. Returns student with projects

#### Update Student
```bash
PUT /api/students/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@university.com",
  "major": "Software Engineering",
  "year": 3
}
```

#### Delete Student
```bash
DELETE /api/students/{id}
Authorization: Bearer <token>
```

### Health Checks

#### Liveness
```bash
GET /health/live
```

Returns 200 if service is running.

#### Readiness
```bash
GET /health/ready
```

Returns 200 if service is ready (database connected).

## Communication

### gRPC Client

Calls project-service to fetch projects:

```go
// Internal call when GET /api/students/{id}
projects, err := s.projectClient.GetProjectsByStudent(ctx, studentID)
```

gRPC endpoint: `project-service:9090` (Kubernetes) or `localhost:9090` (local)

### NATS Producer

Publishes events when student is viewed:

```json
{
  "student_id": 42,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Topic: `student.viewed`

## Validation

Service layer validates:
- First name and last name are required
- Email must be valid format
- Year must be between 1-10
- Email must be unique (DB constraint)

## Local Development

### Prerequisites
- Go 1.25+
- PostgreSQL

### Install Dependencies

```bash
go mod download
```

### Environment Variables

Required:
- `ENV=local` - Loads config.local.yaml
- `JWT_SECRET=your-secret-key` - JWT signing secret

Optional:
- `LOG_LEVEL=debug` - Log level (debug/info/warn/error)

### Start Dependencies

```bash
# Start PostgreSQL, NATS, and OTEL collector
docker-compose up postgres nats -d
```

### Run Service

```bash
# From project root
ENV=local JWT_SECRET=your-secret go run ./services/student-service/cmd/server
```

Service starts on port 8080.

### Configuration

Edit `configs/config.local.yaml`:

```yaml
server:
  port: 8080
  shutdownTimeout: 30s

database:
  host: localhost
  port: 5439
  user: postgres
  password: postgres
  database: university

projectService:
  grpcAddress: localhost:9090

nats:
  url: nats://localhost:4222

otel:
  endpoint: http://localhost:4317
  insecure: true
```

## Domain Layers

### Model (model.go)
- Entity definitions (Student, User)
- Bun tags for ORM mapping
- JSON tags for API response

### Repository (repository.go)
- Interface for DB operations
- CRUD methods with Bun ORM
- Returns Go errors

### Service (service.go)
- Business logic
- Input validation
- Error handling
- gRPC client calls
- NATS event publishing
- Structured logging

### HTTP (http.go)
- REST handlers
- Request/Response mapping
- HTTP status codes
- JWT middleware

## Error Handling

Layered error handling:
- **Repository**: Database errors
- **Service**: Domain errors (ErrStudentNotFound, ErrInvalidInput, ErrUnauthorized)
- **HTTP**: HTTP status codes (401, 404, 400, 500)

## Logging

The application uses Slog for structured JSON logging.

Service layer logs every operation:
- **Info**: Successful operations
- **Warn**: Validation errors, not found records
- **Error**: Database errors, gRPC errors, NATS errors

All logs include `trace_id` for correlation with distributed traces.

## Observability

### Metrics

Exposed via OpenTelemetry:
- HTTP request rate, latency, errors
- gRPC client call rate, latency, errors
- NATS publish rate, latency, errors
- Database connection pool stats
- Go runtime metrics (goroutines, memory, GC)

View in Prometheus: http://localhost:30090

### Distributed Tracing

All operations include trace context:
- HTTP request → gRPC call → NATS publish

View traces in Grafana: http://localhost:30300

### Logs

JSON logs with trace IDs:
```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "student fetched successfully",
  "trace_id": "abc123...",
  "student_id": 42
}
```

View logs in Loki (Grafana Explore).

## Testing

```bash
# Run all tests
make test

# Run only student-service tests
go test ./services/student-service/...

# With verbose output
go test -v ./services/student-service/...
```

Tests use shared PostgreSQL and NATS containers for fast execution (~2-3s).

## Best Practices

1. **Dependency Injection** - Dependencies injected via constructors
2. **Interface Segregation** - Each layer defines its own interfaces
3. **Error Wrapping** - Using `fmt.Errorf` with `%w`
4. **Context Propagation** - context.Context in every method
5. **Validation** - Validation at service layer
6. **Structured Logging** - Structured logging with trace IDs
7. **Graceful Shutdown** - Safe service termination
8. **OpenTelemetry** - Distributed tracing across all communication
