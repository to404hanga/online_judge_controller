package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type HealthHandler struct {
	log loggerv2.Logger
}

var _ Handler = (*HealthHandler)(nil)

func NewHealthHandler(log loggerv2.Logger) *HealthHandler {
	return &HealthHandler{
		log: log,
	}
}

func (h *HealthHandler) Register(r *gin.Engine) {
	r.GET("/health", h.HealthCheck)
}

func (h *HealthHandler) HealthCheck(ctx *gin.Context) {
	h.log.Info("health check")
	ctx.Status(http.StatusOK)
}
