# Testing Guide

## Quick Start

```bash
# Run all tests (fastest, recommended)
make test

# Or using bash script
./scripts/test-all.sh

# Or manually
go test ./student-service/... ./project-service/...
```

## All Testing Options

### 1. Using Makefile (Recommended ⭐)

```bash
# Run all tests (shared container, fast)
make test                    # ~5s for 20 tests

# Test individual services
make test-student            # Student service only
make test-project            # Project service only

# Integration tests (isolated containers)
make test-integration        # ~40s (slow but isolated)

# All tests (shared + integration)
make test-all                # Runs both

# Tests with coverage
make test-coverage           # Shows coverage %

# Verbose output
make test-verbose            # See all test details

# Race detection
make test-race               # Find race conditions

# Clean cache
make clean                   # Clear test cache

# Help
make help                    # Show all commands
```

### 2. Using Bash Scripts

```bash
# Run all tests
./scripts/test-all.sh
```

### 3. Using Go Commands Directly

```bash
# All tests in monorepo
go test ./student-service/... ./project-service/...

# With verbose output
go test -v ./student-service/... ./project-service/...

# With coverage
go test -cover ./student-service/... ./project-service/...

# Specific service
go test ./student-service/...
go test ./project-service/...

# Specific package
go test ./student-service/internal/student
go test ./project-service/internal/project

# Integration tests (with build tag)
go test -tags=integration ./student-service/...
go test -tags=integration ./project-service/...

# Specific test
go test -run TestCreateStudent ./student-service/internal/student
go test -run TestProjectService_Shared/CreateProject ./project-service/internal/project
```

### 4. IDE Integration

#### GoLand / IntelliJ IDEA
1. Right-click on `grud` folder → "Run 'go test grud/...'"
2. Or use the test runner in each `*_test.go` file
3. Configure run configuration: "Run" → "Edit Configurations" → Add "Go Test"

#### VS Code
1. Install "Go" extension
2. Click "run test" or "debug test" above each test function
3. Or use Command Palette: "Go: Test All Packages in Workspace"

## Test Types

### Shared Container Tests (Default)

**When:** Local development, fast feedback
**Speed:** ~2.5s per service
**Run:** `make test` or `go test ./...`

```go
func TestStudentService_Shared(t *testing.T) {
    pgContainer := testdb.SetupSharedPostgres(t)
    defer pgContainer.Cleanup(t)
    // All subtests share one container
}
```

### Integration Tests (Build Tag)

**When:** CI/CD, complete isolation needed
**Speed:** ~2s per test (each gets own container)
**Run:** `make test-integration` or `go test -tags=integration ./...`

```go
//go:build integration
// +build integration

func TestCreateStudent(t *testing.T) {
    // Each test gets isolated container
}
```

## Performance Comparison

```
Shared Container Tests:
  Student Service:  2.6s  (10 tests)
  Project Service:  2.4s  (10 tests)
  Total:           ~5.0s  ✅ Fast!

Integration Tests:
  Student Service:  20s   (10 tests)
  Project Service:  18s   (10 tests)
  Total:           ~38s   ⚠️  Slow but isolated

Speedup: 7-8x faster with shared container! 🚀
```

## CI/CD Pipeline

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      # Fast tests (shared container)
      - name: Run tests
        run: make test

      # Full isolation tests
      - name: Run integration tests
        run: make test-integration

      # Coverage report
      - name: Coverage
        run: |
          go test -coverprofile=coverage.out ./student-service/... ./project-service/...
          go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

### GitLab CI Example

```yaml
test:
  stage: test
  image: golang:1.21
  script:
    - make test
    - make test-integration
  coverage: '/coverage: \d+\.\d+/'
```

## Troubleshooting

### Tests are slow
```bash
# Make sure you're using shared container (not integration tests)
make test          # ✅ Fast (shared)
make test-integration  # ❌ Slow (isolated)
```

### Tests fail with "container already exists"
```bash
# Clean up docker containers
docker ps -a | grep postgres | awk '{print $1}' | xargs docker rm -f

# Clean test cache
make clean
```

### Tests affect each other
```bash
# Make sure each subtest cleans tables
t.Run("Test", func(t *testing.T) {
    testdb.CleanupTables(t, pgContainer.DB, "students", "projects")
    // ...
})
```

### "no test files" warnings
This is normal - not all packages have tests. To hide these warnings:
```bash
go test ./student-service/... ./project-service/... 2>&1 | grep -v "no test files"
```

## Code Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./student-service/... ./project-service/...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

## Best Practices

### ✅ DO

1. **Use `make test` for daily development** - fastest feedback
2. **Run `make test-integration` before commits** - catch integration issues
3. **Clean tables in each subtest** - ensures isolation
4. **Use TestApp pattern** - simplifies component setup

### ❌ DON'T

1. **Don't use `t.Parallel()` with shared containers** - causes conflicts
2. **Don't forget to clean test cache** - can cause false positives
3. **Don't run integration tests in watch mode** - too slow

## Watch Mode (Auto-run on changes)

Requires [entr](https://github.com/eradman/entr):

```bash
# Install entr (macOS)
brew install entr

# Watch and auto-run tests
make test-watch

# Or manually
find . -name '*.go' | entr -c go test ./student-service/... ./project-service/...
```

## VS Code Test Explorer

Add to `.vscode/settings.json`:

```json
{
  "go.testOnSave": false,
  "go.testFlags": ["-v"],
  "go.testTimeout": "300s"
}
```

## Summary

**For daily development:**
```bash
make test              # ⚡ Fast (5s)
```

**Before committing:**
```bash
make test-all          # 🔒 Complete (shared + integration)
```

**In CI/CD:**
```bash
make test              # Fast check
make test-integration  # Full isolation
make test-coverage     # Track coverage
```

**Pro tips:**
- Use `make test` during development for instant feedback
- Run `make test-integration` before pushing to catch edge cases
- Keep tests fast by using shared containers
- Use TestApp pattern to avoid boilerplate
