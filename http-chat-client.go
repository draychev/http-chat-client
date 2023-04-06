package main

import (
	"fmt"
	"net/http"

	"github.com/draychev/http-chat-client/pkg/client"

	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("http-chat-client/main")

func main() {
	quit := make(chan interface{})
	ready := make(chan interface{})
	go client.NewChatClient(quit, ready)
	<-ready
	log.Info().Msgf("Starting chat client for user %s; Listening on port %d; connecting to server %s on port %d",
		client.EnvVarUserName, client.Config.WebServerPortNumber, client.Config.ChatServerFQDN, client.Config.ChatServerPort)
	log.Fatal().Err(http.ListenAndServe(fmt.Sprintf(":%d", client.Config.WebServerPortNumber), nil)).Msg("Error starting server")
	close(quit)
}
