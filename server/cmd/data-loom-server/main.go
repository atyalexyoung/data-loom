package main

import (
	"net/http"

	_ "github.com/atyalexyoung/data-loom/server/internal/logging"
	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/server"
)

func main() {
	log.Info("Entering main...")

	// create hub of clients
	clientHub := network.NewClientHub()
	// create topic manager
	topicManager := network.NewTopicManager()

	wsServer := server.NewWebSocketServer(clientHub, topicManager)
	log.Info("Server starting at http://localhost:8080")
	http.ListenAndServe(":8080", wsServer.Handler())
}
