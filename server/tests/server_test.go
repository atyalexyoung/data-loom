package tests

import (
	"context"
	"log"
	"net"
	"net/http"
	"testing"

	"github.com/atyalexyoung/data-loom/server/internal/config"
	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/server"
	"github.com/atyalexyoung/data-loom/server/internal/storage"
	"github.com/atyalexyoung/data-loom/server/internal/topic"
)

func startTestServer(t *testing.T) (*http.Server, context.CancelFunc, string, storage.Storage) {
	cfg := config.Load() // maybe load a test config
	cfg.StoragePath = t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	db, err := storage.NewStorage(cfg, ctx)
	if err != nil {
		t.Fatal(err)
	}

	clientHub := network.NewClientHub()
	topicManager := topic.NewTopicManager(db)
	wsServer := server.NewWebSocketServer(clientHub, topicManager, cfg)

	srv := &http.Server{
		Addr:    ":0", // OS assigns free port
		Handler: wsServer.Handler(),
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		log.Printf("serving on: %s", srv.Addr)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// return the actual listening address
	return srv, cancel, "ws://" + ln.Addr().String() + "/ws", db
}
