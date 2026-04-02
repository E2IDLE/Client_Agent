package peers

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// POST /peers/:id
func (h *PeersHandler) Post(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  id,
		})
		return
	}

	var sendDataRequest dtos.SendDataRequest
	if err := c.ShouldBindJSON(&sendDataRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  sendDataRequest,
		})
		return
	}

	//implement file transfer via stream
	//connection is on
	//h.service.Sessions.ConnectionsMu.Lock()
	//h.service.Sessions.Connections[]

	c.JSON(http.StatusOK, dtos.SendDataResponse{
		TransferId: "",
		Message:    "OK",
	})
}
