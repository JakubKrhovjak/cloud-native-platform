package message

import (
	"context"
	"time"

	"grud/common/metrics"

	"github.com/uptrace/bun"
)

type Repository interface {
	Create(ctx context.Context, message *Message) error
	GetByEmail(ctx context.Context, email string) ([]*Message, error)
}

type repository struct {
	db      *bun.DB
	metrics *metrics.Metrics
}

func NewRepository(db *bun.DB, m *metrics.Metrics) Repository {
	return &repository{
		db:      db,
		metrics: m,
	}
}

func (r *repository) Create(ctx context.Context, message *Message) error {
	start := time.Now()
	_, err := r.db.NewInsert().Model(message).Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "insert", "messages", time.Since(start), err)

	if err != nil {
		return err
	}
	// Reload to get DB-generated timestamps
	start = time.Now()
	err = r.db.NewSelect().Model(message).WherePK().Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "messages", time.Since(start), err)

	return err
}

func (r *repository) GetByEmail(ctx context.Context, email string) ([]*Message, error) {
	start := time.Now()
	var messages []*Message
	err := r.db.NewSelect().
		Model(&messages).
		Where("email = ?", email).
		Order("created_at DESC").
		Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "messages", time.Since(start), err)

	return messages, err
}
