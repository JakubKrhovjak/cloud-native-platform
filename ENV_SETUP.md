# Environment Setup Guide

## 📋 Available Environments

- **local** - Pro development v IDE (localhost)
- **qa** - Pro testování v Dockeru (docker hostnames)
- **prod** - Pro production (nastaví se při deployi)

## 🚀 Quick Start

### Pro tebe (IDE Development - local profile)

```bash
# 1. Spusť databáze
docker-compose up postgres postgres_projects -d

# 2. Spusť project-service v IDE
cd project-service
# IDE automaticky načte .env.local (nebo přidej run configuration)
go run ./cmd/server

# 3. Spusť student-service v IDE
cd student-service
go run ./cmd/server
```

### Pro mě (Docker QA Testing)

```bash
# Spustí všechno v Dockeru s QA profilem
ENV=qa docker-compose up -d

# Nebo bez ENV (defaultně použije qa)
docker-compose up -d
```

## 📁 Structure

```
project-service/
├── .env.local       ← localhost URLs (pro IDE)
├── .env.qa          ← docker URLs (pro Docker)
└── .env.example     ← template

student-service/
├── .env.local       ← localhost URLs (pro IDE)
├── .env.qa          ← docker URLs (pro Docker)
└── .env.example     ← template
```

## 🔧 Configuration Differences

| Config | Local (IDE) | QA (Docker) |
|--------|-------------|-------------|
| **DB Host** | `localhost:5439/5440` | `postgres:5432` |
| **Project Service** | `http://localhost:8081` | `http://project_api:8081` |
| **gRPC** | `localhost:9090` | `project_api:9090` |

## 💡 Tips

### Spustit s jiným profilem
```bash
# Local profile
ENV=local docker-compose up -d

# QA profile (default)
docker-compose up -d
```

### Změnit config
```bash
# Pro local development (IDE)
vim project-service/.env.local

# Pro Docker testing
vim project-service/.env.qa
```

## 🎯 Best Practices

✅ **DO:**
- Commit `.env.local` and `.env.qa` (non-sensitive defaults)
- Use `.env.local` when running in IDE
- Use `.env.qa` for Docker testing

❌ **DON'T:**
- Don't commit `.env` or `*.env.prod`
- Don't hardcode URLs in code
- Don't mix local and docker hostnames

## 🐛 Troubleshooting

### "Connection refused" v IDE
→ Používáš `.env.local`? Mělo by být `localhost` ne `project_api`

### "No such host" v Dockeru
→ Docker používá `.env.qa`? Mělo by být `project_api` ne `localhost`

### Změnit mezi profily
```bash
# Restartuj s novým profilem
ENV=local docker-compose down
ENV=local docker-compose up -d
```
