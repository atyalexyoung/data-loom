package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("shutting down...")
		cancel()
	}()

	db, err := storage.NewStorage(cfg, ctx)
	if err != nil {
		log.Fatal("Error when setting up storage with error: ", err)
		return
	}
	defer db.Close()

	clientHub := network.NewClientHub()
	topicManager := topic.NewTopicManager(db)
	wsServer := server.NewWebSocketServer(clientHub, topicManager, cfg)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: wsServer.Handler(),
	}

	go func() {
		log.Infof("server starting at addr: %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error: ", err)
		}
	}()

	ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown: ", err)
	}
}
