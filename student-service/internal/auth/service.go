package auth

import (
	"context"
	"errors"
	"time"

	"student-service/internal/student"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

type Service struct {
	authRepo    *Repository
	studentRepo student.Repository
}

func NewService(authRepo *Repository, studentRepo student.Repository) *Service {
	return &Service{
		authRepo:    authRepo,
		studentRepo: studentRepo,
	}
}

// Register creates a new student account
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Check if email exists
	existingStudent, _ := s.studentRepo.GetByEmail(ctx, req.Email)
	if existingStudent != nil {
		return nil, ErrEmailExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create student
	newStudent := &student.Student{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Major:     req.Major,
		Year:      req.Year,
	}

	createdStudent, err := s.studentRepo.Create(ctx, newStudent)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateTokenPair(ctx, createdStudent)
}

// Login authenticates a student and returns tokens
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	// Find student by email
	stud, err := s.studentRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(stud.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	return s.generateTokenPair(ctx, stud)
}

// RefreshAccessToken generates a new access token using refresh token
func (s *Service) RefreshAccessToken(ctx context.Context, refreshTokenString string) (*AuthResponse, error) {
	// Validate refresh token
	refreshToken, err := s.authRepo.GetRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Get student
	stud, err := s.studentRepo.GetByID(ctx, refreshToken.StudentID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Generate new token pair
	return s.generateTokenPair(ctx, stud)
}

// Logout invalidates refresh token
func (s *Service) Logout(ctx context.Context, refreshTokenString string) error {
	return s.authRepo.DeleteRefreshToken(ctx, refreshTokenString)
}

// LogoutAll invalidates all refresh tokens for a student
func (s *Service) LogoutAll(ctx context.Context, studentID int) error {
	return s.authRepo.DeleteAllStudentTokens(ctx, studentID)
}

// generateTokenPair creates access and refresh tokens
func (s *Service) generateTokenPair(ctx context.Context, stud *student.Student) (*AuthResponse, error) {
	accessToken, err := GenerateAccessToken(stud.ID, stud.Email)
	if err != nil {
		return nil, err
	}

	// Generate refresh token (random, 7 days)
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store refresh token in database
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.authRepo.CreateRefreshToken(ctx, stud.ID, refreshToken, expiresAt); err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Student:      stud,
	}, nil
}
