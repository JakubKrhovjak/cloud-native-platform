# University Management Microservices

Microservice architecture for university management with PostgreSQL databases and Bun ORM.

## Quick Start

### Run Tests
```bash
make test              # Fast tests (shared container, ~5s)
./scripts/test-all.sh  # Alternative bash script
```

### Run Services
```bash
docker-compose up -d   # Start all services
```

See [TESTING.md](TESTING.md) for complete testing guide.

## Architecture

The project consists of two independent microservices:

1. **student-service** - student management
2. **project-service** - project management

Each microservice has:
- Its own PostgreSQL database
- Its own API server
- Independent deployment
- Domain-Driven Design (DDD) structure

## Technologies

- Go 1.25
- PostgreSQL 16
- Bun ORM
- Gorilla Mux (HTTP router)
- Zap/Slog Logger (structured logging)
- Docker & Docker Compose

## Services

### Student Service
- Port: `8080`
- Database: `university` (port `5439`)
- Endpoints: `/api/students`

### Project Service
- Port: `8081`
- Database: `projects` (port `5440`)
- Endpoints: `/api/projects`

## Project Structure

```
grud/
├── student-service/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── db/
│   │   ├── logger/
│   │   ├── student/
│   │   │   ├── model.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── http.go
│   │   └── app/
│   ├── go.mod
│   └── Dockerfile
│
├── project-service/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── db/
│   │   ├── logger/
│   │   ├── project/
│   │   │   ├── model.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── http.go
│   │   └── app/
│   ├── go.mod
│   └── Dockerfile
│
├── docker-compose.yml
├── go.work
└── README.md
```

## Installation and Setup

### Prerequisites
- Docker
- Docker Compose

### Start All Services

```bash
docker-compose up -d
```

This command starts:
- PostgreSQL database for students (port 5439)
- PostgreSQL database for projects (port 5440)
- Student API (port 8080)
- Project API (port 8081)

### Check Running Services

```bash
docker-compose ps
```

### View Logs

```bash
# All services
docker-compose logs -f

# Student service only
docker-compose logs -f student_api

# Project service only
docker-compose logs -f project_api

# Databases
docker-compose logs -f postgres
docker-compose logs -f postgres_projects
```

### Stop Services

```bash
docker-compose down
```

### Stop and Remove Data

```bash
docker-compose down -v
```

## Student Service API

### Student Model
```json
{
  "id": 1,
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@university.com",
  "major": "Computer Science",
  "year": 2
}
```

### Endpoints

#### Create Student
```bash
POST http://localhost:8080/api/students
Content-Type: application/json

{
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@university.com",
  "major": "Computer Science",
  "year": 2
}
```

#### Get All Students
```bash
GET http://localhost:8080/api/students
```

#### Get Student by ID
```bash
GET http://localhost:8080/api/students/{id}
```

#### Update Student
```bash
PUT http://localhost:8080/api/students/{id}
Content-Type: application/json

{
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@university.com",
  "major": "Software Engineering",
  "year": 3
}
```

#### Delete Student
```bash
DELETE http://localhost:8080/api/students/{id}
```

## Project Service API

### Project Model
```json
{
  "id": 1,
  "name": "Web Application",
  "description": "Modern web app with Go backend",
  "status": "in_progress",
  "start_date": "2024-01-15T00:00:00Z",
  "end_date": null
}
```

### Endpoints

#### Create Project
```bash
POST http://localhost:8081/api/projects
Content-Type: application/json

{
  "name": "Web Application",
  "description": "Modern web app with Go backend",
  "status": "in_progress",
  "start_date": "2024-01-15T00:00:00Z"
}
```

#### Get All Projects
```bash
GET http://localhost:8081/api/projects
```

#### Get Project by ID
```bash
GET http://localhost:8081/api/projects/{id}
```

#### Update Project
```bash
PUT http://localhost:8081/api/projects/{id}
Content-Type: application/json

{
  "name": "Web Application",
  "description": "Updated description",
  "status": "completed",
  "start_date": "2024-01-15T00:00:00Z",
  "end_date": "2024-03-20T00:00:00Z"
}
```

