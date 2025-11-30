package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var (
	sharedContainer *PostgresContainer
	sharedOnce      sync.Once
	sharedMu        sync.Mutex
)

// PostgresContainer wraps the postgres testcontainer
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	DB        *bun.DB
	DSN       string
}

// SetupSharedPostgres creates a single PostgreSQL container shared across all tests
// This is the RECOMMENDED approach for all tests - much faster than isolated containers
//
// IMPORTANT: Tests using shared container CANNOT run in parallel!
//
// Usage:
//
//	func TestMyService(t *testing.T) {
//	    pgContainer := testdb.SetupSharedPostgres(t)
//	    defer pgContainer.Cleanup(t)  // ← Only call once at top level
//
//	    pgContainer.RunMigrations(t, (*MyModel)(nil))
//
//	    t.Run("Test1", func(t *testing.T) {
//	        testdb.CleanupTables(t, pgContainer.DB, "my_table")
//	        // ... test
//	    })
//	}
func SetupSharedPostgres(t *testing.T) *PostgresContainer {
	t.Helper()

	sharedOnce.Do(func() {
		ctx := context.Background()
		pgContainer, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2),
			),
		)
		require.NoError(t, err)

		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, err)

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(connStr)))
		db := bun.NewDB(sqldb, pgdialect.New())

		err = db.Ping()
		require.NoError(t, err)

		sharedContainer = &PostgresContainer{
			Container: pgContainer,
			DB:        db,
			DSN:       connStr,
		}
	})

	return sharedContainer
}

func (pc *PostgresContainer) Cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if pc.DB != nil {
		pc.DB.Close()
	}

	if pc.Container != nil {
		if err := pc.Container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}
}

func (pc *PostgresContainer) RunMigrations(t *testing.T, models ...interface{}) {
	t.Helper()
	ctx := context.Background()

	// Create tables
	for _, model := range models {
		_, err := pc.DB.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		require.NoError(t, err, "failed to create table")
	}

	// Create trigger function for updated_at (for models with timestamps)
	_, err := pc.DB.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';
	`)
	require.NoError(t, err, "failed to create trigger function")
}

func (pc *PostgresContainer) CreateUpdateTrigger(t *testing.T, tableName string) {
	t.Helper()
	ctx := context.Background()

	query := fmt.Sprintf(`
		DROP TRIGGER IF EXISTS update_%s_updated_at ON %s;
		CREATE TRIGGER update_%s_updated_at
			BEFORE UPDATE ON %s
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();
	`, tableName, tableName, tableName, tableName)

	_, err := pc.DB.ExecContext(ctx, query)
	require.NoError(t, err, "failed to create trigger for table %s", tableName)
}

func CleanupTables(t *testing.T, db *bun.DB, tables ...string) {
	t.Helper()

	ctx := context.Background()

	for _, table := range tables {
		_, err := db.ExecContext(ctx, "TRUNCATE "+table+" RESTART IDENTITY CASCADE")
		require.NoError(t, err, "failed to truncate table: %s", table)
	}
}
