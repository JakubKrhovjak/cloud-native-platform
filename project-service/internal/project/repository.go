package project

import (
	"context"
	"time"

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
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(project).Exec(ctx)
	return err
}

func (r *repository) GetAll(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := r.db.NewSelect().Model(&projects).Scan(ctx)
	return projects, err
}

func (r *repository) GetByID(ctx context.Context, id int) (*Project, error) {
	project := new(Project)
	err := r.db.NewSelect().Model(project).Where("id = ?", id).Scan(ctx)
	return project, err
}

func (r *repository) Update(ctx context.Context, project *Project) error {
	project.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(project).WherePK().Exec(ctx)
	return err
}

func (r *repository) Delete(ctx context.Context, id int) error {
	project := &Project{ID: id}
	_, err := r.db.NewDelete().Model(project).WherePK().Exec(ctx)
	return err
}
