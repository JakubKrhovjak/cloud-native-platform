package message

import (
	"context"

	"github.com/uptrace/bun"
)

type Repository interface {
	Create(ctx context.Context, message *Message) error
}

type repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, message *Message) error {
	_, err := r.db.NewInsert().Model(message).Exec(ctx)
	if err != nil {
		return err
	}
	// Reload to get DB-generated timestamps
	return r.db.NewSelect().Model(message).WherePK().Scan(ctx)
}
