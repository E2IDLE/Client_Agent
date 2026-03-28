package handler

import (
	"directp2p_client_agent/pkg/handler/auth"
	"directp2p_client_agent/pkg/middleware"
	"directp2p_client_agent/pkg/service"
	"directp2p_client_agent/pkg/session"

	"github.com/gin-gonic/gin"
)

func SetupRouter(sessions *session.Store) *gin.Engine {
	router := gin.New()
	router.Use(middleware.SetTraceID())
	router.Use(middleware.RequestLogger())
	router.Use(gin.Recovery())

	registerAPIRoutes(router, sessions)

	return router
}

func registerAPIRoutes(router *gin.Engine, sessions *session.Store) {
	p2pSvc := service.NewP2PService(sessions)
	p2pH := auth.NewP2PHandler(p2pSvc)
	p2pH.RegisterRoutes(&router.RouterGroup)

	protected := router.Group("")
	protected.Use(middleware.AuthRequired(sessions))
}
