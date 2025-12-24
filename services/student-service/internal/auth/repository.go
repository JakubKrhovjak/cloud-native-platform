package auth

import (
	"context"
	"time"

	"grud/common/metrics"

	"github.com/uptrace/bun"
)

type Repository struct {
	db      *bun.DB
	metrics *metrics.Metrics
}

func NewRepository(db *bun.DB, m *metrics.Metrics) *Repository {
	return &Repository{
		db:      db,
		metrics: m,
	}
}

// CreateRefreshToken stores a new refresh token
func (r *Repository) CreateRefreshToken(ctx context.Context, studentID int, token string, expiresAt time.Time) error {
	start := time.Now()
	refreshToken := &RefreshToken{
		StudentID: studentID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	_, err := r.db.NewInsert().Model(refreshToken).Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "insert", "refresh_tokens", time.Since(start), err)

	return err
}

func (r *Repository) cleanTokens(ctx context.Context, id int) error {
	start := time.Now()
	_, err := r.db.NewDelete().
		Model(&RefreshToken{}).
		Where("student_id = ?", id).
		Exec(ctx)
	r.metrics.Database.RecordQuery(ctx, "delete", "refresh_tokens", time.Since(start), err)
	return err
}

// GetRefreshToken retrieves a refresh token by token string
func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	start := time.Now()
	refreshToken := &RefreshToken{}
	err := r.db.NewSelect().
		Model(refreshToken).
		Where("token = ?", token).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)

	r.metrics.Database.RecordQuery(ctx, "select", "refresh_tokens", time.Since(start), err)

	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}

// DeleteRefreshToken removes a refresh token (for logout)
func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	start := time.Now()
	_, err := r.db.NewDelete().
		Model((*RefreshToken)(nil)).
		Where("token = ?", token).
		Exec(ctx)
	r.metrics.Database.RecordQuery(ctx, "delete", "refresh_tokens", time.Since(start), err)

	return err
}

// DeleteExpiredTokens removes all expired refresh tokens (cleanup)
func (r *Repository) DeleteExpiredTokens(ctx context.Context) error {
	start := time.Now()
	_, err := r.db.NewDelete().
		Model((*RefreshToken)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)

	r.metrics.Database.RecordQuery(ctx, "delete", "refresh_tokens", time.Since(start), err)

	return err
}

// DeleteAllStudentTokens removes all refresh tokens for a student
func (r *Repository) DeleteAllStudentTokens(ctx context.Context, studentID int) error {
	start := time.Now()
	_, err := r.db.NewDelete().
		Model((*RefreshToken)(nil)).
		Where("student_id = ?", studentID).
		Exec(ctx)
	if r.metrics != nil {
		r.metrics.Database.RecordQuery(ctx, "delete", "refresh_tokens", time.Since(start), err)
	}
	return err
}
