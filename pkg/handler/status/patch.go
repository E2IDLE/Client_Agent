package status

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/dtos"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *StatusHandler) Patch(c *gin.Context) {
	var patchAgentStatusRequest dtos.PatchAgentStatusRequest
	if err := c.ShouldBindJSON(&patchAgentStatusRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			consts.KeyECode:   consts.ErrCodeInvalidParam,
			consts.KeyMessage: consts.MsgInvalidRequest,
			consts.KeyDetail:  patchAgentStatusRequest,
		})
		return
	}

	h.service.Sessions.Status.AgentName = patchAgentStatusRequest.Name

	c.JSON(http.StatusOK, h.service.Sessions.Status)
}
