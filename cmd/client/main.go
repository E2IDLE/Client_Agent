package main

import (
	"directp2p_client_agent/pkg/consts"
	"directp2p_client_agent/pkg/handler"
	"directp2p_client_agent/pkg/session"

	"github.com/gin-gonic/gin"
)

const (
	defaultServerPort = ":8080"
)

func main() {
	if consts.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	sessions := session.NewStore()
	router := handler.SetupRouter(sessions)

	if err := router.Run(defaultServerPort); err != nil {
		panic(err)
	}
}
