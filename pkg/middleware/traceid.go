package middleware

import (
	"context"
	"directp2p_client_agent/pkg/consts"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func SetTraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.ReplaceAll(uuid.New().String(), "-", "")
		c.Set(consts.TraceIDKey, id)
		c.Header(consts.TraceIDHeader, id)

		ctx := context.WithValue(c.Request.Context(), consts.CtxTraceID, id)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
