package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/health", h.Health)
	router.GET("/ready", h.Ready)
}

type HealthResponse struct {
	Status string `json:"status"`
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *Handler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ready"})
}
