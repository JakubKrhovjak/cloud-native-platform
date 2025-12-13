package project

import (
	"context"
	"errors"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type Service interface {
	CreateProject(ctx context.Context, project *Project) error
	GetAllProjects(ctx context.Context) ([]Project, error)
	GetProjectByID(ctx context.Context, id int) (*Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, id int) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) CreateProject(ctx context.Context, project *Project) error {
	return s.repo.Create(ctx, project)
}

func (s *service) GetAllProjects(ctx context.Context) ([]Project, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) GetProjectByID(ctx context.Context, id int) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) UpdateProject(ctx context.Context, project *Project) error {
	return s.repo.Update(ctx, project)
}

func (s *service) DeleteProject(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
