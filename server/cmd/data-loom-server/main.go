package main

import (
	"net/http"

	"github.com/atyalexyoung/data-loom/server/internal/config"
	_ "github.com/atyalexyoung/data-loom/server/internal/logging"
	"github.com/atyalexyoung/data-loom/server/internal/storage"
	"github.com/atyalexyoung/data-loom/server/internal/topic"
	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/server"
)

func main() {
	log.Info("Entering main...")

	cfg := config.Load()

	db, err := storage.NewStorage(cfg)
	if err != nil {
		log.Fatal("Error when setting up storage with error: ", err)
		return
	}
	defer db.Close()

	// create hub of clients
	clientHub := network.NewClientHub()
	// create topic manager
	topicManager := topic.NewTopicManager(db)

	wsServer := server.NewWebSocketServer(clientHub, topicManager)
	log.Info("Server starting at http://localhost:8080")
	http.ListenAndServe(":8080", wsServer.Handler())
}
