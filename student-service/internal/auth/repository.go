package auth

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// CreateRefreshToken stores a new refresh token
func (r *Repository) CreateRefreshToken(ctx context.Context, studentID int, token string, expiresAt time.Time) error {
	refreshToken := &RefreshToken{
		StudentID: studentID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	_, err := r.db.NewInsert().Model(refreshToken).Exec(ctx)
	return err
}

// GetRefreshToken retrieves a refresh token by token string
func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	refreshToken := &RefreshToken{}
	err := r.db.NewSelect().
		Model(refreshToken).
		Where("token = ?", token).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}

// DeleteRefreshToken removes a refresh token (for logout)
func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := r.db.NewDelete().
		Model((*RefreshToken)(nil)).
		Where("token = ?", token).
		Exec(ctx)

	return err
}

// DeleteExpiredTokens removes all expired refresh tokens (cleanup)
func (r *Repository) DeleteExpiredTokens(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*RefreshToken)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)

	return err
}

// DeleteAllStudentTokens removes all refresh tokens for a student
func (r *Repository) DeleteAllStudentTokens(ctx context.Context, studentID int) error {
	_, err := r.db.NewDelete().
		Model((*RefreshToken)(nil)).
		Where("student_id = ?", studentID).
		Exec(ctx)

	return err
}