#### Delete Project
```bash
DELETE http://localhost:8081/api/projects/{id}
```

## Local Development (without Docker)

### Student Service

```bash
cd student-service
go mod download
export DB_HOST=localhost
export DB_PORT=5439
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=university
export PORT=8080
go run cmd/server/main.go
```

### Project Service

```bash
cd project-service
go mod download
export DB_HOST=localhost
export DB_PORT=5440
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=projects
export PORT=8081
go run cmd/server/main.go
```

## Go Workspace

The project uses Go workspace for working with multiple modules:

```bash
# Update workspace
go work sync

# Build all services
go work use ./student-service ./project-service
```

## Microservices Architecture Benefits

- **Independent Deployment** - each service can be deployed separately
- **Scalability** - services can be scaled independently as needed
- **Technology Freedom** - each service can use different technologies
- **Fault Isolation** - failure of one service doesn't affect others
- **Team Autonomy** - different teams can work on different services
- **Database per Service** - each service has its own database

## Direct Database Access

### Student Database
```bash
docker exec -it university_db psql -U postgres -d university
```

### Project Database
```bash
docker exec -it projects_db psql -U postgres -d projects
```

## Domain-Driven Design

Each microservice uses DDD structure:

### Model Layer
- Entity definitions
- Bun tags for ORM mapping
- JSON tags for API response

### Repository Layer
- Interface for DB operations
- CRUD methods with Bun ORM
- Returns Go errors

### Service Layer
- Business logic
- Input validation
- Error handling
- Structured logging

### HTTP Layer
- REST handlers
- Request/Response mapping
- HTTP status codes

## Best Practices

1. **Dependency Injection** - dependencies are injected via constructors
2. **Interface Segregation** - each layer defines its own interfaces
3. **Error Wrapping** - using `fmt.Errorf` with `%w`
4. **Context Propagation** - context.Context in every method
5. **Validation** - validation at service layer
6. **Structured Logging** - structured logging for monitoring
7. **Graceful Shutdown** - safe service termination

## Troubleshooting

### Database Connection Issues

```bash
docker-compose logs postgres
docker-compose logs postgres_projects
```

### Port Already in Use

Change ports in `docker-compose.yml`

### Rebuild Docker Images

```bash
docker-compose up -d --build
```

### Reset Databases

```bash
docker-compose down -v
docker-compose up -d
```

## Testing

### Quick Commands

```bash
# Run all tests (default, fast)
make test                    # ~5s for 20 tests

# Test individual services
make test-student           # Student service only
make test-project           # Project service only

# Integration tests (isolated containers)
make test-integration       # ~40s (complete isolation)

# With coverage
make test-coverage          # Coverage report

# Help
make help                   # Show all commands
```

### Manual Testing

```bash
# All tests in monorepo
go test ./student-service/... ./project-service/...

# Specific service
go test ./student-service/...

# With verbose output
go test -v ./student-service/... ./project-service/...

# Integration tests
go test -tags=integration ./student-service/...
```

### Performance

**Shared Container Tests:**
```
Student Service:  2.6s  (10 tests)
Project Service:  2.4s  (10 tests)
Total:           ~5.0s  ✅ Fast!
```

**Integration Tests:**
```
Student Service:  20s   (10 tests)
Project Service:  18s   (10 tests)
Total:           ~40s   ⚠️  Slower but isolated
```

**Speedup: 8× faster with shared container!** 🚀

See [TESTING.md](TESTING.md) for complete guide.

## Documentation

- [TESTING.md](TESTING.md) - Complete testing guide
- [testing/TESTING_CONVENTIONS.md](testing/TESTING_CONVENTIONS.md) - Test naming conventions
- [testing/INDUSTRY_PATTERNS.md](testing/INDUSTRY_PATTERNS.md) - Industry testing patterns
- [testing/SPRING_VS_GO_TESTING.md](testing/SPRING_VS_GO_TESTING.md) - Spring vs Go comparison
- [testing/SHARED_POSTGRES_MIGRATION.md](testing/SHARED_POSTGRES_MIGRATION.md) - Migration guide
