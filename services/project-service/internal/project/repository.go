package project

import (
	"context"
	"database/sql"
	"time"

	"grud/common/metrics"

	"github.com/uptrace/bun"
)

type Repository interface {
	Create(ctx context.Context, project *Project) error
	GetAll(ctx context.Context) ([]Project, error)
	GetByID(ctx context.Context, id int) (*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id int) error
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

func (r *repository) Create(ctx context.Context, project *Project) error {
	start := time.Now()
	_, err := r.db.NewInsert().Model(project).Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "insert", "projects", time.Since(start), err)

	if err != nil {
		return err
	}

	start = time.Now()
	err = r.db.NewSelect().Model(project).WherePK().Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "projects", time.Since(start), err)

	return err
}

func (r *repository) GetAll(ctx context.Context) ([]Project, error) {
	start := time.Now()
	var projects []Project
	err := r.db.NewSelect().Model(&projects).Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "projects", time.Since(start), err)

	return projects, err
}

func (r *repository) GetByID(ctx context.Context, id int) (*Project, error) {
	start := time.Now()
	project := new(Project)
	err := r.db.NewSelect().Model(project).Where("id = ?", id).Scan(ctx)
	r.metrics.Database.RecordQuery(ctx, "select", "projects", time.Since(start), err)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return project, nil
}

func (r *repository) Update(ctx context.Context, project *Project) error {
	start := time.Now()
	result, err := r.db.NewUpdate().
		Model(project).
		Column("name").
		WherePK().
		Exec(ctx)
	r.metrics.Database.RecordQuery(ctx, "update", "projects", time.Since(start), err)

	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int) error {
	start := time.Now()
	_, err := r.db.NewDelete().Model(&Project{ID: id}).WherePK().Exec(ctx)
	r.metrics.Database.RecordQuery(ctx, "delete", "projects", time.Since(start), err)

	if err != nil {
		return err
	}

	return nil
}
