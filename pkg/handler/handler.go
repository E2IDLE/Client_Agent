package handler

import (
	"Client_Agent/pkg/handler/peers"
	"Client_Agent/pkg/handler/status"
	"Client_Agent/pkg/handler/stream"
	"Client_Agent/pkg/middleware"
	"Client_Agent/pkg/service"
	"Client_Agent/pkg/session"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(sessions *session.Store) *gin.Engine {
	router := gin.New()
	router.Use(middleware.SetTraceID())
	router.Use(middleware.RequestLogger())
	router.Use(cors.Default())
	router.Use(gin.Recovery())

	registerAPIRoutes(router, sessions)

	return router
}

func registerAPIRoutes(router *gin.Engine, sessions *session.Store) {
	statusSvc := service.NewStatusService(sessions)
	statusH := status.NewStatusHandler(statusSvc)
	statusH.RegisterRoutes(&router.RouterGroup)

	P2PSvc := service.NewP2PService(sessions)
	peersH := peers.NewPeersHandler(P2PSvc)
	peersH.RegisterRoutes(&router.RouterGroup)

	StreamSvc := service.NewStreamService(sessions)
	StreamH := stream.NewStreamHandler(StreamSvc)
	StreamH.RegisterRoutes(&router.RouterGroup)
}
