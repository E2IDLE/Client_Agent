package main

import (
	"Client_Agent/pkg/consts"
	"Client_Agent/pkg/handler"
	"Client_Agent/pkg/session"

	"github.com/gin-gonic/gin"
)

const (
	defaultServerPort = ":17432"
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
