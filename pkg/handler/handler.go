package handler

import (
	"Client_Agent/pkg/handler/status"
	"Client_Agent/pkg/middleware"
	"Client_Agent/pkg/service"
	"Client_Agent/pkg/session"

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
	statusSvc := service.NewStatusService(sessions)
	statusH := status.NewStatusHandler(statusSvc)
	statusH.RegisterRoutes(&router.RouterGroup)
}
