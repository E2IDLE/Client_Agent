package auth

import (
	"directp2p_client_agent/pkg/consts"
	"directp2p_client_agent/pkg/dtos"
	"directp2p_client_agent/pkg/errors"
	"directp2p_client_agent/pkg/service"
	"directp2p_client_agent/pkg/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const PathSessionStart = "/session/start"

type P2PHandler struct {
	service *service.P2PService
}

func NewP2PHandler(svc *service.P2PService) *P2PHandler {
	return &P2PHandler{service: svc}
}

func (h *P2PHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST(PathSessionStart, h.SessionStartRequest)
}

func ClearAuthorizationCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(consts.AuthorizationCookieName, "", -1, consts.CookiePath, "", false, true)
}

func handleError(c *gin.Context, msg string, err error) {
	if appErr, ok := errors.As(err); ok {
		c.JSON(appErr.Status, gin.H{consts.KeyMessage: appErr.Message})
		return
	}

	util.LogError(c.Request.Context(), msg, logrus.Fields{"error": err.Error()})
	c.JSON(http.StatusInternalServerError, gin.H{consts.KeyMessage: consts.MsgInternalServerError})
}

// POST /session/start
func (h *P2PHandler) SessionStartRequest(c *gin.Context) {
	var request dtos.StartSessionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{consts.KeyMessage: consts.MsgInvalidRequest})
		return
	}

	token, err := h.service.CreateSession(request.TargetUser)
	if err != nil {
		return
	}

	h.service.TryHolePunch(token.SessionID)

	c.JSON(http.StatusCreated, gin.H{
		consts.KeySessionId: token,
	})
}
