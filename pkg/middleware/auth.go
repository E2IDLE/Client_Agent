package middleware

import (
	"directp2p_client_agent/pkg/consts"
	"directp2p_client_agent/pkg/errors"
	"directp2p_client_agent/pkg/session"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired 인증이 필요한 api 들을 위한 미들웨어로 header 토큰으로 인증 완료 후 사용자 정보를 context 에 저장한다.
func AuthRequired(sessions *session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := TokenFromRequest(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": consts.MsgMissingToken})
			return
		}

		_, ok := sessions.Lookup(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": errors.ErrInvalidToken.Error()})
			return
		}

		c.Next()
	}
}

func TokenFromRequest(c *gin.Context) string {
	headerValue := strings.TrimSpace(c.GetHeader(consts.AuthorizationHeader))
	if headerValue != "" {
		return headerValue
	}

	cookieValue, err := c.Cookie(consts.AuthorizationCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookieValue)
}
