package server

import (
	"fmt"
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

// requireDataDecorator will verify that the message has a "Data" field, and that
// it has some data in it.
func (s *WebSocketServer) requireDataDecorator(next HandlerFunc) HandlerFunc {
	log.Trace("Returning require message data decorator.")
	return func(c *network.Client, msg network.WebSocketMessage) {
		if msg.Data == nil { // check if null
			s.AckResponseError(c, msg, fmt.Errorf("message data was null."))
			return
		}
		if len(msg.Data) == 0 { // check if empty
			s.AckResponseError(c, msg, fmt.Errorf("message data was empty."))
			return
		}
		next(c, msg) // if we good, then move along.
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
