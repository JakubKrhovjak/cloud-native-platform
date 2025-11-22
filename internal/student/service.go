package student

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"go.uber.org/zap"
)

var (
	ErrStudentNotFound = errors.New("student not found")
	ErrInvalidEmail    = errors.New("invalid email format")
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
	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) CreateStudent(ctx context.Context, student *Student) error {
	s.logger.Info("creating student",
		zap.String("email", student.Email),
		zap.String("first_name", student.FirstName),
		zap.String("last_name", student.LastName),
	)

	if err := validateStudent(student); err != nil {
		s.logger.Error("validation failed",
			zap.Error(err),
			zap.String("email", student.Email),
		)
		return err
	}

	if err := s.repo.Create(ctx, student); err != nil {
		s.logger.Error("failed to create student",
			zap.Error(err),
			zap.String("email", student.Email),
		)
		return err
	}

	s.logger.Info("student created successfully",
		zap.Int("id", student.ID),
		zap.String("email", student.Email),
	)
	return nil
}

func (s *service) GetAllStudents(ctx context.Context) ([]Student, error) {
	s.logger.Info("fetching all students")

	students, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.Error("failed to fetch students", zap.Error(err))
		return nil, err
	}

	s.logger.Info("students fetched successfully", zap.Int("count", len(students)))
	return students, nil
}

func (s *service) GetStudentByID(ctx context.Context, id int) (*Student, error) {
	s.logger.Info("fetching student by ID", zap.Int("id", id))

	if id <= 0 {
		s.logger.Warn("invalid student ID", zap.Int("id", id))
		return nil, ErrInvalidInput
	}

	student, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to fetch student", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	if student == nil {
		s.logger.Warn("student not found", zap.Int("id", id))
		return nil, ErrStudentNotFound
	}

	s.logger.Info("student fetched successfully",
		zap.Int("id", id),
		zap.String("email", student.Email),
	)
	return student, nil
}

func (s *service) UpdateStudent(ctx context.Context, student *Student) error {
	s.logger.Info("updating student",
		zap.Int("id", student.ID),
		zap.String("email", student.Email),
	)

	if student.ID <= 0 {
		s.logger.Warn("invalid student ID for update", zap.Int("id", student.ID))
		return ErrInvalidInput
	}

	if err := validateStudent(student); err != nil {
		s.logger.Error("validation failed for update",
			zap.Error(err),
			zap.Int("id", student.ID),
		)
		return err
	}

	err := s.repo.Update(ctx, student)
	if err != nil {
		s.logger.Error("failed to update student",
			zap.Error(err),
			zap.Int("id", student.ID),
		)
		return ErrStudentNotFound
	}

	s.logger.Info("student updated successfully",
		zap.Int("id", student.ID),
		zap.String("email", student.Email),
	)
	return nil
}

func (s *service) DeleteStudent(ctx context.Context, id int) error {
	s.logger.Info("deleting student", zap.Int("id", id))

	if id <= 0 {
		s.logger.Warn("invalid student ID for deletion", zap.Int("id", id))
		return ErrInvalidInput
	}

	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("failed to delete student",
			zap.Error(err),
			zap.Int("id", id),
		)
		return ErrStudentNotFound
	}

	s.logger.Info("student deleted successfully", zap.Int("id", id))
	return nil
}

func validateStudent(student *Student) error {
	if student.FirstName == "" {
		return fmt.Errorf("%w: first name is required", ErrInvalidInput)
	}
	if student.LastName == "" {
		return fmt.Errorf("%w: last name is required", ErrInvalidInput)
	}
	if student.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if !isValidEmail(student.Email) {
		return ErrInvalidEmail
	}
	if student.Year < 0 || student.Year > 10 {
		return fmt.Errorf("%w: year must be between 0 and 10", ErrInvalidInput)
	}
	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
