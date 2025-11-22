package student

import (
	"context"

	"github.com/uptrace/bun"
)

type Repository interface {
	Create(ctx context.Context, student *Student) error
	GetAll(ctx context.Context) ([]Student, error)
	GetByID(ctx context.Context, id int) (*Student, error)
	Update(ctx context.Context, student *Student) error
	Delete(ctx context.Context, id int) error
}

type repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, student *Student) error {
	_, err := r.db.NewInsert().Model(student).Exec(ctx)
	return err
}

func (r *repository) GetAll(ctx context.Context) ([]Student, error) {
	var students []Student
	err := r.db.NewSelect().Model(&students).Scan(ctx)
	return students, err
}

func (r *repository) GetByID(ctx context.Context, id int) (*Student, error) {
	student := new(Student)
	err := r.db.NewSelect().Model(student).Where("id = ?", id).Scan(ctx)
	return student, err
}

func (r *repository) Update(ctx context.Context, student *Student) error {
	_, err := r.db.NewUpdate().Model(student).WherePK().Exec(ctx)
	return err
}

func (r *repository) Delete(ctx context.Context, id int) error {
	student := &Student{ID: id}
	_, err := r.db.NewDelete().Model(student).WherePK().Exec(ctx)
	return err
}
