package stream

import (
	"Client_Agent/pkg/service"

	"github.com/gin-gonic/gin"
)

const (
	PathStream = "/stream"
)

type StreamHandler struct {
	service *service.StreamService
}

func NewStreamHandler(svc *service.StreamService) *StreamHandler {
	return &StreamHandler{service: svc}
}

func (h *StreamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET(PathStream, h.Get)
}
