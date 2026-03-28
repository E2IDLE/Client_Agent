package main

import (
	"directp2p_client_agent/pkg/handler"
	"directp2p_client_agent/pkg/session"
)

const (
	defaultServerPort = ":8080"
)

func main() {
	sessions := session.NewStore()
	router := handler.SetupRouter(sessions)

	if err := router.Run(defaultServerPort); err != nil {
		panic(err)
	}
}
