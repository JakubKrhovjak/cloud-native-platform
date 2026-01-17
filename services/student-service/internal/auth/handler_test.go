package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	commonmetrics "grud/common/metrics"
	"grud/testing/testdb"
	"student-service/internal/auth"
	"student-service/internal/student"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Shared(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set JWT_SECRET for tests
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv("JWT_SECRET")

	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	// Run migrations for students and refresh_tokens tables
	pgContainer.RunMigrations(t, (*student.Student)(nil), (*auth.RefreshToken)(nil))

	// Create handler ONCE and reuse across all subtests
	mockMetrics := commonmetrics.NewMock()
	studentRepo := student.NewRepository(pgContainer.DB, mockMetrics)
	authRepo := auth.NewRepository(pgContainer.DB, mockMetrics)
	authService := auth.NewService(authRepo, studentRepo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	authHandler := auth.NewHandler(authService, logger)
	router := gin.New()
	authHandler.RegisterRoutes(router)

	t.Run("Register_Success", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		payload := map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"email":     "john.doe@example.com",
			"password":  "password123",
			"major":     "Computer Science",
			"year":      2,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response auth.AuthResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotNil(t, response.Student)

		// Verify auth cookie was set
		cookies := w.Result().Cookies()
		var foundAuthCookie bool
		for _, cookie := range cookies {
			if cookie.Name == "token" {
				foundAuthCookie = true
				assert.Equal(t, response.AccessToken, cookie.Value)
				break
			}
		}
		assert.True(t, foundAuthCookie, "token cookie should be set")
	})

	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Create a student first
		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		existingStudent := &student.Student{
			FirstName: "Existing",
			LastName:  "User",
			Email:     "duplicate@example.com",
			Password:  string(hashedPassword),
			Major:     "Physics",
			Year:      1,
		}
		_, err := pgContainer.DB.NewInsert().Model(existingStudent).Exec(ctx)
		require.NoError(t, err)

		// Try to register with same email
		payload := map[string]interface{}{
			"firstName": "New",
			"lastName":  "User",
			"email":     "duplicate@example.com",
			"password":  "password456",
			"major":     "Mathematics",
			"year":      2,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "email already exists")
	})

	t.Run("Register_ValidationError", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Missing required fields
		payload := map[string]interface{}{
			"email":    "invalid",
			"password": "short",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Login_Success", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Create a student first
		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		testStudent := &student.Student{
			FirstName: "Jane",
			LastName:  "Smith",
			Email:     "jane.smith@example.com",
			Password:  string(hashedPassword),
			Major:     "Engineering",
			Year:      3,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
		require.NoError(t, err)

		// Login with correct credentials
		payload := map[string]interface{}{
			"email":    "jane.smith@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response auth.AuthResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotNil(t, response.Student)

		// Verify auth cookie was set
		cookies := w.Result().Cookies()
		var foundAuthCookie bool
		for _, cookie := range cookies {
			if cookie.Name == "token" {
				foundAuthCookie = true
				assert.Equal(t, response.AccessToken, cookie.Value)
				break
			}
		}
		assert.True(t, foundAuthCookie, "token cookie should be set")
	})

	t.Run("Login_InvalidPassword", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Create a student
		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
		testStudent := &student.Student{
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
			Password:  string(hashedPassword),
			Major:     "Biology",
			Year:      2,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
		require.NoError(t, err)

		// Try login with wrong password
		payload := map[string]interface{}{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email or password")
	})

	t.Run("Login_UserNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		payload := map[string]interface{}{
			"email":    "nonexistent@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email or password")
	})

	t.Run("Login_ValidationError", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Missing required fields
		payload := map[string]interface{}{
			"email": "invalid-email",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Refresh_Success", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Create a student and get tokens via registration
		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		testStudent := &student.Student{
			FirstName: "Refresh",
			LastName:  "Test",
			Email:     "refresh@example.com",
			Password:  string(hashedPassword),
			Major:     "Chemistry",
			Year:      4,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
		require.NoError(t, err)

		// Login to get refresh token
		loginPayload := map[string]interface{}{
			"email":    "refresh@example.com",
			"password": "password123",
		}
		loginBody, _ := json.Marshal(loginPayload)

		loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()

		router.ServeHTTP(loginW, loginReq)
		require.Equal(t, http.StatusOK, loginW.Code)

		var loginResponse auth.AuthResponse
		err = json.NewDecoder(loginW.Body).Decode(&loginResponse)
		require.NoError(t, err)

		// Use refresh token to get new access token
		refreshPayload := map[string]interface{}{
			"refreshToken": loginResponse.RefreshToken,
		}
		refreshBody, _ := json.Marshal(refreshPayload)

		refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(refreshBody))
		refreshReq.Header.Set("Content-Type", "application/json")
		refreshW := httptest.NewRecorder()

		router.ServeHTTP(refreshW, refreshReq)

		assert.Equal(t, http.StatusOK, refreshW.Code)

		var refreshResponse auth.AuthResponse
		err = json.NewDecoder(refreshW.Body).Decode(&refreshResponse)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshResponse.AccessToken)
		assert.NotEmpty(t, refreshResponse.RefreshToken)
	})

	t.Run("Refresh_InvalidToken", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		payload := map[string]interface{}{
			"refreshToken": "invalid-token",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired refresh token")
	})

	t.Run("Logout_Success", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students", "refresh_tokens")

		// Create student and login
		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		testStudent := &student.Student{
			FirstName: "Logout",
			LastName:  "Test",
			Email:     "logout@example.com",
			Password:  string(hashedPassword),
			Major:     "Art",
			Year:      1,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
		require.NoError(t, err)

		// Login to get refresh token
		loginPayload := map[string]interface{}{
			"email":    "logout@example.com",
			"password": "password123",
		}
		loginBody, _ := json.Marshal(loginPayload)

		loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()

		router.ServeHTTP(loginW, loginReq)
		require.Equal(t, http.StatusOK, loginW.Code)

		var loginResponse auth.AuthResponse
		err = json.NewDecoder(loginW.Body).Decode(&loginResponse)
		require.NoError(t, err)

		// Logout
		logoutPayload := map[string]interface{}{
			"refreshToken": loginResponse.RefreshToken,
		}
		logoutBody, _ := json.Marshal(logoutPayload)

		logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(logoutBody))
		logoutReq.Header.Set("Content-Type", "application/json")
		logoutW := httptest.NewRecorder()

		router.ServeHTTP(logoutW, logoutReq)

		assert.Equal(t, http.StatusNoContent, logoutW.Code)

		// Verify refresh token is invalidated
		refreshPayload := map[string]interface{}{
			"refreshToken": loginResponse.RefreshToken,
		}
		refreshBody, _ := json.Marshal(refreshPayload)

		refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(refreshBody))
		refreshReq.Header.Set("Content-Type", "application/json")
		refreshW := httptest.NewRecorder()

		router.ServeHTTP(refreshW, refreshReq)

		assert.Equal(t, http.StatusUnauthorized, refreshW.Code, "refresh token should be invalid after logout")
	})
}
