package student

import (
	"context"
	"errors"
)

var (
	ErrStudentNotFound = errors.New("student not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type Service interface {
	CreateStudent(ctx context.Context, student *Student) (*Student, error)
	GetAllStudents(ctx context.Context) ([]Student, error)
	GetStudentByID(ctx context.Context, id int) (*Student, error)
	UpdateStudent(ctx context.Context, student *Student) error
	DeleteStudent(ctx context.Context, id int) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) CreateStudent(ctx context.Context, student *Student) (*Student, error) {
	return s.repo.Create(ctx, student)
}

func (s *service) GetAllStudents(ctx context.Context) ([]Student, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) GetStudentByID(ctx context.Context, id int) (*Student, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) UpdateStudent(ctx context.Context, student *Student) error {
	if student.ID <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Update(ctx, student)
}

func (s *service) DeleteStudent(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, id)
}
