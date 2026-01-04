# Student Service

Mikroservis pro správu studentů univerzity s PostgreSQL databází a Bun ORM.

## Technologie

- Go 1.25
- PostgreSQL 16
- Bun ORM
- Gorilla Mux (HTTP router)
- Slog Logger (strukturované logování)
- Docker & Docker Compose

## Architektura

Projekt používá Domain-Driven Design (DDD) strukturu:

```
student-service/
├── cmd/
│   └── server/
│       └── main.go              # Entry point s graceful shutdown
│
├── internal/
│   ├── config/
│   │   └── config.go            # Konfigurace aplikace
│   │
│   ├── db/
│   │   └── db.go                # Databázové připojení a migrace
│   │
│   ├── logger/
│   │   └── logger.go            # Slog logger konfigurace
│   │
│   ├── student/                 # STUDENT DOMÉNA
│   │   ├── model.go             # Student entity
│   │   ├── repository.go        # DB operace
│   │   ├── service.go           # Business logika s logováním
│   │   └── http.go              # HTTP handlers
│   │
│   └── app/
│       └── app.go               # Bootstrap aplikace
│
├── go.mod
├── Dockerfile
└── README.md
```

## Student Model

```json
{
  "id": 1,
  "first_name": "Jan",
  "last_name": "Novák",
  "email": "jan.novak@university.cz",
  "major": "Computer Science",
  "year": 2
}
```

## API Endpointy

### Vytvořit studenta
```bash
POST /api/students
Content-Type: application/json

{
  "first_name": "Jan",
  "last_name": "Novák",
  "email": "jan.novak@university.cz",
  "major": "Computer Science",
  "year": 2
}
```

### Získat všechny studenty
```bash
GET /api/students
```

### Získat studenta podle ID
```bash
GET /api/students/{id}
```

### Aktualizovat studenta
```bash
PUT /api/students/{id}
Content-Type: application/json

{
  "first_name": "Jan",
  "last_name": "Novák",
  "email": "jan.novak@university.cz",
  "major": "Software Engineering",
  "year": 3
}
```

### Smazat studenta
```bash
DELETE /api/students/{id}
```

## Validace

Service vrstva obsahuje validaci:
- First name a last name jsou povinné
- Email musí být validní formát
- Year musí být mezi 0-10
- Email musí být unikátní (DB constraint)

## Lokální vývoj

### Předpoklady
- Go 1.25+
- PostgreSQL

### Instalace závislostí

```bash
go mod download
```

### Nastavení proměnných prostředí

```bash
export DB_HOST=localhost
export DB_PORT=5439
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=university
export PORT=8080
```

### Spuštění aplikace

```bash
go run cmd/student-service/main.go
```

### Build

```bash
go build -o student-service cmd/student-service/main.go
./student-service
```

## Konfigurace

Aplikace používá proměnné prostředí:

| Proměnná | Výchozí hodnota | Popis |
|----------|----------------|-------|
| DB_HOST | localhost | Hostname PostgreSQL |
| DB_PORT | 5439 | Port PostgreSQL |
| DB_USER | postgres | Databázový uživatel |
| DB_PASSWORD | postgres | Databázové heslo |
| DB_NAME | university | Název databáze |
| PORT | 8080 | Port API serveru |
| ENV | development | Environment |

## Domain Layers

### Model (model.go)
- Definice entity Student
- Bun tagy pro ORM mapping
- JSON tagy pro API response

### Repository (repository.go)
- Interface pro DB operace
- CRUD metody s Bun ORM
- Vrací Go errors

### Service (service.go)
- Business logika
- Validace vstupů
- Error handling
- Strukturované logování

### HTTP (http.go)
- REST handlers
- Request/Response mapping
- HTTP status codes

## Error Handling

Vrstvené error handling:
- **Repository**: database errors
- **Service**: domain errors (ErrStudentNotFound, ErrInvalidInput)
- **HTTP**: HTTP status codes (404, 400, 500)

## Logování

Aplikace používá Slog logger pro strukturované logování.

Service vrstva loguje každou operaci:
- **Info**: úspěšné operace
- **Warn**: validační chyby, neexistující záznamy
- **Error**: databázové chyby

## Best Practices

1. Dependency Injection
2. Interface segregation
3. Error wrapping
4. Context propagation
5. Validation na service vrstvě
6. Structured logging
7. Graceful shutdown
