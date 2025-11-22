package student

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateStudent(ctx context.Context, student *Student) error {
	if err := validateStudent(student); err != nil {
		return err
	}
	return s.repo.Create(ctx, student)
}

func (s *service) GetAllStudents(ctx context.Context) ([]Student, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) GetStudentByID(ctx context.Context, id int) (*Student, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	student, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentNotFound
	}
	return student, nil
}

func (s *service) UpdateStudent(ctx context.Context, student *Student) error {
	if student.ID <= 0 {
		return ErrInvalidInput
	}
	if err := validateStudent(student); err != nil {
		return err
	}
	err := s.repo.Update(ctx, student)
	if err != nil {
		return ErrStudentNotFound
	}
	return nil
}

func (s *service) DeleteStudent(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return ErrStudentNotFound
	}
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
