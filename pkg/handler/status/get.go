package status

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *StatusHandler) Get(c *gin.Context) {
	status := h.service.GetStatus()

	c.JSON(http.StatusOK, status)
}
