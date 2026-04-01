package peers

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *PeersHandler) Post(c *gin.Context) {
	var postPeersRequest dtos.PostPeersRequest
	if err := c.ShouldBindJSON(&postPeersRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  postPeersRequest,
		})
		return
	}

	h.service.TryHolePunch(postPeersRequest.PeerMultiAddress)
}
