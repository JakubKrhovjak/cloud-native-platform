package project

import (
	"context"
	"database/sql"

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
	db *bun.DB
}

func NewRepository(db *bun.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, project *Project) error {
	_, err := r.db.NewInsert().Model(project).Exec(ctx)
	if err != nil {
		return err
	}
	// Reload to get DB-generated timestamps
	return r.db.NewSelect().Model(project).WherePK().Scan(ctx)
}

func (r *repository) GetAll(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := r.db.NewSelect().Model(&projects).Scan(ctx)
	return projects, err
}

func (r *repository) GetByID(ctx context.Context, id int) (*Project, error) {
	project := new(Project)
	err := r.db.NewSelect().Model(project).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return project, nil
}

func (r *repository) Update(ctx context.Context, project *Project) error {
	result, err := r.db.NewUpdate().
		Model(project).
		Column("name").
		WherePK().
		Exec(ctx)
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
	project := &Project{ID: id}
	result, err := r.db.NewDelete().Model(project).WherePK().Exec(ctx)
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
