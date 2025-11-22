package student

import (
	"context"
	"database/sql"

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
	if err != nil {
		return nil, err
	}
	return students, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*Student, error) {
	student := new(Student)
	err := r.db.NewSelect().Model(student).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" || err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return student, nil
}

func (r *repository) Update(ctx context.Context, student *Student) error {
	result, err := r.db.NewUpdate().Model(student).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int) error {
	student := &Student{ID: id}
	result, err := r.db.NewDelete().Model(student).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
