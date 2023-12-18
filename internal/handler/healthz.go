package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthzHandler struct{}

func NewHealthzHandler(r *gin.RouterGroup) {
	handler := &healthzHandler{}

	r.GET("/healthz", handler.Healthz)
}

func (h *healthzHandler) Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}
