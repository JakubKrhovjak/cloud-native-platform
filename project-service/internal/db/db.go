package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"

	"project-service/internal/config"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func New(cfg config.DatabaseConfig) *bun.DB {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

	return NewWithDSN(dsn)
}

// NewWithDSN creates a new database connection with a custom DSN (useful for testing)
func NewWithDSN(dsn string) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.Ping(); err != nil {
		log.Fatal("Error pinging database:", err) // Fatal is OK here - can't run without DB
	}

	slog.Info("database connected successfully")
	return db
}

func Close(db *bun.DB) {
	if db != nil {
		db.Close()
	}
}

func RunMigrations(ctx context.Context, db *bun.DB, models ...interface{}) error {
	for _, model := range models {
		_, err := db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create table for model: %w", err)
		}
	}

	// Create trigger function for updated_at if it doesn't exist
	_, err := db.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';
	`)
	if err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Create trigger for projects table
	_, err = db.ExecContext(ctx, `
		DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
		CREATE TRIGGER update_projects_updated_at
			BEFORE UPDATE ON projects
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();
	`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	slog.Info("database migrations completed successfully")
	return nil
}
