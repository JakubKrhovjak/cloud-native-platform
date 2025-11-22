package student

import (
	"context"
	"errors"
	"log/slog"
)

var (
	ErrStudentNotFound = errors.New("student not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type Service interface {
	CreateStudent(ctx context.Context, student *Student) error
	GetAllStudents(ctx context.Context) ([]Student, error)
	GetStudentByID(ctx context.Context, id int) (*Student, error)
	UpdateStudent(ctx context.Context, student *Student) error
	DeleteStudent(ctx context.Context, id int) error
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) CreateStudent(ctx context.Context, student *Student) error {
	s.logger.Info("creating student", "email", student.Email)

	if err := s.repo.Create(ctx, student); err != nil {
		s.logger.Error("failed to create student", "email", student.Email)
		return err
	}

	return nil
}

func (s *service) GetAllStudents(ctx context.Context) ([]Student, error) {
	s.logger.Info("fetching all students")

	students, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.Error("failed to fetch students")
		return nil, err
	}

	return students, nil
}

func (s *service) GetStudentByID(ctx context.Context, id int) (*Student, error) {
	s.logger.Info("fetching student by ID")

	if id <= 0 {
		s.logger.Warn("invalid student ID")
		return nil, ErrInvalidInput
	}

	student, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to fetch student")
		return nil, err
	}

	if student == nil {
		s.logger.Warn("student not found")
		return nil, ErrStudentNotFound
	}

	return student, nil
}

func (s *service) UpdateStudent(ctx context.Context, student *Student) error {
	s.logger.Info("updating student", "email", student.Email)

	if student.ID <= 0 {
		s.logger.Warn("invalid student ID for update")
		return ErrInvalidInput
	}

	err := s.repo.Update(ctx, student)
	if err != nil {
		s.logger.Error("failed to update student", "email", student.Email)
		return ErrStudentNotFound
	}

	return nil
}

func (s *service) DeleteStudent(ctx context.Context, id int) error {
	s.logger.Info("deleting student")

	if id <= 0 {
		s.logger.Warn("invalid student ID for deletion")
		return ErrInvalidInput
	}

	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("failed to delete student")
		return ErrStudentNotFound
	}

	return nil
}
