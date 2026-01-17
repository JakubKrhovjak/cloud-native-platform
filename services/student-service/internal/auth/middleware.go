package auth

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	// StudentIDKey is the context key for student ID
	StudentIDKey contextKey = "student_id"
	// EmailKey is the context key for email
	EmailKey contextKey = "email"
)

// AuthMiddleware validates JWT from cookie and adds claims to context
func AuthMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from cookie
		cookie, err := c.Request.Cookie("token")
		if err != nil {
			logger.Warn("no auth cookie found", "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Validate JWT
		claims, err := ValidateAccessToken(cookie.Value)
		if err != nil {
			logger.Warn("invalid token", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Add claims to context
		ctx := context.WithValue(c.Request.Context(), StudentIDKey, claims.StudentID)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)
		c.Request = c.Request.WithContext(ctx)

		// Call next handler
		c.Next()
	}
}

// GetStudentID extracts student ID from context
func GetStudentID(ctx context.Context) (int, bool) {
	studentID, ok := ctx.Value(StudentIDKey).(int)
	return studentID, ok
}

// GetEmail extracts email from context
func GetEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(EmailKey).(string)
	return email, ok
}

// SetAuthCookie sets JWT token in secure HttpOnly cookie
func SetAuthCookie(w http.ResponseWriter, token string) {
	// Determine SameSite based on environment
	sameSite := http.SameSiteStrictMode
	env := os.Getenv("ENV")
	if env == "development" || env == "local" {
		sameSite = http.SameSiteLaxMode // Allow testing from Postman
	}

	// Secure cookies require HTTPS - enable for production environments
	secure := env == "production" || env == "prod" || env == "gcp-gke"

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,     // XSS protection
		Secure:   secure,   // HTTPS only in production
		SameSite: sameSite, // CSRF protection
		Path:     "/",
		MaxAge:   900, // 15 minutes
	})
}

// ClearAuthCookie removes the auth cookie
func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   os.Getenv("ENV") != "local",
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   -1, // Delete cookie
	})
}
