package main

import (
	"log"
	"net/http"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/server"
)

func main() {
	log.Println("Entering main...")

	// create hub of clients
	clientHub := network.NewClientHub()
	// create topic manager
	topicManager := network.NewTopicManager()

	wsServer := server.NewWebSocketServer(clientHub, topicManager)
	log.Println("Server starting at http://localhost:8080")
	http.ListenAndServe(":8080", wsServer.Handler())
}
