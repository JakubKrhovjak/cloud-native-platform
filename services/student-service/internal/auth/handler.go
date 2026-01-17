package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   *Service
	logger    *slog.Logger
	validator *validator.Validate
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service:   service,
		logger:    logger,
		validator: validator.New(),
	}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	router.POST("/auth/refresh", h.Refresh)
	router.POST("/auth/logout", h.Logout)
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to decode request", "error", err)
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Warn("validation failed", "error", err)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			c.String(http.StatusConflict, err.Error())
			return
		}
		h.logger.Error("registration failed", "error", err)
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}

	// Set access token in cookie
	SetAuthCookie(c.Writer, resp.AccessToken)

	// Return response with refresh token in body
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to decode request", "error", err)
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Warn("validation failed", "error", err)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.String(http.StatusUnauthorized, err.Error())
			return
		}
		h.logger.Error("login failed", "error", err)
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}

	h.logger.Info("student logged in", "email", req.Email)

	// Set access token in cookie
	SetAuthCookie(c.Writer, resp.AccessToken)

	// Return response with refresh token in body
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to decode request", "error", err)
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Warn("validation failed", "error", err)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			c.String(http.StatusUnauthorized, err.Error())
			return
		}
		h.logger.Error("token refresh failed", "error", err)
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}

	// Set new access token in cookie
	SetAuthCookie(c.Writer, resp.AccessToken)

	// Return response with new refresh token
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to decode request", "error", err)
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.logger.Error("logout failed", "error", err)
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}

	// Clear auth cookie
	ClearAuthCookie(c.Writer)

	h.logger.Info("student logged out")

	c.Status(http.StatusNoContent)
}
