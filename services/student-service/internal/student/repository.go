package student

import (
	"context"
	"database/sql"
	"time"

	"grud/common/metrics"

	"github.com/uptrace/bun"
)

type Repository interface {
	Create(ctx context.Context, student *Student) (*Student, error)
	GetAll(ctx context.Context) ([]Student, error)
	GetByID(ctx context.Context, id int) (*Student, error)
	GetByEmail(ctx context.Context, email string) (*Student, error)
	Update(ctx context.Context, student *Student) error
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

func (r *repository) Create(ctx context.Context, student *Student) (*Student, error) {
	start := time.Now()
	_, err := r.db.NewInsert().Model(student).Returning("*").Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "insert", "students", time.Since(start), err)

	if err != nil {
		return nil, err
	}
	return student, nil
}

func (r *repository) GetAll(ctx context.Context) ([]Student, error) {
	start := time.Now()
	var students []Student
	err := r.db.NewSelect().Model(&students).Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "students", time.Since(start), err)

	return students, err
}

func (r *repository) GetByID(ctx context.Context, id int) (*Student, error) {
	start := time.Now()
	student := new(Student)
	err := r.db.NewSelect().Model(student).Where("id = ?", id).Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "students", time.Since(start), err)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return student, nil
}

func (r *repository) Update(ctx context.Context, student *Student) error {
	start := time.Now()
	result, err := r.db.NewUpdate().Model(student).WherePK().Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "update", "students", time.Since(start), err)

	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrStudentNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int) error {
	start := time.Now()
	student := &Student{ID: id}
	result, err := r.db.NewDelete().Model(student).WherePK().Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "delete", "students", time.Since(start), err)

	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrStudentNotFound
	}
	return nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*Student, error) {
	start := time.Now()
	student := new(Student)
	err := r.db.NewSelect().
		Model(student).
		Where("email = ?", email).
		Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "students", time.Since(start), err)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return student, nil
}
