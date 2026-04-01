package peers

import (
	"Client_Agent/pkg/service"

	"github.com/gin-gonic/gin"
)

const (
	PathPeers = "/peers"
)

type PeersHandler struct {
	service *service.P2PService
}

func NewPeersHandler(svc *service.P2PService) *PeersHandler {
	return &PeersHandler{service: svc}
}

func (h *PeersHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET(PathPeers, h.Get)
	rg.POST(PathPeers, h.Post)
}
