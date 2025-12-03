package auth

import (
	"time"

	"github.com/uptrace/bun"
)

// RefreshToken stores refresh tokens in database
type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens,alias:rt"`

	ID        int       `bun:"id,pk,autoincrement"`
	StudentID int       `bun:"student_id,notnull"`
	Token     string    `bun:"token,unique,notnull"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Major     string `json:"major"`
	Year      int    `json:"year" validate:"min=0,max=10"`
}

// RefreshRequest is the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// AuthResponse is the response for successful authentication
type AuthResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	Student      interface{} `json:"student"`
}
