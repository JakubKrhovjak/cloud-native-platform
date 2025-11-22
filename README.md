# Student Management REST API

REST API pro správu studentů univerzity s PostgreSQL databází a Bun ORM.

## Technologie

- Go 1.25
- PostgreSQL 16
- Bun ORM
- Gorilla Mux (HTTP router)
- Zap Logger (strukturované logování)
- Docker & Docker Compose

## Architektura

Projekt používá **Domain-Driven Design (DDD)** strukturu:

```
grud/
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
│   │   └── logger.go            # Zap logger konfigurace
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
└── docker-compose.yml
```

### Výhody této architektury:

- **Separation of Concerns** - každá vrstva má svou zodpovědnost
- **Testovatelnost** - snadné mockování interfaces
- **Škálovatelnost** - snadné přidávání nových domén
- **Maintainability** - čistý a přehledný kód
- **Professional** - standard pro enterprise projekty

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

## Instalace a spuštění

### Předpoklady
- Docker
- Docker Compose

### Spuštění s Docker Compose

```bash
docker-compose up -d
```

Tento příkaz:
- Stáhne PostgreSQL 16 Alpine image
- Vytvoří databázi `university` na portu 5439
- Sestaví a spustí Go API na portu 8080
- Automaticky provede migrace (vytvoří tabulku `students`)

### Kontrola běžících služeb

```bash
docker-compose ps
```

### Zobrazení logů

```bash
# Všechny služby
docker-compose logs -f

# Pouze API
docker-compose logs -f api

# Pouze databáze
docker-compose logs -f postgres
```

### Zastavení služeb

```bash
docker-compose down
```

### Zastavení a smazání dat

```bash
docker-compose down -v
```

## Lokální vývoj (bez Dockeru)

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

### Spuštění PostgreSQL

```bash
docker run --name postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=university -p 5439:5432 -d postgres:16-alpine
```

### Spuštění aplikace

```bash
go run cmd/server/main.go
```

### Build

```bash
go build -o server cmd/server/main.go
./server
```

## Testování API

### Pomocí curl

#### Vytvořit studenta
```bash
curl -X POST http://localhost:8080/api/students \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jan",
    "last_name": "Novák",
    "email": "jan.novak@university.cz",
    "major": "Computer Science",
    "year": 2
  }'
```

#### Získat všechny studenty
```bash
curl http://localhost:8080/api/students
```

#### Získat studenta podle ID
```bash
curl http://localhost:8080/api/students/1
```

#### Aktualizovat studenta
```bash
curl -X PUT http://localhost:8080/api/students/1 \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jan",
    "last_name": "Novák",
    "email": "jan.novak@university.cz",
    "major": "Software Engineering",
    "year": 3
  }'
```

#### Smazat studenta
```bash
curl -X DELETE http://localhost:8080/api/students/1
```

## Konfigurace

Aplikace používá proměnné prostředí pro konfiguraci:

| Proměnná | Výchozí hodnota | Popis |
|----------|----------------|-------|
| DB_HOST | localhost | Hostname PostgreSQL |
| DB_PORT | 5439 | Port PostgreSQL |
| DB_USER | postgres | Databázový uživatel |
| DB_PASSWORD | postgres | Databázové heslo |
| DB_NAME | university | Název databáze |
| PORT | 8080 | Port API serveru |
| ENV | development | Environment (development/production) |

## Domain Layers

### Model (model.go)
- Definice entity Student
- Bun tagy pro ORM mapping
- JSON tagy pro API response

### Repository (repository.go)
- Interface pro DB operace
- CRUD metody s Bun ORM
- Vrací Go errors (sql.ErrNoRows)

### Service (service.go)
- Business logika
- Validace vstupů
- Error handling
- Transformace repository errors na domain errors
- **Strukturované logování** každé operace (Zap)

### HTTP (http.go)
- REST handlers
- Request/Response mapping
- HTTP status codes
- Error responses

## Error Handling

Aplikace používá vrstvené error handling:
- **Repository**: vrací database errors
- **Service**: transformuje na domain errors (ErrStudentNotFound, ErrInvalidInput)
- **HTTP**: mapuje na HTTP status codes (404, 400, 500)

## Graceful Shutdown

Aplikace podporuje graceful shutdown:
- Catch SIGINT/SIGTERM signály
- 10 sekundový timeout pro dokončení požadavků
- Bezpečné uzavření databázového spojení

## Přidání nové domény

Pro přidání nové domény (např. `book`):

1. Vytvoř adresář `internal/book/`
2. Vytvoř soubory:
   - `model.go` - definice entity
   - `repository.go` - DB operace
   - `service.go` - business logika
   - `http.go` - HTTP handlers
3. Zaregistruj v `internal/app/app.go`
4. Přidej migrace do `db.RunMigrations()`

## Troubleshooting

### Problém s připojením k databázi

```bash
docker-compose logs postgres
```

### Port již používán

Změň port v `docker-compose.yml`:
```yaml
ports:
  - "5440:5432"  # Pro PostgreSQL
  - "8081:8080"  # Pro API
```

### Rebuild Docker image

```bash
docker-compose up -d --build
```

## Přímý přístup k databázi

```bash
docker exec -it university_db psql -U postgres -d university
```

SQL příkazy:
```sql
-- Zobrazit všechny studenty
SELECT * FROM students;

-- Vytvořit studenta
INSERT INTO students (first_name, last_name, email, major, year)
VALUES ('Jan', 'Novák', 'jan@example.com', 'CS', 2);

-- Smazat všechny studenty
TRUNCATE TABLE students RESTART IDENTITY;
```

## Logování

Aplikace používá **Zap logger** od Uberu pro strukturované logování.

### Konfigurace Loggeru

- **Development mode** (výchozí): barevný výstup, debug úroveň, čitelný formát
- **Production mode** (ENV=production): JSON formát, optimalizováno pro výkon

### Co se loguje

Service vrstva loguje každou operaci:

- **Info**: úspěšné operace
  - Vytvoření studenta s emailem a ID
  - Načtení studentů (včetně počtu)
  - Update a delete operace

- **Warn**: validační chyby, neexistující záznamy
  - Neplatné ID
  - Student nebyl nalezen

- **Error**: databázové chyby, validační selhání
  - Chyby při komunikaci s DB
  - Validační chyby při vytváření/updatu

### Příklad logů

```json
{
  "level": "info",
  "timestamp": "2024-01-15T14:30:00.123Z",
  "caller": "student/service.go:39",
  "msg": "creating student",
  "email": "jan.novak@university.cz",
  "first_name": "Jan",
  "last_name": "Novák"
}

{
  "level": "info",
  "timestamp": "2024-01-15T14:30:00.456Z",
  "caller": "student/service.go:61",
  "msg": "student created successfully",
  "id": 1,
  "email": "jan.novak@university.cz"
}
```

## Best Practices

1. **Dependency Injection** - všechny dependencies jsou injectované přes konstruktory
2. **Interface segregation** - každá vrstva definuje své interface
3. **Error wrapping** - použití `fmt.Errorf` s `%w` pro error wrapping
4. **Context propagation** - context.Context je předáván přes všechny vrstvy
5. **Validation** - validace na service vrstvě, ne v handleru
6. **Separation of concerns** - každá vrstva má svou zodpovědnost
7. **Structured logging** - Zap logger pro strukturované a výkonné logování
