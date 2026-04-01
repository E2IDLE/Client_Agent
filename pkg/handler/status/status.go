package status

import (
	"Client_Agent/pkg/service"

	"github.com/gin-gonic/gin"
)

const (
	PathStatus = "/status"
)

type StatusHandler struct {
	service *service.StatusService
}

func NewStatusHandler(svc *service.StatusService) *StatusHandler {
	return &StatusHandler{service: svc}
}

func (h *StatusHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET(PathStatus, h.Get)
	rg.POST(PathStatus, h.Post)
}
