package peers

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST /peers
func (h *PeersHandler) Connect(c *gin.Context) {
	var postPeersRequest dtos.PostPeersRequest
	if err := c.ShouldBindJSON(&postPeersRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  postPeersRequest,
		})
		return
	}

	if err := h.service.TryHolePunch(postPeersRequest.PeerMultiAddress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  postPeersRequest,
		})
		return
	}

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
