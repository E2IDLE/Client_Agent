package middleware

import (
	"bytes"
	"directp2p_client_agent/pkg/consts"
	"directp2p_client_agent/pkg/util"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	responseBodyLogLimit = 512

	logTypeMethod      = "method"
	logTypePath        = "path"
	logTypeQuery       = "query"
	logTypeIP          = "client_ip"
	logTypeUserAgent   = "user_agent"
	logTypeContentType = "content_type"
	logTypeRequestBody = "request_body"

	logTypeStatus   = "status"
	logTypeLatency  = "latency_ms"
	logTypeRespSize = "response_size"
	logTypeRespBody = "response_body"

	logSkipPathStatic = "/static/"
	logSkipPathHealth = "/health"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func RequestLogger() gin.HandlerFunc {
	log := util.GetLogger()

	return func(c *gin.Context) {
		if shouldSkip(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		traceID := c.GetString(consts.TraceIDKey)

		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestBody = util.MaskSensitiveFormat(string(bodyBytes))
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		log.WithFields(logrus.Fields{
			consts.TraceIDKey:  traceID,
			logTypeMethod:      c.Request.Method,
			logTypePath:        c.Request.URL.Path,
			logTypeQuery:       c.Request.URL.RawQuery,
			logTypeIP:          c.ClientIP(),
			logTypeUserAgent:   c.Request.UserAgent(),
			logTypeContentType: c.ContentType(),
			logTypeRequestBody: requestBody,
		}).Info("요청 수신")

		rbw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = rbw

		c.Next()

		latency := time.Since(start)
		responseBody := rbw.body.String()
		if len(responseBody) > responseBodyLogLimit {
			responseBody = responseBody[:responseBodyLogLimit] + "...(truncated)"
		}
		responseBody = util.MaskSensitiveFormat(responseBody)

		log.WithFields(logrus.Fields{
			consts.TraceIDKey: traceID,
			logTypeStatus:     c.Writer.Status(),
			logTypeLatency:    latency.Milliseconds(),
			logTypeRespSize:   c.Writer.Size(),
			logTypeRespBody:   responseBody,
		}).Info("응답 완료")
	}
}

func shouldSkip(path string) bool {
	if strings.HasPrefix(path, logSkipPathStatic) {
		return true
	}
	if path == logSkipPathHealth {
		return true
	}
	if path == "/" {
		return true
	}
	return false
}
