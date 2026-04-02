package peers

import (
	"Client_Agent/pkg/dtos"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *PeersHandler) Get(c *gin.Context) {
	connectedPeer := h.service.Sessions.Status.ConnectedPeer
	peers := []dtos.PeerInfo{}

	for i := 0; i < len(connectedPeer); i++ {
		peers = append(peers, dtos.PeerInfo{
			PeerId:         connectedPeer[i].Id,
			Nickname:       connectedPeer[i].Nickname,
			ConnectionType: connectedPeer[i].ConnectionType,
			Status:         "connected",
			Rtt:            connectedPeer[i].RTT,
			PacketLoss:     "null",
			MultiAddress:   connectedPeer[i].Address,
		})
	}

	c.JSON(http.StatusOK, peers)
}
