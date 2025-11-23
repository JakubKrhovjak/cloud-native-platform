package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

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
		cfg.DBName,
	)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.Ping(); err != nil {
		log.Fatal("Error pinging database:", err)
	}

	log.Println("Database connected successfully")
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
	log.Println("Database migrations completed successfully")
	return nil
}
