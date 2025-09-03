package server

import (
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
)

func (s *WebSocketServer) requireTopicDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning require message topic decorator.")
	return func(c *network.Client, msg network.WebSocketMessage) {
		if len(strings.TrimSpace(msg.Topic)) == 0 {
			s.SendToClient(c, network.Response{
				Id:      msg.Id,
				Type:    msg.Action,
				Code:    http.StatusBadRequest,
				Message: "topic was null or empty",
			})
			return
		}
		next(c, msg)
	}
}

func (s *WebSocketServer) requireDataDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning require message data decorator.")
	return func(c *network.Client, msg network.WebSocketMessage) {
		if msg.Data == nil {
			s.SendToClient(c, network.Response{
				Id:      msg.Id,
				Type:    msg.Action,
				Code:    http.StatusBadRequest,
				Message: "data was not supplied",
			})
			return
		}
		next(c, msg)
	}
}

func (s *WebSocketServer) metricsDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning metrics decorator")
	return func(c *network.Client, msg network.WebSocketMessage) {
		start := time.Now()
		next(c, msg)
		duration := time.Since(start)
		log.Infof("Handling client: %s for msg type: %s, took %v", c.Id, msg.Action, duration)
	}
}
